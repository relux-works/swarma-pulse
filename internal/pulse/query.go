// Package pulse provides an explicit local, unsigned, self-only product-card view.
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
	if revision != "sha256:"+digest(raw) {
		return nil, refusal("source_digest")
	}
	if agent == "" || len(agent) > 512 || !utf8.ValidString(agent) || strings.ContainsFunc(agent, unicode.IsControl) {
		return nil, refusal("caller_context")
	}
	card, issue := cardv4.Parse(raw, 3)
	if issue != nil {
		return nil, refusal("card_invalid")
	}
	if card["schema"] != "product-card/v3" {
		return nil, refusal("unsupported_schema")
	}
	if issue := cardv4.Validate(raw, cardv4.Options{Version: 3}); issue != nil {
		return nil, refusal("card_invalid")
	}
	product := obj(card["product"])
	bindings := obj(card["bindings"])
	connections := obj(card["connections"])
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
		if str(b["identity_prefix"])+str(member["id"]) != agent {
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
			projects = append(projects, object{"key": pkey, "repo": p["repo"], "desired_state": p["desired_state"], "location": object{"kind": "local", "path": p["path"]}})
		}
		if len(projects) != 1 {
			return nil, refusal("self_project_missing")
		}
	}
	result := object{"schema": "pulse-view/v1", "mode": "local-unsigned", "product": product["id"], "member": key,
		"caller": object{"agent": agent, "cryptographically_verified": false, "trust": "single-user local trust"}, "known_limit": knownLimit,
		"source":            object{"schema": "product-card/v3", "revision": revision, "sha256": digest(raw), "card_root": filepath.Clean(cardRoot), "freshness": "unknown", "observed_at": nil},
		"query_observed_at": now.UTC().Format(time.RFC3339Nano), "self": viewSelf, "peers": []any{}, "direct_reports": []any{}, "descendants": []any{}, "superiors": []any{}, "projects": projects,
		// This constant does not reveal hidden node counts or edge existence.
		"hierarchy_status": "self-only; hierarchy not disclosed",
		"observations":     object{"runtime": object{"state": "unknown"}, "registration": object{"state": "unknown"}, "installed_profile": object{"state": "unknown"}, "assignment": object{"state": "unknown"}}}
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, refusal("output_encoding")
	}
	return append(encoded, '\n'), nil
}
