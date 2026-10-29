# Read-only product-card validator

`cmd/card-v4-validate` is a thin Go CLI over `internal/cardv4`. The library accepts bytes and explicit source/reservation
premises. It has no filesystem, network, process, credential, registration or runtime API except the optional metadata-only
`LayoutFS` interface. The CLI reads named files and opens an admitted filesystem root; it creates no product artifacts.

## Input and diagnostic order

The default schema is v4. `--version 3` selects the written FORMAT v3.1 source contract, including its legacy string,
endpoint and selector rules. This is an independent Go reader: it never executes another validator or derives its
expected outcomes from another implementation. A v4 document under that mode refuses at the v3 discriminator.
Legacy aliases expand with cycle/expansion limits; v4 rejects tags, anchors/aliases, merges, duplicate keys and multiple
documents before conversion to generic maps. UTF-8, byte, depth, count and reporting-chain limits are enforced separately.

Validation deliberately reports the first failure in this deterministic order:

1. Raw size/UTF-8, YAML document shape and syntax, then Node-tree restrictions/depth in input order.
2. Schema discriminator; forbidden credentials/live facts in sorted mapping-key order; collection bounds.
3. Closed object shape, unknown fields in sorted-key order, required then optional fields in sorted-key order;
   named maps use sorted keys, arrays use their declared order.
4. Cross-references and binding/topology facts, then members and projects in document order; local graph walks use
   sorted logical keys. Source checks run where the referencing member is checked.
5. Optional local layout, only after static acceptance.

Diagnostics contain implementation-owned rule names and indexed locations. Mapping keys controlled by the input never
appear in them. V3 refusals map to FORMAT rule IDs R1–R16; malformed collections and documents retain exit 2. A refusal
returns no partial document/view and prints no input value, path, parser exception or backtrace. Tests assert exact
rule/location/exit triples rather than accepting any failure. Future callers must not treat first-failure ordering as an
exhaustive list of all problems in an invalid card.

`--source-manifest` names a JSON mapping with `source_pins: [{file, revision, sha256}]`. Each file is relative to that
manifest's directory; a rooted read refuses escape. Actual source bytes are supplied to the library with their kind,
revision and digest, never a precomputed identity map or a fetch instruction. V3 sources must pass their own reader.
A supplied service descriptor is separately admitted data containing member, MXID, connection and admission reference;
accepting the synthetic descriptor does not qualify a new service onboarding API.
`--reservations` reads `reviewed_reservation_premises` from a JSON mapping. Reservations must match the full holder,
domain, product, primary role and star. Neither input grants a real reservation or changes a store.

A member's first `a2a.rooms` entry declares its own team room. A project's worker-channel declaration does not override a
manager's migrated room: the supplied cutover fixtures intentionally retain project channels while changing the chosen
member's associated desired a2a fields. Room resolution, membership and independent audience authority remain external.

## Read-only comparison and layout APIs

`VerifyConversionPreservation` compares supplied source/target cards and explicit mappings/path bases. It never produces
a converted card. It checks source declarations, project/member membership, paths, profile selection, bindings, selectors
including multiplicity/ownership, runtime, desired state and optional station. It refuses unsupported source extensions
rather than silently dropping them. It currently checks the supplied local-to-local relocation disposition; separately
reviewed changed local/remote conversions remain a later converter capability.
`VerifyPilotPreservation` permits only the selected member's identity, binding and associated a2a delta. Logical source
pins, station and all non-pilot facts remain exact. Rollback data specifies complete prior-v4 bytes; this unit does not execute
restoration, schema downgrade or file replacement.

`CheckLayout` takes an already admitted root through a read-only interface. Lexical paths and each symlink target are
checked before target metadata access. Profile directories have the narrower profiles_root boundary. Rooted filesystem
access prevents an out-of-product-root dereference even if links change during a check. Remote locations/task locators
are skipped; an independently declared member clone may share their old local spelling and remains a bounded declaration.
No checkout, directory or task file is created to make a check pass.

## Executable evidence and limits

Go tests execute all 34 selected accepted vectors, their positive neighbors and the seven original full-card negatives.
`internal/cardv4/testdata/vector-oracles.json` and `negative-oracles.json` record language-neutral case IDs, expected
result, rule, location and exit code. The original fixture files remain unchanged. Parser, collection and hop-boundary
pairs exercise their actual isolated stages, not unrelated whole-card failures. Six v3 fixture sources retain recorded
4-member/3-project/1-profile outcomes. Rule-cited v3 counterexamples are in `v3-rule-cases.json`; the optional external
source lane retains its recorded counts without persisting its contents.

Additional tests cover opaque source-attested logical keys, closed schemas, service branches, source authority,
reference/parent failures, arbitrary-length version comparisons, invalid encodings, output safety and input immutability.
Known-violation controls establish that the byte-inventory and captured-output checks detect a changed file/extra partial
file and a planted partial value. The outside-access spy is armed after isolated tree setup; an inert forbidden inspection
must be recorded by the control, while escaped-link operations record zero destination attempts.

The static import/API admission scan records and denies every process API in production source; its planted process
import/call is a failing control. It also rejects writes, network-client/unsafe/plugin imports and unadmitted dependencies.
There is no production process-executor interface or child allowance. This is a source admission instrument, not an OS
sandbox or a trace of arbitrary dependency machine code. The YAML module is pinned; tests exercise its real Node parser.

The three atomic-converter/registration-tripwire vectors are explicitly deferred to the converter unit. The actual
service agent-launch refusal and conversational control require an external launcher integration; they are unverified
here. Static service acceptance does not claim that runtime pair passed. Pulse caller/admission and live migration are
also outside this validator. Go's test runner executes every selected subtest after an earlier failure; a failed selected
lane makes the overall test command fail. An unselected external integration lane is reported as skipped, never passed.
