package pulse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func companyFixture(t *testing.T) object {
	t.Helper()
	c := fixture(t)
	h := obj(c["product"])
	c["schema"] = "company-card/v1"
	c["company"] = object{"id": "example-company", "description": "Synthetic company", "privacy": h["privacy"], "keeper": "company_keeper", "default_binding": h["default_binding"]}
	delete(c, "product")
	delete(c, "projects")
	keeper := obj(arr(c["members"])[0])
	keeper["id"] = "company_keeper"
	keeper["parent"] = nil
	keeper["pulse"] = completeExtension()
	c["members"] = []any{keeper}
	c["products"] = []any{}
	return c
}
func companyEnvironment(t *testing.T, c object) object {
	t.Helper()
	raw := encode(t, c)
	b, err := projectView(raw, "sha256:"+digest(raw), "/fixture/company", "station_company_keeper", fixedTime, "environment")
	if err != nil {
		t.Fatal(err)
	}
	var v object
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func inventoryEntry(id, keeper string) object {
	return object{"id": id, "description": "Synthetic product", "host": object{"kind": "local", "path": "../" + id}, "card": object{"path": "../" + id + "/product-card.yaml"}, "keeper": id + "/" + keeper, "superior": "example-company/company_keeper", "parent_alias": "external_keeper"}
}
func companyMember(t *testing.T, id, parent string) object {
	m := obj(arr(fixture(t)["members"])[3])
	m["id"] = id
	m["parent"] = parent
	delete(m, "project")
	return m
}

// Company is a distinct discriminant with reused v3 predicates: explicit root
// and declared external string parents pass, while old product null still fails.
func TestCompanyCoreContract(t *testing.T) {
	c := companyFixture(t)
	v := companyEnvironment(t, c)
	s := obj(v["self"])
	if s["scope"] != "company" || s["level"] != "company" || s["project"] != nil || len(arr(v["projects"])) != 0 || obj(arr(v["sources"])[0])["schema"] != "company-card/v1" {
		t.Fatal(s, v)
	}
	raw := encode(t, c)
	_, err := project(raw, "sha256:"+digest(raw), "/fixture/company", "station_company_keeper", fixedTime)
	wantRule(t, err, "unsupported_schema")
	obj(arr(c["members"])[0])["parent"] = "external_keeper"
	_ = companyEnvironment(t, c)
	changes := []func(object){
		func(c object) { delete(obj(arr(c["members"])[0]), "parent") },
		func(c object) { c["extra"] = true },
		func(c object) { obj(c["company"])["extra"] = true },
		func(c object) { obj(arr(c["members"])[0])["project"] = "invented" },
		func(c object) {
			m := companyMember(t, "other", "company_keeper")
			m["parent"] = nil
			c["members"] = append(arr(c["members"]), m)
		},
		func(c object) { m := companyMember(t, "other", "missing"); c["members"] = append(arr(c["members"]), m) },
		func(c object) { m := companyMember(t, "other", "other"); c["members"] = append(arr(c["members"]), m) },
		func(c object) {
			m := obj(arr(fixture(t)["members"])[0])
			m["id"] = "second"
			m["parent"] = nil
			c["members"] = append(arr(c["members"]), m)
		},
		func(c object) { obj(obj(c["profiles"])["control"])["selected"] = object{"version": "not-a-version"} },
		func(c object) { obj(arr(c["members"])[0])["binding"] = "missing" },
		func(c object) { obj(arr(c["members"])[0])["a2a"] = nil },
	}
	for i, change := range changes {
		c := companyFixture(t)
		change(c)
		raw := encode(t, c)
		_, err := projectView(raw, "sha256:"+digest(raw), "/fixture/company", "station_company_keeper", fixedTime, "environment")
		if err == nil || err.Error() != "RULE card_invalid: $" {
			t.Fatal(i, err)
		}
	}
	product := fixture(t)
	obj(arr(product["members"])[0])["parent"] = nil
	raw = encode(t, product)
	_, err = projectView(raw, "sha256:"+digest(raw), "/fixture/product", "station_aurora_keeper", fixedTime, "environment")
	wantRule(t, err, "card_invalid")
}

type universeFixture struct {
	root       string
	selections map[string]sourceSelection
	files      []string
}

func writeUniverse(t *testing.T, cards map[string]object) *universeFixture {
	t.Helper()
	u := &universeFixture{root: ownDirectory(t), selections: map[string]sourceSelection{}}
	for name, c := range cards {
		dir := filepath.Join(u.root, name)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		leaf := "product-card.yaml"
		if c["schema"] == "company-card/v1" {
			leaf = "company-card.yaml"
		}
		path := filepath.Join(dir, leaf)
		raw := encode(t, c)
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		u.selections[name] = sourceSelection{id: cardID(c), schema: str(c["schema"]), path: path, revision: "sha256:" + digest(raw)}
		u.files = append(u.files, path)
	}
	return u
}
func (u *universeFixture) selectSources(t *testing.T, primary string, additional ...string) selectors {
	t.Helper()
	p := u.selections[primary]
	entries := []any{}
	for _, name := range additional {
		a := u.selections[name]
		relative, err := filepath.Rel(filepath.Dir(p.path), a.path)
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, object{"id": a.id, "schema": a.schema, "path": relative, "revision": a.revision})
	}
	raw := sourcesBytes(t, entries...)
	manifest := filepath.Join(u.root, "sources-"+primary+".json")
	if err := os.WriteFile(manifest, raw, 0600); err != nil {
		t.Fatal(err)
	}
	u.files = append(u.files, manifest)
	return selectors{card: p.path, revision: p.revision, view: "discovery", sources: manifest, sourcesRevision: "sha256:" + digest(raw), root: u.root}
}
func discoveryResult(t *testing.T, s selectors, agent string) (object, error) {
	t.Helper()
	cards, manifest, err := loadDiscovery(s, os.Getuid())
	if err != nil {
		return nil, err
	}
	b, err := projectDiscovery(cards, manifest, agent, fixedTime)
	if err != nil {
		return nil, err
	}
	var v object
	err = json.Unmarshal(b, &v)
	return v, err
}
func relatedKeys(v any) []string {
	out := []string{}
	for _, value := range arr(v) {
		out = append(out, str(obj(value)["key"]))
	}
	return out
}
func requireKeys(t *testing.T, v any, want ...string) {
	t.Helper()
	if want == nil {
		want = []string{}
	}
	if !reflect.DeepEqual(relatedKeys(v), want) {
		t.Fatal("keys", relatedKeys(v), "want", want)
	}
}
func twoProductUniverse(t *testing.T) (object, object, object) {
	c := companyFixture(t)
	a := fixture(t)
	b := fixture(t)
	obj(b["product"])["id"] = "borealis"
	obj(obj(b["bindings"])["legacy"])["identity_prefix"] = "second_"
	c["products"] = []any{inventoryEntry("aurora", "aurora_keeper"), inventoryEntry("borealis", "aurora_keeper")}
	return c, a, b
}

