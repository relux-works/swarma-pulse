# V5 declared environment and explicit targets

`pulse me --local-unsigned --view environment` accepts one strict pinned
`product-card/v5` in addition to the existing v3 product/company self contract.
It validates the original complete document before resolving the invoking
`A2A_AGENT`. No caller override, referenced source read, configuration lookup,
network operation, card write or runtime launch is introduced.

```sh
.temp/bin/pulse me --local-unsigned --view environment \
  --card /absolute/path/product-card.yaml --revision sha256:LOWERCASE64HEX \
  --format json
.temp/bin/pulse me --local-unsigned --view environment --member vault_dev \
  --card /absolute/path/product-card.yaml --revision sha256:LOWERCASE64HEX \
  --format json
```

Use an admitted binary, actual card path and independently checked byte pin.
The file/UID/ancestry boundary and local unsigned trust limit in the
[query contract](query-contract.md) apply unchanged. Query success grants no
identity, managerial, filesystem, signing or launch authority.

## Source declaration

Only conversational `kind: agent` members with responsibility other than
`service` may supply `pulse`. Human/service records forbid it. The closed object
requires `schema: pulse-member-environment/v1`; optional keys are
`a2a_config_profile`, `working_root` and `network`. Network is exactly
`{mode: direct}` or `{mode: proxy, ref: SELECTOR}`. Selectors match
`[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}`, with case preserved.

Working coordinates are nonempty UTF-8 strings without control characters,
leading slash/tilde, backslash or an initial URI/drive scheme. Dot and sibling
components remain legal data. No existence, containment, realpath or target read
is implied. The eventual consumer independently admits each coordinate.

Every present extension is checked, including unselected members, by both the
environment and governance readers. Raw nulls retain `null_value`; privacy and
bounds precede object grammar. Missing schema/mode/proxy-ref is `missing_field`;
unsupported present schema/mode is `enum_value`. Other isolated grammar rules
are `object_shape`, `unknown_field`, `environment_selector_format` and
`environment_path_format`. Query decoding maps strict v5 validation issues to
`card_invalid`; legacy v3/company error mapping is unchanged. Only already
validated public key values retain their existing typed privacy exception.

## Select a subject

Without `--member`, the subject is the actual caller and the output discriminator
is `pulse-view/v6`. Its environment remains `pulse-environment/v1`.

`--member VALUE` is optional only on `--view environment` and only for v5.
VALUE is the complete exact, unqualified `member.id`: no trimming, normalization,
portable-id restriction, slash splitting, MXID/title interpretation or roster
lookup. Missing/empty/duplicate selector values or use with another view is
usage before source access. Legacy target mode refuses `unsupported_schema`.

After complete v5 validation, resolve the actual caller with the existing exact
matching/ambiguity rules. Humans never match; a matching service caller refuses
`environment_invalid`. An eligible agent may explicitly select itself. A
different target requires the caller's responsibility to be keeper/orchestrator
and the target's exact `parent` to equal that caller's local member id. Ancestors,
peers, grandchildren and another manager's members are excluded. A title such as
“keeper” grants no privilege.

Targets must be conversational agents with a valid declared profile/runtime.
They need no chat admission or running identity. Parked/retired states remain
readable declarations. A seat absent from `members` is never materialized.
The caller context remains unchanged throughout.

## Closed output

All objects emit exactly the listed fields; optional values use explicit nulls.
JSON is one buffered object plus newline. Existing encoder and transport limits
apply.

| Object | Exact field names |
| --- | --- |
| v6 root | schema, mode, query, caller, known_limit, sources, self, peers, direct_reports, descendants, superiors, projects, products, boundaries, hierarchy_status, observations, query_observed_at |
| v7 target root | The same seventeen names, replacing self with target |
| Source receipt | id, schema, revision, sha256, card_root, freshness, observed_at, validation |
| Self or target | key, kind, role, responsibility, scope, desired_state, resolved_name, execution_binding, profile, project, project_state, level, environment |
| Environment | schema, state, profile, runtime, a2a, working_root, network, missing |
| Environment a2a | agent, resolved_name, binding, operator, host, connection, server_name, config_profile |

Self mode has schema `pulse-view/v6` and query `{view: environment}`. Target mode
has schema `pulse-view/v7`, query `{view: environment, member: VALUE}`, a `target`
object and no `self`. `caller` is always exactly `{agent: ACTUAL_CONTEXT,
cryptographically_verified: false, trust: single-user local trust}`.
Mode is `local-unsigned`; known_limit is exactly
`caller not cryptographically verified; single-user local trust`.

There is one receipt for the supplied original v5 bytes. Its id/schema are the
declared product id and `product-card/v5`; revision is the checked content pin,
sha256 its lowercase digest and card_root the cleaned actual input parent.
Freshness is `unknown`, observed_at null and validation is exactly
`local declarations only; source attestations and star reservations unverified`.
No source attestation or star reservation has been authenticated.

Subject fields derive only from that subject. Kind is agent; role is its free
title, responsibility its mechanical code. Scope and level are product for
keeper, project for orchestrator and member otherwise. Project is
`product-id/project-id`, with project_state declared. Execution binding is
exactly `{key, operator, host, connection}`. Profile is exactly
`{requirement: {context, compatible}, selected: {version, source_rev,
lock_sha256}}`. Desired state is not liveness.

