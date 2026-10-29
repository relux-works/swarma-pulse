package cardv4

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

type caseOracle struct {
	ID       string `json:"id"`
	Rule     string `json:"rule"`
	Location string `json:"location"`
	Exit     int    `json:"exit"`
	Result   string `json:"result"`
}

func number(v any) int { n, _ := v.(int64); return int(n) }
func lookup(root any, path []any) any {
	for _, key := range path {
		if s, ok := key.(string); ok {
			root = obj(root)[s]
		} else {
			root = list(root)[number(key)]
		}
	}
	return root
}
func patch(t *testing.T, root Object, patches []any) {
	t.Helper()
	for _, v := range patches {
		p := obj(v)
		path := list(p["path"])
		if len(path) == 0 {
			t.Fatal("fixture setup: empty patch")
		}
		parent := lookup(root, path[:len(path)-1])
		key := path[len(path)-1]
		op := str(p["op"])
		if k, ok := key.(string); ok {
			m := obj(parent)
			_, exists := m[k]
			if m == nil || (op == "add" && exists) || (op != "add" && !exists) {
				t.Fatal("fixture setup: map patch")
			}
			switch op {
			case "replace", "add":
				m[k] = p["value"]
			case "remove":
				delete(m, k)
			default:
				t.Fatal("fixture setup: patch op")
			}
		} else {
			a := list(parent)
			i := number(key)
			if i < 0 || i >= len(a) {
				t.Fatal("fixture setup: index patch")
			}
			switch op {
			case "replace":
				a[i] = p["value"]
			case "remove":
				a = append(a[:i], a[i+1:]...)
				container := lookup(root, path[:len(path)-2])
				pk := path[len(path)-2]
				if s, ok := pk.(string); ok {
					obj(container)[s] = a
				} else {
					list(container)[number(pk)] = a
				}
			default:
				t.Fatal("fixture setup: array op")
			}
		}
	}
}
func overlay(a, b Object) Object {
	out := copyObject(a)
	for k, v := range b {
		out[k] = v
	}
	return out
}

