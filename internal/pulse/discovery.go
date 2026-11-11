package pulse

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/relux-works/swarma-pulse/internal/cardv4"
)

type querySource struct {
	selection sourceSelection
	raw       []byte
	card      object
}
type graphMember struct {
	source            *querySource
	member            object
	key, level, agent string
	resolved          any
	parent            any
}

func sourceBudget(additional, totalBytes, members int) error {
	if additional > maxAdditionalSources || totalBytes > maxSourceBytes || members > maxUniverseMembers {
		return refusal("input_limit")
	}
	return nil
}
func loadDiscovery(s selectors, uid int) ([]*querySource, object, error) {
	for _, path := range []string{s.card, s.sources} {
		if err := insideReadRoot(s.root, path); err != nil {
			return nil, nil, err
		}
	}
	manifest, err := readCardBounded(s.sources, uid, "input_limit")
	if err != nil {
		return nil, nil, err
	}
	selections, err := parseSourceManifest(manifest, s.sourcesRevision, s.card, s.root)
	if err != nil {
		return nil, nil, err
	}
	selections = append([]sourceSelection{{path: s.card, revision: s.revision}}, selections...)
	cards := make([]*querySource, len(selections))
	ids := map[string]bool{}
	total, members, companies := 0, 0, 0
	var company *querySource
	load := func(i int) error {
		selection := selections[i]
		raw, err := readCardBounded(selection.path, uid, "input_limit")
		if err != nil {
			return err
		}
		total += len(raw)
		if err := sourceBudget(len(selections)-1, total, members); err != nil {
			return err
		}
		c, err := decodeQueryCard(raw, selection.revision, true)
		if err != nil {
			return err
		}
		id := cardID(c)
		if selection.id != "" && (selection.id != id || selection.schema != c["schema"]) {
			return refusal("source_conflict")
		}
		if ids[id] {
			return refusal("source_conflict")
		}
		ids[id] = true
		if c["schema"] == "company-card/v1" {
			companies++
			if companies > 1 {
				return refusal("source_conflict")
			}
		}
		members += len(arr(c["members"]))
		if err := sourceBudget(len(selections)-1, total, members); err != nil {
			return err
		}
		selection.id = id
		selection.schema = str(c["schema"])
		cards[i] = &querySource{selection: selection, raw: raw, card: c}
		if selection.schema == "company-card/v1" {
			company = cards[i]
		}
		return nil
	}
	// Load an explicitly supplied company before opening its additional product
	// sources. Remote inventory is never resolved as a local input path.
	companyIndex := -1
	for i, selection := range selections {
		if selection.schema == "company-card/v1" {
			if companyIndex != -1 {
				return nil, nil, refusal("source_conflict")
			}
			companyIndex = i
		}
	}
	if companyIndex >= 0 {
		if err := load(companyIndex); err != nil {
			return nil, nil, err
		}
	}
	remote := func(id string) bool {
		if company == nil {
			return false
		}
		for _, v := range arr(company.card["products"]) {
			e := obj(v)
			if e["id"] == id && obj(e["host"])["kind"] == "remote" {
				return true
			}
		}
		return false
	}
	if cards[0] == nil {
		if err := load(0); err != nil {
			return nil, nil, err
		}
	}
	if remote(cards[0].selection.id) {
		return nil, nil, refusal("remote_source_unsupported")
	}
	for i := 1; i < len(selections); i++ {
		if cards[i] != nil {
			continue
		}
		if remote(selections[i].id) {
			return nil, nil, refusal("remote_source_unsupported")
		}
		if err := load(i); err != nil {
			return nil, nil, err
		}
	}

	// Preserve primary-first receipts; only additional sources sort by source id.
	sort.Slice(cards[1:], func(i, j int) bool { return cards[i+1].selection.id < cards[j+1].selection.id })
	return cards, object{"revision": s.sourcesRevision, "sha256": digest(manifest), "root": s.root}, nil
}