Projects contains only that subject's `{key, repo, desired_state, location}`.
Location is exactly `{kind: local, path}` or `{kind: remote, operator,
execution_host, project_locator, optional checkout_path}`. Relationship, product
and boundary arrays are empty. Hierarchy status is exactly
`self-only; hierarchy not disclosed` or, for target mode,
`target-only; hierarchy not disclosed`. Observations contains exactly runtime,
registration, installed_profile and assignment, each `{state: unknown}`.
Query_observed_at is actual query time in UTC RFC3339Nano, not resource freshness.

Environment profile is exactly `{key, source, requirement, selected}`. Local
source is `{source: PRODUCT_ID, base: profiles_root, profiles_root: RELATIVE,
path: RELATIVE}`; registry source is `{source: PRODUCT_ID, base: registry,
repo: REPOSITORY, directory: RELATIVE}`. Interpret local profiles_root relative
to the actual card parent and profile path relative to it. Registry directory
uses its declared repository. No profile/repository is opened. Runtime is
exactly `{kind, model, effort}`; absent effort is null, with no default or marker.

Working_root is `{source: PRODUCT_ID, base: card, path: RELATIVE, origin}` or
null. Precedence by presence is pulse.working_root, member.clone, then local
project.location.path; origin records that exact choice. Invalid present values
refuse rather than fall through. Neither a remote project_locator nor its
checkout_path supplies this fallback. Explicit pulse/clone roots remain
card-relative coordinates even for a remote project. Network is exactly the
declared branch or null, with no default direct mode.

Self environment uses schema `pulse-environment/v1`; a2a has the caller's exact
declared full MXID/localpart and its binding/connection facts. Missing is the
sorted subset of a2a.config_profile, network, working_root for null fields.

Explicit targets use schema `pulse-environment/v2`. A chat target's subject
resolved_name and a2a resolved_name are its exact declared MXID; a2a.agent is its
bare localpart. A non-chat target has all three identity values null even when
a prospective identity is declared. Binding/operator/host/connection/server_name
come from the target's explicit binding/connection; config_profile retains its
own value or null. No HTTPS endpoint is exported. Target missing is the sorted
subset of a2a.agent, a2a.resolved_name, a2a.config_profile, network, working_root.
The first two occur together exactly when target.a2a is absent. State is complete
iff missing is empty; otherwise incomplete. Neither state proves launchability.
The v1 environment grammar is not widened by target nullable identity.

## Refusal order and compatibility

Public CLI order is selector validation, real/effective UID equality, empty
caller-context refusal, safe source read, source digest, remaining caller-context
validity, schema/strict validation, caller resolution, subject eligibility and
projection. The internal byte helper's digest-before-context ordering does not
replace public CLI controls.

After source validation and caller eligibility, target errors are
`target_unknown`, `target_ineligible` for human/service, `target_forbidden` for
the relationship, then `environment_invalid` for inherited coordinates.
Duplicate member ids are whole-card `card_invalid`; caller ambiguity remains
`caller_ambiguous`. There is no target_ambiguous branch. Validation refusals have
exit 1, empty stdout and exactly `RULE NAME: $` plus newline; usage is exit 2.
`output_write` may interrupt final delivery and is a distinct transport failure.

Legacy company/v3 no-selector environment keeps exact v2 behavior. V4 environment
still refuses. Default, hierarchy, routes and discovery continue to refuse v5.
Governance keeps its v5 projection and never exports pulse. The v3-to-v4
converter still rejects unsupported pulse instead of stripping it.

## Authoring and usable adoption

Explicit custodian authoring requires immutable original bytes/pin, reviewed
host/external/path mappings and a full two-way declaration inventory. Include
the keeper and all unselected members, profiles, bindings, rooms and selector
multiplicity. Record a projectless keeper's required host-project assignment
separately from preserved facts; only its recorded project/project_state/projects
changes are allowed. Preserve the resolved root or its absence and exact
missing/state. If required host fallback invents an unrepresentable root, refuse
lossless authoring; a separately approved new root is a declaration change.
An already-conflicting keeper project also requires a separate approved change.

Each non-chat agent requires explicit reviewed prospective binding/domain-star
identity and required connection/reservation premises. Record those as new facts,
preserving absent a2a and native route. Missing/conflicting premises refuse.
Prospective declarations create no account or runtime. Services keep their
separate branch. Owner public records and key purposes retain their own intake.

Qualify the reviewed reader, exact authored candidate, preservation ledger and
actual manager's same-target input before a separately reviewed canonical
card/pin swap. Record staging and intended canonical parents and their explicit
coordinate mapping separately. Retain exact original rollback bytes/pin. After
replacement re-read canonical bytes/pin and obtain separate canonical self and
manager target receipts. A partial/mismatched card-pin state refuses safely.

Usable adoption additionally requires the separately admitted manager launch of
the actual stable target using canonical selections, with actual profile/model
readback. Query success, incomplete preservation and a staged receipt do not
satisfy this gate. Bind all receipts to the same reader/candidate/path/target.
Missing readback remains pending. Card rollback restores bytes/pin, not a running
session; it grants no reset/restart. Pulse performs none of these custody or
runtime operations itself.
