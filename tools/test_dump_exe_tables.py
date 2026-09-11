import unittest

import dump_exe_tables


class NativeItemEffectRowsTest(unittest.TestCase):
    def test_class_names_cover_native_text_indices_zero_through_twenty_eight(self):
        self.assertEqual(len(dump_exe_tables.CLASS_NAMES), 29)
        self.assertEqual(dump_exe_tables.CLASS_NAMES[26], "？？？")
        self.assertEqual(dump_exe_tables.CLASS_NAMES[27], "　　")
        self.assertEqual(dump_exe_tables.CLASS_NAMES[28], "？？？")

    def test_runtime_view_is_shifted_one_byte_from_normalized_rows(self):
        base = dump_exe_tables.ANCHORS["item"][0]
        stride = 0x17
        data = bytearray(base + stride * 2 + 1)
        normalized_row0 = bytes(range(0x10, 0x10 + stride))
        normalized_row1 = bytes(range(0x40, 0x40 + stride))
        data[base:base + stride] = normalized_row0
        data[base + stride:base + stride * 2] = normalized_row1
        data[base + stride * 2] = 0x7E

        rows = dump_exe_tables.dump_native_item_effect_rows(data, count=2)

        self.assertEqual(len(rows), 2)
        self.assertEqual(rows[0]["off"], hex(base + 1))
        self.assertEqual(rows[0]["linear"], "0x602ad")
        self.assertEqual(
            bytes.fromhex(rows[0]["raw"]),
            normalized_row0[1:] + normalized_row1[:1],
        )
        self.assertEqual(
            bytes.fromhex(rows[1]["raw"]),
            normalized_row1[1:] + b"\x7e",
        )

    def test_incomplete_runtime_row_is_not_exported(self):
        base = dump_exe_tables.ANCHORS["item"][0]
        data = bytearray(base + 0x17)

        self.assertEqual(
            dump_exe_tables.dump_native_item_effect_rows(data, count=1),
            [],
        )

    def test_native_movement_cost_rows_have_exact_29_by_20_boundary(self):
        base = 0x55446
        data = bytearray(base + 29 * 20)
        for selector in range(29):
            data[base + selector * 20:base + (selector + 1) * 20] = bytes([selector] * 20)

        rows = dump_exe_tables.dump_native_movement_cost_rows(data)

        self.assertEqual(len(rows), 29)
        self.assertEqual(rows[0]["linear"], "0x61646")
        self.assertEqual(rows[28]["linear"], hex(0x61646 + 28 * 20))
        self.assertEqual(rows[17]["costs"], [17] * 20)


if __name__ == "__main__":
    unittest.main()
