#!/usr/bin/env python3
import tempfile
import unittest
from pathlib import Path

import build_full_locale_content as subject


class FullLocaleContentTest(unittest.TestCase):
    def test_guarded_translation_never_sends_variables_terms_or_identifiers(self):
        seen = []

        def convert(text):
            seen.append(text)
            return "<" + text + ">"

        got = subject.translate_guarded("索爾 modifier 造成 %d 傷害", [("索爾", "Sol")], convert)
        self.assertEqual(got, "Sol modifier< 造成 >%d< 傷害>")
        self.assertEqual(seen, [" 造成 ", " 傷害"])

    def test_story_pointer_uses_canonical_line_identity(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            subject.write_json(root / "ch01.json", {
                "document_id": "story/ch01",
                "scenes": [{"scene_id": "scene/a", "lines": [{"line_id": "line/a"}]}],
            })
            entries = [{
                "string_id": "legacy.text", "role": "dialogue", "text": "台詞",
                "source": {"file": "remake/assets/story/ch01.json", "json_pointer": "/scenes/0/lines/0/text"},
            }]
            self.assertEqual(subject.canonical_story_ids(entries, root), {"legacy.text": "line/a/text"})


if __name__ == "__main__":
    unittest.main()
