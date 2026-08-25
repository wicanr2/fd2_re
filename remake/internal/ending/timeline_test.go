package ending

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func TestNative2BCE5TimelineIsRecoveredButNotPlayable(t *testing.T) {
	timeline, err := LoadTimeline("../../assets/endings/native_2bce5.json")
	if err != nil {
		t.Fatal(err)
	}
	if timeline.NativeHandler != "0x2bce5" || timeline.Resource.Archive != "FDOTHER.DAT" || timeline.Resource.Index != 0x36 {
		t.Fatalf("timeline header = %#v", timeline)
	}
	if timeline.Ready() {
		t.Fatal("opaque native ending timeline must remain fail-closed")
	}
	if len(timeline.Segments) != 18 {
		t.Fatalf("segment count = %d, want 18", len(timeline.Segments))
	}
	if timeline.Segments[17].Op != "native_post_composite_opaque" {
		t.Fatalf("post-composite gate = %#v", timeline.Segments[17])
	}
	if timeline.Segments[15].FirstFrameFormula != "(i%4)+1" || timeline.Segments[15].SecondFrameFormula != "(i%4)+5" || timeline.Segments[16].FirstFrameFormula != "(i%4)+1" || timeline.Segments[16].SecondFrameFormula != "(i%4)+5" {
		t.Fatalf("composite frame formulas = %#v / %#v", timeline.Segments[15], timeline.Segments[16])
	}
	if timeline.Segments[12].Op != "blit_frame_sequence" || timeline.Segments[15].Op != "native_composite_loop_opaque" || timeline.Segments[16].Op != "native_composite_loop_baseline" {
		t.Fatalf("frame schedule landmarks = %#v", timeline.Segments)
	}
	if frame := timeline.Segments[0].Frame; frame == nil || *frame != 0 || timeline.Segments[0].Target != "offscreen" || timeline.Segments[0].Stride != 320 || timeline.Segments[0].Transparent == nil || *timeline.Segments[0].Transparent != -1 {
		t.Fatalf("first native blit = %#v", timeline.Segments[0])
	}
	ani := timeline.Segments[3]
	if ani.Op != "ani_play" || ani.ANIResource == nil || *ani.ANIResource != 2 || ani.FrameDelayMs != 100 || ani.Skippable == nil || *ani.Skippable {
		t.Fatalf("ANI prefix = %#v", ani)
	}
	blocks := timeline.Segments[9].ElseDialogue
	if len(blocks) != 5 || blocks[0].PortraitID != 37 || blocks[0].SourceDAT != "FDTXT_030" || blocks[0].Script != "ch30.json" || blocks[0].StringIndex != 2 || blocks[0].SceneIndex != 1 || blocks[0].Line != 0 || blocks[0].Count != 6 || blocks[4].StringIndex != 6 || blocks[4].Line != 12 || blocks[4].Count != 1 {
		t.Fatalf("first ending dialogue branch = %#v", blocks)
	}
	if got := []int{blocks[0].NativeUtterances[0].Operand, blocks[0].NativeUtterances[1].Operand, blocks[0].NativeUtterances[2].Operand, blocks[0].NativeUtterances[3].Operand, blocks[0].NativeUtterances[4].Operand, blocks[0].NativeUtterances[5].Operand}; !equalInts(got, []int{4, 24, 126, 0, 126, 122}) {
		t.Fatalf("first ending native speakers = %v", got)
	}
	if got := blocks[0].NativeUtterances[2]; got.ControlSource != "fdtxt" || got.Control != "FFEF" || len(got.Pages) != 2 {
		t.Fatalf("paged ending utterance = %#v", got)
	}
	if got := blocks[4].NativeUtterances[0].Pages; len(got) != 1 || len(got[0]) != 4 {
		t.Fatalf("four-row ending page = %#v", got)
	}
	blocks = timeline.Segments[9].ThenDialogue
	if len(blocks) != 1 || blocks[0].PortraitID != 4 || blocks[0].SourceDAT != "FDTXT_027" || blocks[0].Script != "ch27.json" || blocks[0].StringIndex != 17 || blocks[0].SceneIndex != 3 || blocks[0].Line != 1 || blocks[0].Count != 1 || blocks[0].NativeUtterances[0].ControlSource != "caller_2c39b" || blocks[0].NativeUtterances[0].Operand != 4 {
		t.Fatalf("first bad-ending dialogue branch = %#v", blocks)
	}
	blocks = timeline.Segments[13].ElseDialogue
	if len(blocks) != 1 || blocks[0].PortraitID != 45 || blocks[0].SourceDAT != "FDTXT_030" || blocks[0].Script != "ch30.json" || blocks[0].StringIndex != 7 || blocks[0].SceneIndex != 1 || blocks[0].Line != 13 || blocks[0].Count != 1 || blocks[0].NativeUtterances[0].Operand != 9 {
		t.Fatalf("second ending dialogue branch = %#v", blocks)
	}
	blocks = timeline.Segments[13].ThenDialogue
	if len(blocks) != 3 || blocks[0].StringIndex != 18 || blocks[0].Line != 2 || blocks[2].StringIndex != 20 || blocks[2].Line != 4 {
		t.Fatalf("second bad-ending dialogue branch = %#v", blocks)
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestNative2BCE5DialogueUtterancesMatchOriginalFDTXTWords(t *testing.T) {
	timeline, err := LoadTimeline("../../assets/endings/native_2bce5.json")
	if err != nil {
		t.Fatal(err)
	}
	glyphs := loadEndingOracleGlyphs(t)
	tables := map[string]*fdtxt.Strings{}
	for _, source := range []string{"FDTXT_027", "FDTXT_030"} {
		raw, err := os.ReadFile("../../../extracted/raw/FDTXT/" + source + ".bin")
		if err != nil {
			t.Skip("extracted ending FDTXT oracle is absent")
		}
		tables[source], err = fdtxt.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, segmentIndex := range []int{9, 13} {
		segment := timeline.Segments[segmentIndex]
		blocks := append(append([]DialogueBlock(nil), segment.ThenDialogue...), segment.ElseDialogue...)
		for _, block := range blocks {
			words, err := tables[block.SourceDAT].Words(block.StringIndex)
			if err != nil {
				t.Fatal(err)
			}
			want, err := decodeEndingOracleUtterances(words, block.PortraitID, glyphs)
			if err != nil {
				t.Fatalf("%s index%d: %v", block.SourceDAT, block.StringIndex, err)
			}
			if !reflect.DeepEqual(block.NativeUtterances, want) {
				t.Fatalf("%s index%d native utterances\ngot  %#v\nwant %#v", block.SourceDAT, block.StringIndex, block.NativeUtterances, want)
			}
		}
	}
}

func loadEndingOracleGlyphs(t *testing.T) map[uint16]string {
	t.Helper()
	raw, err := os.ReadFile("../../../docs/data/glyph_map.json")
	if err != nil {
		t.Fatal(err)
	}
	var encoded map[string]json.RawMessage
	if err := json.Unmarshal(raw, &encoded); err != nil {
		t.Fatal(err)
	}
	glyphs := make(map[uint16]string, len(encoded))
	for key, value := range encoded {
		if key == "_comment" {
			continue
		}
		var index int
		var text string
		if _, err := fmt.Sscanf(key, "%d", &index); err != nil || index < 0 || index > 0xffff {
			t.Fatalf("invalid glyph map key %q", key)
		}
		if err := json.Unmarshal(value, &text); err != nil {
			t.Fatal(err)
		}
		glyphs[uint16(index)] = text
	}
	return glyphs
}

func decodeEndingOracleUtterances(words []uint16, callerPortrait int, glyphs map[uint16]string) ([]NativeDialogueUtterance, error) {
	isSpeaker := func(word uint16) bool { return word >= 0xffec && word <= 0xffef }
	hasSpeaker := false
	for _, word := range words {
		hasSpeaker = hasSpeaker || isSpeaker(word)
	}
	decodePages := func(body []uint16) ([][]string, error) {
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
		for _, word := range body {
			switch word {
			case 0xfffe:
				flushRow()
			case 0xfffd:
				flushPage()
			default:
				text, ok := glyphs[word]
				if !ok {
					return nil, fmt.Errorf("glyph %#x is absent from glyph_map", word)
				}
				row += text
			}
		}
		flushPage()
		return pages, nil
	}
	if !hasSpeaker {
		pages, err := decodePages(words)
		if err != nil {
			return nil, err
		}
		return []NativeDialogueUtterance{{ControlSource: "caller_2c39b", Operand: callerPortrait, Pages: pages}}, nil
	}
	var utterances []NativeDialogueUtterance
	for cursor := 0; cursor < len(words); {
		if !isSpeaker(words[cursor]) {
			cursor++
			continue
		}
		if cursor+1 >= len(words) {
			return nil, fmt.Errorf("speaker control at end of string")
		}
		control, operand := words[cursor], words[cursor+1]
		cursor += 2
		start := cursor
		for cursor < len(words) && !isSpeaker(words[cursor]) {
			cursor++
		}
		pages, err := decodePages(words[start:cursor])
		if err != nil {
			return nil, err
		}
		utterances = append(utterances, NativeDialogueUtterance{
			ControlSource: "fdtxt", Control: fmt.Sprintf("%04X", control), Operand: int(operand), Pages: pages,
		})
	}
	return utterances, nil
}
