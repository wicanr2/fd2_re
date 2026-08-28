package dato

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writeSeparatedPortrait(t *testing.T, root string, resource, count int, rgba bool) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for mouth := 0; mouth < count; mouth++ {
		path := filepath.Join(root, fmt.Sprintf("DATO_%03d_m%d.png", resource, mouth))
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		var source image.Image
		if rgba {
			source = image.NewNRGBA(image.Rect(0, 0, 2, 2))
		} else {
			palette := color.Palette{color.Black, color.White, color.RGBA{R: 255, A: 255}}
			indexed := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
			indexed.Pix = []byte{byte(mouth % 3), 1, 2, 0}
			source = indexed
		}
		if err := png.Encode(file, source); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadSeparatedResourcePreservesIndexedFrames(t *testing.T) {
	root := t.TempDir()
	writeSeparatedPortrait(t, root, 26, 4, false)
	frames, err := LoadSeparatedResource(root, 26)
	if err != nil || len(frames) != 4 {
		t.Fatalf("frames=%d err=%v", len(frames), err)
	}
	for mouth, frame := range frames {
		if frame.Width != 2 || frame.Height != 2 || len(frame.Pixels) != 4 || frame.Pixels[0] != byte(mouth%3) {
			t.Fatalf("frame %d=%+v", mouth, frame)
		}
	}
}

func TestLoadSeparatedResourceFailsClosed(t *testing.T) {
	root := t.TempDir()
	writeSeparatedPortrait(t, root, 26, 3, false)
	if _, err := LoadSeparatedResource(root, 26); err == nil {
		t.Fatal("three-frame portrait was accepted")
	}
	root = t.TempDir()
	writeSeparatedPortrait(t, root, 26, 4, true)
	if _, err := LoadSeparatedResource(root, 26); err == nil {
		t.Fatal("RGBA portrait was accepted as indexed provenance")
	}
}

func TestSeparatedResourceMatchesOriginalArchive(t *testing.T) {
	archive := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2/DATO.DAT")
	portraitRoot := filepath.Clean("../../generated-assets/fd2-original-b97caf22/portraits")
	if _, err := os.Stat(archive); err != nil {
		t.Skipf("player-provided DATO.DAT is absent: %v", err)
	}
	if _, err := os.Stat(filepath.Join(portraitRoot, "DATO_026_m3.png")); err != nil {
		t.Skipf("separated portrait pack is absent: %v", err)
	}
	want, err := DecodeResource(archive, 26)
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadSeparatedResource(portraitRoot, 26)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("frames=%d, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index].Width != want[index].Width || got[index].Height != want[index].Height ||
			!bytes.Equal(got[index].Pixels, want[index].Pixels) {
			t.Fatalf("frame %d differs from original archive", index)
		}
	}
}
