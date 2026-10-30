# Product cards, stable identities and Pulse — specification v1, revision 3

Status: proposed; implementation is not yet specified as available. Conformance vectors are in
[conformance-v1.json](conformance-v1.json).

## 1. Problem and result

A member needs an authoritative description of its product, projects and team, a stable name that identifies its
federation, and a way to discover peers, reports and superiors. Today the card describes desired staff but assumes local
project checkouts; identity onboarding derives names from a workstation; and no member-facing projection exists.

This specification defines one card contract, one identity resolver and a read-only Pulse query. It also specifies the
move to `gr.relux.works`. Implementation, account provisioning, runtime cutover and deletion are separate operations.
A card or query never grants authority to perform them.

The initial consumer is the swarm product. Its card belongs in its explicit host project, `swarma-host`. The format and
Pulse implementation belong in `swarma-pulse`; each other product keeps its own card in its `<product>-host` project.
The host project is a product inventory anchor, not a machine fingerprint and not a network service by implication.

## 2. Baseline and compatibility

The baseline is FORMAT v3.1 with schema `product-card/v3`. Its separation of desired facts from observed state, profile
requirements from selected locks, manager-required from member-owned subscriptions, and chat from non-chat execution
remains in force. An upgrade must preserve those guarantees.

The proposed new schema is `product-card/v4`. A v3.1 card remains valid under its existing validator until an explicit,
reviewed conversion. New software must identify the schema before validation; it must never reinterpret an old card as
v4 or silently rewrite it. A v3 reader must refuse a v4 schema. Conversion is a separate, deterministic operation with a
reviewable old-to-new mapping; it neither registers accounts nor changes running consumers. Conversion selects
`legacy-exact` for every existing chat identity and preserves its exact MXID, connection and logical key. V4 can therefore
contain legacy and new identities together without requiring a v3 reader to consume v4. A member moves to the new
scheme only at its individual cutover; changing the card schema alone changes no resolved identity.

The existing Ruby validator is the baseline for v3.1, not proof that v4 is implemented. A future v4 validator must use
closed objects, reject unknown and duplicate keys and unsupported YAML features, and emit diagnostics containing rule
and safe location only. Accept UTF-8 YAML mappings, sequences and scalar values in the documented fields; reject tags,
merge keys, aliases and multiple documents. Legacy anchors must be expanded during explicit conversion. Limits are
1 MiB per card, 1,000 projects, 10,000 members, 32 levels of document nesting and 64 reporting hops. Exceeding a limit
is a named refusal, with no partial output or write. These are proposed v1 limits, not existing validator behavior.

## 3. Ownership, provenance and location

The format maintainer owns the schema, validator contract and compatibility policy. A product custodian owns its card's
facts, appointments and review routing. Project custodians submit fact changes to that writer. Neither role gains
custody of another product or operator's identities, secrets or running sessions.

The canonical file is `product-card.yaml` at the root of the declared host project. The file carries a schema and product
id. A handoff carries a locator, immutable revision and SHA-256 of the bytes, and the recipient verifies and acknowledges
its own binding. A local-only source is valid; GitHub publication is not a prerequisite or a permission to publish.
Privacy and access follow the owning product. Operational receipts and private rollout inventories stay in their existing
custody, outside distributed product files.

There is one canonical writer and revision per product. Replicas are explicitly identified snapshots. A conflicting
snapshot is refused rather than merged by timestamp. Moving the current card to `swarma-host` requires a reviewed path
conversion, byte/revision evidence, consumer updates and a verified readback before the former location is retired.
The host's actual creation and publication need their own admitted route.

## 4. Card data contract

All fields not changed below retain their documented v3.1 meaning. A complete v4 schema and conversion fixtures must be
reviewed before coding the validator; this document defines its required structure and semantics.

| Area | Required contract |
| --- | --- |
| Product | Existing id, description, privacy and keeper; add `host_project`, referencing a declared project. Keep default binding and desired topology. |
| Projects | Unique id, description, repository reference or explicit local-only designation, desired state, manager, task-system reference and channels. Add discriminated `location`, described below. |
| Members | Immutable portable `id`, stable primary role, project, parent, profile, lifecycle, desired state, context mode and explicit runtime/model. Add discriminated `identity` and `scope` (`product`, `project`, `member`); sessionless services use the exception below. |
| Bindings | Operator and execution-host references, named connection, and `identity_domain`. Retire arbitrary `identity_prefix` from new resolution; `legacy-exact` preserves an attested old full MXID unchanged. |
| Connections | Named HTTPS endpoint and Matrix server name, with no credentials or credential-store contents. |
| Profiles | One declared source, compatible requirement and immutable selected version/source revision/lock digest. Selection is not evidence of installation or launch permission. |
| Relations | Parent references resolve locally or to an explicit external product/member reference. No dangling references, duplicate resolved identities or cycles. |
| Topology | Product Space, lobby and escalations; explicit room ids plus known aliases/server names and foreign grouping references. Membership, power and grouping remain distinct. |

