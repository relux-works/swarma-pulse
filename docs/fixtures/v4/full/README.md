# Full synthetic v4 conversion fixtures

These are complete card documents for design review, encoded as JSON within the supported YAML subset. They are not
runtime or account evidence. No v4 validator/converter exists. Synthetic profiles, admissions, stations, source revisions
and star reservations are premises, never real installations or allocations. The source manifest pins actual input bytes.

- source-v3.json: baseline v3 card; three chat members and one parent-native developer, explicit/inherited bindings,
  populated optional station and neighboring absent stations. Validate with the existing v3 reader, without layout checks.
- converted-v4.json: all existing chat identities remain legacy-exact; non-chat prospective identity is explicitly supplied.
- pilot-v4.json: only beacon_orch identity/binding/associated desired a2a fields change; its station is preserved.
- rollback-v4.json: byte-exact prior-v4 converted card only, not source-v3.json bytes or a schema downgrade. No
  migration state, cursor or receipt is added to a card.
- remote-v4.json: a separate reviewed location disposition, not a claim that local-to-remote relocation is automatic.
- services-v4.json: separate descriptor-backed legacy a2a service with zero selectors and a prospective parent-native
  service without an account/profile/model/hooks. The service descriptor is synthetic source evidence, not a shipped API.
- expectations.json: independently specified source/pilot maps, original path targets and preserved station.
- reject-*.json and negative-cases.json: full counterexample cards with the intended refusal contract and a nearby valid
  control. Station loss/change are conversion-preservation failures against the source, not invalid standalone optionality.

Use paths relative to this directory for manifest entries. Fixture layout roots are symbolic under fixture-root;
source and target card directories are siblings; the target is the host project itself (location and keeper clone
become .), while sibling project/profile targets coincide. Their task-system path
base changes from card-relative ../project/tasks.md to project-relative tasks.md, preserving the same target. No on-disk
project tree or remote checkout is created.

The [precise family vectors](VECTORS.md) and vectors-index.json supply concrete legacy, cutover, service,
location, carry-over, raw-parser and boundary inputs now. Each has an explicit valid neighbor, intended refusal and
forbidden effects. Parser/count/graph budget pairs name their isolated stage; their positives do not claim whole-card
semantic validity. The two full tuple cards exercise exactly 64 and 65 characters. Operation-cases.json separately
records service agent-launch refusal with a conversational admission control. All execution remains pending; only
authoring and document consistency are claimed here.

Compatibility pairs preserve the actual v3 source keys and selectors: compat-id includes beacon=orch with a retained
logical source pin through cutover; compat-repeated-required preserves within-group multiplicity; compat-cross-group
preserves equal selectors under two distinct ownership declarations. Both source and expected complete converted
cards are supplied, with actual source-byte pins and explicit expectations. No normalization or deduplication occurs.

coverage.json maps each outline fixture requirement to concrete input files/case ids; the compatibility families add
regression witnesses for supported ids and selector ownership. VECTORS.md defines exact construction/mutation semantics,
including the separate budget stages and inert no-registration instrument control.
