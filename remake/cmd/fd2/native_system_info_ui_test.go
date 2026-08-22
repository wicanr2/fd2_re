package main

import (
	"encoding/binary"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func nativeSystemInfoRuntimeTestAssets(t *testing.T) *campaign.NativeSystemInfoAssets {
	t.Helper()
	const stringCount = 700
	text := make([]byte, stringCount*2+stringCount*4)
	for index := 0; index < stringCount; index++ {
		offset := stringCount*2 + index*4
		binary.LittleEndian.PutUint16(text[index*2:], uint16(offset))
		binary.LittleEndian.PutUint16(text[offset:], 0)
		binary.LittleEndian.PutUint16(text[offset+2:], fdtxt.StringEnd)
	}
	strings, err := fdtxt.Parse(text)
	if err != nil {
		t.Fatal(err)
	}
	fontRaw := make([]byte, fdtxt.GlyphBytes)
	fontRaw[0] = 0x80
	font, err := fdtxt.ParseFont(fontRaw)
	if err != nil {
		t.Fatal(err)
	}
	assets := &campaign.NativeSystemInfoAssets{
		Numbers: battle.NativeItemPanelDataAssets{
			Frames: map[int]fdother.Frame{}, Strings: strings, Font: font,
		},
		Font: font,
	}
	geometry := [4][2]int{{102, 17}, {170, 117}, {170, 16}, {63, 15}}
	for index, size := range geometry {
		assets.Panels[index] = fdother.LMI1Entry{
			Width: size[0], Height: size[1], Pixels: make([]byte, size[0]*size[1]),
		}
	}
	for index := 31; index <= 51; index++ {
		assets.Numbers.Frames[index] = fdother.Frame{
			Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, byte(index)},
		}
	}
	return assets
}

func nativeSystemInfoRuntimeTestGame(t *testing.T) *Game {
	t.Helper()
	unit := &battle.Unit{
		NativeRecordByte5: 0x80, HasNativeRecordByte5: true,
		NativeRecordByte6: 0, HasNativeRecordByte6: true,
		BattleFig: 1, HasBattleFig: true,
		NativeRecordRace: 1, HasNativeRecordRace: true,
	}
	return &Game{
		st:             &battle.State{Units: []*battle.Unit{unit}, NativeRoundCounter: 3},
		handlerChapter: 0, gold: 123,
		nativeMapVGA:           make([]byte, fdother.NativeSystemInfoBytes),
		nativeMapDAC:           make([]byte, 256*3),
		nativeSystemInfoAssets: nativeSystemInfoRuntimeTestAssets(t),
		partyMembers:           map[int]bool{},
	}
}

func TestPrepareNativeSystemInfoUIRunsTwelveOpenWaitTwelveClose(t *testing.T) {
	g := nativeSystemInfoRuntimeTestGame(t)
	state, err := g.prepareNativeSystemInfoUI()
	if err != nil || len(state.opening) != 12 || len(state.closing) != 12 ||
		len(state.steady) != fdother.NativeSystemInfoBytes {
		t.Fatalf("prepared state=%#v err=%v", state, err)
	}
	g.nativeSystemInfoUI = state
	for present := 0; present < 12; present++ {
		g.nativeSystemInfoUI.drawn = true
		g.stepNativeSystemInfoUI()
	}
	if g.nativeSystemInfoUI == nil || g.nativeSystemInfoUI.phase != "steady" {
		t.Fatalf("opening did not reach steady state: %#v", g.nativeSystemInfoUI)
	}
	phase := g.nativeFDOTHERPalettePhase
	g.nativeSystemInfoUI.drawn = true
	g.stepNativeSystemInfoUI()
	if g.nativeFDOTHERPalettePhase != (phase+1)&15 {
		t.Fatalf("steady palette phase=%d, want %d", g.nativeFDOTHERPalettePhase, (phase+1)&15)
	}
	g.closeNativeSystemInfoUI()
	for present := 0; present < 12; present++ {
		g.nativeSystemInfoUI.drawn = true
		g.stepNativeSystemInfoUI()
	}
	if g.nativeSystemInfoUI != nil {
		t.Fatalf("closing did not restore battlefield: %#v", g.nativeSystemInfoUI)
	}
}

func TestPrepareNativeSystemInfoUIFailsBeforePublicationWithoutRawCount(t *testing.T) {
	g := nativeSystemInfoRuntimeTestGame(t)
	g.st.Units[0].HasNativeRecordRace = false
	if state, err := g.prepareNativeSystemInfoUI(); err == nil || state != nil {
		t.Fatalf("missing provenance published state=%#v err=%v", state, err)
	}
}

func TestNestedSystemOverlayUsesRuntimeSaveAndFD2SAVGates(t *testing.T) {
	g := nativeSystemInfoRuntimeTestGame(t)
	g.nativeActionCells = make([]*ebiten.Image, nativeActionOverlayCellCount)
	for _, index := range []int{36, 41, 44, 45} {
		g.nativeActionCells[index] = ebiten.NewImage(1, 1)
	}
	t.Setenv("FD2_NATIVE_SAVE", filepath.Join(t.TempDir(), "missing-FD2.SAV"))
	state, ok := g.nativeNestedSystemOverlayState()
	if !ok {
		t.Fatal("nested system overlay was rejected")
	}
	want := [4]int{36, 41, 44, 45}
	for direction := range want {
		got, err := state.CellIndex(direction)
		if err != nil || got != want[direction] {
			t.Fatalf("direction %d cell=%d err=%v, want %d", direction, got, err, want[direction])
		}
	}
}
