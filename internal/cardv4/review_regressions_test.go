package cardv4

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// Key syntax obeys exactly the same v4 node restrictions as value syntax;
// ordinary keys remain valid, and v3's admitted anchors remain unaffected.
func TestKeyNodeRestrictions(t *testing.T) {
	for _, test := range []struct{ raw, rule string }{
		{`{&key "schema": "product-card/v4"}`, "yaml_alias"},
		{`{!!str "schema": "product-card/v4"}`, "explicit_tag"},
		{`{!fixture "schema": "product-card/v4"}`, "explicit_tag"},
		{`{a: &key schema, *key: value}`, "yaml_alias"},
		{`{<<: {}}`, "yaml_merge"},
	} {
		doc, e := Parse([]byte(test.raw), 4)
		if doc != nil || e == nil || e.Rule != test.rule {
			t.Fatalf("expected %s got %v", test.rule, e)
		}
	}
	if _, e := Parse([]byte(`{"schema":"product-card/v4"}`), 4); e != nil {
		t.Fatal(e)
	}
	if _, e := Parse([]byte(`{&key "schema":"product-card/v3"}`), 3); e != nil {
		t.Fatal("legacy key anchor", e)
	}
}

// Whitespace/control characters cannot hide in a repository component. The
// valid neighbor differs only by the forbidden character's absence.
func TestRepositoryWhitespace(t *testing.T) {
	if !repo("git.example/owner/name") {
		t.Fatal("positive repository")
	}
	for _, c := range []rune{' ', '\t', '\n', '\r', '\v', '\f', '\u0085', '\u00a0', '\u2003', '\u2028', 0, '\u007f'} {
		card := fixture(t, "converted-v4.json")
		obj(list(card["projects"])[0])["repo"] = "git.example/owner/na" + string(c) + "me"
		if repo(obj(list(card["projects"])[0])["repo"]) {
			t.Fatal("repository grammar accepted forbidden code point")
		}
		raw := encoded(t, card)
		raw = bytes.ReplaceAll(raw, []byte(string(c)), []byte(fmt.Sprintf("\\u%04x", c)))
		e := Validate(raw, fixtureOptions(t))
		if e == nil || e.Rule != "repository_format" || e.Location != "$.projects[0].repo" {
			t.Fatal("repository whitespace/control refusal", e)
		}
	}
}

// The original delimiter matters even when URL parsing yields an empty fragment.
func TestOriginFragmentDelimiter(t *testing.T) {
	for _, test := range []struct {
		url   string
		valid bool
	}{{"https://gr.relux.works/", true}, {"https://gr.relux.works/#", false}, {"https://gr.relux.works/#fragment", false}, {"https://gr.relux.works#", false}, {"https://gr.relux.works?", false}} {
		card := fixture(t, "converted-v4.json")
		obj(obj(card["connections"])["target-gr"])["homeserver"] = test.url
		e := Validate(encoded(t, card), fixtureOptions(t))
		if test.valid {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.Rule != "https_origin" || e.Location != "$.connections[1].homeserver" {
			t.Fatal("origin delimiter refusal", e)
		}
	}
}

// Competing failures pin the actual documented order: v4 mapping key/value pairs
// in input order, then service branch exclusions before generic object fields.
func TestCompetingDiagnosticOrder(t *testing.T) {
	for _, test := range []struct{ raw, rule, location string }{
		{`{a: !!str value, <<: {}}`, "explicit_tag", "$[0]"},
		{`{<<: {}, a: !!str value}`, "yaml_merge", "$[0]"},
		{`{a: !!str value, &key b: value}`, "explicit_tag", "$[0]"},
		{`{&key a: value, b: !!str value}`, "yaml_alias", "$[0]"},
	} {
		_, e := Parse([]byte(test.raw), 4)
		if e == nil || e.Rule != test.rule || e.Location != test.location {
			t.Fatalf("expected %s %s got %v", test.rule, test.location, e)
		}
	}
	for _, test := range []struct {
		name, rule, location string
		mutate               func(Object)
	}{
		{"service exclusions", "service_profile_forbidden", "$.members[4].profile", func(m Object) { m["profile"] = "control"; m["unknown"] = true; delete(m, "id") }},
		{"service exclusion priority", "service_staffing_forbidden", "$.members[4].staffing", func(m Object) {
			m["staffing"] = []any{}
			m["independent_of_producer"] = true
			m["chat_admission"] = "synthetic"
		}},
		{"runtime precheck", "service_runtime_forbidden", "$.members[4].runtime", func(m Object) {
			r := obj(m["runtime"])
			r["model"] = "synthetic"
			r["unknown"] = true
			delete(r, "kind")
		}},
		{"unknown before required", "unknown_field", "$.members[4][13]", func(m Object) { m["unknown"] = true; delete(m, "id") }},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := fixture(t, "services-v4.json")
			test.mutate(obj(list(c["members"])[4]))
			e := Validate(encoded(t, c), fixtureOptions(t))
			if e == nil || e.Rule != test.rule || e.Location != test.location {
				t.Fatalf("expected %s %s got %v", test.rule, test.location, e)
			}
		})
	}
}

