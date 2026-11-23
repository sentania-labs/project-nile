#!/usr/bin/env python3
"""Read original numeric resource/trade capabilities, excluding maps and prose.

The optional decoder is zlib contrib/blast's standalone TEST executable, a
public PKWARE DCL decoder. It decodes bounded city/resource metadata and exports no map geometry.
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
        if chunks[142]["unpacked_size"] != 15200:
            raise FormatError("unsupported empire object layout")
        objects = decode(chunks[142])
        if chunks[143]["unpacked_size"] != 16200:
            raise FormatError("unsupported empire route layout")
        routes = decode(chunks[143])
        own = [i for i in range(200) if objects[i*76] == 1 and objects[i*76+1] and objects[i*76+24] == 0]
        if len(own) != 1:
            raise FormatError("expected exactly one original player city")
        cities = []
        for city_id in range(61):
            offset = city_id * 106
            row = payload[offset:offset + 106]
            if not row[0]:
                continue
            if any(flag not in (0, 1) for flag in row[6:78]):
                raise FormatError("non-boolean city resource flag")
            if row[3] >= len(names) or row[4] >= 20:
                raise FormatError("city name or route index out of bounds")
            object_id = read(row, "h", 86)[0]
            if not 0 <= object_id < 200:
                raise FormatError("invalid city object reference")
            obj = objects[object_id*76:(object_id+1)*76]
            if obj[0] != 1 or not obj[1] or obj[25] != row[3] or obj[26] != row[4]:
                raise FormatError("city and empire object identity disagree")
            potential_sells = sorted(set(r for r in obj[30:44] if 0 < r <= 30))
            potential_buys = sorted(set(r for r in obj[52:60] if 0 < r <= 30))
            limits = {str(r): read(quotas, "I", row[4]*144+r*4)[0]*100
                      for r in sorted(set(potential_sells+potential_buys))}
            travel_months = 0
            for route_id in range(50):
                route = routes[route_id*324:(route_id+1)*324]
                ends = read(route, "2h", 316)
                if route[322] and route[321] >= 2 and own[0] in ends and object_id in ends:
                    length = read(route, "i", 308)[0]
                    if length < 0:
                        raise FormatError("negative empire route length")
                    travel_months = max(1, length*route[0]//400)
                    break
            cities.append({"id": city_id, "type_raw": row[2], "name_id_raw": row[3], "name": names[row[3]],
                           "route_id": row[4], "empire_object_id": object_id, "army_travel_months": travel_months,
                           "potential_buys": potential_buys, "potential_sells": potential_sells, "initially_open": bool(row[5]),
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
                         "political_routes": "Original0x00441c50 joins event Location to city route via0x004385a0. Latent sells/buys at Empire-object offsets30/52 are confirmed by0x00436d20/50; object identity comes from city offset86. No object coordinates or images are exported.",
                         "army_travel": "Original0x0043fd60→0x0043b520 joins player/city Empire object IDs to active route endpoints. Descriptor143 has50×324byte records. Travel=max(1,length_i32_at308*header_byte0/400), confirmed0x0043b8c0. Only derived month counts exported, no points or coordinates.",
                         "names": "Only geographic city names from original English group195; no briefings or prose.",
                         "limits": "No map coordinates, routes, images, original prose or executable code. God bonuses remain separate mechanics."},
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
