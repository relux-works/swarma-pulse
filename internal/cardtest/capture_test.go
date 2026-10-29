package cardtest

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The moved inventory still detects byte/link changes and extra output, without
// following links. Captured buffers retain a planted partial-output violation.
func TestSharedConverterInstruments(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "input")
	if e := os.WriteFile(path, []byte("before"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink("input", filepath.Join(root, "link")); e != nil {
		t.Fatal(e)
	}
	before := Inventory(t, root)
	if !reflect.DeepEqual(before, Inventory(t, root)) {
		t.Fatal("unchanged inventory")
	}
	for _, mutation := range []func() error{
		func() error { return os.WriteFile(path, []byte("after"), 0600) },
		func() error {
			if e := os.Remove(filepath.Join(root, "link")); e != nil {
				return e
			}
			return os.Symlink("missing", filepath.Join(root, "link"))
		},
		func() error { return os.WriteFile(filepath.Join(root, "partial"), nil, 0600) },
	} {
		prior := Inventory(t, root)
		if e := mutation(); e != nil {
			t.Fatal(e)
		}
		if reflect.DeepEqual(prior, Inventory(t, root)) {
			t.Fatal("inventory plant undetected")
		}
	}
	code, out, diag := Capture(func(_ []string, out, diag io.Writer) int { fmt.Fprintln(diag, "RULE parse_document: $"); return 2 }, nil)
	if code != 2 || out != "" || diag != "RULE parse_document: $\n" {
		t.Fatal("capture control")
	}
	_, out, _ = Capture(func(_ []string, out, diag io.Writer) int { fmt.Fprint(out, "SYNTHETIC-CANARY"); return 2 }, nil)
	if out == "" {
		t.Fatal("partial output plant undetected")
	}
}
