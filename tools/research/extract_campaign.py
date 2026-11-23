#!/usr/bin/env python3
"""Extract factual campaign metadata; never export original map/image/text payloads.

Independent Python implementation of the documented, bounded v147/v149 layout.
See README.md for provenance, interpretation limits, and reproduction commands.
"""

import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path
import struct


REFERENCE_COMMIT = "e8c58b147a00e71a894ac6de1f33885efb0f8b16"
REFERENCE_ROOT = f"https://github.com/dalerank/Akhenaten/blob/{REFERENCE_COMMIT}/"
KNOWN_HASHES = {
    "campaign.txt": "28df4bb13015ec8ed19e1b8a004547dbfc60f301dff10866f338fcdc051856f2",
    "MISSION1.PAK": "42fc2104f66bf7a5b23dada1830a40bbe4ef0582d8d9c1cf3de4f6e5bc725b19",
    "Pharaoh_Text.eng": "d060eec7c23caab444602d198de89ad65335feff3f71f35bc8319d333aa11580",
}
GOALS = ("culture", "prosperity", "monuments", "kingdom", "housing_count", "housing_level_index")
EVENT_COLUMNS = (
    ("event_id", "h", 4), ("type_code", "B", 6), ("month_raw", "B", 7),
    ("item_value_raw", "h", 8), ("item_fixed_raw", "h", 10),
    ("item_min_raw", "h", 12), ("item_max_raw", "h", 14),
    ("amount_value_raw", "h", 16), ("amount_fixed_raw", "h", 18),
    ("amount_min_raw", "h", 20), ("amount_max_raw", "h", 22),
    ("year_raw", "h", 24), ("time_aux_1_raw", "h", 28), ("time_aux_2_raw", "h", 30),
    ("on_completed_action_raw", "h", 40), ("on_refusal_action_raw", "h", 42),
    ("trigger_code", "B", 44), ("tag_raw", "H", 46), ("months_initial_raw", "B", 48),
    ("festival_deity_raw", "b", 58), ("invasion_target_raw", "b", 60),
    ("parameter_raw", "i", 82), ("on_too_late_action_raw", "h", 86),
    ("on_defeat_action_raw", "h", 88), ("sender_faction_raw", "b", 90),
    ("subtype_raw", "b", 100), ("city_id_raw", "b", 101),
)
EVENT_TYPE_INTERPRETATIONS = {
    0: "none", 1: "request", 2: "invasion", 6: "sea_trade_problem", 7: "land_trade_problem",
    8: "wage_increase", 9: "wage_decrease", 10: "contaminated_water", 11: "gold_mine_collapse",
    12: "clay_pit_flood", 13: "demand_increase", 14: "demand_decrease", 15: "price_increase",
    16: "price_decrease", 17: "reputation_increase", 18: "reputation_decrease",
    19: "city_status_change", 20: "message", 21: "failed_flood", 22: "perfect_flood",
    23: "gift_from_pharaoh", 24: "locusts", 25: "frogs", 26: "hailstorm", 27: "blood_river",
    28: "crime_wave", 29: "mummy", 30: "foreign_army_attack_warning",
    31: "distant_battle", 32: "distant_battle_won",
}


class FormatError(ValueError):
    """Unsupported or inconsistent source data. No output should be promoted."""


def digest(data):
    return hashlib.sha256(data).hexdigest()


def read(data, fmt, offset, end=None):
    limit = len(data) if end is None else min(end, len(data))
    size = struct.calcsize("<" + fmt)
    if offset < 0 or offset + size > limit:
        raise FormatError(f"truncated {fmt} field at byte {offset}")
    return struct.unpack_from("<" + fmt, data, offset)


def directory(data):
    entries = read(data, "50I", 0)
    starts = []
    ended = False
    for offset in entries:
        if offset == 0:
            ended = True
        elif ended or offset < 200 or offset >= len(data) or (starts and offset <= starts[-1]):
            raise FormatError("invalid PAK directory order, gap, or bounds")
        else:
            starts.append(offset)
    if not starts or starts[0] != 200:
        raise FormatError("unsupported PAK directory/header size")
    return list(zip(starts, starts[1:] + [len(data)]))


