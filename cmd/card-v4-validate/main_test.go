package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const fixtureRoot = "../../docs/fixtures/v4/full"

func inventory(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	if e := filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			result[rel] = "dir"
			return nil
		}
		raw, e := os.ReadFile(path)
		sum := sha256.Sum256(raw)
		result[rel] = hex.EncodeToString(sum[:])
		return e
	}); e != nil {
		t.Fatal(e)
	}
	return result
}

// CLI verdicts have exact exit/stdout/diagnostic boundaries. Refusal emits no
// partial card, input value, filename or parser backtrace and changes no bytes.
func TestCLI(t *testing.T) {
	dir := t.TempDir()
	entries, e := os.ReadDir(fixtureRoot)
	if e != nil {
		t.Fatal(e)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, e := os.ReadFile(filepath.Join(fixtureRoot, entry.Name()))
		if e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(filepath.Join(dir, entry.Name()), raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	common := []string{"--source-manifest", filepath.Join(dir, "manifest.json"), "--reservations", filepath.Join(dir, "expectations.json")}
	cases := []struct {
		name            string
		args            []string
		exit            int
		out, diagnostic string
	}{
		{"valid", append(append([]string{}, common...), filepath.Join(dir, "converted-v4.json")), 0, "valid\n", ""},
		{"missing-source", []string{filepath.Join(dir, "converted-v4.json")}, 1, "", "RULE legacy_source_missing: $.members[0].identity.source_ref\n"},
		{"discriminator", []string{"--version", "3", filepath.Join(dir, "converted-v4.json")}, 2, "", "RULE not-a-product-card/v3-document: $.schema\n"},
		{"v3", []string{"--version", "3", filepath.Join(dir, "source-v3.json")}, 0, "valid\n", ""},
		{"usage", []string{"--unknown=SYNTHETIC-CANARY"}, 2, "", "RULE usage: $\n"},
		{"missing-file", []string{filepath.Join(dir, "SYNTHETIC-CANARY")}, 2, "", "RULE input_unreadable: $\n"},
		{"layout-root-required", append(append([]string{}, common...), "--check-layout", filepath.Join(dir, "converted-v4.json")), 2, "", "RULE layout_root_required: $\n"},
	}
	before := inventory(t, dir)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out, diag bytes.Buffer
			exit := run(c.args, &out, &diag)
			if exit != c.exit || out.String() != c.out || diag.String() != c.diagnostic {
				t.Fatalf("exit %d; stdout %q; safe diagnostic %q", exit, out.String(), diag.String())
			}
			if !reflect.DeepEqual(before, inventory(t, dir)) {
				t.Fatal("CLI modified isolated input")
			}
		})
	}
}
func safeOutput(out, diagnostic string) bool {
	return out == "" && diagnostic == "RULE parse_document: $\n"
}

// This literal canary is inert. Its raw key/value never appears in a parse
// refusal; the output instrument detects a deliberately planted partial value.
func TestCapturedOutputControl(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "card")
	if e := os.WriteFile(path, []byte("SYNTHETIC-CANARY: [unterminated"), 0600); e != nil {
		t.Fatal(e)
	}
	var out, diag bytes.Buffer
	if run([]string{path}, &out, &diag) != 2 || !safeOutput(out.String(), diag.String()) {
		t.Fatal("unsafe parse diagnostic")
	}
	if safeOutput("SYNTHETIC-CANARY", diag.String()) {
		t.Fatal("partial-output control undetected")
	}
}

// Malformed envelopes, out-of-root manifest references and unreadable supplied
// sources fail before validation, without leaking their raw path or bytes.
func TestCLIInputBoundaries(t *testing.T) {
	dir := t.TempDir()
	card := filepath.Join(fixtureRoot, "converted-v4.json")
	for _, c := range []struct{ name, body, flag, want string }{
		{"manifest syntax", "[", "--source-manifest", "RULE source_manifest_invalid: $\n"},
		{"manifest escape", `{"source_pins":[{"file":"../outside"}]}`, "--source-manifest", "RULE source_path_escape: $\n"},
		{"manifest missing", `{"source_pins":[{"file":"missing"}]}`, "--source-manifest", "RULE source_unreadable: $\n"},
		{"reservation syntax", "[", "--reservations", "RULE reservations_invalid: $\n"},
		{"large manifest", strings.Repeat(" ", 1048577), "--source-manifest", "RULE source_manifest_invalid: $\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(dir, "input")
			if e := os.WriteFile(path, []byte(c.body), 0600); e != nil {
				t.Fatal(e)
			}
			before := inventory(t, dir)
			var out, diag bytes.Buffer
			if run([]string{c.flag, path, card}, &out, &diag) != 2 || out.Len() != 0 || diag.String() != c.want {
				t.Fatal("input boundary verdict", diag.String())
			}
			if !reflect.DeepEqual(before, inventory(t, dir)) {
				t.Fatal("input changed")
			}
		})
	}
}

