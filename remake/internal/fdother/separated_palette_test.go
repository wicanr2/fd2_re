package fdother

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSeparatedEndingPaletteMatchesFixedArchive(t *testing.T) {
	root := filepath.Join("..", "..", "generated-assets", "fd2-original-b97caf22", "palette")
	archive := filepath.Join("..", "..", "..", "org_game", "炎龍騎士團", "FLAME2", "FDOTHER.DAT")
	if _, err := os.Stat(filepath.Join(root, "fdother_057.json")); os.IsNotExist(err) {
		t.Skip("separated ending palette is absent")
	} else if err != nil {
		t.Fatal(err)
	}
	want, err := ReadResource(archive, 57)
	if os.IsNotExist(err) {
		t.Skip("player-provided FDOTHER.DAT is absent")
	} else if err != nil {
		t.Fatal(err)
	}
	got, palette, err := LoadSeparatedFDOTHERPalette(root, 57)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) || len(palette) != 256 {
		t.Fatalf("separated ending palette bytes=%d colors=%d", len(got), len(palette))
	}
}

func TestSeparatedEndingPaletteRejectsOutOfRangeComponent(t *testing.T) {
	root := filepath.Join("..", "..", "generated-assets", "fd2-original-b97caf22", "palette")
	raw, err := os.ReadFile(filepath.Join(root, "fdother_057.json"))
	if os.IsNotExist(err) {
		t.Skip("separated ending palette is absent")
	} else if err != nil {
		t.Fatal(err)
	}
	var doc separatedPaletteDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc.Components[0] = 64
	temp := t.TempDir()
	corrupt, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "fdother_057.json"), corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadSeparatedFDOTHERPalette(temp, 57); err == nil {
		t.Fatal("out-of-range ending palette component unexpectedly passed")
	}
}
