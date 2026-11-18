package pulse

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/swarma-pulse/internal/cardv4"
)

func v5EnvironmentFixture(t *testing.T) object {
	t.Helper()
	c := governanceCard(t)
	for _, index := range []int{0, 1, 2, 3} {
		m := obj(arr(c["members"])[index])
		m["pulse"] = completeExtension()
		p := obj(m["pulse"])
		p["a2a_config_profile"] = "Config-" + str(m["id"])
		p["working_root"] = "../unopened/" + str(m["id"])
		obj(p["network"])["ref"] = "Proxy-" + str(m["id"])
	}
	// Distinct declarations make a manager-value substitution observable.
	childProfile := object{"path": "child-role",
		"requirement": object{"context": "child-context", "compatible": ">=0.2.0 <0.3.0"},
		"selected":    object{"version": "0.2.0", "source_rev": strings.Repeat("c", 40), "lock_sha256": strings.Repeat("d", 64)}}
	obj(obj(c["bindings"])["gr"])["operator"] = "child-operator"
	obj(obj(c["bindings"])["gr"])["host"] = "child-host"
	obj(c["profiles"])["child-profile"] = childProfile
	m := obj(arr(c["members"])[3])
	m["profile"], m["runtime"] = "child-profile", object{"kind": "claude_code", "model": "child-model", "effort": "child-effort"}
	return c
}

func v5EnvironmentResult(t *testing.T, c object, caller, member string) object {
	t.Helper()
	raw := encode(t, c)
	if _, issue := cardv4.ParseGovernanceSnapshot(raw, governanceTime); issue != nil {
		t.Fatalf("positive fixture invalid: %v", issue)
	}
	b, err := projectSelectedView(raw, "sha256:"+digest(raw), "/fixture/card", caller, governanceTime, "environment", member)
	if err != nil {
		t.Fatal(err)
	}
	var v object
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, encode(t, c)) || !bytes.HasSuffix(b, []byte("\n")) {
		t.Fatal("source changed or missing newline")
	}
	return v
}

func v5EnvironmentShape(t *testing.T, v object, target bool) {
	t.Helper()
	subject := "self"
	if target {
		subject = "target"
	}
	governanceObjectKeys(t, v, "schema mode query caller known_limit sources "+subject+" peers direct_reports descendants superiors projects products boundaries hierarchy_status observations query_observed_at")
	governanceObjectKeys(t, v["caller"], "agent cryptographically_verified trust")
	governanceObjectKeys(t, v["query"], "view"+map[bool]string{true: " member"}[target])
	if len(arr(v["sources"])) != 1 {
		t.Fatal("extra source")
	}
	governanceObjectKeys(t, arr(v["sources"])[0], "id schema revision sha256 card_root freshness observed_at validation")
	s := obj(v[subject])
	governanceObjectKeys(t, s, "key kind role responsibility scope desired_state resolved_name execution_binding profile project project_state level environment")
	governanceObjectKeys(t, s["execution_binding"], "key operator host connection")
	governanceObjectKeys(t, s["profile"], "requirement selected")
	governanceObjectKeys(t, obj(s["profile"])["requirement"], "context compatible")
	governanceObjectKeys(t, obj(s["profile"])["selected"], "version source_rev lock_sha256")
	e := obj(s["environment"])
	governanceObjectKeys(t, e, "schema state profile runtime a2a working_root network missing")
	governanceObjectKeys(t, e["a2a"], "agent resolved_name binding operator host connection server_name config_profile")
	governanceObjectKeys(t, e["runtime"], "kind model effort")
	governanceObjectKeys(t, e["profile"], "key source requirement selected")
	governanceObjectKeys(t, obj(e["profile"])["source"], "source base profiles_root path")
	governanceObjectKeys(t, e["working_root"], "source base path origin")
	governanceObjectKeys(t, e["network"], "mode ref")
	governanceObjectKeys(t, v["observations"], "runtime registration installed_profile assignment")
	for _, value := range obj(v["observations"]) {
		if !reflect.DeepEqual(value, object{"state": "unknown"}) {
			t.Fatal("fabricated observation")
		}
	}
	for _, key := range []string{"peers", "direct_reports", "descendants", "superiors", "products", "boundaries"} {
		if !reflect.DeepEqual(v[key], []any{}) {
			t.Fatal("extra export", key)
		}
	}
}

