package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type modernHDIndexedTransitionSource struct {
	Input       indexedmap.NativeFrameInput
	MapIndex    int
	Palette     color.Palette
	LUTs        [][]byte
	Passes      []fdother.IndexedTransitionPass
	Backgrounds [][]byte
}

func indexedHDInputIdentity(in indexedmap.NativeTransitionFrameInput) ([]any, error) {
	if in.SelectorCache == nil {
		return nil, errors.New("modern HD indexed transition: selector cache absent")
	}
	keys := make([]int, len(in.Units))
	for i, unit := range in.Units {
		key, err := in.SelectorCache.KeyForSlot(unit.Slot)
		if err != nil {
			return nil, err
		}
		keys[i] = key
	}
	var parallax []byte
	if in.Parallax != nil {
		parallax = in.Parallax.Pixels
	}
	return []any{in.Cells, in.Controls, in.TerrainLUT, in.Units, in.ForegroundUnits, keys, in.MapWidth, in.CameraX, in.CameraY,
		in.Flip, in.TerrainCycle, in.IdleCycle, in.MovingCycle, in.PixelShift, in.ParallaxScrollX, in.ParallaxScrollY, parallax}, nil
}

func (j *nativeIndexedTransitionJob) modernHDSignature() ([32]byte, error) {
	s := j.hdSource
	if s == nil {
		return [32]byte{}, errors.New("modern HD indexed transition: source absent")
	}
	frozen, err := indexedHDInputIdentity(nativeTransitionInput(s.Input))
	if err != nil {
		return [32]byte{}, err
	}
	current, err := indexedHDInputIdentity(j.input)
	if err != nil {
		return [32]byte{}, err
	}
	raw, err := json.Marshal([]any{frozen, current, s.MapIndex, s.Palette, s.LUTs, s.Passes, s.Backgrounds,
		j.spec, j.schedule, j.baseline, j.frameWork, j.frameVGA, j.work})
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(raw), nil
}

func (g *Game) freezeModernHDIndexedTransition(j *nativeIndexedTransitionJob) error {
	in, err := g.unitPresentHDInput(j.input)
	if err != nil {
		return err
	}
	palette, err := fdother.VGAPaletteFromDAC(j.baseline)
	if err != nil {
		return err
	}
	s := &modernHDIndexedTransitionSource{Input: in, MapIndex: g.nativeMapAssets.MapIndex, Palette: palette}
	ground := nativeTransitionInput(in)
	ground.Units, ground.ForegroundUnits = nil, nil
	for i, frame := range j.schedule.Frames {
		lut, err := fdother.NativeIndexedTransitionLUT(g.nativeMapAssets.LUTs, frame.LUTIndex)
		if err != nil {
			return err
		}
		pass, err := fdother.BuildNativeIndexedTransitionPass(j.spec.TileX, j.spec.TileY, frame.Radius, j.spec.StartY, j.spec.EndY)
		if err != nil {
			return err
		}
		work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
		if i > 0 {
			copy(work, j.frameWork[i-1])
		}
		vga := make([]byte, indexedmap.NativeMapVGASize)
		if err := indexedmap.ComposeNativeTransitionFrame(work, vga, ground, pass, lut); err != nil {
			return err
		}
		s.Passes = append(s.Passes, pass)
		s.LUTs = append(s.LUTs, append([]byte(nil), lut...))
		s.Backgrounds = append(s.Backgrounds, vga)
	}
	j.hdSource = s
	j.hdSignature, err = j.modernHDSignature()
	return err
}

