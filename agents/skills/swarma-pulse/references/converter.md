# Explicit product-card conversion

`card-v4-convert` converts one supplied v3.1 source to a separately requested v4 destination. `internal/cardv4.Convert`
performs the pure transformation and returns a complete deterministic JSON document, also valid YAML, or no bytes.
`internal/cardpublish` implements only destination publication. The command has no network, child-process, account,
registration, credential, room, hook, session or migration API. It never discovers or replaces a live card automatically.
Use only explicitly admitted source, destination and premises; fixture reservations are synthetic examples.

## Inputs and preservation

The command requires `--root`, `--source`, `--inputs` and `--destination`. Source and destination paths are relative to
an explicitly admitted physical root. The JSON inputs use the `ConversionInputs` contract:

- `source_revision`, `source_sha256`, `host_project`, `source_card_root`, `target_card_root` and `admitted_root` are
  required reviewed coordinates and source authority. The source digest covers its original bytes, before alias expansion.
- `external_parent_mapping` maps every old external id to its exact qualified product/member reference.
- `additional_connections`, `additional_bindings`, `prospective_nonchat_choices` and `reviewed_reservation_premises`
  supply any additional declarations and prospective nonchat identities. Missing choices or reservations refuse.
- Optional `project_locations` can confirm equivalent local placement. Changed local or remote disposition refuses;
  a future lossless disposition contract requires its own oracle. Existing remote v4 cards remain valid under the validator.

Bases are admitted canonical coordinates, not discovered filesystem locations. Their common `admitted_root` can be a
logical prefix, such as `fixture-root`, which `--root` maps to the admitted physical tree. The CLI verifies each source/
destination parent coordinate against the corresponding base. All references are rebased to the same declared targets;
the command does not open project locators, task files, profile repositories or clone directories to invent missing facts.
A caller admitting coordinates remains responsible for their canonical meaning and exclusive custody during conversion.

Every existing chat member becomes `legacy-exact`, independently derived from the pinned source binding/prefix/id/server.
Default bindings become explicit. Nonchat tuples come only from supplied reviewed choices. Unsupported source extensions
refuse; logical ids remain exact, including opaque legacy keys. Product-level members name the reviewed host project.
Station, runtime/profile/locks, staffing, desired states, all projects and required/self selector order/multiplicity remain
lossless. V3 aliases/merges expand before strict v4 emission. Conversion validates the source, checks its digest, constructs
the complete target, validates it against actual source bytes/reservations, reparses the serialized bytes and runs the
separate preservation checker. Missing premises, an incompatible v4 representation or a false pin returns no document.

The output has sorted JSON mapping keys and two-space indentation, followed by one newline. Sequence order and string
values are preserved. Complete decoded fixture comparisons and canonical byte comparisons establish the expected output;
the authored fixture files themselves are unchanged. Fixture expectation files also contain oracle metadata, which is not
an operational input. Conversion does not execute a per-member cutover or rollback. Its expected complete prior document
and pilot/rollback preservation comparisons remain independently tested.

Operational premise keys use the exact documented spelling. Case-folded variants, including an alias beside its canonical
key, refuse before conversion. Each reservation record requires exactly `holder`, `domain`, `product`, `primary_role` and
`star`, with nonempty string values; unknown or differently cased keys refuse. Other top-level fixture observation fields
remain ignored and cannot select operational values.

## Publication and failure results

The publisher opens and retains the admitted destination directory through native `os.Root`, using the patched toolchain
required by go.mod. It rejects source inode aliases, destination symlinks/types and escaping directory paths. It creates
`.card-v4-convert-<destination-basename>.tmp` with mode 0600 and exclusive creation, writes complete validated bytes,
checks the full write, fsyncs the file, closes it, atomically renames it over the requested destination, then fsyncs the
pinned destination directory. Rename is the publication commit point. Only the complete destination replacement is allowed.
Previously opened readers retain the prior inode. The command prints only `converted` on successful durable publication.

A reported pre-rename failure preserves source and prior destination; the owned temp is removed. If cleanup itself fails,
`publication_cleanup_failed` explicitly reports that a temp remains. A later invocation refuses any existing temp with
`publication_leftover_temp`, including a symlink, and never reads or adopts it. Termination before rename can leave that
same temp; after inspecting the isolated operation, an authorized caller must remove it explicitly before retrying.
Termination after rename cannot be represented as an unchanged destination.

A directory-fsync failure after rename returns `Published=true, Durable=false` from the adapter. The CLI exits 3 with
`publication_complete_durability_unconfirmed`: the complete destination is already published, its crash durability is
unconfirmed, and blind retry/rollback is inappropriate. The successful path completes both fsyncs; storage hardware and
filesystem guarantees remain external. This does not promise recovery from arbitrary hardware loss or uncatchable termination.

Rule refusals exit 1; usage/read/encoding errors exit 2. Diagnostics contain only fixed rule and safe indexed location,
without input values, filenames, source fragments or backtraces; no card bytes are printed. A failure to deliver a receipt
does not reverse a completed destination publication. The adapter is generic and requires already validated complete bytes.

## Checks and limits

All 37 accepted vectors execute, including actual member-conversion interruption after two members, complete native
publication with zero registration attempts, and an inert synthetic registration-tripwire control. Native write, short-write,
file-sync, close, rename, directory-sync and cleanup failures have explicit result/state assertions and valid neighbors.
Prior-run temps, aliasing, escaping and trailing-separator paths are isolated fixtures. CLI tests compare the full expected
output and exact source/destination/inventory effects. The whole prior rollback document and non-pilot preservation remain
separate oracles; tests do not perform live rollback.

Fault interfaces/function values are unexported; public entry points always select nil/native operations. There is no
fault flag, environment switch or build tag. Source admission rejects process/network/registration APIs and fault activation,
with known-violation controls. The pure library has its existing stricter effects check; filesystem writes occur only in
the publisher. These are bounded source/API/inventory/captured-output instruments, not an OS sandbox, syscall trace or
proof about arbitrary future dependencies. The registration control has no endpoint or transport. Actual service-session
refusal/conversational launch qualification and Linux execution remain external.
