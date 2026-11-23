"""Bounded Gold-format checks; exact source verification needs licensed files."""

import json
import os
from pathlib import Path
import struct
import tempfile
import unittest

import extract_campaign as base
import extract_cleopatra as cleopatra
from test_extract_campaign import synthetic_pak


class ExpansionFormatTests(unittest.TestCase):
    def test_expanded_directory_and_record_require_explicit_variant(self):
        original, _ = synthetic_pak()
        data = bytearray(struct.pack('<60I', 240, *([0]*59)) + original[200:])
        struct.pack_into('<II', data, 244, 160, 181)
        with self.assertRaises(base.FormatError):
            base.directory(data)
        span, = base.directory(data, 60)
        with self.assertRaises(base.FormatError):
            base.record_chunks(data, *span)
        chunks = base.record_chunks(data, *span, layouts=cleopatra.GOLD_LAYOUTS)
        self.assertEqual(len(chunks), 181)
        self.assertEqual(base.scenario_metadata(data, chunks[75]['offset'])['initial_funds'], 12345)
        struct.pack_into('<I', data, 244, 161)
        with self.assertRaises(base.FormatError):
            base.record_chunks(data, *span, layouts=cleopatra.GOLD_LAYOUTS)

    def test_case_insensitive_source_is_unambiguous(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root/'mission1.pak').write_bytes(b'fixture')
            self.assertEqual(cleopatra.source_file(root, 'MISSION1.PAK').name, 'mission1.pak')
            (root/'MISSION1.PAK').write_bytes(b'other')
            with self.assertRaises(base.FormatError):
                cleopatra.source_file(root, 'MISSION1.PAK')


@unittest.skipUnless(os.environ.get('NILE_GOLD_DIR') and os.environ.get('NILE_BLAST_DECODER'),
                     'set NILE_GOLD_DIR and NILE_BLAST_DECODER for licensed Gold verification')
class GoldSourceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.result, cls.events, cls.economy = cleopatra.extract(
            Path(os.environ['NILE_GOLD_DIR']), Path(os.environ['NILE_BLAST_DECODER']))
        cls.missions = {m['id']: m for m in cls.result['missions']}

    def test_lowercase_gold_installation(self):
        source = Path(os.environ['NILE_GOLD_DIR'])
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for name in cleopatra.GOLD_HASHES:
                (root/name.lower()).symlink_to(cleopatra.source_file(source, name).resolve())
            actual = cleopatra.extract(root, Path(os.environ['NILE_BLAST_DECODER']))
            self.assertEqual(actual, (self.result, self.events, self.economy))

    def test_complete_expansion_without_replacing_base(self):
        self.assertEqual(list(self.missions), list(range(38, 53)))
        self.assertEqual([len(p['mission_ids']) for p in self.result['periods']], [3, 4, 3, 5])
        self.assertEqual(sum(len(m['records']) for m in self.events), 414)
        self.assertEqual(sum(len(m['cities']) for m in self.economy['missions']), 241)
        self.assertTrue(all(m['source_record']['format_version'] == 160 and
                            m['source_record']['descriptor_count'] == 181 for m in self.missions.values()))
        self.assertEqual([m['events']['declared_count'] for m in self.missions.values()],
                         [20, 12, 29, 32, 23, 27, 28, 40, 10, 20, 40, 45, 25, 43, 20])

    def test_original_goals_preserve_enabled_flags_and_deadlines(self):
        first = self.missions[38]
        self.assertEqual((first['name'], first['start_year_signed'], first['initial_funds']),
                         ('Thutmose in the Valley', -1490, 8000))
        self.assertEqual(first['map']['width_tiles'], 226)
        self.assertEqual(first['goals']['population'], {'value': 2000, 'enabled_raw': 0})
        self.assertEqual(first['monument_codes_raw'], [33, 0, 0])
        self.assertEqual(first['burial_requirements_raw'], {'15': 3, '17': 4, '18': 4, '19': 6, '28': 1})
        self.assertEqual(self.missions[39]['goals']['time_limit_years'], {'value': 8, 'enabled_raw': 1})
        self.assertEqual(self.missions[42]['goals']['time_limit_years'], {'value': 7, 'enabled_raw': 1})
        self.assertEqual(self.missions[45]['goals']['housing_count'], {'value': 5, 'enabled_raw': 1})
        self.assertEqual(self.missions[45]['goals']['housing_level_index'], {'value': 8, 'enabled_raw': 1})
        self.assertEqual(self.missions[49]['monument_codes_raw'], [29, 28, 0])
        self.assertEqual(self.missions[51]['monument_codes_raw'], [30, 27, 0])
        self.assertEqual(self.missions[52]['goals']['survival_years'], {'value': 6, 'enabled_raw': 1})
        self.assertEqual(self.missions[52]['enemy_id_raw'], 9)

    def test_new_resource_names_and_economic_identity_joins(self):
        names = self.result['resource_text_labels']
        self.assertEqual([names[str(i)] for i in range(31, 36)], ['Oil', 'Henna', 'Paint', 'Lamps', 'Marble'])
        self.assertEqual(self.result['monument_text_labels']['37'], 'Abu Simbel')
        cities = {c['id']: c for c in self.economy['missions'][0]['cities']}
        self.assertEqual(cities[9]['name'], 'Deir el-Medina')
        self.assertEqual(cities[9]['sells'], [4, 11, 16, 32])
        self.assertEqual(cities[1]['name'], 'Itjtawy')
        self.assertEqual(cities[1]['cost_to_open'], 825)
        self.assertTrue(cities[1]['sea_trade'])
        self.assertEqual(cities[1]['annual_limits']['34'], 1500)
        self.assertEqual(cities[6]['potential_sells'], [18, 31])
        self.assertEqual(cities[6]['sells'], [])
        self.assertEqual(cities[6]['annual_limits']['31'], 1500)
        for mission in self.economy['missions']:
            self.assertEqual(len([c for c in mission['cities'] if c['type_raw'] == 0]), 1)
            for city in mission['cities']:
                self.assertTrue(set(city['sells']).issubset(city['potential_sells']))
                self.assertTrue(set(city['buys']).issubset(city['potential_buys']))
                self.assertNotIn('x', city)
                self.assertNotIn('y', city)

    def test_checked_in_files_reproduce_exactly(self):
        target = Path(__file__).resolve().parents[2]/'internal/campaign/data'
        with tempfile.TemporaryDirectory() as temporary:
            output = Path(temporary)
            cleopatra.write_outputs(self.result, self.events, self.economy, output)
            for name in ('cleopatra-campaign.json', 'cleopatra-campaign-events.jsonl', 'cleopatra-campaign-economy.json'):
                self.assertEqual((target/name).read_bytes(), (output/name).read_bytes(), name)


if __name__ == '__main__':
    unittest.main()