def record_chunks(data, start, end):
    """Walk descriptor lengths only. Never execute stored pointers or decompress."""
    version, count = read(data, "II", start + 4, end)
    if version not in (147, 149) or count != 175:
        raise FormatError(f"unsupported scenario layout version={version}, descriptors={count}")
    read(data, "6000s", start + 12, end)
    cursor = start
    chunks = []
    for index in range(count):
        compressed, _, unit, units, _ = read(data, "5I", start + 12 + 20 * index, end)
        if compressed not in (0, 1):
            raise FormatError(f"invalid compression flag at descriptor {index}")
        unpacked_size = unit * units
        # Observed map/state chunks are bounded well below this. Reject huge descriptors.
        if unpacked_size > 64 * 1024 * 1024:
            raise FormatError(f"unreasonable chunk size at descriptor {index}")
        stored_size = unpacked_size
        encoded = False
        if compressed:
            prefix, = read(data, "I", cursor, end)
            cursor += 4
            encoded = prefix != 0x80000000
            stored_size = prefix if encoded else unpacked_size
        if cursor + stored_size > end:
            raise FormatError(f"descriptor {index} crosses its mission boundary")
        chunks.append({"offset": cursor, "stored_size": stored_size,
                       "unpacked_size": unpacked_size, "encoded": encoded})
        cursor += stored_size
    if cursor != end:
        raise FormatError(f"mission descriptors end at {cursor}, expected {end}")
    for index, expected in ((75, 1592), (136, 18628)):
        if chunks[index]["unpacked_size"] != expected or chunks[index]["encoded"]:
            raise FormatError(f"unsupported metadata chunk {index}")
    return chunks


def campaign_definition(data):
    names, periods, missions = [], [], []
    section = None
    current = None
    choice_target = None
    for line_number, original in enumerate(data.decode("cp1252").splitlines(), 1):
        line = original.split(";", 1)[0].strip()
        if not line:
            continue
        if line.startswith("[") and line.endswith("]"):
            section = line[1:-1]
            current = None
            choice_target = None
            if section != "MISSION_NAMES":
                periods.append({"id": section.lower(), "mission_ids": [], "initial_path_choices": []})
        elif section == "MISSION_NAMES":
            # Geographic names only, not the briefing or mission narrative.
            name, separator, aliases = line.partition(" (")
            names.append({"name": name, "aliases": [aliases.rstrip(")")] if separator else []})
        elif line.startswith("mission="):
            fields = [int(x.strip()) for x in line.split("=", 1)[1].split(",")]
            if not periods or len(fields) != 4:
                raise FormatError("unsupported campaign mission/merge layout")
            mission_id, _, _, path_id = fields
            if mission_id < 0 or mission_id >= len(names) or any(m["id"] == mission_id for m in missions):
                raise FormatError("duplicate or out-of-range campaign mission id")
            current = {"id": mission_id, **names[mission_id], "period": periods[-1]["id"],
                       "path_id": path_id, "campaign_line": line_number, "choice_after_path_ids": []}
            missions.append(current)
            periods[-1]["mission_ids"].append(mission_id)
            choice_target = None
        elif line.startswith("choicescreen="):
            choice_target = current["choice_after_path_ids"] if current else periods[-1]["initial_path_choices"]
        elif line.startswith("choice="):
            if choice_target is None:
                raise FormatError("campaign choice without choice screen")
            choice_target.append(int(line.split("=", 1)[1].split(",")[0]))
        else:
            raise FormatError(f"unsupported campaign directive on line {line_number}")
    if len(missions) != len(names) or {m["id"] for m in missions} != set(range(len(names))):
        raise FormatError("campaign names and mission definitions do not cover the same IDs")
    by_id = {m["id"]: m for m in missions}
    for period_index, period in enumerate(periods):
        members = [by_id[i] for i in period["mission_ids"]]

        def resolve(paths, candidates):
            result = []
            for path in paths:
                found = next((m["id"] for m in candidates if m["path_id"] == path), None)
                if found is None:
                    raise FormatError(f"unresolved campaign path {path}")
                if found not in result:
                    result.append(found)
            return result

        period["entry_mission_ids"] = resolve(period["initial_path_choices"] or [0], members)
        for index, mission in enumerate(members):
            candidates = members[index + 1:]
            if mission["choice_after_path_ids"]:
                mission["next_mission_ids"] = resolve(mission["choice_after_path_ids"], candidates)
            else:
                following = next((m["id"] for m in candidates if m["path_id"] in (0, mission["path_id"])), None)
                if following is not None:
                    mission["next_mission_ids"] = [following]
                elif period_index + 1 < len(periods):
                    next_period = periods[period_index + 1]
                    mission["next_mission_ids"] = resolve(next_period["initial_path_choices"] or [0],
                                                         [by_id[i] for i in next_period["mission_ids"]])
                else:
                    mission["next_mission_ids"] = []
    return periods, sorted(missions, key=lambda m: m["id"])


