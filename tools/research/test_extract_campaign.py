"""Synthetic corruption tests; optional original-corpus verification via NILE_ORIGINAL_DIR."""

import importlib.util
import json
import os
from pathlib import Path
import struct
import tempfile
import unittest


MODULE = Path(__file__).with_name("extract_campaign.py")
SPEC = importlib.util.spec_from_file_location("extract_campaign", MODULE)
campaign = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(campaign)


def synthetic_pak(compressed=True):
    """Original test data, with one skipped map-like chunk and two metadata chunks."""
    descriptors = [(0, 0, 0, 0, 0)] * 175
    for index in range(4):
        descriptors[index] = (0, 0x12345678, 1, 1, 0)
    descriptors[4] = descriptors[5] = (0, 0, 4, 1, 0)
    descriptors[6] = (0, 0, 20, 300, 0)
    descriptors[7] = (1, 0, 8, 1, 0)
    descriptors[75] = (0, 0, 1592, 1, 0)
    descriptors[136] = (0, 0, 18628, 1, 0)
    schema = b"".join(struct.pack("<5I", *descriptor) for descriptor in descriptors)
    header = struct.pack("<III", 0, 147, 175) + schema + bytes((300 - 175) * 20)
    skipped = struct.pack("<I", 3) + b"ART" if compressed else struct.pack("<I", 0x80000000) + b"ARTDATA!"
    scenario = bytearray(1592)
    struct.pack_into("<h", scenario, 0, -1234)
    struct.pack_into("<i", scenario, 32, 12345)
    struct.pack_into("<2i", scenario, 44, 84, 112)
    narrative = b"SYNTHETIC SOURCE NARRATIVE!"
    scenario[60:60 + len(narrative)] = narrative
    struct.pack_into("<i", scenario, 860, 6)
    struct.pack_into("<i", scenario, 864, 2)
    scenario[872:874] = b"\x01\x01"
    struct.pack_into("<h", scenario, 1032 + 27 * 2, 1)
    events = bytearray(18628)
    struct.pack_into("<h", events, 0, 1)
    struct.pack_into("<hBB", events, 4, 23, 1, 7)
    struct.pack_into("<h", events, 24, 3)
    # A stale unused record must not be interpreted as a second active event.
    struct.pack_into("<hBB", events, 124 + 4, 999, 2, 9)
    body = header + skipped + scenario + events
    pak = bytearray(struct.pack("<50I", 200, *([0] * 49)) + body)
    return pak, 200 + len(header) + len(skipped)


class ParserTests(unittest.TestCase):
    def test_compressed_and_uncompressed_skip_lengths(self):
        for compressed in (True, False):
            with self.subTest(compressed=compressed):
                pak, scenario_offset = synthetic_pak(compressed)
                start, end = campaign.directory(pak)[0]
                chunks = campaign.record_chunks(pak, start, end)
                self.assertEqual(chunks[75]["offset"], scenario_offset)
                self.assertEqual(chunks[136]["offset"], scenario_offset + 1592)
                metadata = campaign.scenario_metadata(pak, scenario_offset)
                self.assertEqual(metadata["start_year_signed"], -1234)
                self.assertEqual(metadata["map"]["height_tiles"], 112)
                self.assertEqual(metadata["goals"]["housing_count"], {"value": 6, "enabled_raw": 1})
                self.assertNotIn("SYNTHETIC SOURCE", json.dumps(metadata))

    def test_zero_declared_events_ignore_stale_records(self):
        pak, scenario_offset = synthetic_pak()
        event_offset = scenario_offset + 1592
        summary, rows = campaign.event_metadata(pak, event_offset)
        self.assertEqual(summary["declared_count"], 1)
        self.assertEqual(rows[0][:3], [23, 1, 7])
        struct.pack_into("<h", pak, event_offset, 0)
        summary, rows = campaign.event_metadata(pak, event_offset)
        self.assertEqual(summary, {"declared_count": 0, "type_code_counts": {}})
        self.assertEqual(rows, [])

    def test_invalid_directories_fail(self):
        for entries in ((200, 0, 300), (200, 200), (200, 100), (199,), (999999,)):
            with self.subTest(entries=entries):
                pak, _ = synthetic_pak()
                struct.pack_into("<50I", pak, 0, *entries, *([0] * (50 - len(entries))))
                with self.assertRaises(campaign.FormatError):
                    campaign.directory(pak)

    def test_record_boundary_and_descriptor_corruption_fail(self):
        mutations = (
            (200 + 4, "I", 999),  # unknown version
            (200 + 8, "I", 301),  # descriptor overflow
            (200 + 12 + 7 * 20, "I", 2),  # unsupported compression
            (200 + 12 + 75 * 20 + 8, "I", 1593),  # changed metadata size
            (200 + 6012, "I", 0x7FFFFFFF),  # compressed payload outside mission
        )
        for offset, fmt, value in mutations:
            with self.subTest(offset=offset):
                pak, _ = synthetic_pak()
                struct.pack_into("<" + fmt, pak, offset, value)
                with self.assertRaises(campaign.FormatError):
                    campaign.record_chunks(pak, 200, len(pak))
        pak, _ = synthetic_pak()
        with self.assertRaises(campaign.FormatError):
            campaign.record_chunks(pak[:-1], 200, len(pak) - 1)
        with self.assertRaises(campaign.FormatError):
            campaign.record_chunks(pak + b"x", 200, len(pak) + 1)

    def test_invalid_event_count_fails(self):
        for count in (-1, 151):
            pak, scenario_offset = synthetic_pak()
            struct.pack_into("<h", pak, scenario_offset + 1592, count)
            with self.assertRaises(campaign.FormatError):
                campaign.event_metadata(pak, scenario_offset + 1592)

    def test_campaign_branches_merge_and_period_transitions(self):
        source = b"""[MISSION_NAMES]
Start
Left
Right
Merge
Final (Alias)
[FIRST]
mission=0,200,0,0
choicescreen=0,0
choice=1,0,0,0
choice=2,0,0,0
mission=1,201,0,1
mission=2,202,0,2
mission=3,203,0,0
[SECOND]
mission=4,204,0,0
"""
        periods, missions = campaign.campaign_definition(source)
        self.assertEqual([m["next_mission_ids"] for m in missions], [[1, 2], [3], [3], [4], []])
        self.assertEqual(periods[1]["entry_mission_ids"], [4])
        self.assertEqual(missions[4]["aliases"], ["Alias"])
        with self.assertRaises(campaign.FormatError):
            campaign.campaign_definition(source.replace(b"choice=2,", b"choice=99,"))


