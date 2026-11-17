package cardv4

import (
	"bytes"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/oasisprotocol/curve25519-voi/curve"
)

var governanceNow = time.Date(2026, 11, 17, 16, 0, 0, 0, time.UTC)

func governanceFixture(t *testing.T) Object {
	t.Helper()
	raw, err := os.ReadFile("../../docs/fixtures/v5/governance-v5.json")
	if err != nil {
		t.Fatal(err)
	}
	c, e := Parse(raw, 5)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func governanceKey(c Object, n int) Object {
	return obj(list(obj(list(c["principals"])[0])["keys"])[n])
}
func setPublicBytes(k Object, raw []byte) {
	value := base64.RawURLEncoding.EncodeToString(raw)
	prefix := map[string]string{"Ed25519": "ed25519:", "P-256": "p256:", "BIP-340": "bip340:"}[str(k["algorithm"])]
	sum := sha256.Sum256(raw)
	k["public_bytes"], k["key_id"], k["public_sha256"] = value, prefix+value, hex.EncodeToString(sum[:])
}

// The public fixture and explicit legal variants satisfy local declaration
// checks. This establishes neither possession nor external-source admission.
func TestGovernanceV5Valid(t *testing.T) {
	cases := map[string]func(Object){
		"fixture":    func(Object) {},
		"undeclared": func(c Object) { delete(obj(c["product"]), "governance") },
		"empty-keys": func(c Object) {
			for _, p := range list(c["principals"]) {
				obj(p)["keys"] = []any{}
			}
		},
		"local-meta":    func(c Object) { obj(obj(c["product"])["governance"])["meta_keeper"] = "beacon_orch" },
		"free-title":    func(c Object) { obj(list(c["members"])[0])["role"] = "Human-readable product lead" },
		"p256-identity": func(c Object) { governanceKey(c, 1)["purpose"] = "identity" },
		"same-id-separate-namespaces": func(c Object) {
			obj(list(c["principals"])[1])["id"] = "beacon_orch"
			obj(obj(c["product"])["governance"])["owners"] = []any{"example-owner", "beacon_orch"}
			obj(list(c["members"])[4])["principal"] = "beacon_orch"
		},
		"retired": func(c Object) { governanceKey(c, 0)["declared_state"] = "retired" },
		"revoked": func(c Object) { governanceKey(c, 0)["declared_state"] = "revoked" },
		"validity-window": func(c Object) {
			k := governanceKey(c, 0)
			k["not_before"] = "2026-11-17T00:00:00Z"
			k["not_after"] = "2026-11-18T00:00:00Z"
		},
		"two-purposes": func(c Object) {
			p := obj(list(c["principals"])[0])
			k := Object{}
			for a, b := range governanceKey(c, 0) {
				k[a] = b
			}
			k["purpose"] = "grant-authority"
			p["keys"] = append(list(p["keys"]), k)
		},
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			c := governanceFixture(t)
			edit(c)
			if _, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow); e != nil {
				t.Fatal(e)
			}
		})
	}
}