type destinationAudit struct {
	t                         *testing.T
	path                      string
	inside, outside           fs.FileInfo
	insideReads, outsideReads int
	swapped                   bool
	swapOnOpen                bool
	nested                    bool
	armed                     bool
}

func (a *destinationAudit) swap() {
	a.t.Helper()
	if a.swapped || !a.armed {
		return
	}
	component, held, target := "profiles", "profiles-held", "elsewhere"
	if a.nested {
		component, held, target = "profiles/group", "profiles/group-held", "../elsewhere"
	}
	if e := os.Rename(filepath.Join(a.path, component), filepath.Join(a.path, held)); e != nil {
		a.t.Fatal(e)
	}
	if e := os.Symlink(target, filepath.Join(a.path, component)); e != nil {
		a.t.Fatal(e)
	}
	a.swapped = true
}
func (a *destinationAudit) record(info fs.FileInfo) {
	if info == nil {
		return
	}
	if os.SameFile(info, a.inside) {
		a.insideReads++
	}
	if os.SameFile(info, a.outside) {
		a.outsideReads++
	}
}

// This adapter records actual returned inode identity, not requested spelling.
// Its swap occurs only in a fresh isolated tree after observation/FD acquisition.
type destinationRoot struct {
	root        *os.Root
	audit       *destinationAudit
	productRoot bool
}

func (r *destinationRoot) Lstat(p string) (fs.FileInfo, error) {
	info, e := r.root.Lstat(p)
	r.audit.record(info)
	if !r.productRoot && p == "group" && r.audit.nested {
		r.audit.swap()
	}
	if r.productRoot && p == "profiles" && !r.audit.swapOnOpen {
		r.audit.swap()
	}
	return info, e
}
func (r *destinationRoot) Readlink(p string) (string, error) { return r.root.Readlink(p) }
func (r *destinationRoot) OpenRoot(p string) (LayoutRoot, error) {
	child, e := r.root.OpenRoot(p)
	if e != nil {
		return nil, e
	}
	if r.productRoot && p == "profiles" && r.audit.swapOnOpen && !r.audit.nested {
		r.audit.swap()
	}
	return &destinationRoot{child, r.audit, false}, nil
}
func (r *destinationRoot) Close() error { return r.root.Close() }
func swapTree(t *testing.T, swapOnOpen, nested bool) (*destinationRoot, *destinationAudit) {
	t.Helper()
	dir := t.TempDir()
	insidePath := "profiles/control"
	if nested {
		insidePath = "profiles/group/control"
	}
	for _, p := range []string{insidePath, "elsewhere/control", "aurora-host", "beacon", "vault"} {
		if e := os.MkdirAll(filepath.Join(dir, p), 0700); e != nil {
			t.Fatal(e)
		}
	}
	for _, p := range []string{"aurora-host", "beacon", "vault"} {
		if e := os.WriteFile(filepath.Join(dir, p, "tasks.md"), []byte("fixture\n"), 0600); e != nil {
			t.Fatal(e)
		}
	}
	inside, e := os.Stat(filepath.Join(dir, insidePath))
	if e != nil {
		t.Fatal(e)
	}
	outside, e := os.Stat(filepath.Join(dir, "elsewhere/control"))
	if e != nil {
		t.Fatal(e)
	}
	if os.SameFile(inside, outside) {
		t.Fatal("race control targets must be distinct")
	}
	root, e := os.OpenRoot(dir)
	if e != nil {
		t.Fatal(e)
	}
	a := &destinationAudit{t: t, path: dir, inside: inside, outside: outside, swapOnOpen: swapOnOpen, nested: nested, armed: true}
	return &destinationRoot{root, a, true}, a
}

