# Product-card v5 — governance MVP format

Status: implementation contract, 2026-11-17. This format adds public owner declarations and human membership to the v4 format. Its first consumer is the explicit [governance query](pulse-governance.md). Existing v3/v4 tools and selectors keep their contracts. A card or public point proves neither identity, possession, authority nor runtime admission.

The station-key registry proposal remains separate. Its incomplete registry references, custody records and caller-envelope changes are unsupported fields in this MVP. This document supersedes that proposal's tentative v5 discriminator allocation; it does not implement those deferred contracts.

## Encoding and bounds

Exactly one UTF-8 YAML mapping, including JSON within that subset. Reject aliases, anchors, tags, merges, duplicate/non-string mapping keys, extra documents and nulls. Every object is closed. No coercion, trimming, normalization or truncation. Optional means absent. Strings are nonempty, booleans are actual booleans and integer counts exclude booleans.

Keep the v4 type grammars and byte/path/root limits: raw size1MiB, nesting32, node budget1000000, projects1000, members10000, traversed reporting edges64. Add principals10000, contacts128 per principal, keys128 per principal, owners10000. Title is v4 text bounded to255 UTF-8 bytes. Principal ids use the portable id grammar in a separate namespace; member logical ids preserve the pinned legacy branch. Equal strings across these namespaces create no link.

V4 text, portable/logical id, slug, canonical DNS/full MXID/room, repository, revision, SHA256, relative path and version-range grammars remain normative as specified in [v4](format-v4.md). In particular the legacy logical-id branch is not narrowed to portable ASCII. No referenced path, source or contact is an automatic read/fetch instruction.

## Complete closed tables

| Object | Required keys and types | Optional keys and types |
| --- | --- | --- |
| Card | schema: literal product-card/v5; product: Product; external_members: list External; connections: nonempty map Connection; bindings: nonempty map Binding; profiles_root: path; profiles: nonempty map Profile; projects: nonempty list Project; members: nonempty list Member; principals: list HumanPrincipal | none |
| Product | id: slug; description: text; privacy: text; keeper: local logical-id; default_binding: binding key; host_project: project id; topology: Topology | rooms: nonempty list RoomDeclaration; governance: Governance |
| RoomDeclaration | role: space, lobby or escalations for Product; project_room for Project; server_name: server | alias: full qualified room alias; room_id: opaque or legacy qualified room ID; at least one required |
| External | product: slug; member: logical-id | none |
| Connection | homeserver: HTTPS origin text; server_name: canonical DNS domain | none |
| Binding | operator: text; host: text; connection: connection key; identity_domain: canonical DNS domain | none |
| Profile | requirement: Requirement; selected: Selection; exactly one of path: path or registry: Registry | none beyond that exclusive source |
| Registry | repo: Repository; directory: path | none |
| Requirement | context: text; compatible: version-range text | none |
| Selection | version: MAJOR.MINOR.PATCH string; source_rev: 7–40 lowercase hex; lock_sha256: sha256 | none |
| Topology | connection: connection key; space: Room; lobby: Room; escalations: Room | grouped_rooms: list Room; upward_escalations: Room |
| Project | id: slug; description: text; repo: Repository or literal local; location: Location; desired_state: State; orchestrator: local logical-id; task_system: TaskSystem; channels: Channels | desired_state_ref: ref; orchestrator_mode: literal member or keeper; orchestrator_ref: ref; rooms: nonempty list RoomDeclaration |
| Local Location | kind: literal local; path: path | none |
| Remote Location | kind: literal remote; operator: text; execution_host: text; project_locator: ref | checkout_path: path |
| Local TaskSystem | kind: literal task-board or tasks-md; path: path | none |
| Remote TaskSystem | kind: literal task-board or tasks-md; locator: ref | none |
| Channels | mode: literal shared or separate; team_room: Room | none |
| Common Agent | id: logical-id; kind: literal agent; role: title; responsibility: Role; project: project id; parent: Parent; scope: Scope; binding: binding key; identity: Identity; lifecycle: persistent or on-demand; desired_state: State; context_mode: continue, fresh or service-only none; runtime: Runtime | desired_state_ref: ref; capabilities: list Role; clone: path; git_dir: path; station: ref; logical_source_ref: LegacySource |
| Conversational Agent additions | profile: profile key | staffing: list Slot; independent_of_producer: boolean; result_route: a2a or parent-native; chat_admission: ref; a2a: A2a |
| Service Agent additions | service_contract_ref: ref; result_route: a2a or parent-native | a2a: ServiceA2a; native_result_ref: ref; selector_policy_ref: ref |
| Conversational Runtime | kind: codex_cli or claude_code; model: text | effort: text |
| Service Runtime | kind: literal none | none |
| Slot | role: non-service Role; profile: profile key; count: nonnegative integer; lifecycle: persistent or on-demand; context_mode: continue or fresh | independent_of_producer: boolean |
| Legacy Identity | scheme: literal legacy-exact; mxid: full MXID; source_ref: LegacySource | none |
| New Identity | scheme: literal domain-star-v1; star: reviewed star slug | none |
| LegacySource | kind: v3-card or service-descriptor; revision: revision; sha256: sha256; member: qualified product/member key | none |
| A2a | connection: connection key; parent_route: Room; rooms: nonempty list Room; subscriptions: Subscriptions | spaces: list Room |
| ServiceA2a | all required A2a keys; result_destination: Room | spaces: list Room |
| Subscriptions | required: list Selector; self: list Selector | none |
| Room Selector | rooms: nonempty list Room; kinds: nonempty list Kind | subtypes: nonempty list text |
| Human Selector | from_mxid: full human MXID | rooms: nonempty list Room; all_events: literal true |