// Reciprocal projections use the same composed graph, declared responsibility
// levels rather than hop depth, full identities, and no related private fields.
func TestDiscoveryReciprocalProjection(t *testing.T) {
	c, a, b := twoProductUniverse(t)
	obj(arr(b["members"])[1])["desired_state"] = "parked"
	obj(arr(a["members"])[2])["pulse"] = completeExtension()
	obj(obj(arr(a["members"])[2])["pulse"])["a2a_config_profile"] = "private-other-marker"
	u := writeUniverse(t, map[string]object{"company": c, "aurora": a, "borealis": b})
	s := u.selectSources(t, "company", "borealis", "aurora")
	v, err := discoveryResult(t, s, "station_company_keeper")
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 17 || v["hierarchy_status"] != "complete" || len(arr(v["products"])) != 2 || len(arr(v["boundaries"])) != 0 || obj(v["caller"])["cryptographically_verified"] != false {
		t.Fatal(v)
	}
	requireKeys(t, v["direct_reports"], "aurora/aurora_keeper", "borealis/aurora_keeper")
	requireKeys(t, v["peers"])
	requireKeys(t, v["superiors"])
	if len(arr(v["descendants"])) != 8 {
		t.Fatal(v["descendants"])
	}
	if str(obj(arr(v["sources"])[0])["id"]) != "example-company" || obj(arr(v["sources"])[1])["id"] != "aurora" {
		t.Fatal(v["sources"])
	}
	for _, field := range []string{"direct_reports", "descendants"} {
		for _, value := range arr(v[field]) {
			summary := obj(value)
			if len(summary) != 7 || summary["environment"] != nil || summary["profile"] != nil {
				t.Fatal(summary)
			}
		}
	}
	if strings.Contains(string(encode(t, v)), "private-other-marker") {
		t.Fatal("related environment leaked")
	}
	for _, p := range arr(v["projects"]) {
		loc := obj(obj(p)["location"])
		if loc["source"] == nil || loc["base"] != "card" {
			t.Fatal(p)
		}
	}
	s = u.selectSources(t, "aurora", "company", "borealis")
	keeper, err := discoveryResult(t, s, "station_aurora_keeper")
	if err != nil {
		t.Fatal(err)
	}
	requireKeys(t, keeper["peers"], "borealis/aurora_keeper")
	requireKeys(t, keeper["superiors"], "example-company/company_keeper")
	requireKeys(t, keeper["direct_reports"], "aurora/beacon_orch", "aurora/storage_orch")
	requireKeys(t, keeper["descendants"], "aurora/beacon_orch", "aurora/storage_orch", "aurora/vault_dev")
	worker, err := discoveryResult(t, s, "station_beacon_orch")
	if err != nil {
		t.Fatal(err)
	}
	requireKeys(t, worker["superiors"], "aurora/aurora_keeper", "example-company/company_keeper")
	requireKeys(t, worker["peers"], "aurora/storage_orch", "borealis/beacon_orch", "borealis/storage_orch")
	// Combined project manager stays product-level even though it owns a project.
	host := obj(arr(a["members"])[0])
	host["project"] = "aurora-host"
	u2 := writeUniverse(t, map[string]object{"aurora": a})
	partial, err := discoveryResult(t, u2.selectSources(t, "aurora"), "station_aurora_keeper")
	if err != nil || obj(partial["self"])["level"] != "product" || partial["hierarchy_status"] != "partial" {
		t.Fatal(partial, err)
	}
}

