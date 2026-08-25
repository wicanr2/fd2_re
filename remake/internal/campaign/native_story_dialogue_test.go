package campaign

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func TestCh24NativeDialogueLayoutsMatchOriginalControlWords(t *testing.T) {
	raw, err := os.ReadFile("../../../extracted/raw/FDTXT/FDTXT_025.bin")
	if err != nil {
		t.Skip("extracted FDTXT_025 oracle is absent")
	}
	stringsTable, err := fdtxt.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	glyphRaw, err := os.ReadFile("../../../docs/data/glyph_map.json")
	if err != nil {
		t.Fatal(err)
	}
	var encoded map[string]json.RawMessage
	if err := json.Unmarshal(glyphRaw, &encoded); err != nil {
		t.Fatal(err)
	}
	glyphs := make(map[uint16]string, len(encoded))
	for key, value := range encoded {
		if key == "_comment" {
			continue
		}
		var text string
		if err := json.Unmarshal(value, &text); err != nil {
			t.Fatal(err)
		}
		var index int
		if _, err := fmt.Sscanf(key, "%d", &index); err != nil || index < 0 || index > 0xffff {
			t.Fatalf("invalid glyph map key %q", key)
		}
		glyphs[uint16(index)] = text
	}
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch24_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("compile ch24 binding: issues=%v err=%v", issues, err)
	}
	byString := map[int][]*NativeDialogueLayout{6: {}, 7: {}}
	for _, beat := range beats {
		if beat.Op == "dialog" && beat.NativeDialogue != nil {
			byString[beat.NativeDialogue.StringIndex] = append(byString[beat.NativeDialogue.StringIndex], beat.NativeDialogue)
		}
	}
	for _, stringIndex := range []int{6, 7} {
		words, err := stringsTable.Words(stringIndex)
		if err != nil {
			t.Fatal(err)
		}
		want, err := decodeOriginalNativeDialogueLayouts("FDTXT_025", stringIndex, words, glyphs)
		if err != nil {
			t.Fatal(err)
		}
		got := byString[stringIndex]
		if len(got) != len(want) {
			t.Fatalf("FDTXT_025 index%d layouts=%d, want %d", stringIndex, len(got), len(want))
		}
		for i := range want {
			if !reflect.DeepEqual(got[i], want[i]) {
				t.Fatalf("FDTXT_025 index%d utterance%d\ngot  %#v\nwant %#v", stringIndex, i, got[i], want[i])
			}
		}
	}
}

func decodeOriginalNativeDialogueLayouts(source string, stringIndex int, words []uint16, glyphs map[uint16]string) ([]*NativeDialogueLayout, error) {
	isSpeaker := func(word uint16) bool { return word >= 0xffec && word <= 0xffef }
	var layouts []*NativeDialogueLayout
	for cursor := 0; cursor < len(words); {
		if !isSpeaker(words[cursor]) {
			cursor++
			continue
		}
		if cursor+1 >= len(words) {
			return nil, fmt.Errorf("speaker control at end of FDTXT string")
		}
		layout := &NativeDialogueLayout{
			SourceDAT: source, StringIndex: stringIndex, Utterance: len(layouts),
			Control: fmt.Sprintf("%04X", words[cursor]), Operand: int(words[cursor+1]),
		}
		cursor += 2
		var pages [][]string
		var page []string
		var row string
		flushRow := func() {
			if row != "" {
				page = append(page, row)
				row = ""
			}
		}
		flushPage := func() {
			flushRow()
			if len(page) > 0 {
				pages = append(pages, page)
				page = nil
			}
		}
		for cursor < len(words) && !isSpeaker(words[cursor]) {
			word := words[cursor]
			cursor++
			switch word {
			case fdtxt.StringEnd:
				cursor = len(words)
			case 0xfffe:
				flushRow()
			case 0xfffd:
				flushPage()
			default:
				text, ok := glyphs[word]
				if !ok {
					return nil, fmt.Errorf("FDTXT glyph %#x is absent from glyph_map", word)
				}
				row += text
			}
		}
		flushPage()
		layout.Pages = pages
		if err := layout.Validate(); err != nil {
			return nil, err
		}
		layouts = append(layouts, layout)
	}
	return layouts, nil
}

