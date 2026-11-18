package pulse

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

var governanceTime = time.Date(2026, 11, 17, 16, 0, 0, 0, time.UTC)

func governanceCard(t *testing.T) object {
	t.Helper()
	raw, err := os.ReadFile("../../docs/fixtures/v5/governance-v5.json")
	if err != nil {
		t.Fatal(err)
	}
	var c object
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	return c
}

func governanceResult(t *testing.T, c object, agent string) object {
	t.Helper()
	raw := encode(t, c)
	before := append([]byte{}, raw...)
	encoded, err := projectView(raw, "sha256:"+digest(raw), "/fixture/card", agent, governanceTime, "governance")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, raw) || !bytes.Equal(before, encode(t, c)) || !bytes.HasSuffix(encoded, []byte("\n")) {
		t.Fatal("source changed or missing output newline")
	}
	var v object
	if err := json.Unmarshal(encoded, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func governanceObjectKeys(t *testing.T, value any, fields string) {
	t.Helper()
	m, ok := value.(map[string]any)
	want := strings.Fields(fields)
	if !ok || len(m) != len(want) {
		t.Fatalf("object shape: want %d fields, got %d", len(want), len(m))
	}
	for _, field := range want {
		if _, present := m[field]; !present {
			t.Fatalf("missing output field %s", field)
		}
	}
}

func governanceClosedShape(t *testing.T, v object, modern bool) {
	t.Helper()
	governanceObjectKeys(t, v, "schema mode product member caller known_limit source query_observed_at self peers direct_reports descendants superiors projects hierarchy_status observations governance")
	governanceObjectKeys(t, v["caller"], "agent cryptographically_verified trust")
	sourceFields := "schema revision sha256 card_root freshness observed_at"
	if modern {
		sourceFields += " validation"
	}
	governanceObjectKeys(t, v["source"], sourceFields)
	governanceObjectKeys(t, v["self"], "key kind role responsibility scope desired_state resolved_name execution_binding profile project project_state")
	self := obj(v["self"])
	governanceObjectKeys(t, self["execution_binding"], "key operator host connection")
	if self["profile"] != nil {
		governanceObjectKeys(t, self["profile"], "requirement selected")
		governanceObjectKeys(t, obj(self["profile"])["requirement"], "context compatible")
		governanceObjectKeys(t, obj(self["profile"])["selected"], "version source_rev lock_sha256")
	}
	for _, value := range arr(v["projects"]) {
		governanceObjectKeys(t, value, "key repo desired_state location")
		location := obj(obj(value)["location"])
		fields := "kind path"
		if location["kind"] == "remote" {
			fields = "kind operator execution_host project_locator"
			if _, present := location["checkout_path"]; present {
				fields += " checkout_path"
			}
		}
		governanceObjectKeys(t, location, fields)
	}
	governanceObjectKeys(t, v["observations"], "runtime registration installed_profile assignment")
	for _, value := range obj(v["observations"]) {
		if !reflect.DeepEqual(value, object{"state": "unknown"}) {
			t.Fatal("observed runtime or authority was fabricated")
		}
	}
	for _, field := range []string{"peers", "direct_reports", "descendants", "superiors"} {
		a, ok := v[field].([]any)
		if !ok || len(a) != 0 {
			t.Fatal("graph disclosure", field)
		}
	}
	governanceObjectKeys(t, v["governance"], "state owners meta_keeper")
	for _, value := range arr(obj(v["governance"])["owners"]) {
		governanceObjectKeys(t, value, "id kind name contacts keys")
		for _, contact := range arr(obj(value)["contacts"]) {
			governanceObjectKeys(t, contact, "kind mxid")
		}
		for _, key := range arr(obj(value)["keys"]) {
			fields := "purpose algorithm encoding public_bytes key_id public_sha256 declared_state declared_state_at"
			for _, optional := range []string{"not_before", "not_after"} {
				if _, present := obj(key)[optional]; present {
					fields += " " + optional
				}
			}
			governanceObjectKeys(t, key, fields)
		}
	}
	if meta := obj(obj(v["governance"])["meta_keeper"]); meta != nil {
		governanceObjectKeys(t, meta, "reference resolution member")
		governanceObjectKeys(t, meta["reference"], "product member")
		if meta["member"] != nil {
			governanceObjectKeys(t, meta["member"], "key kind role responsibility scope desired_state project principal")
		}
	}
}

// The frozen table is closed at every projected object. Public records remain
// declarations, owner/key order is deterministic, and empty keys stay arrays.
func TestGovernanceClosedOutput(t *testing.T) {
	c := governanceCard(t)
	declaredKeys := arr(obj(arr(c["principals"])[0])["keys"])
	obj(declaredKeys[0])["not_before"] = "2026-11-17T00:00:00Z"
	obj(declaredKeys[0])["not_after"] = "2026-11-18T00:00:00Z"
	v := governanceResult(t, c, "station_beacon_orch")
	governanceClosedShape(t, v, true)
	self, source, g := obj(v["self"]), obj(v["source"]), obj(v["governance"])
	if v["schema"] != "pulse-view/v5" || v["mode"] != "local-unsigned" || v["product"] != "aurora" || v["member"] != "aurora/beacon_orch" || v["known_limit"] != knownLimit || v["hierarchy_status"] != "self-only; hierarchy not disclosed" || v["query_observed_at"] != governanceTime.Format(time.RFC3339Nano) {
		t.Fatal("wrong output constants")
	}
	if !reflect.DeepEqual(v["caller"], object{"agent": "station_beacon_orch", "cryptographically_verified": false, "trust": "single-user local trust"}) || self["kind"] != "agent" || self["role"] != "orchestrator" || self["responsibility"] != "orchestrator" || self["scope"] != "project" || self["resolved_name"] != "@station_beacon_orch:example.org" || self["project"] != "aurora/beacon" || self["project_state"] != "declared" {
		t.Fatal("wrong caller/self declaration")
	}
	expectedSelf := object{
		"key": "aurora/beacon_orch", "kind": "agent", "role": "orchestrator", "responsibility": "orchestrator", "scope": "project", "desired_state": "active",
		"resolved_name": "@station_beacon_orch:example.org", "project": "aurora/beacon", "project_state": "declared",
		"execution_binding": object{"key": "legacy", "operator": "fixture", "host": "station", "connection": "legacy-example"},
		"profile": object{
			"requirement": object{"context": "fixture-context", "compatible": ">=0.1.0 <0.2.0"},
			"selected":    object{"version": "0.1.0", "source_rev": strings.Repeat("a", 40), "lock_sha256": strings.Repeat("b", 64)},
		},
	}
	if !reflect.DeepEqual(self, expectedSelf) || !reflect.DeepEqual(v["projects"], []any{object{"key": "aurora/beacon", "repo": "local", "desired_state": "active", "location": object{"kind": "local", "path": "../beacon"}}}) {
		t.Fatal("self/profile/project values differ from the declaration contract")
	}
	if source["schema"] != "product-card/v5" || source["revision"] != "sha256:"+digest(encode(t, c)) || source["sha256"] != digest(encode(t, c)) || source["card_root"] != "/fixture/card" || source["freshness"] != "unknown" || source["observed_at"] != nil || source["validation"] != "local declarations only; source attestations and star reservations unverified" {
		t.Fatal("wrong source boundary")
	}
	owners := arr(g["owners"])
	if g["state"] != "declared" || len(owners) != 2 || obj(owners[0])["id"] != "example-owner" || obj(owners[1])["id"] != "example-participant" {
		t.Fatal("wrong owner order")
	}
	keys := arr(obj(owners[0])["keys"])
	if len(keys) != 3 || obj(keys[0])["algorithm"] != "P-256" || obj(keys[1])["algorithm"] != "BIP-340" || obj(keys[2])["algorithm"] != "Ed25519" {
		t.Fatal("keys must sort by purpose then key_id")
	}
	if !reflect.DeepEqual(owners[0], object{"id": "example-owner", "kind": "human", "name": "Example owner", "contacts": []any{object{"kind": "matrix", "mxid": "@owner:example.org"}}, "keys": []any{declaredKeys[1], declaredKeys[2], declaredKeys[0]}}) {
		t.Fatal("public records or optional validity bounds were rewritten")
	}
	if empty, ok := obj(owners[1])["keys"].([]any); !ok || len(empty) != 0 {
		t.Fatal("empty public material must be an empty array")
	}
	if !reflect.DeepEqual(g["meta_keeper"], object{"reference": object{"product": "constellation", "member": "external_keeper"}, "resolution": "external_unresolved", "member": nil}) {
		t.Fatal("external reference was resolved or rewritten")
	}
	before := encode(t, c)
	declaredGovernance(c)
	if !bytes.Equal(before, encode(t, c)) {
		t.Fatal("projection sorting mutated source maps/slices")
	}
}

// Legacy output changes only by the specified governance additions. Legacy
// extensions do not gain v5 semantics; absent v5 governance is undeclared.
func TestGovernanceLegacyAndAbsent(t *testing.T) {
	for _, version := range []string{"v3", "v4", "v5"} {
		t.Run(version, func(t *testing.T) {
			c := fixture(t)
			if version == "v3" {
				obj(c["product"])["governance"] = object{"owners": "legacy extension, not v5"}
				obj(arr(c["members"])[0])["kind"] = "human"
			} else if version == "v4" {
				c = v4Fixture(t)
			} else {
				c = governanceCard(t)
				delete(obj(c["product"]), "governance")
			}
			v := governanceResult(t, c, "station_aurora_keeper")
			governanceClosedShape(t, v, version != "v3")
			if !reflect.DeepEqual(v["governance"], object{"state": "undeclared", "owners": []any{}, "meta_keeper": nil}) {
				t.Fatal("legacy or absent metadata acquired semantics")
			}
			if version != "v5" {
				b, err := projectView(encode(t, c), "sha256:"+digest(encode(t, c)), "/fixture/card", "station_aurora_keeper", governanceTime, "")
				if err != nil {
					t.Fatal(err)
				}
				var expected object
				if err := json.Unmarshal(b, &expected); err != nil {
					t.Fatal(err)
				}
				expected["schema"], expected["governance"] = "pulse-view/v5", object{"state": "undeclared", "owners": []any{}, "meta_keeper": nil}
				obj(expected["self"])["kind"], obj(expected["self"])["responsibility"] = "agent", obj(expected["self"])["role"]
				if !reflect.DeepEqual(v, expected) {
					t.Fatal("legacy projection changed beyond specified fields")
				}
			}
		})
	}
}

// Local meta-keepers expose only their agreed summary. Humans never acquire
// execution fields, and contacts cannot add a caller or an ambiguous match.
func TestGovernanceLocalMetaAndHumanBoundary(t *testing.T) {
	for _, human := range []bool{false, true} {
		c := governanceCard(t)
		obj(obj(c["product"])["governance"])["meta_keeper"] = "aurora_keeper"
		if human {
			m := obj(arr(c["members"])[0])
			h := object{"kind": "human", "principal": "example-owner"}
			for _, k := range []string{"id", "role", "responsibility", "project", "parent", "scope", "desired_state"} {
				h[k] = m[k]
			}
			arr(c["members"])[0] = h
		}
		v := governanceResult(t, c, "station_beacon_orch")
		governanceClosedShape(t, v, true)
		meta := obj(obj(v["governance"])["meta_keeper"])
		kind := "agent"
		var principal any
		if human {
			kind, principal = "human", "example-owner"
		}
		if meta["resolution"] != "resolved" || !reflect.DeepEqual(meta["reference"], object{"product": "aurora", "member": "aurora_keeper"}) || !reflect.DeepEqual(meta["member"], object{"key": "aurora/aurora_keeper", "kind": kind, "role": "keeper", "responsibility": "keeper", "scope": "product", "desired_state": "active", "project": "aurora/aurora-host", "principal": principal}) {
			t.Fatal("wrong local meta summary")
		}
	}
	c := governanceCard(t)
	obj(arr(c["principals"])[0])["contacts"] = []any{object{"kind": "matrix", "mxid": "@station_beacon_orch:example.org"}, object{"kind": "matrix", "mxid": "@owner:example.org"}}
	v := governanceResult(t, c, "station_beacon_orch")
	if !reflect.DeepEqual(obj(arr(obj(v["governance"])["owners"])[0])["contacts"], obj(arr(c["principals"])[0])["contacts"]) {
		t.Fatal("contacts reordered")
	}
	for _, agent := range []string{"owner", "human_researcher", "station_human_researcher", "@owner:example.org"} {
		raw := encode(t, c)
		out, err := projectView(raw, "sha256:"+digest(raw), "/fixture/card", agent, governanceTime, "governance")
		wantRule(t, err, "caller_unknown")
		if len(out) != 0 {
			t.Fatal("human contact acquired a caller")
		}
	}
}

// Titles never select role codes or scope; remote projects remain coordinates.
func TestGovernanceTitleAndRemoteCoordinates(t *testing.T) {
	c := governanceCard(t)
	m := obj(arr(c["members"])[1])
	m["identity"] = object{"scheme": "domain-star-v1", "star": "sirius"}
	m["role"] = "keeper"
	location := object{"kind": "remote", "operator": "fixture-operator", "execution_host": "fixture-host", "project_locator": "example/beacon", "checkout_path": "../beacon"}
	p := obj(arr(c["projects"])[1])
	p["location"], p["task_system"] = location, object{"kind": "tasks-md", "locator": "example/tasks"}
	v := governanceResult(t, c, "example.org_aurora_beacon_orch_sirius")
	governanceClosedShape(t, v, true)
	self := obj(v["self"])
	if self["role"] != "keeper" || self["responsibility"] != "orchestrator" || self["scope"] != "project" || self["resolved_name"] != "@example.org_aurora_beacon_orch_sirius:example.org" || !reflect.DeepEqual(obj(arr(v["projects"])[0])["location"], location) {
		t.Fatal("title or remote coordinate changed mechanics")
	}
}

// Existing chat service contracts stay usable without an invented profile;
// non-chat services cannot impersonate a conversational caller.
func TestGovernanceServices(t *testing.T) {
	raw, err := os.ReadFile("../../docs/fixtures/v4/full/services-v4.json")
	if err != nil {
		t.Fatal(err)
	}
	var c object
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"product-card/v4", "product-card/v5"} {
		c["schema"] = version
		if version == "product-card/v5" {
			c["principals"] = []any{}
			for _, value := range arr(c["members"]) {
				m := obj(value)
				m["kind"], m["responsibility"] = "agent", m["role"]
			}
		}
		v := governanceResult(t, c, "station_signal_service")
		governanceClosedShape(t, v, true)
		if self := obj(v["self"]); self["profile"] != nil || self["kind"] != "agent" || self["responsibility"] != "service" || self["scope"] != "member" {
			t.Fatal("service acquired an invented execution profile")
		}
		bytes := encode(t, c)
		nonchat := obj(arr(c["members"])[len(arr(c["members"]))-1])
		if nonchat["id"] != "health_service" || nonchat["result_route"] != "parent-native" || obj(nonchat["a2a"]) != nil {
			t.Fatal("missing non-chat service control")
		}
		domain := str(obj(obj(c["bindings"])[str(nonchat["binding"])])["identity_domain"])
		prospective := strings.Join([]string{domain, "aurora", str(nonchat["project"]), "service", str(obj(nonchat["identity"])["star"])}, "_")
		out, err := projectView(bytes, "sha256:"+digest(bytes), "/fixture/card", prospective, governanceTime, "governance")
		wantRule(t, err, "caller_unknown")
		if len(out) != 0 {
			t.Fatal("non-chat service acquired caller output")
		}
	}
}

