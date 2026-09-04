import unittest

import decode_story_text as subject


class DecodeStoryTextTest(unittest.TestCase):
    def test_direct_unit_index_is_not_mapped_to_global_character_name(self):
        got = subject.decode_string([0xFFED, 7, subject.OPEN, 40, 39, subject.CLOSE])
        self.assertEqual(got, [("單位索引 7", subject.g2s([40, 39]))])

    def test_identity_tag_is_not_rendered_as_a_glyph(self):
        got = subject.decode_string([0xFFEF, 77, subject.OPEN, 40, 39, subject.CLOSE])
        self.assertEqual(got, [("身分標籤 77", subject.g2s([40, 39]))])
        self.assertNotIn("約", got[0][0])


if __name__ == "__main__":
    unittest.main()
