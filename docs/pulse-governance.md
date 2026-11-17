# Pulse governance view

The explicit governance view reads one pinned product card and returns declared owner public records, the meta-keeper reference and the agent caller's own summary. It does not authenticate keys, broaden the hierarchy or fetch another source.

## Invoke and input

Use the supported fixed Go toolchain and existing build instructions. The explicit query is:

    pulse me --local-unsigned --view governance --card /absolute/product-card.yaml --revision sha256:LOWERCASE64HEX --format json

There is no caller, clock, key, policy or member override. The current A2A_AGENT localpart must match exactly one chat-admitted agent declaration. A human contact cannot become a caller. Existing separately admitted nonconversational service callers retain their constraints. Blank, invalid, unknown and ambiguous contexts refuse. A full MXID is not a localpart selector.

Input is product-card/v3, product-card/v4 or [product-card/v5](format-v5.md). Company cards are unsupported. The complete selected document is validated before output, including unselected principals/keys. V4/v5 local declarations are checked while source attestations and external star reservations remain unverified. Legacy v3/v4 output always has undeclared governance; legacy extension names are not interpreted as v5 fields.

The same UID/effective-UID, invoking-owner regular-file, no-write-by-others, safe nonsymlink path and1MiB bounds from the [query contract](../agents/skills/swarma-pulse/references/query-contract.md) apply. All selectors are explicit. Discovery source/root selectors refuse for governance. No referenced path, public-key store, network, account or runtime is accessed. The query performs no write.

Existing selectors keep their v1-v4 schemas and acceptance. They reject v5 cards. Only the explicit governance selector accepts v5 in this first slice; no old decoder silently receives the new shape.

## Exact closed output

One JSON document followed by one newline, schema pulse-view/v5, with exactly17 top-level keys. All objects below are closed. No input-controlled key becomes a diagnostic label.

| Key | Exact value/type |
| --- | --- |
| schema | literal pulse-view/v5 |
| mode | literal local-unsigned |
| product | selected product id |
| member | qualified product/member key |
| caller | Caller |
| known_limit | literal caller not cryptographically verified; single-user local trust |
| source | Source |
| query_observed_at | existing UTC RFC3339Nano query time |
| self | Self |
| peers | empty array |
| direct_reports | empty array |
| descendants | empty array |
| superiors | empty array |
| projects | caller's zero-or-one declared Project summary |
| hierarchy_status | literal self-only; hierarchy not disclosed |
| observations | Observations |
| governance | GovernanceView |

These empty graph lists mean undisclosed. Product governance visibility grants no authority or cross-product view.

| Object | Exhaustive required fields | Optional fields |
| --- | --- | --- |
| Caller | agent: exact local context; cryptographically_verified: false; trust: single-user local trust | none |
| Source | schema: selected card discriminator; revision: explicit sha256 pin; sha256: exact raw digest; card_root: clean selected card parent; freshness: unknown; observed_at: null | validation, present exactly for v4/v5 |
| Self | key: qualified member; kind: agent; role: title; responsibility: mechanical Role; scope: product/project/member; desired_state: declared State; resolved_name: full declared MXID; execution_binding: ExecutionBinding; profile: ProfileView or service-only null; project: qualified key or null; project_state: declared/undeclared | none |
| ExecutionBinding | key: binding map key; operator: text; host: text; connection: connection key | none |
| ProfileView | requirement: ProfileRequirement; selected: ProfileSelection | none |
| ProfileRequirement | context: text; compatible: version range | none |
| ProfileSelection | version: selected version; source_rev: selected revision; lock_sha256: selected digest | none |
| Project | key: qualified project; repo: declared repository; desired_state: State; location: LocationView | none |
| Local LocationView | kind: local; path: declared card-relative path | none |
| Remote LocationView | kind: remote; operator: text; execution_host: text; project_locator: ref | checkout_path |
| Observations | runtime: Unknown; registration: Unknown; installed_profile: Unknown; assignment: Unknown | none |
| Unknown | state: unknown | none |
| GovernanceView | state: declared/undeclared; owners: list OwnerView; meta_keeper: MetaKeeper or null | none |
| OwnerView | id: local principal id; kind: human; name: declared title; contacts: list Contact; keys: list PublicKey | none |
| MetaKeeper | reference: QualifiedReference; resolution: resolved/external_unresolved; member: MemberDeclaration or null | none |
| QualifiedReference | product: product id; member: complete logical member id | none |
| MemberDeclaration | key: qualified key; kind: human/agent; role: title; responsibility: Role; scope: Scope; desired_state: State; project: qualified project key; principal: human principal id or agent-only null | none |

Contact/PublicKey are exactly the closed public-only input descriptors in the v5 format. No vault handle or private path is added. Profile is null only for a valid nonconversational service. For legacy self, kind=agent and the original mechanical role is copied to role and responsibility. That read-only output normalization creates no written card.

Source.validation is exactly local declarations only; source attestations and star reservations unverified for v4/v5. V3 omits that field. Query time is not source freshness. The unknown observations are constant, not process/account probes.

With governance absent or legacy input, state=undeclared, owners=[] and meta_keeper=null. This means metadata not supplied, not authenticated absence of owners. Malformed supplied metadata refuses instead of becoming undeclared. With valid v5 governance, state=declared, owners contains only referenced principals and meta_keeper is nonnull. An owner's keys=[] means public material not supplied.

Owners sort by principal id using raw string order. Contact order is preserved; keys sort by purpose then key_id. A local meta-keeper reference qualifies with this product and resolves to its member declaration. An exact declared External reference is external_unresolved with member=null. No additional snapshot is opened and no external summary is guessed.

## Trust and refusal

Public keys remain declarations. Canonical point parsing is not private-key possession, verified owner identity, chain/revocation verification or current signing authority. No first owner/key is selected. A BIP-340 identity declaration does not become a grant issuer. Changing the card does not change any protected issuer pin.

Selector refusal exits2 before file access. Other input/caller/card refusals exit1 with one safe RULE name and indexed location, empty stdout and no mutation. Invalid v5 shapes, points, key refs, times or forbidden facts collapse at the query boundary to the existing safe card_invalid class. Exact rule details belong to parser tests. A transport output_write error is distinct and may leave partial stdout bytes. No input value, key bytes, path or parser exception appears in diagnostics.

Retain valid nearby controls for legacy cards, agent/human/service shapes, two owners/empty keys, unresolved meta keeper and all17 output keys. Negative cases cover old-selector v5 refusal, human caller impersonation, title-derived privileges, bad canonical point/encoding/purpose/digest, duplicate/dangling references, invalid unselected key, source pin/path/root/UID and the live-status scanner. These claims must be backed by executed tests before delivery; this specification itself is not a passing test.
