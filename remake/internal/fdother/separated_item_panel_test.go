package fdother

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestSeparatedSystemInfoPanelsMatchFixedArchive(t *testing.T) {
	uiRoot := filepath.Join("..", "..", "generated-assets", "fd2-original-b97caf22", "ui")
	archive := filepath.Join("..", "..", "..", "org_game", "炎龍騎士團", "FLAME2", "FDOTHER.DAT")
	if _, err := os.Stat(filepath.Join(uiRoot, "fdother_005_item_panel", "resource.json")); os.IsNotExist(err) {
		t.Skip("separated FDOTHER #5 bank is absent")
	} else if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(archive); os.IsNotExist(err) {
		t.Skip("player-provided FDOTHER.DAT is absent")
	} else if err != nil {
		t.Fatal(err)
	}
	got, err := LoadSeparatedSystemInfoPanels(uiRoot)
	if err != nil {
		t.Fatal(err)
	}
	want, err := DecodeLMI1Resource(archive, 5)
	if err != nil {
		t.Fatal(err)
	}
	for offset := range got {
		index := 0x85 + offset
		if got[offset].Width != want[index].Width || got[offset].Height != want[index].Height ||
			!bytes.Equal(got[offset].Pixels, want[index].Pixels) {
			t.Fatalf("separated system info entry %#x differs from fixed archive", index)
		}
	}
}

func TestSeparatedSystemInfoPanelsFailsClosedWithoutCompleteBank(t *testing.T) {
	if _, err := LoadSeparatedSystemInfoPanels(t.TempDir()); err == nil {
		t.Fatal("missing separated system info bank was accepted")
	}
}

func TestSeparatedItemPanelEntriesFailClosedWithoutCompletePack(t *testing.T) {
	if _, err := LoadSeparatedItemPanelEntries(t.TempDir()); err == nil {
		t.Fatal("incomplete separated item panel pack was accepted")
	}
}
