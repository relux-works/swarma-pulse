package cardv4

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// ExactStringRecords checks complete, case-sensitive record keys before a
// typed JSON decoder can fold aliases. Empty lists remain explicit premises.
func ExactStringRecords(value any, fields ...string) bool {
	seen := map[string]bool{}
	for _, field := range fields {
		if field == "" || seen[field] {
			return false
		}
		seen[field] = true
	}
	rows, ok := value.([]any)
	if !ok {
		return false
	}
	for _, row := range rows {
		m := obj(row)
		if m == nil || len(m) != len(fields) {
			return false
		}
		for _, field := range fields {
			v, ok := m[field].(string)
			if !ok || v == "" {
				return false
			}
		}
	}
	return true
}

// DecodeConversionInputs selects only exact operational names. Other fixture
// observations stay ignored, but case-folded aliases of operational keys refuse.
func DecodeConversionInputs(raw []byte) (ConversionInputs, *Issue) {
	var in ConversionInputs
	refuse := func() (ConversionInputs, *Issue) {
		return ConversionInputs{}, shape("conversion_inputs_invalid", "$")
	}
	var doc map[string]json.RawMessage
	if len(raw) > MaxBytes || !utf8.Valid(raw) || json.Unmarshal(raw, &doc) != nil || doc == nil {
		return refuse()
	}
	known := words("source_revision source_sha256 host_project source_card_root target_card_root admitted_root external_parent_mapping additional_connections additional_bindings prospective_nonchat_choices reviewed_reservation_premises project_locations")
	selected := map[string]json.RawMessage{}
	for k, value := range doc {
		if known[k] {
			selected[k] = value
			continue
		}
		for name := range known {
			if strings.EqualFold(k, name) {
				return refuse()
			}
		}
	}
	if raw, present := selected["reviewed_reservation_premises"]; present {
		var value any
		if json.Unmarshal(raw, &value) != nil || !ExactStringRecords(value, "holder", "domain", "product", "primary_role", "star") {
			return refuse()
		}
	}
	canonical, e := json.Marshal(selected)
	if e != nil || json.Unmarshal(canonical, &in) != nil {
		return refuse()
	}
	return in, nil
}
