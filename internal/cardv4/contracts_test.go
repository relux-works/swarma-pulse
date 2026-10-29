package cardv4

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

// Paired mutations cover closed branches, reference resolution and realistic
// forbidden transitions independently of the finite accepted vectors.
func TestContractNeighbors(t *testing.T) {
	cases := []struct {
		name, rule string
		mutate     func(Object)
	}{
		{"null", "null_value", func(c Object) { obj(c["product"])["description"] = nil }},
		{"unknown nested", "unknown_field", func(c Object) { obj(obj(c["profiles"])["control"])["extension"] = true }},
		{"wrong collection", "collection_shape", func(c Object) { c["members"] = int64(42) }},
		{"duplicate project", "duplicate_project", func(c Object) { list(c["projects"])[1] = list(c["projects"])[0] }},
		{"unknown binding", "binding_reference", func(c Object) { obj(list(c["members"])[3])["binding"] = "missing" }},
		{"domain mismatch", "binding_domain_mismatch", func(c Object) { obj(obj(c["bindings"])["gr"])["identity_domain"] = "other.example" }},
		{"unknown connection", "connection_reference", func(c Object) { obj(obj(c["bindings"])["gr"])["connection"] = "missing" }},
		{"profile reference", "profile_reference", func(c Object) { obj(list(c["members"])[3])["profile"] = "missing" }},
		{"project reference", "project_reference", func(c Object) { obj(list(c["members"])[3])["project"] = "missing" }},
		{"scope", "scope_role", func(c Object) { obj(list(c["members"])[3])["scope"] = "project" }},
		{"host anchor", "host_project_anchor", func(c Object) { obj(obj(list(c["projects"])[0])["location"])["path"] = "../beacon" }},
		{"parent", "reporting_line", func(c Object) { obj(list(c["members"])[3])["parent"] = "beacon_orch" }},
		{"keeper decision", "reporting_line", func(c Object) { delete(obj(list(c["projects"])[0]), "orchestrator_ref") }},
		{"external duplicate", "duplicate_external", func(c Object) {
			c["external_members"] = append(list(c["external_members"]), list(c["external_members"])[0])
		}},
		{"external same product", "external_self_product", func(c Object) { obj(list(c["external_members"])[0])["product"] = "aurora" }},
		{"manager a2a", "route_contract", func(c Object) { delete(obj(list(c["members"])[1]), "a2a") }},
		{"worker a2a route", "route_contract", func(c Object) { obj(list(c["members"])[3])["result_route"] = "a2a" }},
		{"worker staffing", "staffing_manager_only", func(c Object) { obj(list(c["members"])[3])["staffing"] = []any{} }},
		{"nonchat legacy", "nonchat_legacy_forbidden", func(c Object) { obj(list(c["members"])[3])["identity"] = obj(list(c["members"])[1])["identity"] }},
		{"profile source exclusivity", "profile_source", func(c Object) {
			obj(obj(c["profiles"])["control"])["registry"] = Object{"repo": "registry.example/catalog/profiles", "directory": "control"}
		}},
		{"version range", "version_range_format", func(c Object) { obj(obj(obj(c["profiles"])["control"])["requirement"])["compatible"] = "latest" }},
		{"version selection", "profile_version_incompatible", func(c Object) { obj(obj(obj(c["profiles"])["control"])["selected"])["version"] = "1.0.0" }},
		{"parent route", "a2a_contract", func(c Object) { obj(obj(list(c["members"])[1])["a2a"])["parent_route"] = "#missing:relux.works" }},
		{"space selector", "a2a_contract", func(c Object) {
			a := obj(obj(list(c["members"])[1])["a2a"])
			space := obj(obj(c["product"])["topology"])["space"]
			a["rooms"] = append(list(a["rooms"]), space)
			obj(list(obj(a["subscriptions"])["required"])[0])["rooms"] = []any{space}
		}},
		{"all events scope", "a2a_contract", func(c Object) {
			obj(list(obj(obj(obj(list(c["members"])[1])["a2a"])["subscriptions"])["required"])[1])["all_events"] = true
		}},
		{"subtype kinds", "a2a_contract", func(c Object) {
			s := obj(list(obj(obj(obj(list(c["members"])[1])["a2a"])["subscriptions"])["required"])[0])
			s["kinds"] = []any{"notice"}
			s["subtypes"] = []any{"landing"}
		}},
		{"binding a2a", "binding_connection_mismatch", func(c Object) { obj(obj(list(c["members"])[1])["a2a"])["connection"] = "target-gr" }},
		{"nonportable new id", "logical_source_required", func(c Object) { obj(list(c["members"])[3])["id"] = "vault=dev" }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			c := fixture(t, "converted-v4.json")
			o := fixtureOptions(t)
			if e := Validate(encoded(t, c), o); e != nil {
				t.Fatal(e)
			}
			test.mutate(c)
			e := Validate(encoded(t, c), o)
			if e == nil || e.Rule != test.rule {
				t.Fatalf("expected %s, got %v", test.rule, e)
			}
		})
	}
}