| Human Member | id: portable id; kind: human; role: title; responsibility: non-service Role; project: project id; parent: Parent; scope: Scope; desired_state: State; principal: principal id | desired_state_ref: ref; capabilities: list Role |
| HumanPrincipal | id: portable id; kind: human; name: title; contacts: list Contact; keys: list PublicKey | none |
| Contact | kind: matrix; mxid: full MXID | none |
| Governance | owners: nonempty unique list principal id; meta_keeper: Parent | none |
| PublicKey | purpose: identity or grant-authority; algorithm: KeyAlgorithm; encoding: KeyEncoding; public_bytes: canonical unpadded base64url; key_id: prefixed canonical public bytes; public_sha256: sha256; declared_state: active, retired or revoked; declared_state_at: UTCSecond | not_before: UTCSecond; not_after: UTCSecond |

Member is either Human Member or Common Agent plus exactly Conversational Agent or Service Agent additions. Agent responsibility=service selects the service branch. Responsibility uses the old closed Role codes: keeper, orchestrator, developer, tester, architect, reviewer, researcher, service. Slot.role and capabilities stay mechanical codes. Member.role is a free title and never selects privilege, scope, identity or runtime.

New human member ids use the existing portable-id predicate. Human records forbid logical_source_ref and runtime/model/profile/context_mode/binding/identity/lifecycle/clone/git_dir/station/staffing/a2a fields. Pinned nonportable legacy ids remain available only to the agent branch; an unpinned nonportable agent id still refuses. Principal resolves to one local HumanPrincipal. Human responsibility=service refuses. Agent principal refuses. Services retain runtime.kind=none, context_mode=none, no profile/model/effort and their exact service-contract/result-route constraints. Kind=agent is their compatibility bucket, not a requirement for an LLM.

## Relations and preservation

Product.keeper resolves to the single responsibility=keeper member. Scope is product for keeper, project for orchestrator and member otherwise. Product-scope members name host_project. Each orchestrator reports to the keeper; other members report to their project's declared manager. Keeper's parent is an exact declared External. All local refs resolve, cycles refuse and reporting-depth bounds apply. Member-mode and keeper-mode project-manager rules remain. Human structural reporting needs no conversational profile. Actual human reviewer independence is established through its review route, not a fabricated fresh model.

External is the existing closed product/member pair, unique, not this product, and declared verbatim in external_members. Owners are unique principal ids resolving locally. Meta_keeper is a local member or declared External, separate from Product.keeper; external resolution remains explicitly unknown. Principal IDs are unique. Contacts in one principal cannot repeat an exact full MXID. Names and contacts grant no custody.

All unchanged v4 constraints apply to the enumerated shared objects: project/member/chat-name uniqueness, binding/connection/domain equality, host-project local anchor, path bases, profile requirements/locks, room completeness, agent chat/result/service rules, selector order/multiplicity and required/self ownership. Every role-dependent agent rule uses responsibility. Domain-star naming uses its fixed code; a free title is never a tuple component. Human contacts do not participate in agent-name resolution or caller matching. A human project manager's optional project-room routing connection is the product default binding's connection; it is not a human execution binding.

Profile and binding maps remain nonempty for agent callers and product topology; human records synthesize no entries. Legacy source pins and prospective tuples preserve the exact v4 local-snapshot versus full-admission distinction. Optional room declarations preserve their alias/ID pairs, routing server, closed shape and all-or-nothing completeness without resolution or JOIN claims.

## Canonical public descriptors

Every key is a declaration in the selected snapshot. Within a principal, (purpose,key_id) is unique. One key can have two explicit purpose records, but purpose never widens implicitly. An exact reference contains principal, purpose and key_id and resolves uniquely. Multiple owners/keys and empty key lists are legal; no first, newest or only-active issuer is inferred.

| Algorithm | Encoding | Decoded public bytes | key_id prefix | Purpose |
| --- | --- | --- | --- | --- |
| Ed25519 | ed25519-raw | Exactly32; canonical compressed Edwards point, nonidentity, prime-order subgroup | ed25519: | identity or grant-authority |
| P-256 | sec1-compressed | Exactly33,02/03 prefix, finite on-curve point, exact canonical round trip | p256: | identity or grant-authority |
| BIP-340 | secp256k1-x-only | Exactly32; x below field prime with a valid even-y lift | bip340: | identity only |