Logical member ids are retained through migration and must not embed a machine binding. Reference equality uses
`product-id/member-id`, never a display name or a star name alone. All members, including product-level members, name a
project; product-level custody is anchored to the declared host project. A stable role describes the appointment;
capabilities describe assignments it may accept. A reviewer assignment does not silently rename a developer.

`lifecycle: persistent` describes a stable member. Legacy `on-demand` means a stable member activated for an assignment
and parked afterwards; it never means creating a new member per story. Task ownership, current assignment, runtime
session, PID, liveness and acceptance state remain in their native systems. A desired staffing slot is not a member or
an admission record. Review still requires a distinct qualified member and a fresh context without producer history.

### Mixed legacy and new identity representation

A chat member has an explicit named `binding` (conversion materializes any inherited default) and exactly one of these closed identity objects:

- `{scheme: legacy-exact, mxid: FULL_MXID, source_ref: PINNED_SOURCE}` preserves an existing identity. `source_ref`
  is `{kind, revision, sha256, member}`. Its kind is `v3-card`, or `service-descriptor` for a separately admitted legacy
  service absent from v3. It pins the pre-conversion source and unchanged qualified member key. The supplied source must resolve that key to exactly this MXID and connection. Missing source, changed key/MXID/connection,
  duplicate identity or a Matrix server inconsistent with the selected connection refuses validation. The resolver
  returns the attested MXID verbatim; it does not parse a host name into a new tuple. For a v3 source, independently
  derive its localpart as the selected source binding's `identity_prefix` plus the unchanged `member.id`, on that
  binding's connection. The source pin covers those input bytes; a pinned precomputed result map alone is insufficient.
- `{scheme: domain-star-v1, star: STAR}` resolves the domain/product/project/role/star tuple from that member and binding.
  The full-domain and reservation rules below apply. `mxid` and a legacy `source_ref` are forbidden in this branch.

The legacy branch can verify/reuse an already admitted existing account; it cannot provision, register or adopt a new
legacy account. No prefix guessing, account-presence probe or token recovery can replace the pinned source. Non-chat
members may declare a prospective `domain-star-v1` tuple without a Matrix account; they cannot select `legacy-exact`.

The card holds one selected identity per member, never live migration state. A reviewed cutover changes only the pilot's
`identity`, `binding` and associated desired a2a connection/room/subscription fields. The private migration receipt pins
the complete prior member record and its source revision for rollback; rollback restores those exact desired fields.
Other member records and their resolved MXIDs stay unchanged. The old connection remains declared while referenced.
Conversion must translate old `a2a.connection` values consistently, rather than deriving identities from the new domain.

The mixed-card fixture supplies three preservable member ids, the source v3 prefix/connection/default and explicit
bindings, source pins, a pilot-only delta and independent expected resolution maps before cutover, after cutover and
after rollback. Its v3 source excerpt suffices to derive every original mapping rather than merely asserting it.
A fourth case changes a non-pilot identity and must fail the preservation invariant. These are binding-resolution excerpts; complete v4 card fixtures will include unchanged
profiles, topology and other required fields when the full schema is finalized.

### Sessionless service members

V4 explicitly adds `role: service`, with `scope: member`, a project and an accountable parent who is that project's
manager. A service has its own logical key, lifecycle and desired state, a separately admitted `service_contract_ref`,
`runtime: {kind: none}` and `context_mode: none`. `model`, `effort`, conversational `profile`, agent-session continuation
and conversational staffing are forbidden. The service contract identifies its actual service runner and health/result
interface; it is a reference, never a launch instruction, credential or evidence of a running process.

A service declares `result_route` explicitly as `a2a` or `parent-native`. For `a2a`, separately admitted chat participation,
a named connection, declared project rooms, `parent_route` and `a2a.result_destination` are required. Its
`a2a.subscriptions.required` and `self` are explicit lists; the service-only baseline policy is empty for both, but omitted
lists still refuse. Nonempty selectors require an explicit service-policy disposition referenced by `selector_policy_ref`;
conversational owner selectors are never added by default. Use a qualified service-aware provisioning route that
preserves this policy, or stop before cutover if none is available. Automatic global membership is forbidden.
For `parent-native`, an admitted service result-interface reference is required and there is no Matrix account or a2a block. Naming code `service` adds no admission or admin/invite power.

