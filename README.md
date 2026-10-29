# swarma-pulse

Swarma-pulse defines product cards, stable member identities and the Pulse query for a member's peers, reports and
superiors. Product cards describe desired teams and project locations; they do not launch members or store live state.

The [specification](docs/specs/product-card-identity-pulse-v1.md), [v4 format](docs/format-v4.md) and
[synthetic fixtures](docs/fixtures/v4/full/README.md) define full-domain star identities, preservation and migration.
The Go read-only validator implements static v4 validation and the v3.1 source-reading rules. Pulse, conversion,
publication, account migration and service/session launch are separate operations and are not supplied by this command.
[Open inputs](UNRESOLVED_QUESTIONS.md) describe their remaining prerequisites.

Members are stable across stories. Their names use a full Matrix domain, product, project, primary role and readable
star suffix. Existing chat identities retain their exact legacy mapping until an explicit per-member cutover.
Product-specific cards belong in their declared host project; the swarm host project is `swarma-host`.

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
Refusal writes one safe rule/indexed-location diagnostic to stderr and no stdout. See [validator boundaries and tests](docs/validator.md).

## Tools

| Tool | Exact entry point | Output |
| --- | --- | --- |
| Go build | `go build -o .temp/bin/card-v4-validate ./cmd/card-v4-validate` | Ignored `.temp/bin/` executable |
| Validator | `.temp/bin/card-v4-validate [options] card.yaml` | Verdict on stdout, safe refusal on stderr |
| Go tests | `go test -count=1 ./...` | All selected test lanes; isolated temporary fixtures |
| Race checks | `go test -race -count=1 ./...` | Race-enabled results for both packages |
| Go vet | `go vet ./...` | Static diagnostics |
| Coverage | `go test -coverprofile=.temp/coverage.out ./...` | `.temp/coverage.out`; inspect with `go tool cover -func=.temp/coverage.out` |
| Git | `git diff --check`; `git verify-commit HEAD` | Whitespace/signature diagnostics |

The optional v3 integration lane takes an explicit source without copying its values:
`CARD_V3_INTEGRATION_INPUT=/absolute/path/product-card.yaml go test -count=1 ./internal/cardv4 -run TestV3SuppliedCard`.
That lane checks the recorded 9-member, 8-project, 4-profile baseline; it is skipped when no input is selected.
