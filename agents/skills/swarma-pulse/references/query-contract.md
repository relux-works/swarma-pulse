# Pulse query contract

`pulse me --local-unsigned` reads one explicit `product-card/v3` or `product-card/v4` snapshot and emits
a self-only `pulse-view/v1` JSON document. It preserves the existing v3.1 reader
semantics, including bounded YAML aliases and permitted legacy extensions. It
never converts a card, changes its coordinates, contacts a service or writes an
input. V4 uses strict parsing and checks the complete local declaration; external
attestations remain unverified. Existing v4 admission tools and the signed-caller
design remain separate contracts.

## Build and invoke

The initial target is a local Unix CLI, validated on macOS arm64. Use the module's
fixed supported Go toolchain, at least Go 1.25.12 (the Go 1.26 branch requires
1.26.5 or newer). No signature tool or cgo-backed account lookup is required.

Build with `-trimpath` to remove checkout filesystem paths from the executable.
Without it, builds of the same source at different paths can have different
artifact hashes. The `-p 2` flag caps build parallelism and is output-neutral;
path independence comes from `-trimpath`. Neither flag changes the source
revision or selects a release.

For an exact-commit build claim, select a signed commit in a clean standalone
clone with its own `.git` directory, rather than a linked worktree. Record
`git rev-parse HEAD` and `git status --porcelain` before building, then read
back the actual executable with `go version -m .temp/bin/pulse` and recheck the
source status. Require `-trimpath=true`, `vcs.revision` equal to the selected
head and `vcs.modified=false`, with clean source before and after. For this
supported Go/Git build route, uncommitted tracked-file changes or non-ignored
untracked files, including documentation, yield `vcs.modified=true` and cannot
support a clean exact-head claim; the stamp does not prove that Go source
changed. An ignored scratch file alone leaves an otherwise clean checkout at
`vcs.modified=false`. Some Go toolchains stamp an enclosing checkout when
building in a linked worktree. Missing or mismatching metadata leaves
attribution unverified; it alone does not prove different source bytes were
compiled. Preserve the artifact and qualify the build/toolchain route rather
than suppress or inject metadata.

For a path-independence check, build the same signed commit in two clean
standalone clones at different absolute paths using the command below. Fix and
record the exact Go version, GOOS/GOARCH and architecture tuning, CGO_ENABLED,
compiler and other build settings identically for both builds. Use
`GOTOOLCHAIN=local` to prevent automatic toolchain selection. Verify both
executables' metadata as above and compare their full SHA-256 hashes (for
example, `sha256sum /first/output/pulse /second/output/pulse` on Linux or
`shasum -a 256 /first/output/pulse /second/output/pulse` on macOS). Unequal hashes
leave that comparison unqualified, even if both revision stamps match.
Equal hashes qualify only the recorded source, toolchain, platform and settings;
`-trimpath` alone is not a reproducibility guarantee across different toolchains,
platforms or CGO configurations. Linux/amd64 with Go 1.26.8 is a supported check
lane; a result from that lane is Linux-only and does not qualify macOS behavior.
Unavailable lanes remain unrun. Matching metadata or hashes do not replace
signature verification, independent review or product checks.

