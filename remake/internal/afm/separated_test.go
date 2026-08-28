package afm

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func separatedAnimationRoot() string {
	candidates := []string{
		"../../generated-assets/fd2-original-b97caf22/animations",
		"../../../generated-assets/fd2-original-b97caf22/animations",
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "ANI_000", "animation.json")); err == nil {
			return candidate
		}
	}
	return ""
}

func TestSeparatedANIExactlyMatchesArchiveOracle(t *testing.T) {
	archive, root := aniPath(), separatedAnimationRoot()
	if archive == "" || root == "" {
		t.Skip("玩家自備 ANI.DAT 或本機分離素材包不存在")
	}
	total := 0
	for resource, want := range expectedFrameCounts {
		oracle, err := DecodeResource(archive, resource)
		if err != nil {
			t.Fatalf("archive ANI #%d: %v", resource, err)
		}
		separated, err := LoadSeparatedResource(root, resource)
		if err != nil {
			t.Fatalf("separated ANI #%d: %v", resource, err)
		}
		if separated.Title != strings.TrimRight(oracle.Title, "\x00 ") || separated.HeaderFrames != want {
			t.Fatalf("ANI #%d metadata mismatch", resource)
		}
		for frame := 0; frame < want; frame++ {
			if !bytes.Equal(separated.IndexedFrames[frame], oracle.IndexedFrames[frame]) {
				t.Fatalf("ANI #%d frame %d indexed pixels differ", resource, frame)
			}
			if !bytes.Equal(separated.Palettes[frame], oracle.Palettes[frame]) {
				t.Fatalf("ANI #%d frame %d palette differs", resource, frame)
			}
		}
		total += want
	}
	if total != 289 {
		t.Fatalf("total ANI frames=%d, want 289", total)
	}
	if _, err := os.Stat(filepath.Join(root, "ANI_009")); !os.IsNotExist(err) {
		t.Fatalf("zero-length ANI #9 must not be exported")
	}
}

func TestSeparatedANIFailsClosedOnInvalidMetadata(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "ANI_000")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "animation.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSeparatedResource(root, 0); err == nil {
		t.Fatal("invalid metadata must fail closed")
	}
	if _, err := LoadSeparatedResource(root, 9); err == nil {
		t.Fatal("zero-length tail resource must never load")
	}
}