A service is not launched by a conversational Curator member command. Passing it to an agent-session launcher must
refuse `service_session_forbidden` before hooks, session/binding writes or model invocation. Its service owner admits
any actual process start separately. Migration proves the owned runner's account/endpoint, expected event or request,
durable handling and attributable result/health readback. It does not demand a conversational idle/busy wake or resumed
inbox. The no-overlap and rollback rules apply to owned service publishers/runners, without stopping another service.

### Project locations

`location.kind` is exactly `local` or `remote`:

- `local` requires `path`, relative to this card's directory. Project task-system paths resolve against that project.
- `remote` requires an operator reference, execution-host reference and project locator on that host. An optional
  checkout path is relative to that host's declared product root and is never resolved on the querying machine.

A remote location has no local `path`. Missing local layout is an error only for a declared local location when layout
checking is requested. A remote project is not reported missing because it is not checked out here. Repository identity
and checkout location are separate: a remote Git repository does not make a local checkout remote. Repository references
contain host/owner/name, never userinfo, query credentials or an embedded access token. Remote task-system references
identify the native system and locator; Pulse does not open them as local files.

All filesystem paths are nonempty relative paths, with no leading `/` or `~`. Existing profile/member paths retain their
v3.1 bases; conversion records each base explicitly and preserves the resolved target. `..` is allowed for the declared
sibling layout but cannot escape an operator-admitted root. Layout checks resolve symlinks and must refuse an escape
before inspecting the destination. Validation does not create missing directories or fetch a remote project.

### Parent graph and participation

The keeper reports to an explicit external parent, a project manager reports to its product keeper, and a project member
reports to its appointed manager. Existing keeper-managed projects remain valid with their decision reference; no extra
orchestrator is invented. External ancestors require a pinned, authorized projection or card reference. An unavailable
external ancestor produces an unresolved boundary, not a fabricated owner or a shorter supposedly complete hierarchy.

Every member can have a logical identity tuple. Only a chat-admitted member has a Matrix account. Existing non-chat
members continue through their admitted parent-native route. New worker chat participation depends on the enacted a2a
policy, per-member admission and qualified onboarding; accepting this spec grants none of those. Their prospective
names can be validated without registering accounts. Project-only members must not acquire global room membership.

Required and self subscriptions stay separate. Required coverage cannot be removed by a self update. Human selectors
use exact full MXIDs, including both owner domains where required. Spaces are structural and cannot be subscription
sources. Room grouping never grants foreign membership, power or encryption changes. No plaintext fallback is permitted
where an independently admitted route requires encryption.

## 5. Stable identity scheme

Owner naming decision, 2026-10-30: the [full-name/compact-Matrix amendment](identity-names-v2.md) supersedes this section's domain-prefix form for future registrations. The `domain-star-v1` rules below remain the unchanged implemented v4 compatibility contract; they are not authority to register a new name under the superseded form.

For `domain-star-v1`, the canonical Matrix localpart is:

```text
<identity-domain>_<product>_<project>_<role-code>_<star>
```

The full MXID is `@<localpart>:<connection.server_name>`. On the Alexis-side target connection both the identity domain
and Matrix server name are exactly `gr.relux.works`. Every binding must equate the identity domain to its verified
Matrix server name; the HTTPS endpoint hostname may differ. The complete domain is the prefix: neither `gr` nor a workstation
slug may replace it. Host and operator remain binding facts, outside the name. A member moving execution hosts within
the same appointment keeps its identity; moving federation, product, project or primary role requires explicit identity
migration while preserving its logical member key and prior work.

The domain is a lowercase ASCII DNS name with at least two valid labels, each 1–63 characters, no empty label, no
leading/trailing hyphen and no trailing dot. Product/project slugs and star slugs match
`[a-z][a-z0-9]*(?:-[a-z0-9]+)*`; each is 1–40 characters. Underscores delimit the five fields and cannot occur inside a
field. Hyphens in canonical project ids are preserved. Inputs must already be canonical; no lossy normalization,
automatic abbreviation, truncation, hash suffix or random fallback is allowed.

| Primary role | Code |
| --- | --- |
| keeper | keeper |
| orchestrator | orch |
| developer | dev |
| tester | tester |
| architect | arch |
| reviewer | reviewer |
| researcher | researcher |
| separately admitted service | service |

