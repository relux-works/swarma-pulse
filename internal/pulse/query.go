// Package pulse provides explicit local unsigned member, environment, discovery, hierarchy and route views.
package pulse

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/relux-works/swarma-pulse/internal/cardv4"
)

const knownLimit = "caller not cryptographically verified; single-user local trust"

type object = map[string]any

func obj(v any) object          { x, _ := v.(map[string]any); return x }
func arr(v any) []any           { x, _ := v.([]any); return x }
func str(v any) string          { x, _ := v.(string); return x }
func refusal(rule string) error { return errors.New("RULE " + rule + ": $") }
func digest(raw []byte) string  { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }

// project validates a single exact source before resolving the unverified local
// context. No authority, remote lookup, extra source or relationship expansion exists.
func project(raw []byte, revision, cardRoot, agent string, now time.Time) ([]byte, error) {
	return projectView(raw, revision, cardRoot, agent, now, "")
}

func projectView(raw []byte, revision, cardRoot, agent string, now time.Time, view string) ([]byte, error) {
	return projectSelectedView(raw, revision, cardRoot, agent, now, view, "")
}

func projectSelectedView(raw []byte, revision, cardRoot, agent string, now time.Time, view, memberID string) ([]byte, error) {
	if memberID != "" && view != "environment" {
		return nil, refusal("usage")
	}
	if view == "hierarchy" || view == "routes" {
		return projectHierarchy(raw, revision, cardRoot, agent, now, view)
	}
	if view != "" && view != "environment" && view != "governance" {
		return nil, refusal("usage")
	}
	if revision != "sha256:"+digest(raw) {
		return nil, refusal("source_digest")
	}
	if agent == "" || len(agent) > 512 || !utf8.ValidString(agent) || strings.ContainsFunc(agent, unicode.IsControl) {
		return nil, refusal("caller_context")
	}
	var card object
	var err error
	if view == "governance" {
		card, err = decodeGovernanceCard(raw, revision, now)
	} else if view == "environment" {
		card, err = decodeEnvironmentCard(raw, revision, now, memberID != "")
	} else {
		card, err = decodeQueryCard(raw, revision, view == "environment")
	}
	if err != nil {
		return nil, err
	}
	return projectCheckedCard(card, raw, revision, cardRoot, agent, now, view, memberID)
}

