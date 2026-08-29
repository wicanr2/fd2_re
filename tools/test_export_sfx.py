#!/usr/bin/env python3
"""export_sfx.py 的格式與正式分離音效清冊契約測試。"""

import struct
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
import export_sfx  # noqa: E402


def make_container(payloads):
    first = 6 + 4 * len(payloads)
    offsets = []
    cursor = first
    for payload in payloads:
        offsets.append(cursor)
        cursor += len(payload)
    return b"LLLLLL" + b"".join(struct.pack("<I", value) for value in offsets) + b"".join(payloads)


class SeparatedSFXFormatTests(unittest.TestCase):
    def make_outer(self, replace=None):
        selected = set(export_sfx.SEPARATED_RESOURCES)
        payloads = []
        for resource in range(export_sfx.SEPARATED_TOP_RESOURCE_COUNT):
            if resource in selected:
                count = export_sfx.SEPARATED_SAMPLE_COUNTS[resource]
                samples = [bytes([resource, sample, 0x80]) for sample in range(count)]
                if replace and resource == replace[0]:
                    samples = replace[1]
                nested = make_container(samples + [b""])
                payloads.append(nested)
            else:
                payloads.append(b"x")
        return make_container(payloads)

    def test_canonical_banks_have_expected_samples_and_exact_tail(self):
        banks = export_sfx._read_separated_resources(self.make_outer())
        self.assertEqual(tuple(banks), export_sfx.SEPARATED_RESOURCES)
        self.assertEqual(sum(map(len, banks.values())), sum(export_sfx.SEPARATED_SAMPLE_COUNTS.values()))
        for resource, samples in banks.items():
            self.assertEqual(len(samples), export_sfx.SEPARATED_SAMPLE_COUNTS[resource])

    def test_title_bank_has_four_samples_and_empty_tail(self):
        banks = export_sfx._read_separated_resources(self.make_outer())
        self.assertEqual(len(banks[77]), 4)
        self.assertEqual(banks[77][2], bytes([77, 2, 0x80]))

    def test_ani1_companion_has_one_sample_and_empty_tail(self):
        banks = export_sfx._read_separated_resources(self.make_outer())
        self.assertEqual(len(banks[78]), 1)
        self.assertEqual(banks[78][0], bytes([78, 0, 0x80]))

    def test_nonempty_tail_is_rejected(self):
        resource = 82
        count = export_sfx.SEPARATED_SAMPLE_COUNTS[resource]
        with self.assertRaises(ValueError):
            export_sfx._read_separated_resources(
                self.make_outer((resource, [b"x"] * count + [b"not-tail"]))
            )

    def test_source_identity_shape_is_stable(self):
        identity = export_sfx._source_identity(b"test")
        self.assertEqual(set(identity), {"name", "size", "md5", "sha256"})
        self.assertEqual(identity["name"], "FDOTHER.DAT")
        self.assertEqual(identity["size"], 4)


if __name__ == "__main__":
    unittest.main()
