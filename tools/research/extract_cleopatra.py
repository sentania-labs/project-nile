#!/usr/bin/env python3
"""Extract licensed Cleopatra mission facts without original maps, prose or code."""

import argparse
import json
from pathlib import Path

from extract_campaign import (EVENT_COLUMNS, EVENT_TYPE_INTERPRETATIONS, FormatError,
                              campaign_definition, digest, directory, event_metadata,
                              read, record_chunks, scenario_metadata)
from extract_campaign_economy import extract as extract_economy

GOLD_HASHES = {
    "campaign.txt": "0ef281c3f32af2a46e49f972f66adf0a93ad5832b0e74252e5d33ea1a5a0c740",
    "MISSION1.PAK": "12753b3330c5df49c1b5072324814f879a769e6793562150fafe5b9d34b91ab2",
    "Pharaoh_Text.eng": "d1476f51aaeddb2f91e6ec1f71628b6d5734318f9393e32275fad63e8c26958e",
}
GOLD_EXECUTABLE_SHA256 = "b21b7d719491bb45dfb324ba95231a5b0960ab25fea1bf3fb21da65da7eca662"
EXPANSION_IDS = range(38, 53)
GOLD_LAYOUTS = ((147, 175), (149, 175), (150, 175), (160, 181))


def source_file(directory_path, name):
    matches = [p for p in directory_path.iterdir() if p.name.lower() == name.lower()]
    if len(matches) != 1 or not matches[0].is_file():
        raise FormatError(f"missing or ambiguous source file: {name}")
    return matches[0]


def text_group(data, group):
    if data[:16] != b"Pharaoh textfile":
        raise FormatError("unsupported language file signature")
    start = 8028 + read(data, "I", 28 + group*8)[0]
    end = 8028 + read(data, "I", 28 + (group+1)*8)[0]
    if not 8028 <= start < end <= len(data):
        raise FormatError("invalid language group bounds")
    values = data[start:end].split(b"\0")
    if values[-1] != b"":
        raise FormatError("unterminated language group")
    return [v.decode("cp1252") for v in values[:-1]]


