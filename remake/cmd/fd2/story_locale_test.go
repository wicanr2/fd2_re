package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

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
