package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func rootPathInventory(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	if e := filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, _ := filepath.Rel(root, path)
		if d.Type()&fs.ModeSymlink != 0 {
			link, e := os.Readlink(path)
			out[rel] = "link:" + link
			return e
		}
		if d.IsDir() {
			out[rel] = "dir"
			return nil
		}
		raw, e := os.ReadFile(path)
		out[rel] = string(raw)
		return e
	}); e != nil {
		t.Fatal(e)
	}
	return out
}

// The inventory observes link spelling without following the link; a known
// changed-link control must differ from the stable positive inventory.
func TestRootPathInventoryControl(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(root, "link")
	if e := os.Symlink("missing-first", link); e != nil {
		t.Fatal(e)
	}
	before := rootPathInventory(t, root)
	if !reflect.DeepEqual(before, rootPathInventory(t, root)) {
		t.Fatal("unstable inventory")
	}
	if e := os.Remove(link); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink("missing-second", link); e != nil {
		t.Fatal(e)
	}
	if reflect.DeepEqual(before, rootPathInventory(t, root)) {
		t.Fatal("changed link undetected")
	}
}

// The production CLI accepts trailing separators on an in-root profiles root,
// but refuses the equivalent escaping symlink spellings without changing either
// tree. This exercises CheckLayout's cleaned OpenRoot path, not a test substitute.
func TestCLIProfileRootTrailingSeparators(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	for _, dir := range []string{"aurora-host", "beacon", "vault", "profiles/control"} {
		if e := os.MkdirAll(filepath.Join(root, dir), 0700); e != nil {
			t.Fatal(e)
		}
	}
	for _, dir := range []string{"aurora-host", "beacon", "vault"} {
		if e := os.WriteFile(filepath.Join(root, dir, "tasks.md"), []byte("fixture"), 0600); e != nil {
			t.Fatal(e)
		}
	}
	if e := os.Mkdir(filepath.Join(outside, "control"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(outside, filepath.Join(root, "escape")); e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile(filepath.Join(fixtureRoot, "converted-v4.json"))
	if e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct{ path, diagnostic string }{
		{"../profiles/", ""}, {"../profiles/./", ""}, {"../escape/", "RULE path_symlink_escape: $.profiles_root\n"}, {"../escape/./", "RULE path_symlink_escape: $.profiles_root\n"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			var c map[string]any
			if json.Unmarshal(raw, &c) != nil {
				t.Fatal("fixture")
			}
			c["profiles_root"] = tc.path
			modified, e := json.Marshal(c)
			if e != nil {
				t.Fatal(e)
			}
			card := filepath.Join(root, "card.json")
			if e := os.WriteFile(card, modified, 0600); e != nil {
				t.Fatal(e)
			}
			before, external := rootPathInventory(t, root), rootPathInventory(t, outside)
			args := []string{"--check-layout", "--root", root, "--card-directory", "aurora-host", "--source-manifest", filepath.Join(fixtureRoot, "manifest.json"), "--reservations", filepath.Join(fixtureRoot, "expectations.json"), card}
			var out, diag bytes.Buffer
			exit := run(args, &out, &diag)
			if tc.diagnostic == "" {
				if exit != 0 || out.String() != "valid\n" || diag.Len() != 0 {
					t.Fatal(exit, diag.String())
				}
			} else if exit != 1 || out.Len() != 0 || diag.String() != tc.diagnostic {
				t.Fatal(exit, diag.String())
			}
			if !reflect.DeepEqual(before, rootPathInventory(t, root)) || !reflect.DeepEqual(external, rootPathInventory(t, outside)) {
				t.Fatal("changed isolated tree")
			}
		})
	}
}

// The real manifest-base/source path accepts an ordinary pinned source. A
// trailing-separator source symlink cannot acquire data outside that base.
func TestCLIManifestTrailingSeparators(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	source, e := os.ReadFile(filepath.Join(fixtureRoot, "source-v3.json"))
	if e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "source-v3.json"), source, 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(outside, "source-v3.json"), source, 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(outside, filepath.Join(root, "escape")); e != nil {
		t.Fatal(e)
	}
	original, e := os.ReadFile(filepath.Join(fixtureRoot, "manifest.json"))
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"source-v3.json", "escape/", "escape/./", "escape/source-v3.json"} {
		t.Run(name, func(t *testing.T) {
			var m map[string]any
			if json.Unmarshal(original, &m) != nil {
				t.Fatal("fixture")
			}
			pin := m["source_pins"].([]any)[0].(map[string]any)
			pin["file"] = name
			m["source_pins"] = []any{pin}
			raw, e := json.Marshal(m)
			if e != nil {
				t.Fatal(e)
			}
			manifest := filepath.Join(root, "manifest.json")
			if e := os.WriteFile(manifest, raw, 0600); e != nil {
				t.Fatal(e)
			}
			before, external := rootPathInventory(t, root), rootPathInventory(t, outside)
			// Preserve /./ in the CLI argument so filepath.Dir runs on that spelling.
			args := []string{"--source-manifest", root + "/./manifest.json", "--reservations", filepath.Join(fixtureRoot, "expectations.json"), filepath.Join(fixtureRoot, "converted-v4.json")}
			var out, diag bytes.Buffer
			exit := run(args, &out, &diag)
			if name == "source-v3.json" {
				if exit != 0 || out.String() != "valid\n" || diag.Len() != 0 {
					t.Fatal(exit, diag.String())
				}
			} else if exit != 2 || out.Len() != 0 || diag.String() != "RULE source_unreadable: $\n" {
				t.Fatal(exit, diag.String())
			}
			if !reflect.DeepEqual(before, rootPathInventory(t, root)) || !reflect.DeepEqual(external, rootPathInventory(t, outside)) {
				t.Fatal("changed manifest boundary")
			}
		})
	}
}
