package cardv4

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func conversionInputs(t *testing.T, file string) ConversionInputs {
	t.Helper()
	var in ConversionInputs
	if e := json.Unmarshal(fixtureBytes(t, file), &in); e != nil {
		t.Fatal(e)
	}
	return in
}

// Independent expected documents, source-derived identities and the separate
// preservation checker agree; repeated conversion is byte deterministic.
func TestConversionExpectedDocuments(t *testing.T) {
	for _, tc := range []struct{ source, premises, expected string }{
		{"source-v3.json", "expectations.json", "converted-v4.json"},
		{"compat-id-source-v3.json", "compat-id-expectations.json", "compat-id-converted-v4.json"},
		{"compat-repeated-required-source-v3.json", "compat-repeated-required-expectations.json", "compat-repeated-required-converted-v4.json"},
		{"compat-cross-group-source-v3.json", "compat-cross-group-expectations.json", "compat-cross-group-converted-v4.json"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			raw := fixtureBytes(t, tc.source)
			before := append([]byte(nil), raw...)
			in := conversionInputs(t, tc.premises)
			converted, e := Convert(raw, in)
			if e != nil {
				t.Fatal(e)
			}
			again, e := Convert(raw, in)
			if e != nil || !bytes.Equal(converted, again) || !bytes.Equal(before, raw) {
				t.Fatal("non-deterministic conversion or source mutation")
			}
			got, e := Parse(converted, 4)
			if e != nil {
				t.Fatal(e)
			}
			expected := fixture(t, tc.expected)
			if !reflect.DeepEqual(got, expected) {
				t.Fatal("complete expected document differs")
			}
			source, _ := Parse(raw, 3)
			if e := VerifyConversionPreservation(source, got, in.PreservationInputs); e != nil {
				t.Fatal(e)
			}
			for _, v := range list(source["members"]) {
				m := obj(v)
				if obj(m["a2a"]) == nil {
					continue
				}
				bk := str(m["binding"])
				if bk == "" {
					bk = str(obj(source["product"])["default_binding"])
				}
				b := obj(obj(source["bindings"])[bk])
				conn := obj(obj(source["connections"])[str(b["connection"])])
				want := "@" + str(b["identity_prefix"]) + str(m["id"]) + ":" + str(conn["server_name"])
				target := indexedObjects(list(got["members"]), "id")[str(m["id"])]
				if obj(target["identity"])["mxid"] != want {
					t.Fatal("source-derived identity differs")
				}
			}
		})
	}
}

