package battle

import (
	"encoding/binary"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestRenderNativeTransferItemRowsMatches2DC55Mode1(t *testing.T) {
	assets := nativeItemPanelTestAssets(t, 0)
	rows := make([]byte, 2*NativeItemEffectRowSize)
	rows[0] = 1
	binary.LittleEndian.PutUint16(rows[1:3], 123)
	binary.LittleEndian.PutUint16(rows[19:21], 1000)
	rows[NativeItemEffectRowSize] = 0x15
	binary.LittleEndian.PutUint16(
		rows[NativeItemEffectRowSize+5:NativeItemEffectRowSize+7], 45,
	)
	binary.LittleEndian.PutUint16(
		rows[NativeItemEffectRowSize+19:NativeItemEffectRowSize+21], 200,
	)
	facility := fdother.RawCell{Width: 1, Height: 1, Pixels: []byte{88}}
	dst := make([]byte, nativeItemPanelBytes)
	if err := RenderNativeTransferItemRows(
		assets, facility, []int{0, 1}, 0, 1, rows, dst,
	); err != nil {
		t.Fatal(err)
	}
	if got := dst[119*320+10]; got != 59 {
		t.Fatalf("item0 category=%d want 59", got)
	}
	if got := dst[119*320+158]; got != 60 {
		t.Fatalf("item1 category=%d want 60", got)
	}
	if got := dst[122*320+38]; got != 205 {
		t.Fatalf("item0 label=%d want 205", got)
	}
	if got := dst[122*320+186]; got != 201 {
		t.Fatalf("selected item1 label=%d want 201", got)
	}
	if got := dst[131*320+105]; got != 88 {
		t.Fatalf("facility cell=%d want 88", got)
	}
	// Mode1 displays 3/4 of row+19: 1000 -> 750 with digit base119.
	if got := dst[131*320+114]; got != 119 {
		t.Fatalf("price leading digit=%d want base119", got)
	}
}

func TestRenderNativeFacilityItemRowsModeZeroUsesFullPrice(t *testing.T) {
	assets := nativeItemPanelTestAssets(t, 0)
	rows := make([]byte, NativeItemEffectRowSize)
	rows[0] = 1
	binary.LittleEndian.PutUint16(rows[19:21], 1000)
	facility := fdother.RawCell{Width: 1, Height: 1, Pixels: []byte{88}}
	full := make([]byte, nativeItemPanelBytes)
	sale := make([]byte, nativeItemPanelBytes)
	if err := RenderNativeFacilityItemRows(
		assets, facility, []int{0}, 0, 0, rows,
		NativeFacilityFullPrice, full,
	); err != nil {
		t.Fatal(err)
	}
	if err := RenderNativeFacilityItemRows(
		assets, facility, []int{0}, 0, 0, rows,
		NativeFacilityThreeQuarterPrice, sale,
	); err != nil {
		t.Fatal(err)
	}
	// Five digits start at x=114 with six-pixel advances. 1000 renders
	// 01000 while 750 renders 00750; the middle digit therefore differs.
	if got := full[131*320+126]; got != 119 {
		t.Fatalf("full-price hundreds digit=%d, want frame119", got)
	}
	if got := sale[131*320+126]; got != 126 {
		t.Fatalf("three-quarter hundreds digit=%d, want frame126", got)
	}
}

func TestRenderNativeFacilityItemRowsWithoutNamesLeavesSafeRectangleBlank(t *testing.T) {
	assets := nativeItemPanelTestAssets(t, 0)
	rows := make([]byte, NativeItemEffectRowSize)
	rows[0] = 1
	binary.LittleEndian.PutUint16(rows[19:21], 1000)
	facility := fdother.RawCell{Width: 1, Height: 1, Pixels: []byte{88}}
	dst := make([]byte, nativeItemPanelBytes)
	if err := RenderNativeFacilityItemRowsWithoutNames(
		assets, facility, []int{0}, 0, 0, rows,
		NativeFacilityFullPrice, dst,
	); err != nil {
		t.Fatal(err)
	}
	for y := 122; y < 138; y++ {
		for x := 38; x < 105; x++ {
			if dst[y*320+x] != 0 {
				t.Fatalf("name rectangle changed at %d,%d: %d", x, y, dst[y*320+x])
			}
		}
	}
	if dst[119*320+10] != 59 || dst[131*320+114] != 119 {
		t.Fatal("suppressing names also removed category or price")
	}
}

func TestNativeFacilityItemListPriceMatchesRendererModes(t *testing.T) {
	rows := make([]byte, NativeItemEffectRowSize)
	binary.LittleEndian.PutUint16(rows[19:], 101)
	full, err := NativeFacilityItemListPrice(
		rows, 0, NativeFacilityFullPrice,
	)
	if err != nil || full != 101 {
		t.Fatalf("full price=%d err=%v", full, err)
	}
	sale, err := NativeFacilityItemListPrice(
		rows, 0, NativeFacilityThreeQuarterPrice,
	)
	if err != nil || sale != 75 {
		t.Fatalf("sale price=%d err=%v", sale, err)
	}
}
