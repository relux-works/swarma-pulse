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
	if view == "hierarchy" || view == "routes" {
		return projectHierarchy(raw, revision, cardRoot, agent, now, view)
	}
	if view != "" && view != "environment" {
		return nil, refusal("usage")
	}
	if revision != "sha256:"+digest(raw) {
		return nil, refusal("source_digest")
	}
	if agent == "" || len(agent) > 512 || !utf8.ValidString(agent) || strings.ContainsFunc(agent, unicode.IsControl) {
		return nil, refusal("caller_context")
	}
	card, err := decodeQueryCard(raw, revision, view == "environment")
	if err != nil {
		return nil, err
	}
	product := cardHeader(card)
	bindings := obj(card["bindings"])
	connections := obj(card["connections"])
	v4 := card["schema"] == "product-card/v4"
	var self, binding, connection object
	bindingKey := ""
	matches := 0
	for _, value := range arr(card["members"]) {
		member := obj(value)
		if obj(member["a2a"]) == nil {
			continue
		}
		key := str(member["binding"])
		if key == "" {
			key = str(product["default_binding"])
		}
		b := obj(bindings[key])
		localpart := str(b["identity_prefix"]) + str(member["id"])
		if v4 {
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
	key := str(product["id"]) + "/" + str(self["id"])
	scope := "member"
	switch self["role"] {
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
		"resolved_name":     "@" + agent + ":" + str(connection["server_name"]),
		"execution_binding": object{"key": bindingKey, "operator": binding["operator"], "host": binding["host"], "connection": binding["connection"]},
		"profile":           object{"requirement": object{"context": requirement["context"], "compatible": requirement["compatible"]}, "selected": object{"version": selected["version"], "source_rev": selected["source_rev"], "lock_sha256": selected["lock_sha256"]}},
		"project":           nil, "project_state": "undeclared"}
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
			if v4 {
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
	if v4 {
		obj(result["source"])["validation"] = "local declarations only; source attestations and star reservations unverified"
	}
	if view == "environment" {
		environment, err := declaredEnvironment(card, self, binding, connection, bindingKey, agent)
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
		result["query"] = object{"view": "environment"}
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