```sh
mkdir -p .temp/bin
GOTOOLCHAIN=local go build -trimpath -mod=readonly -p 2 -o .temp/bin/pulse ./cmd/pulse
go version -m .temp/bin/pulse
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

Mode, card, revision and format are explicit. Unknown, duplicate, missing or unsupported flags
refuse. The explicit views below add only their documented selectors. Signed caller,
signature, policy, host-config, trust and clock selectors are unsupported. Omitted mode never falls back to unsigned behavior.
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

## V4 self-only snapshot

Use the same explicit selectors, input ownership/path checks and exact-byte pin
with `product-card/v4`. No conversion or additional input is implied. The reader
checks the complete v4 schema and local graph, including hidden members, binding
domains, references, reporting lines and duplicate chat identities. V4 aliases,
merges, tags, nulls and unknown fields refuse `card_invalid`; v3 keeps its existing
legacy reader semantics.

For chat members, `A2A_AGENT` must equal the exact localpart of the declared
`legacy-exact` MXID or of the `domain-star-v1` name:
`DOMAIN_PRODUCT_PROJECT_ROLECODE_STAR`. Role codes are `keeper`, `orch`, `dev`,
`tester`, `arch`, `reviewer`, `researcher` and `service`. Full MXIDs and unmatched
localparts refuse `caller_unknown`; multiple matches on distinct servers refuse
`caller_ambiguous`. Non-chat members never become caller matches. This is an
unverified local claim with the same single-user trust boundary as v3.

The response retains `pulse-view/v1`. Self's declared project uses its v4
`location` branch: local `path`, or remote `operator`, `execution_host`,
`project_locator` and optional `checkout_path`. These are coordinates only;
none is opened or fetched. Only self's project is returned. V4's explicit host
project is returned for a keeper; the v3 undeclared-project behavior is unchanged.

The v4 source adds `validation: local declarations only; source attestations and
star reservations unverified`. Referenced source bytes, legacy/logical-key
attestations and external reservation ownership are not verified by this
single-card read. Their declared shapes, member keys and selected server must
still agree locally. Use the separate admission validator with independently
supplied sources and reservations to verify those premises. A Pulse result
cannot substitute for that verdict or authorize registration, deployment or launch.

| Neighbor of a valid v4 self query | Refusal (empty stdout) |
| --- | --- |
| Mismatching exact byte pin | Exit 1, `RULE source_digest` |
| Relative/non-clean path, including traversal | Exit 1, `RULE input_path` |
| Symlink or unsafe input ancestry | Exit 1, `RULE input_trust` |
| Unknown/full-MXID/non-chat caller | Exit 1, `RULE caller_unknown` |
| Ambiguous localpart across servers | Exit 1, `RULE caller_ambiguous` |
| Malformed/unsupported schema version | Exit 1, `RULE unsupported_schema` |
| Invalid strict-v4 syntax or complete local graph | Exit 1, `RULE card_invalid` |
| V4 with environment or as primary discovery card | Exit 1, `RULE unsupported_schema` |
| V4 entry in discovery sources manifest | Exit 1, `RULE source_manifest_invalid` |

V4 supports the default self-only and explicit hierarchy views. Environment/discovery continue to
accept only their existing v3 product and v1 company schemas, including the
discovery sources manifest. All hierarchy lists stay empty in the default v4 result;
runtime, registration, installed profile and assignment observations stay unknown.

## One-card hierarchy view

Add `--view hierarchy` to the explicit unsigned invocation:

```sh
.temp/bin/pulse me --local-unsigned --view hierarchy \
  --card /absolute/path/product-card.yaml \
  --revision sha256:EXACT_LOWERCASE_64_HEX_DIGEST --format json
```

This view reads exactly one pinned `product-card/v3` or `product-card/v4` snapshot.
It follows no file references, external parents or remote locations. Company cards
refuse `unsupported_schema`; discovery source selectors refuse `usage` before input
access. Default, environment and discovery behavior is unchanged.

Output is `pulse-view/v3` with the same 17 root field names as v2:
`schema`, `mode`, `caller`, `known_limit`, `query`, `sources`, `query_observed_at`,
`self`, `peers`, `direct_reports`, `descendants`, `superiors`, `projects`, `products`,
`boundaries`, `hierarchy_status`, `observations`. `query` is `{view: hierarchy}`.
`sources` has one receipt with id, schema, revision, sha256, card_root, unknown
freshness and null observed_at; v4 also keeps its existing `validation` limitation.
Self has the baseline nine self fields plus responsibility `level`, without an
environment. Related summaries have the same seven public fields as discovery.
Others' profile, runtime, config, subscriptions and environment are not projected.

Relationships reuse discovery's responsibility levels and ordering: same-level
peers independently of parent, immediate direct reports, all reachable descendants
including direct reports, and nearest-first superiors. Non-chat names stay null;
parked and retired desired declarations remain present. V4 identities use the same
`legacy-exact` or `domain-star-v1` resolution as the self-only reader.

External parents remain `{source: CARD_ID, external: ALIAS}` boundaries; a v4
product/member external parent uses `PRODUCT/MEMBER` for `external`. Nothing
resolves that declaration. `hierarchy_status` is partial when a declared boundary
exists. `projects` contains only self's and returned members' projects, sorted by
qualified key. Locations retain `source`, `base: card` and the exact declared local
or remote fields, without checking their existence. `products` is empty because
one product card supplies no company inventory.

The complete card is validated before projection; the default reader's v3 alias
and extension semantics and strict v4 syntax/local-validation limits remain.
Invalid graphs refuse `card_invalid`; duplicate full identities refuse
`identity_conflict` when allowed by the source schema, while equal localparts on
distinct servers refuse `caller_ambiguous` for that caller. Existing context, pin,
safe-file and output-write refusals apply. The shared graph enforces its existing
64-hop budget (`input_limit` above it); product role rules still apply and are not
relaxed to admit arbitrary reporting chains. Query time and declarations establish
no registration, JOIN, live runtime, cryptographic caller or transport authority.
DM descriptors and paired room alias/ID declarations require their separate contracts.

## Declared environment view

Add `--view environment` to the same invocation to request `pulse-view/v2`:

```sh
.temp/bin/pulse me --local-unsigned --view environment \
  --card /absolute/path/product-card.yaml \
  --revision sha256:EXACT_LOWERCASE_64_HEX_DIGEST --format json