func materialize(cards []*querySource) (map[string]*graphMember, error) {
	nodes := map[string]*graphMember{}
	identities := map[string]string{}
	for _, s := range cards {
		h := cardHeader(s.card)
		local := map[string]bool{}
		for _, v := range arr(s.card["members"]) {
			local[str(obj(v)["id"])] = true
		}
		for _, v := range arr(s.card["members"]) {
			m := obj(v)
			key := s.selection.id + "/" + str(m["id"])
			if nodes[key] != nil {
				return nil, refusal("source_conflict")
			}
			level := "member"
			switch m["role"] {
			case "keeper":
				level = "product"
			case "orchestrator":
				level = "project"
			}
			if s.selection.schema == "company-card/v1" {
				level = "company"
			}
			n := &graphMember{source: s, member: m, key: key, level: level}
			if parent, present := m["parent"]; present && parent != nil {
				if p := obj(parent); p != nil {
					n.parent = object{"source": s.selection.id, "external": str(p["product"]) + "/" + str(p["member"])}
				} else {
					p := str(parent)
					if local[p] {
						n.parent = s.selection.id + "/" + p
					} else {
						n.parent = object{"source": s.selection.id, "external": p}
					}
				}
			}
			if obj(m["a2a"]) != nil {
				bk := str(m["binding"])
				if bk == "" {
					bk = str(h["default_binding"])
				}
				binding := obj(obj(s.card["bindings"])[bk])
				connection := obj(obj(s.card["connections"])[str(binding["connection"])])
				n.agent = str(binding["identity_prefix"]) + str(m["id"])
				mx := "@" + n.agent + ":" + str(connection["server_name"])
				if s.selection.schema == "product-card/v4" {
					mx = cardv4.DeclaredMXID(s.card, m)
					n.agent, _, _ = strings.Cut(strings.TrimPrefix(mx, "@"), ":")
				}
				if identities[mx] != "" {
					return nil, refusal("identity_conflict")
				}
				identities[mx] = key
				n.resolved = mx
			}
			nodes[key] = n
		}
	}
	return nodes, nil
}

func composeCompany(cards []*querySource, nodes map[string]*graphMember) ([]any, []any, error) {
	var company *querySource
	byID := map[string]*querySource{}
	for _, s := range cards {
		byID[s.selection.id] = s
		if s.selection.schema == "company-card/v1" {
			company = s
		}
	}
	inventory, boundaries := []any{}, []any{}
	if company == nil {
		return inventory, boundaries, nil
	}
	referenced := map[string]bool{}
	for _, value := range arr(company.card["products"]) {
		e := obj(value)
		id := str(e["id"])
		referenced[id] = true
		source := byID[id]
		host := obj(e["host"])
		card := obj(e["card"])
		resolution := "not_supplied"
		if host["kind"] == "remote" {
			resolution = "remote_unavailable"
			if source != nil {
				return nil, nil, refusal("remote_source_unsupported")
			}
		} else if source != nil {
			expected := filepath.Clean(filepath.Join(filepath.Dir(company.selection.path), str(card["path"])))
			if source.selection.path != expected {
				return nil, nil, refusal("source_reference_mismatch")
			}
			if pin, present := card["revision"]; present && pin != source.selection.revision {
				return nil, nil, refusal("source_digest")
			}
			keeper := nodes[str(e["keeper"])]
			if source.selection.schema != "product-card/v3" || source.selection.id+"/"+str(cardHeader(source.card)["keeper"]) != e["keeper"] || keeper == nil || keeper.member["parent"] != e["parent_alias"] || obj(keeper.parent)["external"] != e["parent_alias"] || nodes[str(e["superior"])] == nil {
				return nil, nil, refusal("relation_conflict")
			}
			keeper.parent = e["superior"]
			resolution = "resolved"
		}
		hostLocator := object{"source": company.selection.id, "base": "card"}
		for k, v := range host {
			hostLocator[k] = v
		}
		cardLocator := object{"source": company.selection.id, "base": "card", "path": card["path"]}
		if pin, present := card["revision"]; present {
			cardLocator["revision"] = pin
		}
		inventory = append(inventory, object{"id": id, "description": e["description"], "host": hostLocator, "card": cardLocator, "keeper": e["keeper"], "superior": e["superior"], "resolution": resolution})
		if resolution != "resolved" {
			boundaries = append(boundaries, object{"from": str(e["superior"]), "relation": "product", "target": id, "reason": resolution})
		}
	}
	for _, s := range cards {
		if s != company && !referenced[s.selection.id] {
			return nil, nil, refusal("source_unreferenced")
		}
	}
	sort.Slice(inventory, func(i, j int) bool { return str(obj(inventory[i])["id"]) < str(obj(inventory[j])["id"]) })
	return inventory, boundaries, nil
}

