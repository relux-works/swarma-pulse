package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/swarma-pulse/internal/cardtest"
)

const fixtures = "../../docs/fixtures/v4/full"

func mustWrite(t *testing.T, path string, raw []byte) {
	t.Helper()
	if e := os.WriteFile(path, raw, 0600); e != nil {
		t.Fatal(e)
	}
}
func read(t *testing.T, path string) []byte {
	t.Helper()
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	return raw
}
func hash(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func args(m map[string]string) []string {
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	a := []string{}
	for _, k := range keys {
		a = append(a, "--"+k, m[k])
	}
	return a
}
func setup(t *testing.T) (string, map[string]string) {
	t.Helper()
	root := t.TempDir()
	if e := os.Mkdir(filepath.Join(root, "fx"), 0700); e != nil {
		t.Fatal(e)
	}
	entries, e := os.ReadDir(fixtures)
	if e != nil {
		t.Fatal(e)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".json") {
			mustWrite(t, filepath.Join(root, "fx", entry.Name()), read(t, filepath.Join(fixtures, entry.Name())))
		}
	}
	a := map[string]string{"root": root, "pilot": "aurora/beacon_orch", "source-manifest": "fx/manifest.json", "reservations": "fx/expectations.json"}
	for i, name := range []string{"prior", "candidate", "rollback"} {
		file := []string{"converted-v4.json", "pilot-v4.json", "rollback-v4.json"}[i]
		a[name] = "fx/" + file
		a[name+"-sha256"] = hash(read(t, filepath.Join(root, a[name])))
	}
	return root, a
}

// The actual read-only CLI accepts complete ordinary/opaque cutover pairs,
// unchanged candidates and in-root links, with exact receipt and no tree writes.
func TestCLIChangeCheck(t *testing.T) {
	for _, name := range []string{"ordinary", "opaque", "unchanged", "in-root link"} {
		t.Run(name, func(t *testing.T) {
			root, a := setup(t)
			switch name {
			case "opaque":
				a["prior"] = "fx/compat-id-converted-v4.json"
				a["candidate"] = "fx/compat-id-pilot-v4.json"
				a["rollback"] = a["prior"]
				a["pilot"] = "aurora/beacon=orch"
				a["reservations"] = "fx/compat-id-expectations.json"
			case "unchanged":
				a["candidate"] = a["prior"]
			case "in-root link":
				if e := os.Symlink("fx/pilot-v4.json", filepath.Join(root, "link")); e != nil {
					t.Fatal(e)
				}
				a["candidate"] = "link/./"
			}
			for _, k := range []string{"prior", "candidate", "rollback"} {
				a[k+"-sha256"] = hash(read(t, filepath.Join(root, filepath.Clean(a[k]))))
			}
			before := cardtest.Inventory(t, root)
			t.Setenv("CARD_V4_CHECK_CHANGE_FAULT", "1")
			code, out, diag := cardtest.Capture(run, args(a))
			if code != 0 || out != "change-valid\n" || diag != "" {
				t.Fatal(code, diag)
			}
			if !reflect.DeepEqual(before, cardtest.Inventory(t, root)) {
				t.Fatal("checker modified inputs")
			}
		})
	}
}

