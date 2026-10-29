# Product-card v4 — complete format candidate

Status: design candidate, 2026-10-29; independent review pending. No v4 validator or converter is supplied. This contract
and the [full fixtures](fixtures/v4/full/README.md) must be accepted before implementing either. The accepted
[specification](specs/product-card-identity-pulse-v1.md) defines identity naming, preservation, eligibility and migration
semantics. The [outline](format-v4-outline.md) records its scope. Neither format acceptance nor a card authorizes a live
registration, room, credential, installation or runtime operation. The gr pilot does not wait for this format.

## Encoding, notation and limits

Input is exactly one UTF-8 YAML mapping. JSON documents are also within that YAML subset. Reject duplicate mapping keys
at every depth, non-string mapping keys, tags, aliases, merge keys and additional documents. Expand v3 anchors during
explicit conversion, before emitting v4. Every object described below is closed: only listed required and optional keys
are legal, including in nested objects. No null values are legal. Optional means absent, never implicit null/default.
Arrays may be empty unless a nonempty condition is stated. All strings are nonempty; booleans are actual booleans and
counts are integers, excluding booleans. No coercion, whitespace trimming, normalization or truncation.

Type vocabulary: `text` is a nonempty UTF-8 string without NUL/control characters; `id` is a portable logical member or
binding/profile/operator/host identifier matching [a-z0-9][a-z0-9._-]{0,127}; product/project/star slugs use the stricter
1–40-character grammar in specification §5. `revision` is an immutable nonsecret locator revision string, 1–200 ASCII
characters from [A-Za-z0-9._:/-]; it is not necessarily a Git SHA. `sha256` is exactly 64 lowercase hex characters.
`ref` is an opaque nonsecret text reference to an independently admitted contract; no embedded credentials. `path` is a
nonempty relative filesystem string with neither leading / nor ~ and no NUL/control characters. All filesystem access
is bounded by an externally admitted root; lexical traversal and resolved symlink escape refuse before destination access.

Maximum raw card size 1 MiB; at most 1,000 projects and 10,000 members; document nesting at most 32 containers, with root
counted as level 1; at most 64 reporting edges on a local superior chain. Unknown field/type, missing required field,
forbidden YAML or exceeded bound refuses the whole document. Safe diagnostics identify rule plus indexed field location,
never raw values, credentials, a backtrace or a partial result. Refusal changes no file, registration or runtime state.

## Closed object field tables

The following required/optional sets are exhaustive. A map means string keys unique within that map; a list preserves
its declared order except where the specification explicitly sorts a query result. Object names refer to these tables.

| Object | Required keys and types | Optional keys and types |
| --- | --- | --- |
| Card | schema: literal product-card/v4; product: Product; external_members: list External; connections: nonempty map Connection; bindings: nonempty map Binding; profiles_root: path; profiles: nonempty map Profile; projects: nonempty list Project; members: nonempty list Member | none |
| Product | id: slug; description: text; privacy: text; keeper: local member id; default_binding: binding key; host_project: project id; topology: Topology | none |
| External | product: slug; member: id | none |
| Connection | homeserver: HTTPS origin text; server_name: canonical DNS domain | none |
| Binding | operator: id; host: id; connection: connection key; identity_domain: canonical DNS domain | none |
| Profile | requirement: Requirement; selected: Selection; exactly one of path: path or registry: Registry | none beyond that exclusive source |
| Registry | repo: Repository; directory: path | none |
| Requirement | context: id; compatible: version-range text | none |
| Selection | version: MAJOR.MINOR.PATCH string; source_rev: 7–40 lowercase hex; lock_sha256: sha256 | none |
| Topology | connection: connection key; space: Room; lobby: Room; escalations: Room | grouped_rooms: list Room; upward_escalations: Room |
| Project | id: slug; description: text; repo: Repository or literal local; location: Location; desired_state: State; orchestrator: local member id; task_system: TaskSystem; channels: Channels | desired_state_ref: ref; orchestrator_mode: literal member or keeper; orchestrator_ref: ref |
| Local Location | kind: literal local; path: path | none |
| Remote Location | kind: literal remote; operator: id; execution_host: id; project_locator: ref | checkout_path: path |
| Local TaskSystem | kind: literal task-board or tasks-md; path: path | none |
| Remote TaskSystem | kind: literal task-board or tasks-md; locator: ref | none |
| Channels | mode: literal shared or separate; team_room: Room | none |
| Common Member | id: id; role: Role; project: project id; parent: Parent; scope: Scope; binding: binding key; identity: Identity; lifecycle: persistent or on-demand; desired_state: State; context_mode: continue, fresh or service-only none; runtime: Runtime | desired_state_ref: ref; capabilities: list Role; clone: path; git_dir: path; station: ref |
| Conversational Member additions | profile: profile key | staffing: list Slot; independent_of_producer: boolean; result_route: a2a or parent-native; chat_admission: ref; a2a: A2a |
| Service Member additions | service_contract_ref: ref; result_route: a2a or parent-native | a2a: ServiceA2a; native_result_ref: ref; selector_policy_ref: ref |
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