// These cases execute all 37 named stages and paired neighbors. Actual agent
// session launch remains a separate external qualification.
func TestAcceptedVectors(t *testing.T) {
	var oracles []caseOracle
	if e := json.Unmarshal(mustRead(t, "testdata/vector-oracles.json"), &oracles); e != nil {
		t.Fatal(e)
	}
	expected := map[string]caseOracle{}
	for _, o := range oracles {
		if _, ok := expected[o.ID]; ok {
			t.Fatal("duplicate oracle")
		}
		expected[o.ID] = o
	}
	seen := map[string]bool{}
	for _, family := range []string{"legacy", "mixed", "service", "locations", "carry-over", "parser", "bounds", "compatibility"} {
		for _, v := range list(fixture(t, "vectors-"+family+".json")["cases"]) {
			c := obj(v)
			id := str(c["id"])
			seen[id] = true
			t.Run(id, func(t *testing.T) {
				oracle, ok := expected[id]
				if !ok {
					t.Fatal("missing independently recorded oracle")
				}
				positive := obj(c["valid_neighbor"])
				if e := executeVector(t, positive, nil); e != nil {
					t.Fatalf("positive control: %v", e)
				}
				counter := obj(c["counterexample"])
				spec := overlay(positive, counter)
				if counter == nil {
					spec = obj(c["operation"])
				}
				issue := executeVector(t, spec, counter)
				assertOracle(t, issue, oracle)
			})
		}
	}
	if len(seen) != 37 || len(expected) != 37 {
		t.Fatalf("selected vector count %d, oracles %d", len(seen), len(expected))
	}
	for id := range expected {
		if !seen[id] {
			t.Fatal("orphan oracle")
		}
	}
}
func assertOracle(t *testing.T, e *Issue, o caseOracle) {
	t.Helper()
	if o.Result == "accept" {
		if e != nil {
			t.Fatal(e)
		}
		return
	}
	if e == nil {
		t.Fatal("required refusal absent")
	}
	if e.Rule != o.Rule || e.Location != o.Location || e.Exit != o.Exit {
		t.Fatalf("expected %s %s exit %d; got %v exit %d", o.Rule, o.Location, o.Exit, e, e.Exit)
	}
}
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func executeVector(t *testing.T, spec, counter Object) *Issue {
	t.Helper()
	if spec["operation"] == "convert-atomic" || spec["operation"] == "registration-tripwire-self-check" {
		return executeConversionVector(t, spec)
	}
	o := fixtureOptions(t)
	if premise := str(spec["reservation_premises_source"]); premise != "" {
		file, key, _ := strings.Cut(premise, "#/")
		decodeReservations(t, fixture(t, file)[key], &o)
	}
	if additions := list(spec["reviewed_reservation_premises"]); additions != nil {
		var r []Reservation
		if e := json.Unmarshal(encoded(t, additions), &r); e != nil {
			t.Fatal(e)
		}
		o.Reservations = append(o.Reservations, r...)
	}
	if selection := list(spec["source_selection"]); selection != nil {
		o.Sources = nil
		for _, p := range selection {
			o.Sources = append(o.Sources, sourceFromPin(t, obj(p)))
		}
	}
	if replacement := obj(spec["replace_supplied_v3_source"]); replacement != nil {
		for i := range o.Sources {
			if o.Sources[i].Revision == "fixture-v3-001" {
				o.Sources[i] = sourceFromPin(t, replacement)
			}
		}
	}
	for _, v := range list(spec["remove_supplied_sources"]) {
		file := str(v)
		for i := len(o.Sources) - 1; i >= 0; i-- {
			if bytes.Equal(o.Sources[i].Bytes, fixtureBytes(t, file)) {
				o.Sources = append(o.Sources[:i], o.Sources[i+1:]...)
			}
		}
	}
	var card Object
	var raw []byte
	if file := str(spec["card"]); file != "" {
		raw = fixtureBytes(t, file)
		card = fixture(t, file)
		patch(t, card, list(spec["target_patches"]))
		raw = encoded(t, card)
	}
	construction := obj(spec["construction"])
	switch construction["kind"] {
	case "nested-mapping-depth":
		var v any = construction["leaf_utf8"]
		for i := 0; i < number(construction["container_count_including_root"]); i++ {
			v = Object{str(construction["wrapper_key"]): v}
		}
		raw = encoded(t, v)
	case "repeat-array-last-entry":
		path := list(construction["path"])
		a := list(lookup(card, path))
		if len(path) != 1 || len(a) == 0 {
			t.Fatal("fixture setup: count construction")
		}
		for len(a) < number(construction["target_count"]) {
			a = append(a, a[len(a)-1])
		}
		card[str(path[0])] = a
	case "acyclic-authorized-external-chain":
		n := number(construction["edge_count"])
		start := str(construction["start"])
		next := str(construction["first_external"])
		edges := map[string]string{start: next}
		for i := 1; i < n; i++ {
			node := fmt.Sprintf("realm-%d/keeper", i+1)
			edges[next] = node
			next = node
		}
		return CheckSuperiorHopBudget(start, edges)
	}
	if transform := obj(spec["raw_transform"]); transform != nil {
		raw = rawConstruction(t, str(spec["card"]), transform)
	}
	switch spec["operation"] {
	case "existing-v3-reader":
		return Validate(raw, Options{Version: 3})
	case "parse-document":
		_, e := Parse(raw, 4)
		return e
	case "check-collection-bounds":
		return CheckCollectionBounds(card)
	case "check-forbidden-desired-facts":
		return CheckForbiddenFacts(card)
	case "validate-card-and-sources":
		return Validate(raw, o)
	case "validate-layout":
		return executeLayout(t, card, obj(spec["layout_fixture"]), obj(spec["replace_symlink"]))
	case "verify-pilot-preservation":
		if e := Validate(raw, o); e != nil {
			return e
		}
		return VerifyPilotPreservation(fixture(t, str(spec["prior"])), card, str(spec["pilot"]))
	case "verify-conversion-preservation":
		if e := Validate(raw, o); e != nil {
			return e
		}
		var in PreservationInputs
		if e := json.Unmarshal(fixtureBytes(t, str(spec["conversion_inputs"])), &in); e != nil {
			t.Fatal(e)
		}
		return VerifyConversionPreservation(fixture(t, str(spec["source"])), card, in)
	default:
		t.Fatal("fixture setup: unsupported operation")
		return nil
	}
}
func sourceFromPin(t *testing.T, p Object) Source {
	t.Helper()
	return Source{"v3-card", str(p["revision"]), str(p["sha256"]), fixtureBytes(t, str(p["file"]))}
}
func rawConstruction(t *testing.T, file string, transform Object) []byte {
	t.Helper()
	switch transform["kind"] {
	case "append-ascii-spaces-to-byte-count":
		b := fixtureBytes(t, str(transform["base"]))
		n := number(transform["target_bytes"])
		if n < len(b) {
			t.Fatal("fixture setup: padding")
		}
		return append(b, bytes.Repeat([]byte(" "), n-len(b))...)
	case "concatenate-yaml-documents":
		var parts [][]byte
		for _, v := range list(transform["documents"]) {
			parts = append(parts, compactFixture(t, str(v)))
		}
		return bytes.Join(parts, []byte(str(transform["separator_utf8"])))
	}
	var doc yaml.Node
	if yaml.Unmarshal(fixtureBytes(t, file), &doc) != nil {
		t.Fatal("fixture setup: raw node")
	}
	return append(renderRawJSON(t, doc.Content[0], nil, transform), '\n')
}
func compactFixture(t *testing.T, file string) []byte { return rawConstruction(t, file, Object{}) }
func renderRawJSON(t *testing.T, n *yaml.Node, path []any, tr Object) []byte {
	t.Helper()
	kind := str(tr["kind"])
	if kind == "yaml-anchor-alias-json-flow" && reflect.DeepEqual(path, list(tr["alias_path"])) {
		return []byte("*fixture_copy")
	}
	var out []byte
	switch n.Kind {
	case yaml.MappingNode:
		parts := [][]byte{}
		if kind == "yaml-merge-json-flow-object" && reflect.DeepEqual(nonNilPath(path), nonNilPath(list(tr["object_path"]))) {
			parts = append(parts, append([]byte("<<: "), encoded(t, tr["merge_mapping"])...))
		}
		for i := 0; i < len(n.Content); i += 2 {
			k := n.Content[i].Value
			key := encoded(t, k)
			p := append(append([]any{}, path...), k)
			value := renderRawJSON(t, n.Content[i+1], p, tr)
			parts = append(parts, append(append(key, ':'), value...))
			if kind == "duplicate-json-object-key" && reflect.DeepEqual(nonNilPath(path), nonNilPath(list(tr["object_path"]))) && k == tr["key"] {
				parts = append(parts, append(append(encoded(t, k), ':'), encoded(t, tr["second_value"])...))
			}
		}
		out = append(append([]byte{'{'}, bytes.Join(parts, []byte{','})...), '}')
	case yaml.SequenceNode:
		parts := [][]byte{}
		for i, c := range n.Content {
			parts = append(parts, renderRawJSON(t, c, append(append([]any{}, path...), int64(i)), tr))
		}
		out = append(append([]byte{'['}, bytes.Join(parts, []byte{','})...), ']')
	case yaml.ScalarNode:
		if n.Tag == "!!str" {
			out = encoded(t, n.Value)
		} else {
			out = []byte(n.Value)
		}
	default:
		t.Fatal("fixture setup: raw kind")
	}
	if kind == "yaml-tag-json-flow-value" && reflect.DeepEqual(path, list(tr["path"])) {
		out = append([]byte(str(tr["tag"])+" "), out...)
	}
	if kind == "yaml-anchor-alias-json-flow" && reflect.DeepEqual(path, list(tr["anchor_path"])) {
		out = append([]byte("&fixture_copy "), out...)
	}
	return out
}
func nonNilPath(p []any) []any {
	if p == nil {
		return []any{}
	}
	return p
}

