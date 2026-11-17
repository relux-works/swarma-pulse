package pulse

import (
	"sort"
	"time"

	"github.com/relux-works/swarma-pulse/internal/cardv4"
)

// The explicit selector alone admits v5. Reparse original bytes with the strict
// reader; a permissive legacy parse never supplies normalized v5 input.
func decodeGovernanceCard(raw []byte, revision string, now time.Time) (object, error) {
	if revision != "sha256:"+digest(raw) {
		return nil, refusal("source_digest")
	}
	c, issue := cardv4.Parse(raw, 3)
	if issue != nil {
		return nil, refusal("card_invalid")
	}
	if c["schema"] != "product-card/v5" {
		return decodeQueryCard(raw, revision, false)
	}
	c, issue = cardv4.ParseGovernanceSnapshot(raw, now)
	if issue != nil {
		return nil, refusal("card_invalid")
	}
	return c, nil
}

// Governance is declared metadata, never an issuer selection or verified owner
// identity. Whole-document validation precedes this projection, including every
// unselected principal/key. Sorting fresh slices leaves the input unchanged.
func declaredGovernance(card object) object {
	result := object{"state": "undeclared", "owners": []any{}, "meta_keeper": nil}
	g := obj(cardHeader(card)["governance"])
	if card["schema"] != "product-card/v5" || g == nil {
		return result
	}
	principals := map[string]object{}
	for _, value := range arr(card["principals"]) {
		p := obj(value)
		principals[str(p["id"])] = p
	}
	ids := make([]string, 0, len(arr(g["owners"])))
	for _, id := range arr(g["owners"]) {
		ids = append(ids, str(id))
	}
	sort.Strings(ids)
	owners := make([]any, 0, len(ids))
	for _, id := range ids {
		p := principals[id]
		keys := append([]any{}, arr(p["keys"])...)
		sort.Slice(keys, func(i, j int) bool {
			a, b := obj(keys[i]), obj(keys[j])
			if a["purpose"] != b["purpose"] {
				return str(a["purpose"]) < str(b["purpose"])
			}
			return str(a["key_id"]) < str(b["key_id"])
		})
		owners = append(owners, object{"id": p["id"], "kind": p["kind"], "name": p["name"], "contacts": p["contacts"], "keys": keys})
	}
	result["state"], result["owners"] = "declared", owners
	meta := object{"resolution": "external_unresolved", "member": nil}
	if id, local := g["meta_keeper"].(string); local {
		meta["reference"] = object{"product": cardID(card), "member": id}
		meta["resolution"] = "resolved"
		for _, value := range arr(card["members"]) {
			m := obj(value)
			if m["id"] != id {
				continue
			}
			var principal any
			if m["kind"] == "human" {
				principal = m["principal"]
			}
			meta["member"] = object{
				"key": cardID(card) + "/" + id, "kind": m["kind"], "role": m["role"],
				"responsibility": m["responsibility"], "scope": m["scope"], "desired_state": m["desired_state"],
				"project": cardID(card) + "/" + str(m["project"]), "principal": principal,
			}
			break
		}
	} else {
		ref := obj(g["meta_keeper"])
		meta["reference"] = object{"product": ref["product"], "member": ref["member"]}
	}
	result["meta_keeper"] = meta
	return result
}