// Explicit absent/remote inventory stays partial without opening its locator;
// supplied invalid, inconsistent or remotely reinterpreted sources refuse whole.
func TestDiscoveryCompositionRefusals(t *testing.T) {
	c, a, b := twoProductUniverse(t)
	u := writeUniverse(t, map[string]object{"company": c, "aurora": a})
	v, err := discoveryResult(t, u.selectSources(t, "company", "aurora"), "station_company_keeper")
	if err != nil || v["hierarchy_status"] != "partial" || len(arr(v["boundaries"])) != 1 || obj(arr(v["products"])[1])["resolution"] != "not_supplied" {
		t.Fatal(v, err)
	}
	cases := []struct {
		name, rule string
		change     func(object, object, object)
	}{
		{"relation", "relation_conflict", func(c, a, b object) { obj(arr(c["products"])[0])["parent_alias"] = "other" }},
		{"actual-path", "source_reference_mismatch", func(c, a, b object) {
			e := obj(arr(c["products"])[0])
			e["host"] = object{"kind": "local", "path": "../another"}
			obj(e["card"])["path"] = "../another/product-card.yaml"
		}},
		{"company-pin", "source_digest", func(c, a, b object) {
			obj(obj(arr(c["products"])[0])["card"])["revision"] = "sha256:" + strings.Repeat("0", 64)
		}},
		{"unreferenced", "source_unreferenced", func(c, a, b object) { c["products"] = arr(c["products"])[:1] }},
		{"remote-supplied", "remote_source_unsupported", func(c, a, b object) {
			obj(arr(c["products"])[0])["host"] = object{"kind": "remote", "operator": "another", "execution_host": "remote", "locator": "opaque://unopened"}
		}},
		{"identity", "identity_conflict", func(c, a, b object) { obj(obj(b["bindings"])["legacy"])["identity_prefix"] = "station_" }},
		{"invalid-other-graph", "card_invalid", func(c, a, b object) { obj(arr(b["members"])[1])["parent"] = "missing" }},
		{"invalid-other-extension", "environment_invalid", func(c, a, b object) { obj(arr(b["members"])[1])["pulse"] = object{"schema": "future"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, a, b := twoProductUniverse(t)
			tc.change(c, a, b)
			u := writeUniverse(t, map[string]object{"company": c, "aurora": a, "borealis": b})
			_, err := discoveryResult(t, u.selectSources(t, "aurora", "company", "borealis"), "station_beacon_orch")
			wantRule(t, err, tc.rule)
		})
	}
	// Separate domains with one localpart refuse caller ambiguity, not implicit selection.
	obj(obj(b["bindings"])["legacy"])["identity_prefix"] = "station_"
	obj(obj(b["connections"])["legacy-relux"])["server_name"] = "other.example"
	u = writeUniverse(t, map[string]object{"aurora": a, "borealis": b})
	_, err = discoveryResult(t, u.selectSources(t, "aurora", "borealis"), "station_beacon_orch")
	wantRule(t, err, "caller_ambiguous")
	// An opaque remote reference is successful without a corresponding supplied card.
	c = companyFixture(t)
	e := inventoryEntry("away", "keeper")
	e["host"] = object{"kind": "remote", "operator": "other", "execution_host": "other", "locator": "https://unopened.invalid/no-fetch"}
	c["products"] = []any{e}
	u = writeUniverse(t, map[string]object{"company": c})
	v, err = discoveryResult(t, u.selectSources(t, "company"), "station_company_keeper")
	if err != nil || obj(arr(v["products"])[0])["resolution"] != "remote_unavailable" {
		t.Fatal(v, err)
	}
}

// CLI success and typed failures preserve every selected card/manifest and do
// not emit partial views or raw private paths/scalars on pre-output refusal.
func TestDiscoveryCLIAndSourcePreservation(t *testing.T) {
	c, a, b := twoProductUniverse(t)
	u := writeUniverse(t, map[string]object{"company": c, "aurora": a, "borealis": b})
	s := u.selectSources(t, "aurora", "company", "borealis")
	type witness struct {
		raw  []byte
		info os.FileInfo
	}
	before := map[string]witness{}
	for _, p := range u.files {
		raw, _ := os.ReadFile(p)
		info, _ := os.Stat(p)
		before[p] = witness{raw, info}
	}
	for _, tc := range []struct {
		name, rule string
		change     func(*selectors)
	}{
		{"positive", "", func(*selectors) {}},
		{"primary-pin", "source_digest", func(s *selectors) { s.revision = "sha256:" + strings.Repeat("0", 64) }},
		{"manifest-pin", "source_digest", func(s *selectors) { s.sourcesRevision = "sha256:" + strings.Repeat("0", 64) }},
		{"manifest-missing", "input_trust", func(s *selectors) { s.sources = filepath.Join(u.root, "missing") }},
		{"read-root", "input_path", func(s *selectors) { s.root = filepath.Dir(s.card) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selected := s
			tc.change(&selected)
			var out, diag bytes.Buffer
			code := runSelected(selected, os.Getuid(), os.Geteuid(), "station_beacon_orch", fixedTime, &out, &diag)
			if tc.rule == "" {
				if code != 0 || diag.Len() != 0 || !json.Valid(out.Bytes()) {
					t.Fatal(code, diag.String())
				}
			} else if code != 1 || out.Len() != 0 || diag.String() != "RULE "+tc.rule+": $\n" {
				t.Fatal(code, out.String(), diag.String())
			}
		})
	}
	for p, w := range before {
		raw, _ := os.ReadFile(p)
		info, _ := os.Stat(p)
		if !bytes.Equal(raw, w.raw) || !os.SameFile(info, w.info) || info.Mode() != w.info.Mode() || !info.ModTime().Equal(w.info.ModTime()) {
			t.Fatal("source changed")
		}
	}
}

// Inclusive source/member/byte budgets and 64 reporting hops accept the exact
// boundary; the adjacent excess refuses, and cycles have their own graph refusal.
func TestUniverseLimitsAndReporting(t *testing.T) {
	if err := sourceBudget(32, 16<<20, 10000); err != nil {
		t.Fatal(err)
	}
	for _, values := range [][3]int{{33, 16 << 20, 10000}, {32, (16 << 20) + 1, 10000}, {32, 16 << 20, 10001}} {
		wantRule(t, sourceBudget(values[0], values[1], values[2]), "input_limit")
	}
	nodes := map[string]*graphMember{}
	for i := 0; i <= 64; i++ {
		key := fmt.Sprint(i)
		n := &graphMember{key: key}
		if i > 0 {
			n.parent = fmt.Sprint(i - 1)
		}
		nodes[key] = n
	}
	if err := checkReportingGraph(nodes); err != nil {
		t.Fatal(err)
	}
	nodes["65"] = &graphMember{key: "65", parent: "64"}
	wantRule(t, checkReportingGraph(nodes), "input_limit")
	delete(nodes, "65")
	nodes["0"].parent = "1"
	wantRule(t, checkReportingGraph(nodes), "relation_conflict")
}

// Nested canonical layout and direct children are legitimate; sibling-prefix,
// parent escapes and wrong leaves are rejected by components, not a denylist.
func TestCompanyLocalCardContainment(t *testing.T) {
	for _, path := range []string{"../aurora/product-card.yaml", "../aurora/product-card/product-card.yaml", "../aurora/nested/../product-card/product-card.yaml"} {
		c := companyFixture(t)
		e := inventoryEntry("aurora", "aurora_keeper")
		obj(e["card"])["path"] = path
		c["products"] = []any{e}
		if err := validateCompany(c); err != nil {
			t.Fatal(path, err)
		}
	}
	for _, path := range []string{"../aurora-other/product-card.yaml", "../outside/product-card.yaml", "../aurora/nested/../../outside/product-card.yaml", "../aurora", "../aurora/wrong.yaml", "/absolute/product-card.yaml", "~home/product-card.yaml", "https:product-card.yaml", "C:product-card.yaml", "bad\n/product-card.yaml", `../aurora/nested\path/product-card.yaml`} {
		c := companyFixture(t)
		e := inventoryEntry("aurora", "aurora_keeper")
		obj(e["card"])["path"] = path
		c["products"] = []any{e}
		wantRule(t, validateCompany(c), "input_path")
	}
	// A matching card descendant does not make a backslash-bearing host valid.
	c := companyFixture(t)
	e := inventoryEntry("aurora", "aurora_keeper")
	obj(e["host"])["path"] = `../aurora\host`
	obj(e["card"])["path"] = `../aurora\host/product-card.yaml`
	c["products"] = []any{e}
	wantRule(t, validateCompany(c), "input_path")
	c, a, _ := twoProductUniverse(t)
	c["products"] = arr(c["products"])[:1]
	obj(obj(arr(c["products"])[0])["card"])["path"] = "../aurora/product-card/product-card.yaml"
	u := writeUniverse(t, map[string]object{"company": c, "aurora": a})
	selection := u.selections["aurora"]
	nested := filepath.Join(filepath.Dir(selection.path), "product-card", "product-card.yaml")
	if err := os.MkdirAll(filepath.Dir(nested), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(selection.path, nested); err != nil {
		t.Fatal(err)
	}
	selection.path = nested
	u.selections["aurora"] = selection
	s := u.selectSources(t, "company", "aurora")
	v, err := discoveryResult(t, s, "station_company_keeper")
	if err != nil || v["hierarchy_status"] != "complete" {
		t.Fatal(v, err)
	}
	// Matching optional company pin passes; omitting it also passed immediately above.
	obj(obj(arr(c["products"])[0])["card"])["revision"] = selection.revision
	raw := encode(t, c)
	cp := u.selections["company"]
	if err := os.WriteFile(cp.path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	cp.revision = "sha256:" + digest(raw)
	u.selections["company"] = cp
	s = u.selectSources(t, "company", "aurora")
	if _, err = discoveryResult(t, s, "station_company_keeper"); err != nil {
		t.Fatal(err)
	}
	// Parent directory symlinks refuse despite the contained normalized spelling.
	actual := filepath.Join(filepath.Dir(filepath.Dir(nested)), "actual")
	if err := os.Rename(filepath.Dir(nested), actual); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(actual, filepath.Dir(nested)); err != nil {
		t.Fatal(err)
	}
	_, err = discoveryResult(t, s, "station_company_keeper")
	wantRule(t, err, "input_trust")
}

// A second company-level keeper with a STRING parent is a valid declaration;
// role repetition does not create another designated null root.
func TestCompanyAdditionalKeeperAndMembers(t *testing.T) {
	c := companyFixture(t)
	second := obj(arr(fixture(t)["members"])[0])
	second["id"] = "delegated_keeper"
	second["parent"] = "company_keeper"
	c["members"] = append(arr(c["members"]), second, companyMember(t, "developer", "delegated_keeper"))
	u := writeUniverse(t, map[string]object{"company": c})
	v, err := discoveryResult(t, u.selectSources(t, "company"), "station_company_keeper")
	if err != nil {
		t.Fatal(err)
	}
	requireKeys(t, v["peers"], "example-company/delegated_keeper", "example-company/developer")
	requireKeys(t, v["direct_reports"], "example-company/delegated_keeper")
	requireKeys(t, v["descendants"], "example-company/delegated_keeper", "example-company/developer")
	second["parent"] = nil
	wantRule(t, validateCompany(c), "card_invalid")
}

// Additional remote sources refuse before opening even a missing local payload;
// two company inputs and source id/schema disagreement never compose a subset.
func TestDiscoveryRemoteReadAndSourceConflicts(t *testing.T) {
	c, a, b := twoProductUniverse(t)
	e := obj(arr(c["products"])[1])
	e["host"] = object{"kind": "remote", "operator": "remote", "execution_host": "host", "locator": "opaque"}
	u := writeUniverse(t, map[string]object{"company": c, "aurora": a, "borealis": b})
	s := u.selectSources(t, "aurora", "borealis", "company")
	if err := os.Remove(u.selections["borealis"].path); err != nil {
		t.Fatal(err)
	}
	_, err := discoveryResult(t, s, "station_beacon_orch")
	wantRule(t, err, "remote_source_unsupported")
	c2 := companyFixture(t)
	obj(c2["company"])["id"] = "second-company"
	u = writeUniverse(t, map[string]object{"company": c, "second": c2})
	_, err = discoveryResult(t, u.selectSources(t, "company", "second"), "station_company_keeper")
	wantRule(t, err, "source_conflict")
	u = writeUniverse(t, map[string]object{"aurora": a, "borealis": b})
	s = u.selectSources(t, "aurora", "borealis")
	raw, _ := os.ReadFile(s.sources)
	var m object
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	obj(arr(m["cards"])[0])["id"] = "wrong-id"
	raw = encode(t, m)
	if err := os.WriteFile(s.sources, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s.sourcesRevision = "sha256:" + digest(raw)
	_, err = discoveryResult(t, s, "station_beacon_orch")
	wantRule(t, err, "source_conflict")
}

// New discovery file limits are typed input_limit; the legacy reader preserves
// its existing input_trust verdict for the same oversized immutable file.
func TestDiscoveryOversizeInputLimit(t *testing.T) {
	root := ownDirectory(t)
	path := writeFixture(t, root, bytes.Repeat([]byte("x"), (1<<20)+1))
	_, err := readCardBounded(path, os.Getuid(), "input_limit")
	wantRule(t, err, "input_limit")
	_, err = readCard(path, os.Getuid())
	wantRule(t, err, "input_trust")
}

// Legacy nonempty product ids keep raw lexical boundary order even when JSON
// escapes their text. Repeated queries retain all semantic fields unchanged.
func TestDiscoveryBoundaryOrderingAndStability(t *testing.T) {
	c := companyFixture(t)
	for _, id := range []string{"Zomitted", "<omitted"} {
		e := inventoryEntry(id, "keeper")
		c["products"] = append(arr(c["products"]), e)
	}
	u := writeUniverse(t, map[string]object{"company": c})
	s := u.selectSources(t, "company")
	first, err := discoveryResult(t, s, "station_company_keeper")
	if err != nil {
		t.Fatal(err)
	}
	boundaries := arr(first["boundaries"])
	if len(boundaries) != 2 || obj(boundaries[0])["target"] != "<omitted" || obj(boundaries[1])["target"] != "Zomitted" {
		t.Fatal(boundaries)
	}
	second, err := discoveryResult(t, s, "station_company_keeper")
	if err != nil {
		t.Fatal(err)
	}
	delete(first, "query_observed_at")
	delete(second, "query_observed_at")
	if !bytes.Equal(encode(t, first), encode(t, second)) {
		t.Fatal("semantic result changed")
	}
}