```

This view supports one `product-card/v3` or `company-card/v1`; it never opens
referenced cards and refuses discovery source selectors. The existing invocation without `--view`
keeps its v1 schema and ignores the new optional member extension.

The v2 root contains exactly `schema`, `mode`, `query`, `caller`, `known_limit`,
`sources`, `self`, `peers`, `direct_reports`, `descendants`, `superiors`, `projects`,
`products`, `boundaries`, `hierarchy_status`, `observations` and
`query_observed_at`. `query` is exactly `{view: environment}`. The singular v1
`source` becomes a one-item `sources` array with the same receipt fields plus
the card `id`. The v1 root `product` and `member` fields are absent.

`self` retains the baseline fields and adds `level` (`company` for company-card
members; otherwise `product` for a keeper, `project` for an orchestrator and
`member` for other roles) and `environment`. Company self has company scope and
no project.
`projects` still contains only the caller's declared project. All relationship,
product and boundary arrays are empty; hierarchy and observations remain unknown
under the same self-only rule.

`self.environment` has exactly `schema: pulse-environment/v1`, `state`, `profile`,
`runtime`, `a2a`, `working_root`, `network` and `missing`. These are declarations,
not observations, installed configuration or permission to launch:

- `profile` contains `key`, the existing `requirement` and `selected` pins, and
  `source`. A local source is `{source: CARD_ID, base: profiles_root,
  profiles_root: RELATIVE, path: RELATIVE}`; interpret `profiles_root` relative
  to that card's actual parent, then `path` relative to it. A registry source is
  `{source: CARD_ID, base: registry, repo: REPOSITORY, directory: RELATIVE}`.
- `runtime` contains `kind`, `model` and `effort`; absent effort is null.
- `a2a` contains `agent`, full `resolved_name`, `binding`, `operator`, `host`,
  named `connection`, `server_name` and `config_profile` (null if undeclared).
  It does not forward the HTTPS endpoint. A consumer resolves connection and
  config selectors through its own admitted configuration.
- `working_root` is `{source: CARD_ID, base: card, path: RELATIVE,
  origin: ORIGIN}` or null. Precedence is `pulse.working_root`, then
  `member.clone`, then the caller's `project.path`. The origin records that
  choice. Its base is the card's actual parent, never the query cwd.
- `network` is null when undeclared, or exactly `{mode: direct}` or
  `{mode: proxy, ref: SELECTOR}`. There is no implicit direct fallback.

To supply the optional launch declarations, add this closed member extension:

```yaml
pulse:
  schema: pulse-member-environment/v1
  a2a_config_profile: demo-config
  working_root: ../beacon
  network:
    mode: proxy
    ref: proxy-main
