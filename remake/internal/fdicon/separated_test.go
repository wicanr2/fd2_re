package fdicon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSeparatedBankMatchesPlayerArchive(t *testing.T) {
	const archive = "../../../org_game/炎龍騎士團/FLAME2/FDICON.B24"
	if _, err := os.Stat(archive); err != nil {
		t.Skip("player-provided FDICON.B24 is absent")
	}
	want, err := DecodeFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadSeparatedBank("../../generated-assets/fd2-original-b97caf22/sprites/fdicon")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Sprites) != len(want.Sprites) || len(got.Sprites) != SeparatedSpriteCount {
		t.Fatalf("separated sprites=%d archive=%d", len(got.Sprites), len(want.Sprites))
	}
	for index := range want.Sprites {
		for layer, pair := range map[string][2][]byte{
			"pixels":     {got.Sprites[index].Pixels, want.Sprites[index].Pixels},
			"mask":       {got.Sprites[index].Mask, want.Sprites[index].Mask},
			"remap_mask": {got.Sprites[index].RemapMask, want.Sprites[index].RemapMask},
		} {
			if string(pair[0]) != string(pair[1]) {
				t.Fatalf("sprite %d %s differs from archive", index, layer)
			}
		}
	}
}

func TestSeparatedBankRejectsIncompletePack(t *testing.T) {
	root := t.TempDir()
	document := `{"schema_version":1,"kind":"fdicon_sprite_bank","asset_id":"sprites/fdicon","status":"decoded","evidence":"confirmed","source":{"file":"FDICON.B24","size":624010,"md5":"46f793540209a063ea73a5373ca14bf4","sha256":"7efb4448d05f19c1e17ebd53f3e3afead235f5c008d5167548d834c3686b1e44"},"sprites":[]}`
	if err := os.WriteFile(filepath.Join(root, "bank.json"), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSeparatedBank(root); err == nil {
		t.Fatal("incomplete separated FDICON bank was accepted")
	}
}
