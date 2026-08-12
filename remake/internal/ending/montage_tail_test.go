package ending

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMontageTailPlansProvenanceBoundRawTables(t *testing.T) {
	tail, err := LoadMontageTail(filepath.Join("..", "..", "assets", "endings", "native_2c194_tail.json"))
	if err != nil {
		t.Fatal(err)
	}
	entries, err := tail.Plan()
	if err != nil {
		t.Fatal(err)
	}
	wantRecord0 := []byte{51, 110, 19, 105, 54, 117, 30, 123, 39, 127, 64, 81, 52, 125, 26, 115, 41, 91, 31, 126}
	wantRecord1 := []byte{103, 20, 83, 28, 124, 38, 93, 34, 112, 44, 86, 53, 80, 55, 120, 36, 106, 60, 122, 50}
	wantGlobal := []byte{4, 3, 51, 14, 25, 18, 40, 53, 22, 24, 28, 17, 30, 31, 50, 33, 34, 52, 36, 47}
	if len(entries) != len(wantRecord0) {
		t.Fatalf("raw tail plan entries = %d, want %d", len(entries), len(wantRecord0))
	}
	for i, entry := range entries {
		want := MontageTailEntry{
			Index:        i,
			Global540FF:  wantGlobal[i],
			Record0Byte6: tailRecordByte6(wantRecord0[i]),
			Record0Byte7: wantRecord0[i],
			Record1Byte6: tailRecordByte6(wantRecord1[i]),
			Record1Byte7: wantRecord1[i],
		}
		if entry != want {
			t.Fatalf("raw tail plan entry %d = %#v, want %#v", i, entry, want)
		}
	}
}

func TestMontageTailRejectsShortOrNonByteRawTable(t *testing.T) {
	tail := MontageTail{Loop: MontageTailLoop{Count: 2}, RawTables: MontageTailRawTable{Global540FF: []int{1}, Record0Byte7: []int{1, 2}, Record1Byte7: []int{1, 2}}}
	if _, err := tail.Plan(); err == nil {
		t.Fatal("short raw table accepted")
	}
	tail.RawTables.Global540FF = []int{1, 0x100}
	if _, err := tail.Plan(); err == nil {
		t.Fatal("non-byte raw table accepted")
	}
}

func TestLoadMontageTailRejectsPreviousSchema(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "endings", "native_2c194_tail.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	obsolete := bytes.Replace(raw, []byte(`"schema_version": 2`), []byte(`"schema_version": 1`), 1)
	if bytes.Equal(obsolete, raw) {
		t.Fatal("tail asset did not contain schema version 2")
	}
	obsoletePath := filepath.Join(t.TempDir(), "native_2c194_tail_obsolete.json")
	if err := os.WriteFile(obsoletePath, obsolete, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadMontageTail(obsoletePath); err == nil {
		t.Fatal("obsolete tail schema was accepted")
	}
}

func TestTailRecordByte6UsesNativeThreshold(t *testing.T) {
	if got := tailRecordByte6(0x4b); got != 2 {
		t.Fatalf("selector 0x4b yielded +6=%d, want 2", got)
	}
	if got := tailRecordByte6(0x4c); got != 0 {
		t.Fatalf("selector 0x4c yielded +6=%d, want 0", got)
	}
}

func TestMontageTailAssetsPreservePaletteFramesAndTerminalImage(t *testing.T) {
	const datPath = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(datPath); os.IsNotExist(err) {
		t.Skip("player-provided FDOTHER.DAT is absent")
	} else if err != nil {
		t.Fatal(err)
	}
	tail, err := LoadMontageTail(filepath.Join("..", "..", "assets", "endings", "native_2c194_tail.json"))
	if err != nil {
		t.Fatal(err)
	}
	assets, err := LoadMontageTailAssets(*tail, datPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets.LoopPalette) != 768 || len(assets.LoopFrames) != 20 ||
		assets.Intro.Width != Width || assets.Intro.Height != Height ||
		assets.Final.Width != Width || assets.Final.Height != Height {
		t.Fatalf("tail assets palette=%d frames=%d intro=%dx%d final=%dx%d",
			len(assets.LoopPalette), len(assets.LoopFrames), assets.Intro.Width, assets.Intro.Height,
			assets.Final.Width, assets.Final.Height)
	}
	compositor := NewIndexedCompositor()
	copy(compositor.Palette[:], assets.LoopPalette)
	copy(compositor.Baseline[:], assets.LoopPalette)
	compositor.baselineKnown = true
	if err := assets.PresentFinal(compositor); err != nil {
		t.Fatal(err)
	}
	visible := 0
	for _, pixel := range compositor.VGA {
		if pixel != 0 {
			visible++
		}
	}
	if visible < 100 {
		t.Fatalf("terminal frame has only %d visible indexed pixels", visible)
	}
}

func TestMontageTailAssetsKeepNativeFrameTableGeometry(t *testing.T) {
	const datPath = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(datPath); os.IsNotExist(err) {
		t.Skip("player-provided FDOTHER.DAT is absent")
	} else if err != nil {
		t.Fatal(err)
	}
	tail, err := LoadMontageTail(filepath.Join("..", "..", "assets", "endings", "native_2c194_tail.json"))
	if err != nil {
		t.Fatal(err)
	}
	assets, err := LoadMontageTailAssets(*tail, datPath)
	if err != nil {
		t.Fatal(err)
	}
	want := [][4]int{
		{15, 7, 55, 98}, {210, 85, 55, 98}, {164, 9, 55, 98}, {41, 91, 71, 98},
		{160, 36, 71, 98}, {45, 11, 71, 98}, {221, 33, 55, 98}, {74, 74, 60, 115},
		{131, 28, 70, 98}, {20, 11, 71, 98}, {193, 83, 71, 98}, {226, 12, 71, 98},
		{57, 84, 71, 98}, {115, 67, 70, 98}, {165, 13, 70, 115}, {31, 86, 78, 98},
		{195, 73, 70, 115}, {77, 7, 100, 115}, {226, 5, 78, 115}, {39, 71, 237, 28},
	}
	if len(assets.LoopFrames) != len(want) {
		t.Fatalf("native frame count = %d, want %d", len(assets.LoopFrames), len(want))
	}
	for i, frame := range assets.LoopFrames {
		got := [4]int{frame.X, frame.Y, frame.Width, frame.Height}
		if got != want[i] {
			t.Fatalf("native frame %d geometry = %v, want %v", i, got, want[i])
		}
	}
}
