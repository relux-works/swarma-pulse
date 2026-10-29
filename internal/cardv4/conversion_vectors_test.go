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

// This inert recorder has no transport, endpoint or credential capability.
// Its known-violation plant is separate from the actual conversion fault.
type registrationTripwire struct{ attempts int }

func (r *registrationTripwire) attempt() *Issue {
	r.attempts++
	return fail("forbidden_registration_attempt", "$")
}

func executeConversionVector(t *testing.T, spec Object) *Issue {
	t.Helper()
	tripwire := &registrationTripwire{}
	if spec["operation"] == "registration-tripwire-self-check" {
		if number(spec["synthetic_attempts"]) == 1 {
			return tripwire.attempt()
		}
		if tripwire.attempts != 0 {
			t.Fatal("tripwire positive")
		}
		return nil
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
	complete, issue := convert(raw, in, checkpoint)
	if issue != nil {
		if len(complete) != 0 || completed != 2 || !reflect.DeepEqual(before, inventory(t, rootPath)) || tripwire.attempts != 0 {
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
	if tripwire.attempts != 0 {
		t.Fatal("registration attempted")
	}
	return nil
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
