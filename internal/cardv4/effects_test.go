package cardv4

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// The production boundary admits no process API at all. This recorder observes
// denied process imports/calls in the AST admission scan; it is not an OS sandbox
// or a claim to trace arbitrary precompiled dependency machine code.
type processSpy struct{ attempts int }

func (s *processSpy) deny() { s.attempts++ }
func admittedSource(src []byte, cli bool, spy *processSpy) bool {
	tree, e := parser.ParseFile(token.NewFileSet(), "candidate.go", src, 0)
	if e != nil {
		return false
	}
	pure := words("bytes crypto/sha256 encoding/hex encoding/json errors fmt io io/fs net/url path/filepath reflect regexp sort strconv strings unicode unicode/utf8 go.yaml.in/yaml/v3")
	// These dependencies expose only the exact public-decoder/type entry
	// points used by the governance contract, not signer or ambient-clock APIs.
	publicAPIs := map[string]map[string]bool{
		"crypto/elliptic": words("P256 UnmarshalCompressed MarshalCompressed"),
		"encoding/base64": words("RawURLEncoding"),
		"time":            words("Time Parse"),
		"github.com/decred/dcrd/dcrec/secp256k1/v4":     words("ParsePubKey"),
		"github.com/oasisprotocol/curve25519-voi/curve": words("CompressedEdwardsY EdwardsPoint"),
	}
	for path := range publicAPIs {
		pure[path] = true
	}
	if cli {
		for _, p := range []string{"os", "flag", "github.com/relux-works/swarma-pulse/internal/cardv4"} {
			pure[p] = true
		}
	}
	aliases := map[string]string{}
	accepted := true
	for _, i := range tree.Imports {
		p, e := strconv.Unquote(i.Path.Value)
		if e != nil {
			return false
		}
		name := filepath.Base(p)
		if p == "github.com/decred/dcrd/dcrec/secp256k1/v4" {
			name = "secp256k1"
		}
		if i.Name != nil {
			name = i.Name.Name
		}
		aliases[name] = p
		if p == "os/exec" || p == "syscall" || p == "unsafe" || p == "plugin" {
			spy.deny()
		}
		if !pure[p] || name == "." || name == "_" {
			accepted = false
		}
	}
	readOS := words("Open OpenRoot Root Exit Args Stdout Stderr")
	writeMethods := words("Write WriteFile WriteString Remove RemoveAll Rename Mkdir MkdirAll Symlink Link Chmod Chown Truncate Create OpenFile StartProcess")
	ast.Inspect(tree, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if writeMethods[sel.Sel.Name] {
			accepted = false
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		p := aliases[id.Name]
		if allowed, constrained := publicAPIs[p]; constrained && !allowed[sel.Sel.Name] {
			accepted = false
		}
		if !cli && p == "fmt" && strings.HasPrefix(sel.Sel.Name, "Print") {
			accepted = false
		}
		if p == "os" && !readOS[sel.Sel.Name] {
			accepted = false
			if sel.Sel.Name == "StartProcess" {
				spy.deny()
			}
		}
		return true
	})
	return accepted
}

// The extended AST boundary admits each required public entry point and
// detects planted private-key, ambient-clock and unlisted-module capabilities.
// It does not type-check or trace dependency machine code.
func TestPublicDecoderEffectAdmission(t *testing.T) {
	for _, tc := range []struct {
		path, alias, selector string
		valid                 bool
	}{
		{"crypto/elliptic", "elliptic", "P256", true},
		{"crypto/elliptic", "elliptic", "UnmarshalCompressed", true},
		{"crypto/elliptic", "elliptic", "MarshalCompressed", true},
		{"encoding/base64", "base64", "RawURLEncoding", true},
		{"time", "time", "Time", true},
		{"time", "time", "Parse", true},
		{"github.com/decred/dcrd/dcrec/secp256k1/v4", "secp256k1", "ParsePubKey", true},
		{"github.com/decred/dcrd/dcrec/secp256k1/v4", "decoder", "ParsePubKey", true},
		{"github.com/oasisprotocol/curve25519-voi/curve", "curve", "CompressedEdwardsY", true},
		{"github.com/oasisprotocol/curve25519-voi/curve", "curve", "EdwardsPoint", true},
		{"crypto/elliptic", "elliptic", "GenerateKey", false},
		{"time", "time", "Now", false},
		{"time", "time", "AfterFunc", false},
		{"github.com/decred/dcrd/dcrec/secp256k1/v4", "secp256k1", "PrivKeyFromBytes", false},
		{"github.com/oasisprotocol/curve25519-voi/curve", "curve", "NewEdwardsBasepointTable", false},
		{"crypto/ed25519", "ed25519", "GenerateKey", false},
	} {
		t.Run(tc.alias+"/"+tc.selector, func(t *testing.T) {
			source := "package p\nimport " + tc.alias + " " + strconv.Quote(tc.path) + "\nvar _ = " + tc.alias + "." + tc.selector + "\n"
			if got := admittedSource([]byte(source), false, &processSpy{}); got != tc.valid {
				t.Fatalf("want admitted=%v, got %v", tc.valid, got)
			}
		})
	}
}

