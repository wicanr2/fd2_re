package campaign

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestCh01PreNativeDialogueLayoutsMatchOriginalControlWords(t *testing.T) {
	raw, err := os.ReadFile("../../../extracted/raw/FDTXT/FDTXT_002.bin")
	if err != nil {
		t.Skip("extracted FDTXT_002 oracle is absent")
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
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch01_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("compile ch01_pre binding: issues=%v err=%v", issues, err)
	}
	byString := map[int][]*NativeDialogueLayout{0: {}, 1: {}, 2: {}, 3: {}}
	for _, beat := range beats {
		if beat.Op == "dialog" && beat.NativeDialogue != nil {
			byString[beat.NativeDialogue.StringIndex] = append(byString[beat.NativeDialogue.StringIndex], beat.NativeDialogue)
		}
	}
	for stringIndex := 0; stringIndex < 4; stringIndex++ {
		words, err := stringsTable.Words(stringIndex)
		if err != nil {
			t.Fatal(err)
		}
		want, err := decodeOriginalNativeDialogueLayouts("FDTXT_002", stringIndex, words, glyphs)
		if err != nil {
			t.Fatal(err)
		}
		got := byString[stringIndex]
		if len(got) != len(want) {
			t.Fatalf("FDTXT_002 index%d layouts=%d, want %d", stringIndex, len(got), len(want))
		}
		for i := range want {
			if !reflect.DeepEqual(got[i], want[i]) {
				t.Fatalf("FDTXT_002 index%d utterance%d\ngot  %#v\nwant %#v", stringIndex, i, got[i], want[i])
			}
		}
	}
}

func TestCh00PreNativeDialogueLayoutsMatchOriginalControlWords(t *testing.T) {
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
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch00_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("compile ch00_pre binding: issues=%v err=%v", issues, err)
	}
	type sourceString struct {
		source string
		index  int
	}
	got := make(map[sourceString][]*NativeDialogueLayout)
	for _, beat := range beats {
		if beat.Op == "dialog" && beat.NativeDialogue != nil {
			key := sourceString{source: beat.NativeDialogue.SourceDAT, index: beat.NativeDialogue.StringIndex}
			got[key] = append(got[key], beat.NativeDialogue)
		}
	}
	total := 0
	for source, count := range map[string]int{"FDTXT_033": 6, "FDTXT_032": 10, "FDTXT_001": 3} {
		raw, err := os.ReadFile("../../../extracted/raw/FDTXT/" + source + ".bin")
		if err != nil {
			t.Skipf("extracted %s oracle is absent", source)
		}
		stringsTable, err := fdtxt.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		for stringIndex := 0; stringIndex < count; stringIndex++ {
			words, err := stringsTable.Words(stringIndex)
			if err != nil {
				t.Fatal(err)
			}
			want, err := decodeOriginalNativeDialogueLayouts(source, stringIndex, words, glyphs)
			if err != nil {
				t.Fatal(err)
			}
			layouts := got[sourceString{source: source, index: stringIndex}]
			if len(layouts) != len(want) {
				t.Fatalf("%s index%d layouts=%d, want %d", source, stringIndex, len(layouts), len(want))
			}
			for i := range want {
				if !reflect.DeepEqual(layouts[i], want[i]) {
					t.Fatalf("%s index%d utterance%d\ngot  %#v\nwant %#v", source, stringIndex, i, layouts[i], want[i])
				}
			}
			total += len(want)
		}
	}
	if total != 97 {
		t.Fatalf("ch00 native dialogue utterances=%d, want 97", total)
	}
}

