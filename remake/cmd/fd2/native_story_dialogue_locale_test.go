package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/localization"
)

func TestLocalizedNativeDialogueLayoutBoundsEnglishAndJapanese(t *testing.T) {
	displayFont := loadFont()
	if displayFont == nil {
		t.Fatal("official CJK font unavailable")
	}
	source := &campaign.NativeDialogueLayout{
		SourceDAT: "FDTXT_001", StringIndex: 0, Utterance: 0,
		Control: "FFEC", Operand: 0, Pages: [][]string{{"原文"}},
	}
	for name, text := range map[string]string{
		"English":  "A translated indexed dialogue keeps its frame, portrait, camera, and progressive text.",
		"Japanese": "翻訳された会話は原作の枠と頭像と逐字表示をそのまま保持します。",
	} {
		t.Run(name, func(t *testing.T) {
			layout, err := localizedNativeDialogueLayout(text, source, displayFont)
			if err != nil {
				t.Fatal(err)
			}
			limit, _ := campaign.NativeDialogueLineGlyphLimit(layout.Control)
			_, _, width, _, rows, _ := campaign.NativeStoryDialogueTextGeometry(layout.Control)
			var joined strings.Builder
			for page, lines := range layout.Pages {
				if len(lines) == 0 || len(lines) > rows {
					t.Fatalf("page %d rows=%d", page, len(lines))
				}
				for _, line := range lines {
					if len([]rune(line)) > limit || displayFont.Width(line, localizedNativeDialogueFontScale) > float64(width) {
						t.Fatalf("line exceeds safe rectangle: %q", line)
					}
					joined.WriteString(line)
				}
			}
			if joined.String() != text {
				t.Fatalf("wrapped text changed: %q", joined.String())
			}
		})
	}
}

func TestLocalizedNativeDialogueBuildsProgressiveIndexedFrames(t *testing.T) {
	displayFont := loadFont()
	dialogueCells := make([]fdother.RawCell, 20)
	for index := range dialogueCells {
		width, height := 1, 1
		switch index {
		case 1, 2, 3, 4:
			width, height = 3, 3
		case 5, 6, 7, 8, 9, 12:
			width, height = 16, 3
		case 10, 11, 14, 15, 16, 17:
			width, height = 3, 16
		case 13:
			width, height = 16, 16
		}
		dialogueCells[index] = fdother.RawCell{Width: width, Height: height, Pixels: make([]byte, width*height)}
		for pixel := range dialogueCells[index].Pixels {
			dialogueCells[index].Pixels[pixel] = 0x4a
		}
	}
	portrait := dato.Frame{Width: 1, Height: 1, Pixels: []byte{0x33}}
	source := &campaign.NativeDialogueLayout{
		SourceDAT: "FDTXT_001", StringIndex: 0, Utterance: 0,
		Control: "FFEC", Operand: 0, Pages: [][]string{{"原文"}},
	}
	layout, err := localizedNativeDialogueLayout("Indexed text 日本語", source, displayFont)
	if err != nil {
		t.Fatal(err)
	}
	frames, err := composeLocalizedNativeDialogueProgressiveFrames(
		make([]byte, 320*200), dialogueCells, portrait, displayFont, layout, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	want := 1
	for _, row := range layout.Pages[0] {
		want += len([]rune(row))
	}
	if len(frames) != want || bytes.Equal(frames[0], frames[len(frames)-1]) {
		t.Fatalf("progressive frames=%d want=%d changed=%v", len(frames), want, !bytes.Equal(frames[0], frames[len(frames)-1]))
	}
}

func TestDrawIndexedLocalizedLineUsesNativePaletteAndRejectsMissingGlyph(t *testing.T) {
	displayFont := loadFont()
	if displayFont == nil {
		t.Fatal("official CJK font unavailable")
	}
	frame := make([]byte, 320*200)
	if err := drawIndexedLocalizedLine(frame, displayFont, "Test日本語", 16, 16, 224, 19); err != nil {
		t.Fatal(err)
	}
	foreground, shadow := 0, 0
	for _, pixel := range frame {
		switch pixel {
		case 0xcd:
			foreground++
		case 0x4c:
			shadow++
		}
	}
	if foreground == 0 || shadow == 0 {
		t.Fatalf("indexed ink missing: foreground=%d shadow=%d", foreground, shadow)
	}
	if err := drawIndexedLocalizedLine(frame, displayFont, string(rune(0x10ffff)), 16, 16, 224, 19); err == nil {
		t.Fatal("missing glyph was accepted")
	}
}

func TestAllHandlerNativeDialoguesCompileForThreeTranslatedLocales(t *testing.T) {
	displayFont := loadFont()
	if displayFont == nil {
		t.Fatal("official CJK font unavailable")
	}
	graph, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	contents := make(map[string]*localization.ContentCatalog)
	for _, localeID := range []string{"zh-Hans", "ja", "en"} {
		contents[localeID], err = loadOfficialLocaleContent(localeID)
		if err != nil {
			t.Fatal(err)
		}
	}
	compiled := 0
	for nodeID, node := range graph.Nodes {
		if node.HandlerBinding == "" {
			continue
		}
		beats, issues, err := campaign.CompileHandlerBinding(assetPath(node.HandlerBinding))
		if err != nil {
			t.Fatalf("%s binding: %v", nodeID, err)
		}
		if len(issues) != 0 {
			// 與正式 enterNode 相同：有 unresolved issue 的 binding 不會進 runtime。
			continue
		}
		beats, err = campaign.ExpandNativeDialogueGroups(beats)
		if err != nil {
			t.Fatalf("%s expand: %v", nodeID, err)
		}
		for _, beat := range beats {
			if beat.Op != "dialog" {
				continue
			}
			layouts := beat.NativeDialogues
			if beat.NativeDialogue != nil {
				layouts = []*campaign.NativeDialogueLayout{beat.NativeDialogue}
			}
			if len(layouts) == 0 {
				continue
			}
			if beat.Script == "" {
				t.Fatalf("%s native dialogue lacks canonical script", nodeID)
			}
			lines, err := loadStoryScriptWithIdentityAt(handlerStoryPath(beat.Script), beat.Scene, beat.SceneIndex)
			if err != nil {
				t.Fatalf("%s story: %v", nodeID, err)
			}
			for index, source := range layouts {
				lineIndex := beat.Line + index
				if lineIndex < 0 || lineIndex >= len(lines) {
					t.Fatalf("%s line %d unavailable", nodeID, lineIndex)
				}
				for localeID, content := range contents {
					text, err := content.StoryText(lines[lineIndex].LineID)
					if err != nil {
						t.Fatalf("%s %s: %v", nodeID, localeID, err)
					}
					layout, err := localizedNativeDialogueLayout(text, source, displayFont)
					if err != nil {
						t.Fatalf("%s %s layout: %v", nodeID, localeID, err)
					}
					x, y, width, lineStep, _, _ := campaign.NativeStoryDialogueTextGeometry(layout.Control)
					for _, page := range layout.Pages {
						for row, text := range page {
							if err := drawIndexedLocalizedLine(make([]byte, 320*200), displayFont, text, x, y+row*lineStep, width, lineStep); err != nil {
								t.Fatalf("%s %s glyphs: %v", nodeID, localeID, err)
							}
						}
					}
				}
				compiled++
			}
		}
	}
	if compiled < 50 {
		t.Fatalf("compiled only %d native dialogue utterances", compiled)
	}
}