```

Only `schema` is required in the extension. Config and proxy selectors match
`[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}`; they are symbolic names. The two network
branches accept exactly the fields shown. Unknown keys/version, wrong types,
malformed selectors or relative paths refuse `RULE environment_invalid: $`
without echoing the supplied value. All supplied member extensions are validated
before projection; only the caller's environment is returned.

Declared relative locators must be nonempty and contain no control characters,
leading slash, backslash, tilde, URI scheme or platform drive prefix. Sibling
`..` components are allowed. Pulse checks syntax only: it never opens these
paths or checks their existence, symlinks or containment. The source file itself
retains the strict input-read boundary described above. The eventual launch
consumer separately admits its working root and profile.

Missing config profile, network or working root produces success with
`state: incomplete`, explicit nulls and sorted `missing` paths
(`a2a.config_profile`, `network`, `working_root`, as applicable). Otherwise the
state is `complete`. Baseline validation already requires profile selection,
runtime kind/model and chat binding/connection. Optional effort is not missing.
Consumers must refuse an incomplete launch description rather than invent values;
a complete description still grants no execution permission. No actual profile,
credential store, process, proxy configuration or session file is consulted.

## Company inventory and caller discovery

Company cards use the new discriminant `company-card/v1`. Their closed root
contains exactly `schema`, `company`, `external_members`, `connections`,
`bindings`, `profiles_root`, `profiles`, `members` and `products`. `company`
contains exactly `id`, `description`, `privacy`, `keeper` and `default_binding`.
Connection, binding, profile and member fields reuse v3 predicates and legacy
compatibility. A company is never presented as a v3 product to the old reader.

The designated `company.keeper` references a local member with role `keeper`.
Every company member requires `parent`. Only that designated keeper may declare
`parent: null`, which ends traversal at the declared company root. Its parent
may instead be a local or declared external string. Other company members,
including another member with role `keeper`, require a string parent. Missing
parent is invalid, and company members cannot have a project reference.
An empty products list is valid. Member `clone` and the environment extension
may declare a working root without inventing a company host project.

Each product inventory entry has exactly these fields:

```yaml
id: aurora
description: Example product
host: {kind: local, path: ../aurora-host}
card: {path: ../aurora-host/product-card/product-card.yaml}
keeper: aurora/product_keeper
superior: example-company/company_keeper
parent_alias: external_company_keeper
```

Ids are unique across company and products. `keeper` is qualified by the entry's
product id; `superior` names a materialized company member. A supplied product
must designate that keeper and declare its parent as the exact external
`parent_alias`. Discovery then resolves that one explicit edge to the superior;
it never edits a card or infers an account from the alias. Disagreement refuses
`relation_conflict`.

Host and card locators use the actual company card's parent as their base.
For a local entry, normalized `card.path` must be a strict component descendant
of normalized `host.path`, directly or nested, with leaf `product-card.yaml`.
A sibling sharing a string prefix, escape or different leaf refuses `input_path`.
The closed `card` object accepts optional `revision: sha256:LOWERCASE64HEX`.
When supplied, that pin must equal the selected product's query pin; disagreement
refuses `source_digest`. Omitting the inventory pin never omits query pinning.
A supplied local product's actual path must equal the company reference or the
query refuses `source_reference_mismatch`. Declaring a locator does not open it.

A remote host has exactly `{kind: remote, operator: ID, execution_host: ID,
locator: PUBLIC_LOCATOR}`. Its nonempty locator has at most 512 bytes and no
control characters. The card path retains its literal company-relative inventory
meaning and receives relative syntax checks only. No remote coordinate mapping,
local read, fetch or execution is inferred. Unsupplied remote entries report
`remote_unavailable`; supplying one as a local source refuses
`remote_source_unsupported`.

Invoke discovery only with all explicit source selectors:

```sh
.temp/bin/pulse me --local-unsigned --view discovery \
  --card /admitted/root/product/product-card.yaml \
  --revision sha256:EXACT_CARD_DIGEST \
  --sources /admitted/root/sources.json \
  --sources-revision sha256:EXACT_MANIFEST_DIGEST \
  --root /admitted/root --format json
