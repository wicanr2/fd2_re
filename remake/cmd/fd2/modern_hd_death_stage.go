package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// This source belongs to the native 1DB65 producer, including pixel themes.
// It is separate from a compound scene: the first 25 copies start at VGA zero
// and A accumulates LMI writes over the final visible pose.
type modernHDDeathStageSource struct {
	stage                     *nativeCh23AdapterState
	baselineWork, baselineVGA []byte
	signature                 [32]byte
	sealed                    bool
}

func (s *modernHDDeathStageSource) seed(work []byte) error {
	if s == nil {
		return nil
	}
	if s.stage == nil {
		return errors.New("native death: same-compose stage unavailable")
	}
	return indexedmap.SeedNativeCh23Staging(work, s.stage.staging)
}

func modernHDDeathInput(in indexedmap.NativeFrameInput, index int) indexedmap.NativeFrameInput {
	if index < 25 {
		in.Frame.RangeMode, in.Frame.SelectionOverlay, in.Frame.ChapterAux = 0, nil, nil
	}
	return in
}

func (j *nativeCh28PostPresentJob) deathStageDigest() ([32]byte, error) {
	s := j.hdStage
	if s == nil || s.stage == nil || len(j.hdInputs) != 26 || len(j.workFrames) != 26 || len(j.vgaFrames) != 26 {
		return [32]byte{}, errors.New("modern HD death: incomplete frozen stage")
	}
	frames := make([][32]byte, 26)
	for i, in := range j.hdInputs {
		frame := &modernHDCompoundMapScene{input: modernHDDeathInput(in, i), work: j.workFrames[i], vga: j.vgaFrames[i], palette: j.palette, mapIndex: 23, stage: s.stage}
		var err error
		frames[i], err = frame.digest()
		if err != nil {
			return [32]byte{}, err
		}
	}
	raw, err := json.Marshal([]any{s.baselineWork, s.baselineVGA, frames, j.hdTargets, j.waits, j.sfxAt})
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(raw), nil
}

func (g *Game) validateModernHDDeathStageSource(j *nativeCh28PostPresentJob) error {
	if g.nativeMapAssets == nil {
		return errors.New("modern HD death: native assets unavailable")
	}
	if g.nativeMapAssets.MapIndex != 23 {
		if j.hdStage != nil {
			return errors.New("modern HD death: stage attached to another map")
		}
		return nil
	}
	if j.hdStage == nil {
		return errors.New("modern HD death: frozen stage absent")
	}
	s := j.hdStage
	digest, err := j.deathStageDigest()
	if err != nil || !s.sealed || digest != s.signature {
		return errors.New("modern HD death: frozen source changed")
	}
	if len(s.baselineWork) != indexedmap.NativeUnitPresentWorkSize || len(s.baselineVGA) != 64000 || len(j.palette) != 256 {
		return errors.New("modern HD death: baseline geometry changed")
	}
	for _, in := range j.hdInputs {
		if in.Frame.TerrainBank != g.nativeMapAssets.Terrain || in.Frame.UnitBank != g.nativeMapAssets.Units || in.Frame.ForegroundBank != g.nativeMapAssets.Terrain || in.Frame.RangeBank != g.nativeMapAssets.Range {
			return errors.New("modern HD death: immutable bank changed")
		}
	}
	work, vga := append([]byte(nil), s.baselineWork...), append([]byte(nil), s.baselineVGA...)
	check := func(index int) error {
		if !bytes.Equal(work, j.workFrames[index]) || !bytes.Equal(vga, j.vgaFrames[index]) {
			return fmt.Errorf("modern HD death: native frame%d reconstruction differs", index)
		}
		return nil
	}
	for i := 0; i < 13; i++ {
		in := nativeTransitionInput(j.hdInputs[i])
		if err := s.seed(work); err != nil {
			return err
		}
		if err := indexedmap.ComposeNativeUnitPresentTerrainSnapshot(work, in); err != nil {
			return err
		}
		if err := indexedmap.RedrawNativeUnitPresentObjects(work, in); err != nil {
			return err
		}
		if err := indexedmap.CopyNativeUnitPresentViewport(vga, work); err != nil {
			return err
		}
		if err := check(i); err != nil {
			return err
		}
	}
	staged := make([]byte, len(work))
	in := nativeTransitionInput(j.hdInputs[19])
	if err := s.seed(staged); err != nil {
		return err
	}
	if err := indexedmap.ComposeNativeUnitPresentTerrainSnapshot(staged, in); err != nil {
		return err
	}
	if err := indexedmap.RedrawNativeUnitPresentObjects(staged, in); err != nil {
		return err
	}
	for i, frame := range fdother.NativeCh28PostOverlayFrames() {
		if frame.Phase == "overlay_b" {
			copy(work, staged)
		}
		for _, target := range j.hdTargets {
			if err := fdother.BlitNativeCh28PostOverlay(g.nativeMapAssets.CommandHealDigits[frame.Entry], work, target.Origin); err != nil {
				return err
			}
		}
		if err := indexedmap.CopyNativeUnitPresentViewport(vga, work); err != nil {
			return err
		}
		if err := check(13 + i); err != nil {
			return err
		}
	}
	if err := s.seed(work); err != nil {
		return err
	}
	inSteady, err := freezeCompoundMapInput(j.hdInputs[25])
	if err != nil {
		return err
	}
	if err := indexedmap.ComposeNativeFrame(work, vga, inSteady); err != nil {
		return err
	}
	return check(25)
}