// Self v6 and explicit self v7 keep the same subject facts; target output is
// closed recursively, uses only target declarations and retains the real caller.
func TestV5EnvironmentSelfAndTargetShapes(t *testing.T) {
	c := v5EnvironmentFixture(t)
	self := v5EnvironmentResult(t, c, "station_storage_orch", "")
	v5EnvironmentShape(t, self, false)
	if self["schema"] != "pulse-view/v6" || obj(self["self"])["kind"] != "agent" || obj(self["self"])["responsibility"] != "orchestrator" {
		t.Fatal(self)
	}
	explicitSelf := v5EnvironmentResult(t, c, "station_storage_orch", "storage_orch")
	v5EnvironmentShape(t, explicitSelf, true)
	se, te := obj(obj(self["self"])["environment"]), obj(obj(explicitSelf["target"])["environment"])
	if se["schema"] != "pulse-environment/v1" || te["schema"] != "pulse-environment/v2" {
		t.Fatal(se, te)
	}
	te["schema"] = "pulse-environment/v1"
	if !reflect.DeepEqual(self["self"], explicitSelf["target"]) {
		t.Fatal("explicit self changed facts")
	}
	for _, chat := range []bool{false, true} {
		c := v5EnvironmentFixture(t)
		m := obj(arr(c["members"])[3])
		if chat {
			m["a2a"] = object{"connection": "target-gr", "parent_route": "#vault:gr.example.org", "rooms": []any{"#vault:gr.example.org"}, "subscriptions": object{"required": []any{}, "self": []any{}}}
			m["chat_admission"] = "synthetic-fixture-admission"
			m["result_route"] = "a2a"
		}
		m["desired_state"] = "parked"
		v := v5EnvironmentResult(t, c, "station_storage_orch", "vault_dev")
		v5EnvironmentShape(t, v, true)
		s := obj(v["target"])
		e := obj(s["environment"])
		a := obj(e["a2a"])
		if v["schema"] != "pulse-view/v7" || obj(v["caller"])["agent"] != "station_storage_orch" || v["hierarchy_status"] != "target-only; hierarchy not disclosed" || !reflect.DeepEqual(v["query"], object{"view": "environment", "member": "vault_dev"}) {
			t.Fatal("target query/caller constants mismatch", v)
		}
		if s["key"] != "aurora/vault_dev" || s["desired_state"] != "parked" || s["scope"] != "member" || s["project"] != "aurora/vault" || s["project_state"] != "declared" || s["level"] != "member" || len(arr(v["projects"])) != 1 || obj(arr(v["projects"])[0])["key"] != "aurora/vault" {
			t.Fatal("target subject identity/state/project mismatch", s)
		}
		if obj(e["profile"])["key"] != "child-profile" || obj(e["runtime"])["model"] != "child-model" || obj(e["runtime"])["effort"] != "child-effort" || obj(e["working_root"])["path"] != "../unopened/vault_dev" || obj(e["network"])["ref"] != "Proxy-vault_dev" {
			t.Fatal("manager values substituted", e)
		}
		if a["binding"] != "gr" || a["operator"] != "child-operator" || a["host"] != "child-host" || a["connection"] != "target-gr" || a["server_name"] != "gr.example.org" || a["config_profile"] != "Config-vault_dev" {
			t.Fatal(a)
		}
		if chat {
			name := "@gr.example.org_aurora_vault_dev_vega:gr.example.org"
			if s["resolved_name"] != name || a["resolved_name"] != name || a["agent"] != strings.Split(strings.TrimPrefix(name, "@"), ":")[0] || e["state"] != "complete" || !reflect.DeepEqual(e["missing"], []any{}) {
				t.Fatal(s)
			}
		} else if s["resolved_name"] != nil || a["resolved_name"] != nil || a["agent"] != nil || e["state"] != "incomplete" || !reflect.DeepEqual(e["missing"], []any{"a2a.agent", "a2a.resolved_name"}) {
			t.Fatal("non-chat identity fabricated", s)
		}
		if obj(arr(v["sources"])[0])["card_root"] != "/fixture/card" || v["known_limit"] != knownLimit {
			t.Fatal(v)
		}
	}
}

