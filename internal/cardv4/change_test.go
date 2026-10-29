package cardv4

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func pinned(raw []byte) Snapshot {
	sum := sha256.Sum256(raw)
	return Snapshot{Bytes: raw, SHA256: hex.EncodeToString(sum[:])}
}

// Both complete cutover pairs pass, including an opaque retained logical key.
// An unchanged candidate is preservation-valid, not evidence of a cutover.
func TestCheckChangeCompletePairs(t *testing.T) {
	for _, tc := range []struct{ prior, candidate, premises, pilot string }{
		{"converted-v4.json", "pilot-v4.json", "expectations.json", "aurora/beacon_orch"},
		{"compat-id-converted-v4.json", "compat-id-pilot-v4.json", "compat-id-expectations.json", "aurora/beacon=orch"},
	} {
		t.Run(tc.pilot, func(t *testing.T) {
			prior, candidate := pinned(fixtureBytes(t, tc.prior)), pinned(fixtureBytes(t, tc.candidate))
			o := fixtureOptions(t)
			decodeReservations(t, fixture(t, tc.premises)["reviewed_reservation_premises"], &o)
			before := encoded(t, []any{prior, candidate, o})
			for _, target := range []Snapshot{candidate, prior} {
				if e := CheckChange(prior, target, prior, tc.pilot, o); e != nil {
					t.Fatal(e)
				}
			}
			if !bytes.Equal(before, encoded(t, []any{prior, candidate, o})) {
				t.Fatal("checker mutated supplied bytes or premises")
			}
		})
	}
}

// These standalone-valid mutations must fail preservation, rather than a schema
// or source setup error. Each case first passes its unchanged valid neighbor.
func TestCheckChangePreservationRefusals(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(Object)
	}{
		{"nonpilot desired state", func(c Object) { obj(list(c["members"])[2])["desired_state"] = "parked" }},
		{"pilot station", func(c Object) { obj(list(c["members"])[1])["station"] = "other-station" }},
		{"pilot model", func(c Object) { obj(obj(list(c["members"])[1])["runtime"])["model"] = "other-model" }},
		{"pilot desired state", func(c Object) { obj(list(c["members"])[1])["desired_state"] = "parked" }},
		{"root fact", func(c Object) { obj(c["product"])["description"] = "Other desired description" }},
		{"project fact", func(c Object) { obj(list(c["projects"])[1])["description"] = "Other project description" }},
		{"selector multiplicity", func(c Object) {
			s := obj(obj(obj(list(c["members"])[0])["a2a"])["subscriptions"])
			s["required"] = append(list(s["required"]), list(s["required"])[0])
		}},
		{"selector ownership", func(c Object) {
			s := obj(obj(obj(list(c["members"])[0])["a2a"])["subscriptions"])
			s["self"] = append(list(s["self"]), list(s["required"])[0])
			s["required"] = list(s["required"])[1:]
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prior, rollback := pinned(fixtureBytes(t, "converted-v4.json")), pinned(fixtureBytes(t, "rollback-v4.json"))
			o := fixtureOptions(t)
			c := fixture(t, "pilot-v4.json")
			if e := CheckChange(prior, pinned(encoded(t, c)), rollback, "aurora/beacon_orch", o); e != nil {
				t.Fatal(e)
			}
			tc.mutate(c)
			raw := encoded(t, c)
			if e := Validate(raw, o); e != nil {
				t.Fatal("counterexample is not standalone-valid", e)
			}
			if e := CheckChange(prior, pinned(raw), rollback, "aurora/beacon_orch", o); e == nil || e.Rule != "nonpilot_preservation" {
				t.Fatal("preservation refusal", e)
			}
		})
	}
}