func (g *Game) validateModernHDIndexedTransition(j *nativeIndexedTransitionJob) error {
	if j == nil || j.hdSource == nil || g.nativeMapAssets == nil || j.hdSource.MapIndex != g.nativeMapAssets.MapIndex ||
		len(j.frameWork) != 9 || len(j.frameVGA) != 9 || len(j.hdSource.Passes) != 9 || len(j.hdSource.LUTs) != 9 ||
		len(j.hdSource.Backgrounds) != 9 || len(j.schedule.Frames) != 9 || len(j.schedule.PaletteDeltas) != 32 {
		return errors.New("modern HD indexed transition: incomplete producer")
	}
	digest, err := j.modernHDSignature()
	if err != nil || digest != j.hdSignature {
		return errors.New("modern HD indexed transition: frozen source changed")
	}
	frozen := nativeTransitionInput(j.hdSource.Input)
	if frozen.TerrainBank != j.input.TerrainBank || frozen.UnitBank != j.input.UnitBank || frozen.ForegroundBank != j.input.ForegroundBank {
		return errors.New("modern HD indexed transition: immutable bank source changed")
	}
	work, vga := make([]byte, indexedmap.NativeUnitPresentWorkSize), make([]byte, indexedmap.NativeMapVGASize)
	for i, pass := range j.hdSource.Passes {
		if err := indexedmap.ComposeNativeTransitionFrame(work, vga, frozen, pass, j.hdSource.LUTs[i]); err != nil {
			return err
		}
		if !bytes.Equal(work, j.frameWork[i]) || !bytes.Equal(vga, j.frameVGA[i]) {
			return fmt.Errorf("modern HD indexed transition: native pass %d changed", i)
		}
	}
	if !bytes.Equal(work, j.work) {
		return errors.New("modern HD indexed transition: final work changed")
	}
	return nil
}

func (g *Game) prepareModernHDIndexedTransition(j *nativeIndexedTransitionJob, set *modernStoryPortraitSet) ([]*modernHDMapFrame, error) {
	if set == nil || len(set.mapSpritesHD) == 0 {
		return nil, nil
	}
	if len(set.mapSpritesHD) != 125 {
		return nil, errors.New("modern HD indexed transition: partial HD world")
	}
	if err := g.validateModernHDIndexedTransition(j); err != nil {
		return nil, err
	}
	s := j.hdSource
	counts := make(map[int]int, len(set.mapSpritesHD))
	for group, images := range set.mapSpritesHD {
		counts[group] = len(images)
	}
	units, err := buildModernHDMapUnitPlan(s.Input.Frame, unitPresentHDBase, counts)
	if err != nil {
		return nil, err
	}
	for _, unit := range units {
		picture := set.mapSpritesHD[unit.Group][unit.Frame]
		if picture == nil || picture.Bounds() != image.Rect(0, 0, 144, 144) {
			return nil, errors.New("modern HD indexed transition: selected HD unit absent or wrong size")
		}
	}
	if j.hdCachedTheme == set && len(j.hdCached) == 41 {
		return j.hdCached, nil
	}
	normal, err := g.prepareModernHDCommandSteady(s.Input, make([]byte, indexedmap.NativeUnitPresentWorkSize), j.frameVGA[0], s.Palette, set)
	if err != nil {
		return nil, err
	}
	frames := make([]*modernHDMapFrame, 0, 41)
	weights := make(map[uint32]modernTerrainColourWeights)
	for i, pass := range s.Passes {
		f, err := g.prepareModernHDRadialMap(modernHDRadialMapSpec{Input: s.Input, Palette: s.Palette,
			Pass: pass, LUT: s.LUTs[i], Background: s.Backgrounds[i]}, set, normal, weights)
		if err != nil {
			return nil, err
		}
		frames = append(frames, unitPresentHDBorder(f, j.frameVGA[i], s.Palette))
	}
	for _, delta := range j.schedule.PaletteDeltas {
		frame := *frames[8]
		frame.PaletteAdd6 = delta
		frames = append(frames, &frame)
	}
	return frames, nil
}

func (g *Game) stageModernHDIndexedTransition(set *modernStoryPortraitSet) (func(), error) {
	j := g.indexedTransition
	if j == nil {
		return func() {}, nil
	}
	frames, err := g.prepareModernHDIndexedTransition(j, set)
	if err != nil {
		return nil, err
	}
	return func() {
		j.hdFrames, j.hdTheme = frames, set
		if frames != nil {
			j.hdCached, j.hdCachedTheme = frames, set
		}
	}, nil
}