// Existing opaque keys remain exact through source attestation and references;
// they are never transformed into paths, diagnostics or non-chat Matrix names.
func TestOpaqueNonchatLogicalID(t *testing.T) {
	for _, id := range []string{"path/like=key", "line\nbreak", strings.Repeat("legacy", 100)} {
		source := fixture(t, "source-v3.json")
		obj(list(source["members"])[3])["id"] = id
		raw := encoded(t, source)
		if e := Validate(raw, Options{Version: 3}); e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(raw)
		hash := hex.EncodeToString(sum[:])
		c := fixture(t, "converted-v4.json")
		m := obj(list(c["members"])[3])
		m["id"] = id
		m["logical_source_ref"] = Object{"kind": "v3-card", "revision": "opaque-fixture-001", "sha256": hash, "member": "aurora/" + id}
		o := fixtureOptions(t)
		o.Sources = append(o.Sources, Source{"v3-card", "opaque-fixture-001", hash, raw})
		o.Reservations[1].Holder = "aurora/" + id
		if e := Validate(encoded(t, c), o); e != nil {
			t.Fatal(e)
		}
	}
}

// Source authority comes from actual bytes and a valid v3 document; a digest
// alone cannot turn precomputed output, an invalid card or another key into it.
func TestSourceAuthorityControls(t *testing.T) {
	for _, kind := range []string{"invalid source", "digest differs", "output only", "missing reservation", "wrong holder"} {
		t.Run(kind, func(t *testing.T) {
			o := fixtureOptions(t)
			c := fixture(t, "converted-v4.json")
			want := "legacy_source_invalid"
			switch kind {
			case "invalid source":
				s := fixture(t, "source-v3.json")
				obj(list(s["members"])[3])["parent"] = "missing"
				o.Sources[0].Bytes = encoded(t, s)
			case "output only":
				o.Sources[0].Bytes = []byte(`{"aurora/beacon_orch":"@station_beacon_orch:relux.works"}`)
			case "digest differs":
				o.Sources[0].Bytes = append(o.Sources[0].Bytes, ' ')
				want = "source_digest_mismatch"
			case "missing reservation":
				o.Reservations = nil
				want = "star_reservation_required"
			case "wrong holder":
				o.Reservations[1].Holder = "aurora/another"
				want = "star_reservation_conflict"
			}
			if kind == "invalid source" || kind == "output only" {
				sum := sha256.Sum256(o.Sources[0].Bytes)
				hash := hex.EncodeToString(sum[:])
				o.Sources[0].SHA256 = hash
				for _, v := range list(c["members"]) {
					identity := obj(obj(v)["identity"])
					if identity["scheme"] == "legacy-exact" {
						obj(identity["source_ref"])["sha256"] = hash
					}
				}
			}
			e := Validate(encoded(t, c), o)
			if e == nil || e.Rule != want {
				t.Fatalf("expected %s got %v", want, e)
			}
		})
	}
}

