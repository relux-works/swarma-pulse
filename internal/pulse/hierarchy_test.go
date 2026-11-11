package pulse

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func hierarchyView(t *testing.T, c object, agent string) object {
	t.Helper()
	raw := encode(t, c)
	b, err := projectView(raw, "sha256:"+digest(raw), "/fixture/card", agent, fixedTime, "hierarchy")
	if err != nil {
		t.Fatal(err)
	}
	var v object
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// The same one-card relationships work in v3 and strict v4. External parents
// stay unresolved, non-chat reports have no name, and related environments stay private.
func TestHierarchyProjection(t *testing.T) {
	for _, version := range []string{"v3", "v4"} {
		t.Run(version, func(t *testing.T) {
			c := fixture(t)
			if version == "v4" {
				c = v4Fixture(t)
			}
			obj(arr(c["members"])[3])["desired_state"] = "parked"
			obj(obj(arr(c["members"])[2])["runtime"])["model"] = "related-runtime-fixture"
			keeper := hierarchyView(t, c, "station_aurora_keeper")
			if bytes.Contains(encode(t, keeper), []byte("related-runtime-fixture")) {
				t.Fatal("related runtime leaked")
			}
			if keeper["schema"] != "pulse-view/v3" || len(keeper) != 17 || keeper["hierarchy_status"] != "partial" || !reflect.DeepEqual(keeper["query"], object{"view": "hierarchy"}) {
				t.Fatal(keeper)
			}
			requireKeys(t, keeper["peers"])
			requireKeys(t, keeper["superiors"])
			requireKeys(t, keeper["direct_reports"], "aurora/beacon_orch", "aurora/storage_orch")
			requireKeys(t, keeper["descendants"], "aurora/beacon_orch", "aurora/storage_orch", "aurora/vault_dev")
			storage := hierarchyView(t, c, "station_storage_orch")
			requireKeys(t, storage["peers"], "aurora/beacon_orch")
			requireKeys(t, storage["superiors"], "aurora/aurora_keeper")
			requireKeys(t, storage["direct_reports"], "aurora/vault_dev")
			requireKeys(t, storage["descendants"], "aurora/vault_dev")
			report := obj(arr(storage["direct_reports"])[0])
			if report["resolved_name"] != nil || report["desired_state"] != "parked" {
				t.Fatal(report)
			}
			for _, field := range []string{"peers", "superiors", "direct_reports", "descendants"} {
				for _, value := range arr(storage[field]) {
					keys := []string{}
					for key := range obj(value) {
						keys = append(keys, key)
					}
					sort.Strings(keys)
					if !reflect.DeepEqual(keys, []string{"desired_state", "key", "level", "parent", "project", "resolved_name", "role"}) {
						t.Fatal("related private fields", value)
					}
				}
			}
			self := obj(storage["self"])
			if self["level"] != "project" || len(self) != 10 || self["resolved_name"] != "@station_storage_orch:relux.works" || self["environment"] != nil {
				t.Fatal(self)
			}
			if len(arr(storage["sources"])) != 1 || len(arr(storage["products"])) != 0 || obj(storage["caller"])["cryptographically_verified"] != false || storage["known_limit"] != knownLimit {
				t.Fatal(storage)
			}
			source := obj(arr(storage["sources"])[0])
			if source["schema"] != "product-card/"+version || source["revision"] != "sha256:"+digest(encode(t, c)) || source["card_root"] != "/fixture/card" || source["freshness"] != "unknown" || source["observed_at"] != nil {
				t.Fatal(source)
			}
			external := "external_keeper"
			if version == "v4" {
				external = "constellation/external_keeper"
				if source["validation"] != "local declarations only; source attestations and star reservations unverified" {
					t.Fatal(source)
				}
			}
			wantBoundary := []any{object{"from": "aurora/aurora_keeper", "relation": "parent", "target": object{"source": "aurora", "external": external}, "reason": "external_unresolved"}}
			if !reflect.DeepEqual(storage["boundaries"], wantBoundary) {
				t.Fatal(storage["boundaries"])
			}
		})
	}
}

// This normal fixture's discovery bytes match the output captured independently
// from base b8ffb301632462ce89e30ff5fd5c4f33aa6bc005 at the same fixed coordinates/time.
// The neighboring alias tests cover the intentional compatibility exception.
func TestDiscoveryBaseByteCompatibility(t *testing.T) {
	c := fixture(t)
	raw := validBytes(t, c)
	s := sourceSelection{id: "aurora", schema: "product-card/v3", path: "/fixture/aurora/product-card.yaml", revision: "sha256:" + digest(raw)}
	manifest := object{"revision": "sha256:" + strings.Repeat("1", 64), "sha256": strings.Repeat("2", 64), "root": "/fixture"}
	got, err := projectDiscovery([]*querySource{{selection: s, raw: raw, card: c}}, manifest, "station_storage_orch", fixedTime)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/discovery-base-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("normal discovery changed from base\nwant %s\ngot %s", want, got)
	}
}