def extract(source_dir, decoder):
    paths = {name: source_file(source_dir, name) for name in GOLD_HASHES}
    files = {name: path.read_bytes() for name, path in paths.items()}
    for name, data in files.items():
        if digest(data) != GOLD_HASHES[name]:
            raise FormatError(f"unverified Gold source: {name}")
    periods, all_missions = campaign_definition(files["campaign.txt"])
    pak = files["MISSION1.PAK"]
    bounds = directory(pak, 60)
    if len(bounds) != 53 or len(all_missions) != 53:
        raise FormatError("expected exactly 53 Gold mission records")
    # Walk every record to verify the framing, even though only expansion facts
    # are exported. No original terrain or other uninterpreted chunks are decoded.
    framing = [record_chunks(pak, *span, layouts=GOLD_LAYOUTS) for span in bounds]
    missions, events = [], []
    for mission_id in EXPANSION_IDS:
        mission = all_missions[mission_id]
        start, end = bounds[mission_id]
        chunks = framing[mission_id]
        version, descriptors = read(pak, "2I", start+4, end)
        if (version, descriptors) != (160, 181):
            raise FormatError("unexpected expansion mission layout")
        scenario = chunks[75]["offset"]
        event = chunks[136]["offset"]
        mission["source_record"] = {
            "offset": start, "bytes": end-start, "sha256": digest(pak[start:end]),
            "format_version": version, "descriptor_count": descriptors,
            "scenario_chunk_offset": scenario, "event_chunk_offset": event,
        }
        mission.update(scenario_metadata(pak, scenario))
        mission["events"], rows = event_metadata(pak, event)
        missions.append(mission)
        events.append({"mission_id": mission_id, "records": rows})
    labels = text_group(files["Pharaoh_Text.eng"], 67)
    if len(labels) != 46 or labels[-1] != "Zoo":
        raise FormatError("unexpected Cleopatra editor labels")
    result = {
        "schema_version": 1,
        "edition": "Pharaoh Gold: Cleopatra campaign, licensed local GOG corpus",
        "sources": {name: {"sha256": digest(data), "bytes": len(data), "matches_verified_corpus": True}
                    for name, data in files.items()},
        "evidence": {
            "numeric_values": "Direct reads of the licensed, hashed Gold files at source_record offsets.",
            "framing": "60 directory entries, 53 missions; all 53 descriptor walks end exactly at the next record or EOF. Expansion IDs38..52 use v160 and181 descriptors.",
            "scenario_fields": "Descriptor75 remains1592 bytes; goal flags, values, resource slots and monument codes retain the base offsets. Values are stored facts, not proof of complete runtime semantics.",
            "events": "Descriptor136 remains18628 bytes: declared signed16 count, capacity150,124-byte records. Only declared records exported.",
            "names": "Only short mission/geographic and structure labels. Briefings, narrative, terrain, routes, original code and assets excluded.",
            "campaign_edges": "Derived from campaign.txt path/period order. Cross-period handoff and starting wealth inheritance need executable validation.",
        },
        "limitations": [
            "Stored objectives do not establish additional executable win, lose or tutorial conditions.",
            "Enemy culture8/9, expansion resources31..35, monument codes and extra chunks require expansion runtime interpretation.",
            "45 editor labels are independent of47 numeric allowance slots; no positional menu join inferred.",
            "Base missions in Gold are not substituted for the separately verified base corpus.",
            "No expansion mission is made playable by this factual extraction.",
        ],
        "scenario_layout": {"descriptor_index": 75, "bytes": 1592, "endianness": "little",
                            "field_offsets_reference": "base-campaign.json:scenario_layout"},
        "event_layout": {"descriptor_index": 136, "bytes": 18628, "record_bytes": 124,
                         "capacity": 150, "declared_count": [0, "i16"],
                         "file": "cleopatra-campaign-events.jsonl",
                         "record_index": "Zero-based position in each mission records array.",
                         "columns": [{"name": name, "offset": at, "struct_type": fmt}
                                     for name, fmt, at in EVENT_COLUMNS]},
        "event_type_interpretations": EVENT_TYPE_INTERPRETATIONS,
        "editor_text_labels": {str(i): label for i, label in enumerate(labels[1:], 1)},
        "resource_text_labels": {str(i): name for i, name in enumerate(text_group(files["Pharaoh_Text.eng"], 23)[:36])},
        "monument_text_labels": {str(i): name.strip() for i, name in enumerate(text_group(files["Pharaoh_Text.eng"], 198))},
        "periods": [p for p in periods if p["mission_ids"][0] in EXPANSION_IDS],
        "missions": missions,
    }
    economy = extract_economy(paths["MISSION1.PAK"], decoder, language_path=paths["Pharaoh_Text.eng"],
                              expected_hashes=GOLD_HASHES,
                              directory_entries=60, mission_ids=EXPANSION_IDS,
                              layouts=GOLD_LAYOUTS, object_bytes=98, resource_max=35)
    economy["edition"] = result["edition"]
    economy["evidence"] = {
        "source": "Direct licensed Gold metadata; base executable addresses do not verify expansion runtime behavior.",
        "layout": "Descriptor50:61 city records of106bytes; descriptor114:20x36x4 quota bytes; descriptor142:200x98 Empire objects. City object ID at86 joins matching object city/name/route fields24..26 for every exported city.",
        "resource_flags": "36 boolean buy/sell slots at6/42; latent Empire-object resource IDs at30..43 and52..59. Retain expansion resource IDs31..35.",
        "quotas": "Original stored loads multiplied by100, as the existing base trade representation. Exported city/route joins include dormant partners.",
        "army_travel": "Descriptor143 retains50x324 route records. Only derived months max(1,length_i32_at308*header_byte0/400) exported; expansion execution equivalence needs verification.",
        "names": "Geographic names only from original English group195.",
        "limits": "No original map coordinates, route points, images, narrative or executable code exported.",
    }
    return result, events, economy


def write_outputs(result, events, economy, output_dir):
    output_dir.mkdir(parents=True, exist_ok=True)
    for name, value in (("cleopatra-campaign.json", result), ("cleopatra-campaign-economy.json", economy)):
        (output_dir/name).write_text(json.dumps(value, indent=2, ensure_ascii=False)+"\n", encoding="utf-8")
    (output_dir/"cleopatra-campaign-events.jsonl").write_text(
        "".join(json.dumps(row, separators=(",", ":"))+"\n" for row in events), encoding="utf-8")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source-dir", type=Path, required=True)
    parser.add_argument("--decoder", type=Path, required=True)
    parser.add_argument("--output-dir", type=Path, required=True)
    args = parser.parse_args()
    try:
        result, events, economy = extract(args.source_dir, args.decoder)
        write_outputs(result, events, economy, args.output_dir)
    except (FormatError, OSError, UnicodeError, ValueError) as error:
        parser.exit(1, f"Cleopatra extraction failed: {error}\n")
    print(f"Extracted {len(result['missions'])} Cleopatra missions, {sum(len(e['records']) for e in events)} declared events and {sum(len(m['cities']) for m in economy['missions'])} city records.")


if __name__ == "__main__":
    main()
