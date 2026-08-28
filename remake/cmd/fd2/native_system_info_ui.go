package main

import (
	"errors"
	"image/color"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type nativeSystemInfoUIState struct {
	opening [][]byte
	steady  []byte
	closing [][]byte
	phase   string
	frame   int
	drawn   bool
	dac     []byte
	palette color.Palette
}

func loadNativeSystemInfoAssets() (*campaign.NativeSystemInfoAssets, error) {
	fdotherPath := nativeFDOTHERPath()
	if fdotherPath == "" {
		return nil, errors.New("native system info: FDOTHER.DAT unavailable")
	}
	panels, err := fdother.DecodeLMI1Resource(fdotherPath, 5)
	if err != nil || len(panels) <= 0x88 {
		return nil, errors.New("native system info: FDOTHER#5 panels 0x85..0x88 unavailable")
	}
	numbers, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		return nil, err
	}
	assets := &campaign.NativeSystemInfoAssets{Numbers: numbers, Font: numbers.Font}
	copy(assets.Panels[:], panels[0x85:0x89])
	return assets, nil
}

func nativeCurrentRuntimeSaveExists() bool {
	path := os.Getenv("FD2_NATIVE_SAVE")
	if path == "" {
		if fdotherPath := nativeFDOTHERPath(); fdotherPath != "" {
			path = filepath.Join(filepath.Dir(fdotherPath), "FD2.SAV")
		}
	}
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (g *Game) nativeNestedSystemOverlayState() (fdother.ActionOverlayState, bool) {
	if g == nil || g.st == nil {
		return fdother.ActionOverlayState{}, false
	}
	saveGateSet := false
	for _, unit := range g.st.Units {
		if unit == nil || !unit.HasNativeRecordByte5 {
			return fdother.ActionOverlayState{}, false
		}
		if unit.NativeRecordByte5&1 == 0 && unit.NativeRecordByte5&0x80 != 0 {
			saveGateSet = true
		}
	}
	state := fdother.NativeNestedSystemActionOverlayState(
		saveGateSet, !nativeCurrentRuntimeSaveExists(),
	)
	for direction := 0; direction < 4; direction++ {
		index, err := state.CellIndex(direction)
		if err != nil || index < 0 || index >= len(g.nativeActionCells) || g.nativeActionCells[index] == nil {
			return fdother.ActionOverlayState{}, false
		}
	}
	return state, true
}

func (g *Game) prepareNativeSystemInfoUI() (*nativeSystemInfoUIState, error) {
	if g == nil || g.nativeSystemInfoAssets == nil || g.st == nil ||
		len(g.nativeMapVGA) != fdother.NativeSystemInfoBytes ||
		len(g.nativeMapDAC) != 256*3 || g.handlerChapter < 0 ||
		g.handlerChapter >= 30 || g.st.NativeRoundCounter < 0 ||
		g.st.NativeRoundCounter > 0xff || g.gold < 0 || uint64(g.gold) > 99999999 {
		return nil, errors.New("native system info: runtime source is incomplete")
	}
	counts, err := campaign.NativeSystemInfoCampCounts(g.st.Units)
	if err != nil {
		return nil, err
	}
	textIndex := 2*g.handlerChapter + 0x255
	if g.handlerChapter == 16 && !g.partyMembers[18] {
		textIndex -= 2
	}
	strings := g.nativeSystemInfoAssets.Numbers.Strings
	if strings == nil {
		return nil, errors.New("native system info: FDTXT strings unavailable")
	}
	line0, err := strings.Words(textIndex)
	if err != nil {
		return nil, err
	}
	line1, err := strings.Words(textIndex + 1)
	if err != nil {
		return nil, err
	}
	information, err := campaign.ComposeNativeSystemInfoSurface(
		*g.nativeSystemInfoAssets,
		campaign.NativeSystemInfoInput{
			RawChapter: byte(g.handlerChapter),
			RawRound:   byte(g.st.NativeRoundCounter),
			Currency:   uint32(g.gold), RawCampCounts: counts,
			Lines: [2][]uint16{line0, line1},
		},
	)
	if err != nil {
		return nil, err
	}
	opening, closing, err := fdother.NativeSystemInfoTransitionFrames(g.nativeMapVGA, information)
	if err != nil {
		return nil, err
	}
	dac := append([]byte(nil), g.nativeMapDAC...)
	palette, err := fdother.VGAPaletteFromDAC(dac)
	if err != nil {
		return nil, err
	}
	return &nativeSystemInfoUIState{
		opening: opening, steady: information, closing: closing,
		phase: "opening", dac: dac, palette: palette,
	}, nil
}

func (g *Game) stepNativeSystemInfoUI() {
	state := g.nativeSystemInfoUI
	if state == nil || !state.drawn {
		return
	}
	state.drawn = false
	if state.phase == "steady" {
		g.nativeFDOTHERPalettePhase = (g.nativeFDOTHERPalettePhase + 1) & 15
		if err := fdother.ApplyNativeDACPaletteCycleE0EF(state.dac, g.nativeFDOTHERPalettePhase); err != nil {
			g.loadErr = "native system info palette cycle: " + err.Error()
			g.nativeSystemInfoUI = nil
			return
		}
		palette, err := fdother.VGAPaletteFromDAC(state.dac)
		if err != nil {
			g.loadErr = "native system info palette: " + err.Error()
			g.nativeSystemInfoUI = nil
			return
		}
		state.palette = palette
		return
	}
	state.frame++
	frames := state.opening
	if state.phase == "closing" {
		frames = state.closing
	}
	if state.frame < len(frames) {
		return
	}
	if state.phase == "opening" {
		state.phase, state.frame = "steady", 0
		return
	}
	g.nativeSystemInfoUI = nil
}

func (g *Game) closeNativeSystemInfoUI() {
	if g.nativeSystemInfoUI == nil || g.nativeSystemInfoUI.phase != "steady" {
		return
	}
	g.nativeSystemInfoUI.phase = "closing"
	g.nativeSystemInfoUI.frame = 0
	g.nativeSystemInfoUI.drawn = false
}

func (g *Game) drawNativeSystemInfoUI(screen *ebiten.Image) bool {
	state := g.nativeSystemInfoUI
	if state == nil {
		return false
	}
	frame := state.steady
	if state.phase == "opening" && state.frame < len(state.opening) {
		frame = state.opening[state.frame]
	} else if state.phase == "closing" && state.frame < len(state.closing) {
		frame = state.closing[state.frame]
	}
	if len(frame) != fdother.NativeSystemInfoBytes {
		return false
	}
	if len(state.palette) != 256 {
		return false
	}
	g.presentNativeClassFrameWithPalette(screen, frame, state.palette)
	state.drawn = true
	return true
}
