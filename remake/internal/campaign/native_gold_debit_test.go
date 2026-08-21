package campaign

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func nativeGoldTestStrip() fdother.RawCell {
	pixels := make([]byte, 6*99)
	for row := 0; row < 99; row++ {
		for col := 0; col < 6; col++ {
			pixels[row*6+col] = byte(row + 1)
		}
	}
	return fdother.RawCell{Width: 6, Height: 99, Pixels: pixels}
}

func TestComposeNativeGoldDebitFramesPreservesOdometerSchedule(t *testing.T) {
	base := make([]byte, NativeShopWidth*NativeShopHeight)
	frames, next, err := ComposeNativeGoldDebitFrames(
		base, nativeGoldTestStrip(), 1000, 50,
	)
	if err != nil {
		t.Fatal(err)
	}
	// 00001000 -> 00000950: the slowest digit descends five values, with
	// nine vertical-strip phases per value.
	if next != 950 || len(frames) != 45 {
		t.Fatalf("next=%d frames=%d, want 950/45", next, len(frames))
	}
	// Hundreds and tens both roll on the first pass. The first phase begins
	// at row digit*9+8; the ninth finishes at the stable digit row.
	if got := frames[0][NativeShopGoldRollOffset+4*6]; got != 9 {
		t.Fatalf("first hundreds roll row=%d, want strip row8 value9", got)
	}
	if got := frames[8][NativeShopGoldRollOffset+4*6]; got != 1 {
		t.Fatalf("final hundreds row=%d, want strip row0 value1", got)
	}
	if got := frames[8][NativeShopGoldRollOffset+5*6]; got != 82 {
		t.Fatalf("final tens row=%d, want strip row81 value82", got)
	}
	if got := frames[0][NativeShopGoldRollOffset+5*6]; got != 90 {
		t.Fatalf("wrapped tens first row=%d, want strip row89 value90", got)
	}
	if got := frames[9][NativeShopGoldRollOffset+6*6]; got != 81 {
		t.Fatalf("second roll first row=%d, want strip row80 value81", got)
	}
	// The units digit never differs and must preserve its background.
	if got := frames[44][NativeShopGoldRollOffset+7*6]; got != 0 {
		t.Fatalf("unchanged units roll pixel=%d, want preserved 0", got)
	}
	for index, got := range frames {
		if got[NativeShopGoldRollOffset-1] != 0 ||
			got[NativeShopGoldRollOffset+8*6] != 0 {
			t.Fatalf("frame%d wrote outside eight digit windows", index)
		}
	}
	// The animated strip starts one VGA row above the stable digit baseline:
	// 0xa7a90 -> (16,98), while NativeShopGoldOffset is (16,99).
	if NativeShopGoldOffset-NativeShopGoldRollOffset != NativeShopWidth {
		t.Fatal("native stable/rolling gold offsets lost their one-row boundary")
	}
	cascade, next, err := ComposeNativeGoldDebitFrames(
		base, nativeGoldTestStrip(), 10_000_000, 1,
	)
	if err != nil || next != 9_999_999 || len(cascade) != 9 {
		t.Fatalf(
			"borrow cascade = frames%d next%d err%v, want 9/9999999",
			len(cascade), next, err,
		)
	}
}

func TestComposeNativeGoldCreditFramesPreservesOdometerSchedule(t *testing.T) {
	base := make([]byte, NativeShopWidth*NativeShopHeight)
	frames, next, err := ComposeNativeGoldCreditFrames(
		base, nativeGoldTestStrip(), 0, 37,
	)
	if err != nil {
		t.Fatal(err)
	}
	// 00000000 -> 00000037: tens and units rise together to 33, then
	// only units continue. Seven value steps, each with nine strip phases.
	if next != 37 || len(frames) != 63 {
		t.Fatalf("next=%d frames=%d, want 37/63", next, len(frames))
	}
	if got := frames[0][NativeShopGoldRollOffset+6*6]; got != 2 {
		t.Fatalf("first tens roll row=%d, want strip row1 value2", got)
	}
	if got := frames[8][NativeShopGoldRollOffset+6*6]; got != 10 {
		t.Fatalf("first tens stable row=%d, want strip row9 value10", got)
	}
	if got := frames[26][NativeShopGoldRollOffset+6*6]; got != 28 {
		t.Fatalf("third tens stable row=%d, want strip row27 value28", got)
	}
	if got := frames[62][NativeShopGoldRollOffset+7*6]; got != 64 {
		t.Fatalf("final units row=%d, want strip row63 value64", got)
	}
	// Tens stop after 33 and must remain unchanged while units reach 37.
	if got, want := frames[62][NativeShopGoldRollOffset+6*6],
		frames[26][NativeShopGoldRollOffset+6*6]; got != want {
		t.Fatalf("stable tens changed after 33: got=%d want=%d", got, want)
	}
	for index, got := range frames {
		if got[NativeShopGoldRollOffset-1] != 0 ||
			got[NativeShopGoldRollOffset+8*6] != 0 {
			t.Fatalf("frame%d wrote outside eight digit windows", index)
		}
	}

	wrap, next, err := ComposeNativeGoldCreditFrames(
		base, nativeGoldTestStrip(), 9_999_999, 1,
	)
	if err != nil || next != 10_000_000 || len(wrap) != 9 {
		t.Fatalf(
			"carry cascade = frames%d next%d err%v, want 9/10000000",
			len(wrap), next, err,
		)
	}
}

func TestComposeNativeGoldDebitFramesRejectsInvalidState(t *testing.T) {
	base := make([]byte, NativeShopWidth*NativeShopHeight)
	strip := nativeGoldTestStrip()
	for _, tc := range []struct {
		base   []byte
		strip  fdother.RawCell
		old    int
		amount int
	}{
		{base[:1], strip, 100, 1},
		{base, fdother.RawCell{Width: 6, Height: 98}, 100, 1},
		{base, strip, -1, 1},
		{base, strip, 100, 101},
		{base, strip, 100_000_000, 1},
	} {
		if _, _, err := ComposeNativeGoldDebitFrames(
			tc.base, tc.strip, tc.old, tc.amount,
		); err == nil {
			t.Fatalf("invalid state accepted: %#v", tc)
		}
	}
	frames, next, err := ComposeNativeGoldDebitFrames(base, strip, 100, 0)
	if err != nil || next != 100 || len(frames) != 0 {
		t.Fatalf("zero debit = frames%d next%d err%v", len(frames), next, err)
	}
}

func TestComposeNativeGoldCreditFramesRejectsInvalidState(t *testing.T) {
	base := make([]byte, NativeShopWidth*NativeShopHeight)
	strip := nativeGoldTestStrip()
	for _, tc := range []struct {
		base   []byte
		strip  fdother.RawCell
		old    int
		amount int
	}{
		{base[:1], strip, 100, 1},
		{base, fdother.RawCell{Width: 6, Height: 98}, 100, 1},
		{base, strip, -1, 1},
		{base, strip, 99_999_999, 1},
		{base, strip, 100, -1},
	} {
		if _, _, err := ComposeNativeGoldCreditFrames(
			tc.base, tc.strip, tc.old, tc.amount,
		); err == nil {
			t.Fatalf("invalid state accepted: %#v", tc)
		}
	}
	frames, next, err := ComposeNativeGoldCreditFrames(base, strip, 100, 0)
	if err != nil || next != 100 || len(frames) != 0 {
		t.Fatalf("zero credit = frames%d next%d err%v", len(frames), next, err)
	}
}