// Complete validation and actual caller eligibility precede exact target
// lookup, branch eligibility, the direct-parent predicate and coordinates.
func TestV5EnvironmentTargetRefusals(t *testing.T) {
	for _, tc := range []struct {
		caller, target, rule string
		edit                 func(object)
	}{
		{"station_storage_orch", "missing", "target_unknown", nil},
		{"station_beacon_orch", "human_researcher", "target_ineligible", nil},
		{"station_storage_orch", "beacon_orch", "target_forbidden", nil},
		{"station_aurora_keeper", "vault_dev", "target_forbidden", nil},
		{"station_storage_orch", "aurora_keeper", "target_forbidden", nil},
		{"unknown", "missing", "caller_unknown", nil},
		{"station_storage_orch", "vault_dev", "card_invalid", func(c object) { obj(obj(arr(c["members"])[1])["pulse"])["extra"] = true }},
		{"unknown", "missing", "card_invalid", func(c object) { obj(obj(arr(c["members"])[1])["pulse"])["extra"] = true }},
		{"station_storage_orch", "vault_dev", "card_invalid", func(c object) { c["members"] = append(arr(c["members"]), arr(c["members"])[3]) }},
		{"station_storage_orch", "vault_dev", "environment_invalid", func(c object) { obj(obj(c["profiles"])["child-profile"])["path"] = "https:remote" }},
	} {
		c := v5EnvironmentFixture(t)
		if tc.edit != nil {
			tc.edit(c)
		}
		raw := encode(t, c)
		b, err := projectSelectedView(raw, "sha256:"+digest(raw), "/fixture/card", tc.caller, governanceTime, "environment", tc.target)
		wantRule(t, err, tc.rule)
		if len(b) != 0 {
			t.Fatal("refusal emitted partial output")
		}
	}
}

// The public selector is view-specific before source access. Empty context is
// checked before unsafe input; digest precedes remaining context/parser checks.
func TestV5EnvironmentCLIPhaseOrder(t *testing.T) {
	c := v5EnvironmentFixture(t)
	raw := encode(t, c)
	path := writeFixture(t, ownDirectory(t), raw)
	args := []string{"me", "--local-unsigned", "--view", "environment", "--member", "vault_dev", "--card", path, "--revision", "sha256:" + digest(raw), "--format", "json"}
	t.Setenv("A2A_AGENT", "station_storage_orch")
	var out, diag bytes.Buffer
	if code := Run(args, &out, &diag); code != 0 || diag.Len() != 0 {
		t.Fatal(code, diag.String())
	}
	for _, bad := range [][]string{
		append(append([]string{}, args...), "--member=vault_dev"),
		{"me", "--local-unsigned", "--view", "environment", "--member=", "--card", "/absent", "--revision", "sha256:" + digest(raw), "--format", "json"},
		{"me", "--local-unsigned", "--view", "governance", "--member", "vault_dev", "--card", "/absent", "--revision", "sha256:" + digest(raw), "--format", "json"},
	} {
		out.Reset()
		diag.Reset()
		if code := Run(bad, &out, &diag); code != 2 || out.Len() != 0 || diag.String() != "RULE usage: $\n" {
			t.Fatal(code, out.String(), diag.String())
		}
	}
	s, err := parseSelectors(args)
	if err != nil {
		t.Fatal(err)
	}
	s.card = "/absent"
	out.Reset()
	diag.Reset()
	if code := runSelected(s, os.Getuid(), os.Geteuid(), "", governanceTime, &out, &diag); code != 1 || out.Len() != 0 || diag.String() != "RULE caller_context: $\n" {
		t.Fatal(code, diag.String())
	}
	s.card = path
	s.revision = "sha256:" + strings.Repeat("0", 64)
	out.Reset()
	diag.Reset()
	if code := runSelected(s, os.Getuid(), os.Geteuid(), "invalid\n", governanceTime, &out, &diag); code != 1 || out.Len() != 0 || diag.String() != "RULE source_digest: $\n" {
		t.Fatal(code, diag.String())
	}
}

