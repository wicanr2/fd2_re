package campaign

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func syntheticNativeChurchMenuEntries() []fdother.LMI1Entry {
	entries := make([]fdother.LMI1Entry, 11)
	for option := 0; option < 4; option++ {
		for pulse := 0; pulse < 2; pulse++ {
			index := 2*option + 3 + pulse
			entries[index] = fdother.LMI1Entry{
				Width: 1, Height: 1, Pixels: []byte{byte(index)},
			}
		}
	}
	return entries
}

func TestNativeChurchMenuTransitionUsesRecoveredDivisors(t *testing.T) {
	background := make([]byte, 320*200)
	entries := syntheticNativeChurchMenuEntries()
	opening, err := NativeChurchMenuTransitionFrames(background, entries, true)
	if err != nil {
		t.Fatal(err)
	}
	closing, err := NativeChurchMenuTransitionFrames(background, entries, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(opening) != 4 || len(closing) != 4 {
		t.Fatalf("transition frame counts opening=%d closing=%d", len(opening), len(closing))
	}
	for pass := 0; pass < 4; pass++ {
		openDivisor, closeDivisor := 4-pass, pass+1
		for option, offset := range nativeChurchMenuOffsets {
			index := byte(2*option + 3)
			if got := opening[pass][nativeChurchMenuBase+offset/openDivisor]; got != index {
				t.Fatalf("opening pass%d option%d pixel=%d want%d", pass, option, got, index)
			}
			if got := closing[pass][nativeChurchMenuBase+offset/closeDivisor]; got != index {
				t.Fatalf("closing pass%d option%d pixel=%d want%d", pass, option, got, index)
			}
		}
	}
}

func TestComposeNativeChurchMenuFramePulsesOnlySelectedCell(t *testing.T) {
	frame, err := ComposeNativeChurchMenuFrame(
		make([]byte, 320*200), syntheticNativeChurchMenuEntries(), 2, 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	for option, offset := range nativeChurchMenuOffsets {
		want := byte(2*option + 3)
		if option == 2 {
			want++
		}
		if got := frame[nativeChurchMenuBase+offset]; got != want {
			t.Fatalf("option%d pixel=%d want%d", option, got, want)
		}
	}
	if got := frame[169*320+201+1]; got != 0x4a {
		t.Fatalf("cleared source pixel=%#x", got)
	}
}

func nativeChurchOriginalTestEntries(t *testing.T, resource []byte) []fdother.LMI1Entry {
	t.Helper()
	entries := make([]fdother.LMI1Entry, 17)
	for _, index := range []int{1, 16} {
		cell, err := fdother.ParseLMI1OpaqueEntry(resource, index)
		if err != nil {
			t.Fatal(err)
		}
		entries[index] = cell
	}
	for index := 3; index <= 10; index++ {
		cell, err := fdother.ParseLMI1RawEntry(resource, index)
		if err != nil {
			t.Fatal(err)
		}
		entries[index] = fdother.LMI1Entry{Width: cell.Width, Height: cell.Height, Pixels: cell.Pixels}
	}
	return entries
}

func TestNativeChurchMenuOriginalResourceCells(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	path := filepath.Join(base, "FDOTHER.DAT")
	if _, err := os.Stat(path); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	resource, err := fdother.ReadResource(path, 14)
	if err != nil {
		t.Fatal(err)
	}
	entries := nativeChurchOriginalTestEntries(t, resource)
	backgroundFrame, err := fdother.ParseLMI1FrameEntry(resource, 0)
	if err != nil {
		t.Fatal(err)
	}
	background := make([]byte, 320*200)
	if err := backgroundFrame.BlitAt(background, 320, 0, -1); err != nil {
		t.Fatal(err)
	}
	opening, err := NativeChurchMenuTransitionFrames(background, entries, true)
	if err != nil {
		t.Fatal(err)
	}
	steady, err := ComposeNativeChurchMenuFrame(background, entries, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	for index := 3; index <= 10; index++ {
		if entries[index].Width != 24 || entries[index].Height != 20 {
			t.Fatalf("FDOTHER#14 cell%d=%dx%d want24x20",
				index, entries[index].Width, entries[index].Height)
		}
	}
	if len(opening) != 4 || len(steady) != 320*200 {
		t.Fatal("native church menu original-resource compositor is incomplete")
	}
}