A code is only a naming vocabulary entry; it does not admit that role to chat or launch. Unknown roles refuse. A combined
keeper/project-manager retains its keeper appointment and code; capabilities and manager references record the combined
responsibility without creating a duplicate member.

Stars are readable astronomical names from a reviewed registry, such as `betelgeuse`, `rigel`, `vega`, `sirius`,
`antares`, `altair`, `deneb` and `capella`. No catalogue numbers or generated suffixes. Reserve a star within
`(identity-domain, product, primary-role)` across that product's projects, with the immutable member key as its holder.
The same logical member keeps the reservation across assignments; concurrent reservations fail atomically. A retired
reservation remains recorded and is not automatically reused. The complete resolved MXID must also be unique across
all admitted host bindings. Stars are proposed by the custodian and reserved through the reviewed card change, not by
racing registrations. Cross-operator reservations require their agreement before accounts are created.

The entire new-scheme localpart must fit a2a's current 64-character ceiling. The resolver rejects overflow before filesystem,
Keychain, registration, room or hook effects. If necessary, custodians propose a separately reviewed canonical project
slug before appointment; the domain and an existing identity are never silently shortened.

The proposed first target is:

```text
@gr.relux.works_swarm_swarma-pulse_orch_betelgeuse:gr.relux.works
```

Its localpart has 49 characters. `swarm` is the product and `swarma-pulse` is the project. Informal examples using
`swarma_pulse` are not a decision to rename or split those ids. The example is a proposed reservation, not a registered
account. Members of other federations use their full agreed domain and require that operator's adoption.

## 6. Pulse query

V1 is a local, read-only CLI over explicitly supplied, authorized card snapshots. It needs no daemon, background
reconciler, account registration or automatic remote discovery. Proposed interface, not a shipped command:

```text
pulse me --card PATH --revision REV --member PRODUCT/MEMBER --caller-binding PATH --format json
```

The query also requires a trusted local caller binding and a visibility policy. Proposed explicit inputs are
`--caller-binding PATH` and optional `--visibility-policy PATH`, plus repeated pairs of `--extra-card PATH` and
`--extra-revision REV`. These are data locators, never a choice of trust anchor or another caller's authority.

The local host's admitted Pulse configuration selects its public trusted-signers set and current execution binding for
the invoking OS principal/runtime. No CLI argument, card field or environment fallback can replace that configuration.
The caller file is UTF-8 JSON with exactly `schema: pulse-caller/v1`, `subject` (qualified member key),
`execution_binding`, `card_sources` (product, revision and SHA-256 entries), `visibility_sha256` (digest or null),
`issued_at` and `expires_at` (UTC timestamps). Duplicate or unknown keys refuse. A detached SSHSIG signature over the
exact file bytes, in namespace `pulse-caller-v1`, must verify against a signer authorized by that host configuration.
The detached format and namespace separation follow the [OpenSSH signature format](https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.sshsig).
The product custodian supplies the member/policy handoff through the operator-admitted local route; neither the file's
claimed signer nor its filename admits a key. Trust provisioning is external to this read-only query.

The subject must equal `--member`, its execution binding must equal the host's admitted caller binding, and its selected
card binding must resolve to that same execution binding. Every supplied snapshot must match its signed allowlist entry,
revision and bytes. Missing/untrusted/expired caller input, unknown host binding, selector/binding mismatch, digest
mismatch, conflicting snapshots for one product or an unauthorized extra source refuses the entire query. No partial
view is returned for these integrity/authority failures. This attests a local admission; it is not remote authentication.

A non-null `visibility_sha256` requires the exact policy bytes via `--visibility-policy`. The policy is closed UTF-8 JSON
with `schema: pulse-visibility/v1`, `subject` and `visible_members` (qualified logical keys); its subject must match the
caller. The signature on the caller binding pins the custodian-authorized policy digest. Self is always visible.
An absent policy digest means the explicit default of self-only visibility; supplying an unpinned policy refuses.
Members outside the allowlist produce anonymous restricted relationship boundaries, without names, keys, profile data
or locators. The policy cannot add an unpinned snapshot or grant access to files outside local authority.

The peer universe is exactly the selected product/card plus the explicitly supplied, allowlisted extra snapshots.
There is no network expansion, ambient workspace scan or implicit cross-product peer lookup. Peer level and relations
are computed within that universe, then filtered by visibility. A referenced external ancestor without an authorized
supplied snapshot is marked unavailable at that boundary; it is not evidence that no ancestor exists. A restricted
ancestor is marked restricted instead. Traversal stops at that boundary and never skips over it to expose higher
ancestors. Neither condition makes the result a complete hierarchy.

