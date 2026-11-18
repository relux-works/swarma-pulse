package pulse

import (
	"reflect"
	"testing"
)

// This fixture-only input qualification checks actual query receipts against
// independent consumer expectations. It neither performs canonical adoption nor
// accepts a launch; those need their own real custodian/manager evidence.
func environmentReceiptRule(view object, expected object) string {
	if view["schema"] != "pulse-view/v7" || obj(view["target"]) == nil || view["self"] != nil {
		return "target_receipt_required"
	}
	if len(arr(view["sources"])) != 1 {
		return "source_receipt_required"
	}
	source := obj(arr(view["sources"])[0])
	target := obj(view["target"])
	if source["sha256"] != expected["sha256"] || source["revision"] != "sha256:"+str(expected["sha256"]) {
		return "candidate_pin_mismatch"
	}
	if source["card_root"] != expected["card_root"] {
		return "actual_parent_mismatch"
	}
	if target["key"] != expected["target"] || obj(view["caller"])["agent"] != expected["caller"] {
		return "subject_mismatch"
	}
	environment := obj(target["environment"])
	if !reflect.DeepEqual(environment["runtime"], expected["runtime"]) || !reflect.DeepEqual(environment["profile"], expected["profile"]) {
		return "consumer_input_mismatch"
	}
	return ""
}

// Pin, actual parent, actual caller and exact target cannot be substituted.
// An incomplete declaration may qualify preserved inputs, but this test records
// no runtime receipt and proves no canonical swap or first real use.
func TestV5EnvironmentCandidateReceiptBinding(t *testing.T) {
	c := v5EnvironmentFixture(t)
	expected := object{
		"sha256": digest(encode(t, c)), "card_root": "/fixture/card", "target": "aurora/vault_dev", "caller": "station_storage_orch",
		"runtime": object{"kind": "claude_code", "model": "child-model", "effort": "child-effort"},
		"profile": object{"key": "child-profile", "source": object{"source": "aurora", "base": "profiles_root", "profiles_root": "../profiles", "path": "child-role"}, "requirement": object{"context": "child-context", "compatible": ">=0.2.0 <0.3.0"}, "selected": object{"version": "0.2.0", "source_rev": "cccccccccccccccccccccccccccccccccccccccc", "lock_sha256": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}},
	}
	v := v5EnvironmentResult(t, c, "station_storage_orch", "vault_dev")
	if rule := environmentReceiptRule(v, expected); rule != "" {
		t.Fatal(rule)
	}
	if obj(obj(v["target"])["environment"])["state"] != "incomplete" {
		t.Fatal("missing chat admission must remain visible")
	}
	for _, tc := range []struct {
		name, rule string
		edit       func(object)
	}{
		{"wrong-pin", "candidate_pin_mismatch", func(v object) { obj(arr(v["sources"])[0])["sha256"] = "changed" }},
		{"staged-is-not-canonical", "actual_parent_mismatch", func(v object) { obj(arr(v["sources"])[0])["card_root"] = "/fixture/staging" }},
		{"other-target", "subject_mismatch", func(v object) { obj(v["target"])["key"] = "aurora/storage_orch" }},
		{"target-caller-mutation", "subject_mismatch", func(v object) { obj(v["caller"])["agent"] = "target-context" }},
		{"manager-model", "consumer_input_mismatch", func(v object) {
			obj(obj(v["target"])["environment"])["runtime"] = object{"kind": "codex_cli", "model": "fixture-model", "effort": "high"}
		}},
		{"same-count-rename", "target_receipt_required", func(v object) { v["self"] = v["target"]; delete(v, "target") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := v5EnvironmentResult(t, c, "station_storage_orch", "vault_dev")
			tc.edit(v)
			if got := environmentReceiptRule(v, expected); got != tc.rule {
				t.Fatalf("want %s got %s", tc.rule, got)
			}
		})
	}
	if rule := environmentReceiptRule(v5EnvironmentResult(t, c, "station_storage_orch", ""), expected); rule != "target_receipt_required" {
		t.Fatal("manager self is not target evidence", rule)
	}
}
