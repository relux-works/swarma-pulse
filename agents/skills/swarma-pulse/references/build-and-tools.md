# Build and tools

Use these commands from an authorized Pulse source checkout with its `go.mod`,
source files and synthetic fixtures. This skill's documentation package does not
include those files or any built executable; its installation does not supply a
product checkout, select a real input or authorize a build, write or query. Use
an existing admitted binary when building is unnecessary. Synthetic examples
remain separate from actual cards and independently admitted custody.

## Build and run

Use Go 1.25.12 or a newer fixed toolchain (the Go 1.26 branch requires 1.26.5 or newer). This includes the rooted-filesystem fix for
[CVE-2026-39822](https://github.com/golang/go/issues/79005); the module directive makes an older local-only compiler
refuse the build. Native validation is tested on macOS arm64 with Go 1.25.12. The only module dependency is
[go.yaml.in/yaml/v3 v3.0.5](https://pkg.go.dev/go.yaml.in/yaml/v3@v3.0.5), the maintained YAML v3 Node API, pinned in
go.mod/go.sum. No language adapter, child process or network client is used by the validator.

```sh
mkdir -p .temp/bin
go build -o .temp/bin/card-v4-validate ./cmd/card-v4-validate
.temp/bin/card-v4-validate \
  --source-manifest docs/fixtures/v4/full/manifest.json \
  --reservations docs/fixtures/v4/full/expectations.json \
  docs/fixtures/v4/full/converted-v4.json
.temp/bin/card-v4-validate --version 3 docs/fixtures/v4/full/source-v3.json
```

The synthetic manifest and reservations are test premises, not real admissions. Actual inputs must be supplied through
their independently admitted custody route. The validator never fetches missing sources or allocates reservations.
An optional layout check needs both `--check-layout --root /admitted/product/root` and
`--card-directory relative/host-project`. The latter is relative to the admitted root, which is opened with Go's
rooted filesystem API. Local projects, task entries and profiles must exist there. Remote projects are not opened;
member clone/git_dir declarations are checked for containment without requiring those checkouts to exist.

Success prints `valid` and exits 0. A rule refusal exits 1; usage, encoding, parse and collection-shape failures exit 2.
Refusal writes one safe rule/indexed-location diagnostic to stderr and no stdout. See [validator boundaries and tests](validator.md).

For an isolated synthetic conversion example:

```sh
mkdir -p .temp/conversion/legacy-card .temp/conversion/aurora-host
cp docs/fixtures/v4/full/source-v3.json .temp/conversion/legacy-card/product-card.yaml
go build -o .temp/bin/card-v4-convert ./cmd/card-v4-convert
.temp/bin/card-v4-convert --root .temp/conversion \
  --source legacy-card/product-card.yaml \
  --inputs docs/fixtures/v4/full/expectations.json \
  --destination aurora-host/product-card.yaml
```

Success prints `converted`; the complete file is published only at the explicit destination. A leftover exclusive temp
refuses, never becomes an input. Exit 3 means the complete destination was published but directory durability could not
be confirmed; inspect the result before any retry. See [converter inputs and publication results](converter.md).

## Tools

| Tool | Exact entry point | Output |
| --- | --- | --- |
| Go build | `go build -o .temp/bin/card-v4-validate ./cmd/card-v4-validate` | Ignored `.temp/bin/` executable |
| Validator | `.temp/bin/card-v4-validate [options] card.yaml` | Verdict on stdout, safe refusal on stderr |
| Converter build | `go build -o .temp/bin/card-v4-convert ./cmd/card-v4-convert` | Ignored `.temp/bin/` executable |
| Converter | `.temp/bin/card-v4-convert --root ROOT --source SOURCE --inputs INPUTS --destination DESTINATION` | Explicit complete destination; fixed receipt or safe refusal |
| Change-check build | `go build -o .temp/bin/card-v4-check-change ./cmd/card-v4-check-change` | Ignored `.temp/bin/` executable |
| Change checker | `.temp/bin/card-v4-check-change --root ROOT --prior PRIOR --prior-sha256 HASH --candidate CANDIDATE --candidate-sha256 HASH --rollback ROLLBACK --rollback-sha256 HASH --pilot PRODUCT/MEMBER --source-manifest MANIFEST --reservations PREMISES` | Fixed receipt or safe refusal; no writes |
| Pulse build | `go build -o .temp/bin/pulse ./cmd/pulse` | Ignored `.temp/bin/` executable |
| Pulse query | `.temp/bin/pulse me --local-unsigned --card ABS --revision sha256:HASH --format json` | Self-only JSON or safe refusal; exact content pin and public `A2A_AGENT` context |
| Go tests | `go test -count=1 ./...` | All selected test lanes; isolated temporary fixtures |
| Race checks | `go test -race -count=1 ./...` | Race-enabled results for all packages |
| Go vet | `go vet ./...` | Static diagnostics |
| Coverage | `go test -coverprofile=.temp/coverage.out ./...` | `.temp/coverage.out`; inspect with `go tool cover -func=.temp/coverage.out` |
| Git | `git diff --check`; `git verify-commit HEAD` | Whitespace/signature diagnostics |

The optional v3 integration lane takes an explicit source without copying its values:
`CARD_V3_INTEGRATION_INPUT=/absolute/path/product-card.yaml go test -count=1 ./internal/cardv4 -run TestV3SuppliedCard`.
That lane checks the recorded 9-member, 8-project, 4-profile baseline; it is skipped when no input is selected.
