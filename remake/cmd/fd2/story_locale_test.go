package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestReviewedOpeningTranslationsUseCanonicalLineIDs(t *testing.T) {
	wants := map[string]map[string]string{
		"ja": {
			"legacy/line/15b3c967fb2b/scenes/0/lines/0": "アレス、もう一度勝負しよう！昨日負けたのが悔しくてたまらない！",
			"legacy/line/15b3c967fb2b/scenes/2/lines/7": "ユニさん、まずは僕と一緒に戻ろう。数日後にはマラ大陸へ出発するよ。心配しないで、必ず無事に家へ送り届けるから。",
		},
		"en": {
			"legacy/line/15b3c967fb2b/scenes/0/lines/0": "Ares, let's have another match! Losing to you yesterday still bothers me!",
		},
		"zh-Hans": {
			"legacy/line/15b3c967fb2b/scenes/1/lines/13": "悠妮?好名字。悠妮小姐,你怎么会记不得怎么来到这里的?",
		},
	}
	for localeID, entries := range wants {
		content, err := loadOfficialLocaleContent(localeID)
		if err != nil {
			t.Fatal(err)
		}
		for lineID, want := range entries {
			got, err := content.StoryText(lineID)
			if err != nil || got != want {
				t.Fatalf("%s %s=%q err=%v, want %q", localeID, lineID, got, err, want)
			}
		}
		assertReviewedContentEntries(t, localeID, entries)
	}
}

func assertReviewedContentEntries(t *testing.T, localeID string, wants map[string]string) {
	t.Helper()
	raw, err := os.ReadFile(assetPath("assets/locales/" + localeID + "/content.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Entries []struct {
			StringID string `json:"string_id"`
			Status   string `json:"status"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	remaining := make(map[string]bool, len(wants))
	for lineID := range wants {
		remaining[lineID+"/text"] = true
	}
	for _, entry := range document.Entries {
		if !remaining[entry.StringID] {
			continue
		}
		if entry.Status != "reviewed" {
			t.Fatalf("%s %s status=%q", localeID, entry.StringID, entry.Status)
		}
		delete(remaining, entry.StringID)
	}
	if len(remaining) != 0 {
		t.Fatalf("%s reviewed entries missing: %v", localeID, remaining)
	}
}

func attachOfficialTestLocale(t *testing.T, g *Game, localeID string) {
	t.Helper()
	catalog, err := loadOfficialLocale(localeID)
	if err != nil {
		t.Fatal(err)
	}
	content, err := loadOfficialLocaleContent(localeID)
	if err != nil {
		t.Fatal(err)
	}
	g.localeID, g.localeCatalog, g.localeContent = localeID, catalog, content
}

func TestAllRuntimeStoriesMatchCanonicalLineIdentities(t *testing.T) {
	paths := assetGlob("assets/story/ch*.json")
	if len(paths) != 35 {
		t.Fatalf("story paths=%d", len(paths))
	}
	allLines := make([]campaign.Line, 0, 1564)
	for _, path := range paths {
		lines, err := loadStoryScriptWithIdentityAt(path, "", nil)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		for index, line := range lines {
			if line.LineID == "" {
				t.Fatalf("%s line %d lacks canonical identity", path, index)
			}
		}
		allLines = append(allLines, lines...)
	}
	if len(allLines) != 1564 {
		t.Fatalf("canonical runtime story lines=%d, want 1564", len(allLines))
	}
	for _, localeID := range localeIDs {
		content, err := loadOfficialLocaleContent(localeID)
		if err != nil {
			t.Fatal(err)
		}
		g := &Game{localeID: localeID, localeContent: content}
		for _, line := range allLines {
			if _, err := g.localizedStoryText(line); err != nil {
				t.Fatalf("%s %s: %v", localeID, line.LineID, err)
			}
		}
	}
}

func TestChapterOneStoryUsesAllOfficialContentCatalogs(t *testing.T) {
	sceneIndex := 0
	lines, err := loadStoryScriptWithIdentityAt("assets/story/ch01.json", "", &sceneIndex)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) == 0 || lines[0].LineID != "legacy/line/7ecb566a60db/scenes/0/lines/0" {
		t.Fatalf("first ch01 line identity = %#v", lines)
	}
	seen := make(map[string]string)
	for _, localeID := range localeIDs {
		content, err := loadOfficialLocaleContent(localeID)
		if err != nil {
			t.Fatalf("load %s content: %v", localeID, err)
		}
		g := &Game{localeID: localeID, localeContent: content}
		got, err := g.resolveCampaignDialogLine(lines[0], nil, nil)
		if err != nil {
			t.Fatalf("resolve %s story: %v", localeID, err)
		}
		want, err := content.StoryText(lines[0].LineID)
		if err != nil || got.Text != want {
			t.Fatalf("resolve %s text = %q, want %q (%v)", localeID, got.Text, want, err)
		}
		seen[localeID] = got.Text
	}
	if seen["zh-Hant"] == seen["en"] || seen["zh-Hant"] == seen["ja"] {
		t.Fatalf("chapter one first line did not use translated content: %#v", seen)
	}
}

func TestStoryLocaleFailsClosedForMissingIdentity(t *testing.T) {
	content, err := loadOfficialLocaleContent("en")
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{localeID: "en", localeContent: content}
	if _, err := g.resolveCampaignDialogLine(campaign.Line{Speaker: 0, Text: "來源"}, nil, nil); err == nil {
		t.Fatal("story line without canonical identity was accepted")
	}
}

func TestBattleEmbeddedEventsUseOfficialStoryContent(t *testing.T) {
	seen := make(map[string][3]string)
	for _, localeID := range localeIDs {
		content, err := loadOfficialLocaleContent(localeID)
		if err != nil {
			t.Fatal(err)
		}
		g := &Game{localeID: localeID, localeContent: content}
		event61, ok := event61DialogueActions(g, 0, 10, 1)
		if !ok || len(event61) != 1 {
			t.Fatalf("%s event61 actions=%d err=%q", localeID, len(event61), g.loadErr)
		}
		event75, ok := event75DialogueActions(g, 0, &battle.Unit{BattleFig: 4, HasBattleFig: true})
		if !ok || len(event75) != 1 || event75[0].Speaker != 4 {
			t.Fatalf("%s event75 actions=%#v err=%q", localeID, event75, g.loadErr)
		}
		event76, ok := event76DialogueActions(g, 2)
		if !ok || len(event76) != 3 {
			t.Fatalf("%s event76 actions=%d err=%q", localeID, len(event76), g.loadErr)
		}
		for _, action := range append(append(event61, event75...), event76...) {
			if action.Text == "" {
				t.Fatalf("%s embedded event produced empty dialogue", localeID)
			}
		}
		seen[localeID] = [3]string{event61[0].Text, event75[0].Text, event76[0].Text}
	}
	if seen["zh-Hant"] == seen["en"] || seen["zh-Hant"] == seen["ja"] {
		t.Fatalf("embedded event dialogue did not use translated catalogs: %#v", seen)
	}
}
