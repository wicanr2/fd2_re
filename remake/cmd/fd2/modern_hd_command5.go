package main

import (
	"fmt"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// Only binds the source inputs consumed by the existing command4/5 producer.
// Publication, RNG, sounds and Draw acknowledgments remain with its owner.
type modernHDCommand5Frames struct {
	opening       *modernHDCommand0Frames
	handler, tail []nativeCompoundPresentedFrame
}

func bindModernHDCommand5Handler(frames []battlepresent.NativeCommand5RenderedFrame, actor *figani.Animation,
	actorID int, targets []*figani.Animation, targetIDs []int, effect *figani.Animation, effectID int,
	palette color.Palette) ([]nativeCompoundPresentedFrame, error) {
	if actor == nil || len(actor.Frames) == 0 || len(targets) == 0 || len(targetIDs) != len(targets) || effect == nil || len(palette) != 256 {
		return nil, fmt.Errorf("modern HD command5: source inputs unavailable")
	}
	out := make([]nativeCompoundPresentedFrame, 0, len(frames))
	for _, rendered := range frames {
		s := rendered.Sources
		if len(s.Base) != 64000 || s.ActorFrame < -1 || s.ActorFrame >= len(actor.Frames) ||
			s.TargetIndex < 0 || s.TargetIndex >= len(targets) || targets[s.TargetIndex] == nil ||
			s.TargetFrame < 0 || s.TargetFrame >= len(targets[s.TargetIndex].Frames) || s.Phase == "" {
			return nil, fmt.Errorf("modern HD command5: source index differs")
		}
		var layers []modernHDBattleSource
		if s.ActorFrame >= 0 {
			layers = append(layers, modernHDCommandFigure(actor, actorID, s.ActorFrame))
		}
		target := modernHDCommandFigure(targets[s.TargetIndex], targetIDs[s.TargetIndex], s.TargetFrame)
		target.DX = s.TargetOffsetX
		layers = append(layers, target)
		for _, layer := range s.Layers {
			if layer.Frame < 0 || layer.Frame >= len(effect.Frames) {
				return nil, fmt.Errorf("modern HD command5: effect index differs")
			}
			layers = append(layers, modernHDBattleSource{Archive: "FDOTHER.DAT", Resource: effectID,
				Index: layer.Frame, Frame: effect.Frames[layer.Frame], DX: layer.X, Fill: -1})
		}
		out = append(out, modernHDCommandFrame(s.Base, layers, palette, s.Phase))
	}
	return out, nil
}

func (f *modernHDCommand5Frames) resources() []int {
	if f == nil || f.opening == nil {
		return nil
	}
	frames := append([]nativeCompoundPresentedFrame(nil), f.opening.prelude...)
	frames = append(frames, f.opening.actor...)
	frames = append(frames, f.handler...)
	frames = append(frames, f.tail...)
	return modernCompoundFigureResources(frames)
}

func (j *nativeCommand5PresentationJob) modernFrame() *nativeCompoundPresentedFrame {
	if j == nil || j.hd == nil || j.hd.opening == nil {
		return nil
	}
	var frames []nativeCompoundPresentedFrame
	switch j.phase {
	case nativeCommand5CommonPrelude:
		frames = j.hd.opening.prelude
	case nativeCommand5Actor:
		frames = j.hd.opening.actor
		if j.frame >= 0 && j.frame < len(j.actorSpecs) && j.actorSpecs[j.frame].Pulse && !j.pulseBlack {
			frames = j.hd.opening.pulse
		}
	case nativeCommand5Handler:
		frames = j.hd.handler
	case nativeCommand5CommonTail:
		frames = j.hd.tail
	}
	if j.frame < 0 || j.frame >= len(frames) {
		return nil
	}
	return &frames[j.frame]
}