For a local-only card the revision is a verified content digest or immutable local Git revision plus verified file bytes.
A service/remote API, if later needed, requires a separate authentication and audience contract. V1 reads only snapshots
provided through the existing handoff and local filesystem authority, never arbitrary URLs or credential directories.
Local callers with permission to read a whole file already have that filesystem access; this CLI promises no secrecy
from another process under the same OS identity or holder of the same admitted caller context. Cross-custody cards
require sanitized snapshots or separately authorized read access before invocation.

The result carries `schema: pulse-view/v1`, product/member ids, source revision and content digest, and:

- `self`: role, scope, logical key, resolved name (or non-chat designation), desired state, profile requirement/selection
  for conversational members or service-contract reference for services, execution binding, project locator and own
  declared parent/result/escalation routes;
- `peers`: visible members at the same responsibility level, with their roles, projects and parent refs;
- `direct_reports` and `descendants`: separate lists derived from parent edges;
- `superiors`: ordered nearest parent first, ending at a verified root or an explicitly unresolved external boundary;
- `projects`: locators needed for these members, preserving local/remote distinction;
- `observations`: separately sourced facts, each marked known, absent, stale, unknown or unavailable, with source and
  observation time where available. V1 may return unknown for all runtime/assignment observations; it must say so.

Responsibility levels are product (keeper), project (orchestrator) and member (other appointed roles). Peer equality uses
this level, not equal graph depth: a worker reporting directly to a combined keeper-manager remains member-level.
The query excludes self from peers and reports, includes parked/retired members with their desired state, and sorts
lists by qualified logical key. Descendants are all reachable reports, with no duplicates; superiors remain ordered by
the parent chain. Rank conveys organizational position, never a permission. Query audience can limit visibility;
restricted boundaries are explicit and may not be described as a complete tree.

Only references and fields needed for the projection are returned. No credentials, raw session traces or private task
payloads enter the card or projection. Profile locks and desired state are declarative facts; actual installed locks,
registered accounts, launch, wake, assignment and accepted results are reported separately. An expired snapshot is
marked stale under an explicitly configured age bound; absent provenance or invalid/cyclic local graph refuses the
whole result. An unreachable external reference gives a partial view with a named unresolved boundary. No stale or
partial view can authorize launch or migration.

## 7. a2a and launcher integration

The a2a source custodian owns changes to naming/onboarding, shared protocol wording and its published CLI/skill. Pulse
supplies a versioned, minimal resolved-member descriptor, not a second eligibility or credential schema. Existing
source-owner review and both operators' shared-policy dispositions remain necessary. This spec does not amend rule 26
or activate an unlanded worker-chat policy on its own.

| Integration point | Required behavior |
| --- | --- |
| Identity resolution | Accept a validated domain/product/project/role/star tuple; preserve legacy resolution only for explicit existing-identity operations. New onboarding uses the selected scheme and never silently falls back. |
| Input descriptor | Carry descriptor schema, card revision/digest, qualified member key, tuple, full MXID, named connection, intended audience/rooms and admission references. Reject tuple/MXID/connection mismatch. The exact transport is source-owner reviewed. |
| Onboarding | Validate the entire tuple, connection, custody/admission and provisioned-room readiness before side effects. Missing rooms do not trigger implicit creation. A named connection selects gr explicitly; the full domain is not passed as a host slug. |
| Connection routing | Use supported named connections, separate from Curator profiles. Endpoint, Matrix domain and store remain pinned. A per-member selection must not change another running member's connection. |
| Registration | Only the target's admitted operation and supported secure credential route. Unavailable bootstrap or ambiguous/pending registration stops that member's registration step without blind retry or legacy fallback. |
| Account identity | Verify full MXID, endpoint and account class. An existing name without locally held admitted access is an ownership conflict, not an invitation to take over. |
| Rooms | Provision new product Spaces/rooms on gr through their custodians. Retain exact shared relux room ids and join by federation; an alias on another server must not retarget a room silently. |
| Notifications | Carry the connection and identity into hooks, native bindings and their children. Prove addressed inbox delivery, eligible idle wake, resumed inbox and attributable reply. A selector or binding status alone is insufficient. |
| Agent cards | Publish the member's own a2a card through its admitted route; the product card remains a separate artifact. No agent card write to another member's files. |
| Launch | Only the selected validated Curator profile and supported launch route. Preserve profile locks, runtime restrictions, one-live custody and independent review. A Pulse result does not launch anything. |

