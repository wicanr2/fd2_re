package battle

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func nativeItemPanelTestStrings(t *testing.T, word uint16) *fdtxt.Strings {
	t.Helper()
	const count = 500
	data := make([]byte, count*2+count*4)
	for index := 0; index < count; index++ {
		offset := count*2 + index*4
		binary.LittleEndian.PutUint16(data[index*2:], uint16(offset))
		binary.LittleEndian.PutUint16(data[offset:], word)
		binary.LittleEndian.PutUint16(data[offset+2:], fdtxt.StringEnd)
	}
	strings, err := fdtxt.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	return strings
}

func nativeItemPanelTestAssets(t *testing.T, textWord uint16) NativeItemPanelDataAssets {
	t.Helper()
	raw := make(map[int]fdother.RawCell)
	for index := 0; index <= 92; index++ {
		raw[index] = fdother.RawCell{
			Width: 1, Height: 1, Pixels: []byte{byte(index)},
		}
	}
	frames := make(map[int]fdother.Frame)
	for index := 0; index <= 140; index++ {
		frames[index] = fdother.Frame{
			Width: 1, Height: 1,
			Pixels: []byte{1, 0, 1, 0, 0, byte(index)},
		}
	}
	fontData := make([]byte, fdtxt.GlyphBytes)
	fontData[0] = 0x80
	font, err := fdtxt.ParseFont(fontData)
	if err != nil {
		t.Fatal(err)
	}
	return NativeItemPanelDataAssets{
		RawCells: raw,
		Frames:   frames,
		Strings:  nativeItemPanelTestStrings(t, textWord),
		Font:     font,
	}
}

func TestRenderNativeItemPanelDataMatches17FC0Subpasses(t *testing.T) {
	assets := nativeItemPanelTestAssets(t, 0)
	record := make([]byte, nativeRecordSize)
	record[6] = 0 // base icon selects entry 54
	record[37] = 1
	binary.LittleEndian.PutUint16(record[64:], 5)
	binary.LittleEndian.PutUint16(record[66:], 10)
	binary.LittleEndian.PutUint16(record[68:], 0)
	binary.LittleEndian.PutUint16(record[70:], 10)
	binary.LittleEndian.PutUint16(record[72:], 12)
	record[34] = 1

	dst := make([]byte, nativeItemPanelBytes)
	if err := RenderNativeItemPanelData(assets, record, dst); err != nil {
		t.Fatal(err)
	}
	// HP bar: base, 50 middle cells, then end. MP zero path starts at x+1.
	if dst[33*320+198] != 23 || dst[33*320+199] != 24 ||
		dst[33*320+249] != 25 {
		t.Fatalf("HP bar=%d/%d/%d", dst[33*320+198], dst[33*320+199], dst[33*320+249])
	}
	if dst[52*320+198] != 0 || dst[52*320+199] != 29 ||
		dst[52*320+300] != 30 {
		t.Fatalf("MP zero bar=%d/%d/%d", dst[52*320+198], dst[52*320+199], dst[52*320+300])
	}
	// The first compared number is unequal and therefore uses base 42.
	if dst[41*320+267] != 42 || dst[41*320+279] != 47 {
		t.Fatalf("compared number=%d/%d", dst[41*320+267], dst[41*320+279])
	}
	// record+34 switches the +72 value to colour base 119.
	if dst[67*320+157] != 119 || dst[67*320+169] != 121 {
		t.Fatalf("alternate number=%d/%d", dst[67*320+157], dst[67*320+169])
	}
	// Three FDTXT strings use foreground 205; base/flag icons follow after.
	if dst[13*320+99] != 205 || dst[13*320+211] != 205 || dst[13*320+251] != 205 {
		t.Fatalf("text foreground=%d/%d/%d", dst[13*320+99], dst[13*320+211], dst[13*320+251])
	}
	if dst[30*320+101] != 54 || dst[68*320+194] != 55 ||
		dst[68*320+229] != 0 {
		t.Fatalf("icons=%d/%d/%d", dst[30*320+101], dst[68*320+194], dst[68*320+229])
	}
}