A Member is exactly Common plus either Conversational additions or Service additions, selected by role=service.
Optional fields from the other branch are forbidden. Identity and Location use their scheme/kind discriminator and
forbid all keys from the other branch. TaskSystem is selected by its project's Location: local has path only; remote
has locator only. There is no permissive extension object. Parent is either a local member id string or one External
object declared verbatim in external_members. Duplicate External pairs refuse; an External cannot name this card's product.
Qualified logical equality is product/member, independent of display names and resolved MXIDs.

State is active, parked or retired. Role is keeper, orchestrator, developer, tester, architect, reviewer, researcher or
service. Scope is product for keeper, project for orchestrator, member for every other role. Capabilities describe
assignments and grant no eligibility. Omitted orchestrator_mode means the preserved v3.1 member mode only; no other
behavioral default is implied. Reviewers and reviewer slots require fresh context and independent_of_producer=true.

Repository is host/owner/name with each component nonempty and no whitespace, userinfo, scheme, query, fragment, token
or password. Room uses the v3.1 form: local alias #[a-z0-9][a-z0-9._-]{0,63}, or ![A-Za-z0-9_.~-]{8,255}, with optional
:server domain. Full MXIDs use exactly @[a-z0-9._=/-]{1,255}:server, with the canonical server domain; human selectors never
use a short name. Preserve valid legacy localparts verbatim rather than imposing new tuple parsing on them. Domain syntax is the full canonical DNS grammar in specification
§5. An HTTPS origin has no userinfo/query/fragment, and no path except an optional trailing slash. Endpoint host may
differ from the identity domain; a binding's identity_domain must equal its selected connection.server_name.

Kind is proposal, question, answer, decision, ack, notice, offer, request, accept, decline, handover, subscribe,
unsubscribe, owner-needed or system. A subtype is only legal with kinds including system. Duplicate rooms, selector
kinds, capabilities or identical selectors refuse. A human all_events selector must include its explicit rooms. Every
selector room, parent_route and result_destination must occur in that member's rooms. Every spaces entry must occur in
rooms, and no room identified as a Space, including product.topology.space, may occur in any selector. Required/self
ownership, exact owner MXID coverage and any admitted audience policy remain external admission constraints; omission
cannot imply a selector set. The first rooms entry is the member's own team room; grouping, joining, power and chat eligibility remain separate facts.

Profile path resolves under profiles_root; Registry.directory resolves in its named repository. Profile requirement
uses one or two space-separated =, >, >=, < or <= comparators against MAJOR.MINOR.PATCH. The selected version must satisfy
both when two exist. Selected revision/lock is desired selection, not installation evidence. No card field contains a
selected secret-store item, runtime continuation, PID, liveness, current assignment or credential material.

## Relations, paths and class constraints

Project ids, member ids and resolved chat MXIDs are unique. Map references, each member's project/profile/binding and
product.host_project/keeper/default_binding must resolve. Local parents resolve and form an acyclic graph. The single
product keeper has an External parent; every orchestrator reports to that keeper. Other members report to the manager
of their project. Unavailable external ancestors are explicitly partial in a projection; they are not local dangling refs.
All product-scope members name host_project. The host project is local with location.path exactly ., anchoring the
canonical card directory. Service parents are the declared project manager.

In member management mode, project.orchestrator names that project's orchestrator. In keeper mode it names the product
keeper and requires orchestrator_ref; a separate orchestrator member of that project may only be parked or retired.
An on-demand member is stable across assignments, not a newly generated member. Staffing is optional desired capacity
on keepers/orchestrators only. station is optional on ANY member; retain exactly the reference to its registered native task-board station,
including through cutover/rollback. It is never treated as live assignment state or discarded because it is not local.

Local Location.path resolves against the card directory. Local TaskSystem.path resolves against that project's resolved
directory. Remote task locator is never opened locally; checkout_path resolves only against the explicitly admitted
product root on that remote host. Member clone/git_dir resolve against the card directory; profiles_root does too.
Conversion records each original base and target outside the card and computes relative v4 paths preserving the exact
resolved targets. Never create a dummy checkout or fetch a remote project to validate its declaration. An optional layout
check inspects local declarations only; missing/escape/local symlink failures are distinct from remote unavailability.

