# swarma-pulse

Swarma-pulse defines product cards, stable member identities and the Pulse query for a member's peers, reports and
superiors. Product cards describe desired teams and project locations; they do not launch members or store live state.

[Specification v1](docs/specs/product-card-identity-pulse-v1.md) covers the proposed card upgrade,
full-domain star identities, read-only projections and migration to `gr.relux.works`.
[Conformance vectors](docs/specs/conformance-v1.json) record required positive and negative examples.
[Open contract inputs](UNRESOLVED_QUESTIONS.md) identify unresolved technical choices and prerequisites.

Status: specification proposal only. No validator for `product-card/v4`, Pulse implementation or migration tooling is
provided. The compatibility baseline remains FORMAT v3.1 (`product-card/v3`). The `pulse` interface is proposed, not a
shipped command. Product-specific cards belong in their declared host project; the swarm host project is `swarma-host`.

Members are stable across stories. Their names use a full Matrix domain, product, project, stable role and readable
star suffix. Existing identities retain their exact legacy mapping until an explicit per-member cutover.

## Tools

Documentation whitespace can be checked with `git diff --check`. Conformance data can be parsed with
`python3 -m json.tool docs/specs/conformance-v1.json`; it prints the formatted fixture to stdout and exercises no product
implementation. Full card fixture syntax can be checked with
`python3 -m json.tool docs/fixtures/v4/full/source-v3.json`. With the compatibility repository alongside this one, run
`ruby ../product-card/validate.rb docs/fixtures/v4/full/source-v3.json` using Ruby 3.1 or newer; omit layout checks because
fixture filesystem roots are symbolic. Candidate signature inspection uses `git verify-commit HEAD` and `git log -1 --show-signature`.
No build command is available. Temporary local outputs, when needed, belong in ignored `.temp/`.

The [v4 format outline](docs/format-v4-outline.md) and its fixture requirements describe the next bounded design work.
They are not a complete schema or an available validator/converter. Format work proceeds independently of the first gr
account pilot through separately qualified existing a2a primitives.

The [complete v4 format candidate](docs/format-v4.md) and [full synthetic fixtures](docs/fixtures/v4/full/README.md) are
available for design review. Neither has implementation acceptance; no v4 validator/converter is supplied. The fixtures
can be parsed as JSON, and source-v3.json can be checked with the existing v3 reader without layout checks. The fixture
manifest pins input bytes and expected mappings; no account or runtime qualification follows from those checks.