func TestRenderNativeItemPanelDataFailsAtomicallyOnTextControl(t *testing.T) {
	assets := nativeItemPanelTestAssets(t, 0xfffe)
	record := make([]byte, nativeRecordSize)
	binary.LittleEndian.PutUint16(record[66:], 1)
	binary.LittleEndian.PutUint16(record[70:], 1)
	dst := make([]byte, nativeItemPanelBytes)
	for index := range dst {
		dst[index] = 0x44
	}
	before := append([]byte(nil), dst...)
	if err := RenderNativeItemPanelData(assets, record, dst); err == nil {
		t.Fatal("control-bearing item text was accepted")
	}
	if string(dst) != string(before) {
		t.Fatal("failed dynamic overlay mutated destination")
	}
}

func TestRenderNativeFacilityNumberAccepts1B41DCurrencyWidth(t *testing.T) {
	assets := nativeItemPanelTestAssets(t, 0)
	dst := make([]byte, nativeItemPanelBytes)
	if err := RenderNativeFacilityNumber(assets, dst, 10, 20, 12345678, 31, 8); err != nil {
		t.Fatal(err)
	}
	for digit, want := range []byte{32, 33, 34, 35, 36, 37, 38, 39} {
		if got := dst[20*320+10+digit*6]; got != want {
			t.Fatalf("digit %d=%d, want %d", digit, got, want)
		}
	}
}

func TestRenderNativeItemPanelRowsMatches184C0GeometryAndSelection(t *testing.T) {
	assets := nativeItemPanelTestAssets(t, 0)
	record := make([]byte, nativeRecordSize)
	for slot := 0; slot < 8; slot++ {
		record[0x0a+slot*2] = 0x80
	}
	record[0x0a], record[0x0b] = 0, 0
	effectRows := make([]byte, NativeItemEffectRowSize)
	effectRows[0] = 0
	binary.LittleEndian.PutUint16(effectRows[1:], 12)
	dst := make([]byte, nativeItemPanelBytes)
	if err := RenderNativeItemPanelRows(assets, record, 0, effectRows, dst); err != nil {
		t.Fatal(err)
	}
	if dst[101*320+13] != 59 {
		t.Fatalf("category icon=%d", dst[101*320+13])
	}
	if dst[103*320+42] != 201 {
		t.Fatalf("selected item text=%d", dst[103*320+42])
	}
	if dst[107*320+110] != 64 {
		t.Fatalf("stat icon=%d", dst[107*320+110])
	}
	if dst[107*320+135] != 42 || dst[107*320+141] != 43 || dst[107*320+147] != 44 {
		t.Fatalf("stat number=%d/%d/%d", dst[107*320+135], dst[107*320+141], dst[107*320+147])
	}
}

func TestNativeItemPanelBaseAndDataWithPlayerAssets(t *testing.T) {
	const (
		fdotherPath  = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
		fdtxtPath    = "../../../org_game/炎龍騎士團/FLAME2/FDTXT.DAT"
		portraitRoot = "../../generated-assets/fd2-original-b97caf22/portraits"
	)
	for _, path := range []string{fdotherPath, fdtxtPath, portraitRoot + "/DATO_000_m0.png"} {
		if _, err := os.Stat(path); err != nil {
			t.Skip("player-provided original archives are absent")
		}
	}
	dst := make([]byte, nativeItemPanelBytes)
	record := make([]byte, nativeRecordSize)
	for slot := 0; slot < 8; slot++ {
		record[0x0a+slot*2] = 0x80
	}
	record[0x0a], record[0x0b] = 0, 0
	binary.LittleEndian.PutUint16(record[64:], 80)
	binary.LittleEndian.PutUint16(record[66:], 100)
	binary.LittleEndian.PutUint16(record[68:], 20)
	binary.LittleEndian.PutUint16(record[70:], 40)
	if err := RenderNativeItemPanelResources(fdotherPath, fdtxtPath, portraitRoot, record, dst); err != nil {
		t.Fatal(err)
	}
	assets, err := LoadNativeItemPanelDataAssets(fdotherPath, fdtxtPath)
	if err != nil {
		t.Fatal(err)
	}
	itemRows, err := LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := RenderNativeItemPanelRows(assets, record, 0, itemRows, dst); err != nil {
		t.Fatal(err)
	}
	nonzero := 0
	for _, pixel := range dst {
		if pixel != 0 {
			nonzero++
		}
	}
	if nonzero < 1000 {
		t.Fatalf("player complete item panel nonzero pixels=%d", nonzero)
	}
}
