package fdtxt

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestLoadSeparatedResourceRebuildsLosslessWords(t *testing.T) {
	root := t.TempDir()
	glyph := 557
	document := separatedResource{
		SchemaVersion: 1, Kind: "fdtxt_word_table", AssetID: "text/FDTXT_031",
		Status: "decoded", Evidence: "confirmed",
		Source: separatedSource{File: "FDTXT.DAT", Resource: 31, Size: fdtxtSourceSize,
			MD5: fdtxtSourceMD5, SHA256: fdtxtSourceSHA256, RawSize: 8},
		Strings: []separatedString{{
			StringID: "FDTXT_031/string_0000", SourceIndex: 0, Text: "『",
			Tokens: []separatedToken{
				{Kind: "control", Control: "FFEF"},
				{Kind: "glyph", GlyphIndex: &glyph, Text: "『"},
			},
		}},
	}
	directory := filepath.Join(root, "FDTXT_031")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "resource.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	strings, err := LoadSeparatedResource(root, 31)
	if err != nil {
		t.Fatal(err)
	}
	words, err := strings.Words(0)
	if err != nil || !bytes.Equal(wordsToBytes(words), []byte{0xef, 0xff, 0x2d, 0x02}) {
		t.Fatalf("words=%#v err=%v", words, err)
	}
}

func TestLoadSeparatedResourceRejectsStaleUTF8Projection(t *testing.T) {
	root := t.TempDir()
	glyph := 557
	document := separatedResource{
		SchemaVersion: 1, Kind: "fdtxt_word_table", AssetID: "text/FDTXT_000",
		Status: "decoded", Evidence: "confirmed",
		Source: separatedSource{File: "FDTXT.DAT", Resource: 0, Size: fdtxtSourceSize,
			MD5: fdtxtSourceMD5, SHA256: fdtxtSourceSHA256, RawSize: 4},
		Strings: []separatedString{{
			StringID: "FDTXT_000/string_0000", SourceIndex: 0, Text: "過期",
			Tokens: []separatedToken{{Kind: "glyph", GlyphIndex: &glyph, Text: "『"}},
		}},
	}
	directory := filepath.Join(root, "FDTXT_000")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(document)
	if err := os.WriteFile(filepath.Join(directory, "resource.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSeparatedResource(root, 0); err == nil {
		t.Fatal("stale UTF-8 projection was accepted")
	}
}

func TestSeparatedResource31MatchesOriginalWords(t *testing.T) {
	const (
		rawPath  = "../../../extracted/raw/FDTXT/FDTXT_031.bin"
		textRoot = "../../generated-assets/fd2-original-b97caf22/text"
	)
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		t.Skip("player-provided FDTXT_031 is absent")
	}
	original, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	separated, err := LoadSeparatedResource(textRoot, 31)
	if err != nil {
		t.Skipf("separated FDTXT_031 is absent: %v", err)
	}
	if separated.Count() != original.Count() {
		t.Fatalf("string count=%d, want %d", separated.Count(), original.Count())
	}
	for index := 0; index < original.Count(); index++ {
		want, _ := original.Words(index)
		got, _ := separated.Words(index)
		if !bytes.Equal(wordsToBytes(got), wordsToBytes(want)) {
			t.Fatalf("string %d words differ", index)
		}
	}
}

func TestSeparatedFontMatchesOriginalBits(t *testing.T) {
	const (
		rawPath  = "../../../extracted/raw/FDOTHER/FDOTHER_004.bin"
		fontRoot = "../../generated-assets/fd2-original-b97caf22/fonts"
	)
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		t.Skip("player-provided FDOTHER_004 is absent")
	}
	original, err := ParseFont(raw)
	if err != nil {
		t.Fatal(err)
	}
	separated, err := LoadSeparatedFont(fontRoot)
	if err != nil {
		t.Skipf("separated FDOTHER_004 font is absent: %v", err)
	}
	if separated.GlyphCount() != original.GlyphCount() {
		t.Fatalf("glyph count=%d, want %d", separated.GlyphCount(), original.GlyphCount())
	}
	for glyph := 0; glyph < original.GlyphCount(); glyph++ {
		for y := 0; y < GlyphHeight; y++ {
			for x := 0; x < GlyphWidth; x++ {
				want, _ := original.GlyphBit(glyph, x, y)
				got, err := separated.GlyphBit(glyph, x, y)
				if err != nil || got != want {
					t.Fatalf("glyph=%d x=%d y=%d got=%v want=%v err=%v", glyph, x, y, got, want, err)
				}
			}
		}
	}
}

func wordsToBytes(words []uint16) []byte {
	out := make([]byte, 0, len(words)*2)
	for _, word := range words {
		out = append(out, byte(word), byte(word>>8))
	}
	return out
}

func TestParseRetainsControlWordsAndStopsAtTerminator(t *testing.T) {
	// Two strings: offsets 4 and 10. The first ends before its span does.
	data := []byte{4, 0, 10, 0, 0x12, 0, 0xfe, 0xff, 0xff, 0xff, 0x34, 0, 0xfd, 0xff, 0xff, 0xff}
	s, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if s.Count() != 2 {
		t.Fatalf("count=%d", s.Count())
	}
	first, _ := s.Words(0)
	second, _ := s.Words(1)
	if len(first) != 2 || first[0] != 0x12 || first[1] != 0xfffe || len(second) != 2 || second[0] != 0x34 || second[1] != 0xfffd {
		t.Fatalf("words=%#v / %#v", first, second)
	}
}

