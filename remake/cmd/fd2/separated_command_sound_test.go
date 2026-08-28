package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSeparatedCommandSoundBanksDecodeWithoutOriginalArchive(t *testing.T) {
	pack := filepath.Clean("../../generated-assets/fd2-original-b97caf22")
	if _, err := os.Stat(filepath.Join(pack, "sfx", "FDOTHER_082", "resource.json")); err != nil {
		t.Skipf("separated sound pack is absent: %v", err)
	}
	t.Setenv("FD2_ASSET_PACK", pack)
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(t.TempDir(), "missing-FDOTHER.DAT"))
	banks, err := loadSeparatedCommandSoundBanks()
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]int{82: 2, 83: 4, 84: 3, 85: 2, 86: 2, 87: 4, 88: 2, 90: 3}
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