func TestCh00PostNativeDialogueLayoutsMatchOriginalControlWords(t *testing.T) {
	raw, err := os.ReadFile("../../../extracted/raw/FDTXT/FDTXT_001.bin")
	if err != nil {
		t.Skip("extracted FDTXT_001 oracle is absent")
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
		var index int
		if err := json.Unmarshal(value, &text); err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Sscanf(key, "%d", &index); err != nil || index < 0 || index > 0xffff {
			t.Fatalf("invalid glyph map key %q", key)
		}
		glyphs[uint16(index)] = text
	}
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch00_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("compile ch00_post binding: issues=%v err=%v", issues, err)
	}
	var got []*NativeDialogueLayout
	for _, beat := range beats {
		if beat.Op == "dialog" && beat.NativeDialogue != nil {
			got = append(got, beat.NativeDialogue)
		}
	}
	words, err := stringsTable.Words(9)
	if err != nil {
		t.Fatal(err)
	}
	want, err := decodeOriginalNativeDialogueLayouts("FDTXT_001", 9, words, glyphs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 13 || !reflect.DeepEqual(got, want) {
		t.Fatalf("FDTXT_001 index9 layouts=%d want=%d\ngot  %#v\nwant %#v", len(got), len(want), got, want)
	}
}

func TestCh22PostNativeDialogueLayoutsMatchOriginalControlWords(t *testing.T) {
	raw, err := os.ReadFile("../../../extracted/raw/FDTXT/FDTXT_023.bin")
	if err != nil {
		t.Skip("extracted FDTXT_023 oracle is absent")
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
		var index int
		if err := json.Unmarshal(value, &text); err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Sscanf(key, "%d", &index); err != nil || index < 0 || index > 0xffff {
			t.Fatalf("invalid glyph map key %q", key)
		}
		glyphs[uint16(index)] = text
	}
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch22_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("compile ch22_post binding: issues=%v err=%v", issues, err)
	}
	got := make(map[int][]*NativeDialogueLayout, 10)
	var collect func([]Beat)
	collect = func(items []Beat) {
		for _, beat := range items {
			if beat.Op == "dialog" && beat.NativeDialogue != nil {
				index := beat.NativeDialogue.StringIndex
				got[index] = append(got[index], beat.NativeDialogue)
			}
			collect(beat.Then)
			collect(beat.Else)
		}
	}
	collect(beats)
	total := 0
	for stringIndex := 8; stringIndex <= 17; stringIndex++ {
		words, err := stringsTable.Words(stringIndex)
		if err != nil {
			t.Fatal(err)
		}
		want, err := decodeOriginalNativeDialogueLayouts("FDTXT_023", stringIndex, words, glyphs)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got[stringIndex], want) {
			t.Fatalf("FDTXT_023 index%d layouts=%d want=%d\ngot  %#v\nwant %#v",
				stringIndex, len(got[stringIndex]), len(want), got[stringIndex], want)
		}
		total += len(want)
	}
	if total != 56 {
		t.Fatalf("ch22_post native dialogue utterances=%d, want 56", total)
	}
}

func TestCh17PostNativeDialogueLayoutsMatchOriginalControlWords(t *testing.T) {
	raw, err := os.ReadFile("../../../extracted/raw/FDTXT/FDTXT_018.bin")
	if err != nil {
		t.Skip("extracted FDTXT_018 oracle is absent")
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
		var index int
		if err := json.Unmarshal(value, &text); err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Sscanf(key, "%d", &index); err != nil || index < 0 || index > 0xffff {
			t.Fatalf("invalid glyph map key %q", key)
		}
		glyphs[uint16(index)] = text
	}
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch17_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("compile ch17_post binding: issues=%v err=%v", issues, err)
	}
	got := make(map[int][]*NativeDialogueLayout, 4)
	for _, beat := range beats {
		if beat.Op == "dialog" && beat.NativeDialogue != nil {
			got[beat.NativeDialogue.StringIndex] = append(got[beat.NativeDialogue.StringIndex], beat.NativeDialogue)
		}
	}
	total := 0
	for stringIndex := 7; stringIndex <= 10; stringIndex++ {
		words, err := stringsTable.Words(stringIndex)
		if err != nil {
			t.Fatal(err)
		}
		want, err := decodeOriginalNativeDialogueLayouts("FDTXT_018", stringIndex, words, glyphs)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got[stringIndex], want) {
			t.Fatalf("FDTXT_018 index%d layouts=%d want=%d\ngot  %#v\nwant %#v",
				stringIndex, len(got[stringIndex]), len(want), got[stringIndex], want)
		}
		total += len(want)
	}
	if total != 21 {
		t.Fatalf("ch17_post native dialogue utterances=%d, want 21", total)
	}
}