@unittest.skipUnless(os.environ.get("NILE_ORIGINAL_DIR"), "set NILE_ORIGINAL_DIR for licensed-source verification")
class OriginalCorpusTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.result, cls.events = campaign.extract(Path(os.environ["NILE_ORIGINAL_DIR"]))

    def test_campaign_coverage_and_nubt(self):
        self.assertEqual(len(self.result["missions"]), 38)
        self.assertEqual([len(p["mission_ids"]) for p in self.result["periods"]], [3, 5, 15, 9, 6])
        nubt = self.result["missions"][0]
        self.assertEqual((nubt["name"], nubt["start_year_signed"], nubt["initial_funds"]), ("Nubt", -3500, 2500))
        self.assertEqual(nubt["map"]["width_tiles"], 84)
        self.assertEqual(nubt["goals"]["housing_count"], {"value": 6, "enabled_raw": 1})
        self.assertEqual(nubt["goals"]["housing_level_index"], {"value": 2, "enabled_raw": 1})
        self.assertEqual(nubt["events"]["declared_count"], 0)
        self.assertEqual(self.events[0]["records"], [])

    def test_representative_goals_and_events(self):
        missions = self.result["missions"]
        self.assertEqual(missions[1]["initial_funds"], 3000)
        self.assertEqual(missions[2]["goals"]["population"], {"value": 600, "enabled_raw": 1})
        self.assertEqual(missions[6]["events"]["declared_count"], 22)
        self.assertEqual(self.events[6]["records"][0][:3], [0, 1, 7])
        self.assertEqual(missions[10]["monument_codes_raw"], [9, 0, 0])
        self.assertEqual(missions[18]["monument_codes_raw"], [21, 16, 14])
        self.assertEqual(missions[23]["events"]["declared_count"], 65)
        self.assertEqual(missions[37]["goals"]["population"], {"value": 12000, "enabled_raw": 1})
        self.assertEqual(sum(len(m["records"]) for m in self.events), 1351)
        self.assertEqual({m["source_record"]["format_version"] for m in missions}, {147, 149})

    def test_every_mission_reachable_without_cycles(self):
        missions = self.result["missions"]
        reached = set()

        def visit(index, ancestors):
            self.assertNotIn(index, ancestors)
            if index in reached:
                return
            reached.add(index)
            for next_id in missions[index]["next_mission_ids"]:
                visit(next_id, ancestors | {index})

        visit(0, set())
        self.assertEqual(reached, set(range(38)))
        self.assertEqual([m["id"] for m in missions if not m["next_mission_ids"]], [36, 37])

    def test_original_labels_are_not_replaced_by_reconstruction(self):
        labels = self.result["editor_text_labels"]
        self.assertEqual(labels["28"], "Bridge")
        self.assertEqual(labels["29"], "Ferry Landing")
        self.assertEqual(labels["45"], "Sphinx")
        self.assertEqual(labels["46"], "Pyramid")
        self.assertNotIn("47", labels)

    def test_checked_in_data_reproduces_byte_for_byte(self):
        target = MODULE.parents[2] / "internal/campaign/data"
        with tempfile.TemporaryDirectory() as temporary:
            output = Path(temporary)
            campaign.write_outputs(self.result, self.events, output)
            for name in ("base-campaign.json", "base-campaign-events.jsonl"):
                self.assertEqual((output / name).read_bytes(), (target / name).read_bytes(), name)


if __name__ == "__main__":
    unittest.main()
