# Open contract inputs

The proposed contract is [specification v1, revision 2](docs/specs/product-card-identity-pulse-v1.md).

| Input | Contract effect | Proposed treatment |
| --- | --- | --- |
| Canonical product/project spelling | Determines the full identity tuple and host-project reference. | Keep naming examples provisional; reserve no name until the canonical ids are settled. |
| a2a resolver and onboarding descriptor | Requires versioned tuple/legacy resolution, explicit named connections and audience validation. | Preserve existing mappings; adopt the source-owned contract before registering new names. |
| gr registration token | The token is not yet on the host; the host owner must provision its supported secure reference. | Keep the old working path until provisioning and secure-tool qualification are complete. Never substitute human credentials. |
| Host-project location | Determines the canonical card root and relative project/profile paths. | Preserve resolved targets during the explicit relocation to the declared host project. |
| Complete per-operator inventory/adoption | Bounds the migration and the identities that may change. | Account for every inventoried identity and retain explicit pending foreign, parked or blocked scope. |
| Pulse host admission and visibility inputs | Determines the trusted signer, caller context, pinned snapshots and visible member set. | Refuse unknown caller authority; use the specified self-only default when no visibility policy is pinned. |
| Additional chat classes, private channels and runtimes | Each capability requires its own enacted eligibility/execution contract. | Preserve currently admitted behavior; naming validation alone grants no participation or launch. |