// V5 uses location.path only for a local project after explicit pulse/clone
// roots; remote locators never become card-relative paths or guessed defaults.
func TestV5EnvironmentRootsAndMissing(t *testing.T) {
	for _, tc := range []struct {
		name, origin, path string
		remote             bool
	}{
		{"pulse", "pulse.working_root", "../unopened/storage_orch", false},
		{"clone", "member.clone", "../vault", false},
		{"local-project", "project.location.path", "../vault", false},
		{"remote-no-fallback", "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := v5EnvironmentFixture(t)
			m := obj(arr(c["members"])[2])
			if tc.name != "pulse" {
				delete(m, "pulse")
			}
			if tc.name == "local-project" || tc.remote {
				delete(m, "clone")
			}
			if tc.remote {
				p := obj(arr(c["projects"])[2])
				p["location"] = object{"kind": "remote", "operator": "remote-operator", "execution_host": "remote-host", "project_locator": "opaque/project", "checkout_path": "../not-a-fallback"}
				p["task_system"] = object{"kind": "tasks-md", "locator": "opaque/tasks"}
			}
			delete(obj(m["runtime"]), "effort")
			v := v5EnvironmentResult(t, c, "station_storage_orch", "")
			e := obj(obj(v["self"])["environment"])
			if obj(e["runtime"])["effort"] != nil {
				t.Fatal("effort guessed")
			}
			if tc.remote {
				if e["working_root"] != nil || !reflect.DeepEqual(e["missing"], []any{"a2a.config_profile", "network", "working_root"}) || e["state"] != "incomplete" {
					t.Fatal(e)
				}
			} else if !reflect.DeepEqual(e["working_root"], object{"source": "aurora", "base": "card", "path": tc.path, "origin": tc.origin}) {
				t.Fatal(e)
			}
		})
	}
}

// A target ID is the complete case-sensitive source string, including legacy
// slash IDs. Neither prefix splitting, a full MXID nor title privilege works.
func TestV5EnvironmentExactTargetAndTitle(t *testing.T) {
	c := v5EnvironmentFixture(t)
	m := obj(arr(c["members"])[3])
	m["id"] = "Legacy/Seat"
	m["role"] = "keeper"
	ref := object{"kind": "v3-card", "revision": "fixture-legacy", "sha256": strings.Repeat("c", 64), "member": "aurora/Legacy/Seat"}
	m["identity"] = object{"scheme": "legacy-exact", "mxid": "@legacy/seat:gr.example.org", "source_ref": ref}
	m["logical_source_ref"] = ref
	m["a2a"] = object{"connection": "target-gr", "parent_route": "#vault:gr.example.org", "rooms": []any{"#vault:gr.example.org"}, "subscriptions": object{"required": []any{}, "self": []any{}}}
	m["chat_admission"] = "synthetic-fixture-admission"
	m["result_route"] = "a2a"
	v := v5EnvironmentResult(t, c, "station_storage_orch", "Legacy/Seat")
	if obj(v["target"])["key"] != "aurora/Legacy/Seat" || obj(v["target"])["resolved_name"] != "@legacy/seat:gr.example.org" {
		t.Fatal(v)
	}
	for _, target := range []string{"legacy/seat", "Seat", "aurora/Legacy/Seat", "@legacy/seat:gr.example.org"} {
		raw := encode(t, c)
		b, err := projectSelectedView(raw, "sha256:"+digest(raw), "/fixture/card", "station_storage_orch", governanceTime, "environment", target)
		wantRule(t, err, "target_unknown")
		if len(b) != 0 {
			t.Fatal("partial output")
		}
	}
	raw := encode(t, c)
	b, err := projectSelectedView(raw, "sha256:"+digest(raw), "/fixture/card", "legacy/seat", governanceTime, "environment", "storage_orch")
	wantRule(t, err, "target_forbidden")
	if len(b) != 0 {
		t.Fatal("title acquired privilege")
	}
}

