package pulse

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// Equal localparts retain their declared homeservers; repeating the same full
// identity preserves equality. This checks output data, not a consumer's dedup.
func TestFederatedNamesPreserveHomeserver(t *testing.T) {
	const agent = "station_beacon_orch"
	var names []string
	for _, tc := range []struct{ name, server string }{
		{"alpha", "alpha.example"},
		{"beta", "beta.example"},
		{"alpha-repeat", "alpha.example"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := fixture(t)
			connection := obj(obj(card["connections"])["legacy-relux"])
			connection["server_name"] = tc.server
			connection["homeserver"] = "https://transport.example"
			v := view(t, validBytes(t, card), agent)
			self, caller := obj(v["self"]), obj(v["caller"])
			want := "@" + agent + ":" + tc.server
			if self["resolved_name"] != want {
				t.Fatalf("full Matrix identity: want %q, got %v", want, self["resolved_name"])
			}
			if caller["agent"] != agent || caller["cryptographically_verified"] != false || caller["trust"] != "single-user local trust" || v["known_limit"] != knownLimit {
				t.Fatal("qualified name changed unverified caller semantics", caller)
			}
			if self["key"] != "aurora/beacon_orch" || v["member"] != self["key"] {
				t.Fatal("Matrix name replaced logical member identity", self)
			}
			names = append(names, self["resolved_name"].(string))
		})
	}
	if len(names) != 3 || names[0] == names[1] || names[0] != names[2] {
		t.Fatal("full Matrix identity equality lost the homeserver", names)
	}
}

// A unique local context succeeds before a valid two-domain collision refuses
// at the public CLI, without output or changes to its file/directory fixture.
func TestFederatedLocalContextAmbiguity(t *testing.T) {
	const agent = "station_beacon_orch"
	card := fixture(t)
	obj(obj(card["connections"])["legacy-relux"])["server_name"] = "alpha.example"
	raw := validBytes(t, card)
	dir := ownDirectory(t)
	path := writeFixture(t, dir, raw)
	args := []string{"me", "--local-unsigned", "--card", path, "--revision", "sha256:" + digest(raw), "--format", "json"}
	t.Setenv("A2A_AGENT", agent)
	var out, diagnostics bytes.Buffer
	if code := Run(args, &out, &diagnostics); code != 0 || diagnostics.Len() != 0 {
		t.Fatal("unique-domain positive control", code, out.String(), diagnostics.String())
	}
	var v object
	if err := json.Unmarshal(out.Bytes(), &v); err != nil || obj(v["self"])["resolved_name"] != "@"+agent+":alpha.example" {
		t.Fatal("unique-domain full Matrix identity", err, out.String())
	}

	obj(card["connections"])["federated-beta"] = object{"homeserver": "https://transport.example", "server_name": "beta.example"}
	obj(card["bindings"])["overlap"] = object{"operator": "fixture", "host": "station", "identity_prefix": "station_beacon_", "connection": "federated-beta"}
	member := obj(arr(card["members"])[2])
	member["id"], member["binding"] = "orch", "overlap"
	obj(member["a2a"])["connection"] = "federated-beta"
	obj(arr(card["projects"])[2])["orchestrator"] = "orch"
	obj(arr(card["members"])[3])["parent"] = "orch"
	raw = validBytes(t, card)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	entriesBefore, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	args[5] = "sha256:" + digest(raw)
	out.Reset()
	diagnostics.Reset()
	if code := Run(args, &out, &diagnostics); code != 1 || out.Len() != 0 || diagnostics.String() != "RULE caller_ambiguous: $\n" {
		t.Fatal("two-domain context must refuse without disclosure", code, out.String(), diagnostics.String())
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || before.Mode() != after.Mode() || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("ambiguous context changed fixture metadata")
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, raw) {
		t.Fatal("ambiguous context changed fixture bytes", err)
	}
	entriesAfter, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var beforeNames, afterNames []string
	for _, entry := range entriesBefore {
		beforeNames = append(beforeNames, entry.Name())
	}
	for _, entry := range entriesAfter {
		afterNames = append(afterNames, entry.Name())
	}
	if !reflect.DeepEqual(beforeNames, afterNames) {
		t.Fatal("ambiguous context changed fixture directory", beforeNames, afterNames)
	}
}
