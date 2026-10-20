package main

import (
	"fmt"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// These sources describe already validated presents. They never advance the
// native owner, publish a numeric transaction or select another random value.
type modernHDCommand0Frames struct {
	prelude, actor, pulse, target, tail []nativeCompoundPresentedFrame
}

type modernHDCommand0Input struct {
	prelude        battlepresent.NativeCommandPreludeInput
	preludeFrames  []battlepresent.NativeCommand24PreludeFrame
	actor          battlepresent.NativeCommand0ActorInput
	actorFrames    []battlepresent.NativeCommand0ActorFrame
	actorIdleID    int
	actorEffectID  int
	targetIdleID   int
	effectID       int
	effect         *figani.Animation
	schedule       figani.NativeCommand0PresentationSchedule
	bases          [][]byte
	luts           [][]byte
	palette, pulse color.Palette
}

func modernHDCommandFrame(base []byte, layers []modernHDBattleSource, palette color.Palette, phase string) nativeCompoundPresentedFrame {
	return nativeCompoundPresentedFrame{hdSources: &modernHDBattleSources{Base: base, Layers: layers}, hdPalette: palette, hdPhase: phase}
}
func modernHDCommandFigure(animation *figani.Animation, resource, index int) modernHDBattleSource {
	return modernHDBattleSource{Frame: animation.Frames[index], Resource: resource, Index: index, Fill: -1}
}
func modernHDCommandShade(base []byte, bg, tai fdother.Frame, lut []byte) ([]byte, error) {
	out := append([]byte(nil), base...)
	bg.X, bg.Y, tai.X, tai.Y = 0, 50, 164, 157
	if lut == nil {
		if err := bg.Blit(out, 320, -1); err != nil {
			return nil, err
		}
		if err := tai.Blit(out, 320, -1); err != nil {
			return nil, err
		}
	} else {
		if err := bg.BlitLUTAt(out, 320, 0, lut); err != nil {
			return nil, err
		}
		if err := tai.BlitLUTAt(out, 320, 0, lut); err != nil {
			return nil, err
		}
	}
	return out, nil
}
func bindModernHDCommandOpening(in modernHDCommand0Input) (*modernHDCommand0Frames, error) {
	if in.prelude.Mode != 0 || len(in.preludeFrames) != 9 || in.prelude.Platform == nil ||
		len(in.prelude.Base) != 64000 || len(in.actor.BaseBefore) != 64000 || len(in.actor.BaseAfter) != 64000 ||
		in.actor.ActorEffect == nil || len(in.actor.ActorEffect.Frames) == 0 ||
		in.actor.FirstTargetIdle == nil || len(in.actor.FirstTargetIdle.Frames) == 0 ||
		len(in.actorFrames) == 0 ||
		len(in.actor.LUT) != 256 || len(in.palette) != 256 || len(in.pulse) != 256 {
		return nil, fmt.Errorf("modern HD command0: sources unavailable")
	}
	frames := &modernHDCommand0Frames{}
	platform := *in.prelude.Platform
	platform.X, platform.Y = 164, 157
	platformLayer := modernHDBattleSource{Resource: -1, Fill: -1,
		Frame: figani.Frame{X: platform.X, Y: platform.Y, Width: platform.Width, Height: platform.Height, Pixels: platform.Indexed, Mask: platform.Mask}}
	preludeBase := append([]byte(nil), in.prelude.Base...)
	if in.prelude.RawSide == 0 {
		if err := platform.Blit(preludeBase, 320, -1); err != nil {
			return nil, err
		}
	}
	for index, original := range in.preludeFrames {
		if original.Stage != 8-index {
			return nil, fmt.Errorf("modern HD command0: prelude stage differs")
		}
		palette, err := fdother.ParseVGAPalette(original.DAC)
		if err != nil {
			return nil, err
		}
		a := modernHDBattleSource{Frame: in.prelude.ActorIdle, Resource: in.actorIdleID, Fill: -1, DX: -original.Stage * 10}
		b := modernHDBattleSource{Frame: in.prelude.FirstTargetIdle, Resource: in.targetIdleID, Fill: -1}
		layers := []modernHDBattleSource{a, b}
		if in.prelude.RawSide != 0 {
			p := platformLayer
			p.DX, a.DX = original.Stage*10, original.Stage*10
			layers = []modernHDBattleSource{b, p, a}
		}
		f := modernHDCommandFrame(preludeBase, layers, palette, "prelude")
		f.hdSubtract = float32(original.Stage*6) / 63
		frames.prelude = append(frames.prelude, f)
	}
	after, err := modernHDCommandShade(in.actor.BaseAfter, in.actor.Background, in.actor.Platform, in.actor.LUT)
	if err != nil {
		return nil, err
	}
	published := false
	for _, present := range in.actorFrames {
		if present.EffectFrame < 0 || present.EffectFrame >= len(in.actor.ActorEffect.Frames) ||
			present.TargetIdleFrame < 0 || present.TargetIdleFrame >= len(in.actor.FirstTargetIdle.Frames) {
			return nil, fmt.Errorf("modern HD command0: actor source index differs")
		}
		published = published || present.PublishMP
		base := in.actor.BaseBefore
		if published {
			base = after
		}
		a := modernHDCommandFigure(in.actor.ActorEffect, in.actorEffectID, present.EffectFrame)
		b := modernHDCommandFigure(in.actor.FirstTargetIdle, in.targetIdleID, present.TargetIdleFrame)
		layers := []modernHDBattleSource{a, b}
		if in.actor.RawSide == 0 {
			layers = []modernHDBattleSource{b, a}
		}
		frames.actor = append(frames.actor, modernHDCommandFrame(base, layers, in.palette, "actor"))
		frames.pulse = append(frames.pulse, modernHDCommandFrame(base, layers, in.pulse, "actor-pulse"))
	}
	return frames, nil
}

func bindModernHDCommand0(in modernHDCommand0Input) (*modernHDCommand0Frames, error) {
	if len(in.bases) != 8 || len(in.luts) <= 14 {
		return nil, fmt.Errorf("modern HD command0: target bases unavailable")
	}
	want, err := figani.BuildNativeCommand0PresentationSchedule(in.prelude.RawSide, in.effect)
	if err != nil || want != in.schedule {
		return nil, fmt.Errorf("modern HD command0: target schedule differs")
	}
	for _, base := range in.bases {
		if len(base) != 64000 {
			return nil, fmt.Errorf("modern HD command0: base unavailable")
		}
	}
	frames, err := bindModernHDCommandOpening(in)
	if err != nil {
		return nil, err
	}
	targetIndex, repeat, stage := 0, 0, 0
	advanceTarget := func() {
		repeat++
		if repeat >= in.actor.FirstTargetIdle.Frames[targetIndex].Delay {
			repeat = 0
			targetIndex = (targetIndex + 1) % len(in.actor.FirstTargetIdle.Frames)
		}
	}
	for step := 0; step < in.schedule.Frames; step++ {
		var layers []modernHDBattleSource
		appendEffects := func(flag byte) {
			for layer, which := range in.schedule.LayerFlags {
				if which != flag {
					continue
				}
				if index, ok := figani.NativeCommand0EffectFrame(step, layer); ok {
					layers = append(layers, modernHDBattleSource{Archive: "FDOTHER.DAT", Resource: in.effectID, Index: index,
						Frame: in.effect.Frames[index], Fill: -1, DX: in.schedule.SideXShift + in.schedule.XOffsets[layer], DY: in.schedule.YOffsets[layer]})
				}
			}
		}
		appendEffects(1)
		layers = append(layers, modernHDCommandFigure(in.actor.FirstTargetIdle, in.targetIdleID, targetIndex))
		appendEffects(0)
		frames.target = append(frames.target, modernHDCommandFrame(in.bases[stage], layers, in.palette, "target"))
		impact := false
		for layer := 0; layer < 7; layer++ {
			impact = impact || figani.NativeCommand0Impact(step, layer)
		}
		if impact {
			stage++
		}
		advanceTarget()
	}
	if stage != figani.NativeCommand0DamageStages {
		return nil, fmt.Errorf("modern HD command0: impact stages differ")
	}
	frames.tail, err = bindModernHDCommandTail(in.bases[7], in.actor.Background, in.actor.Platform, in.luts,
		in.actor.ActorEffect, in.actorEffectID, in.actor.FirstTargetIdle, in.targetIdleID, targetIndex, repeat, in.palette)
	if err != nil {
		return nil, err
	}
	return frames, nil
}

func bindModernHDCommandTail(base []byte, bg, tai fdother.Frame, luts [][]byte, actor *figani.Animation,
	actorID int, target *figani.Animation, targetID, targetIndex, repeat int, palette color.Palette) ([]nativeCompoundPresentedFrame, error) {
	if len(base) != 64000 || len(luts) <= 14 || actor == nil || len(actor.Frames) == 0 ||
		target == nil || targetIndex < 0 || targetIndex >= len(target.Frames) || len(palette) != 256 {
		return nil, fmt.Errorf("modern HD command tail: sources unavailable")
	}
	var frames []nativeCompoundPresentedFrame
	for pass := 0; pass < 4; pass++ {
		var lut []byte
		if pass < 3 {
			lut = luts[12+pass]
		}
		shaded, err := modernHDCommandShade(base, bg, tai, lut)
		if err != nil {
			return nil, err
		}
		layers := []modernHDBattleSource{modernHDCommandFigure(actor, actorID, 0), modernHDCommandFigure(target, targetID, targetIndex)}
		frames = append(frames, modernHDCommandFrame(shaded, layers, palette, "tail"))
		if target.Frames[targetIndex].Delay <= 0 {
			return nil, fmt.Errorf("modern HD command tail: idle delay unavailable")
		}
		repeat++
		if repeat >= target.Frames[targetIndex].Delay {
			repeat = 0
			targetIndex = (targetIndex + 1) % len(target.Frames)
		}
	}
	return frames, nil
}

func (f *modernHDCommand0Frames) resources() []int {
	if f == nil {
		return nil
	}
	frames := append([]nativeCompoundPresentedFrame(nil), f.prelude...)
	frames = append(frames, f.actor...)
	frames = append(frames, f.target...)
	frames = append(frames, f.tail...)
	return modernCompoundFigureResources(frames)
}

func (j *nativeCommand0PresentationJob) modernFrame() *nativeCompoundPresentedFrame {
	if j == nil || j.hd == nil {
		return nil
	}
	var frames []nativeCompoundPresentedFrame
	switch j.phase {
	case nativeCommand0Prelude:
		frames = j.hd.prelude
	case nativeCommand0Actor:
		frames = j.hd.actor
		if j.frame >= 0 && j.frame < len(j.actorSpecs) && j.actorSpecs[j.frame].Pulse && !j.pulseBlack {
			frames = j.hd.pulse
		}
	case nativeCommand0Target:
		frames = j.hd.target
	case nativeCommand0Tail:
		frames = j.hd.tail
	}
	if j.frame < 0 || j.frame >= len(frames) {
		return nil
	}
	return &frames[j.frame]
}
