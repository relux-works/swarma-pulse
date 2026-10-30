# Product hosts, identity keys and personal Pulse views — proposal

Status: specification amendment for review, dated 2026-10-30. No command, schema implementation,
key enrollment or runtime deployment is supplied by this document. The accepted
[identity specification](product-card-identity-pulse-v1.md) and [v4 format](../format-v4.md)
remain the implemented contracts. The [naming amendment](identity-names-v2.md) records the full-name/compact-Matrix choice;
identity and obligations belong to the logical seat. Successor-schema and actual namespace admission
remain open. Examples here use logical member identifiers only.

## 1. Required result and boundaries

A product has one declared structure host. The host contains its product card and the public-key
registry for its participants. It can be a dedicated project repository inside a multi-project product,
or the existing project repository when the product has just one project. Every participant uses its
personal Pulse projection to learn its place: self, peers, superiors, reports and relevant projects.
Routine member discovery does not parse the raw card.

A participant's private key belongs to the operator where that participant runs. It never travels with
the product host or another operator's checkout. A participant remains one logical member across
assignments; moving machines is an explicit placement change, not a second identity or permission to
run a duplicate. Distinguish logical identity/obligations, station placement (OS principal, home, clone
and profile), and the actual runtime executor/lease epoch. Replacing an executor does not transfer
the member's obligations. These are conceptual layers, not permission to launch or a replacement
for the existing task-station reference. Operators may each run their own participants while reading
one agreed product host and receiving views spanning authorized members of both operators.

Public-key material is public *material*, not a requirement to publish the repository publicly. Preserve
the product's privacy and repository access controls. Product membership, runtime placement, a public
key binding, proof of private-key possession and authority to act are separate facts. A verified key
never authorizes launch, room access, a deployment or a repository write.

Matrix bearer credentials and the configured human Git author/signing keys remain separate from
member identity keys. This proposal stores no credential, private key, vault blob or raw session data.

## 2. Host layouts and execution locations

| Layout | Canonical host contents | Other project checkouts |
| --- | --- | --- |
| Dedicated host project | `product-card.yaml`, `identity-keys/index.json`, operator registry documents, public key files, and the product's declared management entry point | Declared by the card with explicit local/remote locations |
| Single-project product | The same files inside the existing project repository; that project is `host_project` | No artificial host repository is required |

Recommended paths, common to both layouts:

```text
product-card.yaml
identity-keys/
  index.json
  operators/<operator-id>.json
  public/<operator-id>/<public-key-id>.spki.der
```

The management entry point identifies the product root and its declared manager; it is not a script
that launches a session. A product root is resolved through the admitted context, never found by
scanning neighboring repositories. The current `host_project` and local `location.path: .` anchor the
canonical snapshot checkout. They do not require every operator to have every project locally.

A remote product participant receives an admitted host snapshot in a local context, or an authorized
sanitized projection. Relative paths use that snapshot's declared root, not the worker's current working
directory. A repository locator is different from an execution-host location. Neither a remote project
nor a remote public-key reference triggers checkout, network discovery or launch.

The card's desired binding names operator and execution host. Actual running placement is a separately
observed, dated fact. A desired binding alone cannot prove that a member is running, absent elsewhere,
or fenced from duplicate execution. An accepted launcher/custody handoff must enforce any eventual
one-active-placement rule. Pulse v1 is read-only and does not raise participants.

## 3. Personal views and deployment recommendation

Recommend one explicit Pulse **context per product on each participating operator's machine**, using
the same read-only executable where available. A context is an admitted product root, trusted snapshot
pins, local caller authority and visibility policy. It does not require a daemon or a separate installation.

| Choice | Benefit | Cost or limitation |
| --- | --- | --- |
| Product context per operator, local read-only CLI | Clear product and operator boundary; works with supplied offline snapshots; matches existing caller admission | Each operator provisions its own admitted context and refreshes pins |
| One daemon per product | May later serve several local callers from one cache | Adds a service lifecycle, caller authentication and cache/freshness contract with no demonstrated v1 need |
| One shared multi-product service | One service/cache administration point | Needs explicit tenant separation, cross-product authorization, remote authentication and outage handling |

Use the first option for v1. A later service must preserve explicit product selection and caller binding;
a product id or a CLI flag cannot change the caller's authority. This recommendation does not adopt a
launcher or a remote service protocol.

Preserve the accepted `pulse-caller/v1` signed handoff, host-selected trusted signers, pinned sources,
visibility allowlist and self-only default. Identity registry data is not a replacement for that trust
anchor. Any extension that pins new registry inputs must have an explicit new caller-envelope version;
the closed v1 decoder must refuse those additional fields. Do not silently change the SSHSIG caller
contract to P-256 member signatures.