// Legacy no-selector views retain their contract; target mode is v5-only.
// Malformed unselected v5 declarations still collapse to card_invalid.
func TestV5EnvironmentLegacyAndWholeCard(t *testing.T) {
	for _, c := range []object{fixture(t), v4Fixture(t), companyFixture(t)} {
		raw := encode(t, c)
		b, err := projectSelectedView(raw, "sha256:"+digest(raw), "/fixture/card", "station_beacon_orch", governanceTime, "environment", "beacon_orch")
		wantRule(t, err, "unsupported_schema")
		if len(b) != 0 {
			t.Fatal("legacy target admitted")
		}
	}
	c := v5EnvironmentFixture(t)
	before := governanceResult(t, c, "station_beacon_orch")
	for _, value := range arr(c["members"]) {
		delete(obj(value), "pulse")
	}
	after := governanceResult(t, c, "station_beacon_orch")
	for _, v := range []object{before, after} {
		source := obj(v["source"])
		delete(source, "revision")
		delete(source, "sha256")
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("pulse leaked into governance")
	}
	for _, bad := range []any{nil, object{"schema": "pulse-member-environment/v1", "network": nil}, object{"schema": "pulse-member-environment/v1", "password": "placeholder"}} {
		c := v5EnvironmentFixture(t)
		obj(arr(c["members"])[1])["pulse"] = bad
		raw := encode(t, c)
		for _, view := range []string{"environment", "governance"} {
			b, err := projectView(raw, "sha256:"+digest(raw), "/fixture/card", "unknown", governanceTime, view)
			wantRule(t, err, "card_invalid")
			if len(b) != 0 {
				t.Fatal("partial invalid source")
			}
		}
	}
}

// Caller ambiguity is resolved before target lookup, and a matching service
// caller never gains a conversational environment. Service targets are refused.
func TestV5EnvironmentCallerAndServiceBoundaries(t *testing.T) {
	c := v5EnvironmentFixture(t)
	m := obj(arr(c["members"])[2])
	m["binding"] = "gr"
	obj(m["a2a"])["connection"] = "target-gr"
	obj(m["identity"])["mxid"] = "@station_beacon_orch:gr.example.org"
	raw := encode(t, c)
	b, err := projectSelectedView(raw, "sha256:"+digest(raw), "/fixture/card", "station_beacon_orch", governanceTime, "environment", "missing")
	wantRule(t, err, "caller_ambiguous")
	if len(b) != 0 {
		t.Fatal("ambiguous caller output")
	}
	raw, err = os.ReadFile("../../docs/fixtures/v4/full/services-v4.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	c["schema"] = "product-card/v5"
	c["principals"] = []any{}
	for _, v := range arr(c["members"]) {
		m := obj(v)
		m["kind"], m["responsibility"] = "agent", m["role"]
	}
	raw = encode(t, c)
	for _, tc := range []struct{ caller, target, rule string }{
		{"station_signal_service", "missing", "environment_invalid"},
		{"station_signal_service", "", "environment_invalid"},
		{"station_beacon_orch", "signal_service", "target_ineligible"},
		{"station_storage_orch", "health_service", "target_ineligible"},
	} {
		b, err := projectSelectedView(raw, "sha256:"+digest(raw), "/fixture/card", tc.caller, governanceTime, "environment", tc.target)
		wantRule(t, err, tc.rule)
		if len(b) != 0 {
			t.Fatal("service output")
		}
	}
}
