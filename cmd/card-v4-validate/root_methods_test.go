package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Defence in depth for every Root metadata/read method in the admitted paths:
// the fixed toolchain accepts an existing in-root neighbor and rejects trailing
// separator escapes, including a second symlink hop, before returning an object.
// The CLI itself uses File.Stat rather than Root.Stat; the latter is an extra
// upstream regression control, not a claim that production calls it.
func TestNativeRootTrailingSeparatorMethods(t *testing.T) {
	base, outside := t.TempDir(), t.TempDir()
	if e := os.Mkdir(filepath.Join(base, "inside"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink("inside", filepath.Join(base, "inside-link")); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(outside, filepath.Join(base, "escape")); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink("escape", filepath.Join(base, "hop")); e != nil {
		t.Fatal(e)
	}
	root, e := os.OpenRoot(base)
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	methods := []struct {
		name, positive string
		call           func(string) error
	}{
		{"OpenRoot", "inside", func(p string) error {
			r, e := root.OpenRoot(p)
			if e == nil {
				r.Close()
			}
			return e
		}},
		{"Open", "inside", func(p string) error {
			f, e := root.Open(p)
			if e == nil {
				f.Close()
			}
			return e
		}},
		{"Stat", "inside", func(p string) error { _, e := root.Stat(p); return e }},
		{"Lstat", "inside-link", func(p string) error { _, e := root.Lstat(p); return e }},
		{"Readlink", "inside-link", func(p string) error { _, e := root.Readlink(p); return e }},
	}
	for _, method := range methods {
		t.Run(method.name, func(t *testing.T) {
			if e := method.call(method.positive); e != nil {
				t.Fatal("positive neighbor")
			}
			for _, p := range []string{"escape/", "escape/./", "hop/"} {
				e := method.call(p)
				if e == nil || !strings.Contains(e.Error(), "path escapes from parent") {
					t.Fatalf("expected native root-escape refusal for %s", p)
				}
			}
		})
	}
}
