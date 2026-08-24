package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type NativeCompoundCommonFrames struct {
	ActorSlide  [][]byte
	EffectSlide [][]byte
	Main        [][]byte
	Tail        [][]byte
}

// BuildNativeCompoundActorFrames preserves the no-MP-marker class19 branch of
// sub_2B659. Each FIGANI delay tick becomes one frame and target idle timing
// advances across the complete actor phase.
func BuildNativeCompoundActorFrames(base []byte, actorEffect, firstTargetIdle *figani.Animation, rawSide byte) ([][]byte, error) {
	const frameBytes = 320 * 200
	if len(base) != frameBytes || actorEffect == nil || len(actorEffect.Frames) == 0 ||
		firstTargetIdle == nil || len(firstTargetIdle.Frames) == 0 || actorEffect.HeaderByte4 == 1 {
		return nil, errors.New("battlepresent: native compound actor contract unavailable")
	}
	for index, frame := range firstTargetIdle.Frames {
		if frame.Delay <= 0 || frame.Width <= 0 || frame.Height <= 0 || len(frame.Pixels) != frame.Width*frame.Height || len(frame.Mask) != len(frame.Pixels) {
			return nil, fmt.Errorf("battlepresent: compound target idle frame %d malformed", index)
		}
	}
	targetFrame, targetRepeat := 0, 0
	frames := make([][]byte, 0)
	for index, actor := range actorEffect.Frames {
		if actor.Delay <= 0 || actor.Width <= 0 || actor.Height <= 0 || len(actor.Pixels) != actor.Width*actor.Height || len(actor.Mask) != len(actor.Pixels) {
			return nil, fmt.Errorf("battlepresent: compound actor frame %d malformed", index)
		}
		for repeat := 0; repeat < actor.Delay; repeat++ {
			pixels := append([]byte(nil), base...)
			drawActor := func() error { return actor.BlitAt(pixels, 320) }
			drawTarget := func() error { return firstTargetIdle.Frames[targetFrame].BlitAt(pixels, 320) }
			if rawSide != 0 {
				if err := drawActor(); err != nil {
					return nil, err
				}
				if err := drawTarget(); err != nil {
					return nil, err
				}
			} else {
				if err := drawTarget(); err != nil {
					return nil, err
				}
				if err := drawActor(); err != nil {
					return nil, err
				}
			}
			frames = append(frames, pixels)
			targetRepeat++
			if targetRepeat >= firstTargetIdle.Frames[targetFrame].Delay {
				targetRepeat = 0
				targetFrame = (targetFrame + 1) % len(firstTargetIdle.Frames)
			}
		}
	}
	if len(frames) == 0 {
		return nil, errors.New("battlepresent: native compound actor has no present boundary")
	}
	return frames, nil
}

// BuildNativeCompoundCommonFrames reproduces the framebuffer-only portion of
// 0x27FC9 after 0x29164/0x2B659 and before the command-specific state tail.
// Audio markers and palette deltas stay in the typed schedule and are owned by
// the Draw-acknowledged Game transaction.
func BuildNativeCompoundCommonFrames(base []byte, actor figani.Frame, effect *figani.Animation, schedule figani.NativeCompoundPresentationSchedule) (NativeCompoundCommonFrames, error) {
	const frameBytes = 320 * 200
	if len(base) != frameBytes || effect == nil || len(effect.Frames) < 2 ||
		schedule.CommandID < 32 || schedule.CommandID > 35 ||
		schedule.EffectResource != schedule.CommandID+33 ||
		schedule.ActorSlideFrames != 8 || schedule.ActorSlideStepX != 20 ||
		schedule.EffectSlideFrames != 9 || schedule.EffectSlideStepX != 30 ||
		schedule.MainFrameCount != len(effect.Frames)-1 {
		return NativeCompoundCommonFrames{}, errors.New("battlepresent: native compound common contract unavailable")
	}
	out := NativeCompoundCommonFrames{
		ActorSlide:  make([][]byte, 0, schedule.ActorSlideFrames),
		EffectSlide: make([][]byte, 0, schedule.EffectSlideFrames),
		Main:        make([][]byte, 0, schedule.MainFrameCount),
	}
	for i := 0; i < schedule.ActorSlideFrames; i++ {
		pixels := append([]byte(nil), base...)
		if err := actor.BlitTranslated(pixels, 320, schedule.ActorSlideStepX*i, 0, nil); err != nil {
			return NativeCompoundCommonFrames{}, fmt.Errorf("battlepresent: compound actor slide %d: %w", i, err)
		}
		out.ActorSlide = append(out.ActorSlide, pixels)
	}
	for j := schedule.EffectSlideFrames - 1; j >= 0; j-- {
		pixels := append([]byte(nil), base...)
		if err := effect.Frames[0].BlitTranslated(pixels, 320, schedule.EffectSlideStepX*j, 0, nil); err != nil {
			return NativeCompoundCommonFrames{}, fmt.Errorf("battlepresent: compound effect slide %d: %w", j, err)
		}
		out.EffectSlide = append(out.EffectSlide, pixels)
	}
	mainBase := append([]byte(nil), base...)
	if schedule.OverlayFirstFrame {
		if err := effect.Frames[0].BlitAt(mainBase, 320); err != nil {
			return NativeCompoundCommonFrames{}, fmt.Errorf("battlepresent: compound steady overlay: %w", err)
		}
	}
	for frame := 1; frame < len(effect.Frames); frame++ {
		pixels := append([]byte(nil), mainBase...)
		if err := effect.Frames[frame].BlitAt(pixels, 320); err != nil {
			return NativeCompoundCommonFrames{}, fmt.Errorf("battlepresent: compound main frame %d: %w", frame, err)
		}
		out.Main = append(out.Main, pixels)
	}
	if schedule.TailEnabled {
		if schedule.TailFrames != 11 {
			return NativeCompoundCommonFrames{}, errors.New("battlepresent: native compound tail count unavailable")
		}
		out.Tail = make([][]byte, 0, schedule.TailFrames)
		for frame := 0; frame < schedule.TailFrames; frame++ {
			pixels := append([]byte(nil), base...)
			index := len(effect.Frames) - 2 + frame%2
			if err := effect.Frames[index].BlitAt(pixels, 320); err != nil {
				return NativeCompoundCommonFrames{}, fmt.Errorf("battlepresent: compound tail frame %d: %w", frame, err)
			}
			out.Tail = append(out.Tail, pixels)
		}
	}
	return out, nil
}