// V3 product keepers' parents are external by the source schema even if an
// alias also names self, an orchestrator or a non-chat local member. Other local
// edges and company keeper-to-keeper edges retain their own schema semantics.
func TestHierarchyExternalParentAliasCollision(t *testing.T) {
	for _, alias := range []string{"external_keeper", "beacon_orch", "aurora_keeper", "vault_dev"} {
		t.Run(alias, func(t *testing.T) {
			c := fixture(t)
			c["external_members"] = []any{alias}
			obj(arr(c["members"])[0])["parent"] = alias
			_ = validBytes(t, c)
			v := hierarchyView(t, c, "station_beacon_orch")
			requireKeys(t, v["superiors"], "aurora/aurora_keeper")
			requireKeys(t, v["peers"], "aurora/storage_orch")
			want := []any{object{"from": "aurora/aurora_keeper", "relation": "parent", "target": object{"source": "aurora", "external": alias}, "reason": "external_unresolved"}}
			if !reflect.DeepEqual(v["boundaries"], want) || v["hierarchy_status"] != "partial" {
				t.Fatal(v)
			}
			u := writeUniverse(t, map[string]object{"aurora": c})
			discovery, err := discoveryResult(t, u.selectSources(t, "aurora"), "station_beacon_orch")
			if err != nil || !reflect.DeepEqual(discovery["boundaries"], want) || discovery["schema"] != "pulse-view/v2" || len(discovery) != 17 {
				t.Fatal(discovery, err)
			}
			requireKeys(t, discovery["superiors"], "aurora/aurora_keeper")
		})
	}
	c := companyFixture(t)
	m := obj(arr(fixture(t)["members"])[1])
	m["id"], m["role"], m["parent"] = "deputy", "keeper", "company_keeper"
	delete(m, "project")
	c["members"] = append(arr(c["members"]), m)
	c["external_members"] = []any{"company_keeper"}
	u := writeUniverse(t, map[string]object{"company": c})
	v, err := discoveryResult(t, u.selectSources(t, "company"), "station_deputy")
	if err != nil || v["hierarchy_status"] != "complete" || len(arr(v["boundaries"])) != 0 {
		t.Fatal(v, err)
	}
	requireKeys(t, v["superiors"], "example-company/company_keeper")
}

// This graph-level control creates an actual cycle after materialization;
// rejection therefore does not depend on violating a product-card role rule.
func TestHierarchyNormalizedCycleControl(t *testing.T) {
	c := fixture(t)
	raw := validBytes(t, c)
	nodes, err := materialize([]*querySource{{selection: sourceSelection{id: "aurora", schema: "product-card/v3"}, raw: raw, card: c}})
	if err != nil {
		t.Fatal(err)
	}
	if err := checkReportingGraph(nodes); err != nil {
		t.Fatal(err)
	}
	nodes["aurora/aurora_keeper"].parent = "aurora/beacon_orch"
	wantRule(t, checkReportingGraph(nodes), "relation_conflict")
}

