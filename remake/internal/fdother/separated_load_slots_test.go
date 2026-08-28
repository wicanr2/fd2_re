package fdother

import (
	"bytes"
	"os"
	"testing"
)

func TestSeparatedLoadSlotsFrameMatchesOriginalEntry16(t *testing.T) {
	const (
		rawPath = "../../../extracted/raw/FDOTHER/FDOTHER_013.bin"
		uiRoot  = "../../generated-assets/fd2-original-b97caf22/ui"
	)
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		t.Skip("player-provided FDOTHER_013 is absent")
	}
	want, err := ParseLMI1OpaqueEntry(raw, 16)
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadSeparatedLoadSlotsFrame(uiRoot)
	if err != nil {
		t.Skipf("separated FDOTHER_013 load-slots pack is absent: %v", err)
	}
	if got.Width != want.Width || got.Height != want.Height ||
		!bytes.Equal(got.Pixels, want.Pixels) {
		t.Fatal("separated load-slots frame differs from FDOTHER #13 entry16")
	}
}

func TestSeparatedLoadSlotsFrameFailsClosedWithoutPack(t *testing.T) {
	if _, err := LoadSeparatedLoadSlotsFrame(t.TempDir()); err == nil {
		t.Fatal("incomplete separated load-slots pack was accepted")
	}
}