// Mutations of otherwise valid cards must fail with the intended contract
// rule, including invalid records that are not selected as owners or callers.
func TestGovernanceV5Rejects(t *testing.T) {
	cases := []struct {
		name, rule string
		edit       func(Object)
	}{
		{"missing-principals", "missing_field", func(c Object) { delete(c, "principals") }},
		{"missing-member-kind", "missing_field", func(c Object) { delete(obj(list(c["members"])[1]), "kind") }},
		{"unknown-member-kind", "enum_value", func(c Object) { obj(list(c["members"])[1])["kind"] = "machine" }},
		{"unknown-root", "unknown_field", func(c Object) { c["station_registry"] = Object{} }},
		{"unknown-key", "unknown_field", func(c Object) { governanceKey(c, 0)["proof"] = "untrusted" }},
		{"duplicate-principal", "duplicate_principal", func(c Object) { c["principals"] = append(list(c["principals"]), list(c["principals"])[0]) }},
		{"duplicate-owner", "duplicate_owner", func(c Object) {
			obj(obj(c["product"])["governance"])["owners"] = []any{"example-owner", "example-owner"}
		}},
		{"empty-owners", "required_nonempty", func(c Object) { obj(obj(c["product"])["governance"])["owners"] = []any{} }},
		{"unknown-owner", "principal_reference", func(c Object) { obj(obj(c["product"])["governance"])["owners"] = []any{"absent"} }},
		{"unknown-meta", "meta_keeper_reference", func(c Object) { obj(obj(c["product"])["governance"])["meta_keeper"] = "absent" }},
		{"unlisted-meta", "meta_keeper_reference", func(c Object) {
			obj(obj(c["product"])["governance"])["meta_keeper"] = Object{"product": "outside", "member": "absent"}
		}},
		{"duplicate-key", "duplicate_public_key", func(c Object) {
			p := obj(list(c["principals"])[0])
			p["keys"] = append(list(p["keys"]), governanceKey(c, 0))
		}},
		{"duplicate-contact", "duplicate_contact", func(c Object) {
			obj(list(c["principals"])[1])["contacts"] = []any{Object{"kind": "matrix", "mxid": "@owner:example.org"}, Object{"kind": "matrix", "mxid": "@owner:example.org"}}
		}},
		{"agent-principal", "unknown_field", func(c Object) { obj(list(c["members"])[0])["principal"] = "example-owner" }},
		{"human-runtime", "unknown_field", func(c Object) { obj(list(c["members"])[4])["runtime"] = Object{"kind": "none"} }},
		{"human-service", "enum_value", func(c Object) { obj(list(c["members"])[4])["responsibility"] = "service" }},
		{"human-principal", "principal_reference", func(c Object) { obj(list(c["members"])[4])["principal"] = "absent" }},
		{"title-not-privilege", "reporting_line", func(c Object) {
			m := obj(list(c["members"])[0])
			m["role"] = "keeper"
			m["responsibility"] = "developer"
		}},
		{"title-control-character", "title_format", func(c Object) { obj(list(c["members"])[0])["role"] = "invalid\x00title" }},
		{"bip-authority", "public_key_combination", func(c Object) { governanceKey(c, 2)["purpose"] = "grant-authority" }},
		{"algorithm-case", "enum_value", func(c Object) { governanceKey(c, 0)["algorithm"] = "ed25519" }},
		{"encoding-mismatch", "public_key_combination", func(c Object) { governanceKey(c, 0)["encoding"] = "sec1-compressed" }},
		{"padding", "public_key_encoding", func(c Object) { governanceKey(c, 0)["public_bytes"] = str(governanceKey(c, 0)["public_bytes"]) + "=" }},
		{"id-mismatch", "public_key_id", func(c Object) { governanceKey(c, 0)["key_id"] = "ed25519:wrong" }},
		{"digest-mismatch", "public_key_digest", func(c Object) { governanceKey(c, 0)["public_sha256"] = strings.Repeat("0", 64) }},
		{"live-status", "live_state_forbidden", func(c Object) { governanceKey(c, 0)["status"] = "active" }},
		{"secret-field", "credential_field_forbidden", func(c Object) { governanceKey(c, 0)["private_key"] = "placeholder" }},
		{"future-state", "declared_state_future", func(c Object) { governanceKey(c, 0)["declared_state_at"] = "2026-11-18T00:00:00Z" }},
		{"fraction-time", "utc_second", func(c Object) { governanceKey(c, 0)["declared_state_at"] = "2026-11-17T00:00:00.0Z" }},
		{"offset-time", "utc_second", func(c Object) { governanceKey(c, 0)["declared_state_at"] = "2026-11-17T00:00:00+00:00" }},
		{"invalid-date", "utc_second", func(c Object) { governanceKey(c, 0)["declared_state_at"] = "2026-02-30T00:00:00Z" }},
		{"equal-window", "public_key_validity", func(c Object) {
			k := governanceKey(c, 0)
			k["not_before"] = "2026-11-17T00:00:00Z"
			k["not_after"] = k["not_before"]
		}},
		{"reversed-window", "public_key_validity", func(c Object) {
			k := governanceKey(c, 0)
			k["not_before"] = "2026-11-18T00:00:00Z"
			k["not_after"] = "2026-11-17T00:00:00Z"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := governanceFixture(t)
			tc.edit(c)
			_, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow)
			if e == nil || e.Rule != tc.rule {
				t.Fatalf("want %s, got %v", tc.rule, e)
			}
		})
	}
}