// Strict v4 tuple names and legacy-exact federated names retain their domains;
// a caller-localpart collision refuses rather than choosing a server.
func TestHierarchyV4IdentitiesAndLocations(t *testing.T) {
	c := v4Fixture(t)
	m := obj(arr(c["members"])[1])
	m["identity"] = object{"scheme": "domain-star-v1", "star": "sirius"}
	p := obj(arr(c["projects"])[1])
	location := object{"kind": "remote", "operator": "fixture-remote", "execution_host": "fixture-host", "project_locator": "fixture/beacon", "checkout_path": "../beacon"}
	p["location"] = location
	p["task_system"] = object{"kind": "tasks-md", "locator": "fixture/tasks"}
	v := hierarchyView(t, c, "relux.works_aurora_beacon_orch_sirius")
	if obj(v["self"])["resolved_name"] != "@relux.works_aurora_beacon_orch_sirius:relux.works" {
		t.Fatal(v["self"])
	}
	want := object{"source": "aurora", "base": "card"}
	for k, value := range location {
		want[k] = value
	}
	found := false
	for _, value := range arr(v["projects"]) {
		p := obj(value)
		if p["key"] == "aurora/beacon" {
			found = true
			if !reflect.DeepEqual(p["location"], want) {
				t.Fatal(p)
			}
		}
	}
	if !found {
		t.Fatal("missing declared remote project", v["projects"])
	}
	c = v4Fixture(t)
	m = obj(arr(c["members"])[2])
	m["binding"] = "gr"
	obj(m["a2a"])["connection"] = "target-gr"
	obj(m["identity"])["mxid"] = "@station_beacon_orch:gr.relux.works"
	v = hierarchyView(t, c, "station_aurora_keeper")
	if obj(arr(v["direct_reports"])[0])["resolved_name"] != "@station_beacon_orch:relux.works" || obj(arr(v["direct_reports"])[1])["resolved_name"] != "@station_beacon_orch:gr.relux.works" {
		t.Fatal(v["direct_reports"])
	}
	raw := encode(t, c)
	b, err := projectView(raw, "sha256:"+digest(raw), "/fixture", "station_beacon_orch", fixedTime, "hierarchy")
	wantRule(t, err, "caller_ambiguous")
	if len(b) != 0 {
		t.Fatal("partial ambiguous output")
	}
}

// Complete-card validation rejects a cycle, unknown parent and invalid hidden
// member; the same valid card succeeds beside each rejected mutation.
func TestHierarchyGraphRefusals(t *testing.T) {
	for _, version := range []string{"v3", "v4"} {
		for _, change := range []struct {
			name  string
			apply func(object)
		}{
			{"cycle", func(c object) { obj(arr(c["members"])[1])["parent"] = "beacon_orch" }},
			{"unknown-parent", func(c object) { obj(arr(c["members"])[1])["parent"] = "missing" }},
			{"hidden-invalid", func(c object) { obj(arr(c["members"])[3])["profile"] = "missing" }},
		} {
			t.Run(version+"/"+change.name, func(t *testing.T) {
				c := fixture(t)
				if version == "v4" {
					c = v4Fixture(t)
				}
				_ = hierarchyView(t, c, "station_aurora_keeper")
				change.apply(c)
				raw := encode(t, c)
				b, err := projectView(raw, "sha256:"+digest(raw), "/fixture", "station_aurora_keeper", fixedTime, "hierarchy")
				wantRule(t, err, "card_invalid")
				if len(b) != 0 {
					t.Fatal("partial invalid graph")
				}
			})
		}
	}
}