type accessSpy struct {
	root     *os.Root
	attempts int
	calls    []string
	denied   []string
}

func (s *accessSpy) OpenRoot(p string) (LayoutRoot, error) {
	s.calls = append(s.calls, p)
	if !localPath(p) {
		s.attempts++
		return nil, fs.ErrPermission
	}
	child, e := s.root.OpenRoot(p)
	if e != nil {
		return nil, e
	}
	return &childAccessSpy{accessSpy: s, root: child, prefix: p}, nil
}

// Child metadata calls are projected back to the shared instrument before the
// root-relative operation; the production boundary itself stays on the child FD.
type childAccessSpy struct {
	accessSpy *accessSpy
	root      *os.Root
	prefix    string
}

func (s *childAccessSpy) Lstat(p string) (fs.FileInfo, error) {
	s.accessSpy.calls = append(s.accessSpy.calls, filepath.Join(s.prefix, p))
	return s.root.Lstat(p)
}
func (s *childAccessSpy) Readlink(p string) (string, error) {
	s.accessSpy.calls = append(s.accessSpy.calls, filepath.Join(s.prefix, p))
	return s.root.Readlink(p)
}
func (s *childAccessSpy) OpenRoot(p string) (LayoutRoot, error) {
	child, e := s.root.OpenRoot(p)
	if e != nil {
		return nil, e
	}
	return &childAccessSpy{s.accessSpy, child, filepath.Join(s.prefix, p)}, nil
}
func (s *childAccessSpy) Close() error { return s.root.Close() }

type testLayoutRoot struct{ *os.Root }

func (r testLayoutRoot) OpenRoot(p string) (LayoutRoot, error) {
	child, e := r.Root.OpenRoot(p)
	if e != nil {
		return nil, e
	}
	return testLayoutRoot{child}, nil
}

