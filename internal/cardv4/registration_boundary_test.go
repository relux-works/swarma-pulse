package cardv4

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// This test-only source/API admission boundary wraps the exercised conversion
// and publication. It records denied capabilities before allowing execution;
// it is not syscall tracing or a sandbox for dependency machine code.
type registrationAdmission struct{ attempts int }

func (r *registrationAdmission) run(t *testing.T, plant bool, operation func() *Issue) *Issue {
	t.Helper()
	accepted := true
	for _, root := range []string{".", "../cardpublish", "../../cmd/card-v4-convert"} {
		entries, e := os.ReadDir(root)
		if e != nil {
			t.Fatal(e)
		}
		files := 0
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			files++
			raw := mustRead(t, filepath.Join(root, entry.Name()))
			if plant && root == "../cardpublish" && entry.Name() == "publication.go" {
				// Add one inert forbidden call to the actual scanned publisher.
				// It has no implementation, endpoint or credential capability.
				raw = append(raw, []byte("\nfunc inertRegistrationPlant(r interface{ Register() }) { r.Register() }\n")...)
			}
			if !r.admit(raw, root == ".") {
				accepted = false
			}
		}
		if files == 0 {
			t.Fatal("source admission skipped an operation package", root)
		}
	}
	if r.attempts != 0 {
		return fail("forbidden_registration_attempt", "$")
	}
	if !accepted {
		return fail("conversion_effect_admission", "$")
	}
	return operation()
}

func (r *registrationAdmission) admit(raw []byte, core bool) bool {
	tree, e := parser.ParseFile(token.NewFileSet(), "operation.go", raw, 0)
	if e != nil {
		return false
	}
	accepted := true
	// Core uses its existing pure-source admission checker. The only admitted
	// local imports in the adapters are the two packages scanned by run above.
	allow := words("errors io io/fs os path/filepath encoding/json flag fmt github.com/relux-works/swarma-pulse/internal/cardv4 github.com/relux-works/swarma-pulse/internal/cardpublish")
	if core && !admittedSource(raw, false, &processSpy{}) {
		accepted = false
	}
	for _, im := range tree.Imports {
		path, e := strconv.Unquote(im.Path.Value)
		if e != nil {
			return false
		}
		if path == "net" || strings.HasPrefix(path, "net/") && path != "net/url" {
			r.attempts++
			accepted = false
		}
		if !core && (!allow[path] || im.Name != nil && (im.Name.Name == "." || im.Name.Name == "_")) {
			accepted = false
		}
	}
	ast.Inspect(tree, func(n ast.Node) bool {
		if selector, ok := n.(*ast.SelectorExpr); ok && (selector.Sel.Name == "Register" || selector.Sel.Name == "RegistrationAttempt") {
			r.attempts++
			accepted = false
		}
		return true
	})
	return accepted
}
