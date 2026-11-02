package pulse

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	environmentSelector = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)
	pathScheme          = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)
)

// Declared paths are coordinates, never files to open. In particular, sibling
// paths may leave the card directory; the eventual consumer admits its own root.
func declaredRelative(v any) bool {
	s, ok := v.(string)
	return ok && s != "" && utf8.ValidString(s) && !strings.ContainsFunc(s, unicode.IsControl) &&
		!strings.HasPrefix(s, "/") && !strings.HasPrefix(s, "\\") && !strings.HasPrefix(s, "~") && !pathScheme.MatchString(s)
}

func onlyKeys(m object, allowed ...string) bool {
	for k := range m {
		found := false
		for _, a := range allowed {
			found = found || k == a
		}
		if !found {
			return false
		}
	}
	return true
}

func validEnvironmentExtension(member object) bool {
	v, present := member["pulse"]
	if !present {
		return true
	}
	p := obj(v)
	if p == nil || p["schema"] != "pulse-member-environment/v1" || !onlyKeys(p, "schema", "a2a_config_profile", "working_root", "network") {
		return false
	}
	if v, present := p["a2a_config_profile"]; present && !environmentSelector.MatchString(str(v)) {
		return false
	}
	if v, present := p["working_root"]; present && !declaredRelative(v) {
		return false
	}
	if v, present := p["network"]; present {
		n := obj(v)
		if n == nil {
			return false
		}
		switch n["mode"] {
		case "direct":
			return len(n) == 1
		case "proxy":
			return len(n) == 2 && environmentSelector.MatchString(str(n["ref"]))
		default:
			return false
		}
	}
	return true
}

func declaredEnvironment(card, member, binding, connection object, bindingKey, agent string) (object, error) {
	sourceID := str(obj(card["product"])["id"])
	profileKey := str(member["profile"])
	profile := obj(obj(card["profiles"])[profileKey])
	requirement, selected := obj(profile["requirement"]), obj(profile["selected"])
	var profileSource object
	if path, present := profile["path"]; present {
		if !declaredRelative(path) || !declaredRelative(card["profiles_root"]) {
			return nil, refusal("environment_invalid")
		}
		profileSource = object{"source": sourceID, "base": "profiles_root", "profiles_root": card["profiles_root"], "path": path}
	} else {
		registry := obj(profile["registry"])
		if !declaredRelative(registry["directory"]) {
			return nil, refusal("environment_invalid")
		}
		profileSource = object{"source": sourceID, "base": "registry", "repo": registry["repo"], "directory": registry["directory"]}
	}
	pulse := obj(member["pulse"])
	var root any
	path, origin := pulse["working_root"], "pulse.working_root"
	if path == nil {
		path, origin = member["clone"], "member.clone"
	}
	if path == nil {
		for _, value := range arr(card["projects"]) {
			project := obj(value)
			if project["id"] == member["project"] {
				path, origin = project["path"], "project.path"
			}
		}
	}
	if path != nil {
		if !declaredRelative(path) {
			return nil, refusal("environment_invalid")
		}
		root = object{"source": sourceID, "base": "card", "path": path, "origin": origin}
	}
	missing := []string{}
	if pulse["a2a_config_profile"] == nil {
		missing = append(missing, "a2a.config_profile")
	}
	if pulse["network"] == nil {
		missing = append(missing, "network")
	}
	if root == nil {
		missing = append(missing, "working_root")
	}
	sort.Strings(missing)
	state := "complete"
	if len(missing) != 0 {
		state = "incomplete"
	}
	runtime := obj(member["runtime"])
	return object{
		"schema": "pulse-environment/v1", "state": state, "missing": missing,
		"profile": object{"key": profileKey, "source": profileSource,
			"requirement": object{"context": requirement["context"], "compatible": requirement["compatible"]},
			"selected":    object{"version": selected["version"], "source_rev": selected["source_rev"], "lock_sha256": selected["lock_sha256"]}},
		"runtime": object{"kind": runtime["kind"], "model": runtime["model"], "effort": runtime["effort"]},
		"a2a": object{"agent": agent, "resolved_name": "@" + agent + ":" + str(connection["server_name"]),
			"binding": bindingKey, "operator": binding["operator"], "host": binding["host"],
			"connection": binding["connection"], "server_name": connection["server_name"], "config_profile": pulse["a2a_config_profile"]},
		"working_root": root, "network": pulse["network"],
	}, nil
}
