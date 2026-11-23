#!/usr/bin/env python3
"""Read original numeric resource/trade capabilities, excluding maps and prose.

The optional decoder is zlib contrib/blast's standalone TEST executable, a
public PKWARE DCL decoder. It reads only the bounded 6466-byte city directory.
No original executable code is used, linked, generated, or exported.
"""
import argparse
import json
from pathlib import Path
import subprocess

from extract_campaign import FormatError, KNOWN_HASHES, digest, directory, read, record_chunks


def extract(source, decoder):
    data = source.read_bytes()
    if digest(data) != KNOWN_HASHES["MISSION1.PAK"]:
        raise FormatError("unverified MISSION1.PAK corpus")
    language = source.with_name("Pharaoh_Text.eng").read_bytes()
    if digest(language) != KNOWN_HASHES["Pharaoh_Text.eng"]:
        raise FormatError("unverified original city names")
    name_start = 8028 + read(language, "I", 28 + 195 * 8)[0]
    name_end = 8028 + read(language, "I", 28 + 196 * 8)[0]
    names = [name.decode("cp1252") for name in language[name_start:name_end].split(b"\0")]
    def decode(chunk):
        payload = data[chunk["offset"]:chunk["offset"] + chunk["stored_size"]]
        if chunk["encoded"]:
            result = subprocess.run([str(decoder)], input=payload, capture_output=True, check=True, timeout=10)
            payload = result.stdout
            if result.stderr:
                raise FormatError("decoder did not consume the entire directory")
        if len(payload) != chunk["unpacked_size"]:
            raise FormatError("directory decompression length mismatch")
        return payload
    missions = []
    for mission_id, bounds in enumerate(directory(data)):
        chunks = record_chunks(data, *bounds)
        chunk = chunks[50]
        if chunk["unpacked_size"] != 6466:
            raise FormatError("unsupported city directory layout")
        payload = decode(chunk)
        if chunks[114]["unpacked_size"] != 2880:
            raise FormatError("unsupported trade quota layout")
        quotas = decode(chunks[114])
        cities = []
        for city_id in range(61):
            offset = city_id * 106
            row = payload[offset:offset + 106]
            if not row[0] or row[2] == 6:
                continue
            if any(flag not in (0, 1) for flag in row[6:78]):
                raise FormatError("non-boolean city resource flag")
            if row[3] >= len(names) or row[4] >= 20:
                raise FormatError("city name or route index out of bounds")
            limits = {str(r): read(quotas, "I", row[4]*144+r*4)[0]*100 for r in range(1,31)
                      if row[2] in (1,3,5) and (row[6+r] or row[42+r])}
            cities.append({"id": city_id, "type_raw": row[2], "name_id_raw": row[3], "name": names[row[3]],
                           "route_id": row[4], "initially_open": bool(row[5]),
                           "buys": [i for i, flag in enumerate(row[6:42]) if flag],
                           "sells": [i for i, flag in enumerate(row[42:78]) if flag],
                           "cost_to_open": read(row, "h", 78)[0], "sea_trade": bool(row[88]), "annual_limits": limits})
        missions.append({"id": mission_id, "cities": cities,
                         "source_chunk": {"descriptor": 50, "offset": chunk["offset"], "stored_bytes": chunk["stored_size"], "decoded_sha256": digest(payload)}})
    return {"schema_version": 1,
            "source_sha256": digest(data),
            "decoder": {"source": "https://github.com/madler/zlib/tree/v1.3.1/contrib/blast", "purpose": "PKWARE DCL container decompression only"},
            "evidence": {"resource_flags": "Original 0x00437810 reads active type0 city sells flags for local resources; 0x00437640 reads type1/3/5 partner sells flags for imports.",
                         "layout": "Descriptor50, 61 records of106bytes; original city base0x006f0c78 and resource base0x006f0ca2 corroborate the public format.",
                         "quotas": "Descriptor114, 20 routes x36resources x4bytes; stored loads converted to100-unit loads. Original0x0043b880 indexes route*36+resource at0x006e92e8.",
                         "names": "Only geographic city names from original English group195; no briefings or prose.",
                         "limits": "No map coordinates, routes, images, original prose or executable code. Travel distances and god bonuses remain separate mechanics."},
            "missions": missions}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--decoder", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    result = extract(args.source, args.decoder)
    args.output.write_text(json.dumps(result, indent=2) + "\n")
    print(f"Extracted numeric economic capabilities for {len(result['missions'])} original missions.")
