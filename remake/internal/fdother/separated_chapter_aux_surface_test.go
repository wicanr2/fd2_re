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

func TestNativeChapterAuxSurfaceForFollowsSub10652Branches(t *testing.T) {
	// sub_10652：raw chapter 9/24/25 → FDOTHER #15，28/29 → #55，其他章節沒有底面。
	for chapter, want := range map[int]int{9: 15, 24: 15, 25: 15, 28: 55, 29: 55} {
		got, ok := NativeChapterAuxSurfaceFor(chapter)
		if !ok || got.Resource != want {
			t.Fatalf("chapter %d → %+v ok=%v, want #%d", chapter, got, ok, want)
		}
	}
	for _, chapter := range []int{0, 8, 10, 23, 30} {
		if _, ok := NativeChapterAuxSurfaceFor(chapter); ok {
			t.Fatalf("chapter %d unexpectedly has an auxiliary surface", chapter)
		}
	}
}

func TestSeparatedChapterAuxSurface15MatchesFixedIdentity(t *testing.T) {
	root := filepath.Join("..", "..", "generated-assets", "fd2-original-b97caf22", "surfaces")
	if _, err := os.Stat(filepath.Join(root, "FDOTHER_015", "resource.json")); err != nil {
		t.Skip("separated FDOTHER #15 is absent")
	}
	contract, _ := NativeChapterAuxSurfaceFor(9)
	got, err := LoadSeparatedChapterAuxSurfaceResource(root, contract)
	if err != nil || len(got.Pixels) != 320*200 {
		t.Fatalf("FDOTHER #15 load: %v", err)
	}
	wrong, _ := NativeChapterAuxSurfaceFor(28)
	wrong.Resource = 15
	if _, err := LoadSeparatedChapterAuxSurfaceResource(root, wrong); err == nil {
		t.Fatal("FDOTHER #15 accepted #55's raw identity")
	}
}