// Each title branch accepts printable text at 255 UTF-8 bytes and rejects 256
// bytes, including a pair with equal rune counts. Control characters are separate.
func TestGovernanceTitleByteBounds(t *testing.T) {
	for _, field := range []string{"agent-role", "human-role", "principal-name"} {
		for _, tc := range []struct {
			name, value string
			bytes       int
			valid       bool
		}{
			{"ascii-limit", strings.Repeat("a", 255), 255, true},
			{"ascii-over", strings.Repeat("a", 256), 256, false},
			{"utf8-limit", strings.Repeat("é", 127) + "a", 255, true},
			{"utf8-over", strings.Repeat("é", 128), 256, false},
		} {
			t.Run(field+"/"+tc.name, func(t *testing.T) {
				if len(tc.value) != tc.bytes || !text(tc.value) {
					t.Fatal("invalid printable byte-boundary control")
				}
				c := governanceFixture(t)
				switch field {
				case "agent-role":
					obj(list(c["members"])[1])["role"] = tc.value
				case "human-role":
					obj(list(c["members"])[4])["role"] = tc.value
				case "principal-name":
					obj(list(c["principals"])[0])["name"] = tc.value
				}
				_, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow)
				if tc.valid && e != nil {
					t.Fatal(e)
				}
				if !tc.valid && (e == nil || e.Rule != "title_format") {
					t.Fatalf("want title_format, got %v", e)
				}
			})
		}
	}
}

// Nonzero unused base64url bits refuse even when lax decoding yields the same
// valid public point; padding and point validity do not prove this property.
func TestGovernanceNonzeroTrailingBits(t *testing.T) {
	for _, n := range []int{0, 2} {
		c := governanceFixture(t)
		k := governanceKey(c, n)
		t.Run(str(k["algorithm"]), func(t *testing.T) {
			if _, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow); e != nil {
				t.Fatal(e)
			}
			canonical := str(k["public_bytes"])
			alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
			last := strings.IndexByte(alphabet, canonical[len(canonical)-1])
			if len(canonical)%4 != 3 || last < 0 || last%4 != 0 {
				t.Fatal("missing two unused trailing-bit control")
			}
			noncanonical := canonical[:len(canonical)-1] + string(alphabet[last|1])
			want, e1 := base64.RawURLEncoding.DecodeString(canonical)
			got, e2 := base64.RawURLEncoding.DecodeString(noncanonical)
			if e1 != nil || e2 != nil || !bytes.Equal(want, got) || noncanonical == canonical {
				t.Fatal("mutation does not isolate unused encoding bits")
			}
			k["public_bytes"] = noncanonical
			k["key_id"] = strings.TrimSuffix(str(k["key_id"]), canonical) + noncanonical
			if _, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow); e == nil || e.Rule != "public_key_encoding" {
				t.Fatalf("want public_key_encoding, got %v", e)
			}
		})
	}
}