func TestFontGlyphBitUsesMSBLeftOrder(t *testing.T) {
	data := make([]byte, GlyphBytes)
	data[0], data[1] = 0x80, 0x01
	f, err := ParseFont(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, point := range []struct{ x, y int }{{0, 0}, {15, 0}} {
		got, err := f.GlyphBit(0, point.x, point.y)
		if err != nil || !got {
			t.Fatalf("bit(%d,%d)=%v err=%v", point.x, point.y, got, err)
		}
	}
	if got, _ := f.GlyphBit(0, 1, 0); got {
		t.Fatal("unexpected set bit")
	}
}

func TestBlitGlyphLeavesZeroBitsTransparent(t *testing.T) {
	data := make([]byte, GlyphBytes)
	data[0], data[1] = 0x80, 0x01
	f, err := ParseFont(data)
	if err != nil {
		t.Fatal(err)
	}
	dst := make([]byte, 20*16)
	for i := range dst {
		dst[i] = 7
	}
	if err := f.BlitGlyph(dst, 20, 2, 0, 42); err != nil {
		t.Fatal(err)
	}
	if dst[2] != 42 || dst[17] != 42 || dst[3] != 7 || dst[22] != 7 {
		t.Fatalf("glyph pixels = %v %v %v %v", dst[2], dst[17], dst[3], dst[22])
	}
}

func TestBlitNativeGlyphMatchesForegroundShadowAndBackgroundABI(t *testing.T) {
	data := make([]byte, GlyphBytes)
	data[0] = 0x80 // top-left source bit
	f, err := ParseFont(data)
	if err != nil {
		t.Fatal(err)
	}
	dst := make([]byte, 20*18)
	for i := range dst {
		dst[i] = 7
	}
	base := 21
	if err := f.BlitNativeGlyph(dst, 20, base, 0, NativeGlyphStyle{Foreground: 0xcd, Shadow: 0x4c, Background: 3}); err != nil {
		t.Fatal(err)
	}
	if dst[base] != 0xcd || dst[base+20-1] != 0x4c || dst[base+20] != 0x4c || dst[base-1] != 7 || dst[base+1] != 3 || dst[base+15*20+15] != 3 {
		t.Fatalf(
			"native glyph pixels=%#x %#x %#x %#x %#x %#x",
			dst[base], dst[base+20-1], dst[base+20],
			dst[base-1], dst[base+1], dst[base+15*20+15],
		)
	}
}

func TestBlitNativeGlyphAdjacentForegroundIsNotOverwrittenByShadow(t *testing.T) {
	data := make([]byte, GlyphBytes)
	data[0] = 0xc0 // adjacent x=0 and x=1 source bits
	f, err := ParseFont(data)
	if err != nil {
		t.Fatal(err)
	}
	dst := make([]byte, 20*18)
	base := 21
	if err := f.BlitNativeGlyph(
		dst, 20, base, 0,
		NativeGlyphStyle{Foreground: 0xcd, Shadow: 0x4c},
	); err != nil {
		t.Fatal(err)
	}
	if dst[base] != 0xcd || dst[base+1] != 0xcd {
		t.Fatalf(
			"adjacent foreground overwritten: %#x %#x",
			dst[base], dst[base+1],
		)
	}
	for _, pos := range []int{base + 20 - 1, base + 20, base + 20 + 1} {
		if dst[pos] != 0x4c {
			t.Fatalf("shadow at %d=%#x", pos, dst[pos])
		}
	}
}

func TestLocateLogicalUtteranceUsesPhysicalLocator(t *testing.T) {
	// Each FFxx word terminates a text chunk.  The second glyph 557 is the
	// original opening quote and therefore marks two visible utterances.
	s := &Strings{words: [][]uint16{{9, OpeningGlyph, 1, 0xfffe, 9, OpeningGlyph, 2, 0xffff}}}
	got, err := s.LocateLogicalUtterance(1)
	if err != nil || got != (LogicalLocator{RawStringIndex: 0, RawUtteranceIndex: 1}) {
		t.Fatalf("locator=%#v err=%v", got, err)
	}
}

func TestPlayerFDTXT031Physical44HasStablePayload(t *testing.T) {
	const path = "../../../extracted/raw/FDTXT/FDTXT_031.bin"
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Skip("player-provided FDTXT_031 is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	s, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if s.Count() != 46 {
		t.Fatalf("physical strings=%d, want 46", s.Count())
	}
	words, err := s.Words(44)
	if err != nil || len(words) != 130 || words[0] != 0x00b5 || words[len(words)-1] != 0x0248 {
		t.Fatalf("physical #44 words=%d first/last=%#x/%#x err=%v", len(words), words[0], words[len(words)-1], err)
	}
}

func TestChapterLoader30UsesFDTXTArchiveResource31(t *testing.T) {
	const (
		archivePath = "../../../org_game/炎龍騎士團/FLAME2/FDTXT.DAT"
		rawPath     = "../../../extracted/raw/FDTXT/FDTXT_031.bin"
	)
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Skip("player-provided FDTXT.DAT is absent")
	}
	entry, err := fdother.ReadResource(archivePath, 31) // 0x1088d increments chapter 30.
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(rawPath)
	if os.IsNotExist(err) {
		t.Skip("extracted FDTXT_031 oracle is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(entry, raw) {
		t.Fatalf("FDTXT.DAT resource 31 does not equal FDTXT_031: %d/%d bytes", len(entry), len(raw))
	}
}