The personal projection shows authorized members across operators with desired operator/host binding,
public identity-key status where visible, and source revision/freshness. It distinguishes unknown actual
placement from a verified observation. Restricted or unavailable graph boundaries stay explicit; a
missing external snapshot is not an empty hierarchy. Private vault handles and unrelated paths are
excluded. Rank or proximity in the graph grants no authority.

A workflow that uses only projections is not filesystem isolation. A worker with Git/read access to a
repository containing the raw card can read it directly. Where the card must be hidden, keep the raw
checkout under separate admitted custody and give workers only sanitized projections/snapshots.
The single-repository layout cannot promise to hide a tracked card from that repository's readers.
Do not add a pretend Pulse access control over already readable files.

## 4. Proposed version boundary

Recommend a separately specified `product-card/v5` for the added member key reference and registry
root reference. This is a proposed version choice, not an accepted or complete v5 format. Carry existing
v4 fields and preservation rules forward explicitly when the complete v5 format is accepted. The shipped
v4 reader/converter/checker continue to accept exactly their existing contracts, reject these new fields
and refuse v5. No v4 fixture, legacy identity or old conversion result changes here.

The future card adds `product.identity_key_registry` with exactly `path` and `sha256`, selecting the
index's exact bytes relative to the admitted host root. This does not embed the Git commit containing
the card into itself. The independently admitted caller context pins the complete card/index snapshot.
The registry index references operator documents; operator documents reference public-key files. No
reverse embedded digest creates a circular card/registry hash dependency.

A member adds `identity_key` with exactly these required fields:

| Field | Proposed value and meaning |
| --- | --- |
| `operator` | Stable admitted operator id; must equal the selected desired binding's operator |
| `generation` | Positive key generation explicitly selected for this member; never implicit newest |
| `key_id` | The mac-keyvault fingerprint: unpadded base64url of SHA-256 of canonical P-256 SPKI DER bytes |
| `reference` | One of the two closed objects below, pointing to public material only |

`reference` is either `{kind: local, path: PATH}` or
`{kind: remote, locator: LOCATOR, sha256: HEX}`. Local paths resolve under the admitted snapshot root
with rooted traversal and symlink-escape refusal. Remote locators identify an immutable repository
revision and file; mutable branches or `latest` refuse. Supplied remote bytes must match both the
locator's pinned digest and `key_id`. Remote never means that a private vault is remotely accessible.
An absent supplied public file is unavailable; verification never fetches it or reads a credential store.

The exact locator grammar, complete v5 field tables and any optional/no-key member class disposition
must be accepted before implementation. Do not assign actual account names or infer a naming scheme
from the logical member ids in this proposal. A later domain/name change must preserve the logical
member binding and follow the separately accepted naming decision.

## 5. Registry proposal and write authority

Recommend one host registry with an operator-owned namespace for each operator. This is one product
registry, not separate product cards or independently authoritative roots. Public key files also belong
to that namespace. A stable operator id is admitted independently, not derived from a workstation name.

| Candidate | Who proposes/authorizes changes | Foreign-key trust | Trade-off |
| --- | --- | --- | --- |
| Flat registry | One product custodian reviews all key entries | Requires the other operator to accept that custodian's endorsement of its members | Smallest file for one operator; obscures each operator's exclusive ownership in shared products |
| One registry, operator documents | Each operator authorizes its document/public keys; product custody admits the index and accepted changes | Receiving operator pins both agreed host and the other operator's authorized signer/binding | Slightly more files; explicit ownership without private-key sharing or another registry service |

Prefer the second candidate for shared products. With one operator it is still one operator document,
not a second system. Membership and public-key bindings remain in the single host. Machine placement
is a card binding, not a second registry namespace: an operator may change machines without changing
its stable operator id. Host-local mac-keyvault keys cannot move to another machine or OS user. A transfer to another
machine or OS principal requires a new target key and an endorsed continuity link; the stable logical
identity, contacts, history and obligations can remain. A separate off-station signing broker would be
a different custody model requiring its own owner decision, not an assumed capability.

Proposed closed JSON documents:

| Object | Required fields | Meaning |
| --- | --- | --- |
| Index | `schema`, `product`, `operators` | Schema literal `pulse-key-index/v1`; product id; nonempty operator-entry list |
| Operator entry | `operator`, `path`, `sha256` | Unique operator id and exact operator-document bytes under this snapshot |
| Operator document | `schema`, `product`, `operator`, `sequence`, `keys` | Schema literal `pulse-operator-keys/v1`; matching product/operator; positive sequence; key entries |
| Key entry | `member`, `generation`, `key_id`, `custody_scope`, `vault_record`, `public_key`, `status`, `status_at`, `not_before`, `not_after`, `supersedes` | Qualified logical member; positive generation; fingerprint; admitted custody scope; declared vault record; public descriptor; status and its UTC effective time; UTC validity interval; predecessor reference or null |
| Predecessor reference | `operator`, `key_id` | Earlier accepted key entry for the same qualified logical member, including across an admitted operator transfer |
| Declared vault record | `kind`, `service`, `purpose`, `version` | Kind literal `key`; service/purpose each match `[a-z0-9-]{1,40}`; explicit positive generation equal to the key entry generation |
| Public descriptor | `format`, `path`, `sha256` | Format literal `p256-spki-der`; public file under the operator's namespace; SHA-256 of exact canonical DER |

Key status is exactly `active`, `retired` or `revoked`. `status_at` states when that status took effect;
future-dated status transitions refuse rather than schedule an action. `not_after` is a timestamp or explicit null;
`not_before` is a timestamp. Unknown/duplicate fields, noncanonical keys, cross-product entries,
duplicate full record bindings or key ids, malformed intervals and namespace escapes refuse.
`custody_scope` identifies the explicitly admitted vault instance (operator, host and OS principal),
using a public opaque reference, not a private filesystem path or the task-station field. The exact
reference grammar/mapping belongs to the complete successor format. Generation is local to that
vault record, not a global member counter: a new vault can start at version 1. A record binding is
unique by `(operator, custody_scope, kind, service, purpose, version)`; the fingerprint must agree.
The selected key's custody scope must match the member's admitted current placement.
Every card key reference must resolve to exactly one entry with the exact operator/member/generation
and public bytes. References to retired or revoked keys remain representable so the projection can
explain their state; they do not pass current verification. At most one key entry per logical member is active. `supersedes`, when non-null, resolves to
an earlier accepted entry for the same logical member and cannot form a cycle. A target entry may
link to another operator's source entry only through an accepted transfer disposition; it cannot edit
that source operator's namespace. Generations need not increase across different vault records. Retired keys permit historical explanation, never current acceptance.
Revoked keys never pass current verification.

Use the confirmed mac-keyvault fingerprint unchanged: unpadded base64url of SHA-256 of canonical
SPKI DER (91 bytes for ec-p256). File `sha256` values use lowercase hexadecimal for the same digest;
these are explicit encodings, not two key identities. Parse exactly P-256 SPKI,
reject extra trailing data, and require canonical re-encoding equality; do not hash PEM whitespace or a
file path as the key identity. The [public contract fixture](../fixtures/identity-keys/README.md) supplies
an inert positive vector for review, never an admitted identity.

Each operator authorizes its own namespace; the product custodian cannot silently impersonate that
operator. Do not claim that JSON sections or Git paths enforce this by themselves. The admitted delivery
route must verify operator authorization and the diff's namespace scope before accepting a registry
change. Hosted repository write access is broader than this semantic authority. Repository administrators
and compromised signing authorities remain within the actual repository/trust model, not prevented by
this proposal. A different operator's unauthorized edit is rejected, including a correctly hashed edit.

For v1, use externally admitted operator signing keys and verified accepted repository commits as the
proposed endorsement route. These are existing authorized human/operator delivery signatures, not the
new member's own signature. Each receiving context pins which signer may endorse which operator and
which host snapshot it accepts. Do not treat an unverified Git signature, HTTPS access, a registry's own
new key, or a commit's claimed author name as endorsement. A stronger alternative is an offline-verifiable federation membership record admitting operator
roots, then each operator root endorsing its members' keys. This requires owner-approved roots,
root-key custody and a signed record/rotation/revocation format; it is not an existing enacted
contract. A flat mirror/index may aid discovery but cannot replace operator endorsement.
The precise cross-operator authorization agreement, including commit endorsements versus an
operator-root chain, remains an owner decision; this proposal grants no new signing access.

## 6. Foreign trust, enrollment, rotation and revocation

Enrollment has two facts: the operator created and retains a particular private-key generation, and the
product accepted that public key for a member. The operator supplies the public export through its own
admitted route; an accepted namespace update binds the exported bytes to the member. Registry data
attests that accepted binding, not independent proof of how the private key was generated or stored.
A future possession challenge requires its own fresh nonce, product/member/generation binding,
expiry and signature domain; no replayable self-signature is silently added here.