Observed a2a documentation already describes named connection isolation, explicit registration authorization and no
cross-server cursor copying. Its current onboarding still derives the legacy host/project/role tuple and has global-room
presets. Those facts are a dated source baseline, not qualification of a new name or a project-only onboarding route.
The source owners must identify the exact released and adopted candidate for each missing capability before rollout.
No local wrapper, forged host slug or direct credential-file edit substitutes for that change.

## 8. Migration to gr.relux.works

A Matrix account cannot be renamed. Migration creates a new admitted account, qualifies its routes and consumers,
then retires active use of the old account. It preserves the logical member key, appointments, accepted work and task
history. It does not delete accounts or old rooms.

### Inventory and order

Moving all ready Alexis-side identities is the top execution priority. Product keepers supply sanitized inventories
of all existing identities, including themselves, project orchestrators,
stable workers, services, parked members and legacy aliases. The migration ledger records logical member, old MXID,
proposed target tuple/MXID, card revision, responsible custodian, audience, dependencies, status/evidence and rollback
owner. Obtain the inventory from custodians and admitted account/member listings, never credential-store enumeration.
Identities absent from a product card are reconciled to a responsible keeper before migration. Unknown scope remains
explicit; seven projects in one card is not evidence that the entire operator inventory is covered.

Order is: this orchestrator first; then the remaining Alexis-side members one keeper at a time at their own safe
boundaries; parked members remain parked and get a target plan without being started. Within each keeper's scope,
choose the next member by dependency and custody readiness. Keep the old parent route usable until the new parent route
is proven. Service identities require their service owner's explicit handoff. The other operator's side is inventoried
and proposed through its keepers and migrates only as far as that operator agrees. No new throwaway identity is required
as a canary: the first real member is the pilot.

The ledger accounts for every inventoried identity as migrated, planned/parked, foreign-adoption-pending or blocked with
an owner and exact dependency. These labels must not be collapsed into "all renamed". Full retirement needs each
required pilot and per-member proof, not a successful registration alone.

### Preconditions

Before each member moves, require the accepted specification, an adopted a2a candidate supporting the exact tuple and
connection, a reserved target name, a reviewed card mapping, explicit account/custody admission, ready rooms/invitations,
known consumers and a supported cutover/rollback path. At this specification baseline, the gr registration token is not
yet on the host. The host owner supplies that token
through the supported secure bootstrap reference before registration; this is distinct from source/tool qualification.
Agents neither read nor transfer its value. Never use or copy the owner's human access token or password as a substitute,
and never place human credentials in any agent-readable item or store. Existing pending registration outcomes use
source-owner recovery, never journal deletion or a second speculative name. Credential-reader signing/trust constraints remain intact.

A schema acceptance is not permission to change a live binary, registration store, hook trust, security posture or
shared daemon. Each operation retains its actual custodian and supported maintenance boundary. Source implementation,
source review, installed adoption and member rollout are separate recorded facts.

### Conversational member state transition

This path applies to conversational Matrix members. Sessionless services use the complete service path below, including
its zero-selector policy; they do not inherit conversational hooks, profile or owner-message proof requirements.

1. **Prepared:** capture sanitized old identity/connection/binding facts, card revision, active task/continuation
   references, consumer ownership, room mapping, subscriptions and rollback steps. Establish one-live custody through
   the authoritative host route; missing liveness telemetry is inconclusive. Preserve old operational state in place.
2. **Registered:** create exactly the reserved target account through the target's admitted secure route. Verify full
   MXID and endpoint. Distinguish no attempt, rejected, ambiguous, created-with-incomplete-local-storage and ready.
   An unknown outcome halts registration recovery for this member and leaves its old path usable.
3. **Provisioned:** join gr replacement rooms and retained shared foreign rooms through authorized invitations. Apply
   required selectors under the new identity and read them back; preserve applicable self choices explicitly. Verify
   the exact owner sender selectors required by that conversational member's admitted policy across its allowed rooms.
   This owner-selector requirement is conditional on conversational admission, not universal to migrating identities.
   No Space subscriptions or unintended global rooms.
4. **Qualified:** verify target membership, sender/address resolution and inbox durability through an admitted staging
   route. Prepare the reviewed hook/launcher/card handoff. A staged account is not a second live conversational session;
   qualification that needs live activation waits for the next step.
5. **Cut over:** at the member custodian's safe point, stop only that member's old consumer through its supported route,
   preserve the in-flight checkpoint, activate the target consumer and profile, then apply the reviewed per-member card
   binding. Never activate both consumers to hide a custody gap. If no safe supported handoff exists, remain prepared.
