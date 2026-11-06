package pulse

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/relux-works/swarma-pulse/internal/cardv4"
)

var fixedTime = time.Date(2026, 10, 31, 8, 0, 0, 0, time.UTC)

func fixture(t *testing.T) object {
	t.Helper()
	raw, err := os.ReadFile("../../docs/fixtures/v4/full/source-v3.json")
	if err != nil {
		t.Fatal(err)
	}
	var c object
	if err = json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	return c
}
func encode(t *testing.T, c object) []byte {
	t.Helper()
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func validBytes(t *testing.T, c object) []byte {
	t.Helper()
	raw := encode(t, c)
	if issue := cardv4.Validate(raw, cardv4.Options{Version: 3}); issue != nil {
		t.Fatalf("invalid positive premise: %v", issue)
	}
	return raw
}
func view(t *testing.T, raw []byte, agent string) object {
	t.Helper()
	b, err := project(raw, "sha256:"+digest(raw), "/fixture/card", agent, fixedTime)
	if err != nil {
		t.Fatal(err)
	}
	var v object
	if err = json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func wantRule(t *testing.T, err error, rule string) {
	t.Helper()
	if err == nil || err.Error() != "RULE "+rule+": $" {
		t.Fatalf("want %s; got %v", rule, err)
	}
}

// Both declared and undeclared self-projects retain exact legacy identity and
// source coordinates; desired facts never become observed runtime or authority.
func TestSelfOnlyShapes(t *testing.T) {
	raw := validBytes(t, fixture(t))
	for _, tc := range []struct {
		agent, scope, project, state string
		count                        int
	}{
		{"station_aurora_keeper", "product", "", "undeclared", 0},
		{"station_beacon_orch", "project", "aurora/beacon", "declared", 1},
	} {
		t.Run(tc.agent, func(t *testing.T) {
			v := view(t, raw, tc.agent)
			s := obj(v["self"])
			source := obj(v["source"])
			if v["schema"] != "pulse-view/v1" || v["mode"] != "local-unsigned" || v["known_limit"] != knownLimit || s["scope"] != tc.scope || s["project_state"] != tc.state {
				t.Fatal(v)
			}
			if s["resolved_name"] != "@"+tc.agent+":relux.works" || obj(v["caller"])["cryptographically_verified"] != false || obj(v["caller"])["trust"] != "single-user local trust" {
				t.Fatal(v)
			}
			if tc.project == "" {
				if s["project"] != nil {
					t.Fatal(s)
				}
			} else if s["project"] != tc.project {
				t.Fatal(s)
			}
			if len(arr(v["projects"])) != tc.count {
				t.Fatal(v)
			}
			if tc.count == 1 {
				p := obj(arr(v["projects"])[0])
				if p["key"] != tc.project || obj(p["location"])["path"] != "../beacon" || len(p) != 4 {
					t.Fatal(p)
				}
			}
			if source["schema"] != "product-card/v3" || source["revision"] != "sha256:"+digest(raw) || source["sha256"] != digest(raw) || source["card_root"] != "/fixture/card" || source["freshness"] != "unknown" || source["observed_at"] != nil || v["query_observed_at"] != fixedTime.Format(time.RFC3339Nano) {
				t.Fatal(source)
			}
			if len(s) != 9 || len(obj(s["profile"])) != 2 || len(obj(s["execution_binding"])) != 4 {
				t.Fatal("unexpected self fields", s)
			}
			for _, key := range []string{"peers", "direct_reports", "descendants", "superiors"} {
				a, ok := v[key].([]any)
				if !ok || len(a) != 0 {
					t.Fatal(key, v[key])
				}
			}
			for _, value := range obj(v["observations"]) {
				if !reflect.DeepEqual(value, object{"state": "unknown"}) {
					t.Fatal(value)
				}
			}
		})
	}
}

// Near valid controls prove exact context, pin, schema and graph refusals; no
// valid same-UID environment spoof can be distinguished in this unsigned mode.
func TestResolutionAndValidationRefusals(t *testing.T) {
	c := fixture(t)
	raw := validBytes(t, c)
	_ = view(t, raw, "station_aurora_keeper")
	for _, agent := range []string{"", "unknown", "station-aurora-keeper", "@station_aurora_keeper:relux.works", "station_vault_dev", "station_aurora_keeper\n"} {
		_, err := project(raw, "sha256:"+digest(raw), "/fixture", agent, fixedTime)
		rule := "caller_unknown"
		if agent == "" || strings.Contains(agent, "\n") {
			rule = "caller_context"
		}
		wantRule(t, err, rule)
	}
	_, err := project(raw, "sha256:"+strings.Repeat("0", 64), "/fixture", "station_aurora_keeper", fixedTime)
	wantRule(t, err, "source_digest")
	badSchema := fixture(t)
	badSchema["schema"] = "product-card/v4"
	b := encode(t, badSchema)
	_, err = project(b, "sha256:"+digest(b), "/fixture", "station_aurora_keeper", fixedTime)
	wantRule(t, err, "unsupported_schema")
	cycle := fixture(t)
	obj(arr(cycle["members"])[1])["parent"] = "beacon_orch"
	b = encode(t, cycle)
	_, err = project(b, "sha256:"+digest(b), "/fixture", "station_aurora_keeper", fixedTime)
	wantRule(t, err, "card_invalid")
	for _, b = range [][]byte{[]byte("not: [valid"), []byte("schema: product-card/v3\nx: &x [*x]\n")} {
		_, err = project(b, "sha256:"+digest(b), "/fixture", "station_aurora_keeper", fixedTime)
		wantRule(t, err, "card_invalid")
	}
	// Distinct binding prefixes + logical IDs can resolve to one legacy localpart.
	duplicate := fixture(t)
	bindings := obj(duplicate["bindings"])
	bindings["overlap"] = object{"operator": "fixture", "host": "station", "identity_prefix": "station_beacon_", "connection": "legacy-relux"}
	m := obj(arr(duplicate["members"])[2])
	m["id"] = "orch"
	m["binding"] = "overlap"
	obj(arr(duplicate["projects"])[2])["orchestrator"] = "orch"
	obj(arr(duplicate["members"])[3])["parent"] = "orch"
	b = validBytes(t, duplicate)
	_, err = project(b, "sha256:"+digest(b), "/fixture", "station_beacon_orch", fixedTime)
	wantRule(t, err, "caller_ambiguous")
}

// Legacy extensions and bounded aliases remain accepted; hidden records and
// manager/parent edges cannot alter the projection except its mandatory byte pin.
func TestLegacyReaderAndHiddenNoninterference(t *testing.T) {
	c := fixture(t)
	raw := validBytes(t, c)
	baseline := view(t, raw, "station_aurora_keeper")
	c["extension"] = object{"note": "permitted legacy extension"}
	profile := obj(obj(c["profiles"])["control"])
	obj(profile["requirement"])["note"] = object{"other_member": "hidden requirement extension"}
	obj(profile["selected"])["note"] = "hidden selected extension"
	members := arr(c["members"])
	worker := object{}
	for k, v := range obj(members[3]) {
		worker[k] = v
	}
	worker["id"] = "hidden_worker"
	worker["parent"] = "beacon_orch"
	worker["project"] = "beacon"
	worker["clone"] = "../beacon"
	c["members"] = append(members, worker)
	obj(arr(c["projects"])[1])["description"] = "hidden project description"
	obj(members[2])["context_mode"] = "fresh"
	raw = validBytes(t, c)
	changed := view(t, raw, "station_aurora_keeper")
	for _, v := range []object{baseline, changed} {
		delete(obj(v["source"]), "revision")
		delete(obj(v["source"]), "sha256")
	}
	if !reflect.DeepEqual(baseline, changed) {
		t.Fatal("hidden data changed projected facts", baseline, changed)
	}
	// Append alias-bearing unknown fields to the complete valid YAML document.
	// YAML merge at the root supplies the unchanged, complete JSON mapping.
	alias := append([]byte("<<: "), raw...)
	alias = append(alias, []byte("\nlegacy_anchor: &shared {value: exact}\nlegacy_alias: *shared\n")...)
	if issue := cardv4.Validate(alias, cardv4.Options{Version: 3}); issue != nil {
		t.Fatalf("legacy alias premise: %v", issue)
	}
	got := view(t, alias, "station_aurora_keeper")
	delete(obj(got["source"]), "revision")
	delete(obj(got["source"]), "sha256")
	if !reflect.DeepEqual(baseline, got) {
		t.Fatal("legacy alias changed projection")
	}
}

func ownDirectory(t *testing.T) string {
	t.Helper()
	// Resolve aliases only on the fixture allocated by testing (for example,
	// macOS /var -> /private/var), never on an input being tested for refusal.
	p, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0700); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(p)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() || info.Mode().Perm() != 0700 {
		t.Fatalf("fixture directory must be a regular 0700 directory: %v", info.Mode())
	}
	return p
}
func writeFixture(t *testing.T, dir string, raw []byte) string {
	t.Helper()
	p := filepath.Join(dir, "card.yaml")
	if err := os.WriteFile(p, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(p)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		t.Fatalf("fixture file must be regular with mode 0600: %v", info.Mode())
	}
	return p
}

// The helper returns a canonical test-owned path and testing removes it after
// the allocating subtest, including the fixture file it contains.
func TestOwnDirectoryLifecycle(t *testing.T) {
	var dir string
	t.Run("allocate", func(t *testing.T) {
		dir = ownDirectory(t)
		if !filepath.IsAbs(dir) {
			t.Fatalf("fixture path must be absolute: %q", dir)
		}
		canonical, err := filepath.EvalSymlinks(dir)
		if err != nil || canonical != dir {
			t.Fatalf("fixture path must be canonical: %q, %v", canonical, err)
		}
		writeFixture(t, dir, []byte("synthetic fixture"))
	})
	if dir == "" {
		t.Fatal("fixture was not allocated")
	}
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		t.Fatalf("testing must remove the allocated fixture: %v", err)
	}
}

