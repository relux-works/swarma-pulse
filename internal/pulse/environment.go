package pulse

import (
	"sort"

	"github.com/relux-works/swarma-pulse/internal/cardv4"
)

// A shared predicate keeps legacy and strict-v5 declaration syntax identical.
func declaredRelative(v any) bool { return cardv4.EnvironmentRelative(v) }

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
	value, present := member["pulse"]
	return !present || cardv4.CheckEnvironmentExtension(value, "$") == nil
}

func declaredEnvironment(card, member, binding, connection object, bindingKey, agent string) (object, error) {
	sourceID := cardID(card)
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
				if card["schema"] == "product-card/v5" {
					location := obj(project["location"])
					if location["kind"] == "local" {
						path, origin = location["path"], "project.location.path"
					}
				} else {
					path, origin = project["path"], "project.path"
				}
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

// Target mode has a distinct nullable-identity schema. Prospective binding facts
// are preserved without manufacturing chat admission from a declared identity.
func targetedEnvironment(card, member, binding, connection object, bindingKey, subjectAgent string) (object, error) {
	environment, err := declaredEnvironment(card, member, binding, connection, bindingKey, subjectAgent)
	if err != nil {
		return nil, err
	}
	environment["schema"] = "pulse-environment/v2"
	if obj(member["a2a"]) == nil {
		a2a := obj(environment["a2a"])
		a2a["agent"], a2a["resolved_name"] = nil, nil
		missing := append(environment["missing"].([]string), "a2a.agent", "a2a.resolved_name")
		sort.Strings(missing)
		environment["missing"], environment["state"] = missing, "incomplete"
	}
	return environment, nil
}
