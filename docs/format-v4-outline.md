# Product-card v4 format outline

Status: bounded design outline, 2026-10-29. The complete closed schema and full-card conversion fixtures remain required
before validator or converter coding. This outline narrows that work; it neither revises the accepted identity contract
nor makes the format a prerequisite for the first gr account migration.

The normative behavior is the [identity and migration specification](specs/product-card-identity-pulse-v1.md). Existing
v3.1 documents retain schema product-card/v3 and their own validator. A v3 document is never interpreted as v4 and a v3
reader must refuse a v4 document. No file is converted on read, registered or launched by this format.

## Changes with a migration purpose

| Area | V4 change | Preserved contract |
| --- | --- | --- |
| Product | Add host_project referencing a declared host project. | Product id, description, privacy, keeper, default binding and topology remain desired facts. |
| Project location | Replace local-only path assumption with closed local/remote location alternatives. | Repository identity, manager mode, task-system reference, desired state and channel declarations remain explicit. |
| Binding | Add identity_domain for new resolution; materialize each chat member's binding. | Operator/host and named connection remain references, never authority or credential contents. |
| Member identity | Exactly legacy-exact with pinned source, or domain-star-v1 with star; unchanged logical member id. | Stories remain assignments to stable members, not new identities. Preserve every legacy MXID and connection during conversion. |
| Member scope | Explicit product/project/member scope; every member names a project. | Product-level custody anchors to host_project; parent graph and combined manager mode retain their requirements. |
| Service member | Service role, runtime none, context none, explicit contract/result route and accountable project parent. | No profile/model/agent hooks/session; empty baseline service selectors remain explicit. |
| Paths | Preserve each source field's v3.1 base and resolved target when converting/relocating. | No absolute/home paths, symlink root escapes or implicit remote checkout. |

No migration state machine is stored in the card. Prepared/registered/provisioned/qualified/cut-over/proven/retired are
operational evidence states. The card holds one selected desired identity; cutover changes only the reviewed member's
identity, binding and associated desired route fields. Full prior records and rollback receipts remain outside the card.

## Explicit v3.1 carry-over inventory

The complete schema must account for every existing v3.1 field below without redesigning its purpose. An optional field
stays optional unless the accepted migration contract explicitly adds a requirement. Existing desired facts are not
installation evidence. Any unlisted field in the eventual closed schema must be refused, not silently discarded.

- Root: schema, product, external_members, connections, bindings, profiles_root, profiles, projects, members.
- Product: id, description, privacy, keeper, default_binding, topology. Topology: connection, space, lobby, escalations,
  optional grouped_rooms/upward_escalations. Grouping grants no membership/power and Spaces are never subscription sources.
- Connection: homeserver/server_name, both public routing metadata only. Binding: operator, host, connection; the old
  identity_prefix remains in the pinned legacy source, not new tuple resolution.
- Profile: exactly one path or registry source; registry repo/directory; compatible requirement context/range; immutable
  selected version/source_rev/lock_sha256. Preserve profile roots, selected locks and compatible selection checks.
- Project: id, description, repo or explicit local-only designation, desired_state/optional desired_state_ref, orchestrator,
  optional orchestrator_mode/orchestrator_ref, task_system kind/path or explicit remote locator, channels mode/team_room.
  Replace the project path with the explicit location alternative; record its original target during conversion.
- Conversational member: id, role, project, parent, profile, runtime kind/model/optional effort, lifecycle, desired_state,
  optional desired_state_ref, context_mode, optional capabilities, clone/git_dir, binding, staffing and independence fields.
  Carry worker result_route/chat_admission where applicable; naming never grants chat admission.
- A2a declaration: connection, parent_route, rooms, optional spaces, subscriptions.required/self. Retain both exact full-MXID
  owner selectors where required; preserve required/self ownership. Human selector rooms/all_events and room selector
  kinds/subtypes retain v3.1 meanings. A service exception must be explicit, not inherited conversational policy.
