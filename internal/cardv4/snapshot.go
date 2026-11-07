package cardv4

// ParseLocalSnapshot checks the complete v4 declaration and its local graph for
// a single-user unsigned read. It does not attest legacy source bytes, logical
// source bytes or external star reservations. Validate remains the admission
// validator and always requires those independently supplied premises.
// No source reference or declared path is opened by this reader.
func ParseLocalSnapshot(raw []byte) (Object, *Issue) {
	c, e := Parse(raw, 4)
	if e != nil {
		return nil, e
	}
	if c["schema"] != "product-card/v4" {
		return nil, shape("not-a-product-card/v4-document", "$.schema")
	}
	if e := CheckForbiddenFacts(c); e != nil {
		return nil, e
	}
	if e := CheckCollectionBounds(c); e != nil {
		return nil, e
	}
	if e := schema(4)(c, "$"); e != nil {
		return nil, e
	}
	if e := relations(c, Options{}, 4, true); e != nil {
		return nil, e
	}
	return c, nil
}

// DeclaredMXID returns the declared chat name from a checked local snapshot.
// It neither proves ownership nor admits a prospective Matrix identity.
func DeclaredMXID(card, member Object) string {
	identity := obj(member["identity"])
	if identity["scheme"] == "legacy-exact" {
		return str(identity["mxid"])
	}
	binding := obj(obj(card["bindings"])[str(member["binding"])])
	domain := str(binding["identity_domain"])
	return "@" + domain + "_" + str(obj(card["product"])["id"]) + "_" +
		str(member["project"]) + "_" + roleCodes[str(member["role"])] + "_" +
		str(identity["star"]) + ":" + domain
}