// Independent public standards fixtures are accepted; neighboring malformed,
// noncanonical, identity and mixed-order points are rejected by their curve gate.
func TestGovernancePublicPoints(t *testing.T) {
	cases := []struct {
		name  string
		n     int
		hex   string
		valid bool
	}{
		{"ed-rfc8032", 0, "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a", true},
		{"ed-identity", 0, "0100000000000000000000000000000000000000000000000000000000000000", false},
		{"ed-order-four", 0, "0000000000000000000000000000000000000000000000000000000000000000", false},
		{"ed-noncanonical", 0, "eeffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f", false},
		{"ed-noncanonical-sign", 0, "0100000000000000000000000000000000000000000000000000000000000080", false},
		{"ed-short", 0, "01", false},
		{"p256-sec2", 1, "036b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c296", true},
		{"p256-uncompressed-tag", 1, "046b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c296", false},
		{"p256-overflow", 1, "02ffffffff00000001000000000000000000000000ffffffffffffffffffffffff", false},
		{"p256-infinity", 1, "00", false},
		// Go's compressed-point negative has an in-field x but no curve lift:
		// https://github.com/golang/go/blob/go1.25.14/src/crypto/elliptic/elliptic_test.go
		{"p256-in-range-no-lift", 1, "02fd4bf61763b46581fd9174d623516cf3c81edd40e29ffa2777fb6cb0ae3ce535", false},
		{"bip340-vector0", 2, "f9308a019258c31049344f85f89d5229b531c845836f99b08601f113bce036f9", true},
		{"bip340-field-overflow", 2, "fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2f", false},
		{"bip340-no-lift", 2, "0000000000000000000000000000000000000000000000000000000000000000", false},
		{"bip340-short", 2, "01", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := governanceFixture(t)
			raw, err := hex.DecodeString(tc.hex)
			if err != nil {
				t.Fatal(err)
			}
			if tc.name == "p256-in-range-no-lift" && (len(raw) != 33 || raw[0] != 2 || bytes.Compare(raw[1:], elliptic.P256().Params().P.Bytes()) >= 0) {
				t.Fatal("no-lift control must have a compressed tag and in-field x")
			}
			setPublicBytes(governanceKey(c, tc.n), raw)
			_, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow)
			if tc.valid && e != nil {
				t.Fatal(e)
			}
			if !tc.valid && (e == nil || e.Rule != "public_key_point") {
				t.Fatalf("want public_key_point, got %v", e)
			}
		})
	}
	t.Run("ed-mixed-order", func(t *testing.T) {
		c := governanceFixture(t)
		var mixed curve.EdwardsPoint
		mixed.Add(curve.ED25519_BASEPOINT_POINT, curve.EIGHT_TORSION[1])
		var compressed curve.CompressedEdwardsY
		compressed.SetEdwardsPoint(&mixed)
		setPublicBytes(governanceKey(c, 0), compressed[:])
		if _, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow); e == nil || e.Rule != "public_key_point" {
			t.Fatalf("want public_key_point, got %v", e)
		}
	})
}

// A canonical public point can resemble a token in base64url. Typed public
// bytes must remain usable while token-like prose outside that descriptor refuses.
func TestGovernancePublicEncodingPrivacyBoundary(t *testing.T) {
	raw, err := base64.RawURLEncoding.DecodeString("Ask-" + strings.Repeat("A", 40))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for i := 0; i < 256; i++ {
		raw[len(raw)-1] = byte(i)
		x, _ := elliptic.UnmarshalCompressed(elliptic.P256(), raw)
		if x != nil {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no positive public point control")
	}
	c := governanceFixture(t)
	setPublicBytes(governanceKey(c, 1), raw)
	if !secretPattern.MatchString(str(governanceKey(c, 1)["public_bytes"])) {
		t.Fatal("control does not exercise token-like encoding")
	}
	before := encoded(t, c)
	checked, e := ParseGovernanceSnapshot(before, governanceNow)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(before, encoded(t, checked)) || !bytes.Equal(before, encoded(t, c)) {
		t.Fatal("public scan copy changed original declaration")
	}
	for _, tc := range []struct {
		name, rule string
		edit       func(Object)
	}{
		{"description", "secret_value_forbidden", func(c Object) { obj(c["product"])["description"] = str(governanceKey(c, 1)["public_bytes"]) }},
		{"wrong-digest", "public_key_digest", func(c Object) { governanceKey(c, 1)["public_sha256"] = strings.Repeat("0", 64) }},
		{"wrong-id", "public_key_id", func(c Object) { governanceKey(c, 1)["key_id"] = "p256:wrong" }},
		{"padded", "public_key_encoding", func(c Object) { k := governanceKey(c, 1); k["public_bytes"] = str(k["public_bytes"]) + "=" }},
		{"off-curve", "public_key_point", func(c Object) { setPublicBytes(governanceKey(c, 1), []byte{0}) }},
		{"wrong-purpose", "public_key_combination", func(c Object) { governanceKey(c, 2)["purpose"] = "grant-authority" }},
		{"private-field", "unknown_field", func(c Object) { governanceKey(c, 1)["private_key"] = "inert-placeholder" }},
		{"root-credential", "credential_field_forbidden", func(c Object) { c["api_key"] = "inert-placeholder" }},
		{"unselected-invalid-key", "public_key_digest", func(c Object) {
			obj(obj(c["product"])["governance"])["owners"] = []any{"example-participant"}
			governanceKey(c, 1)["public_sha256"] = strings.Repeat("0", 64)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := governanceFixture(t)
			setPublicBytes(governanceKey(card, 1), raw)
			tc.edit(card)
			if _, e := ParseGovernanceSnapshot(encoded(t, card), governanceNow); e == nil || e.Rule != tc.rule {
				t.Fatalf("want %s, got %v", tc.rule, e)
			}
		})
	}
	obj(obj(c["product"])["governance"])["owners"] = []any{"example-participant"}
	if _, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow); e != nil {
		t.Fatal("unselected valid public point", e)
	}
	obj(c["product"])["description"] = str(governanceKey(c, 1)["public_bytes"])
	if _, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow); e == nil || e.Rule != "secret_value_forbidden" {
		t.Fatalf("want prose privacy refusal, got %v", e)
	}
	for _, version := range []int{3, 4} {
		card := fixture(t, "source-v3.json")
		if version == 4 {
			card = fixture(t, "converted-v4.json")
		}
		obj(card["product"])["description"] = str(governanceKey(c, 1)["public_bytes"])
		issue := Validate(encoded(t, card), Options{Version: version})
		want := "R6"
		if version == 4 {
			want = "secret_value_forbidden"
		}
		if issue == nil || issue.Rule != want {
			t.Fatalf("legacy %d want %s, got %v", version, want, issue)
		}
	}
}

