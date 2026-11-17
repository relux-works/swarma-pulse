# Governance fixtures

governance-v5.json is an inert v5 local-snapshot fixture for the public [format](../../format-v5.md) and [query](../../pulse-governance.md). It derives neutral agent declarations from the existing v4 example's structure and adds a human researcher, two owner principals, an empty-key owner, public identity/grant descriptors and an unresolved external meta-keeper. No fixture key is an admitted owner, issuer or trust root.

Public constants are the RFC8032 first Ed25519 public-key example, the SEC2 P-256 generator's compressed public point and BIP340 test-vector0's x-only public key. Only public points are present. They do not authorize signing, derive a private key or grant ownership.

Primary references: [RFC 8032, section 7.1](https://datatracker.ietf.org/doc/html/rfc8032#section-7.1),
[Standards for Efficient Cryptography 2 (SEC 2), section 2.4.2](https://www.secg.org/sec2-v2.pdf),
and [BIP-340 test vectors](https://github.com/bitcoin/bips/blob/master/bip-0340/test-vectors.csv).

The valid fixture is intended for an explicit evaluation time after2026-11-17T00:00:00Z. The real CLI uses its current query clock. Parser tests use an explicit clock and must distinguish a future declaration timestamp from a malformed key.

Required nearby controls:

- Old selectors refuse v5; existing valid v3/v4 selectors remain unchanged.
- Governance on v3/v4 returns undeclared and ignores owner-like legacy extensions.
- Human caller/contact data never impersonates an agent; role/title never changes responsibility.
- Canonical Ed25519 point, P-256 compressed point and BIP340 identity pass; identity point, mixed-order/noncanonical Ed25519, off-curve/field-overflow points and BIP340 grant-purpose fail.
- Duplicate principal/contact/key-pair, dangling owner/human/meta reference, unsupported fields, key-id/digest mismatch and future state time refuse before output.
- A malformed key belonging to an unselected principal still rejects the whole card.
- Valid declared_state is paired with a forbidden live status plant; the expected guard refusal, not any crash or setup error, is the control.
- Complete output has exactly17 top-level fields and the declared key order rules, with no verification or hidden graph claims.

These are contract fixtures and expected claims. Runtime test evidence is supplied separately; this file does not claim that the new implementation already passes them.

All domains, owner contacts and rooms are neutral example values. Legacy source pins illustrate local-snapshot declarations only; the reader does not fetch or attest them. This fixture is not an exact migration-preservation oracle.
