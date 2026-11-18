package pulse

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// This finite source admission check covers the environment decoder/projection
// files. It is neither syscall tracing nor a sandbox for dependency machine code.
func environmentSourceAdmission(raw []byte) string {
	f, err := parser.ParseFile(token.NewFileSet(), "projection.go", raw, 0)
	if err != nil {
		return "syntax_error"
	}
	imports := map[string]bool{}
	for _, path := range strings.Fields("crypto/sha256 encoding/hex encoding/json errors path/filepath strings time unicode unicode/utf8 sort github.com/relux-works/swarma-pulse/internal/cardv4") {
		imports[path] = true
	}
	aliases := map[string]string{}
	for _, im := range f.Imports {
		path, err := strconv.Unquote(im.Path.Value)
		if err != nil {
			return "syntax_error"
		}
		if !imports[path] || im.Name != nil && (im.Name.Name == "." || im.Name.Name == "_") {
			return "forbidden_source_effect"
		}
		parts := strings.Split(path, "/")
		name := parts[len(parts)-1]
		if im.Name != nil {
			name = im.Name.Name
		}
		aliases[name] = path
	}
	apis := map[string]map[string]bool{
		"time": {"Time": true, "RFC3339Nano": true},
		"github.com/relux-works/swarma-pulse/internal/cardv4": {"Parse": true, "ParseGovernanceSnapshot": true, "EnvironmentRelative": true, "CheckEnvironmentExtension": true, "DeclaredMXID": true, "ValidateCompanyCore": true, "Validate": true, "Options": true, "ParseLocalSnapshot": true},
	}
	verdict := ""
	ast.Inspect(f, func(n ast.Node) bool {
		s, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		for _, name := range []string{"Open", "OpenRoot", "ReadFile", "WriteFile", "OpenFile", "Start", "Run", "Dial", "Getenv"} {
			if s.Sel.Name == name {
				verdict = "forbidden_source_effect"
			}
		}
		if id, ok := s.X.(*ast.Ident); ok {
			if allow, constrained := apis[aliases[id.Name]]; constrained && !allow[s.Sel.Name] {
				verdict = "forbidden_source_effect"
			}
		}
		return true
	})
	return verdict
}

// A planted target-path read in the actual query source is rejected by the
// intended source-effect control, with the unchanged source as a legal neighbor.
func TestEnvironmentProjectionEffectAdmission(t *testing.T) {
	for _, path := range []string{"query.go", "environment.go", "environment_source.go", "governance.go", "company.go"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if verdict := environmentSourceAdmission(raw); verdict != "" {
			t.Fatal(path, verdict)
		}
		if path == "query.go" {
			plant := strings.Replace(string(raw), "import (", "import (\n\"os\"", 1) + "\nfunc inertTargetRead(path string){ _,_ = os.ReadFile(path) }\n"
			if verdict := environmentSourceAdmission([]byte(plant)); verdict != "forbidden_source_effect" {
				t.Fatal("target read plant was not detected", verdict)
			}
		}
	}
}
