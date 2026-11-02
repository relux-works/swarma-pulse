package pulse

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Discovery selectors are an explicit all-or-nothing opt-in. Malformed selectors
// refuse as usage before touching even the deliberately nonexistent source path.
func TestDiscoverySelectorBoundary(t *testing.T) {
	base := []string{"me", "--local-unsigned", "--view", "discovery", "--card", "/absent/card", "--revision", "sha256:" + strings.Repeat("a", 64), "--format", "json", "--sources", "/absent/sources.json", "--sources-revision", "sha256:" + strings.Repeat("b", 64), "--root", "/absent"}
	s, err := parseSelectors(base)
	if err != nil || s.view != "discovery" || s.root != "/absent" {
		t.Fatal(s, err)
	}
	for _, key := range []string{"--sources", "--sources-revision", "--root"} {
		args := append([]string{}, base...)
		for i, a := range args {
			if a == key {
				args = append(args[:i], args[i+2:]...)
				break
			}
		}
		var out, diag bytes.Buffer
		if code := Run(args, &out, &diag); code != 2 || out.Len() != 0 || diag.String() != "RULE usage: $\n" {
			t.Fatal(key, code, diag.String())
		}
	}
	for _, view := range []string{"environment", ""} {
		args := append([]string{}, base...)
		args[3] = view
		_, err := parseSelectors(args)
		wantRule(t, err, "usage")
	}
	for _, root := range []string{"relative", "/a/../b", "/a\n", ""} {
		args := append([]string{}, base...)
		args[len(args)-1] = root
		_, err := parseSelectors(args)
		wantRule(t, err, "usage")
	}
	for _, flag := range []string{"--sources", "--sources-revision", "--root", "--view"} {
		_, err := parseSelectors(append(append([]string{}, base...), flag, "/absent"))
		wantRule(t, err, "usage")
	}
}

func sourcesBytes(t *testing.T, entries ...any) []byte {
	t.Helper()
	if entries == nil {
		entries = []any{}
	}
	return encode(t, object{"schema": "pulse-sources/v1", "cards": entries})
}
func sourceEntry(id, path string) object {
	return object{"id": id, "schema": "product-card/v3", "path": path, "revision": "sha256:" + strings.Repeat("a", 64)}
}

// Manifest JSON is closed and independently pinned. Sibling paths are legitimate
// only inside the selected read root; duplicate normalized files never merge.
func TestSourceManifestContract(t *testing.T) {
	raw := sourcesBytes(t, sourceEntry("sibling", "../sibling/product-card.yaml"))
	items, err := parseSourceManifest(raw, "sha256:"+digest(raw), "/fixture/primary/product-card.yaml", "/fixture")
	if err != nil || len(items) != 1 || items[0].path != "/fixture/sibling/product-card.yaml" {
		t.Fatal(items, err)
	}
	_, err = parseSourceManifest(raw, "sha256:"+strings.Repeat("0", 64), "/fixture/primary/product-card.yaml", "/fixture")
	wantRule(t, err, "source_digest")
	cases := []struct {
		name string
		raw  []byte
		rule string
	}{
		{"yaml", []byte("schema: pulse-sources/v1\ncards: []\n"), "source_manifest_invalid"},
		{"duplicate-json-key", []byte(`{"schema":"pulse-sources/v1","cards":[],"cards":[]}`), "source_manifest_invalid"},
		{"wrong-schema", encode(t, object{"schema": "unknown", "cards": []any{}}), "source_manifest_invalid"},
		{"null-cards", encode(t, object{"schema": "pulse-sources/v1", "cards": nil}), "source_manifest_invalid"},
		{"extra", encode(t, object{"schema": "pulse-sources/v1", "cards": []any{}, "extra": true}), "source_manifest_invalid"},
		{"primary-repeat", sourcesBytes(t, sourceEntry("repeat", "product-card.yaml")), "source_conflict"},
		{"duplicate-id", sourcesBytes(t, sourceEntry("same", "a"), sourceEntry("same", "b")), "source_conflict"},
		{"duplicate-path", sourcesBytes(t, sourceEntry("a", "../sibling/card"), sourceEntry("b", "../sibling/x/../card")), "source_conflict"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseSourceManifest(tc.raw, "sha256:"+digest(tc.raw), "/fixture/primary/product-card.yaml", "/fixture")
			wantRule(t, err, tc.rule)
		})
	}
	for _, path := range []string{"/absolute", "~home", "https:remote", "C:drive", "\\\\host\\file", "bad\x00", "bad\n", "../../escape"} {
		b := sourcesBytes(t, sourceEntry("p", path))
		_, err := parseSourceManifest(b, "sha256:"+digest(b), "/fixture/primary/product-card.yaml", "/fixture")
		wantRule(t, err, "input_path")
	}
	for _, change := range []func(object){func(e object) { e["extra"] = true }, func(e object) { e["schema"] = "product-card/v4" }, func(e object) { e["revision"] = "bad" }, func(e object) { delete(e, "id") }} {
		e := sourceEntry("p", "../p/card")
		change(e)
		b := sourcesBytes(t, e)
		_, err := parseSourceManifest(b, "sha256:"+digest(b), "/fixture/primary/product-card.yaml", "/fixture")
		wantRule(t, err, "source_manifest_invalid")
	}
	entries := []any{}
	for i := 0; i < 33; i++ {
		entries = append(entries, sourceEntry("p", "p"))
	}
	b := sourcesBytes(t, entries...)
	_, err = parseSourceManifest(b, "sha256:"+digest(b), "/fixture/primary/card", "/fixture")
	wantRule(t, err, "input_limit")
	// The exact maximum entry count is admitted before duplicate/card validation.
	entries = entries[:32]
	for i := range entries {
		entries[i] = sourceEntry(strings.Repeat("x", i+1), strings.Repeat("x", i+1))
	}
	b = sourcesBytes(t, entries...)
	got, err := parseSourceManifest(b, "sha256:"+digest(b), "/fixture/primary/card", "/fixture")
	if err != nil || len(got) != 32 {
		t.Fatal(len(got), err)
	}
}

// Read-root containment is lexical first, followed by the shipped no-symlink,
// owner-held file reader. No refusal mutates the file bytes, mode, inode or mtime.
func TestDiscoveryReadBoundaryPreservation(t *testing.T) {
	root := ownDirectory(t)
	raw := sourcesBytes(t)
	path := writeFixture(t, root, raw)
	before, _ := os.Stat(path)
	if err := insideReadRoot(root, path); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{root, filepath.Dir(root), filepath.Join(filepath.Dir(root), "outside")} {
		wantRule(t, insideReadRoot(root, p), "input_path")
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	_, err := readCard(link, os.Getuid())
	wantRule(t, err, "input_trust")
	b, err := readCard(path, os.Getuid())
	if err != nil || !json.Valid(b) || !bytes.Equal(b, raw) {
		t.Fatal(err)
	}
	after, _ := os.Stat(path)
	if !os.SameFile(before, after) || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("source metadata changed")
	}
	got, _ := os.ReadFile(path)
	if !reflect.DeepEqual(raw, got) {
		t.Fatal("source bytes changed")
	}
}