// Whole-card invalidity, including unselected owner material, collapses to the
// fixed safe refusal before output. The nearby valid source must work first.
func TestGovernanceQueryRefusals(t *testing.T) {
	governanceResult(t, governanceCard(t), "station_beacon_orch")
	for _, edit := range []func(object){
		func(c object) { obj(obj(c["product"])["governance"])["owners"] = []any{"absent"} },
		func(c object) { obj(arr(c["members"])[4])["runtime"] = object{"kind": "none"} },
		func(c object) {
			obj(obj(c["product"])["governance"])["owners"] = []any{"example-participant"}
			obj(arr(obj(arr(c["principals"])[0])["keys"])[0])["public_sha256"] = strings.Repeat("0", 64)
		},
		func(c object) { obj(arr(obj(arr(c["principals"])[0])["keys"])[2])["purpose"] = "grant-authority" },
		func(c object) { obj(c["product"])["description"] = "sk-" + strings.Repeat("A", 40) },
	} {
		c := governanceCard(t)
		edit(c)
		raw := encode(t, c)
		out, err := projectView(raw, "sha256:"+digest(raw), "/fixture/card", "station_beacon_orch", governanceTime, "governance")
		wantRule(t, err, "card_invalid")
		if len(out) != 0 {
			t.Fatal("invalid document leaked a partial view")
		}
	}
	raw := encode(t, governanceCard(t))
	for _, bad := range [][]byte{append([]byte("unused: &a value\n"), raw...), append(append([]byte{}, raw...), []byte("\n---\n{}")...)} {
		out, err := projectView(bad, "sha256:"+digest(bad), "/fixture/card", "station_beacon_orch", governanceTime, "governance")
		wantRule(t, err, "card_invalid")
		if len(out) != 0 {
			t.Fatal("strict YAML refusal leaked output")
		}
	}
	for _, view := range []string{"", "hierarchy", "routes"} {
		out, err := projectView(raw, "sha256:"+digest(raw), "/fixture/card", "station_beacon_orch", governanceTime, view)
		wantRule(t, err, "unsupported_schema")
		if len(out) != 0 {
			t.Fatal("old selector admitted v5")
		}
	}
	if out, err := projectView(raw, "sha256:"+digest(raw), "/fixture/card", "station_beacon_orch", time.Time{}, "governance"); len(out) != 0 {
		t.Fatal("zero evaluation clock leaked output")
	} else {
		wantRule(t, err, "card_invalid")
	}
	collision := governanceCard(t)
	m := obj(arr(collision["members"])[2])
	m["binding"] = "gr"
	obj(m["a2a"])["connection"] = "target-gr"
	obj(m["identity"])["mxid"] = "@station_beacon_orch:gr.example.org"
	collisionBytes := encode(t, collision)
	if out, err := projectView(collisionBytes, "sha256:"+digest(collisionBytes), "/fixture/card", "station_beacon_orch", governanceTime, "governance"); len(out) != 0 {
		t.Fatal("ambiguous caller leaked output")
	} else {
		wantRule(t, err, "caller_ambiguous")
	}
	company := encode(t, companyFixture(t))
	out, err := projectView(company, "sha256:"+digest(company), "/fixture/card", "station_company_keeper", governanceTime, "governance")
	wantRule(t, err, "unsupported_schema")
	if len(out) != 0 {
		t.Fatal("company acquired product governance")
	}
}

