package pulse

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/relux-works/swarma-pulse/internal/cardv4"
)

const maxAdditionalSources = 32
const maxSourceBytes = 16 << 20
const maxUniverseMembers = 10000
const maxReportingHops = 64

type sourceSelection struct{ id, schema, path, revision string }

func validRevision(s string) bool {
	if len(s) != 71 || !strings.HasPrefix(s, "sha256:") {
		return false
	}
	for _, c := range s[7:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func cleanAbsolute(s string) bool {
	return filepath.IsAbs(s) && filepath.Clean(s) == s && utf8.ValidString(s) && !strings.ContainsFunc(s, unicode.IsControl)
}
func insideReadRoot(root, path string) error {
	if !cleanAbsolute(root) || !cleanAbsolute(path) || path == root {
		return refusal("input_path")
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return refusal("input_path")
	}
	return nil
}
func sourcePath(root, base string, v any) (string, error) {
	if !declaredRelative(v) {
		return "", refusal("input_path")
	}
	path := filepath.Clean(filepath.Join(base, str(v)))
	if err := insideReadRoot(root, path); err != nil {
		return "", err
	}
	return path, nil
}

// The manifest admits only pinned, explicit JSON source coordinates. Parsing
// reuses the bounded duplicate-key decoder; it performs no filesystem access.
func parseSourceManifest(raw []byte, revision, primary, root string) ([]sourceSelection, error) {
	if revision != "sha256:"+digest(raw) {
		return nil, refusal("source_digest")
	}
	if len(raw) > cardv4.MaxBytes {
		return nil, refusal("input_limit")
	}
	if !json.Valid(raw) {
		return nil, refusal("source_manifest_invalid")
	}
	m, issue := cardv4.Parse(raw, 4)
	if issue != nil || m["schema"] != "pulse-sources/v1" || !onlyKeys(m, "schema", "cards") || len(m) != 2 {
		return nil, refusal("source_manifest_invalid")
	}
	entries, ok := m["cards"].([]any)
	if !ok {
		return nil, refusal("source_manifest_invalid")
	}
	if len(entries) > maxAdditionalSources {
		return nil, refusal("input_limit")
	}
	ids, paths := map[string]bool{}, map[string]bool{primary: true}
	items := []sourceSelection{}
	for _, v := range entries {
		e := obj(v)
		if len(e) != 4 || !onlyKeys(e, "id", "schema", "path", "revision") || !sourceID(e["id"]) || !validRevision(str(e["revision"])) || (e["schema"] != "company-card/v1" && e["schema"] != "product-card/v3") {
			return nil, refusal("source_manifest_invalid")
		}
		path, err := sourcePath(root, filepath.Dir(primary), e["path"])
		if err != nil {
			return nil, err
		}
		id := str(e["id"])
		if ids[id] || paths[path] {
			return nil, refusal("source_conflict")
		}
		ids[id], paths[path] = true, true
		items = append(items, sourceSelection{id: id, schema: str(e["schema"]), path: path, revision: str(e["revision"])})
	}
	return items, nil
}
func sourceID(v any) bool { s, ok := v.(string); return ok && s != "" }
