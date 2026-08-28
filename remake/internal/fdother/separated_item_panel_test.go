package fdother

import (
	"bytes"
	"os"
	"testing"
)

func TestSeparatedItemPanelEntriesMatchOriginalCodecs(t *testing.T) {
	const (
		rawPath = "../../../extracted/raw/FDOTHER/FDOTHER_005.bin"
		uiRoot  = "../../generated-assets/fd2-original-b97caf22/ui"
	)
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		t.Skip("player-provided FDOTHER_005 is absent")
	}
	separated, err := LoadSeparatedItemPanelEntries(uiRoot)
	if err != nil {
		t.Skipf("separated FDOTHER_005 panel is absent: %v", err)
	}
	for _, index := range itemPanelOpaqueEntries {
		want, err := ParseLMI1OpaqueEntry(raw, index)
		got, ok := separated.Opaque[index]
		if err != nil || !ok || got.Width != want.Width || got.Height != want.Height || !bytes.Equal(got.Pixels, want.Pixels) {
			t.Fatalf("opaque entry %d differs: ok=%v err=%v", index, ok, err)
		}
	}
	for _, index := range itemPanelRawEntries {
		want, err := ParseLMI1RawEntry(raw, index)
		got, ok := separated.Raw[index]
		if err != nil || !ok || got.Width != want.Width || got.Height != want.Height || !bytes.Equal(got.Pixels, want.Pixels) {
			t.Fatalf("raw entry %d differs: ok=%v err=%v", index, ok, err)
		}
	}
	for _, index := range itemPanelFrameEntries {
		want, err := ParseLMI1FrameEntry(raw, index)
		if err != nil {
			t.Fatalf("frame entry %d source: %v", index, err)
		}
		wantIndexed, wantMask, err := want.IndexedLayers()
		got, ok := separated.Frames[index]
		if err != nil || !ok || got.Width != want.Width || got.Height != want.Height ||
			!bytes.Equal(got.Indexed, wantIndexed) || !bytes.Equal(got.Mask, wantMask) {
			t.Fatalf("frame entry %d differs: ok=%v err=%v", index, ok, err)
		}
	}
}

func TestSeparatedItemPanelEntriesFailClosedWithoutCompletePack(t *testing.T) {
	if _, err := LoadSeparatedItemPanelEntries(t.TempDir()); err == nil {
		t.Fatal("incomplete separated item panel pack was accepted")
	}
}