// Exact owner-held regular bytes pass; mode, symlink/ancestor, finite-size,
// FIFO/directory and path failures refuse without modifying the original file.
func TestSafeFileBoundary(t *testing.T) {
	raw := validBytes(t, fixture(t))
	dir := ownDirectory(t)
	p := writeFixture(t, dir, raw)
	b, err := readCard(p, os.Getuid())
	if err != nil || !bytes.Equal(b, raw) {
		t.Fatal(err)
	}
	cases := []struct {
		name  string
		setup func() string
		rule  string
	}{
		{"relative", func() string { return "card.yaml" }, "input_path"},
		{"unclean", func() string { return dir + "/../" + filepath.Base(dir) + "/card.yaml" }, "input_path"},
		{"missing", func() string { return filepath.Join(dir, "absent") }, "input_trust"},
		{"symlink", func() string {
			x := filepath.Join(dir, "link")
			if err := os.Symlink(p, x); err != nil {
				t.Fatal(err)
			}
			return x
		}, "input_trust"},
		{"ancestor-symlink", func() string {
			x := filepath.Join(dir, "alias")
			if err := os.Symlink(dir, x); err != nil {
				t.Fatal(err)
			}
			return filepath.Join(x, "card.yaml")
		}, "input_trust"},
		{"directory", func() string { return dir }, "input_trust"},
		{"fifo", func() string {
			x := filepath.Join(dir, "fifo")
			if err := syscall.Mkfifo(x, 0600); err != nil {
				t.Fatal(err)
			}
			return x
		}, "input_trust"},
		{"oversize", func() string {
			x := filepath.Join(dir, "huge")
			if err := os.WriteFile(x, make([]byte, cardv4.MaxBytes+1), 0600); err != nil {
				t.Fatal(err)
			}
			return x
		}, "input_trust"},
		{"writable", func() string {
			x := filepath.Join(dir, "writable")
			if err := os.WriteFile(x, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(x, 0666); err != nil {
				t.Fatal(err)
			}
			return x
		}, "input_trust"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { _, err := readCard(tc.setup(), os.Getuid()); wantRule(t, err, tc.rule) })
	}
	_, err = readCard(p, os.Getuid()+1)
	wantRule(t, err, "input_trust")
	b, err = os.ReadFile(p)
	if err != nil || !bytes.Equal(b, raw) {
		t.Fatal("input changed", err)
	}
}

// The public CLI requires affirmative mode and exact selectors before any file
// read. Intended query refusals keep stdout empty and leave input bytes intact.
func TestCLIControlsAndEffects(t *testing.T) {
	raw := validBytes(t, fixture(t))
	dir := ownDirectory(t)
	p := writeFixture(t, dir, raw)
	args := []string{"me", "--local-unsigned", "--card", p, "--revision", "sha256:" + digest(raw), "--format", "json"}
	t.Setenv("A2A_AGENT", "station_aurora_keeper")
	var out, diagnostics bytes.Buffer
	if code := Run(args, &out, &diagnostics); code != 0 || diagnostics.Len() != 0 || !json.Valid(out.Bytes()) {
		t.Fatal(code, out.String(), diagnostics.String())
	}
	before, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	invalid := [][]string{nil, {"me"}, append([]string{"me"}, args[2:]...), append(append([]string{}, args...), "--local-unsigned"), append(append([]string{}, args...), "--card", p)}
	for _, flag := range []string{"--member", "--caller-binding", "--signature", "--visibility-policy", "--extra-card", "--config", "--trust", "--clock"} {
		invalid = append(invalid, append(append([]string{}, args...), flag, "forbidden"))
	}
	upper := append([]string{}, args...)
	upper[5] = "sha256:" + strings.Repeat("A", 64)
	invalid = append(invalid, upper)
	for i, bad := range invalid {
		out.Reset()
		diagnostics.Reset()
		if code := Run(bad, &out, &diagnostics); code != 2 || out.Len() != 0 || diagnostics.String() != "RULE usage: $\n" {
			t.Fatalf("case %d: %d %q %q", i, code, out.String(), diagnostics.String())
		}
	}
	s, _ := parseSelectors(args)
	for _, tc := range []struct {
		uid, euid             int
		agent, revision, rule string
	}{
		{os.Getuid(), os.Geteuid() + 1, "station_aurora_keeper", s.revision, "os_principal"},
		{os.Getuid(), os.Geteuid(), "", s.revision, "caller_context"},
		{os.Getuid(), os.Geteuid(), "unknown", s.revision, "caller_unknown"},
		{os.Getuid(), os.Geteuid(), "station_aurora_keeper", "sha256:" + strings.Repeat("0", 64), "source_digest"},
	} {
		out.Reset()
		diagnostics.Reset()
		selected := s
		selected.revision = tc.revision
		if code := runSelected(selected, tc.uid, tc.euid, tc.agent, fixedTime, &out, &diagnostics); code != 1 || out.Len() != 0 || diagnostics.String() != "RULE "+tc.rule+": $\n" {
			t.Fatal(code, out.String(), diagnostics.String())
		}
	}
	after, err := os.ReadDir(dir)
	if err != nil || len(before) != len(after) {
		t.Fatal("unexpected input-directory write", err)
	}
	b, err := os.ReadFile(p)
	if err != nil || !bytes.Equal(b, raw) {
		t.Fatal("input changed", err)
	}
}
