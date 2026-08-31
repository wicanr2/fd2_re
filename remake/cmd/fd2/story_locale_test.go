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
			"legacy/line/15b3c967fb2b/scenes/0/lines/0":  "アレス、もう一度勝負しよう！昨日負けたのが悔しくてたまらない！",
			"legacy/line/15b3c967fb2b/scenes/2/lines/7":  "ユニさん、まずは僕と一緒に戻ろう。数日後にはマラ大陸へ出発するよ。心配しないで、必ず無事に家へ送り届けるから。",
			"legacy/line/ae86adb52dac/scenes/0/lines/0":  "父上、ソールです。謁見に参りました。",
			"legacy/line/ae86adb52dac/scenes/1/lines/17": "それじゃ仕方ないな。もう引き受けるしかないのか？",
			"legacy/line/7ecb566a60db/scenes/1/lines/5":  "ただの強盗じゃない。こいつらはマラ大陸の沿岸を荒らし、人殺しも略奪も何でもする海賊だって聞いている。",
			"legacy/line/7ecb566a60db/scenes/7/lines/8":  "ぼ、僕はハノです。これから、よろしくお願いします。",
			"legacy/line/7ecb566a60db/scenes/1/lines/13": "ならば容赦するな！　行け！",
			"legacy/line/7ecb566a60db/scenes/2/lines/10": "若者たちは元気だな！　お前たち、行くぞ！",
			"legacy/line/7ecb566a60db/scenes/3/lines/12": "くそっ！　あの小僧二人、俺を完全に無視しやがって！　殺せ！　一人も逃がすな！",
			"legacy/line/7ecb566a60db/scenes/4/lines/4":  "ソール、これから人の縄張りに入るんだから、せめて礼儀正しくしなよ！",
			"legacy/line/7ecb566a60db/scenes/7/lines/12": "お父さん、安心してください！　それじゃ行きます！",
		},
		"en": {
			"legacy/line/15b3c967fb2b/scenes/0/lines/0":  "Ares, let's have another match! Losing to you yesterday still bothers me!",
			"legacy/line/ae86adb52dac/scenes/0/lines/0":  "I am Sol. I have come to pay my respects, Father.",
			"legacy/line/ae86adb52dac/scenes/1/lines/16": "Ever since I was a child, I wanted to go abroad and have grand adventures—not sit on a throne for the rest of my life. I am only Father's adopted son and have no royal blood, so I always thought Dean would inherit the throne and I would be free. But now...",
			"legacy/line/7ecb566a60db/scenes/3/lines/4":  "That must be the pirate leader! At last, a worthy opponent has appeared. Leave this one to me!",
			"legacy/line/7ecb566a60db/scenes/7/lines/8":  "My name is... Hano. I hope we can get along.",
			"legacy/line/7ecb566a60db/scenes/1/lines/13": "Then show them no mercy! Attack!",
			"legacy/line/7ecb566a60db/scenes/3/lines/10": "I can't stand this guy. Sol, let's stop arguing and take him down together!",
			"legacy/line/7ecb566a60db/scenes/3/lines/12": "Damn it! Those two brats don't respect me at all! Kill them! Don't let a single one escape!",
			"legacy/line/7ecb566a60db/scenes/7/lines/5":  "Of course, sir! There’s no problem with that at all! Isn’t that right, Yuni?",
		},
		"zh-Hans": {
			"legacy/line/15b3c967fb2b/scenes/1/lines/13": "悠妮?好名字。悠妮小姐,你怎么会记不得怎么来到这里的?",
			"legacy/line/7ecb566a60db/scenes/0/lines/2":  "太好了。悠妮，你……嗯，坐了这么久的船，有点累吧？",
			"legacy/line/7ecb566a60db/scenes/4/lines/2":  "我们是亚克斯王国的海岸巡防队，消灭肆虐沿海的海盗本是我们的职责，这些海盗就交给我们来处理，请各位放心！",
			"legacy/line/7ecb566a60db/scenes/1/lines/13": "那就杀无赦！上啊！",
			"legacy/line/7ecb566a60db/scenes/2/lines/2":  "什么？待我看看……啊哈，这不是海盗在打劫旅客吗？居然敢在我们门前抢人，胆子不小啊！",
			"legacy/line/7ecb566a60db/scenes/3/lines/12": "可恶啊！这两个小子全不把我放在眼里！给我杀！一个都别放过！",
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