// Human managers/reviewers retain graph responsibilities without acquiring
// execution fields; existing service agents retain their exact v4 restrictions.
func TestGovernanceV5MemberBranches(t *testing.T) {
	for _, responsibility := range []string{"keeper", "orchestrator", "reviewer"} {
		t.Run("human-"+responsibility, func(t *testing.T) {
			c := governanceFixture(t)
			n := map[string]int{"keeper": 0, "orchestrator": 1, "reviewer": 4}[responsibility]
			original := obj(list(c["members"])[n])
			human := Object{}
			for _, k := range []string{"id", "role", "responsibility", "project", "parent", "scope", "desired_state"} {
				human[k] = original[k]
			}
			human["kind"] = "human"
			human["principal"] = "example-owner"
			if responsibility == "reviewer" {
				human["responsibility"] = "reviewer"
			}
			list(c["members"])[n] = human
			if _, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow); e != nil {
				t.Fatal(e)
			}
		})
	}
	for _, forbidden := range []string{"", "profile", "runtime", "principal"} {
		t.Run("service-"+forbidden, func(t *testing.T) {
			c := fixture(t, "services-v4.json")
			c["schema"] = "product-card/v5"
			c["principals"] = []any{}
			var service Object
			for _, value := range list(c["members"]) {
				m := obj(value)
				m["kind"] = "agent"
				m["responsibility"] = m["role"]
				if m["role"] == "service" {
					service = m
				}
			}
			if service == nil {
				t.Fatal("missing service control")
			}
			want := ""
			switch forbidden {
			case "profile":
				service["profile"] = "unexpected"
				want = "service_profile_forbidden"
			case "runtime":
				obj(service["runtime"])["model"] = "unexpected"
				want = "service_runtime_forbidden"
			case "principal":
				service["principal"] = "example-owner"
				want = "unknown_field"
			}
			_, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow)
			if want == "" && e != nil {
				t.Fatal(e)
			}
			if want != "" && (e == nil || e.Rule != want) {
				t.Fatalf("want %s, got %v", want, e)
			}
		})
	}
}