// Both entry paths validate the original complete document before sharing this
// self-only projection. No version relabeling or partial-card validation occurs.
func projectCheckedCard(card object, raw []byte, revision, cardRoot, agent string, now time.Time, view, memberID string) ([]byte, error) {
	product := cardHeader(card)
	bindings := obj(card["bindings"])
	connections := obj(card["connections"])
	v5 := card["schema"] == "product-card/v5"
	modern := card["schema"] == "product-card/v4" || v5
	var self, binding, connection object
	bindingKey := ""
	matches := 0
	for _, value := range arr(card["members"]) {
		member := obj(value)
		if obj(member["a2a"]) == nil || v5 && member["kind"] == "human" {
			continue
		}
		key := str(member["binding"])
		if key == "" {
			key = str(product["default_binding"])
		}
		b := obj(bindings[key])
		localpart := str(b["identity_prefix"]) + str(member["id"])
		if modern {
			name := cardv4.DeclaredMXID(card, member)
			localpart, _, _ = strings.Cut(strings.TrimPrefix(name, "@"), ":")
		}
		if localpart != agent {
			continue
		}
		matches++
		self = member
		binding = b
		bindingKey = key
		connection = obj(connections[str(b["connection"])])
	}
	if matches == 0 {
		return nil, refusal("caller_unknown")
	}
	if matches != 1 {
		return nil, refusal("caller_ambiguous")
	}
	// The invoking caller remains fixed. A declared target changes only the
	// projection subject, after complete validation and caller eligibility.
	if view == "environment" && v5 && self["responsibility"] == "service" {
		return nil, refusal("environment_invalid")
	}
	subjectAgent := agent
	var resolvedName any = "@" + agent + ":" + str(connection["server_name"])
	if memberID != "" {
		var target object
		for _, value := range arr(card["members"]) {
			if member := obj(value); member["id"] == memberID {
				target = member
				break
			}
		}
		if target == nil {
			return nil, refusal("target_unknown")
		}
		if target["kind"] != "agent" || target["responsibility"] == "service" {
			return nil, refusal("target_ineligible")
		}
		manager := self["responsibility"] == "keeper" || self["responsibility"] == "orchestrator"
		if target["id"] != self["id"] && (!manager || target["parent"] != self["id"]) {
			return nil, refusal("target_forbidden")
		}
		self = target
		bindingKey = str(self["binding"])
		binding = obj(bindings[bindingKey])
		connection = obj(connections[str(binding["connection"])])
		subjectAgent, resolvedName = "", nil
		if obj(self["a2a"]) != nil {
			name := cardv4.DeclaredMXID(card, self)
			subjectAgent, _, _ = strings.Cut(strings.TrimPrefix(name, "@"), ":")
			resolvedName = name
		}
	}
	key := str(product["id"]) + "/" + str(self["id"])
	scope := "member"
	role := self["role"]
	if v5 {
		role = self["responsibility"]
	}
	switch role {
	case "keeper":
		scope = "product"
	case "orchestrator":
		scope = "project"
	}
	if card["schema"] == "company-card/v1" {
		scope = "company"
	}
	profile := obj(obj(card["profiles"])[str(self["profile"])])
	requirement, selected := obj(profile["requirement"]), obj(profile["selected"])
	viewSelf := object{"key": key, "role": self["role"], "scope": scope, "desired_state": self["desired_state"],
		"resolved_name":     resolvedName,
		"execution_binding": object{"key": bindingKey, "operator": binding["operator"], "host": binding["host"], "connection": binding["connection"]},
		"profile":           object{"requirement": object{"context": requirement["context"], "compatible": requirement["compatible"]}, "selected": object{"version": selected["version"], "source_rev": selected["source_rev"], "lock_sha256": selected["lock_sha256"]}},
		"project":           nil, "project_state": "undeclared"}
	if view == "governance" || view == "environment" && v5 {
		viewSelf["kind"], viewSelf["responsibility"] = "agent", role
		if role == "service" {
			viewSelf["profile"] = nil
		}
	}
	projects := []any{}
	if projectID := str(self["project"]); projectID != "" {
		for _, value := range arr(card["projects"]) {
			p := obj(value)
			if p["id"] != projectID {
				continue
			}
			pkey := str(product["id"]) + "/" + projectID
			viewSelf["project"] = pkey
			viewSelf["project_state"] = "declared"
			location := object{"kind": "local", "path": p["path"]}
			if modern {
				declared := obj(p["location"])
				location = object{"kind": declared["kind"]}
				for _, k := range []string{"path", "operator", "execution_host", "project_locator", "checkout_path"} {
					if value, present := declared[k]; present {
						location[k] = value
					}
				}
			}
			projects = append(projects, object{"key": pkey, "repo": p["repo"], "desired_state": p["desired_state"], "location": location})
		}
		if len(projects) != 1 {
			return nil, refusal("self_project_missing")
		}
	}
	result := object{"schema": "pulse-view/v1", "mode": "local-unsigned", "product": product["id"], "member": key,
		"caller": object{"agent": agent, "cryptographically_verified": false, "trust": "single-user local trust"}, "known_limit": knownLimit,
		"source":            object{"schema": card["schema"], "revision": revision, "sha256": digest(raw), "card_root": filepath.Clean(cardRoot), "freshness": "unknown", "observed_at": nil},
		"query_observed_at": now.UTC().Format(time.RFC3339Nano), "self": viewSelf, "peers": []any{}, "direct_reports": []any{}, "descendants": []any{}, "superiors": []any{}, "projects": projects,
		// This constant does not reveal hidden node counts or edge existence.
		"hierarchy_status": "self-only; hierarchy not disclosed",
		"observations":     object{"runtime": object{"state": "unknown"}, "registration": object{"state": "unknown"}, "installed_profile": object{"state": "unknown"}, "assignment": object{"state": "unknown"}}}
	if modern {
		obj(result["source"])["validation"] = "local declarations only; source attestations and star reservations unverified"
	}
	if view == "governance" {
		result["schema"] = "pulse-view/v5"
		result["governance"] = declaredGovernance(card)
	}
	if view == "environment" {
		projectEnvironment := declaredEnvironment
		if memberID != "" {
			projectEnvironment = targetedEnvironment
		}
		environment, err := projectEnvironment(card, self, binding, connection, bindingKey, subjectAgent)
		if err != nil {
			return nil, err
		}
		viewSelf["level"] = scope
		viewSelf["environment"] = environment
		source := obj(result["source"])
		source["id"] = product["id"]
		delete(result, "source")
		delete(result, "product")
		delete(result, "member")
		result["schema"] = "pulse-view/v2"
		if v5 {
			result["schema"] = "pulse-view/v6"
		}
		result["query"] = object{"view": "environment"}
		if memberID != "" {
			result["schema"] = "pulse-view/v7"
			result["query"] = object{"view": "environment", "member": memberID}
			result["target"] = viewSelf
			delete(result, "self")
			result["hierarchy_status"] = "target-only; hierarchy not disclosed"
		}
		result["sources"] = []any{source}
		result["products"] = []any{}
		result["boundaries"] = []any{}
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, refusal("output_encoding")
	}
	return append(encoded, '\n'), nil
}