- Staffing: role, profile, count, lifecycle, context_mode, optional independent_of_producer. Slots remain desired capacity;
  no assignment, live generation, current task, status, PID, secret or credential-store contents may appear.

The final field table must settle field types, allowed enums, requiredness and cross-references for these existing facts
and the narrow changes. This outline deliberately does not claim a complete machine-validating schema. In particular,
qualified external references and remote task locators need closed shapes before coding; no permissive catch-all is implied.

## Identity and service discriminators

A chat member's legacy-exact branch has only scheme/mxid/source_ref. The source_ref has kind/revision/sha256/member and
pins actual resolver inputs, including unchanged source ids, selected connection and prefix. Independently derive the
old name; a hash of an expected output map is insufficient. A new branch has only scheme/star; derive full domain,
product, project, stable role code and star from the selected desired record. No legacy provisioning or mixed branch keys.

Non-chat members may have a prospective new tuple but no Matrix account or legacy-exact branch. Sessionless services
with result_route parent-native have an explicit admitted native result reference and no a2a block. Chat-admitted services
have explicit named connection/project audience/result destination, zero baseline required/self selectors and no
conversational profile, model, effort, continuation or hooks. Nonempty service selectors need a separate policy reference.
An agent-session launch must refuse service_session_forbidden before any hook/session/model effect; the schema alone
cannot prove this runtime property.

## Complete fixture set required next

The [fixture requirements](fixtures/v4/fixture-requirements.json) are review inputs, not runnable full cards. Produce full
synthetic v3.1 and v4 cards containing profiles, topology, projects and every required field, with independently derived
resolution/conversion expectations. Expand legacy YAML anchors explicitly during conversion. Pin original bytes before
conversion and preserve logical ids, relation targets, profile selection, route fields and resolved relative targets.

| Fixture family | Positive control | Required counterexample/invariant |
| --- | --- | --- |
| Legacy conversion | Default and explicit source binding; complete v4 selects legacy-exact for all existing chat members. | Missing pin, wrong rehashed id, changed MXID/connection/key, duplicate identity or old-reader reinterpretation refuses without output. |
| Mixed cutover | One new-tuple pilot, unchanged non-pilots, exact prior-record rollback. | Non-pilot mutation or partially written card refuses; no registration follows from conversion. |
| Service | Complete zero-selector a2a service and parent-native service; seven-stage evidence remains outside the card. | Implicit owner selectors, a profile/model/session, or a2a on parent-native refuses specifically. |
| Location | Local sibling and declared remote reference preserve original targets. | Absolute/home/empty path, symlink escape or remote interpreted as missing local path refuses. |
| Carry-over | Profile locks, manager modes, topology, room/owner selectors and existing non-chat routes remain equal. | Changed required selectors, profile selection, live state, credential fields or unknown keys refuse. |
| Parser and bounds | UTF-8 document within explicit byte/count/depth/hop bounds; valid 64-character tuple. | Duplicate keys, tags/aliases/merge keys/multiple documents, oversized/deep input, 65-character tuple or truncation refuses. |

Expectations must distinguish explicit conversion policy from validation. A v3 source may contain anchors accepted by
its own reader; the converter expands them before producing v4, whose parser refuses aliases. No fault injection touches
real cards, credentials or running members. The eventual checker must demonstrate a bounded known violation control and
nearby valid controls; source-derived fixture consistency now does not qualify product refusal paths.

## Review and implementation boundary

Review the complete schema and full fixtures against the accepted preservation/service contracts before coding. Then
implement and independently qualify validator/converter behavior; run all selected lanes and report every result. The
first gr pilot may use separately qualified existing a2a primitives while this work proceeds. A source or installation
gap blocks only the affected runtime operation, and no format draft authorizes a registration, consumer switch or room
creation. Nonessential projection implementation remains behind ready migration work.
