// Package cardv4 validates desired product-card facts without performing effects.
// Callers supply immutable source bytes, reservations and optional read-only layout access.
package cardv4

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

const MaxBytes = 1 << 20

type Object = map[string]any

// Issue contains only implementation-owned rule names and indexed locations.
type Issue struct {
	Rule     string `json:"rule"`
	Location string `json:"location"`
	Exit     int    `json:"exit"`
}

func (e *Issue) Error() string      { return "RULE " + e.Rule + ": " + e.Location }
func fail(rule, at string) *Issue   { return &Issue{rule, at, 1} }
func shape(rule, at string) *Issue  { return &Issue{rule, at, 2} }
func index(at string, n int) string { return fmt.Sprintf("%s[%d]", at, n) }
func field(at, k string) string     { return at + "." + k }
func obj(x any) Object              { v, _ := x.(map[string]any); return v }
func list(x any) []any              { v, _ := x.([]any); return v }
func str(x any) string              { v, _ := x.(string); return v }
func sorted(m Object) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Parse checks document encoding and budgets. V3 permits anchors/aliases needed by
// existing sources; expansion remains bounded and cyclic aliases refuse.
func Parse(raw []byte, version int) (Object, *Issue) {
	if len(raw) > MaxBytes {
		return nil, shape("raw_byte_limit", "$")
	}
	if !utf8.Valid(raw) {
		return nil, shape("invalid_utf8", "$")
	}
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	var n yaml.Node
	if dec.Decode(&n) != nil || len(n.Content) != 1 || n.Content[0].Kind != yaml.MappingNode {
		return nil, shape("parse_document", "$")
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, shape("multiple_documents", "$")
	}
	budget := 0
	value, issue := nodeValue(n.Content[0], "$", 0, version, map[*yaml.Node]bool{}, &budget)
	if issue != nil {
		return nil, issue
	}
	return obj(value), nil
}
func nodeValue(n *yaml.Node, at string, depth, version int, active map[*yaml.Node]bool, budget *int) (any, *Issue) {
	*budget++
	if *budget > 1000000 {
		return nil, shape("expansion_limit", at)
	}
	if active[n] {
		return nil, shape("alias_cycle", at)
	}
	active[n] = true
	defer delete(active, n)
	if version == 4 {
		if e := v4NodeRestriction(n, at); e != nil {
			return nil, e
		}
	}
	if n.Kind == yaml.AliasNode {
		return nodeValue(n.Alias, at, depth, version, active, budget)
	}
	if n.Kind == yaml.MappingNode || n.Kind == yaml.SequenceNode {
		depth++
		if depth > 32 {
			return nil, shape("nesting_limit", at)
		}
	}
	switch n.Kind {
	case yaml.MappingNode:
		out := Object{}
		seen := map[string]bool{}
		// Legacy YAML merges are expanded before explicit entries; no v4 merge survives.
		for i := 0; version == 3 && i < len(n.Content); i += 2 {
			k := n.Content[i]
			if k.Tag == "!!merge" {
				v, e := nodeValue(n.Content[i+1], index(at, i/2), depth, version, active, budget)
				if e != nil {
					return nil, e
				}
				ms := []any{v}
				if a, ok := v.([]any); ok {
					ms = a
				}
				for _, m := range ms {
					if obj(m) == nil {
						return nil, shape("parse_document", at)
					}
					for key, val := range obj(m) {
						if _, ok := out[key]; !ok {
							out[key] = val
						}
					}
				}
			}
		}
		for i := 0; i < len(n.Content); i += 2 {
			k := n.Content[i]
			loc := index(at, i/2)
			if version == 4 {
				if e := v4NodeRestriction(k, loc); e != nil {
					return nil, e
				}
			}
			if k.Tag == "!!merge" {
				continue
			}
			if k.Kind != yaml.ScalarNode || k.Tag != "!!str" {
				return nil, shape("non_string_key", loc)
			}
			if seen[k.Value] {
				return nil, shape("duplicate_key", loc)
			}
			seen[k.Value] = true
			v, e := nodeValue(n.Content[i+1], loc, depth, version, active, budget)
			if e != nil {
				return nil, e
			}
			out[k.Value] = v
		}
		return out, nil
	case yaml.SequenceNode:
		out := make([]any, 0, len(n.Content))
		for i, c := range n.Content {
			v, e := nodeValue(c, index(at, i), depth, version, active, budget)
			if e != nil {
				return nil, e
			}
			out = append(out, v)
		}
		return out, nil
	case yaml.ScalarNode:
		switch n.Tag {
		case "!!str":
			return n.Value, nil
		case "!!bool":
			v, e := strconv.ParseBool(n.Value)
			if e != nil {
				return nil, shape("scalar_type", at)
			}
			return v, nil
		case "!!int":
			var v int64
			if n.Decode(&v) != nil {
				return nil, shape("scalar_type", at)
			}
			return v, nil
		case "!!null":
			if version == 4 {
				return nil, shape("null_value", at)
			}
			return nil, nil
		default:
			return nil, shape("scalar_type", at)
		}
	}
	return nil, shape("parse_document", at)
}
func v4NodeRestriction(n *yaml.Node, at string) *Issue {
	if n.Style&yaml.TaggedStyle != 0 {
		return shape("explicit_tag", at)
	}
	if n.Anchor != "" || n.Kind == yaml.AliasNode {
		return shape("yaml_alias", at)
	}
	if n.Tag == "!!merge" {
		return shape("yaml_merge", at)
	}
	return nil
}