def editor_labels(data):
    if data[:16] != b"Pharaoh textfile":
        raise FormatError("unsupported language file signature")
    # Observed language table: 28-byte header, 1000 offset/metadata pairs, text at 8028.
    start = 8028 + read(data, "I", 28 + 67 * 8)[0]
    end = 8028 + read(data, "I", 28 + 68 * 8)[0]
    if not 8028 <= start < end <= len(data):
        raise FormatError("invalid editor-label group bounds")
    labels = data[start:end].split(b"\0")
    if len(labels) != 48 or labels[-1] != b"":
        raise FormatError("unsupported original English editor-label group")
    # Skip the UI heading. Export short structure/category names only.
    return {str(index): label.decode("cp1252") for index, label in enumerate(labels[1:-1], 1)}


def scenario_metadata(data, offset):
    def value(fmt, at):
        return read(data, fmt, offset + at, offset + 1592)[0]

    goals = {name: {"value": value("i", 844 + i * 4), "enabled_raw": value("B", 868 + i)}
             for i, name in enumerate(GOALS)}
    for name, at in (("time_limit_years", 880), ("survival_years", 888), ("population", 896)):
        goals[name] = {"value": value("i", at + 4), "enabled_raw": value("i", at)}
    width, height = value("i", 44), value("i", 48)
    if not (1 <= width <= 228 and 1 <= height <= 228):
        raise FormatError("invalid original map dimensions")
    return {
        "start_year_signed": value("h", 0), "initial_funds": value("i", 32),
        "map": {"width_tiles": width, "height_tiles": height, "terrain_exported": False},
        "goals": goals,
        "empire_id_raw": value("h", 4), "enemy_id_raw": value("h", 36),
        "player_rank_raw": value("h", 650),
        "gods_known_raw": [value("B", 10 + i * 2) for i in range(5)],
        "rescue_loan_raw": value("i", 956),
        "milestone_years_raw": [value("i", 960 + i * 4) for i in range(3)],
        "editor_allow_flags_raw": list(read(data, "47h", offset + 1032, offset + 1592)),
        "monument_codes_raw": list(read(data, "3H", offset + 1288, offset + 1592)),
        "burial_requirements_raw": {str(i): value("I", 1296 + i * 4) for i in range(36)
                                    if value("I", 1296 + i * 4) != 0},
    }


def event_metadata(data, offset):
    count, = read(data, "h", offset, offset + 18628)
    if not 0 <= count <= 150:
        raise FormatError(f"unsupported declared event count {count}")
    # Count zero does not imply a zero-filled allocation. Ignore stale unused records.
    records = [[read(data, fmt, offset + index * 124 + field_offset, offset + 18628)[0]
                for _, fmt, field_offset in EVENT_COLUMNS] for index in range(count)]
    types = Counter(record[1] for record in records)
    summary = {"declared_count": count, "type_code_counts": {str(k): types[k] for k in sorted(types)}}
    return summary, records