func (s *accessSpy) Lstat(p string) (fs.FileInfo, error) {
	s.calls = append(s.calls, p)
	blocked := !localPath(p)
	for _, prefix := range s.denied {
		blocked = blocked || p == prefix || strings.HasPrefix(p, prefix+string(filepath.Separator))
	}
	if blocked {
		s.attempts++
		return nil, fs.ErrPermission
	}
	return s.root.Lstat(p)
}
func (s *accessSpy) Readlink(p string) (string, error) {
	s.calls = append(s.calls, p)
	if !localPath(p) {
		s.attempts++
		return "", fs.ErrPermission
	}
	return s.root.Readlink(p)
}
func executeLayout(t *testing.T, c, setup, replace Object) *Issue {
	t.Helper()
	dir := t.TempDir()
	for _, v := range list(setup["directories"]) {
		if e := os.MkdirAll(filepath.Join(dir, str(v)), 0700); e != nil {
			t.Fatal(e)
		}
	}
	for _, v := range list(setup["files"]) {
		m := obj(v)
		if e := os.WriteFile(filepath.Join(dir, str(m["path"])), []byte(str(m["utf8"])), 0600); e != nil {
			t.Fatal(e)
		}
	}
	for _, v := range list(setup["symlinks"]) {
		m := obj(v)
		if replace != nil && replace["path"] == m["path"] {
			m = replace
		}
		if e := os.Symlink(str(m["target"]), filepath.Join(dir, str(m["path"]))); e != nil {
			t.Fatal(e)
		}
	}
	before := inventory(t, dir)
	root, e := os.OpenRoot(filepath.Join(dir, str(setup["admitted_root"])))
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	spy := &accessSpy{root: root}
	for _, v := range list(setup["denied_inspections"]) {
		spy.denied = append(spy.denied, str(v))
	}
	rel, e := filepath.Rel(str(setup["admitted_root"]), str(setup["card_directory"]))
	if e != nil {
		t.Fatal(e)
	}
	issue := CheckLayout(c, rel, spy)
	if spy.attempts != 0 {
		t.Fatal("forbidden outside inspection attempted")
	}
	// A member's bounded clone declaration may share the remote project's old
	// local spelling. Metadata reads for that independent declaration are legal;
	// the absent remote project/task directory must not cause layout failure.
	if !reflect.DeepEqual(before, inventory(t, dir)) {
		t.Fatal("layout modified fixture")
	}
	return issue
}
func inventory(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	e := filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, _ := filepath.Rel(root, p)
		if d.Type()&fs.ModeSymlink != 0 {
			v, e := os.Readlink(p)
			out[rel] = "link:" + v
			return e
		}
		if d.IsDir() {
			out[rel] = "directory"
			return nil
		}
		b, e := os.ReadFile(p)
		h := sha256.Sum256(b)
		out[rel] = hex.EncodeToString(h[:])
		return e
	})
	if e != nil {
		t.Fatal(e)
	}
	return out
}

// The instrument must record an actual forbidden inspection even though the
// underlying rooted filesystem also refuses it. This is an inert temp control.
func TestOutsideAccessSpyControl(t *testing.T) {
	r, e := os.OpenRoot(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	s := &accessSpy{root: r}
	if _, e = s.Lstat("."); e != nil || s.attempts != 0 {
		t.Fatal("positive control")
	}
	if _, e = s.Lstat("../outside"); e == nil || s.attempts != 1 {
		t.Fatal("known violation was not detected")
	}
	s.denied = []string{"inside"}
	if _, e = s.Lstat("inside"); e == nil || s.attempts != 2 {
		t.Fatal("narrower-boundary control undetected")
	}
}

// Every original full-card negative retains its neighboring accepted card and
// specific refusal invariant, including optional station preservation.
func TestOriginalNegatives(t *testing.T) {
	var oracles []caseOracle
	if e := json.Unmarshal(mustRead(t, "testdata/negative-oracles.json"), &oracles); e != nil {
		t.Fatal(e)
	}
	cases := listFromJSON(t, fixtureBytes(t, "negative-cases.json"))
	if len(cases) != len(oracles) {
		t.Fatal("negative coverage")
	}
	for i, v := range cases {
		c := obj(v)
		t.Run(str(c["id"]), func(t *testing.T) {
			if oracles[i].ID != c["id"] {
				t.Fatal("oracle id mismatch")
			}
			o := fixtureOptions(t)
			if e := Validate(fixtureBytes(t, str(c["unchanged_valid_control"])), o); e != nil {
				t.Fatal(e)
			}
			target := fixture(t, str(c["file"]))
			issue := Validate(fixtureBytes(t, str(c["file"])), o)
			if c["operation"] == "verify-conversion-preservation" {
				var in PreservationInputs
				if e := json.Unmarshal(fixtureBytes(t, "expectations.json"), &in); e != nil {
					t.Fatal(e)
				}
				issue = VerifyConversionPreservation(fixture(t, "source-v3.json"), target, in)
			}
			assertOracle(t, issue, oracles[i])
		})
	}
}
func listFromJSON(t *testing.T, b []byte) []any {
	t.Helper()
	var v any
	if e := json.Unmarshal(b, &v); e != nil {
		t.Fatal(e)
	}
	return list(v)
}
