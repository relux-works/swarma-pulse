# Open contract inputs

The proposed contract is [specification v1](docs/specs/product-card-identity-pulse-v1.md).

| Input | Contract effect | Proposed treatment |
| --- | --- | --- |
| Zone/project namespace admission | The 2026-10-30 [naming amendment](docs/specs/identity-names-v2.md) settles full DNS/compact Matrix mapping and identity/obligations belonging to the seat; reverse lookup needs a pinned unique project owner. | Preserve existing mappings; admit no reservation/account until namespace and source/custody prerequisites are satisfied. |
| a2a resolver and onboarding descriptor | Requires versioned tuple/legacy resolution, explicit named connections and audience validation. | Preserve existing mappings; qualify the supported low-level named-connection route for the first migration. Implement a source change first only when an actual migration-blocking gap is demonstrated. |
| gr registration token | The token is not yet on the host; the host owner must provision its supported secure reference. | Keep the old working path until provisioning and secure-tool qualification are complete. Never substitute human credentials. |
| Host-project location | Determines the canonical card root and relative project/profile paths. | Preserve resolved targets during the explicit relocation to the declared host project. |
| Complete per-operator inventory/adoption | Bounds the migration and the identities that may change. | Account for every inventoried identity and retain explicit pending foreign, parked or blocked scope. |
| Pulse actual local use and future caller authority | The [local unsigned query](docs/pulse.md) reads one exact v3/v4 product card for self or opt-in hierarchy under the invoking OS identity; `A2A_AGENT` is unverified and freshness is unknown. | Qualify the actual input/context and invocation separately from fixtures. Cryptographic callers, broader visibility and the signed successor remain future work; no signer, host-config or conversion is required by this explicit MVP. |
| Agent DM routes and paired room declarations | Hierarchy projects current card relationships. A versioned directed full-MXID DM descriptor and paired room alias/ID amendment need separate accepted interfaces. | Preserve exact opaque room IDs and separate server routing when those fields are admitted; querying remains read-only and does not provision or verify JOINs. |
| Additional chat classes, private channels and runtimes | Each capability requires its own enacted eligibility/execution contract. | Preserve currently admitted behavior; naming validation alone grants no participation or launch. |

Delivery ordering, 2026-10-29: the first gr account pilot and its supported a2a/launcher preparation proceed independently
of the v4 card format, validator and converter. The complete v4 schema and conversion fixtures gate validator/converter
coding only. Existing low-level named-connection registration and explicit onboarding may be qualified without a card
reader. Final product/project spelling, secure bootstrap provisioning, any demonstrated required source fix and the
explicit runtime custody boundary still gate the corresponding live operation. Until v4 can represent the target, the
existing card may carry a reviewed pending-conversion note in a supported desired-fact field; actual migration evidence
stays outside the card. This ordering clarification leaves the specification's identity and preservation contracts intact.

The [complete v4 format](docs/format-v4.md) and [synthetic inputs](docs/fixtures/v4/full/README.md) are accepted design
contracts. The read-only Go validator and explicit converter are now supplied;
The optional read-only [desired-change checker](docs/change-check.md) verifies a candidate and exact prior rollback;
The read-only [Pulse query](docs/pulse.md) is supplied for one explicitly admitted snapshot. Actual host input/adoption, changed-location conversion dispositions and live integration acceptance remain separate. Future Go consumers may import the reviewed
`internal/cardv4` package from this module; consumers outside this module need a separately reviewed public-package split.
No consumer adoption, v3 retirement or live migration follows from validator availability.

The [host and identity-key registry amendment](docs/specs/product-host-key-registry-v1.md) is a proposal for review:

| Input | Contract effect | Proposed treatment |
| --- | --- | --- |
| Shared product host and canonical writer | Determines repository custody, card changes and admitted snapshots across operators. | Present shared-host, operator-hosted and canonical-history mirror options; preserve one agreed writer. |
| Grant integration and operator namespace admission | The local owner decision of 2026-10-30 selects scoped verbal trust now and the existing grant model later; human-signed delivery is provenance, not operational authority. | Pin exact module versions, key-purpose mappings, physical repository/root-key custody and rollout before integration. The other operator's owner has not yet accepted this proposal; no new verifier, grant registry or live trust is enacted. |
| mac-keyvault enrollment mapping | Source contract fixes explicit schema-2 generation, SPKI fingerprint and strict low-S signatures; real member-to-record selection remains unassigned. | Preserve the confirmed public contract and test vector; provision no key or private access during specification preparation. |
| Card and caller-envelope version | Adds registry/member public references without changing closed v4 or caller v1 decoding. | Propose explicit successor versions; accept complete field/locator contracts before implementation. |
| Cross-operator placement and visibility | Determines accepted participant location, transfer and personal view boundaries. | Keep logical identity and executor distinct; changing host/OS user creates a new key linked by operator endorsement. Shared policy remains open. |
