# Full identity names and compact Matrix names — naming amendment

Status: specification amendment for review. The naming choice below records the owner decision dated
2026-10-30; field/version/resolver proposals still require acceptance before implementation. No account
is registered and no resolver, card schema or migration command is implemented by this amendment.

## 1. Dated decision and scope

Owner decision, 2026-10-30: «нарвится но эт полный резолв имени а в матриксе swarma-pulse.orch-betelgeuse».
The selected mapping is:

```text
Full name: orch-betelgeuse.swarma-pulse.swarm.gr.relux.works
Matrix ID: @swarma-pulse.orch-betelgeuse:gr.relux.works
```

For new names this supersedes the 2026-10-29 domain-as-localpart-prefix form in
[specification v1, section 5](product-card-identity-pulse-v1.md#5-stable-identity-scheme).
Product and zone remain in the full name; the Matrix localpart contains project, role and star only.
`swarm` is the product, `swarma-pulse` the project. This is not a product/project rename, reservation,
account-existence claim or authorization to create an account.

The full name uses DNS syntax; this specification creates no DNS records and claims no live network
endpoint. Identity lookup uses admitted registry data, not ambient DNS as an authority.

The existing `domain-star-v1` schema, validator, converter, source pins, reservations and fixture verdicts
remain unchanged as compatibility contracts. Existing Matrix accounts cannot be renamed. A live
cutover still creates a separately admitted account and follows the accepted preservation/custody plan.
No existing account is silently remapped from its old MXID.

Identity lifetime/seat ownership remains an open owner input. This amendment defines the mapping of
an admitted naming tuple; it does not decide whether an appointment, role or project transfer creates
a new seat, renames an existing seat, or retains a historical name. Such a transfer needs an explicit
accepted disposition before changing a live account. Changing model or restarting a runtime does not
by itself allocate a name or a new member.

## 2. Canonical inputs and vocabulary

The naming tuple is `(zone, product, project, role, star)`, plus the stable qualified logical member key
that holds its reservation. `zone` is the selected connection's verified Matrix server name, not its
HTTPS endpoint hostname. The target connection uses `gr.relux.works`. A different zone requires its
own admitted connection/custody; a string suffix alone does not provision a Matrix server.

All inputs are already canonical lowercase ASCII. Product/project/star match
`[a-z][a-z0-9]*(?:-[a-z0-9]+)*` and contain 1–40 ASCII bytes. Zone is a DNS name with at least two
labels, each 1–63 bytes, no empty label, leading/trailing hyphen or trailing dot. No port, Unicode,
IDNA conversion, uppercase folding, whitespace trimming, percent decoding, abbreviation, truncation,
hash suffix or random fallback is allowed. Reject a noncanonical input as supplied.

Retain the existing closed primary-role mapping:

| Role | Code |
| --- | --- |
| keeper | keeper |
| orchestrator | orch |
| developer | dev |
| tester | tester |
| architect | arch |
| reviewer | reviewer |
| researcher | researcher |
| separately admitted service | service |

Unknown roles refuse. These codes contain no hyphen; split `role-star` at its first hyphen and require
an exact known code. All remaining bytes form the complete star slug, including any internal hyphens.
A known code is naming vocabulary, not admission to chat, launch or a protected action. A combined
manager retains its existing primary-role appointment and code.

Stars remain readable reviewed astronomical names. The immutable logical member key holds its star
reservation across assignments. Preserve the earlier reservation scope `(zone, product, primary-role)`
across projects until a separate accepted decision changes it. The new compact localpart does not
silently weaken those reservations. Retired reservations are retained, never automatically reused.

## 3. Forward mapping

Given the canonical tuple and the admitted registry snapshot:

```text
role-star   = role-code + "-" + star
full-name   = role-star + "." + project + "." + product + "." + zone
localpart   = project + "." + role-star
matrix-id   = "@" + localpart + ":" + zone
```

Validate every component and length before producing a candidate. Verify the project/zone belongs to
the tuple's product, the role/star reservation belongs to the exact logical member, and the resulting
MXID has no other current or retired holder. The registry is independently admitted and pinned; its
own claimed product/owner does not establish authority. A pure formatting preview is not a reservation
or proof of registration. Resolver output intended for admission requires these checks, not strings alone.

For a full-name input, use the explicitly admitted zone suffix, not a guess at a public suffix. Remove
exactly `.` plus that zone, then require exactly three remaining labels: `role-star`, `project`, `product`.
Parse/validate them as above, check registry ownership and derive the exact same MXID. An unadmitted
zone, extra/missing label or product mismatch refuses.

## 4. Reverse mapping and its necessary registry

MXID bytes alone cannot recover product: the product was removed from the localpart. No algorithm
may invent that lost field or select the first card it finds. The reverse operation is a **checked lookup**
against the same admitted zone/project registry revision used for forward resolution.

1. Parse exactly `@LOCALPART:ZONE`; require the selected connection's verified zone.
2. Split the localpart at its single dot into `project` and `role-star`. Validate both; split the first
   hyphen in `role-star` into a known role code and complete star slug.
3. Resolve `(zone, project)` to exactly one product in the pinned registry. Require an admitted record
   for this role/star/logical member. Missing, conflicting, stale or unavailable registry input refuses
   canonical reverse resolution; it never triggers remote discovery or an account operation.
4. Reconstruct `role-star.project.product.zone` and run the forward checks. Require byte-for-byte
   equality with the original MXID. Return tuple, logical member, registry revision/digest and full name.

The inverse is exact within that accepted snapshot. It is not a timeless bijection across arbitrary
registry changes. Historical reverse resolution requires the explicit historical snapshot and cannot
claim current validity. A syntactically parseable unknown MXID can be reported as syntax only; it is
not a resolved identity and must not acquire a guessed product/member.

Project ownership changes or a proposed alias must be an explicit accepted transition with retained
history. Do not allow a mutable project-owner lookup to reinterpret a previously pinned identity.
A full-name or MXID lookup also respects the caller's visibility policy; a restricted identity is not
made discoverable merely because its name is well formed.

## 5. Zone-wide project uniqueness and collision handling

A project slug is unique within a zone **across products**. Retain the reserved/tombstoned owner so
an old localpart cannot later identify a different product. A second product cannot adopt that project
slug even if its immediate role/star combination would happen to produce a different MXID.

There are three separate checks:

| Reservation | Holder and purpose |
| --- | --- |
| `(zone, project)` | One admitted product/project identity; recovers product from compact MXID |
| `(zone, product, primary-role, star)` | One immutable logical member; preserves the existing star allocation guarantee |
| Complete MXID | One immutable logical member, including retained retired bindings |

An admitted single writer/atomic compare-and-set reservation route must cover the actual zone scope.
Local file checks on two disconnected machines are not an atomic allocation guarantee. Shared-zone
operators must agree that route and authority before new reservations; a card's desired tuple is not
an allocation. Pulse's read-only lookup does not create or change any reservation.

On conflict, refuse the candidate before file/Keychain/account/room/hook/launch effects and preserve
all existing holders. Same-holder retries may return the same reservation only when every tuple,
source pin and admission field is unchanged. Different holders or conflicting product owners refuse.
Never fix a collision by suffixing, shortening, changing star/role/product, selecting a different zone,
or overwriting the winner automatically.

Options for a real conflict go through the product custodian to the owner:

| Option | Trade-off |
| --- | --- |
| Choose a distinct canonical project id for the new/conflicting admission | Recommended: keeps accepted existing names and one zone; requires an explicit approved id/mapping change |
| Explicitly rename one already existing conflicting project | Broader reference/history review; existing account migration and repository renaming remain separate decisions |
| Use another owner-approved verified Matrix zone | Changes connection/federation/custody and may require new infrastructure; not an automatic naming escape |

If two legacy products already share a project slug in one zone, preserve their legacy mappings and
refuse compact-scheme admission until the conflict has an accepted disposition. Do not retroactively
claim that the old registry already met this new global invariant.

## 6. Exact length rules

All selected characters are ASCII, so character and UTF-8 byte counts agree. Independently require:

- Matrix localpart: `len(project) + 1 + len(role-code) + 1 + len(star) <= 64` bytes; minimum components
  remain nonempty. The ceiling applies to the localpart, not the full MXID or DNS name.
- Every DNS label: 1–63 bytes, including the combined `role-star` label.
- Full DNS name: at most 253 ASCII bytes without a trailing root dot, including separators.

Passing one bound does not waive another. The selected first example's localpart is
`swarma-pulse.orch-betelgeuse`, exactly 28 bytes. Do not apply the earlier 49-byte prefix calculation
to this new form. An overlong candidate refuses; no account or filesystem normalization is attempted.

## 7. Versioning and a2a integration boundary

Propose the new identity discriminator `project-star-v1` in an explicitly accepted successor card
format. It is distinct from `domain-star-v1`; the complete successor tables and registry/caller input
contracts must be accepted before implementation. The shipped v4 tools must reject the new discriminator
and unknown registry fields instead of silently changing their old naming equation.

A future resolver needs explicit selected connection, tuple, logical member, pinned zone/project and
star/MXID reservation inputs. It returns canonical full name and exact MXID together, with source pins.
The onboarding descriptor must compare tuple/full name/MXID/server/connection/registry revision and
intended audience. A low-level registration command accepting the localpart's characters proves only
lexical eligibility, not that it enforces these semantics or owns a reservation.

Source integration review must ground what the installed named-connection route actually checks.
Do not invent a new `ensure` mode, credential accessor, collision allocator or migration API here.
Missing semantic checks may be supplied by the admitted reviewed handoff if its contract is sufficient;
only a demonstrated blocker requires a source change before the corresponding live operation.

No new scheme is admitted merely by merging this specification. The selected first pilot waits for
the admitted phase-2 secure-store installation and its existing key/custody sitting. Seat-ownership input,
actual reservations and the filled source/runtime handoff remain distinct facts. No account, Keychain,
keyvault, key creation/export/signing, room change or session launch is part of this document.

## 8. Contract examples and negative expectations

The [review fixtures](../fixtures/identity-names-v2.json) are specification examples, not inputs accepted by
the shipped validator or an executable new resolver. Their registries are explicitly synthetic premises.

| Claim | Valid control | Required refusal or limit |
| --- | --- | --- |
| Selected mapping | The exact owner example maps to the 28-byte localpart and back using its pinned project owner | Domain-prefix localpart cannot be silently treated as the new scheme |
| Reverse lookup | Unique `(zone, project)` owner and member reservation reproduce exact full name | Missing/conflicting product owner refuses; no product guessed from MXID |
| Project uniqueness | Same reserved project/product/holder retry is unchanged | Same project in the zone owned by another product refuses even if star differs |
| Existing star guarantee | Member keeps its admitted product/role star reservation across assignments | Another member's reservation or a tombstoned MXID cannot be reused |
| Lexical mapping | Known role with complete hyphenated star maps exactly | Unknown role, extra dot, uppercase, Unicode, trailing dot or lossy normalization refuses |
| Localpart boundary | 38-byte synthetic project + `researcher` + `alpha-centauri` gives 64 bytes | 39-byte project with that same role/star gives 65 bytes and refuses before effects |
| DNS bounds | Every component and complete full name fits DNS limits | Overlong zone label/full name refuses even when localpart fits |
| Snapshot authority | Current admitted registry matches its pin and caller visibility | Stale, altered, unauthorized or restricted lookup cannot return a canonical member |
| Compatibility | Existing old-scheme/v4 fixtures keep their exact accepted results | New discriminator to v4 refuses; no rewrite or old-account rename |
| Read-only resolution | Supplied admitted inputs yield bounded name output | No scan, fetch, reserve, registration, private-key access or launch |

Later executable tests must exercise the actual production resolver and no-effect boundary, with these
positive and forbidden controls. Passing a string-format example alone cannot prove namespace
allocation, registry endorsement, installed-host qualification or permission to create an account.
