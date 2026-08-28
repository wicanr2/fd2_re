package campaign

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func loadNativeTownTextAssets(
	t *testing.T, base string,
) (*fdtxt.Strings, *fdtxt.Font) {
	t.Helper()
	textRaw, err := fdother.ReadResource(filepath.Join(base, "FDTXT.DAT"), 0)
	if err != nil {
		t.Fatal(err)
	}
	strings, err := fdtxt.Parse(textRaw)
	if err != nil {
		t.Fatal(err)
	}
	fontRaw, err := fdother.ReadResource(
		filepath.Join(base, "FDOTHER.DAT"), 4,
	)
	if err != nil {
		t.Fatal(err)
	}
	font, err := fdtxt.ParseFont(fontRaw)
	if err != nil {
		t.Fatal(err)
	}
	return strings, font
}

func TestDecodeAndComposeNativeTownUsesOriginalResources(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided original resources are absent")
	}
	assets, err := DecodeNativeTownAssetsArchive(
		fdotherPath, filepath.Join(base, "FDICON.B24"),
	)
	if err != nil {
		t.Fatal(err)
	}
	for variant, background := range assets.Backgrounds {
		if len(background) != NativeTownWidth*NativeTownHeight {
			t.Fatalf("variant %d background=%d", variant, len(background))
		}
	}
	if assets.Label.Width != 62 || assets.Label.Height != 26 {
		t.Fatalf("label=%dx%d", assets.Label.Width, assets.Label.Height)
	}
	strings, font := loadNativeTownTextAssets(t, base)
	frames := make([][]byte, 4)
	for pulse := range frames {
		frames[pulse], err = ComposeNativeTownFrame(
			assets, strings, font, 0, 0, pulse,
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(frames[pulse]) != NativeTownWidth*NativeTownHeight {
			t.Fatalf("pulse %d frame=%d", pulse, len(frames[pulse]))
		}
	}
	if bytes.Equal(frames[0], frames[1]) ||
		bytes.Equal(frames[1], frames[2]) {
		t.Fatal("town pulse sprites did not change the indexed frame")
	}
	if !bytes.Equal(frames[1], frames[3]) {
		t.Fatal("native pulse sequence did not map cycle 3 back to frame 1")
	}
	for y := 0; y < NativeTownHeight; y++ {
		for x := 0; x < NativeTownWidth; x++ {
			if x >= 4 && x < 316 && y >= 4 && y < 196 {
				continue
			}
			if frames[0][y*NativeTownWidth+x] != 0 {
				t.Fatalf("native viewport border (%d,%d) is not index zero", x, y)
			}
		}
	}
}

func TestComposeNativeTownRejectsInvalidState(t *testing.T) {
	if _, err := ComposeNativeTownFrame(
		nil, nil, nil, 0, 0, 0,
	); err == nil {
		t.Fatal("nil native town assets were accepted")
	}
}