// A different valid profile or source pin cannot hide inside the allowed pilot
// identity/binding/a2a delta; opaque keys and their actual source remain exact.
func TestCheckChangeProfileAndLogicalPin(t *testing.T) {
	for _, logical := range []bool{false, true} {
		prior, target := fixture(t, "compat-id-converted-v4.json"), fixture(t, "compat-id-pilot-v4.json")
		o := fixtureOptions(t)
		decodeReservations(t, fixture(t, "compat-id-expectations.json")["reviewed_reservation_premises"], &o)
		if !logical {
			for _, c := range []Object{prior, target} {
				obj(c["profiles"])["alternative"] = copyObject(obj(obj(c["profiles"])["control"]))
			}
		}
		p := pinned(encoded(t, prior))
		if e := CheckChange(p, pinned(encoded(t, target)), p, "aurora/beacon=orch", o); e != nil {
			t.Fatal(e)
		}
		m := obj(list(target["members"])[1])
		if logical {
			pin := obj(m["logical_source_ref"])
			old := str(pin["revision"])
			pin["revision"] = "fixture-alternate-001"
			for _, s := range o.Sources {
				if s.Revision == old {
					s.Revision = str(pin["revision"])
					o.Sources = append(o.Sources, s)
					break
				}
			}
		} else {
			m["profile"] = "alternative"
		}
		raw := encoded(t, target)
		if e := Validate(raw, o); e != nil {
			t.Fatal("counterexample standalone validation", e)
		}
		if e := CheckChange(p, pinned(raw), p, "aurora/beacon=orch", o); e == nil || e.Rule != "nonpilot_preservation" {
			t.Fatal("pilot fact changed", e)
		}
	}
}

// Invalid pins, missing authority and invalid documents refuse the whole result.
// Rollback requires exact prior bytes even when its decoded data is equal.
func TestCheckChangeInputRefusals(t *testing.T) {
	for _, name := range []string{"missing pin", "false pin", "missing pilot", "wrong pilot", "missing source", "changed source bytes", "missing reservation", "malformed prior", "malformed candidate", "malformed rollback", "oversized", "rollback formatting", "rollback is candidate"} {
		t.Run(name, func(t *testing.T) {
			p, c, r := pinned(fixtureBytes(t, "converted-v4.json")), pinned(fixtureBytes(t, "pilot-v4.json")), pinned(fixtureBytes(t, "rollback-v4.json"))
			o, pilot := fixtureOptions(t), "aurora/beacon_orch"
			if e := CheckChange(p, c, r, pilot, o); e != nil {
				t.Fatal(e)
			}
			want := "pilot_member_missing"
			switch name {
			case "missing pin":
				c.SHA256 = ""
				want = "snapshot_pin_invalid"
			case "false pin":
				c.SHA256 = strings.Repeat("a", 64)
				want = "snapshot_digest_mismatch"
			case "missing pilot":
				pilot = ""
			case "wrong pilot":
				pilot = "aurora/storage_orch"
				want = "nonpilot_preservation"
			case "missing source":
				o.Sources = nil
				want = "legacy_source_missing"
			case "changed source bytes":
				o.Sources[0].Bytes = append(append([]byte{}, o.Sources[0].Bytes...), ' ')
				want = "source_digest_mismatch"
			case "missing reservation":
				o.Reservations = nil
				want = "star_reservation_required"
			case "malformed prior":
				p = pinned([]byte("SYNTHETIC-CANARY: ["))
				want = "parse_document"
			case "malformed candidate":
				c = pinned([]byte("SYNTHETIC-CANARY: ["))
				want = "parse_document"
			case "malformed rollback":
				r = pinned([]byte("SYNTHETIC-CANARY: ["))
				want = "parse_document"
			case "oversized":
				c = pinned(bytes.Repeat([]byte(" "), MaxBytes+1))
				want = "raw_byte_limit"
			case "rollback formatting":
				r = pinned(append(append([]byte{}, p.Bytes...), '\n'))
				want = "rollback_bytes_mismatch"
			case "rollback is candidate":
				r = c
				want = "rollback_bytes_mismatch"
			}
			before := encoded(t, []any{p, c, r, o})
			if e := CheckChange(p, c, r, pilot, o); e == nil || e.Rule != want {
				t.Fatalf("want %s, got %v", want, e)
			}
			if !reflect.DeepEqual(before, encoded(t, []any{p, c, r, o})) {
				t.Fatal("refusal mutated inputs")
			}
		})
	}
}