func TestCh12PostNativeDialogueLayoutsMatchOriginalControlWords(t *testing.T) {
	raw, err := os.ReadFile("../../../extracted/raw/FDTXT/FDTXT_013.bin")
	if err != nil {
		t.Skip("extracted FDTXT_013 oracle is absent")
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
		var index int
		if err := json.Unmarshal(value, &text); err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Sscanf(key, "%d", &index); err != nil || index < 0 || index > 0xffff {
			t.Fatalf("invalid glyph map key %q", key)
		}
		glyphs[uint16(index)] = text
	}
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch12_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("compile ch12_post binding: issues=%v err=%v", issues, err)
	}
	var got []*NativeDialogueLayout
	for _, beat := range beats {
		if beat.Op == "dialog" && beat.NativeDialogue != nil {
			got = append(got, beat.NativeDialogue)
		}
	}
	words, err := stringsTable.Words(9)
	if err != nil {
		t.Fatal(err)
	}
	want, err := decodeOriginalNativeDialogueLayouts("FDTXT_013", 9, words, glyphs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 12 || !reflect.DeepEqual(got, want) {
		t.Fatalf("FDTXT_013 index9 layouts=%d want=%d\ngot  %#v\nwant %#v", len(got), len(want), got, want)
	}
}

func TestCh19PostNativeDialogueLayoutsMatchOriginalControlWords(t *testing.T) {
	raw, err := os.ReadFile("../../../extracted/raw/FDTXT/FDTXT_020.bin")
	if err != nil {
		t.Skip("extracted FDTXT_020 oracle is absent")
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
		var index int
		if err := json.Unmarshal(value, &text); err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Sscanf(key, "%d", &index); err != nil || index < 0 || index > 0xffff {
			t.Fatalf("invalid glyph map key %q", key)
		}
		glyphs[uint16(index)] = text
	}
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch19_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("compile ch19_post binding: issues=%v err=%v", issues, err)
	}
	got := make(map[int][]*NativeDialogueLayout, 6)
	var collect func([]Beat)
	collect = func(items []Beat) {
		for _, beat := range items {
			if beat.Op == "dialog" && beat.NativeDialogue != nil {
				got[beat.NativeDialogue.StringIndex] = append(got[beat.NativeDialogue.StringIndex], beat.NativeDialogue)
			}
			collect(beat.Then)
			collect(beat.Else)
		}
	}
	collect(beats)
	total := 0
	for stringIndex := 11; stringIndex <= 16; stringIndex++ {
		words, err := stringsTable.Words(stringIndex)
		if err != nil {
			t.Fatal(err)
		}
		want, err := decodeOriginalNativeDialogueLayouts("FDTXT_020", stringIndex, words, glyphs)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got[stringIndex], want) {
			t.Fatalf("FDTXT_020 index%d layouts=%d want=%d\ngot  %#v\nwant %#v",
				stringIndex, len(got[stringIndex]), len(want), got[stringIndex], want)
		}
		total += len(want)
	}
	if total != 29 {
		t.Fatalf("ch19_post native dialogue utterances=%d, want 29", total)
	}
}

func TestCh07PostNativeDialogueLayoutsMatchOriginalControlWords(t *testing.T) {
	raw, err := os.ReadFile("../../../extracted/raw/FDTXT/FDTXT_008.bin")
	if err != nil {
		t.Skip("extracted FDTXT_008 oracle is absent")
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
		var index int
		if err := json.Unmarshal(value, &text); err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Sscanf(key, "%d", &index); err != nil || index < 0 || index > 0xffff {
			t.Fatalf("invalid glyph map key %q", key)
		}
		glyphs[uint16(index)] = text
	}
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch07_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("compile ch07_post binding: issues=%v err=%v", issues, err)
	}
	got := make(map[int][]*NativeDialogueLayout, 2)
	for _, beat := range beats {
		if beat.Op == "dialog" && beat.NativeDialogue != nil {
			got[beat.NativeDialogue.StringIndex] = append(got[beat.NativeDialogue.StringIndex], beat.NativeDialogue)
		}
	}
	total := 0
	for stringIndex := 3; stringIndex <= 4; stringIndex++ {
		words, err := stringsTable.Words(stringIndex)
		if err != nil {
			t.Fatal(err)
		}
		want, err := decodeOriginalNativeDialogueLayouts("FDTXT_008", stringIndex, words, glyphs)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got[stringIndex], want) {
			t.Fatalf("FDTXT_008 index%d layouts=%d want=%d\ngot  %#v\nwant %#v",
				stringIndex, len(got[stringIndex]), len(want), got[stringIndex], want)
		}
		total += len(want)
	}
	if total != 8 {
		t.Fatalf("ch07_post native dialogue utterances=%d, want 8", total)
	}
}

