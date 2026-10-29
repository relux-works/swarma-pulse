# Precise isolated fixture vectors

Status: authored design inputs only; none of these future product operations has run. The vectors are data, not a new
command, fixture runner, source resolver or permission to create accounts/files outside an isolated test directory.
Each vectors-*.json case names its family, exact inputs, valid neighbor, expected refusal/result and forbidden effects.
The seven original full-card negatives remain in negative-cases.json; their inputs are not replaced by these vectors.

## Input and mutation semantics

All file names resolve in this directory. Start each case from fresh independent copies of the named files; never
mutate a previous case. source_pins selects the actual immutable source bytes/revisions in manifest.json. A
source_selection replaces that set for this case; replace_supplied_v3_source replaces the baseline v3 entry only;
remove_supplied_sources removes actual supplied bytes while retaining their manifest references. None fetches a source.
Synthetic star reservations/admissions in expectations.json and explicit case additions are premises, not real grants.
Each compatibility pair names its complete conversion-input file in compatibility-expectations.json; those inputs
carry the actual variant source pin and unchanged path/binding choices, with explicit renamed-key expectations where
needed. Additional gr connection/binding definitions and the admitted root are explicit inputs, never inferred from
the expected output.

For a rejection case, valid_neighbor is the complete positive setup. Overlay counterexample on that setup; target_patches
are then applied to its card in list order. A path is an array of exact mapping keys or zero-based array indexes.
replace requires an existing location; add requires an absent mapping key; remove requires an existing location and
has no value. A patch error is a fixture setup failure, never the expected product refusal. Values are literal JSON.
A counterexample card replaces the input card, retaining the selected operation/context. A target source_ref revision
and digest change does not change its member key unless a patch explicitly says so. Every source selection supplies
its actual bytes, not a precomputed MXID map. valid_neighbor.expected applies only to the positive control; the counterexample uses the outer expected_refusal
and its explicit expected_exit, never an inherited accept oracle. Positive expected means acceptance by the named
operation/stage only.
Forbidden effects apply to the rejecting operation; a successful conversion neighbor explicitly permits one complete
atomic destination replacement. All other writes, network, registration and credential operations remain forbidden.
Safe refusal diagnostics may identify the rule and indexed field; raw values, backtraces and partial card/view output
are forbidden. A different setup error or refusal for an unrelated rule does not satisfy a vector.

The standalone-valid mixed-reject-nonpilot-state.json changes storage_orch.desired_state only, from active to parked.
It remains valid with the original source mappings. Comparing it with converted-v4.json under the allowed beacon_orch
cutover must specifically refuse nonpilot_preservation. reject-nonpilot-change.json is the separate legacy-source
mismatch witness; it is not claimed as a cutover-preservation control.

## Named operation boundaries

- existing-v3-reader uses the unchanged supplied v3.1 reader without layout checks; schema refusal must have exit 2
  and identify not a product-card/v3 document, rather than a path/setup failure.
- validate-card-and-sources runs the complete proposed static contract with supplied immutable sources and synthetic
  admission premises. verify-conversion-preservation additionally compares all original desired facts with their
  explicit representation/path transformations. verify-pilot-preservation compares a prior v4 card with the proposed
  pilot, allowing only that member's identity/binding/associated a2a fields. Neither starts a runtime operation.
- convert-atomic is a future explicit conversion into one isolated destination. Its complete source revision,
  mappings/path roots/prospective choice and expected card are supplied. Interpret only source_revision/source_sha256,
  host_project, external_parent_mapping, source_card_root/target_card_root/admitted_root, additional_connections,
  additional_bindings and prospective_nonchat_choices from conversion_inputs as conversion parameters. The resolution
  maps/path targets and nonchat_members lists are expected observations, not instructions to create extra members.
  Reviewed reservation premises are separate synthetic admission inputs. Materialize the source bytes named by card
  at source_materialized_path under a fresh isolated test_root; destination is relative to that same test root.
  No actual host root is used. Initial destination bytes are exactly the
  UTF-8 destination_initial_utf8 value. A fault after exactly two members have been processed, before publication,
  must leave those bytes unchanged and produce no partial stdout/result or abandoned output file. Its positive
  neighbor without the fault emits the complete expected card through one atomic replacement. Compare its parsed
  data with expected_output; arbitrary output whitespace is not a preservation failure. Prior-v4 rollback alone uses
  the separate explicitly byte-exact comparison.
- An armed registration_tripwire records every attempted registration before transport and fails on the first attempt.
  The positive conversion must record zero attempts, including rejected ones. registration-tripwire-self-check supplies
  zero versus one inert synthetic attempt to prove that instrument detects the forbidden event without a network
  endpoint or credential. A skipped/unapplied spy, setup failure or transport error does not satisfy that control.