6. **Proven (conversational member):** a peer sends an addressed event; it reaches the new durable inbox, causes an
   eligible idle wake, is read by the resumed member and gets an attributable reply visible in the original topic. Verify an ordinary owner message
   from each required owner MXID as well. Confirm one active consumer, correct room ids and pinned gr connection.
   Test busy delivery at a later safe point before marking full notification readiness; do not interrupt a human draft.
   A sessionless service instead supplies the service-runner proof defined in §4, with no conversational wake check.
7. **Retired from active use:** announce old-to-new mapping, update authoritative cards/launch references and callers,
   check no active dependency remains on the old binding, and record acceptance of the per-member evidence. Keep old
   account and state preserved with no active hooks/consumer. No old chat deletion, shared room deletion or revocation
   is included in this step.

### Sessionless service state transition

All seven stages apply through the admitted service contract. The positive conformance trace reaches retirement with
`subscriptions: {required: [], self: []}`, no agent hooks/profile, no model invocation and no conversational binding.
Empty lists are verified from the actual service route; an empty desired list alone does not prove actual account data.
A service-specific selector policy may override the empty lists only through its explicit disposition; it never inherits
conversational defaults. The neighboring negative control rejects an implicitly added owner selector.

| State | Service-specific evidence and action |
| --- | --- |
| Prepared | Preserve the service owner, admitted runner/result interface, old account/connection, durable result checkpoint, allowlisted project rooms, explicit selector policy and rollback record. No conversational task/profile/hook is created. |
| Registered | For a chat service, create and verify exactly the reserved service account through the admitted secure route. Ambiguous outcome stops recovery here with the old runner preserved. A non-chat service records this account stage as not applicable, without creating an account. |
| Provisioned | A chat service uses the service-aware route to join only allowlisted rooms and reads back the admitted selectors, empty for the positive control. No global/Space/owner defaults are added. A parent-native service verifies its admitted result interface instead and has no rooms or selectors. |
| Qualified | Validate the target runner configuration and result/health interface without activating a second publisher. Stage account/audience/readiness evidence through the service route, never an agent inbox hook or conversational profile. |
| Cut over | At its owner's safe durable-result checkpoint, stop only the old owned runner before starting the new one through the service contract. Apply the per-member desired binding/identity change. If no safe runner handoff exists, remain prepared. |
| Proven | An expected event/request is handled durably and produces the attributable result or health readback on the new route. Verify the declared account/endpoint/audience and actual selector policy; no idle wake, owner-message receipt or resumed conversational inbox is required. |
| Retired from active use | Verify old runner inactive, new route proven, no active old dependency and preserved old account/state. Update admitted callers and desired references; keep actual selectors empty in the positive control. No account deletion or conversational hooks/profile are introduced. |

For a non-chat service, registration/membership stages remain explicit not-applicable records under the service contract,
not a failed chat onboarding. A failure restores its prior runner/result interface without creating a Matrix identity.
Chat-service rollback stops the new owned runner first and restores the exact old identity/binding/runner checkpoint;
old state remains preserved. Neither service path creates overlapping active publishers or manufactures result receipts.

Failure before cutover leaves the old member working. Failure after cutover stops only the new owned consumer first,
then restores the old supported binding/card/consumer at the recorded checkpoint. Verify delivery again and mark the
new account staged with the actual partial effects. Do not erase a new remote account to simulate atomic rollback.
Never copy cross-server sync cursors, mailbox receipts or leases; retained old pending mail is drained or recorded under
its old account and new handoffs reference it without manufacturing a receipt.

### Room and card transition

New Alexis-side product Spaces and message rooms are hosted on gr. Existing Alexis-only relux conversations are
abandoned for new work after the corresponding handoff; preserve any unresolved decisions through linked, sanitized
continuation references. Old chat cleanup is a later exact-operation decision. Shared relux rooms stay on relux and the
new gr account joins them by federation. No membership/power change on the other operator's side is inferred.

During rollout keep old and new named connections declared and select a binding per member. Do not switch the product
default until every member still using it has an explicit binding and the resulting mapping is reviewed. Update room
ids and aliases separately from account domains; no domain string replacement over YAML or runtime state. Migration
receipts live in the native tracker; the card stores desired bindings and evidence locators, never current rollout state.

## 9. Acceptance scenarios

These are requirements for future executable fixtures and exact-candidate review, not tests claimed to have run. Each
negative scenario requires its specific refusal/invariant and a nearby valid control. Run all selected lanes and report
all outcomes, including lanes that could not start; one red lane must not suppress the others.

