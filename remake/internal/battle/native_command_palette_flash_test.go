package battle

import "testing"

func TestLoadNativeCommandPaletteFlashTablePreservesIDABytes(t *testing.T) {
	table, err := LoadNativeCommandPaletteFlashTable("../../assets/data/native_command_palette_flash.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{17, 18, 19} {
		phases, err := table.NativeCommandPaletteFlashPhases(id)
		if err != nil {
			t.Fatal(err)
		}
		if len(phases) != 8 {
			t.Fatalf("command %d phases=%d", id, len(phases))
		}
		for i, rgb := range phases {
			want := [3]byte{}
			if i%2 == 0 {
				want = [3]byte{0x32, 0x32, 0x32}
			}
			if rgb != want {
				t.Fatalf("command %d phase %d=%v want %v", id, i, rgb, want)
			}
		}
	}
}

func TestNativeCommandPaletteFlashPhasesRejectsUnknownID(t *testing.T) {
	table := &NativeCommandPaletteFlashTable{Cycles: 4, TicksPerPhase: 1, Entries: make([][3]byte, 36)}
	if _, err := table.NativeCommandPaletteFlashPhases(-1); err == nil {
		t.Fatal("negative command accepted")
	}
	if _, err := table.NativeCommandPaletteFlashPhases(36); err == nil {
		t.Fatal("out-of-range command accepted")
	}
}
