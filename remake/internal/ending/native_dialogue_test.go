package ending

import (
	"bytes"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func TestNativeDialogueUsesSixBandOpeningAndFiveBandClosingRestore(t *testing.T) {
	background := make([]byte, endingDialogueBytes)
	base := make([]byte, endingDialogueBytes)
	for y := 112; y < 198; y++ {
		for x := 5; x < 315; x++ {
			base[y*320+x] = byte(y - 111)
		}
	}
	opening, err := ComposeNativeDialogueOpeningFrames(background, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(opening) != 6 {
		t.Fatalf("opening frames=%d, want 6", len(opening))
	}
	for i, y := range []int{177, 164, 151, 138, 125, 112} {
		if opening[i][y*320+5] != 1 {
			t.Fatalf("opening frame%d did not start at y=%d", i, y)
		}
	}
	closing, err := ComposeNativeDialogueClosingFrames(background, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(closing) != 6 || !bytes.Equal(closing[5], background) {
		t.Fatalf("closing frames=%d or source restore mismatch", len(closing))
	}
	for i, y := range []int{125, 138, 151, 164, 177} {
		if closing[i][y*320+5] != 1 {
			t.Fatalf("closing frame%d did not start at y=%d", i, y)
		}
	}
}

func TestNativeDialogueProgressiveFramesAllowFourRowsAndPublishOneGlyph(t *testing.T) {
	fontRaw := make([]byte, 2*fdtxt.GlyphBytes)
	for row := 0; row < fdtxt.GlyphHeight; row++ {
		fontRaw[row*2] = 0x80
		fontRaw[fdtxt.GlyphBytes+row*2] = 0x40
	}
	font, err := fdtxt.ParseFont(fontRaw)
	if err != nil {
		t.Fatal(err)
	}
	pages := [][]string{{"甲", "乙", "甲", "乙"}}
	frames, err := ComposeNativeDialogueProgressiveFrames(
		make([]byte, endingDialogueBytes), font, map[string]int{"甲": 0, "乙": 1}, pages, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 5 {
		t.Fatalf("progressive frames=%d, want frame0 plus four glyphs", len(frames))
	}
	for i := 1; i < len(frames); i++ {
		if bytes.Equal(frames[i-1], frames[i]) {
			t.Fatalf("glyph frame%d did not change indexed output", i)
		}
	}
	if frames[4][endingDialogueTextOffset+3*endingDialogueLineStep+1] != 0xcd {
		t.Fatal("fourth ending row was not rendered at the proven line step")
	}
}

func TestNativeDialogueBaseUsesRightEdgePortraitAndFailsClosed(t *testing.T) {
	background := make([]byte, endingDialogueBytes)
	cells := make([]fdother.RawCell, 18)
	for i := range cells {
		cells[i] = fdother.RawCell{Width: 1, Height: 1, Pixels: []byte{byte(i + 1)}}
	}
	portrait := dato.Frame{Width: 2, Height: 1, Pixels: []byte{0x21, 0x22}}
	base, err := ComposeNativeDialogueBase(background, cells, portrait)
	if err != nil {
		t.Fatal(err)
	}
	if base[endingDialoguePortraitAnchor] != 0x21 || base[endingDialoguePortraitAnchor-1] != 0x22 {
		t.Fatalf("right-edge portrait bytes=%#x/%#x", base[endingDialoguePortraitAnchor], base[endingDialoguePortraitAnchor-1])
	}
	if _, err := ComposeNativeDialogueBase(background, cells[:17], portrait); err == nil {
		t.Fatal("incomplete FDOTHER #5 cells were accepted")
	}
}

func TestNativeDialoguePlaybackPreservesBlockPageUtteranceAndClosingGates(t *testing.T) {
	frame := func(value byte) []byte {
		out := make([]byte, endingDialogueBytes)
		out[0] = value
		return out
	}
	block := func(base byte, pages int) NativeDialogueBlockFrames {
		opening := make([][]byte, 6)
		closing := make([][]byte, 6)
		for i := range opening {
			opening[i], closing[i] = frame(base+byte(i)), frame(base+20+byte(i))
		}
		pageFrames := make([][][]byte, pages)
		mouth := make([][]byte, pages)
		for i := range pageFrames {
			pageFrames[i] = [][]byte{frame(base + 40 + byte(i)), frame(base + 50 + byte(i))}
			mouth[i] = frame(base + 60 + byte(i))
		}
		return NativeDialogueBlockFrames{Opening: opening, Utterances: []NativeDialogueUtteranceFrames{{Pages: pageFrames, MouthOpen: mouth}, {Pages: [][][]byte{{frame(base + 70)}}, MouthOpen: [][]byte{frame(base + 71)}}}, Closing: closing}
	}
	p, err := NewNativeDialoguePlayback([]NativeDialogueBlockFrames{block(1, 2), block(100, 1)})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		if got := p.CurrentFrame(false)[0]; got != byte(1+i) {
			t.Fatalf("opening frame%d=%d", i, got)
		}
		p.Step()
	}
	for !p.Waiting() {
		p.Step()
	}
	if p.CurrentFrame(true)[0] != 61 || !p.Confirm() {
		t.Fatal("first page did not expose mouth frame or accept confirm")
	}
	for !p.Waiting() {
		p.Step()
	}
	if p.Page != 1 || !p.Confirm() {
		t.Fatalf("second page state=%#v", p)
	}
	for !p.Waiting() {
		p.Step()
	}
	if p.Utterance != 1 || !p.Confirm() || p.Phase != NativeDialogueClosing {
		t.Fatalf("second utterance did not enter closing: %#v", p)
	}
	for i := 0; i < 6; i++ {
		p.Step()
	}
	if p.Block != 1 || p.Phase != NativeDialogueOpening {
		t.Fatalf("next block state=%#v", p)
	}
	for !p.Waiting() {
		p.Step()
	}
	if !p.Confirm() {
		t.Fatal("final block first utterance did not confirm")
	}
	for !p.Waiting() {
		p.Step()
	}
	if !p.Confirm() {
		t.Fatal("final block second utterance did not confirm")
	}
	for !p.Done() {
		p.Step()
	}
	if p.CurrentFrame(false) != nil {
		t.Fatal("completed playback retained a visible frame")
	}
}
