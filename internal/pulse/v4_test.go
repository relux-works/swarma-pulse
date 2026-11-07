package pulse

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/swarma-pulse/internal/cardv4"
)

func v4Fixture(t *testing.T) object {
	t.Helper()
	raw, err := os.ReadFile("../../docs/fixtures/v4/full/converted-v4.json")
	if err != nil {
		t.Fatal(err)
	}
	var c object
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestV4SelfOnly(t *testing.T) {
	for _, tc := range []struct{ agent, member, scope, project string }{
		{"station_aurora_keeper", "aurora_keeper", "product", "aurora-host"},
		{"station_beacon_orch", "beacon_orch", "project", "beacon"},
		{"relux.works_aurora_beacon_orch_sirius", "beacon_orch", "project", "beacon"},
	} {
		t.Run(tc.agent, func(t *testing.T) {
			c := v4Fixture(t)
			if strings.HasPrefix(tc.agent, "relux.works_") {
				obj(arr(c["members"])[1])["identity"] = object{"scheme": "domain-star-v1", "star": "sirius"}
			}
			raw := encode(t, c)
			v := view(t, raw, tc.agent)
			s := obj(v["self"])
			if s["key"] != "aurora/"+tc.member || s["scope"] != tc.scope || s["project"] != "aurora/"+tc.project || s["project_state"] != "declared" || s["resolved_name"] != "@"+tc.agent+":relux.works" {
				t.Fatal(s)
			}
			if len(s) != 9 || len(v) != 16 || v["schema"] != "pulse-view/v1" || v["known_limit"] != knownLimit || obj(v["caller"])["cryptographically_verified"] != false {
				t.Fatal(v)
			}
			source := obj(v["source"])
			if source["schema"] != "product-card/v4" || source["sha256"] != digest(raw) || source["revision"] != "sha256:"+digest(raw) || source["validation"] != "local declarations only; source attestations and star reservations unverified" || source["freshness"] != "unknown" || source["observed_at"] != nil {
				t.Fatal(source)
			}
			if len(arr(v["projects"])) != 1 || obj(arr(v["projects"])[0])["key"] != "aurora/"+tc.project {
				t.Fatal(v["projects"])
			}
			for _, k := range []string{"peers", "direct_reports", "descendants", "superiors"} {
				if a, ok := v[k].([]any); !ok || len(a) != 0 {
					t.Fatal(k, v[k])
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

func TestV4CompleteSnapshotRefusals(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(object)
	}{
		{"relabeled-v3", func(c object) { obj(c["bindings"])["legacy"] = obj(fixture(t)["bindings"])["legacy"] }},
		{"hidden-invalid-profile", func(c object) { obj(arr(c["members"])[3])["profile"] = "missing" }},
		{"hidden-invalid-graph", func(c object) { obj(arr(c["members"])[2])["parent"] = "beacon_orch" }},
		{"unknown-field", func(c object) { c["extension"] = true }},
		{"null", func(c object) { obj(c["product"])["description"] = nil }},
		{"binding-domain", func(c object) { obj(obj(c["bindings"])["legacy"])["identity_domain"] = "foreign.example" }},
		{"legacy-server", func(c object) {
			obj(obj(arr(c["members"])[1])["identity"])["mxid"] = "@station_beacon_orch:foreign.example"
		}},
		{"legacy-key", func(c object) {
			obj(obj(obj(arr(c["members"])[1])["identity"])["source_ref"])["member"] = "aurora/foreign"
		}},
		{"logical-key", func(c object) {
			obj(arr(c["members"])[3])["logical_source_ref"] = object{"kind": "v3-card", "revision": "fixture", "sha256": strings.Repeat("a", 64), "member": "aurora/foreign"}
		}},
		{"duplicate-chat", func(c object) {
			obj(obj(arr(c["members"])[2])["identity"])["mxid"] = "@station_beacon_orch:relux.works"
		}},
		{"nonchat-legacy", func(c object) { obj(arr(c["members"])[3])["identity"] = obj(arr(c["members"])[1])["identity"] }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := v4Fixture(t)
			tc.mutate(c)
			raw := encode(t, c)
			out, err := project(raw, "sha256:"+digest(raw), "/fixture", "station_aurora_keeper", fixedTime)
			wantRule(t, err, "card_invalid")
			if len(out) != 0 {
				t.Fatal("partial output")
			}
		})
	}
	for _, schema := range []any{"product-card/v04", "product-card/v4.0", "product-card/v5", 4, nil} {
		c := v4Fixture(t)
		c["schema"] = schema
		raw := encode(t, c)
		_, err := project(raw, "sha256:"+digest(raw), "/fixture", "station_aurora_keeper", fixedTime)
		wantRule(t, err, "unsupported_schema")
	}
	raw := encode(t, v4Fixture(t))
	for _, bad := range [][]byte{
		append([]byte("<<: "), raw...),
		append([]byte("unused: &x value\n"), raw...),
		[]byte("schema: product-card/v4\nschema: product-card/v4\n"),
		append(append([]byte{}, raw...), []byte("\n---\n{}\n")...),
	} {
		_, err := project(bad, "sha256:"+digest(bad), "/fixture", "station_aurora_keeper", fixedTime)
		wantRule(t, err, "card_invalid")
	}
}

func TestV4CallerBoundary(t *testing.T) {
	raw := encode(t, v4Fixture(t))
	for _, agent := range []string{"foreign_identity", "@station_beacon_orch:relux.works", "beacon_orch", "station_vault_dev", "gr.relux.works_aurora_vault_dev_vega"} {
		out, err := project(raw, "sha256:"+digest(raw), "/fixture", agent, fixedTime)
		wantRule(t, err, "caller_unknown")
		if len(out) != 0 {
			t.Fatal("partial output")
		}
	}
	// Localpart collisions across declared servers must not choose a domain.
	c := v4Fixture(t)
	m := obj(arr(c["members"])[2])
	m["binding"] = "gr"
	obj(m["a2a"])["connection"] = "target-gr"
	obj(m["identity"])["mxid"] = "@station_beacon_orch:gr.relux.works"
	raw = encode(t, c)
	_, err := project(raw, "sha256:"+digest(raw), "/fixture", "station_beacon_orch", fixedTime)
	wantRule(t, err, "caller_ambiguous")
}

func TestV4HiddenNoninterference(t *testing.T) {
	c := v4Fixture(t)
	before := view(t, encode(t, c), "station_aurora_keeper")
	obj(arr(c["members"])[3])["desired_state"] = "parked"
	obj(arr(c["projects"])[2])["description"] = "hidden project details"
	after := view(t, encode(t, c), "station_aurora_keeper")
	for _, v := range []object{before, after} {
		delete(obj(v["source"]), "sha256")
		delete(obj(v["source"]), "revision")
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("hidden facts affected view")
	}
}

func TestV4RemoteProjectIsOnlyACoordinate(t *testing.T) {
	c := v4Fixture(t)
	p := obj(arr(c["projects"])[1])
	location := object{"kind": "remote", "operator": "fixture-remote", "execution_host": "fixture-host", "project_locator": "fixture/beacon", "checkout_path": "../beacon"}
	p["location"] = location
	p["task_system"] = object{"kind": "tasks-md", "locator": "fixture/tasks"}
	v := view(t, encode(t, c), "station_beacon_orch")
	if !reflect.DeepEqual(obj(arr(v["projects"])[0])["location"], location) {
		t.Fatal(v["projects"])
	}
}

func TestV4ReadOnlyCLIBoundary(t *testing.T) {
	for _, version := range []int{3, 4} {
		c := fixture(t)
		if version == 4 {
			c = v4Fixture(t)
		}
		raw := encode(t, c)
		dir := ownDirectory(t)
		path := writeFixture(t, dir, raw)
		symlink := filepath.Join(dir, "link.yaml")
		if err := os.Symlink(path, symlink); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct{ name, path, revision, agent, rule string }{
			{"valid", path, "sha256:" + digest(raw), "station_beacon_orch", ""},
			{"foreign", path, "sha256:" + digest(raw), "foreign_identity", "caller_unknown"},
			{"pin", path, "sha256:" + strings.Repeat("0", 64), "station_beacon_orch", "source_digest"},
			{"traversal", dir + "/child/../" + filepath.Base(path), "sha256:" + digest(raw), "station_beacon_orch", "input_path"},
			{"symlink", symlink, "sha256:" + digest(raw), "station_beacon_orch", "input_trust"},
		} {
			t.Run(string(rune('0'+version))+"/"+tc.name, func(t *testing.T) {
				var out, diagnostics bytes.Buffer
				exit := runSelected(selectors{card: tc.path, revision: tc.revision}, os.Getuid(), os.Geteuid(), tc.agent, fixedTime, &out, &diagnostics)
				if tc.rule == "" {
					if exit != 0 || out.Len() == 0 || diagnostics.Len() != 0 {
						t.Fatal(exit, diagnostics.String())
					}
				} else if exit != 1 || out.Len() != 0 || diagnostics.String() != "RULE "+tc.rule+": $\n" {
					t.Fatal(exit, out.String(), diagnostics.String())
				}
			})
		}
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(raw, got) {
			t.Fatal("input changed", err)
		}
	}
}

func TestV4SnapshotDoesNotAdmitIdentity(t *testing.T) {
	raw := encode(t, v4Fixture(t))
	if _, issue := cardv4.ParseLocalSnapshot(raw); issue != nil {
		t.Fatal(issue)
	}
	if issue := cardv4.Validate(raw, cardv4.Options{Version: 4}); issue == nil || issue.Rule != "legacy_source_missing" {
		t.Fatal("admission gate changed", issue)
	}
	_, err := projectView(raw, "sha256:"+digest(raw), "/fixture", "station_aurora_keeper", fixedTime, "environment")
	wantRule(t, err, "unsupported_schema")
	// New declared identities also remain behind the separate reservation gate.
	c := v4Fixture(t)
	for i, value := range arr(c["members"]) {
		obj(value)["identity"] = object{"scheme": "domain-star-v1", "star": []string{"vega", "polaris", "altair", "deneb"}[i]}
	}
	raw = encode(t, c)
	if _, issue := cardv4.ParseLocalSnapshot(raw); issue != nil {
		t.Fatal(issue)
	}
	if issue := cardv4.Validate(raw, cardv4.Options{Version: 4}); issue == nil || issue.Rule != "star_reservation_required" {
		t.Fatal("reservation gate changed", issue)
	}
}
