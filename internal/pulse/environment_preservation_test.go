package pulse

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/swarma-pulse/internal/cardv4"
)

func environmentPreservationFixtures(t *testing.T) (object, object, object) {
	t.Helper()
	values := []object{}
	for _, name := range []string{"environment-source-v3.json", "environment-authored-v5.json", "environment-preservation-ledger.json"} {
		raw, err := os.ReadFile("../../docs/fixtures/v5/" + name)
		if err != nil {
			t.Fatal(err)
		}
		var value object
		if err := json.Unmarshal(raw, &value); err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
	}
	return values[0], values[1], values[2]
}

// This bounded fixture oracle separately accounts for v5/pulse facts, then
// reuses the unchanged v3/v4 complete declaration comparison for the remaining
// inventory. It is not a converter or permission to normalize a real card.
func environmentPreservationRule(t *testing.T, source, target, ledger object) string {
	t.Helper()
	if issue := cardv4.Validate(encode(t, source), cardv4.Options{Version: 3}); issue != nil {
		return "source_invalid"
	}
	if _, issue := cardv4.ParseGovernanceSnapshot(encode(t, target), governanceTime); issue != nil {
		return "target_invalid"
	}
	var in cardv4.PreservationInputs
	if err := json.Unmarshal(encode(t, ledger), &in); err != nil {
		t.Fatal(err)
	}
	if target["schema"] != "product-card/v5" || !reflect.DeepEqual(target["principals"], []any{}) {
		return "new_v5_declarations"
	}
	oldByID, newByID := map[string]object{}, map[string]object{}
	for _, v := range arr(source["members"]) {
		m := obj(v)
		oldByID[str(m["id"])] = m
	}
	for _, v := range arr(target["members"]) {
		m := obj(v)
		newByID[str(m["id"])] = m
	}
	for id, m := range oldByID {
		n := newByID[id]
		if n == nil {
			return "member_inventory"
		}
		if n["kind"] != "agent" || n["responsibility"] != m["role"] || n["role"] != m["role"] {
			return "mechanical_title_preservation"
		}
		if m["role"] == "keeper" {
			if project := str(m["project"]); project != "" && project != in.HostProject {
				return "keeper_project_conflict"
			}
			mapping := obj(ledger["keeper_project_materialization"])
			if m["project"] == nil && (mapping["member"] != id || mapping["source_project"] != nil || mapping["target_project"] != n["project"]) {
				return "keeper_project_ledger"
			}
		}
		if obj(m["a2a"]) == nil {
			choice := in.ProspectiveNonchatChoices[cardID(source)+"/"+id]
			if choice == nil {
				return "nonchat_premise_missing"
			}
			if obj(choice["identity"])["scheme"] != "domain-star-v1" || !reflect.DeepEqual(choice["identity"], n["identity"]) || choice["binding"] != n["binding"] {
				return "nonchat_premise_conflict"
			}
			if n["a2a"] != nil || n["result_route"] != m["result_route"] {
				return "nonchat_route_preservation"
			}
			reservation := arr(ledger["reviewed_reservation_premises"])
			binding := obj(obj(target["bindings"])[str(n["binding"])])
			if len(reservation) != 1 || !reflect.DeepEqual(reservation[0], object{"holder": cardID(source) + "/" + id, "domain": binding["identity_domain"], "product": cardID(source), "primary_role": m["role"], "star": obj(n["identity"])["star"]}) {
				return "nonchat_reservation_premise"
			}
			continue
		}
		bindingKey := str(m["binding"])
		if bindingKey == "" {
			bindingKey = str(cardHeader(source)["default_binding"])
		}
		caller := str(obj(obj(source["bindings"])[bindingKey])["identity_prefix"]) + id
		oldRaw, newRaw := encode(t, source), encode(t, target)
		oldBytes, err := projectView(oldRaw, "sha256:"+digest(oldRaw), in.SourceCardRoot, caller, governanceTime, "environment")
		if err != nil {
			return "source_environment_invalid"
		}
		newBytes, err := projectView(newRaw, "sha256:"+digest(newRaw), in.TargetCardRoot, caller, governanceTime, "environment")
		if err != nil {
			return "target_environment_invalid"
		}
		var oldView, newView object
		if json.Unmarshal(oldBytes, &oldView) != nil || json.Unmarshal(newBytes, &newView) != nil {
			t.Fatal("invalid view encoding")
		}
		before, after := obj(oldView["self"]), obj(newView["self"])
		for _, pair := range []struct {
			subject object
			root    string
		}{{before, in.SourceCardRoot}, {after, in.TargetCardRoot}} {
			coordinate := obj(obj(pair.subject["environment"])["working_root"])
			if coordinate != nil {
				coordinate["path"] = filepath.Clean(filepath.Join(pair.root, str(coordinate["path"])))
				if coordinate["origin"] == "project.location.path" {
					coordinate["origin"] = "project.path"
				}
			}
		}
		be, ae := obj(before["environment"]), obj(after["environment"])
		if !reflect.DeepEqual(be["working_root"], ae["working_root"]) {
			return "root_absence_state_preservation"
		}
		if !reflect.DeepEqual(be["missing"], ae["missing"]) || be["state"] != ae["state"] {
			return "environment_missing_state_preservation"
		}
		delete(after, "kind")
		delete(after, "responsibility")
		if m["role"] == "keeper" && m["project"] == nil {
			if after["project"] != cardID(source)+"/"+in.HostProject || after["project_state"] != "declared" || len(arr(newView["projects"])) != 1 || before["project"] != nil || before["project_state"] != "undeclared" || len(arr(oldView["projects"])) != 0 {
				return "keeper_project_materialization"
			}
			after["project"], after["project_state"] = nil, "undeclared"
		}
		if !reflect.DeepEqual(before, after) {
			return "environment_projection_preservation"
		}
	}
	// Exact pulse comparison is a separate accounted inventory, including the
	// non-chat member. These fixtures explicitly retain equivalent parent bases.
	for id, m := range oldByID {
		if !reflect.DeepEqual(m["pulse"], newByID[id]["pulse"]) {
			return "pulse_inventory_preservation"
		}
	}
	var a, b object
	if json.Unmarshal(encode(t, source), &a) != nil || json.Unmarshal(encode(t, target), &b) != nil {
		t.Fatal("copy failed")
	}
	delete(b, "principals")
	for _, v := range arr(a["members"]) {
		delete(obj(v), "pulse")
	}
	for _, v := range arr(b["members"]) {
		m := obj(v)
		delete(m, "pulse")
		delete(m, "kind")
		delete(m, "responsibility")
	}
	if issue := cardv4.VerifyConversionPreservation(a, b, in); issue != nil {
		return issue.Rule
	}
	return ""
}