- validate-layout checks local project directories, their task entries and local profile directories in the supplied
  isolated tree. Layout fixtures list every required directory/file and symlink. Symbolic sandbox is the test root,
  not a real repository path; the test runner must instantiate it only in its isolated temp directory. Symlink targets
  resolve relative to the link's parent. sandbox/outside exists in the isolated fixture so dangling-link/missing-target
  failure cannot substitute for escape detection. Arm the spy after fixture setup, before the tested operation.
  The outside_access_spy records destination inspection/open, not reading the
  in-root link metadata. Refuse escape before following it into sandbox/outside. The safe-link neighbor is inside the
  admitted root. A remote location/task locator is never opened locally. Member clone/git_dir declarations retain
  their path bases and containment rule, but this optional layout operation does not require those checkouts to exist.
- check-forbidden-desired-facts checks the live-state/credential-field rule before closed-object shape validation.
  Those explicit new fields would also fail the latter stage; these vectors isolate the forbidden-fact check rather
  than prescribing an arbitrary first diagnostic for the whole validator. The credential canary is the literal
  SYNTHETIC-NOT-A-CREDENTIAL; no real token, key or credential value is supplied.

## Raw document constructions

Unless replaced by a construction, input is the exact named file bytes. All raw transformations start from the parsed
named JSON fixture rendered in UTF-8, no BOM, mapping insertion order preserved, separators comma and colon with no
extra spaces, no ASCII escaping of Unicode, one final LF. Mapping key/value tokens still use JSON string escaping.

- duplicate-json-object-key inserts a second key/value pair immediately after the original named key, with the given
  second_value. All other fields remain unchanged. Equal duplicate values are still forbidden duplicate keys.
- yaml-tag-json-flow-value prefixes the named string value token with the literal tag and one space.
- yaml-anchor-alias-json-flow prefixes the anchor_path value token with &fixture_copy and one space, and replaces
  the alias_path value token with *fixture_copy. The anchor is defined before the alias. No unresolved alias is used.
- yaml-merge-json-flow-object inserts the unquoted key << with the given literal merge_mapping as its value into the
  named mapping, before its other keys. No alias is needed; the existing operator field explicitly overrides the merge
  to the same value. This isolates forbidden merge syntax instead of failing because an anchor is undefined.
- concatenate-yaml-documents joins the two complete rendered documents with separator_utf8. Each document alone is
  a valid card, but the combined input is forbidden.
- append-ascii-spaces-to-byte-count starts from the exact named file bytes and appends U+0020 until the UTF-8 byte
  length equals target_bytes. A smaller target than the base size is a setup error. The 1 MiB input is the valid
  byte-boundary neighbor; 1 MiB + 1 refuses before parsing. Whitespace changes no semantic card fields or source pins.

## Independent budget stages

parse-document checks encoding, one-mapping/YAML restrictions and byte/depth budgets, without running closed-field or
relationship validation. The ordinary converted-v4.json is also a valid whole-card neighbor. nested-mapping-depth
constructs N mappings recursively: the root is level 1, each has exactly wrapper_key pointing to the next mapping,
and mapping N contains that key pointing to leaf_utf8. N=32 passes this parser-stage budget; N=33 refuses. These
parser-only mappings intentionally do not claim to be schema-valid product cards.

check-collection-bounds consumes an already decoded complete card-shaped mapping and checks its project/member array
counts only. repeat-array-last-entry keeps all original entries, then deep-copies the original last entry until the
specified count is reached. The valid stage controls have exactly 1,000/10,000 entries; the refusal neighbors have
1,001/10,001. Their repeated ids deliberately do not claim whole-card semantic acceptance. These stage inputs isolate
count guards from byte limits, duplicate-id and reservation checks; executing only the full parser on an oversized
encoding would not demonstrate the count guard. Normal full-card acceptance controls remain required alongside them.

check-superior-hop-budget receives an already authorized acyclic external relationship fixture. Construct nodes 0..N:
node 0 is start, node 1 is first_external, nodes i>=2 have product realm-i and member keeper. Every node before N has
exactly one edge to its successor. Node N is an explicitly unavailable boundary with no supplied outgoing edge. The
provided projection sources are assumed authorized and pinned for this isolated graph-budget operation. N=64 accepts
64 traversed edges and reports that partial boundary; N=65 refuses the complete result. No local worker hierarchy or
missing authorization is used to trigger the refusal; this is not execution of the Pulse caller/admission path.

The full ordinary bounds-tuple-64-v4.json has one additional keeper-managed project and prospective parent-native member,
with the explicit synthetic altair reservation in its vector. Its localpart is exactly 64 characters. The neighboring
card changes only the new project slug and matching member project reference, making that localpart 65; both project
slugs remain under 40 characters. The failure must be localpart overflow, never shortening, a reservation race or a
missing profile. Neither card allocates or registers anything.

Every refusal label denotes its named invariant; it is a future expected behavior, not an existing error API. A later
implementation must execute all selected lanes and the known-violation instrument control. These authored inputs do
not claim any runtime verdict, fault injection, symlink check or live qualification has already passed.
