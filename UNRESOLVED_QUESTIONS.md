# Unresolved decisions

The proposed contract is [specification v1](docs/specs/product-card-identity-pulse-v1.md). None of the following entries
claims acceptance, provisioning or rollout authority.

| Input or decision | Accountable route | Recommendation and alternatives | Blocks |
| --- | --- | --- | --- |
| Exact specification verdict | Product custodian routes independent exact-head review. | Review the signed candidate as a whole; require revisions for concrete defects. | Implementation. |
| a2a naming/shared policy and onboarding descriptor | a2a source custodian and affected operators. | Adopt the five-field domain/product/project/role/star resolver; keep legacy operations explicit during migration. Alternative transport may change after source advice without weakening validation. | New identity onboarding. |
| Secure gr registration and pending-recovery qualification | Host owner through product custodian; a2a source owner qualifies tooling. | Complete the supported secure route. Preserve the old working path while unavailable; never read/export bootstrap values or retry ambiguous registrations. | First account registration. |
| Canonical host project and card relocation | Owning product custodian, with host/project custodians. | Use explicit `swarma-host` for swarm, preserve privacy and relative targets; defer relocation until custody and consumers are verified. | Card relocation. |
| Complete per-operator inventory and foreign adoption | Each operator's keepers through the shared advice process. | Account for all identities, pilot this orchestrator, then migrate keeper by keeper. Foreign pending scope remains explicit. | Claim that all identities migrated. |
| Publication availability | Repository custodian. | Use the admitted local signed candidate for review until the private repository is provisioned through its authorized route. | Hosted PR/check/landing flow. |
| Worker chat admission, isolation, private channels and additional runtimes | Their existing policy/source custodians. | Preserve current non-chat/admitted behavior; do not couple this specification to automatic adoption of unlanded policies. | Only the affected new capability. |
