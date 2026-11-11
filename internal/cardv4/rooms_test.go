package cardv4

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"
)

func addRoomDeclarations(c Object) {
	obj(c["product"])["rooms"] = []any{
		Object{"role": "space", "server_name": "relux.works", "alias": "#space:foreign.example", "room_id": "!AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"},
		Object{"role": "lobby", "server_name": "relux.works", "alias": "#lobby:foreign.example"},
		Object{"role": "escalations", "server_name": "relux.works", "room_id": "!escalationsRoom:remote.example"},
	}
	for i, value := range list(c["projects"]) {
		obj(value)["rooms"] = []any{Object{"role": "project_room", "server_name": "relux.works", "alias": fmt.Sprintf("#project-%d:foreign.example", i), "room_id": fmt.Sprintf("!projectRoom%02d:remote.example", i)}}
	}
}

// Both validators accept absent legacy metadata and complete declarations with
// opaque/foreign/alias-only/ID-only routes. Invalid neighbors reject at the new
// invariant or preceding parse/shape rule without mutating input. Local snapshots share that invariant.
func TestOptionalRoomDeclarations(t *testing.T) {
	for _, version := range []int{3, 4} {
		t.Run(fmt.Sprintf("v%d", version), func(t *testing.T) {
			file := "source-v3.json"
			o := Options{Version: version}
			if version == 4 {
				file, o = "converted-v4.json", fixtureOptions(t)
			}
			legacy := fixtureBytes(t, file)
			if e := Validate(legacy, o); e != nil {
				t.Fatal("legacy positive", e)
			}
			makeCard := func() Object {
				c := fixture(t, file)
				addRoomDeclarations(c)
				return c
			}
			valid := encoded(t, makeCard())
			if e := Validate(valid, o); e != nil {
				t.Fatal("amended positive", e)
			}
			if version == 4 {
				if _, e := ParseLocalSnapshot(valid); e != nil {
					t.Fatal(e)
				}
			}
			for _, tc := range []struct {
				name string
				edit func(Object)
			}{
				{"missing-product-set", func(c Object) { delete(obj(c["product"]), "rooms") }},
				{"missing-project-set", func(c Object) { delete(obj(list(c["projects"])[1]), "rooms") }},
				{"empty-set", func(c Object) { obj(c["product"])["rooms"] = []any{} }},
				{"null-set", func(c Object) { obj(c["product"])["rooms"] = nil }},
				{"missing-role", func(c Object) { delete(obj(list(obj(c["product"])["rooms"])[0]), "role") }},
				{"unknown-field", func(c Object) { obj(list(obj(c["product"])["rooms"])[0])["ignored"] = true }},
				{"wrong-role", func(c Object) { obj(list(obj(c["product"])["rooms"])[0])["role"] = "project_room" }},
				{"missing-reference", func(c Object) {
					m := obj(list(obj(c["product"])["rooms"])[0])
					delete(m, "alias")
					delete(m, "room_id")
				}},
				{"null-reference", func(c Object) { obj(list(obj(c["product"])["rooms"])[0])["alias"] = nil }},
				{"unqualified-alias", func(c Object) { obj(list(obj(c["product"])["rooms"])[0])["alias"] = "#local" }},
				{"malformed-id", func(c Object) { obj(list(obj(c["product"])["rooms"])[0])["room_id"] = "!tiny" }},
				{"malformed-server", func(c Object) { obj(list(obj(c["product"])["rooms"])[0])["server_name"] = "bad server" }},
				{"wrong-routing-server", func(c Object) { obj(list(obj(c["product"])["rooms"])[0])["server_name"] = "other.example" }},
				{"duplicate-role", func(c Object) {
					obj(c["product"])["rooms"] = append(list(obj(c["product"])["rooms"]), list(obj(c["product"])["rooms"])[0])
				}},
				{"duplicate-cross-scope-id", func(c Object) {
					obj(list(obj(list(c["projects"])[0])["rooms"])[0])["room_id"] = "!AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
				}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					c := makeCard()
					tc.edit(c)
					raw := encoded(t, c)
					before := append([]byte(nil), raw...)
					e := Validate(raw, o)
					// Existing parse/shape diagnostics precede the new semantic rule.
					want := "room_declaration"
					shapeRules := map[string]string{"empty-set": "required_nonempty", "null-set": "array_shape", "missing-role": "missing_field", "unknown-field": "unknown_field", "wrong-role": "enum_value"}
					if rule := shapeRules[tc.name]; rule != "" {
						want = rule
						if version == 3 {
							want = "R13"
							if tc.name == "wrong-role" {
								want = "R11"
							}
						}
					}
					if version == 4 && (tc.name == "null-set" || tc.name == "null-reference") {
						want = "null_value"
					}
					if e == nil || e.Rule != want {
						t.Fatalf("want %s, got %v", want, e)
					}
					if !bytes.Equal(raw, before) {
						t.Fatal("source mutated")
					}
					if version == 4 {
						if _, e := ParseLocalSnapshot(raw); e == nil || e.Rule != want {
							t.Fatal("snapshot bypass", e)
						}
					}
				})
			}
		})
	}
}

