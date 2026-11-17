package cardv4

import (
	"bytes"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/oasisprotocol/curve25519-voi/curve"
)

func cardVersion(c Object) int {
	if c["schema"] == "product-card/v5" {
		return 5
	}
	if c["schema"] == "product-card/v4" {
		return 4
	}
	return 3
}
func mechanicalRole(m Object, version int) string {
	if version == 5 {
		return str(m["responsibility"])
	}
	return str(m["role"])
}
func title(v any) bool { return text(v) && len(str(v)) <= 255 }
func humanMemberSchema(parent, state, role validator) validator {
	return object(map[string]validator{
		"id": scalar(func(v any) bool { return portable.MatchString(str(v)) }, "logical_id_format"), "kind": enumeration("human"),
		"role":           scalar(title, "title_format"),
		"responsibility": enumeration("keeper orchestrator developer tester architect reviewer researcher"),
		"project":        scalar(func(v any) bool { return slug.MatchString(str(v)) && len(str(v)) <= 40 }, "slug_format"),
		"parent":         parent, "scope": enumeration("product project member"), "desired_state": state,
		"principal": scalar(func(v any) bool { return portable.MatchString(str(v)) }, "principal_id_format"),
	}, map[string]validator{
		"desired_state_ref": scalar(text, "text_format"), "capabilities": array(role, false),
	}, true)
}
func utcSecond(v any) bool {
	s := str(v)
	t, err := time.Parse("2006-01-02T15:04:05Z", s)
	return err == nil && len(s) == 20 && t.Format("2006-01-02T15:04:05Z") == s
}
func principalSchema() validator {
	txt := scalar(text, "text_format")
	key := object(map[string]validator{
		"purpose":      enumeration("identity grant-authority"),
		"algorithm":    enumeration("Ed25519 P-256 BIP-340"),
		"encoding":     enumeration("ed25519-raw sec1-compressed secp256k1-x-only"),
		"public_bytes": txt, "key_id": txt,
		"public_sha256":     scalar(func(v any) bool { return digest.MatchString(str(v)) }, "digest_format"),
		"declared_state":    enumeration("active retired revoked"),
		"declared_state_at": scalar(utcSecond, "utc_second"),
	}, map[string]validator{
		"not_before": scalar(utcSecond, "utc_second"), "not_after": scalar(utcSecond, "utc_second"),
	}, true)
	return object(map[string]validator{
		"id":   scalar(func(v any) bool { return portable.MatchString(str(v)) }, "principal_id_format"),
		"kind": enumeration("human"), "name": scalar(title, "title_format"),
		"contacts": array(object(map[string]validator{"kind": enumeration("matrix"), "mxid": scalar(mxid, "mxid_format")}, nil, true), false),
		"keys":     array(key, false),
	}, nil, true)
}

// ParseGovernanceSnapshot validates a whole v5 declaration at an explicit
// observation time. It performs no source reads, signature verification, key
// custody, registration or authority selection.
func ParseGovernanceSnapshot(raw []byte, now time.Time) (Object, *Issue) {
	if now.IsZero() {
		return nil, fail("evaluation_time", "$")
	}
	c, e := Parse(raw, 5)
	if e != nil {
		return nil, e
	}
	if c["schema"] != "product-card/v5" {
		return nil, shape("not-a-product-card/v5-document", "$.schema")
	}
	privacyIssue := CheckForbiddenFacts(c)
	if privacyIssue != nil && privacyIssue.Rule != "secret_value_forbidden" {
		return nil, privacyIssue
	}
	if e = CheckCollectionBounds(c); e != nil {
		return nil, e
	}
	if e = governanceBounds(c); e != nil {
		return nil, e
	}
	if e = schema(5)(c, "$"); e != nil {
		return nil, e
	}
	if e = relations(c, Options{}, 5, true); e != nil {
		return nil, e
	}
	if e = governanceRelations(c, now); e != nil {
		return nil, e
	}
	if privacyIssue != nil {
		// Only two values in already fully validated PublicKey records are
		// typed public data rather than prose. Keep every other field/value
		// subject to the complete scanner, and return the original card.
		if e = CheckForbiddenFacts(publicValueScanCopy(c)); e != nil {
			return nil, e
		}
	}
	return c, nil
}