func TestComposeNativeStoryDialoguePageUsesOriginalIndexedAssets(t *testing.T) {
	base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided original FDOTHER.DAT is absent")
	}
	resource5, err := fdother.ReadResource(fdotherPath, 5)
	if err != nil {
		t.Fatal(err)
	}
	cells := make([]fdother.RawCell, 20)
	for index := 1; index <= 19; index++ {
		cells[index], err = fdother.ParseLMI1RawEntry(resource5, index)
		if err != nil {
			t.Fatal(err)
		}
	}
	fontRaw, err := fdother.ReadResource(fdotherPath, 4)
	if err != nil {
		t.Fatal(err)
	}
	font, err := fdtxt.ParseFont(fontRaw)
	if err != nil {
		t.Fatal(err)
	}
	portraits, err := dato.DecodeResource(filepath.Join(base, "DATO.DAT"), 26)
	if err != nil || len(portraits) == 0 {
		t.Fatalf("DATO#26 portrait: count=%d err=%v", len(portraits), err)
	}
	rawIndex, err := os.ReadFile("../../assets/fonts/unicode_to_glyph.json")
	if err != nil {
		t.Fatal(err)
	}
	var encoded map[string]json.RawMessage
	if err := json.Unmarshal(rawIndex, &encoded); err != nil {
		t.Fatal(err)
	}
	index := make(map[string]int, len(encoded))
	for key, value := range encoded {
		if key == "_comment" {
			continue
		}
		var glyph int
		if err := json.Unmarshal(value, &glyph); err != nil {
			t.Fatal(err)
		}
		index[key] = glyph
	}
	layout := &NativeDialogueLayout{
		SourceDAT: "FDTXT_025", StringIndex: 6, Utterance: 0,
		Control: "FFED", Operand: 16,
		Pages: [][]string{{"『真是驚險的一戰！約拿老頭", "　，你從哪裡找來這些生力軍", "　？如果沒有他們，這一戰的"}, {"　結果就會改觀了。』"}},
	}
	background := make([]byte, 320*200)
	for i := range background {
		background[i] = 7
	}
	page0, err := ComposeNativeStoryDialoguePage(background, cells, portraits[0], font, index, layout, 0)
	if err != nil {
		t.Fatal(err)
	}
	page1, err := ComposeNativeStoryDialoguePage(background, cells, portraits[0], font, index, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page0) != 320*200 || len(page1) != 320*200 || page0[0] != 7 || page0[nativeStoryUpperText] == 7 {
		t.Fatalf("native story page did not preserve source/outside frame or render text")
	}
	if string(page0) == string(page1) {
		t.Fatal("two FFFD pages produced an identical indexed frame")
	}
	if background[nativeStoryUpperText] != 7 {
		t.Fatal("native story compositor mutated its caller-owned background")
	}
	progressive, err := ComposeNativeStoryDialogueProgressiveFrames(
		background, cells, portraits[0], font, index, layout, 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	wantFrames := 1
	for _, row := range layout.Pages[1] {
		wantFrames += len([]rune(row))
	}
	if len(progressive) != wantFrames {
		t.Fatalf("progressive frames=%d, want frame0 plus %d glyphs", len(progressive), wantFrames-1)
	}
	if string(progressive[len(progressive)-1]) != string(page1) {
		t.Fatal("progressive final frame differs from stable page")
	}
	if string(progressive[0]) == string(progressive[len(progressive)-1]) {
		t.Fatal("progressive sequence did not publish any visible glyph")
	}
	bad := *layout
	bad.Pages = [][]string{{"🙂"}}
	if _, err := ComposeNativeStoryDialoguePage(background, cells, portraits[0], font, index, &bad, 0); err == nil {
		t.Fatal("unknown editable glyph did not fail closed")
	}
}