// A pinned service source and an equal target MXID still need the selected
// connection's exact server. Rehashing the source cannot attest that agreement.
// Existing target/source and source/connection refusal precedence is preserved.
func TestLegacyServiceServerAgreement(t *testing.T) {
	for _, tc := range []struct{ name, server, target, homeserver, rule, location string }{
		{"matching rehashed source", "relux.works", "relux.works", "https://matrix.relux.works", "", ""},
		{"foreign server", "other.example", "other.example", "https://matrix.relux.works", "legacy_source_connection_mismatch", "$.members[4].identity.mxid"},
		{"server suffix", "relux.works.other.example", "relux.works.other.example", "https://matrix.relux.works", "legacy_source_connection_mismatch", "$.members[4].identity.mxid"},
		{"server subdomain", "sub.relux.works", "sub.relux.works", "https://matrix.relux.works", "legacy_source_connection_mismatch", "$.members[4].identity.mxid"},
		{"target differs", "other.example", "relux.works", "https://matrix.relux.works", "legacy_source_mismatch", "$.members[4].identity.mxid"},
		{"source connection differs", "relux.works", "relux.works", "https://other.example", "legacy_source_connection_mismatch", "$.members[4].identity.source_ref"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, o := fixture(t, "services-v4.json"), fixtureOptions(t)
			if e := Validate(encoded(t, c), o); e != nil {
				t.Fatal(e)
			}
			source := fixture(t, "legacy-service-source.json")
			source["mxid"] = "@station_signal_service:" + tc.server
			obj(source["connection"])["homeserver"] = tc.homeserver
			raw := encoded(t, source)
			sum := sha256.Sum256(raw)
			hash := hex.EncodeToString(sum[:])
			found := false
			for i := range o.Sources {
				if o.Sources[i].Kind == "service-descriptor" {
					o.Sources[i].Bytes, o.Sources[i].SHA256 = raw, hash
					found = true
				}
			}
			if !found {
				t.Fatal("service source mutation not applied")
			}
			identity := obj(obj(list(c["members"])[4])["identity"])
			obj(identity["source_ref"])["sha256"] = hash
			identity["mxid"] = "@station_signal_service:" + tc.target
			e := Validate(encoded(t, c), o)
			if tc.rule == "" {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil || e.Rule != tc.rule || e.Location != tc.location || e.Exit != 1 {
				t.Fatalf("expected %s at %s, exit1; got %v", tc.rule, tc.location, e)
			}
		})
	}
}

// Comparators operate on arbitrary-length decimal components, without an
// undocumented machine-integer limit or numeric wraparound.
func TestVersionGrammar(t *testing.T) {
	for _, c := range []struct {
		v, r string
		yes  bool
	}{{"0.1.0", ">=0.1.0 <0.2.0", true}, {"0.2.0", ">=0.1.0 <0.2.0", false}, {"1.2.3", "=1.2.3", true}, {"1.2.3", ">1.2.3", false}, {"1.2.3", "<=1.2.3", true}, {"1.2.3", "<1.2.4", true}, {"1.2.3", ">1.2.2", true}, {"999999999999999999999999.0.0", ">999999999999999999999998.0.0", true}, {"1.2.3", "latest", false}, {"1.2.3", "=1.2", false}} {
		if compatible(c.v, c.r) != c.yes {
			t.Fatal("version comparison contract")
		}
	}
}

// V3 has its own documented text/endpoint/version grammar, rather than silently
// inheriting stricter new tuple/DNS/repository spellings.
func TestV3LegacyGrammar(t *testing.T) {
	c := fixture(t, "source-v3.json")
	obj(obj(c["connections"])["legacy-relux"])["homeserver"] = "https://localhost/"
	obj(obj(c["profiles"])["control"])["registry"] = Object{"repo": "legacy-registry", "directory": "control"}
	delete(obj(obj(c["profiles"])["control"]), "path")
	obj(obj(obj(c["profiles"])["control"])["selected"])["version"] = "00.01.00"
	if e := Validate(encoded(t, c), Options{Version: 3}); e != nil {
		t.Fatal(e)
	}
}

// Sessionless service branches reject conversational facts and route crossover;
// neither acceptance nor refusal invokes a real agent/session launcher.
func TestServiceBranches(t *testing.T) {
	for _, c := range []struct {
		name, want string
		mutate     func(Object)
	}{
		{"missing native reference", "native_result_required", func(c Object) { delete(obj(list(c["members"])[5]), "native_result_ref") }},
		{"native policy", "parent_native_selector_policy_forbidden", func(c Object) { obj(list(c["members"])[5])["selector_policy_ref"] = "synthetic-policy" }},
		{"chat native reference", "service_native_result_forbidden", func(c Object) { obj(list(c["members"])[4])["native_result_ref"] = "synthetic-native" }},
		{"context", "enum_value", func(c Object) { obj(list(c["members"])[5])["context_mode"] = "continue" }},
		{"runtime effort", "service_runtime_forbidden", func(c Object) { obj(obj(list(c["members"])[5])["runtime"])["effort"] = "high" }},
		{"chat admission field", "service_chat_admission_forbidden", func(c Object) { obj(list(c["members"])[4])["chat_admission"] = "synthetic-admission" }},
	} {
		t.Run(c.name, func(t *testing.T) {
			card := fixture(t, "services-v4.json")
			o := fixtureOptions(t)
			if e := Validate(encoded(t, card), o); e != nil {
				t.Fatal(e)
			}
			c.mutate(card)
			e := Validate(encoded(t, card), o)
			if e == nil || e.Rule != c.want {
				t.Fatalf("expected %s got %v", c.want, e)
			}
		})
	}
}

