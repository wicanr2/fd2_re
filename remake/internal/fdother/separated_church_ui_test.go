package fdother

import (
	"bytes"
	"os"
	"testing"
)

func TestSeparatedChurchUIAssetsMatchOriginalCodecs(t *testing.T) {
	const (
		rawPath = "../../../extracted/raw/FDOTHER/FDOTHER_014.bin"
		uiRoot  = "../../generated-assets/fd2-original-b97caf22/ui"
	)
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		t.Skip("player-provided FDOTHER_014 is absent")
	}
	got, err := LoadSeparatedChurchUIAssets(uiRoot)
	if err != nil {
		t.Skipf("separated FDOTHER_014 church pack is absent: %v", err)
	}
	for _, index := range churchUIOpaqueEntries {
		want, err := ParseLMI1OpaqueEntry(raw, index)
		if err != nil || got.Entries[index].Width != want.Width ||
			got.Entries[index].Height != want.Height ||
			!bytes.Equal(got.Entries[index].Pixels, want.Pixels) {
			t.Fatalf("opaque entry %d differs: %v", index, err)
		}
	}
	wantPrice, err := ParseLMI1RawEntry(raw, 15)
	if err != nil || got.PriceCell.Width != wantPrice.Width ||
		got.PriceCell.Height != wantPrice.Height ||
		!bytes.Equal(got.PriceCell.Pixels, wantPrice.Pixels) {
		t.Fatalf("raw entry 15 differs: %v", err)
	}
	for _, index := range churchUIFrameEntries {
		want, err := ParseLMI1FrameEntry(raw, index)
		if err != nil {
			t.Fatalf("frame entry %d: %v", index, err)
		}
		wantIndexed, wantMask, err := want.IndexedLayers()
		var frame Frame
		if index == 0 {
			frame = got.Background
		} else {
			frame = got.ReviveFX[index-23]
		}
		if err != nil || frame.Width != want.Width || frame.Height != want.Height ||
			!bytes.Equal(frame.Indexed, wantIndexed) || !bytes.Equal(frame.Mask, wantMask) {
			t.Fatalf("frame entry %d differs: %v", index, err)
		}
	}
}

func TestSeparatedChurchUIAssetsFailClosedWithoutPack(t *testing.T) {
	if _, err := LoadSeparatedChurchUIAssets(t.TempDir()); err == nil {
		t.Fatal("incomplete separated church UI pack was accepted")
	}
}
