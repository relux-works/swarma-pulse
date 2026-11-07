package pulse

import (
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/relux-works/swarma-pulse/internal/cardv4"
)

func cardHeader(c object) object {
	if c["schema"] == "company-card/v1" {
		return obj(c["company"])
	}
	return obj(c["product"])
}
func cardID(c object) string { return str(cardHeader(c)["id"]) }

func validateCompany(c object) error {
	if issue := cardv4.ValidateCompanyCore(c); issue != nil {
		return refusal("card_invalid")
	}
	ids := map[string]bool{cardID(c): true}
	members := map[string]bool{}
	for _, v := range arr(c["members"]) {
		members[cardID(c)+"/"+str(obj(v)["id"])] = true
	}
	for _, v := range arr(c["products"]) {
		e := obj(v)
		h := obj(e["host"])
		r := obj(e["card"])
		id := str(e["id"])
		if len(e) != 7 || !onlyKeys(e, "id", "description", "host", "card", "keeper", "superior", "parent_alias") || !sourceID(e["id"]) || !sourceID(e["description"]) || !sourceID(e["parent_alias"]) || !strings.HasPrefix(str(e["keeper"]), id+"/") || len(str(e["keeper"])) <= len(id)+1 || !members[str(e["superior"])] || ids[id] {
			return refusal("card_invalid")
		}
		ids[id] = true
		if !onlyKeys(r, "path", "revision") || len(r) < 1 {
			return refusal("card_invalid")
		}
		if !declaredRelative(r["path"]) {
			return refusal("input_path")
		}
		if pin, present := r["revision"]; present && !validRevision(str(pin)) {
			return refusal("card_invalid")
		}
		switch h["kind"] {
		case "local":
			if !containedInventoryCard(h["path"], r["path"]) {
				return refusal("input_path")
			}
			if len(h) != 2 || !onlyKeys(h, "kind", "path") {
				return refusal("card_invalid")
			}
		case "remote":
			locator := str(h["locator"])
			if len(h) != 4 || !onlyKeys(h, "kind", "operator", "execution_host", "locator") || !sourceID(h["operator"]) || !sourceID(h["execution_host"]) || locator == "" || len(locator) > 512 || !utf8.ValidString(locator) || strings.ContainsFunc(locator, unicode.IsControl) {
				return refusal("card_invalid")
			}
		default:
			return refusal("card_invalid")
		}
	}
	return nil
}

// All cards keep their actual format and source bytes. Company core validation
// shares v3 predicates but never enters the old product query/validator route.
func decodeQueryCard(raw []byte, revision string, allowCompany bool) (object, error) {
	if revision != "sha256:"+digest(raw) {
		return nil, refusal("source_digest")
	}
	c, issue := cardv4.Parse(raw, 3)
	if issue != nil {
		return nil, refusal("card_invalid")
	}
	switch c["schema"] {
	case "product-card/v3":
		if issue := cardv4.Validate(raw, cardv4.Options{Version: 3}); issue != nil {
			return nil, refusal("card_invalid")
		}
	case "product-card/v4":
		if allowCompany {
			return nil, refusal("unsupported_schema")
		}
		// Reparse the original bytes with strict v4 YAML rules. A legacy parse
		// must never normalize away aliases, merges or other v4 refusals.
		c, issue = cardv4.ParseLocalSnapshot(raw)
		if issue != nil {
			return nil, refusal("card_invalid")
		}
	case "company-card/v1":
		if !allowCompany {
			return nil, refusal("unsupported_schema")
		}
		if err := validateCompany(c); err != nil {
			return nil, err
		}
	default:
		return nil, refusal("unsupported_schema")
	}
	if allowCompany {
		for _, v := range arr(c["members"]) {
			if !validEnvironmentExtension(obj(v)) {
				return nil, refusal("environment_invalid")
			}
		}
	}
	return c, nil
}

// Coordinate-only containment: inspect path components after normalization,
// never prefix strings or an unaccessed host's filesystem.
func containedInventoryCard(host, card any) bool {
	if !declaredRelative(host) || !declaredRelative(card) || filepath.Base(filepath.Clean(str(card))) != "product-card.yaml" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(str(host)), filepath.Clean(str(card)))
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
