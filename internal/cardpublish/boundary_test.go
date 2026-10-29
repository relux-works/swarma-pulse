package cardpublish

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

// This is source admission and an inert forbidden-API recorder, not syscall
// tracing or an OS sandbox. No endpoint, account or credential is available.
type boundaryRecorder struct{ registration, process int }

func admits(raw []byte, cli bool, recorder *boundaryRecorder) bool {
	if strings.Contains(string(raw), "go:build") || strings.Contains(string(raw), "+build") {
		return false
	}
	tree, e := parser.ParseFile(token.NewFileSet(), "candidate.go", raw, 0)
	if e != nil {
		return false
	}
	allow := map[string]bool{"errors": true, "io": true, "io/fs": true, "os": true, "path/filepath": true}
	if cli {
		for _, p := range []string{"encoding/json", "flag", "fmt", "github.com/relux-works/swarma-pulse/internal/cardpublish", "github.com/relux-works/swarma-pulse/internal/cardv4"} {
			allow[p] = true
		}
	}
	names := map[string]string{}
	accepted := true
	for _, im := range tree.Imports {
		p, e := strconv.Unquote(im.Path.Value)
		if e != nil {
			return false
		}
		name := filepath.Base(p)
		if im.Name != nil {
			name = im.Name.Name
		}
		names[name] = p
		if !allow[p] || name == "." || name == "_" {
			accepted = false
		}
		if p == "os/exec" || p == "syscall" || p == "unsafe" {
			recorder.process++
		}
		if p == "net/http" || p == "net" {
			recorder.registration++
		}
	}
	allowedOS := map[string]bool{}
	for _, s := range []string{"Root", "File", "Open", "OpenRoot", "Exit", "Args", "Stdout", "Stderr", "O_WRONLY", "O_CREATE", "O_EXCL", "SameFile"} {
		allowedOS[s] = true
	}
	ast.Inspect(tree, func(n ast.Node) bool {
		if function, ok := n.(*ast.FuncDecl); ok && ast.IsExported(function.Name.Name) {
			if cli || (function.Name.Name != "Publish" && function.Name.Name != "Error") {
				accepted = false
			}
		}
		if field, ok := n.(*ast.Field); ok {
			for _, name := range field.Names {
				if ast.IsExported(name.Name) && strings.Contains(strings.ToLower(name.Name), "fault") {
					accepted = false
				}
			}
		}
		call, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if call.Sel.Name == "Register" || call.Sel.Name == "RegistrationAttempt" {
			recorder.registration++
			accepted = false
		}
		if call.Sel.Name == "StartProcess" {
			recorder.process++
			accepted = false
		}
		if cli {
			switch call.Sel.Name {
			case "Write", "WriteFile", "WriteString", "OpenFile", "Remove", "RemoveAll", "Rename", "Create", "Mkdir", "MkdirAll":
				accepted = false
			}
		}
		if id, ok := call.X.(*ast.Ident); ok && names[id.Name] == "os" && !allowedOS[call.Sel.Name] {
			accepted = false
		}
		return true
	})
	// Production has only these four flags; no flag can expose a fault seam.
	ast.Inspect(tree, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || (sel.Sel.Name != "String" && sel.Sel.Name != "Bool" && sel.Sel.Name != "Int") {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || id.Name != "flags" {
			return true
		}
		if len(call.Args) == 0 {
			accepted = false
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok {
			accepted = false
			return true
		}
		value, e := strconv.Unquote(lit.Value)
		if e != nil || (value != "source" && value != "inputs" && value != "root" && value != "destination") {
			accepted = false
		}
		return true
	})
	return accepted
}

// Only the publisher has filesystem-write capability. Every production file in
// both adapters passes admission; inert process/registration/env/fault plants
// must fail, alongside an admitted no-effect positive control.
func TestConverterEffectAndFaultAdmission(t *testing.T) {
	for _, root := range []string{".", "../../cmd/card-v4-convert"} {
		entries, e := os.ReadDir(root)
		if e != nil {
			t.Fatal(e)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			raw, e := os.ReadFile(filepath.Join(root, entry.Name()))
			if e != nil {
				t.Fatal(e)
			}
			r := &boundaryRecorder{}
			if !admits(raw, root != ".", r) || r.registration != 0 || r.process != 0 {
				t.Fatal("production effect/fault admission", entry.Name())
			}
		}
	}
	if !admits([]byte("package p;func f(){}"), false, &boundaryRecorder{}) {
		t.Fatal("positive admission")
	}
	for _, plant := range []string{
		"package p;import \"os\";func f(){os.Getenv(\"INERT_FAULT\")}",
		"//go:build fault\npackage p\n",
		"package p;type Options struct{ Fault bool }",
		"package p;func PublishWithFault(f func()){f()}",
		"package p;func f(){flags.String(\"fault\",\"\",\"\")}",
		"package p;import \"os\";func f(){os.WriteFile(\"inert\",nil,0600)}",
	} {
		if admits([]byte(plant), true, &boundaryRecorder{}) {
			t.Fatal("fault/write plant admitted")
		}
	}
	registration := &boundaryRecorder{}
	if admits([]byte("package p;func f(r interface{Register()}){r.Register()}"), false, registration) || registration.registration != 1 {
		t.Fatal("registration plant undetected")
	}
	process := &boundaryRecorder{}
	if admits([]byte("package p;import \"os/exec\";func f(){exec.Command(\"inert\")}"), false, process) || process.process != 1 {
		t.Fatal("process plant undetected")
	}
}

// The exported conversion/publication paths select nil/native operations, never
// a caller-controlled fault callback. AST checks operate on actual wrappers.
func TestPublicPathsUseNativeSeams(t *testing.T) {
	for _, tc := range []struct {
		path, name, callee string
		native             bool
	}{{"../cardv4/conversion.go", "Convert", "convert", false}, {"publication.go", "Publish", "publish", true}} {
		raw, e := os.ReadFile(tc.path)
		if e != nil {
			t.Fatal(e)
		}
		tree, e := parser.ParseFile(token.NewFileSet(), tc.path, raw, 0)
		if e != nil {
			t.Fatal(e)
		}
		found := false
		for _, d := range tree.Decls {
			f, ok := d.(*ast.FuncDecl)
			if !ok || f.Name.Name != tc.name {
				continue
			}
			ast.Inspect(f.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				name, ok := call.Fun.(*ast.Ident)
				if !ok || name.Name != tc.callee {
					return true
				}
				last := call.Args[len(call.Args)-1]
				if tc.native {
					literal, ok := last.(*ast.CompositeLit)
					if ok {
						kind, ok := literal.Type.(*ast.Ident)
						found = ok && kind.Name == "nativeOps"
					}
				} else {
					value, ok := last.(*ast.Ident)
					found = ok && value.Name == "nil"
				}
				return true
			})
		}
		if !found {
			t.Fatal("public fault seam exposed", tc.name)
		}
	}
}