```

The separately pinned sources manifest is UTF-8 JSON, not YAML:

```json
{
  "schema": "pulse-sources/v1",
  "cards": [
    {
      "id": "example-company",
      "schema": "company-card/v1",
      "path": "../company/company-card.yaml",
      "revision": "sha256:LOWERCASE64HEX"
    }
  ]
}
```

The root has exactly `schema` and `cards`; entries have exactly the four shown
fields and select additional cards only. Every entry path is relative to the
primary card's parent, regardless of the manifest's own directory. The primary
may be a company or product. An empty `cards` array is valid and still requires
all selectors. Duplicate keys, ids, normalized paths, repeated primary,
unsupported schema and mismatched actual id/schema refuse the whole query.
At most one company may be supplied. With a company, every supplied product must
be in its inventory or `source_unreferenced` refuses. Product-only discovery
needs no invented company root.

`--root` must be an absolute clean admitted read root. Primary, manifest and
additional source files must be strictly inside it after normalization. Sibling
`..` references are allowed within that root; an escaping path refuses
`input_path`. All actually opened files retain the ownership, ancestor mode,
nonsymlink, UID and bounded-read checks above. A symlink inside the root still
refuses. A supplied missing path refuses instead of becoming a partial result;
a deliberately unsupplied reference produces a boundary without opening it.
There is no automatic source search, Git lookup or fallback to a different view.

Limits are 1 MiB per card or manifest, 32 additional cards, 16 MiB total supplied
card bytes, 10,000 materialized members and 64 reporting hops, plus the existing
per-card parser/member/project limits. Discovery breaches refuse `input_limit`.
The complete supplied graphs and all environment extensions are checked before
projection, including non-self records. Duplicate resolved full MXIDs refuse
`identity_conflict`; equal localparts on distinct domains instead remain distinct
accounts and refuse `caller_ambiguous` when selecting the caller. Reporting
cycles and inconsistent or dangling materialized edges refuse the whole query.

Discovery returns the same closed 17-field v2 root as environment. `query` is
`{view: discovery, sources: {revision, sha256, root}}`. Root-level `sources`
contains the primary first, then additional sources sorted by id, each with its
own actual card parent, exact byte pin and unknown freshness. Project, clone,
profile and company locators retain their owning card's base.

Related summaries contain exactly `key`, `role`, `level`, `desired_state`,
`resolved_name`, `project` and `parent`. Keys are `card-id/member-id`; non-chat
members have null resolved names. Parent is a qualified key, null at the declared
root, or `{source: CARD_ID, external: ALIAS}` for an unresolved external edge.
Summaries exclude other members' environment, profile pins/paths, private routes,
subscriptions and config/proxy selectors. Only self receives its environment.

- `peers` contains every other materialized member at self's responsibility level,
  independently of graph depth or parent. Levels are company for company members,
  product for product keepers, project for product orchestrators, otherwise member.
  A keeper managing a project remains product-level.
- `direct_reports` contains immediate reports; `descendants` contains all reachable
  reports, including direct ones, once each.
- `superiors` is nearest-first and stops at a root or unresolved ancestor.
- `projects` contains only self's and returned summaries' projects, with qualified
  keys, repository, desired state and source-qualified local locations.
- `products` contains all company inventory for company self; for a product caller
  it contains self's product and products of returned summaries. Each has exactly
  id, description, source-qualified host/card locators, keeper, superior and
  `resolution: resolved|not_supplied|remote_unavailable`.

Except for nearest-first superiors, lists sort by qualified key or product id.
Parked and retired desired declarations remain present. Each unresolved parent
edge and each unsupplied or remote inventory entry in the entire supplied universe
produces a boundary with exactly `from`, `relation`, `target` and `reason`.
Reasons are `external_unresolved`, `not_supplied` and `remote_unavailable`.
Boundaries sort by raw source-qualified from, relation, target and reason
(parent object targets compare source then external). `hierarchy_status` is
`partial` with any boundary, otherwise `complete` relative to supplied declarations.
A digest, resolved edge or complete hierarchy proves no authentication, current
appointment, visibility entitlement, registered account or launch authority.
Observations stay unknown and query time remains separate from source freshness.

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

Each view requires exactly one matching localpart in its selected card or
explicitly supplied source universe. If two declared bindings match that context on different
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
Card migration, authenticated visibility, cryptographic callers and live observations
require their own future accepted contracts. Discovery checks also pair reciprocal
company/product/member projections with partial inventory, explicit remote and
missing-source refusals, source agreement, complete-graph validation, budget
boundaries, summary privacy and stable ordering. Fixture evidence does not prove
actual company/product adoption.
