---
name: swarma-pulse
description: Use the shipped Pulse CLI to read a local unsigned self-only member view, interpret its identity and source fields, or select the existing product-card validator, converter and change checker. Use for these module interfaces; account registration, runtime launch and proposed signed or plugin interfaces are separate operations.
---

# Use Pulse

Read [the query contract](references/query-contract.md) for the complete input, trust
and output boundary. Use the shipped `pulse me --local-unsigned` command for a
self-only view of one explicitly selected `product-card/v3` snapshot. Its output
schema is `pulse-view/v1`. The v4 tools are separate; selecting a v4 card does not
make it a supported Pulse query input.

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
  exact `binding.identity_prefix + member.id` for chat members, through the
  declared or default binding, with exactly one match. A full MXID, normalized
  punctuation or an arbitrary member override is not a caller selector.
- Pass **all** selectors explicitly: `me --local-unsigned --card ABS
  --revision sha256:LOWERCASE64HEX --format json`. There is no query config file,
  implicit card discovery, extra-card merge, `--member`, `--config`, signature,
  policy, trust or clock override. Missing mode refuses instead of falling back.

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
go build -o .temp/bin/pulse ./cmd/pulse
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

## Interpret the result without expanding authority

| Value | Interpretation |
| --- | --- |
| `self.resolved_name` | Full source-declared `@localpart:homeserver`; preserve the complete ID in shared display and account addressing/comparison. A friendly label may accompany it. |
| `caller.agent` | Unverified local context; not a qualified account address or authenticated sender. |
| `member`, `self.key` | Logical `product/member`, separate from the Matrix account. |
| `self.execution_binding`, `self.profile`, `projects` | Declared desired facts, not current process, installed-profile or scheduling evidence. |
| `source.schema`, `revision`, `sha256`, `card_root` | Supplied schema, exact byte pin and real input-parent coordinates. Freshness is `unknown`; `observed_at` is null. |
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

`peers`, `direct_reports`, `descendants` and `superiors` are always empty.
`hierarchy_status` is `self-only; hierarchy not disclosed`; empty lists do not
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
| Unsupported schema or invalid complete v3 graph | Exit 1 with `RULE unsupported_schema` or `RULE card_invalid`, no conversion or partial view. |
| Unsafe, missing or unreadable input | Exit 1 with the applicable input rule; no substitute source or input writes. |

Diagnostics use `RULE name: $` plus newline. The entire v3 graph is validated,
including relationships, before projection; accepted bounded aliases and legacy
extensions retain the v3 reader semantics. Do not impose v4's closed-field rules
on v3 or accept only the self record from an invalid graph. A final stdout
transport failure returns `RULE output_write`/exit 1 and can interrupt delivery;
that is distinct from an intended pre-output refusal with empty stdout.
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

Signed callers, broader hierarchy, v5 composition, compact naming/migration and
remote/plugin interfaces in the design documents remain proposals. Do not invoke
unimplemented `pulse verify` or treat illustrative future pins as executable
fixtures or as a production canonical-JSON rule.