// Different logical v3 members may not share one full Matrix identity, even
// when the caller is a different unambiguous account.
func TestHierarchyIdentityConflict(t *testing.T) {
	c := fixture(t)
	_ = hierarchyView(t, c, "station_aurora_keeper")
	obj(c["bindings"])["overlap"] = object{"operator": "fixture", "host": "station", "identity_prefix": "station_beacon_", "connection": "legacy-relux"}
	m := obj(arr(c["members"])[2])
	m["id"] = "orch"
	m["binding"] = "overlap"
	obj(arr(c["projects"])[2])["orchestrator"] = "orch"
	obj(arr(c["members"])[3])["parent"] = "orch"
	raw := validBytes(t, c)
	b, err := projectView(raw, "sha256:"+digest(raw), "/fixture", "station_aurora_keeper", fixedTime, "hierarchy")
	wantRule(t, err, "identity_conflict")
	if len(b) != 0 {
		t.Fatal("partial duplicate identity output")
	}
}

// The public CLI accepts only one pinned product source, refuses extra selectors
// before access, and preserves file bytes, mode, mtime and directory entries on both paths.
func TestHierarchyCLIReadOnlyBoundary(t *testing.T) {
	for _, version := range []string{"v3", "v4"} {
		t.Run(version, func(t *testing.T) {
			c := fixture(t)
			if version == "v4" {
				c = v4Fixture(t)
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
			args := []string{"me", "--local-unsigned", "--view", "hierarchy", "--card", path, "--revision", "sha256:" + digest(raw), "--format", "json"}
			t.Setenv("A2A_AGENT", "station_aurora_keeper")
			var out, diagnostics bytes.Buffer
			if code := Run(args, &out, &diagnostics); code != 0 || diagnostics.Len() != 0 || !json.Valid(out.Bytes()) {
				t.Fatal(code, out.String(), diagnostics.String())
			}
			var result object
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result["schema"] != "pulse-view/v3" || !reflect.DeepEqual(result["query"], object{"view": "hierarchy"}) {
				t.Fatal(result)
			}
			requireKeys(t, result["direct_reports"], "aurora/beacon_orch", "aurora/storage_orch")
			requireKeys(t, result["descendants"], "aurora/beacon_orch", "aurora/storage_orch", "aurora/vault_dev")
			for _, flag := range []string{"--sources", "--sources-revision", "--root", "--view"} {
				bad := append(append([]string{}, args...), flag, "/unread/fixture")
				out.Reset()
				diagnostics.Reset()
				if code := Run(bad, &out, &diagnostics); code != 2 || out.Len() != 0 || diagnostics.String() != "RULE usage: $\n" {
					t.Fatal(flag, code, out.String(), diagnostics.String())
				}
			}
			selected, err := parseSelectors(args)
			if err != nil {
				t.Fatal(err)
			}
			for _, rule := range []string{"source_digest", "caller_unknown", "input_trust"} {
				s := selected
				agent := "station_aurora_keeper"
				switch rule {
				case "source_digest":
					s.revision = "sha256:" + strings.Repeat("0", 64)
				case "caller_unknown":
					agent = "station_vault_dev"
				case "input_trust":
					s.card = filepath.Join(dir, "link")
					if err := os.Symlink(path, s.card); err != nil {
						t.Fatal(err)
					}
				}
				out.Reset()
				diagnostics.Reset()
				if code := runSelected(s, os.Getuid(), os.Geteuid(), agent, fixedTime, &out, &diagnostics); code != 1 || out.Len() != 0 || diagnostics.String() != "RULE "+rule+": $\n" {
					t.Fatal(rule, code, out.String(), diagnostics.String())
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
			if err != nil {
				t.Fatal(err)
			}
			afterEntries, err := os.ReadDir(dir)
			if err != nil || len(entries) != len(afterEntries) || !bytes.Equal(raw, actual) || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
				t.Fatal("input changed", err)
			}
		})
	}
	c := companyFixture(t)
	raw := encode(t, c)
	b, err := projectView(raw, "sha256:"+digest(raw), "/fixture", "station_company_keeper", fixedTime, "hierarchy")
	wantRule(t, err, "unsupported_schema")
	if len(b) != 0 {
		t.Fatal("partial company output")
	}
}
