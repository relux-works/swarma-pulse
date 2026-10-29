package cardv4

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/swarma-pulse/internal/cardpublish"
)

// The same source/API gate admits complete conversion/publication and denies
// one inert registration plant before either call, preserving the prior tree.
func executeConversionVector(t *testing.T, spec Object) *Issue {
	t.Helper()
	selfCheck := spec["operation"] == "registration-tripwire-self-check"
	synthetic, expectedAttempts := 0, 0
	if selfCheck {
		synthetic = number(spec["synthetic_attempts"])
		if synthetic != 0 && synthetic != 1 {
			t.Fatal("unsupported synthetic registration count")
		}
		if synthetic == 1 && spec["transport"] != "inert-synthetic-recorder-no-endpoint" {
			t.Fatal("self-check requires inert transport premise")
		}
		expectedAttempts = synthetic
		// Both self-check neighbors use the real conversion/publication setup;
		// only an in-memory forbidden API plant differs in the admission input.
		for _, v := range list(fixture(t, "vectors-mixed.json")["cases"]) {
			if c := obj(v); c["id"] == "mixed-no-registration" {
				spec = obj(c["operation"])
				break
			}
		}
		if spec["operation"] != "convert-atomic" {
			t.Fatal("missing conversion control setup")
		}
	}
	if armed, present := spec["registration_tripwire"]; present {
		if armed != "armed" {
			t.Fatal("registration tripwire not armed")
		}
		if _, present := spec["expected_registration_attempts"]; !present {
			t.Fatal("missing registration count oracle")
		}
		if !selfCheck {
			expectedAttempts = number(spec["expected_registration_attempts"])
		}
	}
	rootPath := t.TempDir()
	sourcePath := strings.TrimPrefix(str(spec["source_materialized_path"]), "fixture-root/")
	destination := strings.TrimPrefix(str(spec["destination"]), "fixture-root/")
	if !filepath.IsLocal(sourcePath) || !filepath.IsLocal(destination) || sourcePath == "" || destination == "" {
		t.Fatal("conversion vector paths")
	}
	for _, path := range []string{sourcePath, destination} {
		if e := os.MkdirAll(filepath.Join(rootPath, filepath.Dir(path)), 0700); e != nil {
			t.Fatal(e)
		}
	}
	raw := fixtureBytes(t, str(spec["card"]))
	if e := os.WriteFile(filepath.Join(rootPath, sourcePath), raw, 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(rootPath, destination), []byte(str(spec["destination_initial_utf8"])), 0600); e != nil {
		t.Fatal(e)
	}
	sourceInfo, e := os.Stat(filepath.Join(rootPath, sourcePath))
	if e != nil {
		t.Fatal(e)
	}
	before := inventory(t, rootPath)
	in := conversionInputs(t, str(spec["conversion_inputs"]))
	completed := 0
	var checkpoint func(int) *Issue
	if fault := obj(spec["fault"]); fault != nil {
		if fault["point"] != "after-member-conversion-before-publication" || number(fault["completed_members"]) != 2 {
			t.Fatal("unsupported fault point")
		}
		checkpoint = func(n int) *Issue {
			completed = n
			if n == 2 {
				return fail("conversion_interrupted", "$.members")
			}
			return nil
		}
	}
	// Admission is connected to this operation: the same recorder scans the
	// actual converter/core/publisher sources before either production call.
	// A plant is parsed and denied, never compiled, called or transported.
	boundary := &registrationAdmission{}
	entered := false
	plant := ""
	if synthetic == 1 {
		plant = "../cardpublish/publication.go"
	}
	issue := boundary.run(t, plant, func() *Issue {
		entered = true
		complete, issue := convert(raw, in, checkpoint)
		if issue != nil {
			if len(complete) != 0 || completed != 2 || !reflect.DeepEqual(before, inventory(t, rootPath)) {
				t.Fatal("partial conversion effect")
			}
			return issue
		}
		expected := fixture(t, str(spec["expected_output"]))
		canonical, e := json.MarshalIndent(expected, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		canonical = append(canonical, '\n')
		if !bytes.Equal(complete, canonical) {
			t.Fatal("complete canonical expected bytes differ")
		}
		root, e := os.OpenRoot(rootPath)
		if e != nil {
			t.Fatal(e)
		}
		defer root.Close()
		result, pubIssue := cardpublish.Publish(root, destination, sourceInfo, complete)
		if pubIssue != nil || !result.Published || !result.Durable || result.TemporaryRemains {
			t.Fatal("actual publication", result, pubIssue)
		}
		// Only this complete destination replacement is allowed; input and directory
		// inventories (including absence of temporary files) remain otherwise exact.
		sum := sha256.Sum256(complete)
		before[destination] = hex.EncodeToString(sum[:])
		if !reflect.DeepEqual(before, inventory(t, rootPath)) {
			t.Fatal("unexpected publication effect")
		}
		return nil
	})
	t.Logf("source/API admission: registration attempts=%d, operation entered=%t", boundary.attempts, entered)
	if boundary.attempts != expectedAttempts {
		t.Fatalf("registration attempts: got %d, want %d", boundary.attempts, expectedAttempts)
	}
	if synthetic == 1 {
		if entered || issue == nil || issue.Rule != "forbidden_registration_attempt" || issue.Location != "$" || issue.Exit != 1 {
			t.Fatal("inert registration plant was not denied before execution", issue)
		}
		if !reflect.DeepEqual(before, inventory(t, rootPath)) {
			t.Fatal("registration refusal changed source/destination or left temporary output")
		}
	} else if !entered {
		t.Fatal("conversion/publication operation never admitted", issue)
	}
	return issue
}

// A fault after two real member transformations returns no document; a normal
// neighbor traverses the same conversion implementation to a complete result.
func TestConversionMemberFault(t *testing.T) {
	raw, in := fixtureBytes(t, "source-v3.json"), conversionInputs(t, "expectations.json")
	if out, e := Convert(raw, in); e != nil || len(out) == 0 {
		t.Fatal(e)
	}
	calls := 0
	out, e := convert(raw, in, func(n int) *Issue {
		calls = n
		if n == 2 {
			return fail("conversion_interrupted", "$.members")
		}
		return nil
	})
	if e == nil || e.Rule != "conversion_interrupted" || calls != 2 || len(out) != 0 {
		t.Fatal("conversion interruption contract", e)
	}
}