// This independently valid second product must refuse the selected first-product
// pilot, even when every snapshot pin and both products' reservations are valid.
func TestCheckChangeDifferentProduct(t *testing.T) {
	p := pinned(fixtureBytes(t, "converted-v4.json"))
	c := fixture(t, "pilot-v4.json")
	o := fixtureOptions(t)
	obj(c["product"])["id"] = "borealis"
	for i, v := range list(c["members"]) {
		m := obj(v)
		star := []string{"rigel", "vega", "sirius", "deneb"}[i]
		m["identity"] = Object{"scheme": "domain-star-v1", "star": star}
		m["binding"] = "gr"
		if a := obj(m["a2a"]); a != nil {
			a["connection"] = "target-gr"
		}
		o.Reservations = append(o.Reservations, Reservation{Holder: "borealis/" + str(m["id"]), Domain: "gr.relux.works", Product: "borealis", PrimaryRole: str(m["role"]), Star: star})
	}
	raw := encoded(t, c)
	if e := Validate(raw, o); e != nil {
		t.Fatal("different-product control is invalid", e)
	}
	if e := CheckChange(p, pinned(fixtureBytes(t, "pilot-v4.json")), p, "aurora/beacon_orch", o); e != nil {
		t.Fatal(e)
	}
	if e := CheckChange(p, pinned(raw), p, "aurora/beacon_orch", o); e == nil || e.Rule != "pilot_member_missing" {
		t.Fatal("foreign product admitted", e)
	}
}

// Rehashing an invalid v3 reporting line and all corresponding snapshot refs
// cannot turn it into valid legacy evidence for the change checker.
func TestCheckChangeInvalidLegacySource(t *testing.T) {
	o := fixtureOptions(t)
	p, c, r := fixture(t, "converted-v4.json"), fixture(t, "pilot-v4.json"), fixture(t, "rollback-v4.json")
	if e := CheckChange(pinned(encoded(t, p)), pinned(encoded(t, c)), pinned(encoded(t, r)), "aurora/beacon_orch", o); e != nil {
		t.Fatal(e)
	}
	source := fixture(t, "source-v3.json")
	obj(list(source["members"])[3])["parent"] = "missing"
	raw := encoded(t, source)
	sourcePin := pinned(raw)
	o.Sources[0].Bytes = raw
	o.Sources[0].SHA256 = sourcePin.SHA256
	for _, doc := range []Object{p, c, r} {
		for _, v := range list(doc["members"]) {
			identity := obj(obj(v)["identity"])
			if identity["scheme"] == "legacy-exact" {
				obj(identity["source_ref"])["sha256"] = sourcePin.SHA256
			}
		}
	}
	if e := CheckChange(pinned(encoded(t, p)), pinned(encoded(t, c)), pinned(encoded(t, r)), "aurora/beacon_orch", o); e == nil || e.Rule != "legacy_source_invalid" {
		t.Fatal("invalid legacy source admitted", e)
	}
}

// The existing unit-2 source/API recorder gates the actual pure checker. Its
// one-call plant is in the actual scanned CLI source, never compiled/executed.
// Both zero and one controls retain the exact isolated input inventory.
func TestCheckChangeRegistrationAdmission(t *testing.T) {
	root := t.TempDir()
	for _, file := range []string{"converted-v4.json", "pilot-v4.json", "rollback-v4.json"} {
		if e := os.WriteFile(filepath.Join(root, file), fixtureBytes(t, file), 0600); e != nil {
			t.Fatal(e)
		}
	}
	before := inventory(t, root)
	for _, plant := range []string{"", "../../cmd/card-v4-check-change/main.go"} {
		boundary := &registrationAdmission{}
		entered := false
		issue := boundary.run(t, plant, func() *Issue {
			entered = true
			return CheckChange(pinned(fixtureBytes(t, "converted-v4.json")), pinned(fixtureBytes(t, "pilot-v4.json")), pinned(fixtureBytes(t, "rollback-v4.json")), "aurora/beacon_orch", fixtureOptions(t))
		})
		if plant == "" {
			if issue != nil || !entered || boundary.attempts != 0 {
				t.Fatal("unapplied checker instrument", issue)
			}
		} else if issue == nil || issue.Rule != "forbidden_registration_attempt" || issue.Location != "$" || issue.Exit != 1 || entered || boundary.attempts != 1 {
			t.Fatal("checker plant not denied", issue)
		}
		if !reflect.DeepEqual(before, inventory(t, root)) {
			t.Fatal("checker changed isolated inputs")
		}
	}
}
