package ending

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
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

func TestBuildMontageTailLoaderBaselineUsesExactSelector1EDeployment(t *testing.T) {
	const gameRoot = "../../../org_game/炎龍騎士團/FLAME2"
	paths := MontageTailLoaderPaths{
		FDFIELD: filepath.Join(gameRoot, "FDFIELD.DAT"),
		FDICON:  filepath.Join(gameRoot, "FDICON.B24"),
	}
	if _, err := os.Stat(paths.FDFIELD); os.IsNotExist(err) {
		t.Skip("player-provided FDFIELD.DAT is absent")
	} else if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(paths.FDICON); os.IsNotExist(err) {
		t.Skip("player-provided FDICON.B24 is absent")
	} else if err != nil {
		t.Fatal(err)
	}
	tail, err := LoadMontageTail(filepath.Join("..", "..", "assets", "endings", "native_2c194_tail.json"))
	if err != nil {
		t.Fatal(err)
	}
	items, err := battle.LoadNativeItemEffectRowPrefix(filepath.Join("..", "..", "assets", "data", "native_item_effect_rows.json"))
	if err != nil {
		t.Fatal(err)
	}
	records := []fdsave.PersistentRecord{{}, {}}
	for index := range records {
		for offset := range records[index].Raw {
			records[index].Raw[offset] = byte(index*0x40 + offset)
		}
		records[index].Raw[5] = byte(0x80 | index)
		records[index].Raw[7] = byte(4 + index*5)
		for slot := 0; slot < 8; slot++ {
			records[index].Raw[0x0a+slot*2] &^= 0x40
		}
		binary.LittleEndian.PutUint16(records[index].Raw[0x37:], uint16(10+index))
		binary.LittleEndian.PutUint16(records[index].Raw[0x39:], uint16(20+index))
		binary.LittleEndian.PutUint16(records[index].Raw[0x3e:], uint16(30+index))
	}
	before := append([]fdsave.PersistentRecord(nil), records...)

	baseline, err := BuildMontageTailLoaderBaseline(*tail, records, paths, items)
	if err != nil {
		t.Fatal(err)
	}
	if baseline.RuntimeCount() != nativeMontageTailDeployRecordCount {
		t.Fatalf("runtime count=%d, want %d", baseline.RuntimeCount(), nativeMontageTailDeployRecordCount)
	}
	first, err := baseline.RuntimeRecord(0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := baseline.RuntimeRecord(1)
	if err != nil {
		t.Fatal(err)
	}
	if first[0] != 17 || first[1] != 18 || second[0] != 1 || second[1] != 43 {
		t.Fatalf("selector 0x1e deployment coordinates=%d,%d / %d,%d", first[0], first[1], second[0], second[1])
	}
	if first[2] != 0 || second[2] != 1 || first[3] != 0 || first[4] != 0 || first[6] != 2 || first[0x31] != 0xff {
		t.Fatalf("first loader overwrite=% x", first[:0x32])
	}
	if first[5] != before[0].Raw[5] || first[7] != before[0].Raw[7] || first[8] != before[0].Raw[8] ||
		second[5] != before[1].Raw[5] || second[7] != before[1].Raw[7] || second[8] != before[1].Raw[8] {
		t.Fatalf("persistent raw provenance was not preserved")
	}
	for offset := 0x22; offset <= 0x27; offset++ {
		if first[offset] != 0 || second[offset] != 0 {
			t.Fatalf("transient offset %#x was not cleared", offset)
		}
	}
	inactive, err := baseline.RuntimeRecord(2)
	if err != nil {
		t.Fatal(err)
	}
	if inactive[5] != 1 || inactive[7] != 0 || inactive[0x31] != 0 {
		t.Fatalf("inactive deployment slot=% x", inactive[:])
	}
	pair, err := baseline.FirstPair()
	if err != nil {
		t.Fatal(err)
	}
	pair[0][0] ^= 0xff
	stillFirst, err := baseline.RuntimeRecord(0)
	if err != nil || stillFirst[0] != first[0] {
		t.Fatalf("baseline record unexpectedly aliases a returned pair: record=%#v err=%v", stillFirst, err)
	}
	if !reflect.DeepEqual(records, before) {
		t.Fatal("loader baseline mutated persistent source records")
	}
}

func TestBuildMontageTailLoaderBaselineFailsClosedBeforeAdmittingTailRenderer(t *testing.T) {
	tail, err := LoadMontageTail(filepath.Join("..", "..", "assets", "endings", "native_2c194_tail.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BuildMontageTailLoaderBaseline(*tail, []fdsave.PersistentRecord{{}}, MontageTailLoaderPaths{}, nil); err == nil {
		t.Fatal("one persistent record unexpectedly admitted")
	}
	tooMany := make([]fdsave.PersistentRecord, nativeMontageTailDeployRecordCount+1)
	if _, err := BuildMontageTailLoaderBaseline(*tail, tooMany, MontageTailLoaderPaths{}, nil); err == nil {
		t.Fatal("32 persistent records unexpectedly admitted to 31 deployment slots")
	}
	var empty MontageTailLoaderBaseline
	if _, err := empty.FirstPair(); err == nil {
		t.Fatal("empty loader baseline unexpectedly exposed a tail pair")
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
