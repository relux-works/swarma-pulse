package pulse

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/relux-works/swarma-pulse/internal/cardv4"
)

// readCard accepts one explicitly selected owner-held file, not a directory or
// ambient source. Owner/root can change these paths; same-UID isolation is not promised.
func readCard(path string, uid int) ([]byte, error) { return readCardBounded(path, uid, "input_trust") }

func readCardBounded(path string, uid int, oversizedRule string) ([]byte, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsRune(path, 0) {
		return nil, refusal("input_path")
	}
	// Keep the leading empty component so the filesystem root is checked too.
	parts := strings.Split(path, string(filepath.Separator))
	current := string(filepath.Separator)
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return nil, refusal("input_trust")
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || (stat.Uid != 0 && stat.Uid != uint32(uid)) || info.Mode().Perm()&0022 != 0 {
			return nil, refusal("input_trust")
		}
		if i != len(parts)-1 {
			if !info.IsDir() {
				return nil, refusal("input_trust")
			}
		} else if info.Size() > cardv4.MaxBytes {
			return nil, refusal(oversizedRule)
		} else if !info.Mode().IsRegular() || stat.Uid != uint32(uid) {
			return nil, refusal("input_trust")
		}
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, refusal("input_unreadable")
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	info, err := file.Stat()
	if err == nil && info.Size() > cardv4.MaxBytes {
		return nil, refusal(oversizedRule)
	}
	if err != nil || !info.Mode().IsRegular() {
		return nil, refusal("input_trust")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(uid) || info.Mode().Perm()&0022 != 0 {
		return nil, refusal("input_trust")
	}
	raw, err := io.ReadAll(io.LimitReader(file, cardv4.MaxBytes+1))
	if err != nil || len(raw) > cardv4.MaxBytes {
		return nil, refusal("input_limit")
	}
	return raw, nil
}