func checkReportingGraph(nodes map[string]*graphMember) error {
	limited := false
	for _, start := range nodes {
		seen := map[string]bool{}
		n := start
		hops := 0
		for n != nil {
			if seen[n.key] {
				return refusal("relation_conflict")
			}
			seen[n.key] = true
			if n.parent == nil {
				break
			}
			hops++
			if hops > maxReportingHops {
				// Keep each walk bounded, but let a relation conflict found by
				// another bounded walk take precedence, regardless of map order.
				limited = true
				break
			}
			parent, local := n.parent.(string)
			if !local {
				break
			}
			n = nodes[parent]
			if n == nil {
				return refusal("relation_conflict")
			}
		}
	}
	if limited {
		return refusal("input_limit")
	}
	return nil
}
func memberProject(n *graphMember) any {
	if p := str(n.member["project"]); p != "" {
		return n.source.selection.id + "/" + p
	}
	return nil
}
func memberSummary(n *graphMember) object {
	return object{"key": n.key, "role": n.member["role"], "level": n.level, "desired_state": n.member["desired_state"], "resolved_name": n.resolved, "project": memberProject(n), "parent": n.parent}
}

func projectDiscovery(cards []*querySource, manifest object, agent string, now time.Time) ([]byte, error) {
	return projectGraph(cards, manifest, agent, now, false)
}