| Contract | Valid control | Forbidden case and required result |
| --- | --- | --- |
| Legacy mapping | Three old identities resolve exactly before conversion; pilot-only cutover and rollback have pinned expected maps. | Missing legacy source, changed old MXID/connection, non-pilot delta or attempted legacy registration refuses without changing another member. |
| Service | An admitted service reaches all seven migration states and retirement with runtime none, explicit result route, zero actual selectors, no agent hooks/profile and a proven owned runner. | An implicit owner selector violates service policy; a model/profile or conversational session launch refuses `service_session_forbidden` before session, hooks or model effects. |
| Naming | Full gr domain, swarm/swarma-pulse, orch/betelgeuse resolves to the example. | Short domain alias, embedded underscore, uppercase or unknown role refuses before any registration or local write. |
| Name budget | Exactly 64 characters is accepted if every component and registry reservation is valid. | 65 characters refuses without truncation, random replacement or partial files. |
| Reservation | Existing holder repeats the same proposal idempotently. | Two members reserve the same domain/product/role/star, even in different projects: second fails with no allocation. |
| Membership | Assign two successive stories to one stable member. | No new logical id, account or star reservation is created by either assignment. |
| Locations | Local sibling checkout and remote project both validate under their respective rules. | Absent remote checkout is not a missing-local error; absolute paths and symlink root escapes refuse before traversal. |
| Graph | Ordinary and combined-manager examples produce correct chains and peer levels. | Cycle, dangling local parent, duplicate member or resolved MXID refuses; missing external source is explicitly partial. |
| Projection | Two distinct trusted caller bindings receive different correct self/report views; peers use only pinned snapshots and authorized visibility. | Unauthorized selector, absent/expired attestation, wrong host binding or digest refuses. A restricted peer exposes no identifier; unavailable ancestor is an explicit boundary. Unknown/stale observations never become permission. |
| Card isolation | Desired profiles/locks and explicit empty subscription lists validate. | Secrets, live-state fields, unknown keys, duplicate YAML keys, tags, aliases and oversize/deep documents refuse with safe locations only. |
| Audience | Non-chat member uses parent-native route; admitted chat member has its declared project audience. | Naming support cannot grant chat, invite/admin rights, global autojoin or remove required owner selectors. |
| Routing | Explicit gr target and federated shared room resolve correctly. | Missing profile, metadata mismatch, redirect to another credential origin or stale fingerprint refuses; no legacy fallback. |
| Recovery | A pre-cutover failure leaves the old consumer and card effective. | Ambiguous registration or mid-storage failure cannot trigger blind retry, state deletion, fabricated receipts or loss of old custody. |
| Cutover | One real member proves idle wake, inbox and reply, with a reversible owned-consumer handoff. | A second active consumer, account-class conflict or missing custodian boundary stops cutover; no peer process is stopped. |
| Retirement | Old consumer is inactive, new one verified and outstanding work referenced. | No old account/room deletion, foreign migration or weakened security policy follows from acceptance of this spec. |

Instrument controls must prove the relevant future validator/runner detects an intentionally invalid isolated fixture.
A crash, skipped control or unrelated setup failure does not count as the intended rejection. No fault injection runs
against user credentials, shared state or live members. Documentation review now checks these pairs for consistency;
implementation qualification remains pending.

## 10. Technical dependencies and priority

The design chooses full structured identities rather than a workstation-derived prefix, explicit local/remote locations
rather than dummy checkouts, and a local read-only query before a network service. The current naming examples are
provisional while the product/project canonical spelling is resolved; no reservation follows from an example.

The execution order is: finalize the minimum mixed-identity schema and conformance inputs needed for migration;
complete and adopt the source-owned a2a naming/connection work; qualify the real pilot; then migrate the remaining ready
Alexis-side members keeper by keeper. Preserve planned/parked/blocked records and continue feasible migration preparation
when a particular member waits on its own dependency. One pilot does not complete migration. Nonessential Pulse
implementation comes after these ready migrations; only a projection capability demonstrated necessary for migration
may precede them. Nothing in this sequence starts a session or changes credentials by itself.

Technical prerequisites are the host-owner-provided gr registration token, a qualified secure registration route,
exact adopted a2a/launcher capabilities, trusted local Pulse caller/policy inputs, each operator's inventory/adoption and
the declared host-project location. An unavailable dependency prevents its own operation; preparation can continue.
Per-member migration evidence and external input status are kept in the native operational system, outside the card.
