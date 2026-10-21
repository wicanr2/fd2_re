package main

import (
	"fmt"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// 只消費既有producer的實際來源，不擁有戰鬥數值或時序。
type modernHDOrdinaryFrames struct {
	opening       *modernHDCommand0Frames
	handler, tail []nativeCompoundPresentedFrame
}

func bindModernHDOrdinarySources(sources []battlepresent.NativeCommandSources, actor *figani.Animation, actorID int,
	targets []*figani.Animation, targetIDs []int, effect *figani.Animation, effectID int, palette color.Palette) ([]nativeCompoundPresentedFrame, error) {
	if actor == nil || len(actor.Frames) == 0 || len(targets) == 0 || len(targets) != len(targetIDs) || effect == nil || len(palette) != 256 {
		return nil, fmt.Errorf("modern HD ordinary: source inputs unavailable")
	}
	out := make([]nativeCompoundPresentedFrame, 0, len(sources))
	for _, s := range sources {
		if len(s.Base) != 64000 || s.Phase == "" || s.ActorFrame < -1 || s.ActorFrame >= len(actor.Frames) ||
			s.TargetIndex < 0 || s.TargetIndex >= len(targets) || targets[s.TargetIndex] == nil ||
			s.TargetFrame < 0 || s.TargetFrame >= len(targets[s.TargetIndex].Frames) {
			return nil, fmt.Errorf("modern HD ordinary: source index differs")
		}
		var layers []modernHDBattleSource
		appendEffects := func(effects []battlepresent.NativeCommandEffectSource) error {
			for _, e := range effects {
				if e.Frame < 0 || e.Frame >= len(effect.Frames) {
					return fmt.Errorf("modern HD ordinary: effect index differs")
				}
				layers = append(layers, modernHDBattleSource{Archive: "FDOTHER.DAT", Resource: effectID, Index: e.Frame,
					Frame: effect.Frames[e.Frame], DX: e.X, DY: e.Y, Fill: -1})
			}
			return nil
		}
		if err := appendEffects(s.Before); err != nil {
			return nil, err
		}
		if s.ActorFrame >= 0 {
			layers = append(layers, modernHDCommandFigure(actor, actorID, s.ActorFrame))
		}
		target := modernHDCommandFigure(targets[s.TargetIndex], targetIDs[s.TargetIndex], s.TargetFrame)
		target.DX, target.DY = s.TargetOffsetX, s.TargetOffsetY
		layers = append(layers, target)
		if err := appendEffects(s.After); err != nil {
			return nil, err
		}
		out = append(out, modernHDCommandFrame(s.Base, layers, palette, s.Phase))
	}
	return out, nil
}

func (f *modernHDOrdinaryFrames) resources() []int {
	if f == nil || f.opening == nil {
		return nil
	}
	frames := append([]nativeCompoundPresentedFrame(nil), f.opening.prelude...)
	frames = append(frames, f.opening.actor...)
	frames = append(frames, f.handler...)
	frames = append(frames, f.tail...)
	return modernCompoundFigureResources(frames)
}

func (j *nativeCommand3PresentationJob) modernFrame() *nativeCompoundPresentedFrame {
	if j == nil || j.hd == nil || j.hd.opening == nil {
		return nil
	}
	var frames []nativeCompoundPresentedFrame
	switch j.phase {
	case nativeCommand3CommonPrelude:
		frames = j.hd.opening.prelude
	case nativeCommand3Actor:
		frames = j.hd.opening.actor
		if j.frame >= 0 && j.frame < len(j.actorSpecs) && j.actorSpecs[j.frame].Pulse && !j.pulseBlack {
			frames = j.hd.opening.pulse
		}
	case nativeCommand3Handler:
		frames = j.hd.handler
	case nativeCommand3CommonTail:
		frames = j.hd.tail
	}
	if j.frame < 0 || j.frame >= len(frames) {
		return nil
	}
	return &frames[j.frame]
}

func (j *nativeCommand7PresentationJob) modernFrame() *nativeCompoundPresentedFrame {
	if j == nil || j.hd == nil || j.hd.opening == nil {
		return nil
	}
	var frames []nativeCompoundPresentedFrame
	switch j.phase {
	case nativeCommand7CommonPrelude:
		frames = j.hd.opening.prelude
	case nativeCommand7Actor:
		frames = j.hd.opening.actor
		if j.frame >= 0 && j.frame < len(j.actorSpecs) && j.actorSpecs[j.frame].Pulse && !j.pulseBlack {
			frames = j.hd.opening.pulse
		}
	case nativeCommand7Handler:
		frames = j.hd.handler
	case nativeCommand7CommonTail:
		frames = j.hd.tail
	}
	if j.frame < 0 || j.frame >= len(frames) {
		return nil
	}
	return &frames[j.frame]
}

func (j *nativeCommand8PresentationJob) modernFrame() *nativeCompoundPresentedFrame {
	if j == nil || j.hd == nil || j.hd.opening == nil {
		return nil
	}
	var frames []nativeCompoundPresentedFrame
	switch j.phase {
	case nativeCommand8CommonPrelude:
		frames = j.hd.opening.prelude
	case nativeCommand8Actor:
		frames = j.hd.opening.actor
		if j.frame >= 0 && j.frame < len(j.actorSpecs) && j.actorSpecs[j.frame].Pulse && !j.pulseBlack {
			frames = j.hd.opening.pulse
		}
	case nativeCommand8Handler:
		frames = j.hd.handler
	case nativeCommand8CommonTail:
		frames = j.hd.tail
	}
	if j.frame < 0 || j.frame >= len(frames) {
		return nil
	}
	return &frames[j.frame]
}

func (j *nativeCommand2PresentationJob) modernFrame() *nativeCompoundPresentedFrame {
	if j == nil || j.hd == nil || j.hd.opening == nil {
		return nil
	}
	var frames []nativeCompoundPresentedFrame
	switch j.phase {
	case nativeCommand2Prelude:
		frames = j.hd.opening.prelude
	case nativeCommand2Actor:
		frames = j.hd.opening.actor
		if j.frame >= 0 && j.frame < len(j.actorSpecs) && j.actorSpecs[j.frame].Pulse && !j.pulseBlack {
			frames = j.hd.opening.pulse
		}
	case nativeCommand2Targets:
		frames = j.hd.handler
	case nativeCommand2Tail:
		frames = j.hd.tail
	}
	if j.frame < 0 || j.frame >= len(frames) {
		return nil
	}
	return &frames[j.frame]
}