// Conversion preserves the full room set. The independent preservation check
// rejects a dropped product set or modified project route beside a valid result.
func TestRoomConversionPreservation(t *testing.T) {
	c := fixture(t, "source-v3.json")
	addRoomDeclarations(c)
	raw := encoded(t, c)
	in := conversionInputs(t, "expectations.json")
	sum := sha256.Sum256(raw)
	in.SourceSHA256 = hex.EncodeToString(sum[:])
	out, e := Convert(raw, in)
	if e != nil {
		t.Fatal(e)
	}
	target, e := Parse(out, 4)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(obj(c["product"])["rooms"], obj(target["product"])["rooms"]) {
		t.Fatal("product room set lost")
	}
	for i, p := range list(c["projects"]) {
		if !reflect.DeepEqual(obj(p)["rooms"], obj(list(target["projects"])[i])["rooms"]) {
			t.Fatal("project room set lost")
		}
	}
	if e := VerifyConversionPreservation(c, target, in.PreservationInputs); e != nil {
		t.Fatal(e)
	}
	delete(obj(target["product"]), "rooms")
	if e := VerifyConversionPreservation(c, target, in.PreservationInputs); e == nil || e.Rule != "conversion_product_preservation" {
		t.Fatal(e)
	}
	target, _ = Parse(out, 4)
	obj(list(obj(list(target["projects"])[0])["rooms"])[0])["alias"] = "#changed:foreign.example"
	if e := VerifyConversionPreservation(c, target, in.PreservationInputs); e == nil || e.Rule != "conversion_project_preservation" {
		t.Fatal(e)
	}
}

// A connection may route to an independently hosted alias. Matching selected
// routing servers pass; another declared but unselected server still refuses.
// Reusing an exact ID for different roles inside one scope remains permitted.
func TestRoomRoutingAndSameScopeReuse(t *testing.T) {
	c := fixture(t, "source-v3.json")
	addRoomDeclarations(c)
	obj(c["connections"])["federated"] = Object{"homeserver": "https://gr.example", "server_name": "gr.example"}
	obj(c["bindings"])["federated"] = Object{"operator": "fixture", "host": "fixture", "identity_prefix": "remote_", "connection": "federated"}
	m := obj(list(c["members"])[1])
	m["binding"] = "federated"
	obj(m["a2a"])["connection"] = "federated"
	var r Object
	for _, p := range list(c["projects"]) {
		if obj(p)["orchestrator"] == m["id"] {
			r = obj(list(obj(p)["rooms"])[0])
			break
		}
	}
	if r == nil {
		t.Fatal("missing manager project")
	}
	r["server_name"], r["alias"] = "gr.example", "#project:relux.works"
	productRooms := list(obj(c["product"])["rooms"])
	obj(productRooms[1])["room_id"] = obj(productRooms[0])["room_id"]
	if e := Validate(encoded(t, c), Options{Version: 3}); e != nil {
		t.Fatal("federated positive", e)
	}
	r["server_name"] = "relux.works"
	if e := Validate(encoded(t, c), Options{Version: 3}); e == nil || e.Rule != "room_declaration" {
		t.Fatal("wrong selected routing server", e)
	}
}