Managers must have A2a and a conversational profile/runtime. Other non-service roles require result_route: parent-native
has no a2a block; a2a requires both A2a and independently admitted chat_admission. This does not enact a new chat class.
Every member has a prospective New Identity or, for an already admitted chat member only, a Legacy Identity. A prospective
non-chat tuple creates no Matrix account and may not select legacy-exact. A member with an a2a block must have its
connection equal the selected binding's connection. No resolved chat identity may collide with another member.

Service role requires scope member, context_mode none and runtime kind none; profile/model/effort, staffing and agent
continuation are forbidden. For result_route=a2a, ServiceA2a is required, native_result_ref forbidden, and chat_admission
is not a service field: the separately admitted service_contract_ref identifies its chat/result admission. required/self
are explicit empty lists unless selector_policy_ref supplies a separate nonempty service-policy disposition. For
parent-native, native_result_ref is required; a2a and selector_policy_ref are forbidden and identity is prospective New
only. The runtime must refuse service_session_forbidden before session/hook/model effects. Schema acceptance alone does
not qualify a service runner, human selector or conversation launcher.

## Legacy sources, new tuples and deterministic conversion

LegacySource identifies supplied immutable bytes outside the card, not a fetch instruction. Verify SHA-256 and revision
against the explicit source manifest. For v3-card, locate the unchanged logical member key and derive its old MXID from
the source's explicit/default binding, identity_prefix + member.id and connection.server_name. Verify source connection
endpoint/server against the selected legacy connection. Return the attested MXID verbatim; no new registration. Missing
source, wrong rehashed id, changed connection/MXID/member key or a precomputed-output-only pin refuses legacy validation.
For service-descriptor use the separately admitted immutable descriptor to resolve the unchanged service key/MXID and
connection; it cannot be replaced with a guessed v3 member. Source descriptors are supplied by an admitted resolver
interface; the synthetic descriptor fixture is evidence of a mapping, not a newly shipped service onboarding API.

New Identity uses the full domain/product/project/role-code/star equation and exact role codes from specification §5.
A reviewed star reservation must exist for domain/product/primary-role across projects; duplicate/reserved-name and
64-character budget checks run before any side effect. Retired reservations are retained. No host-derived prefix,
normalization or random suffix. The registry/reservation custody is separate from a card's desired tuple.

Explicit conversion has these preconditions: a valid source v3.1 card under its own reader; source bytes/revision pin;
a supplied reviewed host-project choice; exact legacy external-parent mappings where the old reference lacks a product;
source/target path bases and admitted roots; explicit local/remote dispositions for any changed location. Missing input
refuses conversion, never guesses. Every source project/member/profile/desired declaration must have a target or an
explicit lossless representation. Unsupported legacy extension keys refuse conversion rather than being silently lost;
station is a supported optional field. Existing valid logical member ids are never normalized or renamed by conversion.

Deterministic output selects legacy-exact for EVERY existing chat member and materializes each binding. Non-chat members
receive only explicitly supplied reviewed prospective tuples; conversion never allocates a star. Remove identity_prefix
only from target bindings, retaining it in the pinned source. Preserve source default/explicit binding semantics, required/
self lists, runtime/profile/locks, staffing, topology, desired states, station and the same resolved relative targets.
Map an old external id only through the supplied External mapping. Anchor product members to the reviewed host project.
No account registration, room membership, credential, session or automatic file replacement follows from conversion.

A separately reviewed per-member cutover changes only that member's identity/binding and associated desired a2a fields.
A non-pilot identity or other desired fact must remain equal. Rollback restores the complete prior member record exactly;
old connection declarations remain while referenced. Operational seven-stage migration evidence never becomes card fields.
The finite fixture set gives concrete expected mappings and counterexamples, not execution of a converter implementation.

## Fixture verdict vocabulary

Fixture refusal labels identify the operation and invariant: conversion_station_preservation compares an attempted
conversion with its source; it does not make optional station mandatory on a standalone card. legacy_source_mismatch
and source_digest_mismatch check supplied source authority; unknown_field enforces closed objects;
service_selector_policy_required rejects nonempty service selectors without their disposition; service_runtime_forbidden
rejects model/effort fields in the service runtime shape. service_session_forbidden is the distinct agent-session launch
refusal for an otherwise valid service record. These labels specify future expected behavior, not a shipped error API.
When several invariants fail, a later implementation must report every applicable safe rule/location or a separately
reviewed deterministic diagnostic order; it must never claim success from an unrelated setup failure.
