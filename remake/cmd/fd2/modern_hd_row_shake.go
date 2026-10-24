package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type modernHDRowShakeSource struct {
	Input            indexedmap.NativeFrameInput
	Seed, InitialVGA []byte
	MapIndex         int
	Signature        [32]byte
	Sealed           bool
}

func (j *transitionRevealJob) modernHDRowShakeSignature() ([32]byte, error) {
	s := j.hdSource
	if s == nil {
		return [32]byte{}, errors.New("modern HD row shake: source absent")
	}
	in := s.Input
	identity, err := indexedHDInputIdentity(nativeTransitionInput(in))
	if err != nil {
		return [32]byte{}, err
	}
	raw, err := json.Marshal([]any{identity, in.HUD, in.Frame.RangeMode, in.Frame.CursorX, in.Frame.CursorY,
		in.Frame.ChapterAuxPhase, s.MapIndex, s.Seed, s.InitialVGA, j.work, j.frames, j.palette})
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(raw), nil
}

func (g *Game) validateModernHDRowShake(j *transitionRevealJob) error {
	if j == nil || j.hdSource == nil || g.nativeMapAssets == nil || j.hdSource.MapIndex != g.nativeMapAssets.MapIndex || len(j.palette) != 256 {
		return errors.New("modern HD row shake: incomplete producer")
	}
	s := j.hdSource
	if s.Input.Frame.TerrainBank != g.nativeMapAssets.Terrain || s.Input.Frame.UnitBank != g.nativeMapAssets.Units ||
		s.Input.Frame.ForegroundBank != g.nativeMapAssets.Terrain || s.Input.Frame.RangeBank != g.nativeMapAssets.Range {
		return errors.New("modern HD row shake: immutable bank changed")
	}
	digest, err := j.modernHDRowShakeSignature()
	if err != nil || s.Sealed && s.Signature != digest {
		return errors.New("modern HD row shake: frozen source changed")
	}
	in, err := freezeModernHDCommandGeometry(s.Input)
	if err != nil {
		return err
	}
	work, vga := append([]byte(nil), s.Seed...), append([]byte(nil), s.InitialVGA...)
	if err := indexedmap.ComposeNative24B4DStaging(work, nativeTransitionInput(in)); err != nil {
		return err
	}
	if err := indexedmap.ComposeNativeFrame(work, vga, in); err != nil {
		return err
	}
	if !bytes.Equal(work, j.work) {
		return errors.New("modern HD row shake: native work changed")
	}
	for row := 0; row < 2; row++ {
		pixels := append([]byte(nil), vga...)
		if err := fdother.CopyNativeTransitionViewport(pixels[0x504:], 320, work,
			fdother.NativeTransitionStageOffset+row*fdother.NativeTransitionStageStride,
			fdother.NativeTransitionStageStride, 312, 192); err != nil {
			return err
		}
		if !bytes.Equal(pixels, j.frames[row]) {
			return errors.New("modern HD row shake: native viewport changed")
		}
	}
	return nil
}

func (g *Game) prepareModernHDRowShake(j *transitionRevealJob, set *modernStoryPortraitSet) ([]*modernHDMapFrame, error) {
	if set == nil || len(set.mapSpritesHD) == 0 {
		return nil, nil
	}
	if len(set.mapSpritesHD) != 125 {
		return nil, errors.New("modern HD row shake: partial HD world")
	}
	if err := g.validateModernHDRowShake(j); err != nil {
		return nil, err
	}
	view := *g
	view.modernStoryPortraits = set
	frames := make([]*modernHDMapFrame, 2)
	for row := range frames {
		frame, err := view.prepareModernHDMapFrameLayout(j.hdSource.Input, j.hdSource.Seed, j.palette, true, row)
		if err != nil {
			return nil, err
		}
		for y := 0; y < 200; y++ {
			for x := 0; x < 320; x++ {
				if x < 4 || x >= 316 || y < 4 || y >= 196 {
					frame.Background.Set(x, y, color.RGBAModel.Convert(j.palette[j.frames[row][y*320+x]]))
				}
			}
		}
		frame.OverlaysAboveUI = true
		frames[row] = frame
	}
	return frames, nil
}

func (g *Game) stageModernHDRowShake(set *modernStoryPortraitSet) (func(), error) {
	j := g.transitionReveal
	if j == nil {
		return func() {}, nil
	}
	frames, err := g.prepareModernHDRowShake(j, set)
	if err != nil {
		return nil, err
	}
	return func() { j.hdFrames, j.hdTheme = frames, set }, nil
}
