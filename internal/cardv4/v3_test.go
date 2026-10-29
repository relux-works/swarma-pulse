package cardv4

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Each negative cites the written v3.1 rule, with a nearby accepted control.
// V4-specific constraints are not used as the legacy source authority.
func TestV3WrittenRules(t *testing.T) {
	for _, v := range listFromJSON(t, mustRead(t, "testdata/v3-rule-cases.json")) {
		c := obj(v)
		t.Run(str(c["id"]), func(t *testing.T) {
			if str(c["citation"]) == "" {
				t.Fatal("missing written rule")
			}
			card := fixture(t, "source-v3.json")
			patch(t, card, jsonPatches(t, c["positive_patches"]))
			if e := Validate(encoded(t, card), Options{Version: 3}); e != nil {
				t.Fatal("positive", e)
			}
			patch(t, card, jsonPatches(t, c["patches"]))
			assertOracle(t, Validate(encoded(t, card), Options{Version: 3}), caseOracle{ID: str(c["id"]), Rule: str(c["rule"]), Location: str(c["location"]), Exit: int(c["exit"].(float64)), Result: "refuse"})
		})
	}
}
func jsonPatches(t *testing.T, v any) []any {
	t.Helper()
	if v == nil {
		return nil
	}
	wrapper := Object{"patches": v}
	parsed, e := Parse(encoded(t, wrapper), 3)
	if e != nil {
		t.Fatal(e)
	}
	return list(parsed["patches"])
}

// R7's optional filesystem check detects a missing local task entry, while the
// complete isolated fixture is accepted. It does not require remote resources.
func TestV3R7Layout(t *testing.T) {
	c := fixture(t, "source-v3.json")
	dir := t.TempDir()
	for _, p := range []string{"legacy-card", "aurora-host", "beacon", "vault", "profiles/control"} {
		if e := os.MkdirAll(filepath.Join(dir, p), 0700); e != nil {
			t.Fatal(e)
		}
	}
	for _, p := range []string{"aurora-host", "beacon", "vault"} {
		if e := os.WriteFile(filepath.Join(dir, p, "tasks.md"), []byte("fixture\n"), 0600); e != nil {
			t.Fatal(e)
		}
	}
	r, e := os.OpenRoot(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	if issue := CheckLayout(c, "legacy-card", r); issue != nil {
		t.Fatal(issue)
	}
	if e := os.Remove(filepath.Join(dir, "beacon/tasks.md")); e != nil {
		t.Fatal(e)
	}
	issue := CheckLayout(c, "legacy-card", r)
	if issue == nil || legacyIssue(issue).Rule != "R7" || issue.Location != "$.projects[1].task_system.path" {
		t.Fatal("R7 missing-entry refusal", issue)
	}
}

// Legacy aliases expand safely and exactly, while v4 refuses their syntax.
func TestV3Aliases(t *testing.T) {
	raw := []byte("schema: product-card/v3\na: &shared {value: exact}\nb: *shared\n")
	c, e := Parse(raw, 3)
	if e != nil || obj(c["a"])["value"] != obj(c["b"])["value"] {
		t.Fatal("legacy alias expansion")
	}
	if _, e := Parse(raw, 4); e == nil || e.Rule != "yaml_alias" {
		t.Fatal("v4 alias accepted")
	}
	if _, e := Parse([]byte("a: &a [*a]\n"), 3); e == nil || e.Rule != "alias_cycle" {
		t.Fatal("cyclic legacy alias accepted")
	}
}

var _ = json.Valid