// The known-violation control reproduces the old broad-root walk and MUST record
// an actual outside destination. The fixed full operation retains the original
// profile FD across the same component replacement and inspects only inside.
func TestProfileComponentSwap(t *testing.T) {
	t.Run("known violation recorder", func(t *testing.T) {
		r, a := swapTree(t, false, false)
		defer r.Close()
		if _, e := r.Lstat("profiles"); e != nil {
			t.Fatal(e)
		}
		if _, e := r.Lstat("profiles/control"); e != nil {
			t.Fatal(e)
		}
		if !a.swapped || a.outsideReads != 1 || a.insideReads != 0 {
			t.Fatal("actual redirected-destination violation not detected")
		}
	})
	t.Run("pinned production boundary", func(t *testing.T) {
		r, a := swapTree(t, true, false)
		defer r.Close()
		if e := CheckLayout(fixture(t, "converted-v4.json"), "aurora-host", r); e != nil {
			t.Fatal(e)
		}
		if !a.swapped || a.insideReads != 1 || a.outsideReads != 0 {
			t.Fatal("profile handle failed to retain actual destination")
		}
	})
	t.Run("nested component cannot escape pinned root", func(t *testing.T) {
		r, a := swapTree(t, false, true)
		defer r.Close()
		c := fixture(t, "converted-v4.json")
		obj(obj(c["profiles"])["control"])["path"] = "group/control"
		a.armed = false
		if e := CheckLayout(c, "aurora-host", r); e != nil || a.insideReads != 1 || a.outsideReads != 0 || a.swapped {
			t.Fatal("unmodified nested profile control", e)
		}
		a.armed = true
		a.insideReads = 0
		e := CheckLayout(c, "aurora-host", r)
		if !a.swapped || e == nil || e.Rule != "layout_unreadable" || e.Location != "$.profiles[0].path" || a.outsideReads != 0 {
			t.Fatal("raced nested link was not refused by narrow native root", e)
		}
	})
}

// Duplicate-key checks precede that entry's invalid value; v3's explicitly
// documented merge prepass precedes an earlier unsupported scalar tag.
func TestAdditionalDiagnosticPrecedence(t *testing.T) {
	_, e := Parse([]byte(`{a: good, a: !!str bad}`), 4)
	if e == nil || e.Rule != "duplicate_key" || e.Location != "$[1]" {
		t.Fatal("duplicate precedence", e)
	}
	_, e = Parse([]byte(`{a: !unsupported value, <<: false}`), 3)
	if e == nil || e.Rule != "parse_document" || e.Location != "$" {
		t.Fatal("v3 merge prepass order", e)
	}
	c := fixture(t, "services-v4.json")
	r := obj(obj(list(c["members"])[4])["runtime"])
	r["effort"] = "high"
	r["unknown"] = true
	delete(r, "kind")
	e = Validate(encoded(t, c), fixtureOptions(t))
	if e == nil || e.Rule != "service_runtime_forbidden" || e.Location != "$.members[4].runtime" {
		t.Fatal("runtime effort precedence", e)
	}
}
