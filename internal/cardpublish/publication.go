// Package cardpublish publishes complete bytes into an explicitly admitted tree.
// It has no conversion, network, process or registration capability.
package cardpublish

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Result distinguishes failures before publication from a complete publication
// whose directory fsync failed. TemporaryRemains requires manual cleanup; a later
// run always refuses that exclusive temp, never adopts it as a complete card.
type Result struct {
	Published        bool
	Durable          bool
	TemporaryRemains bool
}
type Issue struct{ Rule string }

func (e *Issue) Error() string { return e.Rule }

type tempFile interface {
	io.Writer
	Sync() error
	Close() error
}
type publicationOps interface {
	create(*os.Root, string) (tempFile, error)
	rename(*os.Root, string, string) error
	remove(*os.Root, string) error
	syncDirectory(*os.File) error
}
type nativeOps struct{}

func (nativeOps) create(r *os.Root, name string) (tempFile, error) {
	return r.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
}
func (nativeOps) rename(r *os.Root, a, b string) error { return r.Rename(a, b) }
func (nativeOps) remove(r *os.Root, name string) error { return r.Remove(name) }
func (nativeOps) syncDirectory(f *os.File) error       { return f.Sync() }

// Publish pins the destination directory under root and uses a private exclusive
// same-directory temp. Source is the FileInfo of the opened input, not its path.
// Its only commit point is rename; a later fsync failure returns Published=true.
// Callers must validate the complete bytes before invoking this generic adapter.
func Publish(root *os.Root, destination string, source fs.FileInfo, complete []byte) (Result, *Issue) {
	return publish(root, destination, source, complete, nativeOps{})
}
func publish(root *os.Root, destination string, source fs.FileInfo, complete []byte, ops publicationOps) (result Result, issue *Issue) {
	refuse := func(rule string) (Result, *Issue) { return result, &Issue{rule} }
	if root == nil || source == nil || len(complete) == 0 || !filepath.IsLocal(destination) || filepath.Clean(destination) == "." {
		return refuse("publication_inputs")
	}
	directory, name := filepath.Split(filepath.Clean(destination))
	if directory == "" {
		directory = "."
	}
	directory = filepath.Clean(directory)
	parent, e := root.OpenRoot(directory)
	if e != nil {
		return refuse("publication_root_escape_or_unreadable")
	}
	defer parent.Close()
	if existing, e := parent.Lstat(name); e == nil {
		if !existing.Mode().IsRegular() {
			return refuse("publication_destination_type")
		}
		if os.SameFile(existing, source) {
			return refuse("publication_source_alias")
		}
	} else if !errors.Is(e, fs.ErrNotExist) {
		return refuse("publication_destination_unreadable")
	}
	dirFile, e := parent.Open(".")
	if e != nil {
		return refuse("publication_directory_unreadable")
	}
	defer dirFile.Close()
	temporary := ".card-v4-convert-" + name + ".tmp"
	f, e := ops.create(parent, temporary)
	if e != nil {
		if errors.Is(e, fs.ErrExist) {
			return refuse("publication_leftover_temp")
		}
		return refuse("publication_temp_create")
	}
	closed := false
	defer func() {
		if !closed {
			_ = f.Close()
		}
		if !result.Published {
			if e := ops.remove(parent, temporary); e != nil && !errors.Is(e, fs.ErrNotExist) {
				result.TemporaryRemains = true
				issue = &Issue{"publication_cleanup_failed"}
			}
		}
	}()
	n, e := f.Write(complete)
	if e != nil || n != len(complete) {
		return refuse("publication_write")
	}
	if e := f.Sync(); e != nil {
		return refuse("publication_temp_sync")
	}
	e = f.Close()
	closed = true
	if e != nil {
		return refuse("publication_close")
	}
	if e := ops.rename(parent, temporary, name); e != nil {
		return refuse("publication_rename")
	}
	result.Published = true
	if e := ops.syncDirectory(dirFile); e != nil {
		return refuse("publication_durability_unconfirmed")
	}
	result.Durable = true
	return result, nil
}
