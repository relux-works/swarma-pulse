# swarma-pulse

Swarma-pulse defines product cards, stable member identities and the Pulse query for a member's peers, reports and
superiors. Product cards describe desired teams and project locations; they do not launch members or store live state.

[Specification v1, revision 2](docs/specs/product-card-identity-pulse-v1.md) covers the proposed card upgrade,
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
implementation. Candidate signature inspection uses `git verify-commit HEAD` and `git log -1 --show-signature`.
No build command is available. Temporary local outputs, when needed, belong in ignored `.temp/`.
