package campaign

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func loadNativeChurchOriginalSceneAssets(t *testing.T) (
	[]byte,
	[]fdother.LMI1Entry,
	[]fdother.RawCell,
	[]fdother.Frame,
	dato.Frame,
	*fdtxt.Strings,
	*fdtxt.Font,
) {
	t.Helper()
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided original resources are absent")
	}
	resource14, err := fdother.ReadResource(fdotherPath, 14)
	if err != nil {
		t.Fatal(err)
	}
	entries14 := nativeChurchOriginalTestEntries(t, resource14)
	backgroundFrame, err := fdother.ParseLMI1FrameEntry(resource14, 0)
	if err != nil {
		t.Fatal(err)
	}
	background := make([]byte, 320*200)
	if err := backgroundFrame.BlitAt(background, 320, 0, -1); err != nil {
		t.Fatal(err)
	}
	resource5, err := fdother.ReadResource(fdotherPath, 5)
	if err != nil {
		t.Fatal(err)
	}
	dialogueCells := make([]fdother.RawCell, 18)
	for index := 1; index <= 17; index++ {
		dialogueCells[index], err = fdother.ParseLMI1RawEntry(resource5, index)
		if err != nil {
			t.Fatalf("dialogue cell %d: %v", index, err)
		}
	}
	digitFrames := make([]fdother.Frame, 10)
	for digit := 0; digit < 10; digit++ {
		digitFrames[digit], err = fdother.ParseLMI1FrameEntry(resource5, 31+digit)
		if err != nil {
			t.Fatalf("digit %d: %v", digit, err)
		}
	}
	portraits, err := dato.DecodeResource(filepath.Join(base, "DATO.DAT"), 131)
	if err != nil {
		t.Fatal(err)
	}
	textRaw, err := fdother.ReadResource(filepath.Join(base, "FDTXT.DAT"), 0)
	if err != nil {
		t.Fatal(err)
	}
	strings, err := fdtxt.Parse(textRaw)
	if err != nil {
		t.Fatal(err)
	}
	fontRaw, err := fdother.ReadResource(fdotherPath, 4)
	if err != nil {
		t.Fatal(err)
	}
	font, err := fdtxt.ParseFont(fontRaw)
	if err != nil {
		t.Fatal(err)
	}
	return background, entries14, dialogueCells, digitFrames, portraits[0], strings, font
}

func TestComposeNativeChurchSceneUsesOriginalMixedCodecResources(t *testing.T) {
	background, entries14, dialogueCells, digits, portrait, strings, font :=
		loadNativeChurchOriginalSceneAssets(t)
	for _, textIndex := range []int{585, 586} {
		frame, err := ComposeNativeChurchScene(
			background, entries14[1], dialogueCells, digits, portrait,
			strings, font, 12345678, textIndex,
		)
		if err != nil {
			t.Fatalf("FDTXT%d: %v", textIndex, err)
		}
		if len(frame) != 320*200 {
			t.Fatalf("FDTXT%d frame bytes=%d", textIndex, len(frame))
		}
		if _, err := NativeChurchMenuTransitionFrames(frame, entries14, true); err != nil {
			t.Fatalf("FDTXT%d menu transition: %v", textIndex, err)
		}
	}
}