func TestCh06PostNativeDialogueLayoutsMatchOriginalControlWords(t *testing.T) {
	raw, err := os.ReadFile("../../../extracted/raw/FDTXT/FDTXT_007.bin")
	if err != nil {
		t.Skip("extracted FDTXT_007 oracle is absent")
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
		var index int
		if err := json.Unmarshal(value, &text); err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Sscanf(key, "%d", &index); err != nil || index < 0 || index > 0xffff {
			t.Fatalf("invalid glyph map key %q", key)
		}
		glyphs[uint16(index)] = text
	}
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch06_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("compile ch06_post binding: issues=%v err=%v", issues, err)
	}
	got := make(map[int][]*NativeDialogueLayout, 2)
	seen := make(map[string]bool, 12)
	callers := make(map[string]bool, 2)
	var collect func([]Beat)
	collect = func(items []Beat) {
		for _, beat := range items {
			if beat.Op == "dialog" && beat.NativeDialogue != nil {
				caller := fmt.Sprintf("%s#%d", beat.Source, beat.NativeDialogue.StringIndex)
				callers[caller] = true
				key := fmt.Sprintf("%s#%d", caller, beat.NativeDialogue.Utterance)
				if !seen[key] {
					seen[key] = true
					got[beat.NativeDialogue.StringIndex] = append(got[beat.NativeDialogue.StringIndex], beat.NativeDialogue)
				}
			}
			collect(beat.Then)
			collect(beat.Else)
		}
	}
	collect(beats)
	total := 0
	for stringIndex := 4; stringIndex <= 5; stringIndex++ {
		words, err := stringsTable.Words(stringIndex)
		if err != nil {
			t.Fatal(err)
		}
		want, err := decodeOriginalNativeDialogueLayouts("FDTXT_007", stringIndex, words, glyphs)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got[stringIndex], want) {
			t.Fatalf("FDTXT_007 index%d layouts=%d want=%d\ngot  %#v\nwant %#v",
				stringIndex, len(got[stringIndex]), len(want), got[stringIndex], want)
		}
		total += len(want)
	}
	if len(callers) != 2 || len(seen) != 12 || total != 12 {
		t.Fatalf("ch06_post native dialogue callers=%d layouts=%d utterances=%d, want 2/12/12", len(callers), len(seen), total)
	}
}

func TestNativeDialogueLayoutUsesControlSpecificOriginalLineLimits(t *testing.T) {
	limits := map[string]int{"FFEC": 13, "FFED": 15, "FFEE": 14, "FFEF": 14}
	for control, limit := range limits {
		t.Run(control, func(t *testing.T) {
			tokens := make([]string, limit)
			for i := range tokens {
				tokens[i] = "甲"
			}
			layout := &NativeDialogueLayout{
				SourceDAT: "FDTXT_TEST", Control: control,
				Pages: [][]string{{strings.Join(tokens, "")}}, GlyphPages: [][][]string{{tokens}},
			}
			if err := layout.Validate(); err != nil {
				t.Fatalf("exact original limit %d rejected: %v", limit, err)
			}
			layout.Pages[0][0] += "乙"
			layout.GlyphPages[0][0] = append(layout.GlyphPages[0][0], "乙")
			if err := layout.Validate(); err == nil {
				t.Fatalf("control %s accepted %d glyphs above limit %d", control, limit+1, limit)
			}
		})
	}
}

