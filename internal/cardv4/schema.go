package cardv4

import "strings"

type validator func(any, string) *Issue

func scalar(check func(any) bool, rule string) validator {
	return func(v any, at string) *Issue {
		if !check(v) {
			return fail(rule, at)
		}
		return nil
	}
}
func enumeration(values string) validator {
	allowed := words(values)
	return scalar(func(v any) bool { return allowed[str(v)] }, "enum_value")
}
func array(v validator, nonempty bool) validator {
	return func(x any, at string) *Issue {
		a, ok := x.([]any)
		if !ok {
			return shape("array_shape", at)
		}
		if nonempty && len(a) == 0 {
			return fail("required_nonempty", at)
		}
		for i, c := range a {
			if e := v(c, index(at, i)); e != nil {
				return e
			}
		}
		return nil
	}
}
func mapping(v validator, nonempty bool) validator {
	return func(x any, at string) *Issue {
		m, ok := x.(map[string]any)
		if !ok {
			return shape("object_shape", at)
		}
		if nonempty && len(m) == 0 {
			return fail("required_nonempty", at)
		}
		for i, k := range sorted(m) {
			if !text(k) {
				return fail("text_format", index(at, i))
			}
			if e := v(m[k], index(at, i)); e != nil {
				return e
			}
		}
		return nil
	}
}
func object(required, optional map[string]validator, closed bool) validator {
	return func(x any, at string) *Issue {
		m, ok := x.(map[string]any)
		if !ok {
			return shape("object_shape", at)
		}
		if closed {
			for i, k := range sorted(m) {
				if _, ok := required[k]; !ok {
					if _, ok := optional[k]; !ok {
						return fail("unknown_field", index(at, i))
					}
				}
			}
		}
		for _, k := range sortedValidators(required) {
			v, ok := m[k]
			if !ok {
				return fail("missing_field", field(at, k))
			}
			if e := required[k](v, field(at, k)); e != nil {
				return e
			}
		}
		for _, k := range sortedValidators(optional) {
			if v, ok := m[k]; ok {
				if e := optional[k](v, field(at, k)); e != nil {
					return e
				}
			}
		}
		return nil
	}
}
func sortedValidators(m map[string]validator) []string {
	keys := Object{}
	for k := range m {
		keys[k] = true
	}
	return sorted(keys)
}
func fields(names string, v validator) map[string]validator {
	m := map[string]validator{}
	for _, s := range strings.Fields(names) {
		m[s] = v
	}
	return m
}
func combine(a, b map[string]validator) map[string]validator {
	out := map[string]validator{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}
func schema(version int) validator {
	closed := version == 4
	txt := scalar(text, "text_format")
	if !closed {
		txt = scalar(logical, "text_format")
	}
	id := scalar(logical, "logical_id_format")
	sl := scalar(func(v any) bool { return slug.MatchString(str(v)) && len(str(v)) <= 40 }, "slug_format")
	path := scalar(relative, "relative_path")
	rm := scalar(room, "room_format")
	mx := scalar(mxid, "mxid_format")
	dom := scalar(dns, "domain_format")
	repository := scalar(repo, "repository_format")
	versionFormat := func(v any) bool { return semver.MatchString(str(v)) }
	if !closed {
		path = scalar(func(v any) bool {
			s := str(v)
			return logical(v) && !strings.HasPrefix(s, "/") && !strings.HasPrefix(s, "~")
		}, "relative_path")
		rm = scalar(func(v any) bool {
			p := strings.Split(str(v), ":")
			return len(p) <= 2 && (aliasLocal.MatchString(p[0]) || roomLocal.MatchString(p[0])) && (len(p) == 1 || legacyServer.MatchString(p[1]))
		}, "room_format")
		mx = scalar(func(v any) bool {
			p := strings.Split(str(v), ":")
			return len(p) == 2 && mxidLocal.MatchString(p[0]) && legacyServer.MatchString(p[1])
		}, "mxid_format")
		repository = txt
		versionFormat = func(v any) bool { return legacyVersion.MatchString(str(v)) }
	}
	boolean := scalar(func(v any) bool { _, ok := v.(bool); return ok }, "boolean_type")
	objv := func(req, opt map[string]validator) validator { return object(req, opt, closed) }
	state := enumeration("active parked retired")
	role := enumeration("keeper orchestrator developer tester architect reviewer researcher")
	if closed {
		role = enumeration("keeper orchestrator developer tester architect reviewer researcher service")
	}
	source := objv(map[string]validator{"kind": enumeration("v3-card service-descriptor"), "revision": scalar(func(v any) bool { return revision.MatchString(str(v)) }, "revision_format"), "sha256": scalar(func(v any) bool { return digest.MatchString(str(v)) }, "digest_format"), "member": id}, nil)
	external := objv(map[string]validator{"product": sl, "member": id}, nil)
	parent := func(v any, at string) *Issue {
		if _, ok := v.(string); ok {
			return id(v, at)
		}
		return external(v, at)
	}
	profile := func(v any, at string) *Issue {
		e := objv(map[string]validator{"requirement": objv(map[string]validator{"context": txt, "compatible": scalar(func(v any) bool { return validRange(str(v), !closed) }, "version_range_format")}, nil), "selected": objv(map[string]validator{"version": scalar(versionFormat, "version_format"), "source_rev": scalar(func(v any) bool { return commit.MatchString(str(v)) }, "revision_format"), "lock_sha256": scalar(func(v any) bool { return digest.MatchString(str(v)) }, "digest_format")}, nil)}, map[string]validator{"path": path, "registry": objv(map[string]validator{"repo": repository, "directory": path}, nil)})(v, at)
		if e != nil {
			return e
		}
		m := obj(v)
		_, p := m["path"]
		_, r := m["registry"]
		if p == r {
			return fail("profile_source", at)
		}
		if !compatibleVersion(str(obj(m["selected"])["version"]), str(obj(m["requirement"])["compatible"]), !closed) {
			return fail("profile_version_incompatible", field(at, "selected.version"))
		}
		return nil
	}
	topology := objv(map[string]validator{"connection": txt, "space": rm, "lobby": rm, "escalations": rm}, map[string]validator{"grouped_rooms": array(rm, false), "upward_escalations": rm})
	connection := objv(map[string]validator{"homeserver": scalar(origin, "https_origin"), "server_name": dom}, nil)
	if !closed {
		connection = objv(map[string]validator{"homeserver": scalar(func(v any) bool { return legacyHTTPS.MatchString(str(v)) }, "https_origin"), "server_name": txt}, nil)
	}
	bindingFields := fields("operator host connection", txt)
	if closed {
		bindingFields["identity_domain"] = dom
	} else {
		bindingFields["identity_prefix"] = txt
	}
	binding := objv(bindingFields, nil)
	selector := func(v any, at string) *Issue {
		m := obj(v)
		if _, human := m["from_mxid"]; human {
			return objv(map[string]validator{"from_mxid": mx}, map[string]validator{"rooms": array(rm, true), "all_events": scalar(func(v any) bool { return v == true }, "all_events_true")})(v, at)
		}
		return objv(map[string]validator{"rooms": array(rm, true), "kinds": array(enumeration("proposal question answer decision ack notice offer request accept decline handover subscribe unsubscribe owner-needed system"), true)}, map[string]validator{"subtypes": array(txt, true)})(v, at)
	}
	a2a := func(service bool) validator {
		req := map[string]validator{"connection": txt, "parent_route": rm, "rooms": array(rm, true), "subscriptions": objv(map[string]validator{"required": array(selector, false), "self": array(selector, false)}, nil)}
		if service {
			req["result_destination"] = rm
		}
		return objv(req, map[string]validator{"spaces": array(rm, false)})
	}
	slot := objv(map[string]validator{"role": enumeration("keeper orchestrator developer tester architect reviewer researcher"), "profile": txt, "count": scalar(func(v any) bool { n, ok := v.(int64); return ok && n >= 0 }, "count_format"), "lifecycle": enumeration("persistent on-demand"), "context_mode": enumeration("continue fresh")}, map[string]validator{"independent_of_producer": boolean})
	project := func(v any, at string) *Issue {
		req := map[string]validator{"id": txt, "description": txt, "repo": scalar(func(v any) bool { return v == "local" || repo(v) }, "repository_format"), "desired_state": state, "orchestrator": id, "task_system": nil, "channels": objv(map[string]validator{"mode": enumeration("shared separate"), "team_room": rm}, nil)}
		taskFields := map[string]validator{"kind": enumeration("task-board tasks-md")}
		if closed {
			req["id"] = sl
			loc := obj(obj(v)["location"])
			if loc["kind"] == "remote" {
				req["location"] = objv(combine(fields("operator execution_host project_locator", txt), map[string]validator{"kind": enumeration("remote")}), map[string]validator{"checkout_path": path})
				taskFields["locator"] = txt
			} else {
				req["location"] = objv(map[string]validator{"kind": enumeration("local"), "path": path}, nil)
				taskFields["path"] = path
			}
		} else {
			req["repo"] = txt
			req["path"] = path
			taskFields["path"] = path
		}
		req["task_system"] = objv(taskFields, nil)
		return objv(req, map[string]validator{"desired_state_ref": txt, "orchestrator_mode": enumeration("member keeper"), "orchestrator_ref": txt})(v, at)
	}
	member := func(v any, at string) *Issue {
		m := obj(v)
		service := closed && m["role"] == "service"
		req := map[string]validator{"id": id, "role": role, "parent": id, "lifecycle": enumeration("persistent on-demand"), "desired_state": state, "context_mode": enumeration("continue fresh"), "runtime": objv(map[string]validator{"kind": enumeration("codex_cli claude_code"), "model": txt}, map[string]validator{"effort": txt})}
		opt := combine(fields("desired_state_ref station binding project", txt), map[string]validator{"capabilities": array(role, false), "clone": path, "git_dir": path})
		if closed {
			req["project"] = sl
			req["binding"] = txt
			req["parent"] = parent
			req["scope"] = enumeration("product project member")
			req["identity"] = func(v any, at string) *Issue {
				if obj(v)["scheme"] == "legacy-exact" {
					return objv(map[string]validator{"scheme": enumeration("legacy-exact"), "mxid": mx, "source_ref": source}, nil)(v, at)
				}
				return objv(map[string]validator{"scheme": enumeration("domain-star-v1"), "star": sl}, nil)(v, at)
			}
			delete(opt, "project")
			delete(opt, "binding")
			opt["logical_source_ref"] = source
		}
		if service {
			for _, k := range []string{"profile", "staffing", "independent_of_producer", "chat_admission"} {
				if _, ok := m[k]; ok {
					return fail("service_"+k+"_forbidden", field(at, k))
				}
			}
			req["context_mode"] = enumeration("none")
			req["runtime"] = func(v any, at string) *Issue {
				r := obj(v)
				if _, ok := r["model"]; ok {
					return fail("service_runtime_forbidden", at)
				}
				if _, ok := r["effort"]; ok {
					return fail("service_runtime_forbidden", at)
				}
				return objv(map[string]validator{"kind": enumeration("none")}, nil)(v, at)
			}
			req["service_contract_ref"] = txt
			req["result_route"] = enumeration("a2a parent-native")
			opt["a2a"] = a2a(true)
			opt["native_result_ref"] = txt
			opt["selector_policy_ref"] = txt
		} else {
			req["profile"] = txt
			opt["a2a"] = a2a(false)
			opt["staffing"] = array(slot, false)
			opt["independent_of_producer"] = boolean
			opt["result_route"] = enumeration("a2a parent-native")
			opt["chat_admission"] = txt
		}
		return objv(req, opt)(v, at)
	}
	productFields := fields("id description privacy keeper default_binding", txt)
	if closed {
		productFields["id"] = sl
		productFields["keeper"] = id
		productFields["host_project"] = sl
	}
	productFields["topology"] = topology
	ext := validator(external)
	if !closed {
		ext = id
	}
	return objv(map[string]validator{"schema": enumeration("product-card/v" + string(rune('0'+version))), "product": objv(productFields, nil), "external_members": array(ext, false), "connections": mapping(connection, true), "bindings": mapping(binding, true), "profiles_root": path, "profiles": mapping(profile, true), "projects": array(project, true), "members": array(member, true)}, nil)
}
