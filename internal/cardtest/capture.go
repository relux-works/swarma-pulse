// Package cardtest shares the converter test instruments; production commands
// do not import it. Inventories are for isolated synthetic trees only.
package cardtest

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func Inventory(t testing.TB, root string) map[string]string {
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

// Capture uses the same isolated buffers for complete CLI receipts/refusals.
func Capture(run func([]string, io.Writer, io.Writer) int, args []string) (int, string, string) {
	var out, diagnostics bytes.Buffer
	code := run(args, &out, &diagnostics)
	return code, out.String(), diagnostics.String()
}
