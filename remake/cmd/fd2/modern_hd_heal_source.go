package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"image"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// The post-transaction redraw belongs to the same captured chapter stage,
// even when a later live world has moved on.
func (g *Game) freezeModernHDHealScene(in indexedmap.NativeFrameInput, work, vga []byte, palette color.Palette, stage *nativeCh23AdapterState, inputErr error) *modernHDCompoundMapScene {
	s := newModernHDCompoundMapScene(in, work, vga, palette, inputErr)
	s.mapIndex = g.nativeMapAssets.MapIndex
	if s.mapIndex == 23 {
		s.stage = cloneNativeCh23AdapterState(stage)
	}
	s.seal()
	if s.err == nil {
		s.err = s.validate(g)
	}
	return s
}

func modernHealSourceDigest(j *nativeCommandHealPresentationJob) ([32]byte, error) {
	if j == nil || j.hdBefore == nil || j.hdScene == nil {
		return [32]byte{}, errors.New("modern HD heal: frozen source absent")
	}
	in, err := indexedHDInputIdentity(nativeTransitionInput(j.hdInput))
	if err != nil {
		return [32]byte{}, err
	}
	var stage []any
	if j.mapStage != nil {
		stage = []any{j.mapStage.staging, j.mapStage.rowShift, j.mapStage.latch, j.mapStage.tickSnapshot, j.mapStage.initialComplete}
	}
	raw, err := json.Marshal([]any{in, j.hdBefore.signature, j.hdScene.signature, stage, j.hdBaseline.HUD,
		j.baselineWork, j.baselineVGA, j.palette, j.hdPasses, j.hdLUTs, j.frontFrames, j.effectFrames, j.maskFrame,
		j.digitFrames, j.effectSources, j.digitSources, j.targets, j.effectTargets})
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(raw), nil
}

func (g *Game) validateModernHDHealSource(j *nativeCommandHealPresentationJob) error {
	if j == nil || j.hdBefore == nil || j.hdScene == nil {
		return errors.New("modern HD heal: frozen source absent")
	}
	if err := j.hdBefore.validate(g); err != nil {
		return err
	}
	if err := j.hdScene.validate(g); err != nil {
		return err
	}
	probe := *j.hdScene
	probe.input, probe.work, probe.vga = j.hdBaseline, j.baselineWork, j.baselineVGA
	digest, err := probe.digest()
	if err != nil || digest != j.hdScene.signature {
		return errors.New("modern HD heal: steady producer changed")
	}
	digest, err = modernHealSourceDigest(j)
	if err != nil || digest != j.hdSignature {
		return errors.New("modern HD heal: source changed")
	}
	return nil
}

// Both production callbacks are the native player/AI heal transactions.
// They write unit scalar/array fields and the native RNG, then defer action
// cleanup to the continuation. Keep unit pointers stable on a failed redraw.
func (g *Game) nativeHealRollback() func() {
	units := make(map[*battle.Unit]battle.Unit, len(g.st.Units))
	for _, u := range g.st.Units {
		if u != nil {
			units[u] = *u
		}
	}
	rng := g.nativeRNGState
	work, vga := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	return func() {
		for u, before := range units {
			*u = before
		}
		g.nativeRNGState = rng
		g.nativeMapWork, g.nativeMapVGA = append(g.nativeMapWork[:0], work...), append(g.nativeMapVGA[:0], vga...)
	}
}

func (g *Game) prepareModernHDRadialStage(spec modernHDRadialMapSpec, set *modernStoryPortraitSet, counts []byte, luts [4][]byte) (*image.RGBA, error) {
	if spec.Stage == nil {
		return nil, errors.New("modern HD radial: whole stage source absent")
	}
	view := *g
	view.modernStoryPortraits = set
	var shades [4]*image.RGBA
	for count := range shades {
		palette := make(color.Palette, 256)
		for i := range palette {
			palette[i] = spec.Palette[luts[count][i]]
		}
		picture, err := view.prepareModernHDMapStage(spec.Input.Frame, palette, -1, spec.Stage)
		if err != nil {
			return nil, err
		}
		if picture == nil || picture.Bounds() != image.Rect(0, 0, 1872, 1152) {
			return nil, errors.New("modern HD radial: whole stage art absent")
		}
		shades[count] = picture
	}
	out := image.NewRGBA(image.Rect(0, 0, 1872, 1152))
	for y := 0; y < 1152; y++ {
		for x := 0; x < 1872; x++ {
			count := healHDCountAt(counts, x/6, y/6)
			if count > 3 {
				return nil, errors.New("modern HD radial: stage remap count outside contract")
			}
			out.SetRGBA(x, y, shades[count].RGBAAt(x, y))
		}
	}
	return out, nil
}