Tags are exact and case-sensitive. Unknown tag/encoding/purpose combinations refuse. Unpadded base64url decode/re-encode equality is mandatory. key_id is prefix plus exactly public_bytes; public_sha256 is lowercase hex SHA256 of decoded bytes. PEM/DER/npub, padding, whitespace, alternate encodings and trailing bytes refuse. X-only lowercase hex display is derivable from a BIP-340 record; it is not another stored key.

UTCSecond is exactly YYYY-MM-DDTHH:MM:SSZ, with a real UTC date, no fraction/offset. declared_state_at cannot exceed the explicit query evaluation time. When both validity bounds exist, not_before precedes not_after. Missing bounds prove no current-use authority. Active, retired and revoked declarations remain representable, never current verification; an unseen later revocation is unknown.

Point operations use pinned upstream implementations:

- github.com/oasisprotocol/curve25519-voi v0.0.0-20251114093237-2ab5a27a1729, BSD-3-Clause: CompressedEdwardsY canonical check, EdwardsPoint decoding, nonidentity and IsTorsionFree. Decoding alone may accept noncanonical encodings. Upstream curve/edwards_test.go supplies canonicality, torsion and mixed-order controls.
- github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1, ISC: ParsePubKey on prefix02 plus x-only bytes, then compressed serialization equality. Upstream pubkey_test.go includes compressed positive, field-overflow and off-curve cases.
- The supported fixed Go toolchain's crypto/elliptic P256 compressed decoder/encoder for P-256.

No local field arithmetic or custom curve implementation. Product wrapper controls use independent RFC8032, SEC2 and BIP340 public fixtures with nearby forbidden mutations. Parser agreement alone proves no identity or authority. BIP-340 may declare an owner identity root for later identity verification; it is not an envelope authority root or landing grant signer. Station SPKI, caller SSH, Git and encryption keys remain separate. No chain lookup occurs.

## Compatibility and local snapshot

Governance checks the whole document/graph, including unselected principals/keys, before output. Referenced logical sources/reservations remain unverified in a local snapshot; none is fetched. Existing v3/v4 admission tools and converter retain their versions and refuse v5. This MVP introduces no automatic migration/publication command.

An explicit reviewed v4-to-v5 authoring transformation preserves every original agent field and exact logical id, Matrix name, path and role mechanic: change schema, add principals=[], keep governance absent, copy old role to responsibility and initial title, add kind=agent. Existing service branches remain unchanged. Infer no person, owner, contact or key. V3 authoring may first use the existing v3-to-v4 conversion with its required premises; unsupported extensions still refuse there. A source with the supported pulse extension instead requires the separately reviewed, complete lossless v3-to-v5 authoring ledger described in the environment contract below. No converter drops those facts. Read-only legacy governance normalization is not written conversion.

Default, hierarchy, routes and discovery keep their existing outputs and refuse v5. Explicit governance and environment accept v5 with their documented distinct output discriminators. Legacy v3/v4 governance is undeclared, even when permissive v3 extension names resemble new fields. Humans are never agent callers; no override or fallback is added.

## Privacy, changes and refusal

Credential/live-state prohibition remains. Only public_bytes and key_id of fully validated v5 PublicKey descriptors are governed by the typed public codec instead of the prose secret-value heuristic. After a first secret_value_forbidden result, validate the entire original card, make a validation-only copy replacing just those two validated values, and rerun the full privacy scanner. No other first-pass error is deferred. Every remaining violation refuses; no output or effect occurs between passes. Keep the original card for projection and never emit the scan copy. Field names, all other values, unselected records and v3/v4 privacy checks retain their prohibitions. Declared key state uses declared_state/declared_state_at, preserving the live status ban. Refusal emits one implementation-owned rule/indexed location, no failed value, and no partial view or mutation. A transport stdout-write failure remains separate and may leave partial bytes.

An owner authenticates their public data/change through the admitted intake. Product custody, source review and signed publication remain separate. A self-signature, listed contact, display title or signed Git commit is not grant authority. Card replacement never refreshes a protected issuer pin. Same-exchange authenticated pinning binds the exact product/card/owner/purpose/key and recipient encryption key. Landing signer selection remains a separate owner decision.

## Optional conversational environment declarations

Conversational agents may declare the closed `pulse-member-environment/v1`
extension; human and service branches forbid it. Both governance and environment
readers validate every supplied extension before selection. The
[complete environment contract](../agents/skills/swarma-pulse/references/environment-v5.md)
defines selectors, source issues, coordinate syntax and the distinct self/target
outputs. No execution, account registration or referenced-resource access follows
from these declarations. Existing v3/v4 validation and conversion contracts are
unchanged.