// All production units retain the no-write/no-network/no-process API surface.
// A planted process import/call must trip the deny-all process recorder; a
// planted write must fail admission, while a pure function passes unchanged.
func TestProductionEffectBoundary(t *testing.T) {
	roots := []string{".", "../../cmd/card-v4-validate", "../../cmd/card-v4-check-change"}
	for _, root := range roots {
		entries, e := os.ReadDir(root)
		if e != nil {
			t.Fatal(e)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			spy := &processSpy{}
			if !admittedSource(mustRead(t, filepath.Join(root, entry.Name())), root != ".", spy) || spy.attempts != 0 {
				t.Fatal("production effect boundary", entry.Name())
			}
		}
	}
	spy := &processSpy{}
	if !admittedSource([]byte("package p\nfunc f() int { return 1 }"), false, spy) || spy.attempts != 0 {
		t.Fatal("pure control rejected")
	}
	if admittedSource([]byte("package p\nimport \"os/exec\"\nfunc f(){exec.Command(\"inert\")}"), false, spy) || spy.attempts != 1 {
		t.Fatal("process violation undetected")
	}
	if admittedSource([]byte("package p\nimport \"os\"\nfunc f(){os.WriteFile(\"inert\",nil,0600)}"), true, &processSpy{}) {
		t.Fatal("write violation undetected")
	}
}

// The bounded byte-inventory instrument detects changed bytes, extra files and
// changed links; unchanged files remain equal. All controls are isolated.
func TestByteInventoryControl(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input")
	if e := os.WriteFile(path, []byte("before"), 0600); e != nil {
		t.Fatal(e)
	}
	before := inventory(t, dir)
	if !reflect.DeepEqual(before, inventory(t, dir)) {
		t.Fatal("stable inventory")
	}
	if e := os.WriteFile(path, []byte("after"), 0600); e != nil {
		t.Fatal(e)
	}
	if reflect.DeepEqual(before, inventory(t, dir)) {
		t.Fatal("changed bytes undetected")
	}
	if e := os.WriteFile(path, []byte("before"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(dir, "partial"), nil, 0600); e != nil {
		t.Fatal(e)
	}
	if reflect.DeepEqual(before, inventory(t, dir)) {
		t.Fatal("extra output undetected")
	}
}

// Pure validation and preservation never mutate supplied maps/bytes. The temp
// inventory assertion additionally covers all layout fixture operations.
func TestInputImmutability(t *testing.T) {
	o := fixtureOptions(t)
	raw := fixtureBytes(t, "converted-v4.json")
	prior := append([]byte{}, raw...)
	sourceCopies := make([][]byte, len(o.Sources))
	for i, s := range o.Sources {
		sourceCopies[i] = append([]byte{}, s.Bytes...)
	}
	if e := Validate(raw, o); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(raw, prior) {
		t.Fatal("input mutation")
	}
	for i, s := range o.Sources {
		if !reflect.DeepEqual(s.Bytes, sourceCopies[i]) {
			t.Fatal("source mutation")
		}
	}
	c := fixture(t, "converted-v4.json")
	before := encoded(t, c)
	if e := VerifyPilotPreservation(c, fixture(t, "pilot-v4.json"), "aurora/beacon_orch"); e != nil {
		t.Fatal(e)
	}
	if string(before) != string(encoded(t, c)) {
		t.Fatal("prior mutated")
	}
}
