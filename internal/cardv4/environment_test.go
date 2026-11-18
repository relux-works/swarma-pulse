package cardv4

import (
	"strings"
	"testing"
)

func environmentDeclaration() Object {
	return Object{"schema": "pulse-member-environment/v1", "a2a_config_profile": "Demo-config_1", "working_root": "../unopened", "network": Object{"mode": "proxy", "ref": "Proxy-main_1"}}
}

// Both chat and non-chat conversational agents accept the closed declaration;
// selectors retain case and boundary length, and paths remain inert coordinates.
func TestV5EnvironmentDeclarations(t *testing.T) {
	for _, member := range []int{0, 1, 3} {
		for _, edit := range []func(Object){
			func(Object) {},
			func(p Object) { p["network"] = Object{"mode": "direct"} },
			func(p Object) { p["a2a_config_profile"] = "A" + strings.Repeat("x", 127); p["working_root"] = "." },
			func(p Object) {
				for k := range p {
					if k != "schema" {
						delete(p, k)
					}
				}
			},
		} {
			c := governanceFixture(t)
			p := environmentDeclaration()
			edit(p)
			obj(list(c["members"])[member])["pulse"] = p
			if _, issue := ParseGovernanceSnapshot(encoded(t, c), governanceNow); issue != nil {
				t.Fatal(member, p, issue)
			}
		}
	}
}

// Every malformed supplied extension is rejected, including unselected members.
// The existing parser/privacy phases own nulls and credentials before grammar.
func TestV5EnvironmentDeclarationRefusals(t *testing.T) {
	cases := []struct {
		name, rule string
		edit       func(Object)
	}{
		{"scalar", "object_shape", func(m Object) { m["pulse"] = "bad" }},
		{"null", "null_value", func(m Object) { m["pulse"] = nil }},
		{"unknown", "unknown_field", func(m Object) { obj(m["pulse"])["extra"] = true }},
		{"schema-missing", "missing_field", func(m Object) { delete(obj(m["pulse"]), "schema") }},
		{"schema-unsupported", "enum_value", func(m Object) { obj(m["pulse"])["schema"] = "pulse-member-environment/v2" }},
		{"network-shape", "object_shape", func(m Object) { obj(m["pulse"])["network"] = false }},
		{"network-null", "null_value", func(m Object) { obj(m["pulse"])["network"] = nil }},
		{"mode-missing", "missing_field", func(m Object) { obj(m["pulse"])["network"] = Object{} }},
		{"mode-unsupported", "enum_value", func(m Object) { obj(m["pulse"])["network"] = Object{"mode": "automatic"} }},
		{"proxy-ref-missing", "missing_field", func(m Object) { obj(m["pulse"])["network"] = Object{"mode": "proxy"} }},
		{"direct-ref", "unknown_field", func(m Object) { obj(m["pulse"])["network"] = Object{"mode": "direct", "ref": "extra"} }},
		{"privacy-before-grammar", "credential_field_forbidden", func(m Object) { obj(m["pulse"])["password"] = "placeholder"; delete(obj(m["pulse"]), "schema") }},
	}
	for _, field := range []string{"a2a_config_profile", "proxy.ref"} {
		for _, value := range []any{false, "", "_leading", "../outside", "x;y", strings.Repeat("a", 129)} {
			field, value := field, value
			cases = append(cases, struct {
				name, rule string
				edit       func(Object)
			}{field, "environment_selector_format", func(m Object) {
				p := obj(m["pulse"])
				if field == "proxy.ref" {
					obj(p["network"])["ref"] = value
				} else {
					p[field] = value
				}
			}})
		}
	}
	for _, value := range []any{false, "", "/absolute", "~home", "C:drive", "https:remote", `nested\path`, "path\n"} {
		value := value
		cases = append(cases, struct {
			name, rule string
			edit       func(Object)
		}{"path", "environment_path_format", func(m Object) { obj(m["pulse"])["working_root"] = value }})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := governanceFixture(t)
			m := obj(list(c["members"])[3])
			m["pulse"] = environmentDeclaration()
			tc.edit(m)
			_, issue := ParseGovernanceSnapshot(encoded(t, c), governanceNow)
			if issue == nil || issue.Rule != tc.rule {
				t.Fatalf("want %s, got %v", tc.rule, issue)
			}
		})
	}
}

// V5 human and service records never acquire launch declarations, and v4's
// closed grammar is not widened by the v5 extension.
func TestEnvironmentVersionAndMemberBranches(t *testing.T) {
	c := governanceFixture(t)
	obj(list(c["members"])[4])["pulse"] = environmentDeclaration()
	if _, issue := ParseGovernanceSnapshot(encoded(t, c), governanceNow); issue == nil || issue.Rule != "unknown_field" {
		t.Fatal(issue)
	}
	c = governanceFixture(t)
	m := obj(list(c["members"])[3])
	m["responsibility"], m["context_mode"], m["runtime"] = "service", "none", Object{"kind": "none"}
	delete(m, "profile")
	m["service_contract_ref"], m["native_result_ref"], m["pulse"] = "fixture-contract", "fixture-result", environmentDeclaration()
	if _, issue := ParseGovernanceSnapshot(encoded(t, c), governanceNow); issue == nil || issue.Rule != "unknown_field" {
		t.Fatal(issue)
	}
	delete(m, "pulse")
	if _, issue := ParseGovernanceSnapshot(encoded(t, c), governanceNow); issue != nil {
		t.Fatal("service positive control", issue)
	}
	c = fixture(t, "converted-v4.json")
	obj(list(c["members"])[1])["pulse"] = environmentDeclaration()
	if _, issue := ParseLocalSnapshot(encoded(t, c)); issue == nil || issue.Rule != "unknown_field" {
		t.Fatal("v4 widened", issue)
	}
}

// This table fixes the full recorded relative-coordinate forms independently
// of a second parser; no lexical acceptance implies permission to open a path.
func TestEnvironmentRelativeCompatibility(t *testing.T) {
	for _, tc := range []struct {
		value any
		valid bool
	}{
		{".", true}, {"..", true}, {"../sibling/path", true}, {"roles/control", true}, {"mixed Case/name", true}, {"a:b", false}, {"C:drive", false}, {"https:remote", false}, {"/absolute", false}, {"~home", false}, {`nested\path`, false}, {"path\n", false}, {"path\x00", false}, {string([]byte{0xff}), false}, {"", false}, {false, false}, {nil, false},
	} {
		if got := EnvironmentRelative(tc.value); got != tc.valid {
			t.Fatalf("coordinate %q: want %t got %t", tc.value, tc.valid, got)
		}
	}
}
