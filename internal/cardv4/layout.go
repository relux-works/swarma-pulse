package cardv4

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
)

// LayoutFS admits metadata reads only, relative to an already opened root.
// Implementations must enforce containment atomically (e.g. os.Root); the walker
// additionally distinguishes lexical escape, link escape and missing entries.
type LayoutFS interface {
	Lstat(string) (fs.FileInfo, error)
	Readlink(string) (string, error)
}

// CheckLayout never opens remote locators or requires member checkouts to exist.
// cardDir is relative to the externally admitted root, not supplied by the card.
func CheckLayout(card Object, cardDir string, access LayoutFS) *Issue {
	if access == nil || !localPath(cardDir) {
		return shape("layout_root", "$")
	}
	base, e := resolvePath(access, ".", cardDir, true, false, "$.host_project")
	if e != nil {
		return e
	}
	profiles, e := resolvePath(access, ".", filepath.Join(base, str(card["profiles_root"])), true, false, "$.profiles_root")
	if e != nil {
		return e
	}
	for i, k := range sorted(obj(card["profiles"])) {
		p := obj(obj(card["profiles"])[k])
		if path, ok := p["path"]; ok {
			candidate := filepath.Join(profiles, str(path))
			if !beneath(profiles, candidate) {
				return fail("path_root_escape", index("$.profiles", i)+".path")
			}
			target, e := resolveBounded(access, profiles, profiles, str(path), true, false, index("$.profiles", i)+".path")
			if e != nil {
				return e
			}
			if !beneath(profiles, target) {
				return fail("path_symlink_escape", index("$.profiles", i)+".path")
			}
		}
	}
	for i, v := range list(card["projects"]) {
		p := obj(v)
		loc := obj(p["location"])
		if loc["kind"] == "remote" {
			continue
		}
		path := str(loc["path"])
		taskBase := true
		if loc == nil {
			path = str(p["path"])
			taskBase = false
		}
		at := index("$.projects", i)
		dir, e := resolvePath(access, base, path, true, false, at+".location.path")
		if e != nil {
			return e
		}
		tb := dir
		if !taskBase {
			tb = base
		}
		if _, e := resolvePath(access, tb, str(obj(p["task_system"])["path"]), false, false, at+".task_system.path"); e != nil {
			return e
		}
	}
	for i, v := range list(card["members"]) {
		m := obj(v)
		for _, key := range []string{"clone", "git_dir"} {
			if p, ok := m[key]; ok {
				if _, e := resolvePath(access, base, str(p), true, true, index("$.members", i)+"."+key); e != nil {
					return e
				}
			}
		}
	}
	return nil
}
func localPath(p string) bool {
	return p != "" && !filepath.IsAbs(p) && p != ".." && !strings.HasPrefix(filepath.Clean(p), ".."+string(filepath.Separator))
}
func beneath(root, p string) bool { rel, e := filepath.Rel(root, p); return e == nil && localPath(rel) }
func resolvePath(access LayoutFS, base, path string, directory, missingOK bool, at string) (string, *Issue) {
	return resolveBounded(access, ".", base, path, directory, missingOK, at)
}
func resolveBounded(access LayoutFS, boundary, base, path string, directory, missingOK bool, at string) (string, *Issue) {
	target := filepath.Clean(filepath.Join(base, path))
	if !localPath(target) || !beneath(boundary, target) {
		return "", fail("path_root_escape", at)
	}
	// Examine each in-root link before following its target. No Lstat is issued on
	// an out-of-root target, including an existing target. Rooted access closes races.
	pending := strings.Split(target, string(filepath.Separator))
	current := "."
	links := 0
	for len(pending) > 0 {
		part := pending[0]
		pending = pending[1:]
		if part == "." {
			continue
		}
		candidate := filepath.Join(current, part)
		if !localPath(candidate) {
			return "", fail("path_root_escape", at)
		}
		info, err := access.Lstat(candidate)
		if err != nil {
			if missingOK && errors.Is(err, fs.ErrNotExist) {
				return target, nil
			}
			return "", fail("layout_missing", at)
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			links++
			if links > 40 {
				return "", fail("path_symlink_cycle", at)
			}
			link, err := access.Readlink(candidate)
			if err != nil {
				return "", fail("path_symlink_unreadable", at)
			}
			if filepath.IsAbs(link) {
				return "", fail("path_symlink_escape", at)
			}
			dest := filepath.Clean(filepath.Join(current, link))
			if !localPath(dest) || !beneath(boundary, dest) {
				return "", fail("path_symlink_escape", at)
			}
			pending = append(strings.Split(dest, string(filepath.Separator)), pending...)
			current = "."
			continue
		}
		if len(pending) > 0 && !info.IsDir() {
			return "", fail("layout_type", at)
		}
		if len(pending) == 0 && directory && !info.IsDir() {
			return "", fail("layout_type", at)
		}
		if len(pending) == 0 && !directory && !info.Mode().IsRegular() {
			return "", fail("layout_type", at)
		}
		current = candidate
	}
	return current, nil
}
