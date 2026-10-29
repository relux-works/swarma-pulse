package cardv4

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
)

// Source is immutable input supplied by an admitted caller, never a fetch instruction.
type Source struct {
	Kind     string
	Revision string
	SHA256   string
	Bytes    []byte
}
type Reservation struct {
	Holder      string `json:"holder"`
	Domain      string `json:"domain"`
	Product     string `json:"product"`
	PrimaryRole string `json:"primary_role"`
	Star        string `json:"star"`
}
type Options struct {
	Version      int
	Sources      []Source
	Reservations []Reservation
}

// Validate returns only a safe refusal or nil. It cannot register, launch, write or fetch.
func Validate(raw []byte, options Options) *Issue {
	version := options.Version
	if version == 0 {
		version = 4
	}
	if version != 3 && version != 4 {
		return shape("usage", "$")
	}
	card, e := Parse(raw, version)
	if e != nil {
		return e
	}
	if card["schema"] != "product-card/v"+string(rune('0'+version)) {
		if version == 3 {
			return shape("not-a-product-card/v3-document", "$.schema")
		}
		return shape("not-a-product-card/v4-document", "$.schema")
	}
	e = validateDecoded(card, options, version)
	if version == 3 && e != nil {
		e = legacyIssue(e)
	}
	return e
}
func validateDecoded(card Object, o Options, version int) *Issue {
	if e := CheckForbiddenFacts(card); e != nil {
		return e
	}
	if e := CheckCollectionBounds(card); e != nil {
		return e
	}
	if e := schema(version)(card, "$"); e != nil {
		return e
	}
	return relations(card, o, version)
}
func legacyIssue(e *Issue) *Issue {
	if strings.HasPrefix(e.Location, "$.product.topology") {
		return &Issue{"R15", e.Location, e.Exit}
	}
	if e.Rule == "retired_workspace" {
		return &Issue{"R13", e.Location, e.Exit}
	}
	if e.Rule == "layout_unreadable" {
		return &Issue{"R7", e.Location, e.Exit}
	}
	if strings.Contains(e.Location, ".a2a") && e.Rule != "binding_connection_mismatch" && e.Rule != "route_contract" {
		return &Issue{"R10", e.Location, e.Exit}
	}
	rules := map[string]string{"relative_path": "R1", "profile_reference": "R2", "duplicate_member": "R3", "reviewer_independence": "R4", "reporting_line": "R5", "reporting_cycle": "R5", "credential_field_forbidden": "R6", "secret_value_forbidden": "R6", "layout_missing": "R7", "live_state_forbidden": "R8", "host_prefixed_id": "R9", "a2a_contract": "R10", "enum_value": "R11", "binding_reference": "R12", "connection_reference": "R12", "binding_connection_mismatch": "R12", "route_contract": "R14", "topology_contract": "R15", "profile_version_incompatible": "R16"}
	r, ok := rules[e.Rule]
	if e.Rule == "parent_native_a2a_forbidden" {
		r, ok = "R14", true
	}
	if !ok {
		r = "R13"
	}
	return &Issue{r, e.Location, e.Exit}
}
func relations(c Object, o Options, version int) *Issue {
	v4 := version == 4
	product := obj(c["product"])
	members := list(c["members"])
	projects := list(c["projects"])
	profiles := obj(c["profiles"])
	bindings := obj(c["bindings"])
	connections := obj(c["connections"])
	mi := map[string]Object{}
	pi := map[string]Object{}
	for i, v := range members {
		m := obj(v)
		id := str(m["id"])
		if mi[id] != nil {
			return fail("duplicate_member", index("$.members", i)+".id")
		}
		mi[id] = m
	}
	for i, v := range projects {
		p := obj(v)
		id := str(p["id"])
		if pi[id] != nil {
			return fail("duplicate_project", index("$.projects", i)+".id")
		}
		pi[id] = p
	}
	if bindings[str(product["default_binding"])] == nil {
		return fail("binding_reference", "$.product.default_binding")
	}
	for i, k := range sorted(bindings) {
		b := obj(bindings[k])
		conn := obj(connections[str(b["connection"])])
		if conn == nil {
			return fail("connection_reference", index("$.bindings", i)+".connection")
		}
		if v4 && b["identity_domain"] != conn["server_name"] {
			return fail("binding_domain_mismatch", index("$.bindings", i)+".identity_domain")
		}
	}
	top := obj(product["topology"])
	if connections[str(top["connection"])] == nil {
		return fail("topology_contract", "$.product.topology.connection")
	}
	keeper := mi[str(product["keeper"])]
	if keeper == nil || keeper["role"] != "keeper" {
		return fail("reporting_line", "$.product.keeper")
	}
	if v4 {
		host := pi[str(product["host_project"])]
		if host == nil || obj(host["location"])["kind"] != "local" || obj(host["location"])["path"] != "." {
			return fail("host_project_anchor", "$.product.host_project")
		}
	}
	externals := map[string]bool{}
	for i, v := range list(c["external_members"]) {
		key := str(v)
		if v4 {
			m := obj(v)
			if m["product"] == product["id"] {
				return fail("external_self_product", index("$.external_members", i))
			}
			key = str(m["product"]) + "/" + str(m["member"])
		}
		if v4 && externals[key] {
			return fail("duplicate_external", index("$.external_members", i))
		}
		externals[key] = true
	}
	keeperCount := 0
	resolved := map[string]bool{}
	edges := map[string]string{}
	for i, v := range members {
		m := obj(v)
		at := index("$.members", i)
		id := str(m["id"])
		role := str(m["role"])
		manager := role == "keeper" || role == "orchestrator"
		service := role == "service"
		bkey := str(m["binding"])
		if bkey == "" {
			bkey = str(product["default_binding"])
		}
		binding := obj(bindings[bkey])
		if binding == nil {
			return fail("binding_reference", at+".binding")
		}
		if !v4 {
			for _, b := range bindings {
				prefix := str(obj(b)["identity_prefix"])
				if prefix != "" && strings.HasPrefix(id, prefix) {
					return fail("host_prefixed_id", at+".id")
				}
			}
		}
		if !service && profiles[str(m["profile"])] == nil {
			return fail("profile_reference", at+".profile")
		}
		p := pi[str(m["project"])]
		if (v4 || role != "keeper") && p == nil {
			return fail("project_reference", at+".project")
		}
		if v4 {
			scope := "member"
			if role == "keeper" {
				scope = "product"
			} else if role == "orchestrator" {
				scope = "project"
			}
			if m["scope"] != scope {
				return fail("scope_role", at+".scope")
			}
			if scope == "product" && m["project"] != product["host_project"] {
				return fail("host_project_anchor", at+".project")
			}
			pin := obj(m["logical_source_ref"])
			if !portable.MatchString(id) && pin == nil {
				return fail("logical_source_required", at+".logical_source_ref")
			}
			if pin != nil {
				if _, _, e := resolveSource(pin, product, id, o, false); e != nil {
					return relocated(e, at+".logical_source_ref")
				}
			}
		}
		if role == "keeper" {
			keeperCount++
			parent := str(m["parent"])
			if v4 {
				q := obj(m["parent"])
				parent = str(q["product"]) + "/" + str(q["member"])
			}
			if m != nil && id != product["keeper"] || !externals[parent] {
				return fail("reporting_line", at+".parent")
			}
		} else {
			parent := str(m["parent"])
			want := str(product["keeper"])
			if role != "orchestrator" {
				want = str(p["orchestrator"])
			}
			if mi[parent] == nil || parent != want {
				return fail("reporting_line", at+".parent")
			}
			edges[id] = parent
		}
		if role == "reviewer" && (m["context_mode"] != "fresh" || m["independent_of_producer"] != true) {
			return fail("reviewer_independence", at)
		}
		if _, ok := m["staffing"]; ok && !manager {
			return fail("staffing_manager_only", at+".staffing")
		}
		for j, v := range list(m["staffing"]) {
			s := obj(v)
			loc := index(at+".staffing", j)
			if profiles[str(s["profile"])] == nil {
				return fail("profile_reference", loc+".profile")
			}
			if s["role"] == "reviewer" && (s["context_mode"] != "fresh" || s["independent_of_producer"] != true) {
				return fail("reviewer_independence", loc)
			}
		}
		a := obj(m["a2a"])
		if !v4 && a != nil {
			if _, present := a["workspace"]; present {
				return fail("retired_workspace", at+".a2a.workspace")
			}
		}
		route := str(m["result_route"])
		if manager {
			if a == nil {
				return fail("route_contract", at+".a2a")
			}
		} else {
			if route != "parent-native" && route != "a2a" {
				return fail("route_contract", at+".result_route")
			}
			if route == "parent-native" && a != nil {
				return fail("parent_native_a2a_forbidden", at+".a2a")
			}
			if route == "a2a" && (a == nil || !service && !text(m["chat_admission"])) {
				return fail("route_contract", at+".result_route")
			}
		}
		if service {
			if route == "parent-native" {
				if !text(m["native_result_ref"]) {
					return fail("native_result_required", at+".native_result_ref")
				}
				if _, ok := m["selector_policy_ref"]; ok {
					return fail("parent_native_selector_policy_forbidden", at+".selector_policy_ref")
				}
			} else {
				if _, ok := m["native_result_ref"]; ok {
					return fail("service_native_result_forbidden", at+".native_result_ref")
				}
				subs := obj(a["subscriptions"])
				if len(list(subs["required"]))+len(list(subs["self"])) > 0 && !text(m["selector_policy_ref"]) {
					return fail("service_selector_policy_required", at+".a2a.subscriptions")
				}
			}
		}
		if a != nil {
			if a["connection"] != binding["connection"] {
				return fail("binding_connection_mismatch", at+".a2a.connection")
			}
			if e := checkA2a(a, top, at+".a2a", !manager && !service, m); e != nil {
				return e
			}
		}
		if v4 {
			identity := obj(m["identity"])
			scheme := str(identity["scheme"])
			if a == nil && scheme == "legacy-exact" {
				return fail("nonchat_legacy_forbidden", at+".identity")
			}
			var name string
			if scheme == "legacy-exact" {
				ref := obj(identity["source_ref"])
				if pin := obj(m["logical_source_ref"]); pin != nil && !reflect.DeepEqual(pin, ref) {
					return fail("legacy_source_key_mismatch", at+".identity.source_ref")
				}
				mx, conn, e := resolveSource(ref, product, id, o, true)
				if e != nil {
					return relocated(e, at+".identity.source_ref")
				}
				if mx != identity["mxid"] {
					return fail("legacy_source_mismatch", at+".identity.mxid")
				}
				if !reflect.DeepEqual(conn, connections[str(binding["connection"])]) {
					return fail("legacy_source_connection_mismatch", at+".identity.source_ref")
				}
				_, server, _ := strings.Cut(mx, ":")
				if server != str(conn["server_name"]) {
					return fail("legacy_source_connection_mismatch", at+".identity.mxid")
				}
				name = mx
			} else {
				domain := str(binding["identity_domain"])
				star := str(identity["star"])
				local := strings.Join([]string{domain, str(product["id"]), str(m["project"]), roleCodes[role], star}, "_")
				if len(local) > 64 {
					return fail("localpart_overflow", at+".identity")
				}
				count := 0
				for _, r := range o.Reservations {
					if r.Domain == domain && r.Product == product["id"] && r.PrimaryRole == role && r.Star == star {
						if r.Holder != str(product["id"])+"/"+id {
							return fail("star_reservation_conflict", at+".identity.star")
						}
						count++
					}
				}
				if count != 1 {
					return fail("star_reservation_required", at+".identity.star")
				}
				name = "@" + local + ":" + domain
			}
			if a != nil {
				if resolved[name] {
					return fail("duplicate_resolved_identity", at+".identity")
				}
				resolved[name] = true
			}
		}
	}
	if keeperCount != 1 {
		return fail("reporting_line", "$.members")
	}
	for _, id := range sortedObjects(mi) {
		if e := CheckSuperiorHopBudget(id, edges); e != nil {
			return e
		}
	}
	for i, v := range projects {
		p := obj(v)
		at := index("$.projects", i)
		manager := mi[str(p["orchestrator"])]
		if manager == nil {
			return fail("reporting_line", at+".orchestrator")
		}
		if p["orchestrator_mode"] == "keeper" {
			if p["orchestrator"] != product["keeper"] || !text(p["orchestrator_ref"]) {
				return fail("reporting_line", at+".orchestrator")
			}
			for _, m := range mi {
				if m["project"] == p["id"] && m["role"] == "orchestrator" && m["desired_state"] == "active" {
					return fail("reporting_line", at+".orchestrator")
				}
			}
		} else {
			if manager["role"] != "orchestrator" || manager["project"] != p["id"] {
				return fail("reporting_line", at+".orchestrator")
			}
		}
	}
	return nil
}
func sortedObjects(m map[string]Object) []string {
	x := Object{}
	for k := range m {
		x[k] = true
	}
	return sorted(x)
}
func relocated(e *Issue, at string) *Issue { return &Issue{e.Rule, at, e.Exit} }
func checkA2a(a, top Object, at string, worker bool, m Object) *Issue {
	if worker && !text(m["chat_admission"]) {
		return fail("a2a_contract", at)
	}
	rooms := map[string]bool{}
	for _, v := range list(a["rooms"]) {
		rooms[str(v)] = true
	}
	if !rooms[str(a["parent_route"])] {
		return fail("a2a_contract", at+".parent_route")
	}
	// rooms[0] declares this member's own team room. Project worker-channel
	// declarations are not a second selector for a manager's migrated room.
	if dest, ok := a["result_destination"]; ok && !rooms[str(dest)] {
		return fail("a2a_contract", at+".result_destination")
	}
	spaces := map[string]bool{str(top["space"]): true}
	for i, v := range list(a["spaces"]) {
		if !rooms[str(v)] {
			return fail("a2a_contract", index(at+".spaces", i))
		}
		spaces[str(v)] = true
	}
	for _, group := range []string{"required", "self"} {
		for i, v := range list(obj(a["subscriptions"])[group]) {
			s := obj(v)
			loc := index(at+".subscriptions."+group, i)
			if s["all_events"] == true && len(list(s["rooms"])) == 0 {
				return fail("a2a_contract", loc+".rooms")
			}
			for _, r := range list(s["rooms"]) {
				if !rooms[str(r)] || spaces[str(r)] {
					return fail("a2a_contract", loc+".rooms")
				}
			}
			if _, ok := s["subtypes"]; ok {
				system := false
				for _, k := range list(s["kinds"]) {
					system = system || k == "system"
				}
				if !system {
					return fail("a2a_contract", loc+".subtypes")
				}
			}
		}
	}
	return nil
}
func resolveSource(ref, product Object, id string, o Options, identity bool) (string, Object, *Issue) {
	if ref["member"] != str(product["id"])+"/"+id {
		return "", nil, fail("legacy_source_key_mismatch", "")
	}
	var source *Source
	for i := range o.Sources {
		s := &o.Sources[i]
		if s.Kind == ref["kind"] && s.Revision == ref["revision"] {
			source = s
			break
		}
	}
	if source == nil {
		return "", nil, fail("legacy_source_missing", "")
	}
	sum := sha256.Sum256(source.Bytes)
	actual := hex.EncodeToString(sum[:])
	if actual != source.SHA256 || actual != ref["sha256"] {
		return "", nil, fail("source_digest_mismatch", "")
	}
	if source.Kind == "service-descriptor" {
		d, e := Parse(source.Bytes, 4)
		if e != nil {
			return "", nil, fail("legacy_source_invalid", "")
		}
		if e := CheckForbiddenFacts(d); e != nil {
			return "", nil, fail("legacy_source_invalid", "")
		}
		if d["member"] != ref["member"] {
			return "", nil, fail("legacy_source_key_missing", "")
		}
		if !text(d["admission_ref"]) || !mxid(d["mxid"]) || schemaConnection(d["connection"]) != nil {
			return "", nil, fail("legacy_source_invalid", "")
		}
		return str(d["mxid"]), obj(d["connection"]), nil
	}
	if Validate(source.Bytes, Options{Version: 3}) != nil {
		return "", nil, fail("legacy_source_invalid", "")
	}
	c, _ := Parse(source.Bytes, 3)
	if obj(c["product"])["id"] != product["id"] {
		return "", nil, fail("legacy_source_key_missing", "")
	}
	for _, v := range list(c["members"]) {
		m := obj(v)
		if m["id"] != id {
			continue
		}
		if !identity {
			return "", nil, nil
		}
		if obj(m["a2a"]) == nil {
			return "", nil, fail("legacy_source_mismatch", "")
		}
		bk := str(m["binding"])
		if bk == "" {
			bk = str(obj(c["product"])["default_binding"])
		}
		b := obj(obj(c["bindings"])[bk])
		conn := obj(obj(c["connections"])[str(b["connection"])])
		name := "@" + str(b["identity_prefix"]) + id + ":" + str(conn["server_name"])
		if !mxid(name) {
			return "", nil, fail("legacy_source_mismatch", "")
		}
		return name, conn, nil
	}
	return "", nil, fail("legacy_source_key_missing", "")
}
func schemaConnection(v any) *Issue {
	return object(map[string]validator{"homeserver": scalar(origin, "https_origin"), "server_name": scalar(dns, "domain_format")}, nil, true)(v, "$")
}
