package localization

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOfficialContentAssetsLoadAndResolveStory(t *testing.T) {
	root := filepath.Join("..", "..", "assets", "locales")
	for _, locale := range []string{"zh-Hant", "zh-Hans", "ja", "en"} {
		catalog, err := LoadOfficialContent(root, locale)
		if err != nil {
			t.Fatalf("official %s content: %v", locale, err)
		}
		if _, err := catalog.StoryText("legacy/line/7ecb566a60db/scenes/0/lines/0"); err != nil {
			t.Fatalf("official %s first ch01 line: %v", locale, err)
		}
	}
}

func TestStoryTextFailsClosed(t *testing.T) {
	catalog := &ContentCatalog{entries: map[string]ContentEntry{
		"line/a/text": {StringID: "line/a/text", IDStatus: "stable_canonical", Role: "dialogue", Text: "ok", Status: "reviewed"},
		"line/b/text": {StringID: "line/b/text", IDStatus: "provisional_legacy_path", Role: "dialogue", Text: "bad", Status: "machine_draft"},
		"line/c/text": {StringID: "line/c/text", IDStatus: "stable_canonical", Role: "dialogue", Text: "bad", Status: "blocked"},
	}}
	if got, err := catalog.StoryText("line/a"); err != nil || got != "ok" {
		t.Fatalf("resolved = %q, %v", got, err)
	}
	for _, id := range []string{"missing", "line/b", "line/c"} {
		if _, err := catalog.StoryText(id); err == nil {
			t.Fatalf("%s unexpectedly resolved", id)
		}
	}
}

func TestLoadOfficialContentRejectsUnknownField(t *testing.T) {
	root := filepath.Join("..", "..", "assets", "locales")
	raw, err := os.ReadFile(filepath.Join(root, "en", "content.json"))
	if err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(string(raw), `"entry_count": 5176,`, `"entry_count": 5176, "future": true,`, 1)
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "en"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "en", "content.json"), []byte(bad), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOfficialContent(dir, "en"); err == nil {
		t.Fatal("unknown top-level content field accepted")
	}
}
