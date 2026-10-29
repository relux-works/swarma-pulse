package cardv4

import (
	"path/filepath"
	"reflect"
	"strings"
)

// PreservationInputs are explicit relocation/mapping premises. This checker
// compares supplied documents; it does not convert or publish a document.
type PreservationInputs struct {
	SourceRevision            string            `json:"source_revision"`
	SourceSHA256              string            `json:"source_sha256"`
	HostProject               string            `json:"host_project"`
	SourceCardRoot            string            `json:"source_card_root"`
	TargetCardRoot            string            `json:"target_card_root"`
	AdmittedRoot              string            `json:"admitted_root"`
	ExternalParentMapping     map[string]Object `json:"external_parent_mapping"`
	AdditionalConnections     Object            `json:"additional_connections"`
	AdditionalBindings        Object            `json:"additional_bindings"`
	ProspectiveNonchatChoices map[string]Object `json:"prospective_nonchat_choices"`
}

func copyObject(m Object) Object {
	out := Object{}
	for k, v := range m {
		out[k] = v
	}
	return out
}
func indexedObjects(a []any, key string) map[string]Object {
	out := map[string]Object{}
	for _, v := range a {
		m := obj(v)
		out[str(m[key])] = m
	}
	return out
}
func equalPaths(baseA, a, baseB, b string) bool {
	return relative(a) && relative(b) && filepath.Clean(filepath.Join(baseA, a)) == filepath.Clean(filepath.Join(baseB, b))
}