// Rooted layout admission checks the isolated canonical host tree, and a missing
// task file fails without writing or treating remote unavailability as absence.
func TestCLIIsolatedLayout(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{"aurora-host", "beacon", "vault", "profiles/control"} {
		if e := os.MkdirAll(filepath.Join(dir, p), 0700); e != nil {
			t.Fatal(e)
		}
	}
	for _, p := range []string{"aurora-host", "beacon", "vault"} {
		if e := os.WriteFile(filepath.Join(dir, p, "tasks.md"), []byte("fixture\n"), 0600); e != nil {
			t.Fatal(e)
		}
	}
	args := []string{"--check-layout", "--root", dir, "--card-directory", "aurora-host", "--source-manifest", filepath.Join(fixtureRoot, "manifest.json"), "--reservations", filepath.Join(fixtureRoot, "expectations.json"), filepath.Join(fixtureRoot, "converted-v4.json")}
	var out, diag bytes.Buffer
	before := inventory(t, dir)
	if run(args, &out, &diag) != 0 || out.String() != "valid\n" || diag.Len() != 0 {
		t.Fatal("layout positive", diag.String())
	}
	if !reflect.DeepEqual(before, inventory(t, dir)) {
		t.Fatal("layout write")
	}
	if e := os.Remove(filepath.Join(dir, "beacon/tasks.md")); e != nil {
		t.Fatal(e)
	}
	out.Reset()
	diag.Reset()
	if run(args, &out, &diag) != 1 || out.Len() != 0 || diag.String() != "RULE layout_missing: $.projects[1].task_system.path\n" {
		t.Fatal("layout negative", diag.String())
	}
}

// The production v3 CLI, rather than a test-side mapper, emits exact FORMAT R7
// diagnostics for a missing task and a missing profile root after valid controls.
func TestV3CLIProductionLayoutRules(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{"legacy-card", "aurora-host", "beacon", "vault", "profiles/control"} {
		if e := os.MkdirAll(filepath.Join(dir, p), 0700); e != nil {
			t.Fatal(e)
		}
	}
	for _, p := range []string{"aurora-host", "beacon", "vault"} {
		if e := os.WriteFile(filepath.Join(dir, p, "tasks.md"), []byte("fixture\n"), 0600); e != nil {
			t.Fatal(e)
		}
	}
	args := []string{"--version", "3", "--check-layout", "--root", dir, "--card-directory", "legacy-card", filepath.Join(fixtureRoot, "source-v3.json")}
	check := func(wantExit int, wantOut, wantDiagnostic string) {
		t.Helper()
		var out, diag bytes.Buffer
		before := inventory(t, dir)
		if exit := run(args, &out, &diag); exit != wantExit || out.String() != wantOut || diag.String() != wantDiagnostic {
			t.Fatalf("exit %d stdout %q safe diagnostic %q", exit, out.String(), diag.String())
		}
		if !reflect.DeepEqual(before, inventory(t, dir)) {
			t.Fatal("v3 CLI changed tree")
		}
	}
	check(0, "valid\n", "")
	if e := os.Remove(filepath.Join(dir, "beacon/tasks.md")); e != nil {
		t.Fatal(e)
	}
	check(1, "", "RULE R7: $.projects[1].task_system.path\n")
	if e := os.WriteFile(filepath.Join(dir, "beacon/tasks.md"), []byte("fixture\n"), 0600); e != nil {
		t.Fatal(e)
	}
	check(0, "valid\n", "")
	if e := os.Rename(filepath.Join(dir, "profiles"), filepath.Join(dir, "profiles-held")); e != nil {
		t.Fatal(e)
	}
	check(1, "", "RULE R7: $.profiles_root\n")
}