// Additional syntax failures return no partially decoded document. Equal nested
// duplicate keys are refused just like unequal top-level duplicates.
func TestParserRefusals(t *testing.T) {
	cases := []struct{ name, raw, want string }{
		{"nested duplicate", "{\"outer\":{\"x\":1,\"x\":1}}", "duplicate_key"},
		{"non-string key", "{1: value}", "non_string_key"},
		{"explicit standard tag", "{x: !!str value}", "explicit_tag"},
		{"bad utf8", string([]byte{'x', ':', ' ', 0xff}), "invalid_utf8"},
		{"float", "{x: 1.5}", "scalar_type"},
		{"empty", "", "parse_document"},
		{"root array", "[]", "parse_document"},
		{"null", "{x: null}", "null_value"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc, e := Parse([]byte(c.raw), 4)
			if doc != nil || e == nil || e.Rule != c.want {
				t.Fatalf("expected %s got %v", c.want, e)
			}
		})
	}
}

// Malicious values and keys never become a diagnostic location, even when their
// ordinary collection position has to be reported.
func TestSafeDiagnostics(t *testing.T) {
	c := fixture(t, "converted-v4.json")
	obj(list(c["members"])[1])["SYNTHETIC-CANARY\nVALUE"] = true
	e := Validate(encoded(t, c), fixtureOptions(t))
	if e == nil || strings.Contains(e.Error(), "CANARY") || strings.Contains(e.Error(), "VALUE") {
		t.Fatal("key leaked")
	}
	c = fixture(t, "converted-v4.json")
	obj(list(c["members"])[3])["profile"] = "SYNTHETIC-CANARY\nVALUE"
	e = Validate(encoded(t, c), fixtureOptions(t))
	if e == nil || strings.Contains(e.Error(), "CANARY") {
		t.Fatal("value leaked")
	}
}

// V3 explicitly retires workspace while leaving unrecognized, nonsecret source
// extensions available to the later converter's separate lossless gate.
func TestV3RetiredWorkspace(t *testing.T) {
	c := fixture(t, "source-v3.json")
	obj(obj(list(c["members"])[1])["a2a"])["workspace"] = "old-workspace"
	e := Validate(encoded(t, c), Options{Version: 3})
	if e == nil || e.Rule != "R13" {
		t.Fatal("retired field accepted", e)
	}
}

// A profile link escaping its narrower profiles_root refuses before inspecting
// an existing destination even when that destination is inside the product root.
func TestProfileBoundary(t *testing.T) {
	setup := Object{"denied_inspections": []any{"elsewhere"}, "admitted_root": "sandbox/product", "card_directory": "sandbox/product/aurora-host", "directories": []any{"sandbox/product/aurora-host", "sandbox/product/beacon", "sandbox/product/vault", "sandbox/product/profiles", "sandbox/product/elsewhere", "sandbox/outside"}, "files": []any{}, "symlinks": []any{Object{"path": "sandbox/product/profiles/control", "target": "../elsewhere"}}}
	e := executeLayout(t, fixture(t, "converted-v4.json"), setup, nil)
	if e == nil || e.Rule != "path_symlink_escape" || e.Location != "$.profiles[0].path" {
		t.Fatal("profile boundary", e)
	}
}

// Comparison cannot drop a repeated legacy project, change a source pin or
// silently preserve an unsupported extension by omitting its representation.
func TestPreservationCompleteness(t *testing.T) {
	var in PreservationInputs
	if e := json.Unmarshal(fixtureBytes(t, "expectations.json"), &in); e != nil {
		t.Fatal(e)
	}
	for _, test := range []struct {
		name, want string
		mutate     func(Object, Object)
	}{
		{"source project multiplicity", "conversion_project_preservation", func(s, t Object) { s["projects"] = append(list(s["projects"]), list(s["projects"])[1]) }},
		{"source pin", "conversion_source_pin_preservation", func(s, t Object) {
			obj(obj(obj(list(t["members"])[1])["identity"])["source_ref"])["revision"] = "other-valid-source"
		}},
		{"source extension", "conversion_extension_unsupported", func(s, t Object) { s["extension"] = Object{"desired": "preserve"} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, target := fixture(t, "source-v3.json"), fixture(t, "converted-v4.json")
			if e := VerifyConversionPreservation(s, target, in); e != nil {
				t.Fatal(e)
			}
			test.mutate(s, target)
			e := VerifyConversionPreservation(s, target, in)
			if e == nil || e.Rule != test.want {
				t.Fatalf("expected %s got %v", test.want, e)
			}
		})
	}
}