// VerifyConversionPreservation compares every source desired fact, rejecting
// unsupported extension keys rather than losing them. Input validation is separate.
func VerifyConversionPreservation(source, target Object, in PreservationInputs) *Issue {
	if in.HostProject == "" || in.SourceCardRoot == "" || in.TargetCardRoot == "" || in.AdmittedRoot == "" || !revision.MatchString(in.SourceRevision) || !digest.MatchString(in.SourceSHA256) {
		return shape("conversion_inputs_missing", "$")
	}
	for _, root := range []string{in.SourceCardRoot, in.TargetCardRoot} {
		if !beneath(in.AdmittedRoot, root) {
			return fail("path_root_escape", "$")
		}
	}
	sp, tp := copyObject(obj(source["product"])), copyObject(obj(target["product"]))
	if tp["host_project"] != in.HostProject {
		return fail("conversion_product_preservation", "$.product.host_project")
	}
	delete(tp, "host_project")
	if !reflect.DeepEqual(sp, tp) {
		return fail("conversion_product_preservation", "$.product")
	}
	ext := []any{}
	for _, v := range list(source["external_members"]) {
		mapped := in.ExternalParentMapping[str(v)]
		if mapped == nil {
			return fail("conversion_external_mapping_required", "$.external_members")
		}
		ext = append(ext, mapped)
	}
	if !reflect.DeepEqual(ext, target["external_members"]) {
		return fail("conversion_external_preservation", "$.external_members")
	}
	expectedConnections := copyObject(obj(source["connections"]))
	for k, v := range in.AdditionalConnections {
		if old, ok := expectedConnections[k]; ok && !reflect.DeepEqual(old, v) {
			return fail("conversion_connection_preservation", "$.connections")
		}
		expectedConnections[k] = v
	}
	if !reflect.DeepEqual(expectedConnections, target["connections"]) {
		return fail("conversion_connection_preservation", "$.connections")
	}
	expectedBindings := Object{}
	for k, v := range obj(source["bindings"]) {
		b := copyObject(obj(v))
		conn := obj(obj(source["connections"])[str(b["connection"])])
		delete(b, "identity_prefix")
		b["identity_domain"] = conn["server_name"]
		expectedBindings[k] = b
	}
	for k, v := range in.AdditionalBindings {
		if old, ok := expectedBindings[k]; ok && !reflect.DeepEqual(old, v) {
			return fail("conversion_binding_preservation", "$.bindings")
		}
		expectedBindings[k] = v
	}
	if !reflect.DeepEqual(expectedBindings, target["bindings"]) {
		return fail("conversion_binding_preservation", "$.bindings")
	}
	if !equalPaths(in.SourceCardRoot, str(source["profiles_root"]), in.TargetCardRoot, str(target["profiles_root"])) {
		return fail("conversion_path_preservation", "$.profiles_root")
	}
	if !reflect.DeepEqual(source["profiles"], target["profiles"]) {
		return fail("conversion_profile_selection_preservation", "$.profiles")
	}
	oldProjects, newProjects := indexedObjects(list(source["projects"]), "id"), indexedObjects(list(target["projects"]), "id")
	if len(oldProjects) != len(newProjects) || len(list(source["projects"])) != len(list(target["projects"])) || len(oldProjects) != len(list(source["projects"])) {
		return fail("conversion_project_preservation", "$.projects")
	}
	for i, v := range list(source["projects"]) {
		p := obj(v)
		t := newProjects[str(p["id"])]
		if t == nil {
			return fail("conversion_project_preservation", index("$.projects", i))
		}
		a, b := copyObject(p), copyObject(t)
		loc := obj(t["location"])
		if loc["kind"] != "local" {
			return fail("conversion_location_disposition_required", index("$.projects", i)+".location")
		}
		if !equalPaths(in.SourceCardRoot, str(p["path"]), in.TargetCardRoot, str(loc["path"])) {
			return fail("conversion_path_preservation", index("$.projects", i)+".location")
		}
		delete(a, "path")
		delete(b, "location")
		ta, tb := copyObject(obj(a["task_system"])), copyObject(obj(b["task_system"]))
		if !equalPaths(in.SourceCardRoot, str(ta["path"]), filepath.Join(in.TargetCardRoot, str(loc["path"])), str(tb["path"])) {
			return fail("conversion_path_preservation", index("$.projects", i)+".task_system.path")
		}
		delete(ta, "path")
		delete(tb, "path")
		a["task_system"], b["task_system"] = ta, tb
		if !reflect.DeepEqual(a, b) {
			return fail("conversion_project_preservation", index("$.projects", i))
		}
	}
	oldMembers, newMembers := indexedObjects(list(source["members"]), "id"), indexedObjects(list(target["members"]), "id")
	if len(oldMembers) != len(newMembers) {
		return fail("conversion_member_preservation", "$.members")
	}
	for i, v := range list(source["members"]) {
		m := obj(v)
		t := newMembers[str(m["id"])]
		at := index("$.members", i)
		if t == nil {
			return fail("conversion_member_preservation", at)
		}
		if !sameOptional(m, t, "station") {
			return fail("conversion_station_preservation", at+".station")
		}
		sa, ta := obj(m["a2a"]), obj(t["a2a"])
		if sa != nil {
			for _, g := range []string{"required", "self"} {
				if !reflect.DeepEqual(obj(sa["subscriptions"])[g], obj(ta["subscriptions"])[g]) {
					rule := "conversion_required_selectors_preservation"
					if g == "self" {
						rule = "conversion_selector_ownership_preservation"
					}
					return fail(rule, at+".a2a.subscriptions."+g)
				}
			}
		}
		a, b := copyObject(m), copyObject(t)
		for _, key := range []string{"clone", "git_dir"} {
			av, aok := a[key]
			bv, bok := b[key]
			if aok != bok || aok && !equalPaths(in.SourceCardRoot, str(av), in.TargetCardRoot, str(bv)) {
				return fail("conversion_path_preservation", at+"."+key)
			}
			delete(a, key)
			delete(b, key)
		}
		binding := str(m["binding"])
		if binding == "" {
			binding = str(sp["default_binding"])
		}
		project := m["project"]
		if m["role"] == "keeper" {
			project = in.HostProject
			a["parent"] = in.ExternalParentMapping[str(m["parent"])]
		}
		a["project"] = project
		if sa != nil {
			pin := obj(obj(t["identity"])["source_ref"])
			if pin["kind"] != "v3-card" || pin["revision"] != in.SourceRevision || pin["sha256"] != in.SourceSHA256 || pin["member"] != str(sp["id"])+"/"+str(m["id"]) {
				return fail("conversion_source_pin_preservation", at+".identity.source_ref")
			}
			if obj(t["identity"])["scheme"] != "legacy-exact" || t["binding"] != binding {
				return fail("conversion_identity_preservation", at+".identity")
			}
		} else {
			choice := in.ProspectiveNonchatChoices[str(sp["id"])+"/"+str(m["id"])]
			if choice == nil || !reflect.DeepEqual(choice["identity"], t["identity"]) || choice["binding"] != t["binding"] {
				return fail("conversion_identity_preservation", at+".identity")
			}
		}
		for _, key := range []string{"scope", "binding", "identity", "logical_source_ref"} {
			delete(b, key)
		}
		delete(a, "binding")
		if !reflect.DeepEqual(a, b) {
			return fail("conversion_member_preservation", at)
		}
	}
	// Unknown top-level source extensions are deliberately unsupported.
	known := words("schema product external_members connections bindings profiles_root profiles projects members")
	for k := range source {
		if !known[k] {
			return fail("conversion_extension_unsupported", "$")
		}
	}
	return nil
}
func sameOptional(a, b Object, k string) bool {
	x, ax := a[k]
	y, by := b[k]
	return ax == by && reflect.DeepEqual(x, y)
}

// VerifyPilotPreservation permits only the chosen member's identity, binding and
// associated a2a facts to change. A removed logical pin or changed station fails.
func VerifyPilotPreservation(prior, target Object, pilot string) *Issue {
	p, id, ok := strings.Cut(pilot, "/")
	if !ok || obj(prior["product"])["id"] != p || obj(target["product"])["id"] != p {
		return fail("pilot_member_missing", "$.members")
	}
	a, b := copyObject(prior), copyObject(target)
	delete(a, "members")
	delete(b, "members")
	if !reflect.DeepEqual(a, b) {
		return fail("nonpilot_preservation", "$")
	}
	old, new := list(prior["members"]), list(target["members"])
	if len(old) != len(new) {
		return fail("nonpilot_preservation", "$.members")
	}
	found := false
	for i, v := range old {
		before, after := copyObject(obj(v)), copyObject(obj(new[i]))
		if before["id"] == id {
			found = true
			for _, k := range []string{"identity", "binding", "a2a"} {
				delete(before, k)
				delete(after, k)
			}
		}
		if !reflect.DeepEqual(before, after) {
			return fail("nonpilot_preservation", index("$.members", i))
		}
	}
	if !found {
		return fail("pilot_member_missing", "$.members")
	}
	return nil
}
