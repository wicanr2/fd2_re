package fdother

import (
	"bytes"
	"os"
	"testing"
)

func TestSeparatedActionCellsMatchOriginalBank(t *testing.T) {
	const (
		rawPath = "../../../extracted/raw/FDOTHER/FDOTHER_002.bin"
		uiRoot  = "../../generated-assets/fd2-original-b97caf22/ui"
	)
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		t.Skip("player-provided FDOTHER_002 is absent")
	}
	want, err := ParseRawCellBank(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadSeparatedActionCells(uiRoot)
	if err != nil {
		t.Skipf("separated action-cell pack is absent: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("cell count=%d want=%d", len(got), len(want))
	}
	for index := range want {
		if got[index].Width != want[index].Width || got[index].Height != want[index].Height ||
			!bytes.Equal(got[index].Pixels, want[index].Pixels) {
			t.Fatalf("action cell %d differs", index)
		}
	}
}

func TestSeparatedActionCellsFailClosedWithoutPack(t *testing.T) {
	if _, err := LoadSeparatedActionCells(t.TempDir()); err == nil {
		t.Fatal("incomplete separated action-cell pack was accepted")
	}
}