// Missing authority, lossy extensions, changed placements and false pins refuse
// with no partial result, paired with the same valid conversion premises.
func TestConversionPremiseRefusals(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ConversionInputs, Object)
		rule   string
	}{
		{"missing source revision", func(in *ConversionInputs, _ Object) { in.SourceRevision = "" }, "conversion_inputs_missing"},
		{"missing hash", func(in *ConversionInputs, _ Object) { in.SourceSHA256 = "" }, "conversion_inputs_missing"},
		{"wrong hash", func(in *ConversionInputs, _ Object) { in.SourceSHA256 = string(bytes.Repeat([]byte("a"), 64)) }, "source_digest_mismatch"},
		{"missing host", func(in *ConversionInputs, _ Object) { in.HostProject = "" }, "conversion_inputs_missing"},
		{"noncanonical base", func(in *ConversionInputs, _ Object) { in.SourceCardRoot += "/./" }, "conversion_inputs_invalid"},
		{"unknown host", func(in *ConversionInputs, _ Object) { in.HostProject = "missing" }, "conversion_host_project"},
		{"host relocated", func(in *ConversionInputs, _ Object) { in.TargetCardRoot = "fixture-root/other" }, "conversion_host_project"},
		{"missing external mapping", func(in *ConversionInputs, _ Object) { in.ExternalParentMapping = nil }, "conversion_external_mapping_required"},
		{"missing tuple", func(in *ConversionInputs, _ Object) { in.ProspectiveNonchatChoices = nil }, "conversion_identity_preservation"},
		{"missing reservation", func(in *ConversionInputs, _ Object) { in.Reservations = nil }, "star_reservation_required"},
		{"path escape", func(in *ConversionInputs, _ Object) { in.SourceCardRoot = "outside/legacy" }, "path_root_escape"},
		{"extension", func(_ *ConversionInputs, c Object) { c["extension"] = true }, "conversion_extension_unsupported"},
		{"binding extension", func(_ *ConversionInputs, c Object) {
			obj(obj(c["bindings"])["legacy"])["identity_domain"] = "other.example"
		}, "conversion_extension_unsupported"},
		{"remote disposition", func(in *ConversionInputs, _ Object) {
			in.ProjectLocations = map[string]Object{"beacon": {"kind": "remote", "operator": "fixture", "execution_host": "station", "project_locator": "synthetic"}}
		}, "conversion_location_disposition_required"},
		{"changed local disposition", func(in *ConversionInputs, _ Object) {
			in.ProjectLocations = map[string]Object{"beacon": {"kind": "local", "path": "../other"}}
		}, "conversion_path_preservation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := conversionInputs(t, "expectations.json")
			c := fixture(t, "source-v3.json")
			if _, e := Convert(fixtureBytes(t, "source-v3.json"), in); e != nil {
				t.Fatal(e)
			}
			tc.mutate(&in, c)
			raw := encoded(t, c)
			// Source mutations are pinned consistently; false-pin cases retain their explicit hash.
			if strings.Contains(tc.name, "extension") {
				sum := sha256.Sum256(raw)
				in.SourceSHA256 = hex.EncodeToString(sum[:])
			} else {
				raw = fixtureBytes(t, "source-v3.json")
			}
			out, e := Convert(raw, in)
			if e == nil || e.Rule != tc.rule || len(out) != 0 {
				t.Fatalf("want %s with no result, got %v", tc.rule, e)
			}
		})
	}
}

// Valid legacy anchors are expanded before emission. Output passes the strict
// v4 parser, retains the anchored text and never contains alias syntax.
func TestConversionExpandsLegacyAliases(t *testing.T) {
	raw := fixtureBytes(t, "source-v3.json")
	raw = bytes.Replace(raw, []byte(`"description": "Synthetic preservation fixture"`), []byte(`"description": &description "Synthetic preservation fixture"`), 1)
	raw = bytes.Replace(raw, []byte(`"description": "Synthetic aurora-host project"`), []byte(`"description": *description`), 1)
	if !bytes.Contains(raw, []byte("*description")) {
		t.Fatal("alias plant missing")
	}
	if e := Validate(raw, Options{Version: 3}); e != nil {
		t.Fatal(e)
	}
	in := conversionInputs(t, "expectations.json")
	sum := sha256.Sum256(raw)
	in.SourceSHA256 = hex.EncodeToString(sum[:])
	out, e := Convert(raw, in)
	if e != nil {
		t.Fatal(e)
	}
	c, e := Parse(out, 4)
	if e != nil {
		t.Fatal(e)
	}
	if obj(list(c["projects"])[0])["description"] != obj(c["product"])["description"] || bytes.Contains(out, []byte("*description")) {
		t.Fatal("alias not expanded losslessly")
	}
}

// The complete conversion matches the independently authored exact rollback
// document; pilot comparison still refuses a neighboring non-pilot mutation.
// This is an oracle comparison, not execution of a live rollback.
func TestConversionRollbackOracle(t *testing.T) {
	out, e := Convert(fixtureBytes(t, "source-v3.json"), conversionInputs(t, "expectations.json"))
	if e != nil {
		t.Fatal(e)
	}
	converted, e := Parse(out, 4)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(converted, fixture(t, "rollback-v4.json")) {
		t.Fatal("complete prior rollback record differs")
	}
	if e := VerifyPilotPreservation(converted, fixture(t, "pilot-v4.json"), "aurora/beacon_orch"); e != nil {
		t.Fatal(e)
	}
	if e := VerifyPilotPreservation(converted, fixture(t, "mixed-reject-nonpilot-state.json"), "aurora/beacon_orch"); e == nil || e.Rule != "nonpilot_preservation" {
		t.Fatal("non-pilot mutation not refused", e)
	}
}