// This private copy is used only after whole-card and descriptor validation.
// It neither mutates the source objects nor supplies the projected result.
func publicValueScanCopy(c Object) Object {
	scan := copyObject(c)
	principals := make([]any, 0, len(list(c["principals"])))
	for _, value := range list(c["principals"]) {
		p := copyObject(obj(value))
		keys := make([]any, 0, len(list(p["keys"])))
		for _, value := range list(p["keys"]) {
			k := copyObject(obj(value))
			k["public_bytes"], k["key_id"] = "validated-public-point", "validated-public-point"
			keys = append(keys, k)
		}
		p["keys"] = keys
		principals = append(principals, p)
	}
	scan["principals"] = principals
	return scan
}
func governanceBounds(c Object) *Issue {
	p, ok := c["principals"].([]any)
	if !ok {
		return nil
	} // The closed schema supplies the missing/type refusal.
	if len(p) > 10000 {
		return fail("principals_limit", "$.principals")
	}
	for i, value := range p {
		for _, k := range []string{"contacts", "keys"} {
			if len(list(obj(value)[k])) > 128 {
				return fail(k+"_limit", field(index("$.principals", i), k))
			}
		}
	}
	if len(list(obj(obj(c["product"])["governance"])["owners"])) > 10000 {
		return fail("owners_limit", "$.product.governance.owners")
	}
	return nil
}
func governanceRelations(c Object, now time.Time) *Issue {
	principals := map[string]bool{}
	for i, value := range list(c["principals"]) {
		p := obj(value)
		at := index("$.principals", i)
		id := str(p["id"])
		if principals[id] {
			return fail("duplicate_principal", at+".id")
		}
		principals[id] = true
		contacts := map[string]bool{}
		for j, value := range list(p["contacts"]) {
			mx := str(obj(value)["mxid"])
			if contacts[mx] {
				return fail("duplicate_contact", index(at+".contacts", j))
			}
			contacts[mx] = true
		}
		keys := map[string]bool{}
		for j, value := range list(p["keys"]) {
			k := obj(value)
			loc := index(at+".keys", j)
			if e := checkPublicKey(k, loc, now); e != nil {
				return e
			}
			identity := str(k["purpose"]) + "/" + str(k["key_id"])
			if keys[identity] {
				return fail("duplicate_public_key", loc)
			}
			keys[identity] = true
		}
	}
	members := map[string]bool{}
	for i, value := range list(c["members"]) {
		m := obj(value)
		members[str(m["id"])] = true
		if m["kind"] == "human" && !principals[str(m["principal"])] {
			return fail("principal_reference", index("$.members", i)+".principal")
		}
	}
	g := obj(obj(c["product"])["governance"])
	if g == nil {
		return nil
	}
	seen := map[string]bool{}
	for i, value := range list(g["owners"]) {
		id := str(value)
		at := index("$.product.governance.owners", i)
		if !principals[id] {
			return fail("principal_reference", at)
		}
		if seen[id] {
			return fail("duplicate_owner", at)
		}
		seen[id] = true
	}
	if id, ok := g["meta_keeper"].(string); ok {
		if !members[id] {
			return fail("meta_keeper_reference", "$.product.governance.meta_keeper")
		}
	} else {
		ref := obj(g["meta_keeper"])
		found := false
		for _, value := range list(c["external_members"]) {
			m := obj(value)
			found = found || m["product"] == ref["product"] && m["member"] == ref["member"]
		}
		if !found {
			return fail("meta_keeper_reference", "$.product.governance.meta_keeper")
		}
	}
	return nil
}
func checkPublicKey(k Object, at string, now time.Time) *Issue {
	type codec struct{ encoding, prefix string }
	codecs := map[string]codec{
		"Ed25519": {"ed25519-raw", "ed25519:"},
		"P-256":   {"sec1-compressed", "p256:"},
		"BIP-340": {"secp256k1-x-only", "bip340:"},
	}
	algorithm := str(k["algorithm"])
	descriptor, ok := codecs[algorithm]
	if !ok || k["encoding"] != descriptor.encoding || algorithm == "BIP-340" && k["purpose"] != "identity" {
		return fail("public_key_combination", at)
	}
	value := str(k["public_bytes"])
	raw, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil || base64.RawURLEncoding.EncodeToString(raw) != value {
		return fail("public_key_encoding", at+".public_bytes")
	}
	if k["key_id"] != descriptor.prefix+value {
		return fail("public_key_id", at+".key_id")
	}
	sum := sha256.Sum256(raw)
	if k["public_sha256"] != hex.EncodeToString(sum[:]) {
		return fail("public_key_digest", at+".public_sha256")
	}
	valid := false
	switch algorithm {
	case "Ed25519":
		var compressed curve.CompressedEdwardsY
		if _, err := compressed.SetBytes(raw); err == nil && compressed.IsCanonicalVartime() {
			var point curve.EdwardsPoint
			if _, err = point.SetCompressedY(&compressed); err == nil {
				valid = !point.IsIdentity() && point.IsTorsionFree()
			}
		}
	case "P-256":
		if len(raw) == 33 && (raw[0] == 2 || raw[0] == 3) {
			x, y := elliptic.UnmarshalCompressed(elliptic.P256(), raw)
			valid = x != nil && y != nil && bytes.Equal(elliptic.MarshalCompressed(elliptic.P256(), x, y), raw)
		}
	case "BIP-340":
		if len(raw) == 32 {
			compressed := append([]byte{2}, raw...)
			point, err := secp256k1.ParsePubKey(compressed)
			valid = err == nil && bytes.Equal(point.SerializeCompressed(), compressed)
		}
	}
	if !valid {
		return fail("public_key_point", at+".public_bytes")
	}
	stateAt, _ := time.Parse("2006-01-02T15:04:05Z", str(k["declared_state_at"]))
	if stateAt.After(now) {
		return fail("declared_state_future", at+".declared_state_at")
	}
	before, b := k["not_before"]
	after, a := k["not_after"]
	if b && a {
		start, _ := time.Parse("2006-01-02T15:04:05Z", str(before))
		end, _ := time.Parse("2006-01-02T15:04:05Z", str(after))
		if !start.Before(end) {
			return fail("public_key_validity", at)
		}
	}
	return nil
}