def extract(source_dir, allow_unverified_source=False):
    files = {name: (source_dir / name).read_bytes() for name in KNOWN_HASHES}
    sources = {name: {"sha256": digest(data), "bytes": len(data),
                      "matches_verified_corpus": digest(data) == KNOWN_HASHES[name]}
               for name, data in files.items()}
    if not allow_unverified_source and not all(source["matches_verified_corpus"] for source in sources.values()):
        raise FormatError("source differs from the verified corpus; research with --allow-unverified-source explicitly")
    periods, missions = campaign_definition(files["campaign.txt"])
    pak = files["MISSION1.PAK"]
    ranges = directory(pak)
    if len(ranges) != len(missions):
        raise FormatError("PAK mission count does not match campaign definition")
    events = []
    for mission in missions:
        start, end = ranges[mission["id"]]
        chunks = record_chunks(pak, start, end)
        scenario_offset = chunks[75]["offset"]
        event_offset = chunks[136]["offset"]
        mission["source_record"] = {
            "offset": start, "bytes": end - start, "sha256": digest(pak[start:end]),
            "format_version": read(pak, "I", start + 4, end)[0], "descriptor_count": 175,
            "scenario_chunk_offset": scenario_offset, "event_chunk_offset": event_offset,
        }
        mission.update(scenario_metadata(pak, scenario_offset))
        mission["events"], rows = event_metadata(pak, event_offset)
        events.append({"mission_id": mission["id"], "records": rows})
    result = {
        "schema_version": 1,
        "edition": "Pharaoh base campaign, local original corpus; Cleopatra not covered",
        "sources": sources,
        "evidence": {
            "numeric_values": "Read directly from the hashed original files at the documented offsets.",
            "field_meanings": "Public format reconstruction, checked against all 38 record boundaries; not all runtime semantics verified.",
            "campaign_edges": "Derived from campaign.txt path/merge rules; not a claim of played-through verification.",
            "event_type_names": "Secondary format interpretations; unknown codes remain numeric.",
            "editor_labels": "Original English language group 67, independent from the numeric flags.",
            "editor_flag_mapping": "Only slots 28 Bridge and 29 Ferry Landing have specific secondary corroboration; other label/flag joins are provisional.",
        },
        "limitations": [
            "Scenario goals are stored criteria, not the complete executable tutorial/completion logic.",
            "Editor flags are not effective build menus. Tutorial and engine unlock rules remain unverified.",
            "47 numeric editor slots but only 46 original labels: slot 47 is unlabeled; no label invented.",
            "Event timing, recurrence, resource units, consequence links and code meanings need original-engine checks.",
            "Declared event count zero exports no events, even when unused allocation contains nonzero bytes.",
            "Original map terrain, object placement, images, prose, executable code and pointer values are excluded.",
            "No Cleopatra campaign files have been verified by this extraction.",
        ],
        "format_references": [REFERENCE_ROOT + path for path in (
            "src/io/chunk_serializer.cpp", "src/io/gamestate/file_schema_sav.cpp",
            "src/scenario/scenario.cpp", "src/scenario/scenario_event_manager.cpp",
            "src/scenario/scenario_event_manager.h", "docs/bridge_editor_allow_triage.md")],
        "scenario_layout": {
            "descriptor_index": 75, "bytes": 1592, "endianness": "little",
            "fields": {"start_year_signed": [0, "i16"], "empire_id_raw": [4, "i16"],
                       "gods_known_raw": [10, "5 x u8, stride 2"], "initial_funds": [32, "i32"],
                       "enemy_id_raw": [36, "i16"], "map_dimensions": [44, "2 x i32"],
                       "player_rank_raw": [650, "i16"], "goal_values": [844, "6 x i32"],
                       "goal_enabled": [868, "6 x u8"], "time_limit_enabled_and_years": [880, "2 x i32"],
                       "survival_enabled_and_years": [888, "2 x i32"],
                       "population_enabled_and_goal": [896, "2 x i32"], "rescue_loan_raw": [956, "i32"],
                       "milestone_years_raw": [960, "3 x i32"], "editor_allow_flags_raw": [1032, "47 x i16"],
                       "monument_codes_raw": [1288, "3 x u16"], "burial_requirements_raw": [1296, "36 x u32"]},
            "goal_order": list(GOALS),
        },
        "event_layout": {"descriptor_index": 136, "bytes": 18628, "record_bytes": 124,
                         "capacity": 150, "declared_count": [0, "i16"],
                         "file": "base-campaign-events.jsonl",
                         "record_index": "Zero-based position in each mission records array.",
                         "columns": [{"name": name, "offset": at, "struct_type": fmt}
                                     for name, fmt, at in EVENT_COLUMNS]},
        "event_type_interpretations": EVENT_TYPE_INTERPRETATIONS,
        "editor_text_labels": editor_labels(files["Pharaoh_Text.eng"]),
        "periods": periods,
        "missions": missions,
    }
    return result, events


def write_outputs(result, events, output_dir):
    output_dir.mkdir(parents=True, exist_ok=True)
    (output_dir / "base-campaign.json").write_text(json.dumps(result, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    # One mission per line; the columns live once in base-campaign.json.
    (output_dir / "base-campaign-events.jsonl").write_text(
        "".join(json.dumps(row, separators=(",", ":")) + "\n" for row in events), encoding="utf-8")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source-dir", type=Path, required=True, help="local licensed original installation")
    parser.add_argument("--output-dir", type=Path, required=True)
    parser.add_argument("--allow-unverified-source", action="store_true",
                        help="research a different corpus; all provenance remains labeled by its actual hash")
    args = parser.parse_args()
    try:
        result, events = extract(args.source_dir, args.allow_unverified_source)
        write_outputs(result, events, args.output_dir)
    except (FormatError, OSError, UnicodeError, ValueError) as error:
        parser.exit(1, f"campaign extraction failed: {error}\n")
    print(f"Extracted {len(result['missions'])} missions, {sum(len(m['records']) for m in events)} declared event records.")
    print("Original map payloads, prose, images and executable code were not exported.")


if __name__ == "__main__":
    main()