The mac-keyvault integration must select a schema-2 address, kind and explicit positive generation.
The documented contract supports non-extractable P-256 keys, SPKI public export, SHA-256 signing
and strict DER low-S signatures, with old generations retained by rotation. The confirmed source
contract identifies a record by `(kind, service, purpose, version)` and
fingerprints its SPKI as above. Bind that full declared tuple and fingerprint, never just an address
resolved to newest. The derived Keychain label is not a caller-supplied address. See the pinned
[record contract](https://github.com/relux-works/mac-infra/blob/e96dc9950bc360e75c8c31b71b131aca4508e0de/internal/keyvault/signerclient/record.go)
and [public-key/signature contract](https://github.com/relux-works/mac-infra/blob/e96dc9950bc360e75c8c31b71b131aca4508e0de/internal/keyvault/signerclient/result.go).
No actual key is created, described, exported or signed as part of this specification. The public record
descriptor is a declared logical address, not a private file/Keychain value or proof of custody. Local
private access configuration stays with the operator; no reference accesses a foreign private store.

A foreign key is accepted only when: the receiving operator has already admitted the endorsing
operator/signer and host; the supplied signed snapshot matches that admission; index/document/public
bytes match; the member/operator/generation match the card; and status/validity/freshness pass.
Trust cannot bootstrap itself from another key in the same untrusted registry. Missing foreign
endorsement yields untrusted, not an automatic trust-on-first-use prompt.

Rotation is an explicit accepted update: the owner creates a new generation locally, supplies its
public bytes, appends a new entry with `supersedes`, and changes the member's selected generation.
The old entry becomes retired or revoked according to the accepted disposition. Old keys are never
silently deleted from historical evidence. A rotation/transfer record is the accepted, operator-endorsed
predecessor link plus its pinned source snapshot/disposition; it is not proof that one private key
moved. Per-record generation and custody scope stay explicit. `sequence` orders that operator's accepted registry documents, not wall-clock account identity.
Changing operators requires both the relinquishing and receiving custody dispositions plus an
accepted placement/card update; it is not an ordinary unilateral same-operator rotation. Cross-host
or cross-OS-user movement under the same operator also creates a fresh target key record. Losing
the old private key must not force its use to authorize recovery: the independently admitted operator
authority owns that decision. No recovery authority is created by this proposal.

Keyvault has no revoked state or revocation list. Its `rotate` retains older usable generations;
`validity.not_after` limits signing, not public verification. Deleting a private generation is destructive
and is not part of this proposal or a required revocation step.

Revocation is an operator-authorized new document, with a higher sequence, marking the compromised
key revoked. Retirement and vault deletion do not count as published revocation; retained private-key
bytes do not keep a revoked registry binding valid. Other operators must receive and admit that new
snapshot. An offline reader cannot know about a later revocation it has not received.

The admitted context therefore pins accepted revisions/digests and operator sequence floors plus an
explicit freshness bound. Read-only `verify` cannot advance those floors itself. A separate admitted
refresh supplies new context and snapshots. Old/replayed sequences or stale inputs cannot return a
current-valid result. First contact without an externally admitted pin cannot claim rollback detection.
Historical validation must name its snapshot and must not be presented as current key validity.

## 7. Proposed read-only `pulse verify`

Proposed interface, not shipped:

```text
pulse verify --context CONTEXT --member PRODUCT/MEMBER --public-key PUBLIC_FILE --format json
```

`CONTEXT` locates an already admitted context; it cannot replace the host's invoking-principal/caller
binding or trust roots. The target member must be visible under the caller's accepted policy. The
public input is explicitly supplied and compared with the selected registry entry. Member selection
cannot impersonate that member or disclose a restricted identity. Local/remote resolution follows
section 4; no registry refresh, network request, filesystem write, subprocess, Keychain call or signing
operation occurs.

Evaluation order: admit caller/context and visibility; admit exact card/index/operator snapshots and
freshness/sequence floors; validate closed shape and relations; resolve exact visible member and key
reference; parse canonical P-256 public bytes and compare digest/generation/operator/member; enforce
active status and validity. An invalid context or inconsistent snapshot refuses the whole result, not a
partial list of identities. Relationship projection and verification use the same admitted snapshot.

Proposed receipt: `schema: pulse-key-verification/v1`, `binding_verified`, visible qualified `member`,
`operator`, `generation`, `key_id`, source revision/digests and `checked_at`. Successful exit 0 means
only that this public key matches the accepted current binding in the supplied fresh snapshot.
It says nothing about possession, actual process liveness, unique active placement, Matrix token
custody, authorization to act, or a revocation update not yet received. Emit no private handle or raw
failed input. Exit 1 is a contract/trust/status refusal; exit 2 is usage/encoding failure. Refusals carry
fixed rule ids and safe indexed locations only, without a restricted member's existence or identifiers.
Exact complete output field types/error taxonomy belong to the accepted implementation contract.

## 8. Shared host/card home — owner options

The one canonical card for a shared product needs an explicit home and accepted writer. These are
options for the operators, not decisions made by this amendment:

| Option | Home and card changes | Consequence |
| --- | --- | --- |
| Product-owned shared host repository | One repository/canonical branch; named card custodian; both operators' accepted change route | One source of structure; requires agreed repository custody, access and review policy |
| One operator hosts the canonical repository | Same single card; the other operator explicitly admits host/custodian and receives snapshots | Least new infrastructure; host availability/control belong to that operator |
| Per-operator replicas of one canonical history | Read-only mirrors pinned to the same accepted revisions; one agreed canonical writer | Useful availability; no concurrent divergent card writers or automatic conflict merging |

Recommend retaining one canonical card and explicit writer with operator-owned key namespaces;
choose its physical hosting/custodian through the existing shared-product decision. Independent cards
with conflicting placement/parents/keys are not a single shared product view. If each operator retains
its own product/card instead, expose authorized pinned cross-product projections and unresolved
boundaries; that is a different ownership model, not an automatic merge.

The conceptual identity/place/executor separation and single canonical writer align with the supplied
federation advice. The proposed operator-root chain, DNS naming and Exchange-like foreign-publication
visibility are design alternatives, not enacted shared rules. Preserve the existing explicitly pinned
visibility allowlist; full own-organisation visibility cannot silently override its self-only default.
Foreign records must be published/admitted by their owner as well as visible to this caller. Operator
ids, namespace authorization, physical host custody and placement transfer remain subject to the
shared agreement. Full-name/Matrix mapping follows the
[naming amendment](identity-names-v2.md); identity and obligations belong to the seat, with actual
reservations still requiring admission.

## 9. Acceptance examples and forbidden outcomes

These are contract counterexamples for review, not executed crypto/runtime tests. Each positive control
and nearby refusal follows the requirements above. Use synthetic public fixtures only in later tests.

| Claim | Valid control | Required refusal or limit |
| --- | --- | --- |
| Both host layouts | Dedicated host and single project supply the same admitted card/index shape | Inventing a second required repository for a single project is unnecessary |
| Stable placement | Same logical member changes declared host with a new target key and endorsed predecessor | A private key does not move with placement; two processes do not prove a single active member |
| Operator ownership | Operator A authorizes only A's changed document/public keys | A's valid signature over an edit to B's namespace is insufficient |
| Foreign endorsement | Receiver pins B's authorized signer and exact accepted host snapshot | A self-endorsed B key in an untrusted registry cannot establish B's identity |
| Exact binding | Visible member, operator, explicit generation and public bytes all match | Same key bytes for the wrong member/operator or a different generation refuse |
| Public-key encoding | Canonical P-256 SPKI DER with its computed SHA-256 id | P-384, trailing data, PEM-hash substitution or changed bytes refuse |
| Key status | Selected active generation inside its validity interval | Revoked, retired, expired or not-yet-valid entry cannot pass current verification |
| Rotation | Accepted higher sequence selects new generation and retains old historical entry | Implicit newest lookup, assumed same key on a new host/user, silent overwrite or unannounced operator transfer refuse |
| Freshness | Snapshot matches admitted pins/floors and explicit age bound | Older sequence, altered document or stale snapshot cannot claim current validity |
| Offline revocation | Fresh accepted snapshot explicitly contains the status used | No claim of observing a later unpublished/unreceived revocation |
| Member references | Admitted local public file or supplied exact immutable remote bytes | Escape, mutable remote locator, missing remote bytes or ambient credential lookup refuse |
| Personal projection | Caller sees its allowed peers/reports/superiors across both operators | Restricted member lookup reveals no identifier/existence; raw-card access is not magically hidden |
| Authority separation | Registry binding is verified using pre-admitted authority | A successful key match grants no caller impersonation, private-key possession or launch permission |
| Compatibility | Existing v4 files and fixtures retain their current verdicts | New fields in v4 and v5 input to a v4 tool refuse; no silent conversion |
| Read-only behavior | Verify supplied bytes and emit a bounded receipt | No write, fetch, vault read, sign, process launch or registry refresh |

Before implementation: accept the version/complete field and locator contracts; retain the confirmed mac-keyvault
public source contract; record cross-operator choices required by the actual shared pilot; add independent
public positive/negative crypto fixtures and meaningful no-effect tests. These gates apply to their
new capability, not to unrelated already-admitted account migration preparation.
