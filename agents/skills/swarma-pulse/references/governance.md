# Declared governance view

Use one existing admitted Pulse executable and one explicitly selected, owner-held
product card at its actual location:

```text
pulse me --local-unsigned --view governance --card ABS --revision sha256:LOWERCASE64HEX --format json
```

The same caller/file/content-pin checks as the [query contract](query-contract.md)
apply. Read exactly one product-card/v3, v4 or v5; company cards and discovery
source/root selectors refuse. No member/caller/clock/key override exists.
V5 is accepted only by this selector. The full original document, graph and all
principals/keys are validated, including unselected records. V5 YAML rejects
aliases, merges, explicit tags, nulls, duplicate keys and extra documents.

Use the session's existing A2A_AGENT localpart. V5 agent names and scopes use the
closed responsibility, not the free-text role title. Exactly one chat agent must
match. Human contacts do not become callers. A valid chat service has profile=null;
a parent-native service cannot become a chat caller. Select no first owner or key.

## Closed result

One JSON document plus newline, with exactly 17 top-level fields:

| Field | Value |
| --- | --- |
| schema | pulse-view/v5 |
| mode | local-unsigned |
| product | selected product id |
| member | qualified product/member key |
| caller | agent, cryptographically_verified=false, trust=single-user local trust |
| known_limit | caller not cryptographically verified; single-user local trust |
| source | schema, revision, sha256, card_root, freshness=unknown, observed_at=null; validation only for v4/v5 |
| query_observed_at | query UTC time, not source freshness |
| self | Self below |
| peers | [] |
| direct_reports | [] |
| descendants | [] |
| superiors | [] |
| projects | caller's zero-or-one Project summary |
| hierarchy_status | self-only; hierarchy not disclosed |
| observations | runtime, registration, installed_profile, assignment; each {state:unknown} |
| governance | Governance below |

All nested objects are closed:

| Object | Required fields | Optional fields |
| --- | --- | --- |
| Self | key, kind=agent, role, responsibility, scope, desired_state, resolved_name, execution_binding, profile, project, project_state | none |
| ExecutionBinding | key, operator, host, connection | none |
| Profile | requirement:{context,compatible}, selected:{version,source_rev,lock_sha256}; service-only null | none |
| Project | key, repo, desired_state, location | none |
| Local location | kind=local, path | none |
| Remote location | kind=remote, operator, execution_host, project_locator | checkout_path |
| Governance | state=declared/undeclared, owners:array, meta_keeper:MetaKeeper/null | none |
| Owner | id, kind=human, name, contacts:array, keys:array | none |
| Contact | kind=matrix, mxid | none |
| PublicKey | purpose, algorithm, encoding, public_bytes, key_id, public_sha256, declared_state, declared_state_at | not_before, not_after |
| MetaKeeper | reference:{product,member}, resolution=resolved/external_unresolved, member:MemberDeclaration/null | none |
| MemberDeclaration | key, kind, role, responsibility, scope, desired_state, project, principal | none |

Self project is a qualified key or null, with project_state=declared/undeclared.
MemberDeclaration project is qualified; principal is a human principal id or
agent-only null. Profiles are declarations, not installed/runtime evidence.
Source.validation is exactly `local declarations only; source attestations and star reservations unverified`.
Legacy self uses kind=agent and copies the old role to responsibility without
writing a new card. Legacy or absent metadata means state=undeclared, owners=[]
and meta_keeper=null; supplied invalid metadata refuses rather than disappearing.

Owners sort by principal id; keys by purpose then key_id. Contacts retain order.
Empty keys means public material not supplied. A local meta-keeper resolves only
to its closed member summary. An exact declared external reference stays
external_unresolved/member=null; no second source is opened.

## Public material and refusals

PublicKey purpose is identity or grant-authority. Ed25519 uses canonical 32-byte
ed25519-raw, nonidentity and torsion-free checks; P-256 uses canonical 33-byte
sec1-compressed. BIP-340 uses 32-byte secp256k1-x-only with even-y lift and permits
identity only. Public bytes are canonical unpadded base64url; key_id is respectively
ed25519:, p256: or bip340: plus those bytes; public_sha256 hashes decoded bytes.
State is declared active/retired/revoked with strict UTC-second timestamps; no live
status is inferred. A fully validated public descriptor can resemble token-like
prose; only its public_bytes/key_id receive the typed public treatment. Other
secret-like prose, credential fields and live-state fields remain forbidden.

Canonical point decoding proves neither possession nor owner identity, signatures,
chain/revocation status, current signing authority or protected issuer admission.
No key store, account, runtime, network or declared reference is accessed; no card
or persistent state is written. Governance visibility grants no new permission.

Usage refuses exit2 before input access. Pin/caller/file/schema/whole-card refusals
exit1 with fixed `RULE name: $` plus newline, empty stdout and unchanged inputs.
Invalid v5 metadata collapses to card_invalid without a failed input value. A
final output_write transport failure is distinct and can leave partial stdout.
Keep a nearby valid request with negative checks; never substitute another card,
weaken permissions or reclassify a grants root as identity to make a query pass.
