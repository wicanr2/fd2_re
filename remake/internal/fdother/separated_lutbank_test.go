package fdother

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSeparatedLUTBankMatchesFixedResource(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "extracted", "raw", "FDOTHER", "FDOTHER_003.bin"))
	if err != nil {
		t.Skip("player-provided FDOTHER #3 is absent")
	}
	want, err := ParseLUTBank(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadSeparatedLUTBank(filepath.Join("..", "..", "generated-assets", "fd2-original-b97caf22", "palette"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("LUT count=%d, want %d", len(got), len(want))
	}
	for index := range want {
		if !bytes.Equal(got[index], want[index]) {
			t.Fatalf("LUT %d differs", index)
		}
	}
}

func TestSeparatedLUTBankFailsClosed(t *testing.T) {
	if _, err := LoadSeparatedLUTBank(t.TempDir()); err == nil {
		t.Fatal("missing LUT bank was accepted")
	}
}
