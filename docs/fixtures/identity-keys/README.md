# Public identity-key contract fixture

This is public test material, not an enrolled product identity or a trusted signer. No private key is
included or generated. The source fixture's private half is not in its repository. Never use service
`test` or this fingerprint for a real member admission.

`fixture-key.json` is byte-for-byte the public vector from
[mac-infra at e96dc9950bc360e75c8c31b71b131aca4508e0de](https://github.com/relux-works/mac-infra/blob/e96dc9950bc360e75c8c31b71b131aca4508e0de/internal/keyvault/testdata/signer-v1/fixture-key.json).
It contains exactly `spki_der_base64`, `digest_hex` and `signature_der_hex`. The declared source test
record is `(kind: key, service: test, purpose: signer, version: 1)`; it is not evidence of a local vault item.

Expected fingerprint: `2nXKWMnWmV5HurpMyvnrt-tepY812lt13cG_EjD402s`, computed as unpadded
base64url(SHA-256(SPKI DER)). The decoded P-256 SPKI DER is 91 bytes. The signature is strict DER,
low-S ECDSA over the given SHA-256 digest. These public facts are test expectations; they grant no
identity, filesystem or signing authority.

| Review case | Required result |
| --- | --- |
| Original SPKI, digest and signature | Public cryptographic verification succeeds; no product binding is inferred |
| Same public key and signature, one digest bit changed | `signature_invalid` |
| Original signature with an extra trailing byte | `invalid_signature` |
| Replace low S with `n - S`, keeping R and strict DER encoding | `high_s_refused`; no permissive fallback |
| Correct fixture proposed as a production registry identity without operator admission | Untrusted, even though the public signature verifies |

These are specification vectors/counterexamples, not an executable test runner or evidence that Pulse
is implemented. Future tests must exercise the production verifier and pair each refusal with the
original valid vector. The registry-binding, ownership and no-effect controls are listed separately in
the [proposal](../../specs/product-host-key-registry-v1.md#9-acceptance-examples-and-forbidden-outcomes).