// The hand-authored fixture and explicit new-fact ledger preserve every source
// declaration, both pulse blocks, keeper mapping and non-chat native semantics.
// This supplies no real-card acceptance or actual launch evidence.
func TestV5EnvironmentAuthoringPreservation(t *testing.T) {
	source, target, ledger := environmentPreservationFixtures(t)
	if got := environmentPreservationRule(t, source, target, ledger); got != "" {
		t.Fatal(got)
	}
	for _, tc := range []struct {
		name, rule string
		edit       func(object, object, object)
	}{
		{"keeper-root-absence", "root_absence_state_preservation", func(s, t, l object) { delete(obj(arr(s["members"])[0]), "clone") }},
		{"keeper-conflicting-project", "keeper_project_conflict", func(s, t, l object) { obj(arr(s["members"])[0])["project"] = "beacon" }},
		{"missing-prospective", "nonchat_premise_missing", func(s, t, l object) { delete(l, "prospective_nonchat_choices") }},
		{"conflicting-prospective", "nonchat_premise_conflict", func(s, t, l object) {
			obj(obj(l["prospective_nonchat_choices"])["aurora/vault_dev"])["binding"] = "legacy"
		}},
		{"missing-reservation", "nonchat_reservation_premise", func(s, t, l object) { l["reviewed_reservation_premises"] = []any{} }},
		{"lost-unselected-pulse", "environment_missing_state_preservation", func(s, t, l object) { delete(obj(obj(arr(t["members"])[0])["pulse"]), "a2a_config_profile") }},
		{"changed-station", "conversion_station_preservation", func(s, t, l object) { obj(arr(t["members"])[1])["station"] = "changed-station" }},
		{"unknown-source-fact", "conversion_extension_unsupported", func(s, t, l object) { s["unaccounted_extension"] = object{"fact": "must-preserve"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, d, l := environmentPreservationFixtures(t)
			tc.edit(s, d, l)
			if got := environmentPreservationRule(t, s, d, l); got != tc.rule {
				t.Fatalf("want %s got %s", tc.rule, got)
			}
		})
	}
	if !strings.Contains(str(ledger["runtime_evidence"]), "not-run") {
		t.Fatal("fixture must not claim actual runtime use")
	}
}

// Existing v3-to-v4 conversion still rejects pulse rather than dropping it;
// the same reviewed premises convert the adjacent pulse-free source normally.
func TestEnvironmentDoesNotWidenConverter(t *testing.T) {
	for _, pulse := range []bool{false, true} {
		source, _, ledger := environmentPreservationFixtures(t)
		if !pulse {
			for _, v := range arr(source["members"]) {
				delete(obj(v), "pulse")
			}
		}
		raw := encode(t, source)
		var in cardv4.ConversionInputs
		if err := json.Unmarshal(encode(t, ledger), &in); err != nil {
			t.Fatal(err)
		}
		in.SourceSHA256 = digest(raw)
		result, issue := cardv4.Convert(raw, in)
		if pulse {
			if issue == nil || issue.Rule != "unknown_field" || len(result) != 0 {
				t.Fatal("pulse conversion boundary", issue)
			}
		} else if issue != nil || len(result) == 0 {
			t.Fatal("pulse-free conversion positive control", issue)
		}
	}
}
