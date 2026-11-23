# Original campaign facts

`extract_campaign.py` independently reads the installed base game's
`campaign.txt`, `MISSION1.PAK` and `Pharaoh_Text.eng`. It produces only numeric
scenario/event facts, short geographic/building names, branch relationships,
source hashes and byte locators. It does not export or decompress original maps,
object placement, artwork, mission prose or executable code. No third-party
implementation is copied into this tool. Public reconstruction work supplies
field-layout hypotheses; direct reads of the original files supply values.

The verified local source is
`/mnt/holding/project-nile/original-wine-VwPQ/Executable/`. These source files stay
outside Git. The source fingerprints are recorded in the output and enforced by
default. `--allow-unverified-source` permits investigating a different local
corpus and records its actual hashes; it does not confer verified status on it.
Other layouts fail instead of being silently treated as this edition.

```sh
python3 tools/research/extract_campaign.py \
  --source-dir /mnt/holding/project-nile/original-wine-VwPQ/Executable \
  --output-dir internal/campaign/data

python3 -m unittest discover -s tools/research -p 'test_*.py' -v

NILE_ORIGINAL_DIR=/mnt/holding/project-nile/original-wine-VwPQ/Executable \
  python3 -m unittest discover -s tools/research -p 'test_*.py' -v
```

The last command also reads the original corpus, checks representative missions,
walks the campaign graph and reproduces the checked-in files byte for byte.
The synthetic tests require no original game files and cover malformed
directories, unsupported versions, descriptor overflow, truncation, compressed
length/sentinel handling, stale event allocation and branch merges.

## Output contract

- `internal/campaign/data/base-campaign.json`: 38 missions in original ID order,
  five periods, derived successor IDs, signed start years, starting funds, map
  dimensions, enabled and disabled goal values, raw allowance flags, monument
  codes, burial requirements, event counts, sources and field offsets.
- `internal/campaign/data/base-campaign-events.jsonl`: one JSON object per
  mission with `mission_id` and `records`. Each record is a numeric array whose
  columns are defined by `event_layout.columns` in the main file. Its position
  is the source event's zero-based index. There are 1,351 declared records.

Disabled criteria retain their stored values. Consumers must check `enabled_raw`;
a nonzero goal does not automatically mean it is required. `*_raw` fields retain
source codes without asserting execution behavior. Map dimensions describe the
source mission, not a reusable or exported original map.

This is a research dataset. It establishes stored configuration, not all original
runtime rules, tutorial steps, events or completion behavior. The source campaign
defines alternative branches; ID order is not a single playthrough or a strictly
chronological history. New Kingdom ends at either mission 36 or 37.

## Format evidence

The PAK starts with 50 little-endian 32-bit offsets. This corpus has 38 nonzero,
ascending entries followed by 12 zeros. Every mission contains a 12-byte header
and 300 reserved descriptors of 20 bytes; 175 descriptors are active. Versions
147 and 149 both occur. A descriptor supplies compression, unit size and count,
along with original-process addresses that are ignored and never exported.
Descriptors cover the header itself as well as the following chunks.

Compressed chunks have a 32-bit stored-length prefix. `0x80000000` means the
following bytes are uncompressed and use the descriptor's original length.
Walking these lengths lands exactly at the next mission offset or EOF for all
38 records. Descriptor 75 contains 1,592 uncompressed scenario bytes; descriptor
136 contains 18,628 uncompressed event bytes. The event allocation holds 150
124-byte records and 28 remaining bytes. The first signed 16-bit value declares
the count. A zero count exports zero records even if unused bytes are nonzero.
Unused records and the trailing bytes are not interpreted or exported.

The byte offsets in the output are independently readable evidence locators.
Semantic interpretations were compared with the public Akhenaten reconstruction
at commit `e8c58b147a00e71a894ac6de1f33885efb0f8b16`:

- [Chunk framing](https://github.com/dalerank/Akhenaten/blob/e8c58b147a00e71a894ac6de1f33885efb0f8b16/src/io/chunk_serializer.cpp)
  and [save chunk ordering](https://github.com/dalerank/Akhenaten/blob/e8c58b147a00e71a894ac6de1f33885efb0f8b16/src/io/gamestate/file_schema_sav.cpp).
- [Scenario field layout](https://github.com/dalerank/Akhenaten/blob/e8c58b147a00e71a894ac6de1f33885efb0f8b16/src/scenario/scenario.cpp).
- [Event field layout](https://github.com/dalerank/Akhenaten/blob/e8c58b147a00e71a894ac6de1f33885efb0f8b16/src/scenario/scenario_event_manager.cpp)
  and [event code interpretations](https://github.com/dalerank/Akhenaten/blob/e8c58b147a00e71a894ac6de1f33885efb0f8b16/src/scenario/scenario_event_manager.h).
- [Bridge/ferry editor-slot investigation](https://github.com/dalerank/Akhenaten/blob/e8c58b147a00e71a894ac6de1f33885efb0f8b16/docs/bridge_editor_allow_triage.md).

These are secondary evidence for Pharaoh, not the original implementation.
Reconstruction mission scripts can override original values and are not used as
the source of campaign facts. For example, original Thinis stores 3,000 initial
funds, while the inspected reconstruction script sets 2,500.

## Checked samples and remaining work

| Mission | Directly extracted configuration |
|---|---|
| 0 Nubt | Year -3500; 2,500 funds; 84×84; enabled housing count 6, level index 2; zero declared events. |
| 1 Thinis | Year -3100; 3,000 funds; 112×112; enabled housing count 10, level index 5. |
| 2 Perwadjyt | Enabled population 600; housing count 10, level index 6. |
| 6 Behdet | Population 2,500; 22 event records, including four code-1 and four code-2 entries. |
| 10 Saqqara | Population 3,500; monument codes `[9, 0, 0]`; 31 declared events. |
| 18 Rostja | Monument codes `[21, 16, 14]`; 44 declared events. Culture/prosperity values 25 are disabled. |
| 23 Thinis | Population 4,500; 65 declared events. This is distinct from tutorial mission 1. |
| 37 Hetepsenusret | Population 12,000; 69 declared events; a terminal campaign branch. |

Nubt's housing goal is consistent with the original-window observation recorded
in `docs/pharaoh-mechanics.md`: the objective names Meager Shanty, the third tier.
Reading the source bytes and matching that observation does not prove the entire
mission can be completed under Nile's current simulation.

Original language group 67 supplies **46** short labels; the numeric region
contains **47** candidate allowance flags. Both are preserved separately. Bridge
and Ferry Landing have corroborated slot mappings (28/29). Other joins remain
provisional; slot 47 has no label here. Original labels 45/46 are Sphinx/Pyramid,
whereas the reconstruction's later language table differs. Effective menus,
especially tutorial unlocks, require original executable tracing and play checks.

Next verification should record initial UI goals, funds, available buildings and
observed event timing for an original saved mission at a specified difficulty.
Exercise each enabled completion predicate and observe the actual transition;
trace tutorial overrides separately. For events, resolve item/resource units,
relative years and months, randomized ranges, recurrence, favor triggers and
consequence links before implementing their behavior. Monument numeric IDs and
burial resource IDs also need a verified edition-specific mapping. Original
terrain and authored placement remain outside this dataset. Cleopatra requires
its own source installation, fingerprints, formats and mission verification.