// The real public Run path accepts exactly the new explicit selector. Invalid
// discovery/override flags fail before file access; actual refusals keep stdout
// empty and preserve source bytes, mode, mtime and directory entries.
func TestGovernanceCLIAndPreservation(t *testing.T) {
	c := governanceCard(t)
	for _, p := range arr(c["principals"]) {
		for _, k := range arr(obj(p)["keys"]) {
			obj(k)["declared_state_at"] = "2020-01-01T00:00:00Z"
		}
	}
	raw := encode(t, c)
	dir := ownDirectory(t)
	path := writeFixture(t, dir, raw)
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	entriesBefore, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"me", "--local-unsigned", "--view", "governance", "--card", path, "--revision", "sha256:" + digest(raw), "--format", "json"}
	t.Setenv("A2A_AGENT", "station_beacon_orch")
	var out, diagnostics bytes.Buffer
	if code := Run(args, &out, &diagnostics); code != 0 || diagnostics.Len() != 0 {
		t.Fatal(code, diagnostics.String())
	}
	var result object
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	governanceClosedShape(t, result, true)
	for _, flag := range []string{"--sources", "--sources-revision", "--root", "--clock", "--member", "--view"} {
		bad := append(append([]string{}, args...), flag, "/unread/fixture")
		bad[5] = "/unread/card"
		out.Reset()
		diagnostics.Reset()
		if code := Run(bad, &out, &diagnostics); code != 2 || out.Len() != 0 || diagnostics.String() != "RULE usage: $\n" {
			t.Fatal(flag, code, diagnostics.String())
		}
	}
	selected, err := parseSelectors(args)
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range []string{"source_digest", "caller_unknown", "os_principal", "input_trust"} {
		s, agent, uid, euid := selected, "station_beacon_orch", os.Getuid(), os.Geteuid()
		switch rule {
		case "source_digest":
			s.revision = "sha256:" + strings.Repeat("0", 64)
		case "caller_unknown":
			agent = "human_researcher"
		case "os_principal":
			euid++
		case "input_trust":
			s.card = filepath.Join(dir, "link")
			if err := os.Symlink(path, s.card); err != nil {
				t.Fatal(err)
			}
		}
		out.Reset()
		diagnostics.Reset()
		if code := runSelected(s, uid, euid, agent, governanceTime, &out, &diagnostics); code != 1 || out.Len() != 0 || diagnostics.String() != "RULE "+rule+": $\n" {
			t.Fatal(rule, code, diagnostics.String())
		}
		if rule == "input_trust" {
			if err := os.Remove(s.card); err != nil {
				t.Fatal(err)
			}
		}
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(raw, actual) || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("input mutation", err)
	}
	entriesAfter, err := os.ReadDir(dir)
	if err != nil || !reflect.DeepEqual(entriesBefore, entriesAfter) {
		t.Fatal("query changed source directory", err)
	}
}

