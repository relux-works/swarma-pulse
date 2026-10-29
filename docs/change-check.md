# Read-only desired-change check

`card-v4-check-change` checks a supplied prior v4 card, a proposed per-member change and an exact rollback snapshot.
It is an optional preparation tool. A passing check proves no registration, custody, cutover or recovery operation
and grants no permission to perform one. It never generates or publishes a card, allocates a reservation or changes
a consumer, session, connection, room or credential store.

Build with Go 1.25.12 or a newer fixed toolchain (Go 1.26 requires 1.26.5 or newer):

```sh
go build -o .temp/bin/card-v4-check-change ./cmd/card-v4-check-change
.temp/bin/card-v4-check-change \
  --root docs/fixtures/v4/full \
  --prior converted-v4.json \
  --prior-sha256 dae2b66a6b38ce9bed4dd916acff53c88f4bb2a046c2d56d49d98cc4cc4ded82 \
  --candidate pilot-v4.json \
  --candidate-sha256 75986ca3681212cce85be2c5ba1b116d13de20e066674470f0675129871b500c \
  --rollback rollback-v4.json \
  --rollback-sha256 dae2b66a6b38ce9bed4dd916acff53c88f4bb2a046c2d56d49d98cc4cc4ded82 \
  --pilot aurora/beacon_orch \
  --source-manifest manifest.json --reservations expectations.json
```

The example is entirely synthetic; its reservations and source labels are premises. For actual preparation, the
custodian supplies authorized snapshots and expected digests through its admitted handoff. A matching caller-supplied
hash proves byte integrity, not the source's authority or an immutable Git revision. The command fetches nothing.
Every flag shown is required; there is no ambient discovery, environment fallback or fault flag.

All file arguments except `--root` are relative to that admitted read root. Source-manifest entries are relative to,
and confined within, the manifest's directory. Native `os.Root` constrains source, snapshot and metadata reads;
cleaned relative spellings and in-root links are allowed, while escapes and nonregular inputs refuse. Each file is
limited to 1 MiB. The sanitized snapshot tree must remain under its custodian's exclusive write custody throughout
the check. A rooted descriptor does not freeze mutable files or make a check safe against concurrent tree replacement.

The manifest supplies `source_pins`, an array of `{file, revision, sha256}` objects. Every actual source file is read
and its digest checked. The reservations file supplies `reviewed_reservation_premises`, using the validator's existing
reservation shape. Unknown fields inside these list entries refuse. Other top-level fixture observation fields are
ignored; they grant no authority. Metadata uses the bounded duplicate-free v4 mapping parser and its YAML restrictions.

After CLI input/metadata admission, the pure `cardv4.CheckChange` API checks prior, candidate and rollback in that order.
For each snapshot it checks the byte budget, required lowercase SHA-256, actual digest and complete v4 validation
against supplied legacy bytes and reservations. `Options.Version` cannot select a v3 interpretation. It then invokes
the existing `VerifyPilotPreservation`: only the selected member's identity, binding and associated desired a2a fields
may differ. All other facts, logical source pins, station, profile/runtime/locks, desired states, ordering and nonpilot
selectors remain exact. An unchanged candidate can pass; that does not claim that a cutover happened.

Finally, rollback must equal the complete pinned prior **byte-for-byte**. Equivalent parsed data with different
whitespace refuses `rollback_bytes_mismatch`; a schema downgrade or a copy of the candidate is not a rollback.
The original preservation contract and allowed change surface are unchanged. This checker does not constrain the
pilot's a2a delta beyond that existing contract and complete v4 validity.

Success prints only `change-valid` and exits 0. A contract/pin refusal exits 1; usage, malformed input, read and shape
failures exit 2. Refusal has empty stdout and one safe `RULE name: location` diagnostic. Snapshot-validation locations
start with `$[0]`, `$[1]` or `$[2]` for prior, candidate or rollback. Existing pilot-comparison locations retain their
card-relative form. CLI metadata/read failures precede the pure API's comparisons; the first applicable rule is reported.
No raw input values, file paths, partial card or migration report are emitted. Actual runtime, service-launch and
Linux qualification remain separate.

Exact focused checks are `go test -count=1 ./internal/cardv4 -run TestCheckChange` and
`go test -count=1 ./cmd/card-v4-check-change -run TestCLIChange`. The tests reuse the converter's inventory, captured-output
and applied source/API admission instruments, including an inert one-call denied plant. Those bounded instruments
are not a syscall trace, OS sandbox or proof about arbitrary dependency machine code. Run all existing lanes as well.
