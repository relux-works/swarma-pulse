package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const fixtures = "../../docs/fixtures/v4/full"

func inventory(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	e := filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, _ := filepath.Rel(root, p)
		if d.Type()&fs.ModeSymlink != 0 {
			link, e := os.Readlink(p)
			out[rel] = "link:" + link
			return e
		}
		if d.IsDir() {
			out[rel] = "dir"
			return nil
		}
		raw, e := os.ReadFile(p)
		out[rel] = string(raw)
		return e
	})
	if e != nil {
		t.Fatal(e)
	}
	return out
}
func setup(t *testing.T) (string, []string) {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"legacy-card", "aurora-host"} {
		if e := os.Mkdir(filepath.Join(root, dir), 0700); e != nil {
			t.Fatal(e)
		}
	}
	for _, f := range []struct{ src, dst string }{{"source-v3.json", "legacy-card/product-card.yaml"}, {"expectations.json", "inputs.json"}} {
		raw, e := os.ReadFile(filepath.Join(fixtures, f.src))
		if e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(filepath.Join(root, f.dst), raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	if e := os.WriteFile(filepath.Join(root, "aurora-host/product-card.yaml"), []byte("PRIOR\n"), 0600); e != nil {
		t.Fatal(e)
	}
	return root, []string{"--root", root, "--source", "legacy-card/product-card.yaml", "--inputs", filepath.Join(root, "inputs.json"), "--destination", "aurora-host/product-card.yaml"}
}

// The real CLI publishes only the complete expected document, preserves source
// and premises, and emits a fixed receipt. A fault-looking environment cannot
// activate the unexported test seams in the production path.
func TestCLIConversion(t *testing.T) {
	root, args := setup(t)
	before := inventory(t, root)
	t.Setenv("CARD_V4_CONVERT_FAULT_AFTER_MEMBER", "2")
	var out, diag bytes.Buffer
	if exit := run(args, &out, &diag); exit != 0 || out.String() != "converted\n" || diag.Len() != 0 {
		t.Fatal(exit, diag.String())
	}
	expected, e := os.ReadFile(filepath.Join(fixtures, "converted-v4.json"))
	if e != nil {
		t.Fatal(e)
	}
	actual, e := os.ReadFile(filepath.Join(root, "aurora-host/product-card.yaml"))
	if e != nil {
		t.Fatal(e)
	}
	var a, b any
	if json.Unmarshal(expected, &a) != nil || json.Unmarshal(actual, &b) != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("complete expected output differs")
	}
	before["aurora-host/product-card.yaml"] = string(actual)
	if !reflect.DeepEqual(before, inventory(t, root)) {
		t.Fatal("unexpected CLI effects")
	}
}

// Usage, missing premises, bad pins, aliases, stale temps and escaping directory
// spellings refuse exactly, leaving both isolated trees and captured output safe.
func TestCLIRefusals(t *testing.T) {
	for _, tc := range []struct {
		name, rule string
		exit       int
	}{
		{"fault flag", "usage", 2}, {"missing source", "source_unreadable", 2}, {"bad pin", "source_digest_mismatch", 1}, {"source escape", "conversion_source_base_mismatch", 1}, {"destination escape", "conversion_destination_base_mismatch", 1}, {"hardlink", "publication_source_alias", 1}, {"symlink", "publication_destination_type", 1}, {"leftover", "publication_leftover_temp", 1}, {"parent escape", "publication_root_escape_or_unreadable", 1}, {"parent escape slash", "publication_root_escape_or_unreadable", 1}, {"wrong target base", "conversion_destination_base_mismatch", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, args := setup(t)
			outside := t.TempDir()
			source, destination := filepath.Join(root, "legacy-card/product-card.yaml"), filepath.Join(root, "aurora-host/product-card.yaml")
			switch tc.name {
			case "fault flag":
				args = append(args, "--fault-after-member=2")
			case "missing source":
				if e := os.Remove(source); e != nil {
					t.Fatal(e)
				}
			case "bad pin":
				raw, e := os.ReadFile(filepath.Join(root, "inputs.json"))
				if e != nil {
					t.Fatal(e)
				}
				var in map[string]any
				if json.Unmarshal(raw, &in) != nil {
					t.Fatal("premises")
				}
				in["source_sha256"] = strings.Repeat("a", 64)
				raw, e = json.Marshal(in)
				if e != nil {
					t.Fatal(e)
				}
				if e := os.WriteFile(filepath.Join(root, "inputs.json"), raw, 0600); e != nil {
					t.Fatal(e)
				}
			case "source escape":
				args[3] = "../source"
			case "destination escape":
				args[7] = "../destination"
			case "wrong target base":
				args[7] = "other/card"
			case "hardlink", "symlink":
				if e := os.Remove(destination); e != nil {
					t.Fatal(e)
				}
				var e error
				if tc.name == "hardlink" {
					e = os.Link(source, destination)
				} else {
					e = os.Symlink("../legacy-card/product-card.yaml", destination)
				}
				if e != nil {
					t.Fatal(e)
				}
			case "leftover":
				if e := os.WriteFile(filepath.Join(root, "aurora-host/.card-v4-convert-product-card.yaml.tmp"), []byte("PARTIAL"), 0600); e != nil {
					t.Fatal(e)
				}
			case "parent escape", "parent escape slash":
				if e := os.Rename(filepath.Join(root, "aurora-host"), filepath.Join(root, "preserved")); e != nil {
					t.Fatal(e)
				}
				if e := os.Symlink(outside, filepath.Join(root, "aurora-host")); e != nil {
					t.Fatal(e)
				}
				if tc.name == "parent escape slash" {
					args[7] = "aurora-host//product-card.yaml"
				}
			}
			before, external := inventory(t, root), inventory(t, outside)
			var out, diag bytes.Buffer
			exit := run(args, &out, &diag)
			if exit != tc.exit || out.Len() != 0 || diag.String() != "RULE "+tc.rule+": $\n" {
				t.Fatal(exit, diag.String())
			}
			if !reflect.DeepEqual(before, inventory(t, root)) || !reflect.DeepEqual(external, inventory(t, outside)) {
				t.Fatal("refusal mutated inputs/destination")
			}
		})
	}
}
