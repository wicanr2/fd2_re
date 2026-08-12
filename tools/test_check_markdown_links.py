import tempfile
import unittest
from pathlib import Path

import check_markdown_links as checker


class MarkdownLinkCheckTest(unittest.TestCase):
    def test_extracts_inline_image_and_reference_targets(self):
        with tempfile.TemporaryDirectory() as temp:
            source = Path(temp) / "doc.md"
            source.write_text(
                "[文件](docs/a.md) ![圖](images/a.png)\n"
                "[證據]: evidence.txt\n"
                "```markdown\n[忽略](missing.md)\n```\n",
                encoding="utf-8",
            )
            self.assertEqual(
                list(checker.iter_targets(source)),
                [(1, "docs/a.md"), (1, "images/a.png"), (2, "evidence.txt")],
            )

    def test_filters_external_anchor_and_glob_targets(self):
        self.assertIsNone(checker.local_path("https://example.com/a"))
        self.assertIsNone(checker.local_path("#section"))
        self.assertIsNone(checker.local_path("docs/*.md"))
        self.assertEqual(checker.local_path("<docs/file name.md>"), "docs/file name.md")


if __name__ == "__main__":
    unittest.main()
