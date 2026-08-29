package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSeparatedCommandSoundBanksDecodeWithoutOriginalArchive(t *testing.T) {
	pack := filepath.Clean("../../generated-assets/fd2-original-b97caf22")
	if _, err := os.Stat(filepath.Join(pack, "sfx", "FDOTHER_080", "resource.json")); err != nil {
		t.Skipf("separated sound pack is absent: %v", err)
	}
	t.Setenv("FD2_ASSET_PACK", pack)
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(t.TempDir(), "missing-FDOTHER.DAT"))
	banks, err := loadSeparatedCommandSoundBanks()
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]int{
		50: 5, 53: 4, 77: 4, 80: 16, 82: 2, 83: 4, 84: 3, 85: 2, 86: 2, 87: 4, 88: 2, 90: 3,
		91: 3, 92: 2, 93: 2, 94: 3, 95: 1,
	}
	for resource, count := range want {
		if len(banks[resource]) != count {
			t.Fatalf("resource %d samples=%d want=%d", resource, len(banks[resource]), count)
		}
		for subresource := 0; subresource < count; subresource++ {
			if len(banks[resource][subresource]) == 0 {
				t.Fatalf("resource %d subresource %d decoded empty", resource, subresource)
			}
		}
	}
}

func TestInstallSeparatedCommandSoundsUsesTitleBankSelectors(t *testing.T) {
	banks := map[int]map[int][]byte{77: {1: []byte("confirm"), 2: []byte("move")}}
	g := &Game{}
	g.installSeparatedCommandSounds(banks)
	if string(g.sfxTitleMove) != "move" || string(g.sfxTitleConfirm) != "confirm" {
		t.Fatalf("title sounds move=%q confirm=%q", g.sfxTitleMove, g.sfxTitleConfirm)
	}
}

func TestSeparatedCommandSoundBanksFailClosedWithoutPack(t *testing.T) {
	t.Setenv("FD2_ASSET_PACK", t.TempDir())
	if _, err := loadSeparatedCommandSoundBanks(); err == nil {
		t.Fatal("missing separated command sound banks were accepted")
	}
}

func TestSeparatedUISoundBankDecodesWithoutOriginalArchive(t *testing.T) {
	pack := filepath.Clean("../../generated-assets/fd2-original-b97caf22")
	if _, err := os.Stat(filepath.Join(pack, "sfx", "FDOTHER_031", "resource.json")); err != nil {
		t.Skipf("separated UI sound pack is absent: %v", err)
	}
	t.Setenv("FD2_ASSET_PACK", pack)
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(t.TempDir(), "missing-FDOTHER.DAT"))
	bank, err := loadSFX()
	if err != nil {
		t.Fatal(err)
	}
	if len(bank) != 13 {
		t.Fatalf("UI sound samples=%d want=13", len(bank))
	}
	for subresource := 0; subresource < 13; subresource++ {
		if len(bank[subresource]) == 0 {
			t.Fatalf("UI sound subresource %d decoded empty", subresource)
		}
	}
}

func TestSeparatedUISoundBankFailsClosedWithoutPack(t *testing.T) {
	t.Setenv("FD2_ASSET_PACK", t.TempDir())
	if _, err := loadSFX(); err == nil {
		t.Fatal("missing separated UI sound bank was accepted")
	}
}
