# Flight Logbook

> Institutional memory. Concise, factual, high-signal.
> Newest entries first. One block per insight.

## 2026-11-17

### 2101 — Governance view preserves the explicit declaration boundary

- RESULT: The opt-in governance selector returns the closed 17-field v5 view while legacy selectors keep their prior schemas and reject v5 sources.
- EVIDENCE: Complete ordinary/race suites, vet and nine actual CLI cases pass; source inputs remain unchanged. An isolated eighteenth output field fails specifically at the closed-shape assertion.
- SCOPE: internal/pulse/governance.go; internal/pulse/query.go; internal/pulse/governance_test.go.
- LIMIT: Public records and unresolved references remain declarations, with no authority or key possession verification.

### 2028 — Typed public scanning preserves the prose boundary

- FIX: Validate the complete v5 card and key descriptors before replacing only public_bytes/key_id in a scanner-only copy; return the original declaration.
- EVIDENCE: Ordinary/race suites and privacy controls pass. Ignoring the second scan in an isolated copy makes the test fail specifically on accepted forbidden prose; malformed descriptors still refuse.
- SCOPE: internal/cardv4/governance.go:73; internal/cardv4/v5_test.go:205.
- LIMIT: Local declarations do not prove possession, identity or authority.

### 2028 — Exact decoder APIs pass the source effect boundary

- FIX: Admit only required curve-decoder, encoding and explicit-time entry points; pair them with forbidden signer, ambient-clock and unlisted-import controls.
- EVIDENCE: Ordinary/race suites pass, including production effect admission and registration/conversion controls; vet passes.
- SCOPE: internal/cardv4/effects_test.go:32; internal/cardv4/effects_test.go:116.
- LIMIT: The AST check does not trace dependency machine code.

### 1832 — Canonical public encoding can resemble a token

- ROOT CAUSE: The prose secret-pattern scanner also examines canonical public point encodings. A finite P-256 control containing a token-like substring fails despite valid encoding, key id and digest.
- SCOPE: internal/cardv4/document.go; internal/cardv4/v5_test.go, TestGovernancePublicEncodingPrivacyBoundary.
- STATUS: Pending a narrowly typed public-descriptor correction; token-like prose must continue to refuse.

### 1831 — Pure dependency additions require source-boundary qualification

- ROOT CAUSE: The source/API effect checker rejects imports outside its closed list, including the new curve decoders. Registration and conversion control tests consequently fail before entering their intended operation.
- SCOPE: internal/cardv4/effects_test.go; internal/cardv4/registration_boundary_test.go.
- STATUS: Pending exact decoder API admission with forbidden-call controls. Dependency machine code remains outside the AST check's guarantee.
