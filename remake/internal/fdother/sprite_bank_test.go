package fdother

import (
	"bytes"
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

func TestDecodeFDSHAP000SpriteBankFromPlayerArchive(t *testing.T) {
	const datPath = "../../../org_game/炎龍騎士團/FLAME2/FDSHAP.DAT"
	if _, err := os.Stat(datPath); os.IsNotExist(err) {
		t.Skip("player-provided FDSHAP.DAT is absent")
	}
	bank, err := DecodeSpriteBankResource(datPath, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(bank.Sprites) != 288 {
		t.Fatalf("FDSHAP#0 sprite count=%d, want 288", len(bank.Sprites))
	}
}

func TestSeparatedFDSHAPBanksMatchAllPlayerArchiveResources(t *testing.T) {
	const datPath = "../../../org_game/炎龍騎士團/FLAME2/FDSHAP.DAT"
	if _, err := os.Stat(datPath); err != nil {
		t.Skip("player-provided FDSHAP.DAT is absent")
	}
	const root = "../../generated-assets/fd2-original-b97caf22/tilesets/fdshap"
	for mapIndex := 0; mapIndex < fdicon.SeparatedFDSHAPMapCount; mapIndex++ {
		wantBank, wantControls, err := DecodeMapTerrainResources(datPath, mapIndex)
		if err != nil {
			t.Fatalf("archive map%d: %v", mapIndex, err)
		}
		gotBank, gotControls, err := fdicon.LoadSeparatedFDSHAPBank(root, mapIndex)
		if err != nil {
			t.Fatalf("separated map%d: %v", mapIndex, err)
		}
		if !bytes.Equal(gotControls, wantControls) || len(gotBank.Sprites) != len(wantBank.Sprites) {
			t.Fatalf("map%d sprites=%d/%d controls_equal=%v", mapIndex, len(gotBank.Sprites), len(wantBank.Sprites), bytes.Equal(gotControls, wantControls))
		}
		for index := range wantBank.Sprites {
			if !bytes.Equal(gotBank.Sprites[index].Pixels, wantBank.Sprites[index].Pixels) ||
				!bytes.Equal(gotBank.Sprites[index].Mask, wantBank.Sprites[index].Mask) ||
				!bytes.Equal(gotBank.Sprites[index].RemapMask, wantBank.Sprites[index].RemapMask) {
				t.Fatalf("map%d tile%d separated layers differ", mapIndex, index)
			}
		}
	}
}

func TestDecodeMapTerrainResourcesPairsFDSHAPMapZero(t *testing.T) {
	const datPath = "../../../org_game/炎龍騎士團/FLAME2/FDSHAP.DAT"
	if _, err := os.Stat(datPath); os.IsNotExist(err) {
		t.Skip("player-provided FDSHAP.DAT is absent")
	}
	bank, controls, err := DecodeMapTerrainResources(datPath, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(bank.Sprites) != 288 || len(controls) != 1200 {
		t.Fatalf("map0 image/control=%d/%d, want 288/1200", len(bank.Sprites), len(controls))
	}
	if _, _, err := DecodeMapTerrainResources(datPath, -1); err == nil {
		t.Fatal("negative map index accepted")
	}
}

func TestDecodeMapTerrainResourcesAllowsTrailingRendererFrames(t *testing.T) {
	const datPath = "../../../org_game/炎龍騎士團/FLAME2/FDSHAP.DAT"
	if _, err := os.Stat(datPath); err != nil {
		t.Skip("player-provided FDSHAP.DAT is absent")
	}
	bank, controls, err := DecodeMapTerrainResources(datPath, 17)
	if err != nil {
		t.Fatal(err)
	}
	if len(bank.Sprites) != 384 || len(controls) != 330*4 {
		t.Fatalf("map17 sprites=%d controls=%d", len(bank.Sprites), len(controls))
	}
}