// The new reader uses the strict YAML/node boundary, not permissive legacy
// parsing; the explicit clock and collection limits cannot be bypassed.
func TestGovernanceV5Boundaries(t *testing.T) {
	for _, tc := range []struct{ raw, rule string }{
		{"x: null", "null_value"},
		{"x: &a one", "yaml_alias"},
		{"x: !!str one", "explicit_tag"},
		{"x: one\nx: two", "duplicate_key"},
		{"x: one\n---\nx: two", "multiple_documents"},
		{"1: one", "non_string_key"},
	} {
		if _, e := ParseGovernanceSnapshot([]byte(tc.raw), governanceNow); e == nil || e.Rule != tc.rule {
			t.Fatalf("want %s, got %v", tc.rule, e)
		}
	}
	if _, e := ParseGovernanceSnapshot(encoded(t, governanceFixture(t)), time.Time{}); e == nil || e.Rule != "evaluation_time" {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		field, rule string
		limit       int
	}{
		{"principals", "principals_limit", 10000}, {"keys", "keys_limit", 128}, {"contacts", "contacts_limit", 128}, {"owners", "owners_limit", 10000},
	} {
		c := governanceFixture(t)
		switch tc.field {
		case "principals":
			c["principals"] = make([]any, tc.limit+1)
		case "owners":
			obj(obj(c["product"])["governance"])["owners"] = make([]any, tc.limit+1)
		default:
			obj(list(c["principals"])[0])[tc.field] = make([]any, tc.limit+1)
		}
		if e := governanceBounds(c); e == nil || e.Rule != tc.rule {
			t.Fatalf("want %s, got %v", tc.rule, e)
		}
	}
}

// Titles never affect a prospective name. Humans have no derived agent name,
// and the original admission APIs keep rejecting v5.
func TestGovernanceV5NamingAndLegacyBoundary(t *testing.T) {
	c := governanceFixture(t)
	m := obj(list(c["members"])[3])
	before := DeclaredMXID(c, m)
	m["role"] = "keeper"
	checked, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow)
	if e != nil {
		t.Fatal(e)
	}
	if after := DeclaredMXID(checked, obj(list(checked["members"])[3])); before == "" || after != before {
		t.Fatal("title changed prospective name")
	}
	if DeclaredMXID(checked, obj(list(checked["members"])[4])) != "" {
		t.Fatal("human acquired agent name")
	}
	for _, version := range []int{3, 4} {
		if e := Validate(encoded(t, c), Options{Version: version}); e == nil || e.Rule != "not-a-product-card/v"+string(rune('0'+version))+"-document" {
			t.Fatal(e)
		}
	}
}

// Human's closed branch rejects every prohibited execution/admission field
// while the same complete card remains a positive control without the field.
func TestGovernanceHumanForbiddenFields(t *testing.T) {
	for _, field := range []string{"runtime", "model", "profile", "context_mode", "binding", "identity", "lifecycle", "clone", "git_dir", "station", "staffing", "a2a", "hooks", "logical_source_ref", "result_route"} {
		t.Run(field, func(t *testing.T) {
			c := governanceFixture(t)
			if _, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow); e != nil {
				t.Fatal(e)
			}
			obj(list(c["members"])[4])[field] = "unexpected"
			if _, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow); e == nil || e.Rule != "unknown_field" {
				t.Fatalf("want unknown_field, got %v", e)
			}
		})
	}
}

// Portable human ids are legal without a source pin. The human closed branch
// cannot bypass the legacy nonportable-agent source-pin requirement.
func TestGovernanceLogicalIDAdmission(t *testing.T) {
	c := governanceFixture(t)
	if _, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow); e != nil {
		t.Fatal(e)
	}
	obj(list(c["members"])[4])["id"] = "Human.Researcher"
	if _, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow); e == nil || e.Rule != "logical_id_format" {
		t.Fatal(e)
	}
	c = fixture(t, "compat-id-converted-v4.json")
	c["schema"] = "product-card/v5"
	c["principals"] = []any{}
	var legacy Object
	for _, value := range list(c["members"]) {
		m := obj(value)
		m["kind"] = "agent"
		m["responsibility"] = m["role"]
		if !portable.MatchString(str(m["id"])) {
			legacy = m
		}
	}
	if legacy == nil {
		t.Fatal("nonportable legacy positive control missing")
	}
	if _, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow); e != nil {
		t.Fatal(e)
	}
	delete(legacy, "logical_source_ref")
	if _, e := ParseGovernanceSnapshot(encoded(t, c), governanceNow); e == nil || e.Rule != "logical_source_required" {
		t.Fatal(e)
	}
}
