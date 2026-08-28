package fdother

import (
	"bytes"
	"os"
	"testing"
)

func TestSeparatedRangeOverlayBankMatchesOriginal(t *testing.T) {
	const root = "../../generated-assets/fd2-original-b97caf22/sprites/fdother_001_range_overlay"
	archive := "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(archive); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	want, err := DecodeNativeRangeOverlayBank(archive)
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadSeparatedRangeOverlayBank(root)
	if err != nil {
		t.Skipf("separated range-overlay pack is absent: %v", err)
	}
	if len(got.Sprites) != len(want.Sprites) {
		t.Fatalf("sprite count=%d want=%d", len(got.Sprites), len(want.Sprites))
	}
	for index := range want.Sprites {
		if !bytes.Equal(got.Sprites[index].Pixels, want.Sprites[index].Pixels) ||
			!bytes.Equal(got.Sprites[index].Mask, want.Sprites[index].Mask) ||
			!bytes.Equal(got.Sprites[index].RemapMask, want.Sprites[index].RemapMask) {
			t.Fatalf("range-overlay sprite %d differs", index)
		}
	}
}

func TestSeparatedRangeOverlayBankFailsClosedWithoutPack(t *testing.T) {
	if _, err := LoadSeparatedRangeOverlayBank(t.TempDir()); err == nil {
		t.Fatal("incomplete range-overlay pack was accepted")
	}
}
