package fdother

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

func solidPreparationSprite(pixel byte) fdicon.Sprite {
	pixels := bytes.Repeat([]byte{pixel}, fdicon.NativeSize*fdicon.NativeSize)
	return fdicon.Sprite{
		Pixels:    pixels,
		Mask:      bytes.Repeat([]byte{1}, len(pixels)),
		RemapMask: make([]byte, len(pixels)),
	}
}

func preparationUnitBank(keys ...int) *fdicon.Bank {
	maxKey := 0
	for _, key := range keys {
		if key > maxKey {
			maxKey = key
		}
	}
	bank := &fdicon.Bank{Sprites: make([]fdicon.Sprite, (maxKey+1)*12)}
	for _, key := range keys {
		for cycle := 0; cycle < 3; cycle++ {
			bank.Sprites[key*12+cycle] = solidPreparationSprite(byte(key*8 + cycle + 1))
		}
	}
	return bank
}

func TestNativePreparationRosterPositions(t *testing.T) {
	tests := []struct {
		index            int
		selected, cursor bool
		x, y             int
	}{
		{index: 0, x: 23, y: 100},
		{index: 9, x: 275, y: 100},
		{index: 10, x: 23, y: 130},
		{index: 29, x: 275, y: 160},
		{index: 10, selected: true, x: 23, y: 133},
		{index: 10, cursor: true, x: 23, y: 134},
	}
	for _, test := range tests {
		x, y, err := NativePreparationRosterPosition(test.index, test.selected, test.cursor)
		if err != nil || x != test.x || y != test.y {
			t.Fatalf("index %d selected=%v cursor=%v: (%d,%d,%v), want (%d,%d,nil)",
				test.index, test.selected, test.cursor, x, y, err, test.x, test.y)
		}
	}
}

func TestMoveNativePreparationRosterCursorUsesTenColumnGrid(t *testing.T) {
	tests := []struct {
		cursor, count int
		scan          byte
		want          int
	}{
		{0, 16, 0x4b, 0},
		{0, 16, 0x4d, 1},
		{15, 16, 0x4d, 15},
		{9, 16, 0x48, 9},
		{10, 16, 0x48, 0},
		{5, 16, 0x50, 15},
		{6, 16, 0x50, 6},
	}
	for _, test := range tests {
		got, err := MoveNativePreparationRosterCursor(test.cursor, test.count, test.scan)
		if err != nil || got != test.want {
			t.Fatalf("cursor=%d count=%d scan=%#x: (%d,%v), want (%d,nil)",
				test.cursor, test.count, test.scan, got, err, test.want)
		}
	}
}

func TestBlitNativePreparationRosterUsesNativeSelectedAndUnselectedPaths(t *testing.T) {
	frame := make([]byte, 320*200)
	units := preparationUnitBank(2, 3)
	cursor := &fdicon.Bank{Sprites: []fdicon.Sprite{solidPreparationSprite(0x55)}}
	err := BlitNativePreparationRoster(
		frame, 320, units, cursor,
		[]int{2, 3}, []bool{false, true}, 1, 2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := frame[100*320+23]; got != (byte(2*8+3)&7)+0x18 {
		t.Fatalf("unselected palette-band pixel=%#x", got)
	}
	if got := frame[103*320+51]; got != byte(3*8+3) {
		t.Fatalf("selected raised pixel=%#x", got)
	}
	// 0x31e80 draws the cursor first. The selected unit covers its middle,
	// leaving the cursor's final row visible below the three-pixel-raised unit.
	if got := frame[127*320+51]; got != 0x55 {
		t.Fatalf("cursor pixel=%#x", got)
	}
	if got := frame[100*320+51]; got != 0 {
		t.Fatalf("selected sprite was not raised by three pixels: %#x", got)
	}
}

func TestBlitNativePreparationRosterFailsAtomically(t *testing.T) {
	frame := bytes.Repeat([]byte{0x44}, 320*200)
	before := append([]byte(nil), frame...)
	units := preparationUnitBank(1)
	units.Sprites[12] = fdicon.Sprite{}
	cursor := &fdicon.Bank{Sprites: []fdicon.Sprite{solidPreparationSprite(0x55)}}
	err := BlitNativePreparationRoster(
		frame, 320, units, cursor,
		[]int{1}, []bool{false}, 0, 0,
	)
	if err == nil {
		t.Fatal("malformed unit sprite unexpectedly rendered")
	}
	if !bytes.Equal(frame, before) {
		t.Fatal("failed preparation render changed destination")
	}
}

func TestDecodeAndComposeNativePreparationAssetsFromPlayerArchive(t *testing.T) {
	base := "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	fdiconPath := filepath.Join(base, "FDICON.B24")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided original assets are absent")
	}
	assets, err := DecodeNativePreparationAssets(fdotherPath, fdiconPath)
	if err != nil {
		t.Fatal(err)
	}
	if assets.UpperRight.Width != 223 || assets.UpperRight.Height != 86 ||
		assets.Lower.Width != 310 || assets.Lower.Height != 99 ||
		assets.UpperLeft.Width != 86 || assets.UpperLeft.Height != 86 {
		t.Fatalf("native preparation geometry changed: %#v", assets)
	}
	frame, err := ComposeNativePreparationFrame(
		assets,
		[]int{0, 9, 4}, []bool{false, true, false},
		1, 0, 15,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) != 320*200 {
		t.Fatalf("native preparation frame length=%d", len(frame))
	}
	if bytes.Equal(frame, make([]byte, len(frame))) {
		t.Fatal("native preparation frame is empty")
	}
	// 0x31ea9 畫 quota（15），0x31edb 畫 quota-selected（14）。兩個數字格
	// 都和原始字形項目比較，避免第一組再次被已選人數悄悄取代。
	want := append([]byte(nil), frame...)
	if err := blitNativePreparationTwoDigits(assets.Digits, want, 61, 35, 15); err != nil {
		t.Fatal(err)
	}
	if err := blitNativePreparationTwoDigits(assets.Digits, want, 61, 73, 14); err != nil {
		t.Fatal(err)
	}
	for _, box := range [][4]int{{61, 35, 12, 8}, {61, 73, 12, 8}} {
		for y := box[1]; y < box[1]+box[3]; y++ {
			start := y*320 + box[0]
			if !bytes.Equal(frame[start:start+box[2]], want[start:start+box[2]]) {
				t.Fatalf("native preparation count differs at box %#v", box)
			}
		}
	}
}