// The real CLI rejects the reviewed parser/repository/origin witnesses against
// the same complete positive card and source/reservation premises.
func TestCLIReviewRegressionWitnesses(t *testing.T) {
	dir := t.TempDir()
	baseline, e := os.ReadFile(filepath.Join(fixtureRoot, "converted-v4.json"))
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "card.json")
	args := []string{"--source-manifest", filepath.Join(fixtureRoot, "manifest.json"), "--reservations", filepath.Join(fixtureRoot, "expectations.json"), path}
	check := func(raw []byte, wantExit int, wantOut, wantDiagnostic string) {
		t.Helper()
		if e := os.WriteFile(path, raw, 0600); e != nil {
			t.Fatal(e)
		}
		before := inventory(t, dir)
		var out, diag bytes.Buffer
		if exit := run(args, &out, &diag); exit != wantExit || out.String() != wantOut || diag.String() != wantDiagnostic {
			t.Fatalf("exit %d stdout %q diagnostic %q", exit, out.String(), diag.String())
		}
		if !reflect.DeepEqual(before, inventory(t, dir)) {
			t.Fatal("CLI mutated witness")
		}
	}
	check(baseline, 0, "valid\n", "")
	anchored := bytes.Replace(baseline, []byte(`"schema":`), []byte(`&key "schema":`), 1)
	if bytes.Equal(anchored, baseline) {
		t.Fatal("anchor plant not applied")
	}
	check(anchored, 2, "", "RULE yaml_alias: $[0]\n")
	for _, space := range []string{"\v", "\u00a0"} {
		var c map[string]any
		if json.Unmarshal(baseline, &c) != nil {
			t.Fatal("witness decode")
		}
		c["projects"].([]any)[0].(map[string]any)["repo"] = "git.example/owner/na" + space + "me"
		raw, e := json.Marshal(c)
		if e != nil {
			t.Fatal(e)
		}
		check(raw, 1, "", "RULE repository_format: $.projects[0].repo\n")
	}
	for _, fragment := range []string{"#", "#fragment"} {
		var c map[string]any
		if json.Unmarshal(baseline, &c) != nil {
			t.Fatal("witness decode")
		}
		c["connections"].(map[string]any)["target-gr"].(map[string]any)["homeserver"] = "https://gr.relux.works/" + fragment
		raw, e := json.Marshal(c)
		if e != nil {
			t.Fatal(e)
		}
		check(raw, 1, "", "RULE https_origin: $.connections[1].homeserver\n")
	}
}

// The CLI accepts the matching service source, then refuses a consistently
// rehashed foreign-server source/target without output or input mutation.
// Restoring only the target retains the distinct source/target mismatch refusal.
func TestCLILegacyServiceServerAgreement(t *testing.T) {
	dir := t.TempDir()
	entries, e := os.ReadDir(fixtureRoot)
	if e != nil {
		t.Fatal(e)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, e := os.ReadFile(filepath.Join(fixtureRoot, entry.Name()))
		if e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(filepath.Join(dir, entry.Name()), raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	read := func(file string) map[string]any {
		t.Helper()
		raw, e := os.ReadFile(filepath.Join(dir, file))
		var d map[string]any
		if e != nil || json.Unmarshal(raw, &d) != nil {
			t.Fatal("isolated fixture decode")
		}
		return d
	}
	write := func(file string, d map[string]any) string {
		t.Helper()
		raw, e := json.Marshal(d)
		if e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(filepath.Join(dir, file), raw, 0600); e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(raw)
		return hex.EncodeToString(sum[:])
	}
	args := []string{"--source-manifest", filepath.Join(dir, "manifest.json"), "--reservations", filepath.Join(dir, "expectations.json"), filepath.Join(dir, "services-v4.json")}
	check := func(exit int, stdout, diagnostic string) {
		t.Helper()
		before := inventory(t, dir)
		var out, diag bytes.Buffer
		if got := run(args, &out, &diag); got != exit || out.String() != stdout || diag.String() != diagnostic {
			t.Fatalf("exit %d stdout %q diagnostic %q", got, out.String(), diag.String())
		}
		if !reflect.DeepEqual(before, inventory(t, dir)) {
			t.Fatal("CLI changed service inputs")
		}
	}
	check(0, "valid\n", "")
	source, manifest, card := read("legacy-service-source.json"), read("manifest.json"), read("services-v4.json")
	source["mxid"] = "@station_signal_service:other.example"
	hash := write("legacy-service-source.json", source)
	manifest["files"].(map[string]any)["legacy-service-source.json"] = hash
	patched := false
	for _, v := range manifest["source_pins"].([]any) {
		pin := v.(map[string]any)
		if pin["file"] == "legacy-service-source.json" {
			pin["sha256"] = hash
			patched = true
		}
	}
	if !patched {
		t.Fatal("source pin mutation not applied")
	}
	identity := card["members"].([]any)[4].(map[string]any)["identity"].(map[string]any)
	identity["source_ref"].(map[string]any)["sha256"] = hash
	identity["mxid"] = source["mxid"]
	manifest["files"].(map[string]any)["services-v4.json"] = write("services-v4.json", card)
	write("manifest.json", manifest)
	check(1, "", "RULE legacy_source_connection_mismatch: $.members[4].identity.mxid\n")
	identity["mxid"] = "@station_signal_service:relux.works"
	manifest["files"].(map[string]any)["services-v4.json"] = write("services-v4.json", card)
	write("manifest.json", manifest)
	check(1, "", "RULE legacy_source_mismatch: $.members[4].identity.mxid\n")
}
