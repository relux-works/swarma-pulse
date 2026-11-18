package cardv4

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var environmentSelector = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)
var environmentScheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

// EnvironmentRelative checks a declared coordinate, not an accessible path.
// Dot and sibling components remain legal; no filesystem operation is implied.
func EnvironmentRelative(v any) bool {
	s, ok := v.(string)
	return ok && s != "" && utf8.ValidString(s) && !strings.ContainsFunc(s, unicode.IsControl) &&
		!strings.HasPrefix(s, "/") && !strings.HasPrefix(s, "~") && !strings.ContainsRune(s, '\\') && !environmentScheme.MatchString(s)
}

// CheckEnvironmentExtension is the shared closed declaration grammar. Callers
// retain their own raw-document/privacy phases and version-specific error map.
func CheckEnvironmentExtension(v any, at string) *Issue {
	selector := scalar(func(v any) bool { return environmentSelector.MatchString(str(v)) }, "environment_selector_format")
	network := func(v any, at string) *Issue {
		m := obj(v)
		if m == nil {
			return shape("object_shape", at)
		}
		if _, present := m["mode"]; !present {
			return fail("missing_field", field(at, "mode"))
		}
		required := map[string]validator{"mode": enumeration("direct proxy")}
		if m["mode"] == "proxy" {
			required["ref"] = selector
		}
		return object(required, nil, true)(v, at)
	}
	return object(map[string]validator{"schema": enumeration("pulse-member-environment/v1")}, map[string]validator{
		"a2a_config_profile": selector,
		"working_root":       scalar(EnvironmentRelative, "environment_path_format"),
		"network":            network,
	}, true)(v, at)
}
