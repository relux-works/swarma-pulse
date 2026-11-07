package pulse

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func environmentView(t *testing.T, c object) object {
	return environmentViewFor(t, c, "station_beacon_orch")
}

func environmentViewFor(t *testing.T, c object, agent string) object {
	t.Helper()
	raw := validBytes(t, c)
	b, err := projectView(raw, "sha256:"+digest(raw), "/fixture/card", agent, fixedTime, "environment")
	if err != nil {
		t.Fatal(err)
	}
	var v object
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func completeExtension() object {
	return object{"schema": "pulse-member-environment/v1", "a2a_config_profile": "Demo-config_1", "working_root": "../unopened/project", "network": object{"mode": "proxy", "ref": "proxy-main"}}
}

// Complete declarations preserve exact pins and explicit bases without claiming
// installation, launch permission, live state, or reading declared target paths.
func TestEnvironmentCompleteShape(t *testing.T) {
	c := fixture(t)
	m := obj(arr(c["members"])[1])
	m["pulse"] = completeExtension()
	v := environmentView(t, c)
	if len(v) != 17 || v["schema"] != "pulse-view/v2" || v["mode"] != "local-unsigned" || v["known_limit"] != knownLimit || !reflect.DeepEqual(v["query"], object{"view": "environment"}) {
		t.Fatal(v)
	}
	s := obj(v["self"])
	e := obj(s["environment"])
	if len(s) != 11 || s["level"] != "project" || s["project"] != "aurora/beacon" || len(e) != 8 || e["schema"] != "pulse-environment/v1" || e["state"] != "complete" || !reflect.DeepEqual(e["missing"], []any{}) {
		t.Fatal(s)
	}
	profile := obj(e["profile"])
	if !reflect.DeepEqual(profile, object{"key": "control", "source": object{"source": "aurora", "base": "profiles_root", "profiles_root": "../profiles", "path": "control"}, "requirement": obj(s["profile"])["requirement"], "selected": obj(s["profile"])["selected"]}) {
		t.Fatal(profile)
	}
	if !reflect.DeepEqual(e["runtime"], m["runtime"]) || !reflect.DeepEqual(e["working_root"], object{"source": "aurora", "base": "card", "path": "../unopened/project", "origin": "pulse.working_root"}) || !reflect.DeepEqual(e["network"], object{"mode": "proxy", "ref": "proxy-main"}) {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(e["a2a"], object{"agent": "station_beacon_orch", "resolved_name": "@station_beacon_orch:relux.works", "binding": "legacy", "operator": "fixture", "host": "station", "connection": "legacy-relux", "server_name": "relux.works", "config_profile": "Demo-config_1"}) {
		t.Fatal(e["a2a"])
	}
	for _, key := range []string{"peers", "direct_reports", "descendants", "superiors", "products", "boundaries"} {
		if !reflect.DeepEqual(v[key], []any{}) {
			t.Fatal(key, v[key])
		}
	}
	if len(arr(v["projects"])) != 1 || v["hierarchy_status"] != "self-only; hierarchy not disclosed" {
		t.Fatal(v)
	}
	raw := validBytes(t, c)
	if !reflect.DeepEqual(v["sources"], []any{object{"id": "aurora", "schema": "product-card/v3", "revision": "sha256:" + digest(raw), "sha256": digest(raw), "card_root": "/fixture/card", "freshness": "unknown", "observed_at": nil}}) {
		t.Fatal(v["sources"])
	}
	for _, observation := range obj(v["observations"]) {
		if !reflect.DeepEqual(observation, object{"state": "unknown"}) {
			t.Fatal(observation)
		}
	}
}

// Missing optional launch facts are successful incomplete views; root precedence
// is extension, clone, project, then null. Optional effort has no guessed default.
func TestEnvironmentMissingAndRootPrecedence(t *testing.T) {
	for _, tc := range []struct{ name, origin, path string }{
		{"extension", "pulse.working_root", "../unopened/project"},
		{"clone", "member.clone", "../beacon"},
		{"project", "project.path", "../beacon"},
		{"none", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := fixture(t)
			m := obj(arr(c["members"])[1])
			agent := "station_beacon_orch"
			if tc.name == "none" {
				m = obj(arr(c["members"])[0])
				agent = "station_aurora_keeper"
			}
			delete(obj(m["runtime"]), "effort")
			if tc.name == "extension" {
				m["pulse"] = object{"schema": "pulse-member-environment/v1", "working_root": tc.path}
			}
			if tc.name == "project" || tc.name == "none" {
				delete(m, "clone")
			}
			e := obj(obj(environmentViewFor(t, c, agent)["self"])["environment"])
			missing := []any{"a2a.config_profile", "network"}
			if tc.name == "none" {
				missing = append(missing, "working_root")
				if e["working_root"] != nil {
					t.Fatal(e)
				}
			} else if !reflect.DeepEqual(e["working_root"], object{"source": "aurora", "base": "card", "path": tc.path, "origin": tc.origin}) {
				t.Fatal(e)
			}
			if e["state"] != "incomplete" || e["network"] != nil || obj(e["a2a"])["config_profile"] != nil || obj(e["runtime"])["effort"] != nil || !reflect.DeepEqual(e["missing"], missing) {
				t.Fatal(e)
			}
		})
	}
}

// The two legitimate profile source forms and affirmative network branches pass;
// profile/working locators remain data even when their target does not exist.
func TestEnvironmentRegistryAndDirect(t *testing.T) {
	c := fixture(t)
	p := obj(obj(c["profiles"])["control"])
	delete(p, "path")
	p["registry"] = object{"repo": "github.com/example/profiles", "directory": "roles/control"}
	m := obj(arr(c["members"])[1])
	m["pulse"] = completeExtension()
	obj(m["pulse"])["network"] = object{"mode": "direct"}
	e := obj(obj(environmentView(t, c)["self"])["environment"])
	if e["state"] != "complete" || !reflect.DeepEqual(e["network"], object{"mode": "direct"}) || !reflect.DeepEqual(obj(e["profile"])["source"], object{"source": "aurora", "base": "registry", "repo": "github.com/example/profiles", "directory": "roles/control"}) {
		t.Fatal(e)
	}
}

// New views reject URI/drive/control-shaped inherited locators that the legacy
// reader permits, while preserving legitimate sibling and relative locators.
func TestEnvironmentInheritedLocatorGrammar(t *testing.T) {
	for _, field := range []string{"clone", "profile.path", "profiles_root", "registry.directory"} {
		for _, path := range []string{"../unopened", "nested/path", "https:remote", "C:drive", "path\n", `nested\path`, `path\`} {
			c := fixture(t)
			profile := obj(obj(c["profiles"])["control"])
			switch field {
			case "clone":
				obj(arr(c["members"])[1])["clone"] = path
			case "profile.path":
				profile["path"] = path
			case "profiles_root":
				c["profiles_root"] = path
			case "registry.directory":
				delete(profile, "path")
				profile["registry"] = object{"repo": "github.com/example/profiles", "directory": path}
			}
			raw := validBytes(t, c)
			_ = view(t, raw, "station_beacon_orch")
			_, err := projectView(raw, "sha256:"+digest(raw), "/fixture/card", "station_beacon_orch", fixedTime, "environment")
			if strings.HasPrefix(path, "../") || path == "nested/path" {
				if err != nil {
					t.Fatal(field, path, err)
				}
			} else {
				wantRule(t, err, "environment_invalid")
			}
		}
	}
}

// Each invalid extension refuses specifically, with no output or echoed value.
// The adjacent legacy call still ignores it and retains its previous v1 bytes
// except the unavoidable source digest, which covers the changed input bytes.
func TestEnvironmentExtensionRefusalsAndLegacy(t *testing.T) {
	bad := []any{nil, "invalid", []any{}, object{}, object{"schema": "pulse-member-environment/v2"}}
	for _, field := range []string{"schema", "a2a_config_profile", "working_root", "network"} {
		p := completeExtension()
		p[field] = 42
		bad = append(bad, p)
	}
	for _, selector := range []string{"", "../config", "https://proxy.invalid", "run command", "a;b", strings.Repeat("a", 129), "_leading", "name\n"} {
		p := completeExtension()
		p["a2a_config_profile"] = selector
		bad = append(bad, p)
		p = completeExtension()
		p["network"] = object{"mode": "proxy", "ref": selector}
		bad = append(bad, p)
	}
	for _, path := range []string{"", "/absolute", "~home", "https:relative", "C:drive", "\\\\server\\share", "path\x00", "path\n", `nested\path`, `path\`} {
		p := completeExtension()
		p["working_root"] = path
		bad = append(bad, p)
	}
	for _, network := range []any{nil, false, object{}, object{"mode": "proxy"}, object{"mode": "proxy", "ref": "good", "extra": true}, object{"mode": "direct", "ref": "extra"}, object{"mode": "automatic"}, object{"mode": "direct", "extra": true}} {
		p := completeExtension()
		p["network"] = network
		bad = append(bad, p)
	}
	p := completeExtension()
	p["extra"] = "forbidden"
	bad = append(bad, p)
	baseline := view(t, validBytes(t, fixture(t)), "station_beacon_orch")
	delete(obj(baseline["source"]), "revision")
	delete(obj(baseline["source"]), "sha256")
	for i, extension := range bad {
		c := fixture(t)
		obj(arr(c["members"])[1])["pulse"] = extension
		raw := validBytes(t, c)
		b, err := projectView(raw, "sha256:"+digest(raw), "/fixture/card", "station_beacon_orch", fixedTime, "environment")
		wantRule(t, err, "environment_invalid")
		if len(b) != 0 {
			t.Fatalf("case %d emitted output", i)
		}
		legacy := view(t, raw, "station_beacon_orch")
		delete(obj(legacy["source"]), "revision")
		delete(obj(legacy["source"]), "sha256")
		if !reflect.DeepEqual(legacy, baseline) {
			t.Fatalf("case %d changed legacy projection", i)
		}
	}
}

// Validation covers the entire supplied graph, but valid other-member extension
// values never escape into self's environment or any related-member collection.
func TestEnvironmentWholeSourceAndPrivacy(t *testing.T) {
	c := fixture(t)
	raw := validBytes(t, c)
	_, err := projectView(raw, "sha256:"+strings.Repeat("0", 64), "/fixture/card", "station_beacon_orch", fixedTime, "environment")
	wantRule(t, err, "source_digest")
	_, err = projectView(raw, "sha256:"+digest(raw), "/fixture/card", "@station_beacon_orch:relux.works", fixedTime, "environment")
	wantRule(t, err, "caller_unknown")
	other := obj(arr(c["members"])[2])
	other["pulse"] = completeExtension()
	obj(other["pulse"])["a2a_config_profile"] = "other-member-marker"
	obj(other["pulse"])["working_root"] = "../other-member-marker"
	v := environmentView(t, c)
	if strings.Contains(string(encode(t, v)), "other-member-marker") {
		t.Fatal("other member environment leaked")
	}
	obj(other["pulse"])["extra"] = "forbidden"
	raw = validBytes(t, c)
	_, err = projectView(raw, "sha256:"+digest(raw), "/fixture/card", "station_beacon_orch", fixedTime, "environment")
	wantRule(t, err, "environment_invalid")
	delete(other, "pulse")
	other["parent"] = other["id"]
	raw = encode(t, c)
	_, err = projectView(raw, "sha256:"+digest(raw), "/fixture/card", "station_beacon_orch", fixedTime, "environment")
	wantRule(t, err, "card_invalid")
}

// The real CLI accepts only the shipped environment selector before file access;
// positive and intended refusals preserve source bytes, mode, inode and mtime.
func TestEnvironmentCLIAndPreservation(t *testing.T) {
	c := fixture(t)
	obj(arr(c["members"])[1])["pulse"] = completeExtension()
	raw := validBytes(t, c)
	p := writeFixture(t, ownDirectory(t), raw)
	before, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("A2A_AGENT", "station_beacon_orch")
	args := []string{"me", "--local-unsigned", "--view", "environment", "--card", p, "--revision", "sha256:" + digest(raw), "--format", "json"}
	var out, diagnostics bytes.Buffer
	if code := Run(args, &out, &diagnostics); code != 0 || diagnostics.Len() != 0 || !json.Valid(out.Bytes()) {
		t.Fatal(code, out.String(), diagnostics.String())
	}
	bad := [][]string{
		{"me", "--view", "environment", "--card", "/absent", "--revision", "sha256:" + digest(raw), "--format", "json"},
		append(append([]string{}, args...), "--view=environment"),
		{"me", "--local-unsigned", "--view"},
	}
	for _, selector := range []string{"discovery", "unknown", ""} {
		bad = append(bad, []string{"me", "--local-unsigned", "--view=" + selector, "--card", "/absent", "--revision", "sha256:" + digest(raw), "--format", "json"})
	}
	for _, flag := range []string{"--sources", "--sources-revision", "--root", "--member"} {
		bad = append(bad, append(append([]string{}, args...), flag, "/absent"))
	}
	for _, args := range bad {
		out.Reset()
		diagnostics.Reset()
		if code := Run(args, &out, &diagnostics); code != 2 || out.Len() != 0 || diagnostics.String() != "RULE usage: $\n" {
			t.Fatal(code, out.String(), diagnostics.String())
		}
	}
	s, err := parseSelectors(args)
	if err != nil {
		t.Fatal(err)
	}
	s.revision = "sha256:" + strings.Repeat("0", 64)
	out.Reset()
	diagnostics.Reset()
	if code := runSelected(s, os.Getuid(), os.Geteuid(), "station_beacon_orch", fixedTime, &out, &diagnostics); code != 1 || out.Len() != 0 || diagnostics.String() != "RULE source_digest: $\n" {
		t.Fatal(code, out.String(), diagnostics.String())
	}
	after, err := os.Stat(p)
	if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() || before.Size() != after.Size() || before.ModTime() != after.ModTime() {
		t.Fatal("source metadata changed", err)
	}
	bytesAfter, err := os.ReadFile(p)
	if err != nil || !bytes.Equal(raw, bytesAfter) {
		t.Fatal("source bytes changed", err)
	}
}

type failingOutput struct{}

func (failingOutput) Write([]byte) (int, error) { return 0, errors.New("transport unavailable") }

// Malformed extension diagnostics remain safe at the CLI and preserve their
// source; a real output transport error is reported as output_write.
func TestEnvironmentCLIExtensionAndTransportRefusals(t *testing.T) {
	c := fixture(t)
	m := obj(arr(c["members"])[1])
	m["pulse"] = completeExtension()
	raw := validBytes(t, c)
	p := writeFixture(t, ownDirectory(t), raw)
	s := selectors{card: p, revision: "sha256:" + digest(raw), view: "environment"}
	var diagnostics bytes.Buffer
	if code := runSelected(s, os.Getuid(), os.Geteuid(), "station_beacon_orch", fixedTime, failingOutput{}, &diagnostics); code != 1 || diagnostics.String() != "RULE output_write: $\n" {
		t.Fatal(code, diagnostics.String())
	}
	obj(m["pulse"])["network"] = object{"mode": "proxy", "ref": "https://forbidden.invalid"}
	raw = validBytes(t, c)
	badPath := writeFixture(t, ownDirectory(t), raw)
	before, err := os.Stat(badPath)
	if err != nil {
		t.Fatal(err)
	}
	s.card, s.revision = badPath, "sha256:"+digest(raw)
	var out bytes.Buffer
	diagnostics.Reset()
	if code := runSelected(s, os.Getuid(), os.Geteuid(), "station_beacon_orch", fixedTime, &out, &diagnostics); code != 1 || out.Len() != 0 || diagnostics.String() != "RULE environment_invalid: $\n" {
		t.Fatal(code, out.String(), diagnostics.String())
	}
	after, err := os.Stat(badPath)
	if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("invalid source metadata changed", err)
	}
	got, err := os.ReadFile(badPath)
	if err != nil || !bytes.Equal(got, raw) {
		t.Fatal("invalid source bytes changed", err)
	}
}
