package fdother

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestReadOriginalCommand0NestedSample(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	raw, err := ReadNestedResource(path, 82, 1)
	if os.IsNotExist(err) {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 13424 {
		t.Fatalf("FDOTHER #82/sub1 bytes=%d want 13424", len(raw))
	}
}

func TestReadNestedResourceRejectsNonContainerAndOutOfRange(t *testing.T) {
	data := append([]byte("LLLLLL"), make([]byte, 4)...)
	binary.LittleEndian.PutUint32(data[6:], 10)
	data = append(data, []byte("not nested")...)
	path := filepath.Join(t.TempDir(), "outer.dat")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadNestedResource(path, 0, 0); err == nil {
		t.Fatal("non-container nested resource accepted")
	}
}
