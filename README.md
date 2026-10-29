# swarma-pulse

Swarma pulse owns three things for the swarm platform:

- the product card format: one card per product, kept in that product's `<product>-host` project. It lists the
  product's projects, remote or local relative to the host, and who works on each;
- the identity scheme for members: a Matrix-domain prefix, then the product, project, role and a stable member
  suffix, unique within a role;
- the pulse query: each member asks pulse for its own projection of the card, meaning its peers, its subordinates
  and its superiors.

Status: project started 2026-10-29. Nothing is built yet. The first unit is a specification of the card format,
the identity scheme and the pulse query, with the a2a integration they need.

Owner decision (2026-10-29): workers are stable members that do not depend on any single story. Every member's
identity follows the scheme above.

## Tools

No implementation toolchain is selected yet. Documentation checks use `git diff --check`; candidate signature
verification uses `git verify-commit HEAD` and `git log -1 --show-signature`.

Coordination entry points are `a2a inbox`, `a2a bind status`, `a2a post ROOM --topic TOPIC --to RECIPIENT TEXT`,
`a2a attach ROOM --topic TOPIC FILE` and `a2a read ROOM --topic TOPIC`. Full inbox/readback logs, task evidence and
scratch go to ignored `.temp/` with numbered filenames. Private operational receipts stay outside product files.

## Proposed specification

[Product cards, stable identities and Pulse — v1](docs/specs/product-card-identity-pulse-v1.md) proposes the card upgrade,
full-domain star identities, a read-only member projection and the migration to `gr.relux.works`. It is a review
candidate. No implementation or migration has been built or activated. Pending external inputs and decisions are in
[UNRESOLVED_QUESTIONS.md](UNRESOLVED_QUESTIONS.md).

The handover baseline remains FORMAT v3.1 (`product-card/v3`). The proposed `product-card/v4` and `pulse` interface are
not shipped commands. Product-specific cards belong in their declared host project; swarm uses `swarma-host`.
