package pulse

import (
	"bytes"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// Run this control with both a protected filesystem root and a writable root in
// an isolated mount namespace. The fixture's own ancestry stays protected in
// both cases, so only the filesystem root changes the admission result.
func TestQueryFilesystemRootTrust(t *testing.T) {
	rootInfo, err := os.Lstat(string(filepath.Separator))
	if err != nil {
		t.Fatal(err)
	}
	stat, ok := rootInfo.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatal("missing root ownership metadata")
	}
	trusted := rootInfo.IsDir() && rootInfo.Mode()&os.ModeSymlink == 0 &&
		(stat.Uid == 0 || stat.Uid == uint32(os.Getuid())) && rootInfo.Mode().Perm()&0022 == 0
	t.Logf("filesystem root mode=%04o trusted=%t", rootInfo.Mode().Perm(), trusted)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw := validBytes(t, fixture(t))
	card := filepath.Join(root, "product-card.yaml")
	if err := os.WriteFile(card, raw, 0600); err != nil {
		t.Fatal(err)
	}
	manifest := sourcesBytes(t)
	sources := filepath.Join(root, "sources.json")
	if err := os.WriteFile(sources, manifest, 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(card)
	if err != nil {
		t.Fatal(err)
	}
	for _, view := range []string{"", "environment", "discovery"} {
		t.Run("view="+view, func(t *testing.T) {
			s := selectors{card: card, revision: "sha256:" + digest(raw), view: view}
			if view == "discovery" {
				s.root, s.sources, s.sourcesRevision = root, sources, "sha256:"+digest(manifest)
			}
			var out, diag bytes.Buffer
			code := runSelected(s, os.Getuid(), os.Geteuid(), "station_aurora_keeper", fixedTime, &out, &diag)
			if trusted {
				if code != 0 || out.Len() == 0 || diag.Len() != 0 {
					t.Fatalf("protected-root control: exit=%d stdout=%d stderr=%q", code, out.Len(), diag.String())
				}
			} else if code != 1 || out.Len() != 0 || diag.String() != "RULE input_trust: $\n" {
				t.Fatalf("unsafe-root refusal: exit=%d stdout=%d stderr=%q", code, out.Len(), diag.String())
			}
		})
	}
	after, err := os.Stat(card)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(card)
	if err != nil || !bytes.Equal(got, raw) || !os.SameFile(before, after) || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("query changed its input")
	}
}
