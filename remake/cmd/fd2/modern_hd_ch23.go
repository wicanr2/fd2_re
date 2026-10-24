package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type modernHDCh23Source struct {
	input                       indexedmap.NativeFrameInput
	seed, initialVGA, work, vga []byte
	palette                     color.Palette
	stage                       *nativeCh23AdapterState
	mapIndex                    int
	signature                   [32]byte
}

func (s *modernHDCh23Source) digest() ([32]byte, error) {
	if s == nil || s.stage == nil {
		return [32]byte{}, errors.New("modern HD ch23: frozen source absent")
	}
	identity, err := indexedHDInputIdentity(nativeTransitionInput(s.input))
	if err != nil {
		return [32]byte{}, err
	}
	raw, err := json.Marshal([]any{identity, s.input.HUD, s.input.Frame.RangeMode, s.input.Frame.CursorX, s.input.Frame.CursorY,
		s.input.Frame.ChapterAuxPhase, s.mapIndex, s.seed, s.initialVGA, s.work, s.vga, s.palette, s.stage.staging, s.stage.rowShift, s.stage.latch, s.stage.tickSnapshot, s.stage.initialComplete})
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(raw), nil
}

func freezeModernHDCh23Source(g *Game, stage *nativeCh23AdapterState, seed, initialVGA []byte, palette color.Palette) (*modernHDCh23Source, error) {
	if g.nativeMapFrameSource == nil || g.nativeMapAssets == nil {
		return nil, errors.New("modern HD ch23: steady producer absent")
	}
	in, err := freezeModernHDCommandGeometry(*g.nativeMapFrameSource)
	if err != nil {
		return nil, err
	}
	in.Frame.SelectorCache = in.Frame.SelectorCache.Clone()
	in.HUDCache = in.Frame.SelectorCache
	s := &modernHDCh23Source{input: in, stage: cloneNativeCh23AdapterState(stage), mapIndex: g.nativeMapAssets.MapIndex,
		seed: append([]byte(nil), seed...), initialVGA: append([]byte(nil), initialVGA...), work: append([]byte(nil), g.nativeMapWork...), vga: append([]byte(nil), g.nativeMapVGA...), palette: append(color.Palette(nil), palette...)}
	s.signature, err = s.digest()
	return s, err
}

func (g *Game) prepareModernHDCh23(j *nativeCh23LoopJob, set *modernStoryPortraitSet) (*modernHDMapFrame, error) {
	if set == nil || len(set.mapSpritesHD) == 0 {
		return nil, nil
	}
	if len(set.mapSpritesHD) != 125 {
		return nil, errors.New("modern HD ch23: incomplete HD world")
	}
	if j == nil || j.hdInputErr != nil {
		if j != nil {
			return nil, j.hdInputErr
		}
		return nil, errors.New("modern HD ch23: owner absent")
	}
	s := j.hdSource
	if s == nil || g.nativeMapAssets == nil || s.mapIndex != g.nativeMapAssets.MapIndex || s.input.Frame.TerrainBank != g.nativeMapAssets.Terrain ||
		s.input.Frame.UnitBank != g.nativeMapAssets.Units || s.input.Frame.ForegroundBank != g.nativeMapAssets.Terrain || s.input.Frame.RangeBank != g.nativeMapAssets.Range {
		return nil, errors.New("modern HD ch23: immutable native producer changed")
	}
	digest, err := s.digest()
	if err != nil || digest != s.signature {
		return nil, errors.New("modern HD ch23: frozen source changed")
	}
	in, err := freezeModernHDCommandGeometry(s.input)
	if err != nil {
		return nil, err
	}
	work, vga := append([]byte(nil), s.seed...), append([]byte(nil), s.initialVGA...)
	if err := indexedmap.ComposeNativeFrame(work, vga, in); err != nil {
		return nil, err
	}
	if !bytes.Equal(work, s.work) || !bytes.Equal(vga, s.vga) {
		return nil, errors.New("modern HD ch23: native replay differs")
	}
	view := *g
	view.modernStoryPortraits = set
	frame, err := view.prepareModernHDMapFrameStageLayout(in, s.seed, s.palette, true, -1, s.stage)
	if err != nil {
		return nil, err
	}
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			if x < 4 || x >= 316 || y < 4 || y >= 196 {
				frame.Background.Set(x, y, s.palette[s.vga[y*320+x]])
			}
		}
	}
	frame.OverlaysAboveUI = true
	return frame, nil
}

func (g *Game) stageModernHDCh23(set *modernStoryPortraitSet) (func(), error) {
	j := g.nativeCh23Loop
	if j == nil {
		return func() {}, nil
	}
	frame, err := g.prepareModernHDCh23(j, set)
	if err != nil {
		return nil, err
	}
	return func() { j.hdFrame, j.hdTheme = frame, set }, nil
}
