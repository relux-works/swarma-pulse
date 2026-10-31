# Pulse local member query

`pulse me --local-unsigned` reads one explicit `product-card/v3` snapshot and emits
a self-only `pulse-view/v1` JSON document. It preserves the existing v3.1 reader
semantics, including bounded YAML aliases and permitted legacy extensions. It
never converts a card, changes its coordinates, contacts a service or writes an
input. Existing v4 tools and the signed-caller design remain separate contracts.

## Build and invoke

The initial target is a local Unix CLI, validated on macOS arm64. Use the module's
fixed supported Go toolchain, at least Go 1.25.12. No signature tool or cgo-backed
account lookup is required.

```sh
mkdir -p .temp/bin
go build -o .temp/bin/pulse ./cmd/pulse
# Select an existing owner-held card and its independently checked byte digest.
.temp/bin/pulse me --local-unsigned \
  --card /absolute/path/product-card.yaml \
  --revision sha256:EXACT_LOWERCASE_64_HEX_DIGEST \
  --format json
```

The invoking session supplies its existing public `A2A_AGENT` localpart. Pulse
compares that string exactly with each chat member's declared/default
`binding.identity_prefix + member.id`. Exactly one match is required in the one
card. It uses the declared connection's server name for the full Matrix name.
There is no punctuation normalization, account-store lookup or `--member` override.
Missing, empty, unknown and ambiguous context refuse; non-chat members cannot be
selected through this context.

All four selectors are explicit. Unknown, duplicate, missing or unsupported flags
refuse. Signed caller, signature, policy, extra-source, host-config, trust and clock
selectors are unsupported. Omitted mode never falls back to unsigned behavior.
`--revision` verifies the selected file's exact content SHA-256; it is not a Git
revision lookup or proof that this snapshot is the current canonical card.

Success exits 0 with one complete JSON document and no diagnostic. A query refusal
exits 1, a usage refusal exits 2; both produce a safe rule diagnostic on stderr and
empty stdout. The full result is buffered before output. A final stdout transport
failure can interrupt delivery and does not modify inputs.

## Local trust and file boundary

Every result explicitly carries mode `local-unsigned`,
`caller.cryptographically_verified: false`, `caller.trust: single-user local trust`
and the exact limitation:

> caller not cryptographically verified; single-user local trust

`A2A_AGENT` is an unverified claim. Another process with the same OS identity can
select another valid member by changing its environment; this mode cannot detect
that impersonation. Self-only projection is not filesystem isolation or authority
to act on the returned facts. Raw card access remains governed by OS permissions.

Real and effective invoking UIDs must match. The card must be one absolute, clean,
owner-held regular file, at most 1 MiB. Its ancestry must consist of nonsymlink
directories owned by root or that UID, without group/world write permission. The
file must belong to the invoking UID and satisfy the same write restriction.
Symlinks, FIFOs, devices, directories, oversized inputs and unsafe ancestry refuse.
The opened file is checked again and the read is bounded; root and the same UID
remain trusted and can change owned paths. No protection against those writers is
claimed. There is no implicit home, profile, checkout, credential, Keychain or
additional card lookup.

The entire v3 graph is validated before projection. Malformed documents, invalid
relationships and unsupported schemas refuse the complete query. The existing
reader's limits apply; legacy extensions are not subject to v4's closed-field
policy. No local layout traversal is performed.

## Returned facts

The output includes the product/member qualified key, self role, derived
responsibility scope, desired state, resolved chat name, declared binding
operator/host/connection and own profile requirement/selection. These are desired
facts, not observations or execution authority.

A member without a declared project has `self.project: null`,
`self.project_state: undeclared` and `projects: []`. In particular, a v3 keeper's
host project is never invented. A declared project contributes only its qualified
key, repository, desired state and local relative path. `source.card_root` is the
actual input file's parent directory; use that base when interpreting the path.

`peers`, `direct_reports`, `descendants` and `superiors` are always empty arrays.
The constant `hierarchy_status` is `self-only; hierarchy not disclosed`, regardless
of hidden record counts or edges. Parent/result/escalation routes, subscriptions,
profile paths, other members and project managers are withheld. Runtime,
registration, installed profile and assignment observations are all `unknown`.

`source.schema`, `revision` and `sha256` identify the exact supplied bytes.
`source.freshness` is `unknown` and `source.observed_at` is null: file mtime is not
observation provenance. `query_observed_at` is the actual query's UTC clock, kept
separate. Changing hidden source bytes necessarily changes the mandatory source
pin; no claim of hiding that fingerprint is made.

## Matrix identity in shared displays

Owner decision, 2026-10-31: shared identity displays retain the full Matrix ID,
including its homeserver. A friendly short label may accompany that ID.

| Field | Meaning | Use |
| --- | --- | --- |
| `self.resolved_name` | Full source-declared `@localpart:homeserver` | Retain the complete ID in shared display and when passing an account identifier to a consumer. |
| `caller.agent` | Unverified local `A2A_AGENT` context | Keep its existing meaning; it is not a qualified account address or authentication proof. |
| `member` and `self.key` | Logical `product/member` key | Identify the logical member independently of its Matrix account or friendly label. |

For example, `@station_beacon_orch:alpha.example` and
`@station_beacon_orch:beta.example` identify distinct Matrix accounts despite
their equal localparts. Consumers must retain both homeservers in display,
addressing and account identity comparisons. Repeating the same full ID preserves
account equality; message-event deduplication is a separate contract. Use the
selected connection's declared `server_name`, not its HTTPS endpoint hostname or
the execution host, when interpreting the returned name.

The current local query still requires exactly one matching localpart in its
single pinned card. If two declared bindings match that context on different
homeservers, it refuses with `RULE caller_ambiguous` and empty stdout. It does not
choose a domain implicitly or introduce a qualified caller selector.

The full name is a card declaration; it does not establish account registration,
authenticated authorship or current runtime state. Pulse emits this data but has
no shared display renderer, messaging operation or transport deduplication.
Its fixture checks do not establish that a downstream consumer implements or
has adopted this contract.

## Verification and rollback

```sh
go test -count=1 ./...
go vet ./...
git diff --check
```

Paired tests cover declared and undeclared self projects; exact, unknown and
ambiguous context; explicit mode/selector refusals; schema, graph and digest
failures; bounded aliases; safe file types/ownership/modes; empty refusal stdout
and unchanged input bytes. Hidden-data comparisons allow only the mandatory
source fingerprint to change. These checks cannot establish cryptographic caller
authenticity or actual host adoption.

Federation fixtures pair equal localparts on different homeservers with a repeated
identical full ID. A unique local-context positive precedes a valid two-domain
ambiguous-context refusal, checking empty stdout and preservation of the fixture
file and directory. Consumer display, addressing and event deduplication need
their own implementation checks.

Before actual use, the input custodian selects the existing card, verifies its
exact content pin and runs the intended existing session context. Qualify a valid
invocation and nearby digest/context refusal with the same actual input. Fixture
success alone does not prove that operation occurred.

The query installs no daemon and creates no persistent state to roll back. Stop
invoking it or use the previous reviewed binary while preserving all input files.
Card migration, broader visibility, cryptographic callers and live observations
require their own future accepted contracts.
