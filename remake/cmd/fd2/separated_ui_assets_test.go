package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writeSeparatedCommandGridFixture(t *testing.T, root string, count int) {
	t.Helper()
	components := make([]int, 256*3)
	for index := range components {
		components[index] = index % 64
	}
	document := separatedPaletteDocument{
		SchemaVersion: 1,
		AssetID:       "palette/fdother_000",
		Components:    components,
	}
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	palettePath := filepath.Join(root, "palette", "fdother_000.json")
	if err := os.MkdirAll(filepath.Dir(palettePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(palettePath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "ui", "action_cells")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < count; index++ {
		file, err := os.Create(filepath.Join(directory, "cell_"+fmtThreeDigits(index)+".png"))
		if err != nil {
			t.Fatal(err)
		}
		im := image.NewNRGBA(image.Rect(0, 0, 2, 2))
		im.SetNRGBA(0, 0, color.NRGBA{R: byte(index), A: 0xff})
		if err := png.Encode(file, im); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func fmtThreeDigits(value int) string {
	return string([]byte{'0' + byte(value/100), '0' + byte(value/10%10), '0' + byte(value%10)})
}

func TestSeparatedCommandGridLoadsWithoutOriginalArchive(t *testing.T) {
	root := t.TempDir()
	writeSeparatedCommandGridFixture(t, root, nativeActionOverlayCellCount)
	t.Setenv("FD2_ASSET_PACK", root)
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(t.TempDir(), "unreadable.dat"))

	palette := loadNativeUIPalette()
	if len(palette) != 256 {
		t.Fatalf("palette=%d, want 256", len(palette))
	}
	cells := loadNativeActionCells(palette)
	if len(cells) != nativeActionOverlayCellCount {
		t.Fatalf("cells=%d, want %d", len(cells), nativeActionOverlayCellCount)
	}
}

func TestSeparatedCommandGridFailsClosedOnMissingCell(t *testing.T) {
	root := t.TempDir()
	writeSeparatedCommandGridFixture(t, root, nativeActionOverlayCellCount-1)
	t.Setenv("FD2_ASSET_PACK", root)
	if cells := loadNativeActionCells(loadNativeUIPalette()); cells != nil {
		t.Fatalf("incomplete cell bank published %d cells", len(cells))
	}
}

func TestSeparatedPaletteRejectsOutOfRangeDAC(t *testing.T) {
	root := t.TempDir()
	writeSeparatedCommandGridFixture(t, root, nativeActionOverlayCellCount)
	path := filepath.Join(root, "palette", "fdother_000.json")
	var document separatedPaletteDocument
	raw, _ := os.ReadFile(path)
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	document.Components[0] = 64
	raw, _ = json.Marshal(document)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_ASSET_PACK", root)
	if palette := loadNativeUIPalette(); palette != nil {
		t.Fatalf("invalid DAC published %d colors", len(palette))
	}
	if cells := loadNativeActionCells(loadNativeUIPalette()); cells != nil {
		t.Fatalf("action cells published without a valid shared palette")
	}
}