// Paired complete controls precede each exact refusal. Source/metadata/snapshot
// escaping paths cannot be read through the admitted root; every refusal keeps
// both isolated trees and captured output unchanged and free of raw input values.
func TestCLIChangeRefusals(t *testing.T) {
	for _, tc := range []struct {
		name, rule, location string
		exit                 int
	}{
		{"fault flag", "usage", "$", 2}, {"missing pilot", "usage", "$", 2}, {"wrong pilot", "nonpilot_preservation", "$.members[1]", 1},
		{"false prior pin", "snapshot_digest_mismatch", "$[0]", 1}, {"false candidate pin", "snapshot_digest_mismatch", "$[1]", 1}, {"false rollback pin", "snapshot_digest_mismatch", "$[2]", 1},
		{"missing pin", "usage", "$", 2}, {"missing root", "input_root_unreadable", "$", 2},
		{"snapshot escape", "snapshot_unreadable", "$[1]", 2}, {"snapshot symlink escape", "snapshot_unreadable", "$[1]", 2}, {"snapshot directory", "snapshot_unreadable", "$[1]", 2},
		{"manifest escape", "source_manifest_invalid", "$", 2}, {"source escape", "source_manifest_invalid", "$", 2}, {"source symlink escape", "source_unreadable", "$", 2},
		{"reservations escape", "reservations_invalid", "$", 2}, {"metadata duplicate", "source_manifest_invalid", "$", 2}, {"pin unknown field", "source_manifest_invalid", "$", 2},
		{"source digest", "source_digest_mismatch", "$", 1}, {"source invalid", "source_invalid", "$", 2}, {"missing reservation", "star_reservation_required", "$[0].members[3].identity.star", 1},
		{"malformed snapshot", "parse_document", "$[1]", 2}, {"oversized snapshot", "snapshot_unreadable", "$[1]", 2}, {"rollback formatting", "rollback_bytes_mismatch", "$[2]", 1},
		{"nonpilot change", "nonpilot_preservation", "$.members[2]", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, a := setup(t)
			if code, out, diag := cardtest.Capture(run, args(a)); code != 0 || out != "change-valid\n" || diag != "" {
				t.Fatal("positive control", code, diag)
			}
			outside := t.TempDir()
			mustWrite(t, filepath.Join(outside, "input"), read(t, filepath.Join(root, a["candidate"])))
			manifestPath := filepath.Join(root, "fx/manifest.json")
			var m map[string]any
			if e := json.Unmarshal(read(t, manifestPath), &m); e != nil {
				t.Fatal(e)
			}
			pins := m["source_pins"].([]any)
			pin := pins[0].(map[string]any)
			writeManifest := func() {
				raw, e := json.Marshal(m)
				if e != nil {
					t.Fatal(e)
				}
				mustWrite(t, manifestPath, raw)
			}
			extra := []string{}
			switch tc.name {
			case "fault flag":
				extra = []string{"--fault-after-member=2"}
			case "missing pilot":
				delete(a, "pilot")
			case "wrong pilot":
				a["pilot"] = "aurora/storage_orch"
			case "false prior pin", "false candidate pin", "false rollback pin":
				a[strings.TrimPrefix(strings.TrimSuffix(tc.name, " pin"), "false ")+"-sha256"] = strings.Repeat("a", 64)
			case "missing pin":
				delete(a, "rollback-sha256")
			case "missing root":
				a["root"] = filepath.Join(root, "missing")
			case "snapshot escape":
				a["candidate"] = "../SYNTHETIC-CANARY"
			case "snapshot symlink escape":
				if e := os.Symlink(outside, filepath.Join(root, "escape")); e != nil {
					t.Fatal(e)
				}
				a["candidate"] = "escape//input"
			case "snapshot directory":
				a["candidate"] = "fx/"
			case "manifest escape":
				a["source-manifest"] = "../SYNTHETIC-CANARY"
			case "source escape":
				pin["file"] = "../SYNTHETIC-CANARY"
				writeManifest()
			case "source symlink escape":
				if e := os.Symlink(filepath.Join(outside, "input"), filepath.Join(root, "fx/escape")); e != nil {
					t.Fatal(e)
				}
				pin["file"] = "escape"
				writeManifest()
			case "reservations escape":
				a["reservations"] = "../SYNTHETIC-CANARY"
			case "metadata duplicate":
				mustWrite(t, manifestPath, []byte(`{"source_pins":[],"source_pins":[]}`))
			case "pin unknown field":
				pin["unknown"] = "SYNTHETIC-CANARY"
				writeManifest()
			case "source digest":
				pin["sha256"] = strings.Repeat("a", 64)
				writeManifest()
			case "source invalid":
				raw := []byte("SYNTHETIC-CANARY: [")
				mustWrite(t, filepath.Join(root, "fx/source-v3.json"), raw)
				pin["sha256"] = hash(raw)
				writeManifest()
			case "missing reservation":
				mustWrite(t, filepath.Join(root, a["reservations"]), []byte(`{"reviewed_reservation_premises":[]}`))
			case "malformed snapshot", "oversized snapshot", "rollback formatting":
				k, raw := "candidate", []byte("SYNTHETIC-CANARY: [")
				if tc.name == "oversized snapshot" {
					raw = []byte(strings.Repeat(" ", 1048577))
				}
				if tc.name == "rollback formatting" {
					k = "rollback"
					raw = append(read(t, filepath.Join(root, a[k])), '\n')
				}
				mustWrite(t, filepath.Join(root, a[k]), raw)
				a[k+"-sha256"] = hash(raw)
			case "nonpilot change":
				a["candidate"] = "fx/mixed-reject-nonpilot-state.json"
				a["candidate-sha256"] = hash(read(t, filepath.Join(root, a["candidate"])))
			}
			before, external := cardtest.Inventory(t, root), cardtest.Inventory(t, outside)
			code, out, diag := cardtest.Capture(run, append(args(a), extra...))
			if code != tc.exit || out != "" || diag != "RULE "+tc.rule+": "+tc.location+"\n" {
				t.Fatalf("exit %d, stdout %q, diagnostic %q", code, out, diag)
			}
			if !reflect.DeepEqual(before, cardtest.Inventory(t, root)) || !reflect.DeepEqual(external, cardtest.Inventory(t, outside)) {
				t.Fatal("checker refusal changed tree")
			}
		})
	}
}
