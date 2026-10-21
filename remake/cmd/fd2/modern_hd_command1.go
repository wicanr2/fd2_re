package main

import (
	"fmt"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// Binds the already composed command1 sources, without running rules or RNG.
type modernHDCommand1Frames struct {
	opening       *modernHDCommand0Frames
	targets, tail []nativeCompoundPresentedFrame
}

func bindModernHDCommand1Sources(sources []battlepresent.NativeCommand1FrameSources,
	actor *figani.Animation, actorID int, targets []*figani.Animation, targetIDs []int,
	effect *figani.Animation, effectID int, palette color.Palette, phase string) ([]nativeCompoundPresentedFrame, error) {
	if actor == nil || len(actor.Frames) == 0 || len(targets) == 0 || len(targets) != len(targetIDs) || effect == nil || len(palette) != 256 {
		return nil, fmt.Errorf("modern HD command1: source inputs unavailable")
	}
	out := make([]nativeCompoundPresentedFrame, 0, len(sources))
	for _, s := range sources {
		targetIndex := 0
		if s.NextTarget {
			targetIndex = 1
		}
		if len(s.Base) != 64000 || s.ActorFrame < -1 || s.ActorFrame >= len(actor.Frames) || targetIndex >= len(targets) || targets[targetIndex] == nil ||
			s.TargetFrame < 0 || s.TargetFrame >= len(targets[targetIndex].Frames) {
			return nil, fmt.Errorf("modern HD command1: source index differs")
		}
		var layers []modernHDBattleSource
		if s.ActorFrame >= 0 {
			layers = append(layers, modernHDCommandFigure(actor, actorID, s.ActorFrame))
		}
		appendEffects := func(effects []battlepresent.NativeCommand1EffectLayer) error {
			for _, e := range effects {
				if e.Frame < 0 || e.Frame >= len(effect.Frames) {
					return fmt.Errorf("modern HD command1: effect index differs")
				}
				layers = append(layers, modernHDBattleSource{Archive: "FDOTHER.DAT", Resource: effectID,
					Index: e.Frame, Frame: effect.Frames[e.Frame], DX: e.X, DY: e.Y, Fill: -1})
			}
			return nil
		}
		if err := appendEffects(s.BeforeTarget); err != nil {
			return nil, err
		}
		target := modernHDCommandFigure(targets[targetIndex], targetIDs[targetIndex], s.TargetFrame)
		target.DX = s.TargetOffsetX
		layers = append(layers, target)
		if err := appendEffects(s.AfterTarget); err != nil {
			return nil, err
		}
		out = append(out, modernHDCommandFrame(s.Base, layers, palette, phase))
	}
	return out, nil
}

func (f *modernHDCommand1Frames) resources() []int {
	return modernCompoundFigureResources(f.sourceFrames())
}

func (f *modernHDCommand1Frames) sourceFrames() []nativeCompoundPresentedFrame {
	if f == nil || f.opening == nil {
		return nil
	}
	frames := append([]nativeCompoundPresentedFrame(nil), f.opening.prelude...)
	frames = append(frames, f.opening.actor...)
	frames = append(frames, f.opening.pulse...)
	frames = append(frames, f.targets...)
	frames = append(frames, f.tail...)
	return frames
}

func (j *nativeCommand1PresentationJob) modernFrame() *nativeCompoundPresentedFrame {
	if j == nil || j.hd == nil || j.hd.opening == nil {
		return nil
	}
	var frames []nativeCompoundPresentedFrame
	switch j.phase {
	case nativeCommand1Prelude:
		frames = j.hd.opening.prelude
	case nativeCommand1Actor:
		frames = j.hd.opening.actor
		if j.frame >= 0 && j.frame < len(j.actorSpecs) && j.actorSpecs[j.frame].Pulse && !j.pulseBlack {
			frames = j.hd.opening.pulse
		}
	case nativeCommand1Targets:
		frames = j.hd.targets
	case nativeCommand1Tail:
		frames = j.hd.tail
	}
	if j.frame < 0 || j.frame >= len(frames) {
		return nil
	}
	return &frames[j.frame]
}
