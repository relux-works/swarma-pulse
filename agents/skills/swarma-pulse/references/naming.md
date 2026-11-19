# New agent names

The owner selected this convention on 2026-10-30 in the Pulse naming amendment
`docs/specs/identity-names-v2.md`. Keepers and orchestrators use it when choosing
new names for an admitted product and project.

- Full name `<role>-<star>.<project>.<product>.<zone>`.
- Matrix ID `@<project>.<role>-<star>:<zone>`.
- Role codes `keeper`, `orch`, `dev`, `tester`, `arch`, `reviewer`,
  `researcher`, and `service` for a separately admitted service.
- The zone comes from the selected Matrix connection's verified server name.
  A host, harness, or model belongs to the execution record.

For example, the tuple for an orchestrator in project `beacon` of product
`aurora` gives the full name `orch-mira.beacon.aurora.example.org` and Matrix ID
`@beacon.orch-mira:example.org`.

Use a canonical lowercase ASCII astronomical star slug. The stable logical
member holds the reservation. Check the admitted registry for the zone-wide
project owner, star uniqueness within the zone, product, and primary role,
and the complete Matrix ID. Preserve retired reservations and stop on a
collision. Limit product, project, and star slugs to 1–40 bytes. The Matrix
localpart is at most 64 bytes, each DNS label is at most 63 bytes, and the full
name is at most 253 bytes. Keep a syntactic name proposal provisional until the
owning reservation route verifies the tuple.

Existing accounts retain their exact Matrix IDs and history. A runtime change
preserves the logical member and its obligations. A change that requires a new
Matrix ID follows an explicitly admitted account cutover with preserved history
and custody. Card declarations carry desired state. Account registration,
launch and liveness each have their own required evidence.

The chosen naming rule for new agents is in force. Compact name resolution,
the proposed successor discriminator and migration remain separate implementation
contracts. For a current Pulse query, use the exact existing card scheme and
localpart supported by the reader, including `legacy-exact`, `domain-star-v1`,
or the v3 binding equation as applicable.
