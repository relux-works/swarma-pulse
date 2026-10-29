package cardv4

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

// Exact record admission refuses malformed and folded fields before struct
// decoding. Empty lists remain valid explicit premises, not implicit authority.
func TestExactMetadataRecords(t *testing.T) {
	for _, tc := range []struct {
		value  any
		fields []string
		valid  bool
	}{
		{[]any{}, []string{"file"}, true},
		{[]any{Object{"file": "source"}}, []string{"file"}, true},
		{nil, []string{"file"}, false},
		{Object{"file": "source"}, []string{"file"}, false},
		{[]any{nil}, []string{"file"}, false},
		{[]any{Object{"file": ""}}, []string{"file"}, false},
		{[]any{Object{"file": int64(1)}}, []string{"file"}, false},
		{[]any{Object{"FILE": "source"}}, []string{"file"}, false},
		{[]any{Object{"file": "source", "FILE": "source"}}, []string{"file"}, false},
		{[]any{Object{"file": "source", "other": "value"}}, []string{"file", "file"}, false},
	} {
		if ExactStringRecords(tc.value, tc.fields...) != tc.valid {
			t.Fatal("exact record admission")
		}
	}
}

// Nonoperational fixture observations retain arbitrary JSON values and cannot
// supply operational values. The actual fixture still decodes without mutation.
func TestConversionPremiseDecoding(t *testing.T) {
	raw := fixtureBytes(t, "expectations.json")
	before := append([]byte{}, raw...)
	in, issue := DecodeConversionInputs(raw)
	if issue != nil || !reflect.DeepEqual(in, conversionInputs(t, "expectations.json")) || !bytes.Equal(raw, before) {
		t.Fatal("valid premise decoding", issue)
	}
	var doc map[string]any
	if e := json.Unmarshal(raw, &doc); e != nil {
		t.Fatal(e)
	}
	doc["ignored_observation"] = []any{0.5, nil, true}
	if got, issue := DecodeConversionInputs(encoded(t, doc)); issue != nil || !reflect.DeepEqual(got, in) {
		t.Fatal("observation became operational", issue)
	}
	for _, bad := range [][]byte{nil, []byte("null"), []byte("[]"), []byte("{"), []byte{0xff}, bytes.Repeat([]byte(" "), MaxBytes+1)} {
		if got, issue := DecodeConversionInputs(bad); issue == nil || issue.Rule != "conversion_inputs_invalid" || issue.Exit != 2 || !reflect.DeepEqual(got, ConversionInputs{}) {
			t.Fatal("malformed premise admission", issue)
		}
	}
	for _, key := range []string{"SOURCE_REVISION", "ſource_revision"} {
		doc[key] = doc["source_revision"]
		if _, issue := DecodeConversionInputs(encoded(t, doc)); issue == nil || issue.Rule != "conversion_inputs_invalid" {
			t.Fatal("folded operational key admitted", issue)
		}
		delete(doc, key)
	}
	doc["source_revision"] = int64(1)
	if _, issue := DecodeConversionInputs(encoded(t, doc)); issue == nil || issue.Rule != "conversion_inputs_invalid" {
		t.Fatal("wrong operational type admitted", issue)
	}
	doc["source_revision"] = in.SourceRevision
	doc["reviewed_reservation_premises"] = []any{Object{"holder": "synthetic"}}
	if _, issue := DecodeConversionInputs(encoded(t, doc)); issue == nil || issue.Rule != "conversion_inputs_invalid" {
		t.Fatal("incomplete reservation admitted", issue)
	}
}
