package fdother

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeNativeChapterAuxSurfaceFromPlayerArchive(t *testing.T) {
	path := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT")
	if _, err := os.Stat(path); err != nil {
		t.Skip("player-provided FDOTHER.DAT unavailable")
	}
	surface, err := DecodeNativeChapterAuxSurface(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(surface.Pixels) != 320*200 {
		t.Fatalf("pixels=%d want %d", len(surface.Pixels), 320*200)
	}
}

func TestBlitNativeChapterAuxViewportUsesRawRowPhaseTable(t *testing.T) {
	pixels := make([]byte, 320*200)
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			pixels[y*320+x] = byte(x)
		}
	}
	dst := make([]byte, 320*192)
	if err := BlitNativeChapterAuxViewport(dst, 320, &NativeChapterAuxSurface{Pixels: pixels}, 15); err != nil {
		t.Fatal(err)
	}
	if got, want := dst[0], byte(1); got != want {
		t.Fatalf("row0 first pixel=%d want %d", got, want)
	}
	if got, want := dst[320], byte(2); got != want {
		t.Fatalf("row1 first pixel=%d want %d", got, want)
	}
	if got := dst[312]; got != 0 {
		t.Fatalf("copy crossed 312-byte viewport: %d", got)
	}
}

func TestBlitNativeChapterAuxViewportRejectsWithoutMutation(t *testing.T) {
	dst := make([]byte, 320*192)
	dst[0] = 0x7a
	if err := BlitNativeChapterAuxViewport(dst, 320, &NativeChapterAuxSurface{Pixels: make([]byte, 320*200)}, 16); err == nil {
		t.Fatal("expected phase rejection")
	}
	if dst[0] != 0x7a {
		t.Fatal("rejected auxiliary viewport mutated destination")
	}
}
