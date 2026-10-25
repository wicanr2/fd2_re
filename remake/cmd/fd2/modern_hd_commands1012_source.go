package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func commandStageIdentity(s *nativeCh23AdapterState) any {
	if s == nil {
		return nil
	}
	return []any{s.staging, s.rowShift, s.latch, s.tickSnapshot, s.initialComplete}
}

func command1012SourceDigest(j *nativeCommand1012Job) ([32]byte, error) {
	if j == nil || j.hdScene == nil {
		return [32]byte{}, errors.New("modern HD command10-12: scene absent")
	}
	in, err := indexedHDInputIdentity(nativeTransitionInput(j.hdInput))
	if err != nil {
		return [32]byte{}, err
	}
	raw, err := json.Marshal([]any{j.hdScene.signature, in, j.schedule, j.baselineWork, j.baselineVGA, j.palette, j.mainFrames, j.resultFrames, j.resultSources})
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(raw), nil
}

func (g *Game) validateModernHDCommand1012Source(j *nativeCommand1012Job) error {
	if j == nil || j.hdScene == nil {
		return errors.New("modern HD command10-12: scene absent")
	}
	if err := j.hdScene.validate(g); err != nil {
		return err
	}
	probe := *j.hdScene
	probe.input, probe.work, probe.vga = j.hdBaseline, j.baselineWork, j.baselineVGA
	digest, err := probe.digest()
	if err != nil || digest != j.hdScene.signature {
		return errors.New("modern HD command10-12: baseline producer changed")
	}
	a, b := j.hdInput.Frame, j.hdScene.input.Frame
	if a.TerrainBank != b.TerrainBank || a.UnitBank != b.UnitBank || a.ForegroundBank != b.ForegroundBank {
		return errors.New("modern HD command10-12: bank source changed")
	}
	digest, err = command1012SourceDigest(j)
	if err != nil || digest != j.hdSignature {
		return errors.New("modern HD command10-12: frozen source changed")
	}
	if len(j.mainFrames) != 60 {
		return errors.New("modern HD command10-12: native frames absent")
	}
	for i := 0; i < 3; i++ {
		work, vga := append([]byte(nil), j.baselineWork...), append([]byte(nil), j.baselineVGA...)
		if err := indexedmap.ComposeNativeCommand10To12Surface(work, vga, nativeTransitionInput(j.hdInput), j.schedule.XOrigins[i], j.schedule.YOrigins[i], j.schedule.SamplingIncrements[i]); err != nil {
			return err
		}
		if !bytes.Equal(vga, j.mainFrames[i+1]) {
			return errors.New("modern HD command10-12: sampled producer changed")
		}
	}
	return nil
}

func commandPreludeSourceDigest(j *native2189AJob) ([32]byte, error) {
	if j == nil || j.hdScene == nil || j.hdSteady == nil {
		return [32]byte{}, errors.New("modern HD prelude: scene absent")
	}
	in, err := indexedHDInputIdentity(nativeTransitionInput(j.hdInput))
	if err != nil {
		return [32]byte{}, err
	}
	raw, err := json.Marshal([]any{j.hdScene.signature, j.hdSteady.signature, in, commandStageIdentity(j.mapStage), j.palette, j.hdTerrain, j.hdRadial, j.hdLUTs, j.workFrames, j.vgaFrames})
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(raw), nil
}

func (g *Game) validateModernHDCommandPreludeSource(j *native2189AJob) error {
	if j == nil || j.hdScene == nil || j.hdSteady == nil {
		return errors.New("modern HD prelude: scene absent")
	}
	if err := j.hdScene.validate(g); err != nil {
		return err
	}
	if err := j.hdSteady.validate(g); err != nil {
		return err
	}
	digest, err := commandPreludeSourceDigest(j)
	if err != nil || digest != j.hdSignature {
		return errors.New("modern HD prelude: frozen source changed")
	}
	if len(j.workFrames) != 11 {
		return errors.New("modern HD prelude: work frames absent")
	}
	a, b := j.hdInput.Frame, j.hdScene.input.Frame
	if a.TerrainBank != b.TerrainBank || a.UnitBank != b.UnitBank || a.ForegroundBank != b.ForegroundBank {
		return errors.New("modern HD prelude: bank source changed")
	}
	work := append([]byte(nil), j.hdScene.work...)
	if j.mapStage != nil {
		if err := indexedmap.SeedNativeCh23Staging(work, j.mapStage.staging); err != nil {
			return err
		}
	}
	input := nativeTransitionInput(j.hdInput)
	if err := indexedmap.ComposeNativeUnitPresentTerrainSnapshot(work, input); err != nil {
		return err
	}
	if !bytes.Equal(work, j.hdTerrain) {
		return errors.New("modern HD prelude: terrain snapshot changed")
	}
	for i, spec := range j.hdRadial {
		frame, vga := append([]byte(nil), work...), append([]byte(nil), j.hdScene.vga...)
		if err := fdother.ApplyRadialLUTRemap(frame[0x8088:], 456, j.hdLUTs[i], spec); err != nil {
			return err
		}
		if err := indexedmap.RedrawNativeUnitPresentObjects(frame, input); err != nil {
			return err
		}
		if err := indexedmap.CopyNativeUnitPresentViewport(vga, frame); err != nil {
			return err
		}
		if !bytes.Equal(frame, j.workFrames[i]) || !bytes.Equal(vga, j.vgaFrames[i]) {
			return errors.New("modern HD prelude: native pass changed")
		}
	}
	work = append([]byte(nil), j.hdScene.work...)
	vga := append([]byte(nil), j.hdScene.vga...)
	if j.mapStage != nil {
		if err := indexedmap.SeedNativeCh23Staging(work, j.mapStage.staging); err != nil {
			return err
		}
	}
	if err := indexedmap.ComposeNativeFrame(work, vga, j.hdSteady.input); err != nil {
		return err
	}
	if !bytes.Equal(work, j.workFrames[10]) || !bytes.Equal(vga, j.vgaFrames[10]) {
		return errors.New("modern HD prelude: final redraw changed")
	}
	return nil
}
