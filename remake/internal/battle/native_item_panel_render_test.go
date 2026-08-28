package battle

import (
	"bytes"
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestRenderNativeItemPanelBasePreservesNativeOrderAndOpaqueZero(t *testing.T) {
	cells := make([]fdother.RawCell, 18)
	for i := range cells {
		cells[i] = fdother.RawCell{Width: 1, Height: 1, Pixels: []byte{byte(i)}}
	}
	// Grid entry 13 first writes at portrait origin 0x0c88. Portrait must
	// overwrite it, including its zero pixel.
	portrait := dato.Frame{Width: 2, Height: 1, Pixels: []byte{0, 0x77}}
	upper := fdother.LMI1Entry{Width: 2, Height: 1, Pixels: []byte{0, 0x55}}
	bottom := fdother.LMI1Entry{Width: 1, Height: 1, Pixels: []byte{0x66}}
	dst := make([]byte, nativeItemPanelBytes)
	for i := range dst {
		dst[i] = 0x99
	}
	if err := RenderNativeItemPanelBase(cells, portrait, upper, bottom, dst); err != nil {
		t.Fatal(err)
	}
	if dst[2245] != 1 || dst[3208] != 0 || dst[3209] != 0x77 {
		t.Fatalf("grid/portrait=%#x/%#x/%#x", dst[2245], dst[3208], dst[3209])
	}
	if at := 7*320 + 92; dst[at] != 0 || dst[at+1] != 0x55 {
		t.Fatalf("upper=%#x/%#x", dst[at], dst[at+1])
	}
	if dst[94*320+5] != 0x66 {
		t.Fatalf("bottom=%#x", dst[94*320+5])
	}
}

func TestRenderNativeItemPanelBaseIsAtomicOnInvalidSource(t *testing.T) {
	cells := make([]fdother.RawCell, 18)
	for i := range cells {
		cells[i] = fdother.RawCell{Width: 1, Height: 1, Pixels: []byte{1}}
	}
	dst := make([]byte, nativeItemPanelBytes)
	for i := range dst {
		dst[i] = 0x44
	}
	before := append([]byte(nil), dst...)
	bad := fdother.LMI1Entry{Width: 2, Height: 1, Pixels: []byte{1}}
	if err := RenderNativeItemPanelBase(cells, dato.Frame{Width: 1, Height: 1, Pixels: []byte{2}}, bad, bad, dst); err == nil {
		t.Fatal("invalid source was accepted")
	}
	if string(dst) != string(before) {
		t.Fatal("failed item panel render mutated destination")
	}
}

func TestRenderNativeItemPanelBaseWithPlayerAssets(t *testing.T) {
	const (
		fdotherPath   = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
		assetPackRoot = "../../generated-assets/fd2-original-b97caf22"
		portraitRoot  = "../../generated-assets/fd2-original-b97caf22/portraits"
	)
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	if _, err := os.Stat(portraitRoot + "/DATO_000_m0.png"); err != nil {
		t.Skip("separated portrait pack is absent")
	}
	dst := make([]byte, nativeItemPanelBytes)
	if err := RenderNativeItemPanelBaseResources(assetPackRoot, portraitRoot, 0, dst); err != nil {
		t.Fatal(err)
	}
	oracle := make([]byte, nativeItemPanelBytes)
	if err := RenderNativeItemPanelBaseResourcesArchive(fdotherPath, portraitRoot, 0, oracle); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dst, oracle) {
		t.Fatal("separated native item panel base differs from fixed archive")
	}
	nonzero := 0
	for _, pixel := range dst {
		if pixel != 0 {
			nonzero++
		}
	}
	if nonzero < 1000 {
		t.Fatalf("player item panel nonzero pixels=%d", nonzero)
	}
}

func TestRenderNativeItemPanelBaseResourcesFailsClosedWithoutSeparatedPack(t *testing.T) {
	dst := make([]byte, nativeItemPanelBytes)
	before := append([]byte(nil), dst...)
	if err := RenderNativeItemPanelBaseResources(t.TempDir(), t.TempDir(), 0, dst); err == nil {
		t.Fatal("missing separated item panel base was accepted")
	}
	if !bytes.Equal(dst, before) {
		t.Fatal("missing separated item panel base mutated destination")
	}
}