func TestNativeDialogueLayoutPreservesMultiRuneOriginalGlyphTokens(t *testing.T) {
	layout := &NativeDialogueLayout{
		SourceDAT: "FDTXT_029", StringIndex: 12, Control: "FFEF", Operand: 126,
		Pages:      [][]string{{"『ASR-07，妳的系統真的受損"}},
		GlyphPages: [][][]string{{{"『", "AS", "R-", "07", "，", "妳", "的", "系", "統", "真", "的", "受", "損"}}},
	}
	if err := layout.Validate(); err != nil {
		t.Fatalf("13 raw glyphs expanded to 16 Unicode runes were rejected: %v", err)
	}
	tokens, err := layout.glyphTokens(0, 0, layout.Pages[0][0])
	if err != nil || len(tokens) != 13 {
		t.Fatalf("multi-rune glyph tokens=%v err=%v", tokens, err)
	}
	layout.GlyphPages[0][0][1] = "A"
	if err := layout.Validate(); err == nil {
		t.Fatal("glyph tokens that no longer join to authored text were accepted")
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
		var glyphPages [][][]string
		var page []string
		var glyphPage [][]string
		var row []string
		flushRow := func() {
			if len(row) != 0 {
				page = append(page, strings.Join(row, ""))
				glyphPage = append(glyphPage, row)
				row = nil
			}
		}
		flushPage := func() {
			flushRow()
			if len(page) > 0 {
				pages = append(pages, page)
				glyphPages = append(glyphPages, glyphPage)
				page = nil
				glyphPage = nil
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
				row = append(row, text)
			}
		}
		flushPage()
		layout.Pages = pages
		multiRune := false
		for _, glyphPage := range glyphPages {
			for _, glyphRow := range glyphPage {
				for _, token := range glyphRow {
					multiRune = multiRune || len([]rune(token)) != 1
				}
			}
		}
		if multiRune {
			layout.GlyphPages = glyphPages
		}
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
	for index := 0; index <= 19; index++ {
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
	if err != nil || len(portraits) < 4 {
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
	page0BeforeMouth := append([]byte(nil), page0...)
	mouth, err := ComposeNativeStoryDialogueMouthFrame(page0, portraits[3], layout)
	if err != nil {
		t.Fatal(err)
	}
	if len(mouth) != 320*200 || string(mouth) == string(page0) {
		t.Fatal("DATO frame3 did not produce a distinct indexed mouth frame")
	}
	if string(page0) != string(page0BeforeMouth) {
		t.Fatal("mouth compositor mutated its caller-owned source")
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
	opening, err := ComposeNativeStoryDialogueOpeningFrames(background, cells, layout)
	if err != nil {
		t.Fatal(err)
	}
	if len(opening) != len(nativeStoryOpeningGridSizes) {
		t.Fatalf("opening frames=%d, want %d", len(opening), len(nativeStoryOpeningGridSizes))
	}
	for i := 1; i < len(opening); i++ {
		if string(opening[i-1]) == string(opening[i]) {
			t.Fatalf("opening grid stage %d did not expand", i)
		}
	}
	if background[nativeStoryUpperFrameY*320+5] != 7 {
		t.Fatal("native story opening mutated its caller-owned background")
	}
	closing, err := ComposeNativeStoryDialogueClosingFrames(background, cells, layout, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(closing) != 5 || string(closing[0]) != string(opening[3]) ||
		string(closing[1]) != string(opening[2]) || string(closing[2]) != string(opening[1]) ||
		string(closing[3]) != string(opening[0]) || string(closing[4]) != string(background) {
		t.Fatal("sub_16B43 snapshot restore order differs from 16x5..background")
	}
	motion, err := ComposeNativeStoryDialogueClosingFrames(background, cells, layout, 2, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(motion) != 5+2+3+1 || string(motion[len(motion)-1]) == string(background) {
		t.Fatalf("closing motion frames=%d or final cursor overlay is missing", len(motion))
	}
	bad := *layout
	bad.Pages = [][]string{{"🙂"}}
	if _, err := ComposeNativeStoryDialoguePage(background, cells, portraits[0], font, index, &bad, 0); err == nil {
		t.Fatal("unknown editable glyph did not fail closed")
	}
}