// Real Run accepts nearby legal declarations, then collapses each new whole-card
// violation to card_invalid with empty stdout and unchanged file/directory state.
func TestGovernanceReviewRefusals(t *testing.T) {
	key := func(c object, n int) object { return obj(arr(obj(arr(c["principals"])[0])["keys"])[n]) }
	check := func(t *testing.T, c object, want string) {
		t.Helper()
		for _, p := range arr(c["principals"]) {
			for _, k := range arr(obj(p)["keys"]) {
				obj(k)["declared_state_at"] = "2020-01-01T00:00:00Z"
			}
		}
		raw := encode(t, c)
		dir := ownDirectory(t)
		path := writeFixture(t, dir, raw)
		before, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		t.Setenv("A2A_AGENT", "station_beacon_orch")
		args := []string{"me", "--local-unsigned", "--view", "governance", "--card", path, "--revision", "sha256:" + digest(raw), "--format", "json"}
		var out, diagnostics bytes.Buffer
		code := Run(args, &out, &diagnostics)
		if want == "" {
			if code != 0 || diagnostics.Len() != 0 {
				t.Fatal(code, diagnostics.String())
			}
			var result object
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			governanceClosedShape(t, result, true)
		} else if code != 1 || out.Len() != 0 || diagnostics.String() != "RULE "+want+": $\n" {
			t.Fatal(code, diagnostics.String(), "stdout bytes", out.Len())
		}
		after, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		actual, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(raw, actual) || !bytes.Equal(raw, encode(t, c)) || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
			t.Fatal("query changed input", err)
		}
		afterEntries, err := os.ReadDir(dir)
		if err != nil || !reflect.DeepEqual(entries, afterEntries) {
			t.Fatal("query changed directory", err)
		}
	}
	for _, purpose := range []string{"identity", "grant-authority"} {
		t.Run("valid-p256-"+purpose, func(t *testing.T) {
			c := governanceCard(t)
			if obj(arr(c["members"])[1])["kind"] != "agent" || obj(arr(c["members"])[4])["kind"] != "human" {
				t.Fatal("missing valid member-kind neighbors")
			}
			key(c, 1)["purpose"] = purpose
			check(t, c, "")
		})
	}
	t.Run("valid-unselected-public-record", func(t *testing.T) {
		c := governanceCard(t)
		obj(obj(c["product"])["governance"])["owners"] = []any{"example-participant"}
		check(t, c, "")
	})
	for _, tc := range []struct {
		name string
		edit func(*testing.T, object)
	}{
		{"missing-member-kind", func(t *testing.T, c object) { delete(obj(arr(c["members"])[1]), "kind") }},
		{"unknown-member-kind", func(t *testing.T, c object) { obj(arr(c["members"])[1])["kind"] = "machine" }},
		{"unselected-p256-no-lift", func(t *testing.T, c object) {
			obj(obj(c["product"])["governance"])["owners"] = []any{"example-participant"}
			point, err := hex.DecodeString("02fd4bf61763b46581fd9174d623516cf3c81edd40e29ffa2777fb6cb0ae3ce535")
			if err != nil {
				t.Fatal(err)
			}
			value := base64.RawURLEncoding.EncodeToString(point)
			k := key(c, 1)
			k["public_bytes"], k["key_id"], k["public_sha256"] = value, "p256:"+value, digest(point)
		}},
		{"nonzero-unused-bits", func(t *testing.T, c object) {
			k := key(c, 0)
			value := str(k["public_bytes"])
			alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
			last := strings.IndexByte(alphabet, value[len(value)-1])
			if len(value)%4 != 3 || last < 0 || last%4 != 0 {
				t.Fatal("missing canonical trailing-bit neighbor")
			}
			noncanonical := value[:len(value)-1] + string(alphabet[last|1])
			k["public_bytes"], k["key_id"] = noncanonical, "ed25519:"+noncanonical
		}},
		{"reversed-window", func(t *testing.T, c object) {
			k := key(c, 0)
			k["not_before"], k["not_after"] = "2026-11-18T00:00:00Z", "2026-11-17T00:00:00Z"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := governanceCard(t)
			tc.edit(t, c)
			check(t, c, "card_invalid")
		})
	}
}
