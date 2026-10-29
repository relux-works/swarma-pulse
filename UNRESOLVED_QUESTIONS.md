# Open contract inputs

The proposed contract is [specification v1](docs/specs/product-card-identity-pulse-v1.md).

| Input | Contract effect | Proposed treatment |
| --- | --- | --- |
| Canonical product/project spelling | Determines the full identity tuple and host-project reference. | Keep naming examples provisional; reserve no name until the canonical ids are settled. |
| a2a resolver and onboarding descriptor | Requires versioned tuple/legacy resolution, explicit named connections and audience validation. | Preserve existing mappings; qualify the supported low-level named-connection route for the first migration. Implement a source change first only when an actual migration-blocking gap is demonstrated. |
| gr registration token | The token is not yet on the host; the host owner must provision its supported secure reference. | Keep the old working path until provisioning and secure-tool qualification are complete. Never substitute human credentials. |
| Host-project location | Determines the canonical card root and relative project/profile paths. | Preserve resolved targets during the explicit relocation to the declared host project. |
| Complete per-operator inventory/adoption | Bounds the migration and the identities that may change. | Account for every inventoried identity and retain explicit pending foreign, parked or blocked scope. |
| Pulse host admission and visibility inputs | Determines the trusted signer, caller context, pinned snapshots and visible member set. | Refuse unknown caller authority; use the specified self-only default when no visibility policy is pinned. |
| Additional chat classes, private channels and runtimes | Each capability requires its own enacted eligibility/execution contract. | Preserve currently admitted behavior; naming validation alone grants no participation or launch. |

Delivery ordering, 2026-10-29: the first gr account pilot and its supported a2a/launcher preparation proceed independently
of the v4 card format, validator and converter. The complete v4 schema and conversion fixtures gate validator/converter
coding only. Existing low-level named-connection registration and explicit onboarding may be qualified without a card
reader. Final product/project spelling, secure bootstrap provisioning, any demonstrated required source fix and the
explicit runtime custody boundary still gate the corresponding live operation. Until v4 can represent the target, the
existing card may carry a reviewed pending-conversion note in a supported desired-fact field; actual migration evidence
stays outside the card. This ordering clarification leaves the specification's identity and preservation contracts intact.

The [complete v4 format](docs/format-v4.md) and [synthetic inputs](docs/fixtures/v4/full/README.md) are accepted design
contracts. The read-only Go validator is now supplied;
converter, Pulse and live integration acceptance are separate. Future Go consumers may import the reviewed
`internal/cardv4` package from this module; consumers outside this module need a separately reviewed public-package split.
No consumer adoption, v3 retirement or live migration follows from validator availability.
