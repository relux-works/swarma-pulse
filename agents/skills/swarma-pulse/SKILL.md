---
name: swarma-pulse
description: Use the shipped Pulse CLI for local unsigned self, declared-environment or caller-discovery views, interpret identity/source boundaries, or select the product-card validator, converter and change checker. Use for these module interfaces; account registration, runtime launch and proposed signed or plugin interfaces are separate operations.
---

# Use Pulse

Read [the query contract](references/query-contract.md) for the complete input, trust
and output boundary. Use the shipped `pulse me --local-unsigned` command for a
self-only view of one explicitly selected `product-card/v3` or `product-card/v4` snapshot, with output
schema `pulse-view/v1`. Add `--view environment` for self declarations from one
v3 product or v1 company card. Explicit `--view discovery` returns caller-relative
peers, reports and superiors from a pinned local source universe. Both opt-in
views return `pulse-view/v2`; read their sections in the query contract before use.
Use `--view hierarchy` for caller-relative relationships from one pinned v3/v4
product card, returning `pulse-view/v3`. See the query contract's
[hierarchy section](references/query-contract.md#one-card-hierarchy-view).
Use `--view routes` for the same hierarchy plus declared DM descriptors and exact
product/project room routes (`pulse-view/v4`). See the [route contract](references/query-contract.md#declared-routes-view).
V4 supports the default self-only and explicit hierarchy/routes views. Its complete local declaration is checked;
external source attestations and star reservations remain unverified and are
marked in `source.validation`. The v4 admission tools remain separate.

The included references provide this skill's documentation without relying on
files outside its folder. Their build and fixture commands run from an authorized
Pulse source checkout, not from an installed skill directory. This documentation
package supplies no binary, source checkout or fixture files and grants no access
to a real card. Select those inputs separately before running a command.

## Select the interface and inputs

- Select an existing admitted binary or build the repository's `cmd/pulse` as
  shown below, using the supported fixed Go toolchain in the
  [build guide](references/build-and-tools.md#build-and-run). Native qualification is macOS arm64;
  do not infer another platform's qualification from it. Before attributing a
  built executable to a reviewed commit, read back its build metadata against
  the selected source as described in that guide.
- Select one absolute, clean card path and an independently checked SHA-256 of
  its exact bytes. `--revision` is that content pin, not a Git lookup or a
  canonical/current-card assertion. Do not copy or relocate a real card to change
  its path interpretation.
- Use the invoking session's existing public `A2A_AGENT` localpart. Resolution is
  exact `binding.identity_prefix + member.id` for v3 chat members, through the
  declared or default binding. For v4 chat members, match the exact localpart of
  `identity.mxid` (`legacy-exact`) or the declared domain/product/project/role/star
  tuple (`domain-star-v1`). Exactly one match is required across the selected card or
  supported discovery universe. A full MXID, normalized
  punctuation or an arbitrary member override is not a caller selector.
- Pass **all** selectors explicitly: `me --local-unsigned --card ABS
  --revision sha256:LOWERCASE64HEX --format json`. There is no query config file,
  implicit card search, `--member`, `--config`, signature,
  policy, trust or clock override. Additional cards require the explicit discovery
  selectors below. Missing mode refuses instead of falling back.

The card must be an invoking-UID-owned regular file of at most 1 MiB. Real and
effective UIDs must match. The complete path must have nonsymlink ancestors owned
by root or that UID, without group/world write permission; the file must also
exclude those write permissions. Missing/unsafe paths, symlinks, directories,
FIFOs and devices refuse. Report the selected input problem; do not weaken
permissions or discover another source as a fallback. Root and the same UID
remain trusted writers; these checks do not isolate same-UID processes.

## Inert example and nearby refusal

These commands use only the repository's synthetic fixture. Run from the
repository root when building and executing are authorized. `.temp/bin/pulse` is
a local build destination, not a claim that a binary is already installed on PATH.

```sh
mkdir -p .temp/bin
GOTOOLCHAIN=local go build -trimpath -mod=readonly -p 2 -o .temp/bin/pulse ./cmd/pulse
pulse_card="$(pwd -P)/docs/fixtures/v4/full/source-v3.json"
pulse_revision='sha256:70ec21346a86efda80dfcf0433b0405acf41ab25751b00b1fdcad25ee504fc11'
A2A_AGENT=station_beacon_orch .temp/bin/pulse me --local-unsigned \
  --card "$pulse_card" --revision "$pulse_revision" --format json
```

The literal pin is the SHA-256 of the unchanged tracked fixture bytes. If those
bytes differ, verify the intended fixture and its new exact pin rather than
reusing this value. Its embedded profile selections are synthetic declarations,
not installed-profile or credential evidence. The command-local context assignment
is for this synthetic example only; it does not register an account or change the
session's messaging identity. For real use, retain the existing session context.

The valid request returns exit 0, no stderr and one complete JSON document.
Relevant values are `member` and `self.key` = `aurora/beacon_orch`,
`self.project` = `aurora/beacon`, `self.project_state` = `declared`, and
`self.resolved_name` = `@station_beacon_orch:relux.works`.
The project location is `../beacon`, relative to the actual `source.card_root`
(the selected card's parent), not the shell's working directory.

Keep the same fixture, context and pin, and omit the required explicit mode:

```sh
A2A_AGENT=station_beacon_orch .temp/bin/pulse me \
  --card "$pulse_card" --revision "$pulse_revision" --format json
```

Expected: exit 2, empty stdout, exactly `RULE usage: $` followed by a newline on
stderr. Selector validation precedes file access. Do not retry by silently adding
unsigned mode to a request that intended a signed caller.

## Select an opt-in view

For a one-card hierarchy, add `--view hierarchy` to the baseline invocation.
For declaration-only DM and room routes, select `--view routes` instead; it reads
one pinned product card and performs no transport or membership operation.
Both accept v3/v4 product cards, validate every member, and return full-MXID
peers, direct reports, descendants and nearest-first superiors. External parents
stay unresolved; non-chat names remain null. Both read no additional file or remote
reference and refuse discovery source selectors. Hierarchy returns `pulse-view/v3`;
routes returns `pulse-view/v4` with declared DM/room arrays. In both,
self keeps its baseline fields plus level, related summaries keep their seven
public fields, and related local/remote project locations retain their source base.
No environment or DM provisioning is implied. Default/environment views remain
unchanged. Discovery now preserves a v3 product keeper's external parent when its
alias also matches a local member ID: it returns an external boundary instead of
`relation_conflict`. All other discovery output remains unchanged.

For self's declared environment, add `--view environment` to the baseline
invocation. It reads only that card and never follows its references. The result
has `self.level` and `self.environment` with profile source/pins, runtime,
a2a connection/config selection, working-root locator and affirmative network
selection. Missing optional launch declarations produce `state: incomplete`,
nulls and sorted `missing`; malformed supplied extensions refuse
`environment_invalid`. Do not inspect actual homes or supply guessed defaults.
Even `state: complete` grants no execution authority. All supplied member
extensions are validated, but only self's environment is returned.

Discovery requires all of these selectors, including for an empty additional list:

```text
pulse me --local-unsigned --view discovery \
  --card ABS --revision sha256:LOWERCASE64HEX \
  --sources ABS --sources-revision sha256:LOWERCASE64HEX --root ABS --format json
```

Select the source manifest and read root through their existing input custody.
The manifest is closed JSON `pulse-sources/v1` with `cards`, an array of additional
`{id, schema, path, revision}` entries. Its paths use the primary card's parent,
not the manifest directory. Each file is pinned; the primary is not repeated.
All input files must remain inside the explicit absolute clean read root with
the existing nonsymlink, owner and write-mode checks. Discovery source selectors
with another view refuse as usage before input access. See the query contract's
[company and discovery section](references/query-contract.md#company-inventory-and-caller-discovery)
for the exact schema, containment, composition, refusal and limit rules.

A company inventories products using company-card-relative host/card references.
Local `product-card.yaml` may be directly or more deeply inside its declared host;
normalization must preserve strict component containment. Keep every real card
in place. A product's own project/profile/clone locators keep its own base.
No reference triggers an implicit read: deliberately unsupplied products and
external parents produce partial boundaries. An explicitly supplied missing
file instead refuses the whole query. Remote inventory remains opaque and
unavailable; supplying it as local refuses, with no fetch or coordinate mapping.

Related summaries have exactly key, role, level, desired_state, resolved_name,
project and parent. They exclude others' environments, profile/config/private
routes and working roots. Same-level peers mean responsibility level, not equal
graph depth. Superiors are nearest-first; reports retain parked/retired desired
states. A partial hierarchy ends at declared unresolved boundaries. A complete
hierarchy is complete only relative to supplied declarations.

## Interpret the result without expanding authority

| Value | Interpretation |
| --- | --- |
| `self.resolved_name` | Full source-declared `@localpart:homeserver`; preserve the complete ID in shared display and account addressing/comparison. A friendly label may accompany it. |
| `caller.agent` | Unverified local context; not a qualified account address or authenticated sender. |
| `member` (v1), `self.key` | Logical `card-id/member-id`, separate from the Matrix account. |
| `self.execution_binding`, `self.profile`, `projects` | Declared desired facts, not current process, installed-profile or scheduling evidence. |
| `source` (v1) or each `sources` item (v2/v3): schema, revision, sha256, card_root | Supplied schema, exact byte pin and real input-parent coordinates. Freshness is `unknown`; `observed_at` is null. |
| `query_observed_at` | Query UTC time, not card freshness or registration time. |

`@station_beacon_orch:alpha.example` and
`@station_beacon_orch:beta.example` are different accounts; repeated identical
full IDs are the same account. Use the selected connection's `server_name`, not
its HTTPS endpoint or execution host. Account comparison and message-event
deduplication are different contracts. Pulse supplies data; it implements no
shared renderer, messaging operation or consumer deduplication. Downstream
compliance and adoption require their own evidence.

Every result states `caller.cryptographically_verified: false`,
`caller.trust: single-user local trust`, and the exact known limitation:
`caller not cryptographically verified; single-user local trust`. A same-UID
process can claim another valid local context; this mode cannot detect that
impersonation. Self-only output and a card's desired role do not authorize an
operation, filesystem access, account action or subordinate launch.

In the default and environment views, `peers`, `direct_reports`, `descendants` and `superiors` are
empty. `hierarchy_status` is `self-only; hierarchy not disclosed`; empty lists do not
prove that no such members exist. Runtime, registration, installed-profile and
assignment observations remain `unknown`. A member without a declared project
returns `self.project: null`, `self.project_state: undeclared`, `projects: []`;
do not invent a project or appointment.

## Distinguish intended refusals

| Neighbor of a valid request | Expected behavior |
| --- | --- |
| Omitted/duplicate/unknown selector, wrong format, malformed content pin | Exit 2, `RULE usage`, empty stdout; no unsigned fallback. |
| Well-formed mismatching byte pin | Exit 1, `RULE source_digest`, empty stdout. |
| Missing/empty context | Exit 1, `RULE caller_context`, empty stdout. |
| Valid but unmatched local context | Exit 1, `RULE caller_unknown`, empty stdout. |
| Valid card with two bindings matching that localpart on different homeservers | Exit 1, `RULE caller_ambiguous`, empty stdout; no implicit domain choice. |
| Unsupported schema or invalid complete supplied graph | Exit 1 with `RULE unsupported_schema` or `RULE card_invalid`, no conversion or partial view. |
| V4 selected with `--view environment` or as the primary discovery card | Exit 1, `RULE unsupported_schema`, empty stdout. V4 supports default self-only and explicit hierarchy/routes. |
| V4 declared in the discovery sources manifest | Exit 1, `RULE source_manifest_invalid`, empty stdout; the manifest's accepted schemas are unchanged. |
| V4 aliases, merges, explicit tags, nulls, unknown fields or inconsistent local identity declarations | Exit 1, `RULE card_invalid`, empty stdout; strict v4 parsing uses the original pinned bytes. |
| Unsafe, missing or unreadable input | Exit 1 with the applicable input rule; no substitute source or input writes. |

Diagnostics use `RULE name: $` plus newline. Every supplied card and its complete
graph are validated,
including relationships, before projection; accepted bounded aliases and legacy
extensions retain the v3 reader semantics. Do not impose v4's closed-field rules
on v3 or accept only the self record from an invalid graph. A final stdout
transport failure returns `RULE output_write`/exit 1 and can interrupt delivery;
that is distinct from an intended pre-output refusal with empty stdout.
For v4, local graph validation does not verify referenced source bytes, source
attestations, logical-key attestations or external star reservations. The result
explicitly says `source.validation: local declarations only; source attestations
and star reservations unverified`. A returned name grants no registration or
execution authority; use the separate validator with admitted premises for that
contract. Source references and declared paths are never followed.
The CLI writes no card or persistent state. To stop using it, stop invoking it or
select a prior reviewed binary; preserve inputs and any separately owned data.

## Other shipped tools and extension boundary

Choose these explicit repository entrypoints only for their own task:

- [Validator](references/validator.md): `cmd/card-v4-validate`, including its
  explicit `--version 3` source-reading mode and v4 premises/layout options.
- [Converter](references/converter.md): `cmd/card-v4-convert`; an explicitly
  selected v3 source is published to a separate selected v4 destination. It is a
  write operation, not a query prerequisite; its exit 3 requires inspecting the
  published result before retry.
- [Desired-change checker](references/change-check.md):
  `cmd/card-v4-check-change`; pinned prior/candidate/rollback and explicit
  premises, without writes. See its contract for exact selectors.

The [tool table](references/build-and-tools.md#tools) gives exact build/invocation
commands. Pulse has no plugin registration API, remote enrollment, account
registration or launcher. No extension-registration example applies to this
implementation. The `.codex/skills/swarma-pulse` and
`.claude/skills/swarma-pulse` repository links point to this source directory for
runtimes that support project-local skill discovery. They do not install a global
skill, change a selected profile, register an identity or prove runtime adoption.
Use the chosen environment's supported reviewed packaging/adoption route for
those separate changes; this product supplies no profile-install command.

Signed callers, authenticated visibility, v5 composition, compact naming/migration and
remote/plugin interfaces in the design documents remain proposals. Do not invoke
unimplemented `pulse verify` or treat illustrative future pins as executable
fixtures or as a production canonical-JSON rule.