// projectGraph shares traversal and ordering across explicitly supplied discovery
// sources and a single hierarchy card. The existing discovery schema stays intact.
func projectGraph(cards []*querySource, manifest object, agent string, now time.Time, hierarchy bool) ([]byte, error) {
	if agent == "" || len(agent) > 512 || !utf8.ValidString(agent) || strings.ContainsFunc(agent, unicode.IsControl) {
		return nil, refusal("caller_context")
	}
	nodes, err := materialize(cards)
	if err != nil {
		return nil, err
	}
	inventory, boundaries, err := composeCompany(cards, nodes)
	if err != nil {
		return nil, err
	}
	if err := checkReportingGraph(nodes); err != nil {
		return nil, err
	}
	var self *graphMember
	matches := 0
	keys := []string{}
	for k, n := range nodes {
		keys = append(keys, k)
		if n.resolved != nil && n.agent == agent {
			self = n
			matches++
		}
	}
	if matches == 0 {
		return nil, refusal("caller_unknown")
	}
	if matches != 1 {
		return nil, refusal("caller_ambiguous")
	}
	sort.Strings(keys)
	baseView := "environment"
	if hierarchy {
		baseView = ""
	}
	base, err := projectView(self.source.raw, self.source.selection.revision, filepath.Dir(self.source.selection.path), agent, now, baseView)
	if err != nil {
		return nil, err
	}
	var result object
	if err := json.Unmarshal(base, &result); err != nil {
		return nil, refusal("output_encoding")
	}
	if hierarchy {
		result["schema"] = "pulse-view/v3"
		obj(result["self"])["level"] = self.level
		delete(result, "source")
		delete(result, "product")
		delete(result, "member")
	}
	peers, reports, descendants, superiors := []any{}, []any{}, []any{}, []any{}
	returned := map[string]bool{self.key: true}
	for _, key := range keys {
		n := nodes[key]
		if n != self && n.level == self.level {
			peers = append(peers, memberSummary(n))
			returned[key] = true
		}
		if n.parent == self.key {
			reports = append(reports, memberSummary(n))
			returned[key] = true
		}
		if n != self {
			for p := n.parent; p != nil; {
				parent, ok := p.(string)
				if !ok {
					break
				}
				if parent == self.key {
					descendants = append(descendants, memberSummary(n))
					returned[key] = true
					break
				}
				p = nodes[parent].parent
			}
		}
		if p := obj(n.parent); p != nil {
			boundaries = append(boundaries, object{"from": n.key, "relation": "parent", "target": p, "reason": "external_unresolved"})
		}
	}
	for parent := self.parent; parent != nil; {
		key, local := parent.(string)
		if !local {
			break
		}
		n := nodes[key]
		superiors = append(superiors, memberSummary(n))
		returned[key] = true
		parent = n.parent
	}
	projects := []any{}
	productIDs, projectIDs := map[string]bool{}, map[string]bool{}
	for key := range returned {
		n := nodes[key]
		productIDs[n.source.selection.id] = true
		if project, ok := memberProject(n).(string); ok {
			projectIDs[project] = true
		}
	}
	for _, source := range cards {
		for _, value := range arr(source.card["projects"]) {
			p := obj(value)
			key := source.selection.id + "/" + str(p["id"])
			if projectIDs[key] {
				location := object{"source": source.selection.id, "base": "card", "kind": "local", "path": p["path"]}
				if source.selection.schema == "product-card/v4" {
					location = object{"source": source.selection.id, "base": "card"}
					for _, field := range []string{"kind", "path", "operator", "execution_host", "project_locator", "checkout_path"} {
						if value, present := obj(p["location"])[field]; present {
							location[field] = value
						}
					}
				}
				projects = append(projects, object{"key": key, "repo": p["repo"], "desired_state": p["desired_state"], "location": location})
			}
		}
	}
	products := []any{}
	for _, value := range inventory {
		if self.level == "company" || productIDs[str(obj(value)["id"])] {
			products = append(products, value)
		}
	}
	sort.Slice(projects, func(i, j int) bool { return str(obj(projects[i])["key"]) < str(obj(projects[j])["key"]) })
	sort.Slice(boundaries, func(i, j int) bool {
		a, b := obj(boundaries[i]), obj(boundaries[j])
		for _, field := range []string{"from", "relation", "target", "reason"} {
			if field == "target" && obj(a[field]) != nil {
				// Parent targets are closed source/external objects; product
				// targets are strings. Relation sorts before this field.
				for _, key := range []string{"source", "external"} {
					x, y := str(obj(a[field])[key]), str(obj(b[field])[key])
					if x != y {
						return x < y
					}
				}
			} else if x, y := str(a[field]), str(b[field]); x != y {
				return x < y
			}
		}
		return false
	})
	receipts := []any{}
	for _, s := range cards {
		receipt := object{"id": s.selection.id, "schema": s.selection.schema, "revision": s.selection.revision, "sha256": digest(s.raw), "card_root": filepath.Dir(s.selection.path), "freshness": "unknown", "observed_at": nil}
		if s.selection.schema == "product-card/v4" {
			receipt["validation"] = "local declarations only; source attestations and star reservations unverified"
		}
		receipts = append(receipts, receipt)
	}
	result["query"] = object{"view": "discovery", "sources": manifest}
	if hierarchy {
		result["query"] = object{"view": "hierarchy"}
	}
	result["sources"] = receipts
	result["peers"] = peers
	result["direct_reports"] = reports
	result["descendants"] = descendants
	result["superiors"] = superiors
	result["projects"] = projects
	result["products"] = products
	result["boundaries"] = boundaries
	result["hierarchy_status"] = "complete"
	if len(boundaries) > 0 {
		result["hierarchy_status"] = "partial"
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, refusal("output_encoding")
	}
	return append(encoded, '\n'), nil
}
