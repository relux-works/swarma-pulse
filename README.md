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

None yet. Scratch, logs and worktrees go to `.temp/`, which git ignores.
