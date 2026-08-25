package battle

import (
	"bytes"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func nativeBattlePanelTestAssets(t *testing.T) NativeItemPanelDataAssets {
	t.Helper()
	assets := nativeItemPanelTestAssets(t, 0)
	assets.BattlePanel = fdother.LMI1Entry{
		Width: nativeBattlePanelWidth, Height: nativeBattlePanelHeight,
		Pixels: bytes.Repeat([]byte{9}, nativeBattlePanelWidth*nativeBattlePanelHeight),
	}
	return assets
}

func nativeBattlePanelTestRecord(t *testing.T, rawSide byte) []byte {
	t.Helper()
	record, err := NativeBattlePanelRecordForUnit(&Unit{
		Lv: 4, HP: 5, MaxHP: 10, MP: 0, MaxMP: 8,
		NativeRecordByte6: rawSide, HasNativeRecordByte6: true,
		NativeRecordByte8: 0, HasNativeRecordByte8: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func TestRenderNativeBattlePanelMatches18C6DBottomGeometry(t *testing.T) {
	dst := make([]byte, 320*200)
	if err := RenderNativeBattlePanel(nativeBattlePanelTestAssets(t),
		nativeBattlePanelTestRecord(t, 0), dst, 3, 7); err != nil {
		t.Fatal(err)
	}
	if dst[154*320] != 9 || dst[195*320+148] != 9 {
		t.Fatal("149x42 opaque panel was not placed at raw-side zero origin")
	}
	if dst[(154+22)*320+21] != 23 || dst[(154+22)*320+22] != 24 ||
		dst[(154+22)*320+72] != 25 {
		t.Fatal("HP bar did not use entries 23/24/25 at the native origin")
	}
	if dst[(154+31)*320+22] != 29 || dst[(154+31)*320+123] != 30 {
		t.Fatal("zero MP bar did not use entries 29/30")
	}
	if dst[(154+4)*320+132] != 31 || dst[(154+4)*320+138] != 35 {
		t.Fatal("two-digit level was not drawn with base31 and six-pixel advance")
	}
	if dst[(154+4)*320+5] != 205 {
		t.Fatal("raw record+8 FDTXT name was not drawn at panel origin+(5,4)")
	}
}

func TestRenderNativeBattlePanelValuesAtReuses18C6DBarAndDigitCore(t *testing.T) {
	dst := make([]byte, 320*200)
	err := RenderNativeBattlePanelValuesAt(nativeBattlePanelTestAssets(t), dst, 0, 154,
		NativeBattlePanelValues{Level: 2, HP: 8, MaxHP: 28, MP: 0, MaxMP: 0})
	if err != nil {
		t.Fatal(err)
	}
	if dst[(154+22)*320+21] != 23 || dst[(154+22)*320+22] != 24 ||
		dst[(154+22)*320+50] != 25 {
		t.Fatal("HP 8/28 did not consume native 23/24/25 cells and +1 endpoint")
	}
	if dst[(154+4)*320+132] != 31 || dst[(154+4)*320+138] != 33 {
		t.Fatal("level 02 did not consume native digit frames")
	}
}

func TestRenderNativeBattlePanelValuesAtFailsAtomically(t *testing.T) {
	assets := nativeBattlePanelTestAssets(t)
	delete(assets.Frames, 31)
	dst := bytes.Repeat([]byte{77}, 320*200)
	want := append([]byte(nil), dst...)
	if err := RenderNativeBattlePanelValuesAt(assets, dst, 0, 154,
		NativeBattlePanelValues{Level: 2, HP: 8, MaxHP: 28}); err == nil {
		t.Fatal("missing digit frame was accepted")
	}
	if !bytes.Equal(dst, want) {
		t.Fatal("failed value render mutated its destination")
	}
}

func TestRenderNativeBattlePanelUsesTopAndRawChapter24Exception(t *testing.T) {
	assets := nativeBattlePanelTestAssets(t)
	top := make([]byte, 320*200)
	if err := RenderNativeBattlePanel(assets, nativeBattlePanelTestRecord(t, 1), top, 17, 23); err != nil {
		t.Fatal(err)
	}
	if top[4*320+171] != 9 || top[154*320] != 0 {
		t.Fatal("nonzero raw side did not select the top panel")
	}
	exception := make([]byte, 320*200)
	if err := RenderNativeBattlePanel(assets, nativeBattlePanelTestRecord(t, 1), exception, 17, 24); err != nil {
		t.Fatal(err)
	}
	if exception[154*320] != 9 || exception[4*320+171] != 0 {
		t.Fatal("raw chapter24 unit17 exception did not select the bottom panel")
	}
}

func TestRenderNativeBattlePanelFailsAtomically(t *testing.T) {
	assets := nativeBattlePanelTestAssets(t)
	delete(assets.RawCells, 23)
	dst := bytes.Repeat([]byte{77}, 320*200)
	want := append([]byte(nil), dst...)
	if err := RenderNativeBattlePanel(assets, nativeBattlePanelTestRecord(t, 0), dst, 0, 0); err == nil {
		t.Fatal("missing raw HP cell was accepted")
	}
	if !bytes.Equal(dst, want) {
		t.Fatal("failed panel render mutated its destination")
	}
}

func TestNativeBattlePanelRecordRequiresRawSelectors(t *testing.T) {
	if _, err := NativeBattlePanelRecordForUnit(&Unit{Lv: 1, HP: 1, MaxHP: 1}); err == nil {
		t.Fatal("missing raw +6/+8 provenance was accepted")
	}
}
