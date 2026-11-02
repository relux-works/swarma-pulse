package cardv4

import "strings"

// ValidateCompanyCore reuses the v3 field predicates for company members and
// local graph facts, without presenting a company as a product to the v3 reader.
// Product inventory coordinates and cross-card composition are checked by Pulse.
func ValidateCompanyCore(c Object) *Issue {
	if e := CheckForbiddenFacts(c); e != nil {
		return e
	}
	if len(list(c["members"])) > 10000 {
		return fail("members_limit", "$")
	}
	if e := cardSchema(3, true)(c, "$"); e != nil {
		return e
	}
	h := obj(c["company"])
	bindings := obj(c["bindings"])
	connections := obj(c["connections"])
	profiles := obj(c["profiles"])
	if bindings[str(h["default_binding"])] == nil {
		return fail("binding_reference", "$")
	}
	for _, v := range bindings {
		if connections[str(obj(v)["connection"])] == nil {
			return fail("connection_reference", "$")
		}
	}
	members := map[string]Object{}
	external := map[string]bool{}
	for _, v := range list(c["external_members"]) {
		external[str(v)] = true
	}
	for _, v := range list(c["members"]) {
		m := obj(v)
		id := str(m["id"])
		if members[id] != nil {
			return fail("duplicate_member", "$")
		}
		members[id] = m
	}
	keeper := members[str(h["keeper"])]
	if keeper == nil || keeper["role"] != "keeper" {
		return fail("reporting_line", "$")
	}
	edges := map[string]string{}
	for _, v := range list(c["members"]) {
		m := obj(v)
		id := str(m["id"])
		role := str(m["role"])
		manager := role == "keeper" || role == "orchestrator"
		if m["parent"] == nil {
			if id != h["keeper"] {
				return fail("reporting_line", "$")
			}
		} else {
			parent := str(m["parent"])
			if members[parent] != nil {
				edges[id] = parent
			} else if !external[parent] {
				return fail("reporting_line", "$")
			}
		}
		if profiles[str(m["profile"])] == nil {
			return fail("profile_reference", "$")
		}
		bkey := str(m["binding"])
		if bkey == "" {
			bkey = str(h["default_binding"])
		}
		b := obj(bindings[bkey])
		if b == nil {
			return fail("binding_reference", "$")
		}
		for _, value := range bindings {
			prefix := str(obj(value)["identity_prefix"])
			if prefix != "" && strings.HasPrefix(id, prefix) {
				return fail("host_prefixed_id", "$")
			}
		}
		if role == "reviewer" && (m["context_mode"] != "fresh" || m["independent_of_producer"] != true) {
			return fail("reviewer_independence", "$")
		}
		if _, present := m["staffing"]; present && !manager {
			return fail("staffing_manager_only", "$")
		}
		for _, value := range list(m["staffing"]) {
			s := obj(value)
			if profiles[str(s["profile"])] == nil {
				return fail("profile_reference", "$")
			}
			if s["role"] == "reviewer" && (s["context_mode"] != "fresh" || s["independent_of_producer"] != true) {
				return fail("reviewer_independence", "$")
			}
		}
		a := obj(m["a2a"])
		route := str(m["result_route"])
		if manager {
			if a == nil {
				return fail("route_contract", "$")
			}
		} else if route != "parent-native" && route != "a2a" || route == "parent-native" && a != nil || route == "a2a" && (a == nil || !text(m["chat_admission"])) {
			return fail("route_contract", "$")
		}
		if a != nil {
			if _, present := a["workspace"]; present {
				return fail("retired_workspace", "$")
			}
			if a["connection"] != b["connection"] {
				return fail("binding_connection_mismatch", "$")
			}
			if e := checkA2a(a, Object{}, "$", !manager, m); e != nil {
				return e
			}
		}
	}
	// Color only local company edges once; the universe enforces the separate
	// reporting-hop budget after explicit cross-card composition.
	done := map[string]bool{}
	for id := range members {
		path := map[string]bool{}
		chain := []string{}
		for at := id; at != "" && !done[at]; at = edges[at] {
			if path[at] {
				return fail("reporting_cycle", "$")
			}
			path[at] = true
			chain = append(chain, at)
		}
		for _, at := range chain {
			done[at] = true
		}
	}
	return nil
}