// CheckCollectionBounds is separate so count guards can be exercised without
// byte-budget or duplicate-id failures obscuring their result.
func CheckCollectionBounds(card Object) *Issue {
	for _, c := range []struct {
		k string
		n int
	}{{"projects", 1000}, {"members", 10000}} {
		a, ok := card[c.k].([]any)
		if !ok {
			return shape("collection_shape", field("$", c.k))
		}
		if len(a) > c.n {
			return fail(c.k+"_limit", field("$", c.k))
		}
	}
	return nil
}

var liveKeys = words("current_task liveness live status session session_id pid running last_seen last_post bound actual_state assignment assignments assignment_status continuation resume_id")

// CheckForbiddenFacts precedes schema validation and never returns input keys.
func CheckForbiddenFacts(value any) *Issue {
	var walk func(any, string) *Issue
	walk = func(v any, at string) *Issue {
		switch x := v.(type) {
		case map[string]any:
			for i, k := range sorted(x) {
				loc := index(at, i)
				if credentialKey.MatchString(k) {
					return fail("credential_field_forbidden", loc)
				}
				if liveKeys[k] {
					return fail("live_state_forbidden", loc)
				}
				if secretPattern.MatchString(k) {
					return fail("secret_value_forbidden", loc)
				}
				if e := walk(x[k], loc); e != nil {
					return e
				}
			}
		case []any:
			for i, c := range x {
				if e := walk(c, index(at, i)); e != nil {
					return e
				}
			}
		case string:
			if secretPattern.MatchString(x) {
				return fail("secret_value_forbidden", at)
			}
		}
		return nil
	}
	return walk(value, "$")
}

// CheckSuperiorHopBudget follows supplied, already authorized graph edges only.
// The missing final edge is a partial boundary, not evidence of an absent person.
func CheckSuperiorHopBudget(start string, edges map[string]string) *Issue {
	seen := map[string]bool{}
	for hops := 0; ; hops++ {
		if seen[start] {
			return fail("reporting_cycle", "$.parent")
		}
		seen[start] = true
		next, ok := edges[start]
		if !ok {
			return nil
		}
		if hops == 64 {
			return fail("superior_hop_limit", "$.parent")
		}
		start = next
	}
}
func words(s string) map[string]bool {
	m := map[string]bool{}
	for _, v := range strings.Fields(s) {
		m[v] = true
	}
	return m
}
