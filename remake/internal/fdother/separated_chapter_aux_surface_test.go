package fdother

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSeparatedChapterAuxSurfaceMatchesFixedResource(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "extracted", "raw", "FDOTHER", "FDOTHER_055.bin"))
	if err != nil {
		t.Skip("player-provided FDOTHER #55 is absent")
	}
	got, err := LoadSeparatedChapterAuxSurface(filepath.Join("..", "..", "generated-assets", "fd2-original-b97caf22", "surfaces"))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 4+320*200 || !bytes.Equal(got.Pixels, raw[4:]) {
		t.Fatal("separated FDOTHER #55 pixels differ")
	}
}

func TestSeparatedChapterAuxSurfaceFailsClosed(t *testing.T) {
	if _, err := LoadSeparatedChapterAuxSurface(t.TempDir()); err == nil {
		t.Fatal("missing chapter auxiliary surface was accepted")
	}
}
