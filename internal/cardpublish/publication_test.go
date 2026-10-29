package cardpublish

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var injected = errors.New("isolated injected fault")

type faultFile struct {
	tempFile
	fault string
}

func (f faultFile) Write(p []byte) (int, error) {
	if f.fault == "write" {
		return 0, injected
	}
	if f.fault == "short" {
		return len(p) - 1, nil
	}
	return f.tempFile.Write(p)
}
func (f faultFile) Sync() error {
	if f.fault == "sync" {
		return injected
	}
	return f.tempFile.Sync()
}
func (f faultFile) Close() error {
	e := f.tempFile.Close()
	if f.fault == "close" {
		return injected
	}
	return e
}

type faultOps struct {
	nativeOps
	fault  string
	events *[]string
}

func (f faultOps) create(r *os.Root, n string) (tempFile, error) {
	p, e := f.nativeOps.create(r, n)
	if e != nil {
		return nil, e
	}
	return faultFile{p, f.fault}, nil
}
func (f faultOps) rename(r *os.Root, a, b string) error {
	if f.fault == "rename" {
		return injected
	}
	return f.nativeOps.rename(r, a, b)
}
func (f faultOps) syncDirectory(d *os.File) error {
	if f.fault == "dirsync" {
		return injected
	}
	return f.nativeOps.syncDirectory(d)
}
func (f faultOps) remove(r *os.Root, n string) error {
	if f.fault == "cleanup" {
		return injected
	}
	return f.nativeOps.remove(r, n)
}
func inventory(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	e := filepath.WalkDir(dir, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, _ := filepath.Rel(dir, p)
		if d.Type()&os.ModeSymlink != 0 {
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
func setup(t *testing.T) (string, *os.Root, fs.FileInfo) {
	t.Helper()
	dir := t.TempDir()
	if e := os.WriteFile(filepath.Join(dir, "source"), []byte("SOURCE"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(dir, "card"), []byte("PRIOR"), 0600); e != nil {
		t.Fatal(e)
	}
	info, e := os.Stat(filepath.Join(dir, "source"))
	if e != nil {
		t.Fatal(e)
	}
	r, e := os.OpenRoot(dir)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { r.Close() })
	return dir, r, info
}

// All pre-rename injected I/O failures leave exact source/destination bytes and
// no temporary file. Directory-sync failure honestly reports an already complete
// publication. The no-fault neighbor publishes exactly the supplied bytes.
func TestPublicationFaults(t *testing.T) {
	for _, tc := range []struct{ fault, rule string }{
		{"", ""}, {"write", "publication_write"}, {"short", "publication_write"}, {"sync", "publication_temp_sync"}, {"close", "publication_close"}, {"rename", "publication_rename"}, {"dirsync", "publication_durability_unconfirmed"},
	} {
		t.Run(tc.fault, func(t *testing.T) {
			dir, r, source := setup(t)
			before := inventory(t, dir)
			complete := []byte("COMPLETE\n")
			result, e := publish(r, "card", source, complete, faultOps{fault: tc.fault})
			if tc.rule == "" {
				if e != nil || !result.Published || !result.Durable {
					t.Fatal(result, e)
				}
			} else if e == nil || e.Rule != tc.rule {
				t.Fatal(result, e)
			}
			after := inventory(t, dir)
			if tc.fault == "" || tc.fault == "dirsync" {
				before["card"] = string(complete)
				if !result.Published || result.Durable != (tc.fault == "") {
					t.Fatal("wrong commit status")
				}
			} else if result.Published || result.Durable || result.TemporaryRemains {
				t.Fatal("false commit status")
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("unexpected source, destination or temp effects")
			}
		})
	}
}

// A prior-run temporary is a refusal, including a symlink; it is never read,
// published, truncated or removed. After explicit isolated cleanup retry works.
func TestLeftoverTemp(t *testing.T) {
	for _, link := range []bool{false, true} {
		t.Run(map[bool]string{false: "file", true: "symlink"}[link], func(t *testing.T) {
			dir, r, source := setup(t)
			temp := filepath.Join(dir, ".card-v4-convert-card.tmp")
			var e error
			if link {
				e = os.Symlink("source", temp)
			} else {
				e = os.WriteFile(temp, []byte("PARTIAL FROM PRIOR RUN"), 0600)
			}
			if e != nil {
				t.Fatal(e)
			}
			before := inventory(t, dir)
			result, issue := Publish(r, "card", source, []byte("COMPLETE"))
			if issue == nil || issue.Rule != "publication_leftover_temp" || result.Published {
				t.Fatal(result, issue)
			}
			if !reflect.DeepEqual(before, inventory(t, dir)) {
				t.Fatal("adopted or changed old temporary")
			}
			if e := os.Remove(temp); e != nil {
				t.Fatal(e)
			}
			if result, e := Publish(r, "card", source, []byte("COMPLETE")); e != nil || !result.Durable {
				t.Fatal(result, e)
			}
		})
	}
}

// The admitted root and opened source identity reject traversal, escaping links,
// destination symlinks and hardlink aliases, with exact byte inventories intact.
func TestPublicationBoundaries(t *testing.T) {
	for _, tc := range []struct{ path, rule string }{
		{"../outside", "publication_inputs"}, {"/outside", "publication_inputs"}, {".", "publication_inputs"}, {"source", "publication_source_alias"}, {"hardlink", "publication_source_alias"}, {"link", "publication_destination_type"}, {"escape/card", "publication_root_escape_or_unreadable"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			dir, r, source := setup(t)
			outside := t.TempDir()
			if e := os.Link(filepath.Join(dir, "source"), filepath.Join(dir, "hardlink")); e != nil {
				t.Fatal(e)
			}
			if e := os.Symlink("source", filepath.Join(dir, "link")); e != nil {
				t.Fatal(e)
			}
			if e := os.Symlink(outside, filepath.Join(dir, "escape")); e != nil {
				t.Fatal(e)
			}
			before, external := inventory(t, dir), inventory(t, outside)
			result, e := Publish(r, tc.path, source, []byte("COMPLETE"))
			if e == nil || e.Rule != tc.rule || result.Published {
				t.Fatal(result, e)
			}
			if !reflect.DeepEqual(before, inventory(t, dir)) || !reflect.DeepEqual(external, inventory(t, outside)) {
				t.Fatal("boundary effect")
			}
		})
	}
}

// Visibility is whole-document: successful replacement changes the destination
// inode while a previously opened reader retains the complete prior file.
func TestAtomicReplacement(t *testing.T) {
	dir, r, source := setup(t)
	old, e := os.Open(filepath.Join(dir, "card"))
	if e != nil {
		t.Fatal(e)
	}
	defer old.Close()
	complete := bytes.Repeat([]byte("complete\n"), 4096)
	if result, e := Publish(r, "card", source, complete); e != nil || !result.Durable {
		t.Fatal(result, e)
	}
	prior := make([]byte, 5)
	if n, e := old.Read(prior); e != nil || n != 5 || string(prior) != "PRIOR" {
		t.Fatal("old reader lost prior bytes")
	}
	now, e := os.ReadFile(filepath.Join(dir, "card"))
	if e != nil || !bytes.Equal(now, complete) {
		t.Fatal("partial replacement")
	}
}

type orderedFile struct {
	tempFile
	events *[]string
}

func (f orderedFile) Write(p []byte) (int, error) {
	*f.events = append(*f.events, "write")
	return f.tempFile.Write(p)
}
func (f orderedFile) Sync() error {
	*f.events = append(*f.events, "file-sync")
	return f.tempFile.Sync()
}
func (f orderedFile) Close() error { *f.events = append(*f.events, "close"); return f.tempFile.Close() }

type orderedOps struct {
	nativeOps
	events *[]string
}

func (o orderedOps) create(r *os.Root, n string) (tempFile, error) {
	f, e := o.nativeOps.create(r, n)
	if e != nil {
		return nil, e
	}
	return orderedFile{f, o.events}, nil
}
func (o orderedOps) rename(r *os.Root, a, b string) error {
	*o.events = append(*o.events, "rename")
	return o.nativeOps.rename(r, a, b)
}
func (o orderedOps) syncDirectory(f *os.File) error {
	*o.events = append(*o.events, "directory-sync")
	return o.nativeOps.syncDirectory(f)
}

// Actual native publication orders complete write, temp fsync, close, atomic
// rename and directory fsync. A planted reordered observation is distinguishable.
func TestPublicationSyncOrder(t *testing.T) {
	_, root, source := setup(t)
	events := []string{}
	if result, e := publish(root, "card", source, []byte("COMPLETE"), orderedOps{events: &events}); e != nil || !result.Durable {
		t.Fatal(result, e)
	}
	expected := []string{"write", "file-sync", "close", "rename", "directory-sync"}
	if !reflect.DeepEqual(events, expected) {
		t.Fatal("wrong native publication order", events)
	}
	events[1], events[3] = events[3], events[1]
	if reflect.DeepEqual(events, expected) {
		t.Fatal("reordered control undetected")
	}
}

type cleanupFaultOps struct{ nativeOps }

func (o cleanupFaultOps) rename(*os.Root, string, string) error { return injected }
func (o cleanupFaultOps) remove(*os.Root, string) error         { return injected }

// If the filesystem also refuses cleanup, report the artifact honestly and
// refuse the next run; prior destination/source remain intact, never adopted.
func TestCleanupFailure(t *testing.T) {
	dir, root, source := setup(t)
	before := inventory(t, dir)
	result, e := publish(root, "card", source, []byte("COMPLETE"), cleanupFaultOps{})
	if e == nil || e.Rule != "publication_cleanup_failed" || result.Published || !result.TemporaryRemains {
		t.Fatal(result, e)
	}
	before[".card-v4-convert-card.tmp"] = "COMPLETE"
	if !reflect.DeepEqual(before, inventory(t, dir)) {
		t.Fatal("unexpected cleanup failure effects")
	}
	if result, e := Publish(root, "card", source, []byte("NEW")); e == nil || e.Rule != "publication_leftover_temp" || result.Published {
		t.Fatal(result, e)
	}
}
