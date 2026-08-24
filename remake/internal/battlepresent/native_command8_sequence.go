package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type NativeCommand8RenderedFrame struct {
	Pixels             []byte
	HPStage            int
	PlaySub1, PlaySub2 bool
}
type NativeCommand8EffectSequence struct {
	Front                         []NativeCommand8RenderedFrame
	Targets                       [][]NativeCommand8RenderedFrame
	Transitions                   [][]NativeCommand8RenderedFrame
	Tail                          []NativeCommand8RenderedFrame
	NextIdleFrame, NextIdleRepeat int
}
type NativeCommand8EffectInput struct {
	FrontBase, TailBase []byte
	TargetBases         [][][]byte
	TransitionBases     [][]byte
	ActorEffect         *figani.Animation
	TargetIdle          []*figani.Animation
	Effect              *figani.Animation
	Schedule            figani.NativeCommand8PresentationSchedule
	RawSide             byte
}

func BuildNativeCommand8EffectSequence(in NativeCommand8EffectInput) (NativeCommand8EffectSequence, error) {
	if len(in.FrontBase) != nativeCommand0SurfaceSize || len(in.TailBase) != nativeCommand0SurfaceSize || in.ActorEffect == nil || len(in.ActorEffect.Frames) == 0 || in.Effect == nil || len(in.TargetIdle) != 1 || len(in.TargetBases) != 1 || len(in.TransitionBases) != 0 || in.Schedule.EffectResource == 0 {
		return NativeCommand8EffectSequence{}, errors.New("battlepresent: incomplete command8 effect sequence input")
	}
	for targetIndex, idle := range in.TargetIdle {
		if idle == nil || len(idle.Frames) == 0 || len(in.TargetBases[targetIndex]) != figani.NativeCommand8DamageStages+1 {
			return NativeCommand8EffectSequence{}, fmt.Errorf("battlepresent: command8 target %d inputs unavailable", targetIndex)
		}
		for stage, base := range in.TargetBases[targetIndex] {
			if len(base) != nativeCommand0SurfaceSize {
				return NativeCommand8EffectSequence{}, fmt.Errorf("battlepresent: command8 target %d stage %d base unavailable", targetIndex, stage)
			}
		}
	}
	for i, base := range in.TransitionBases {
		if len(base) != nativeCommand0SurfaceSize {
			return NativeCommand8EffectSequence{}, fmt.Errorf("battlepresent: command8 transition %d base unavailable", i)
		}
	}

	out := NativeCommand8EffectSequence{Targets: make([][]NativeCommand8RenderedFrame, len(in.TargetIdle))}
	state := figani.NewNativeCommand8State()
	for i := 0; i < figani.NativeCommand8FrontFrames; i++ {
		frame, err := figani.PlanNativeCommand8DrawFrame(state, in.Schedule)
		if err != nil {
			return NativeCommand8EffectSequence{}, err
		}
		pixels, err := ComposeNativeCommand8OrbitFrame(in.FrontBase, in.ActorEffect, in.TargetIdle[0], in.Effect, frame)
		if err != nil {
			return NativeCommand8EffectSequence{}, fmt.Errorf("battlepresent: command8 front frame %d: %w", i, err)
		}
		out.Front = append(out.Front, NativeCommand8RenderedFrame{Pixels: pixels, PlaySub1: frame.PlaySub1, PlaySub2: frame.PlaySub2})
		state = frame.Next
	}
	lastIdleFrame, lastIdleRepeat := 0, 0
	for targetIndex, idle := range in.TargetIdle {
		planned, next, err := figani.BuildNativeCommand8TargetSequence(state, in.Schedule)
		if err != nil {
			return NativeCommand8EffectSequence{}, err
		}
		idleFrame, idleRepeat, hpStage := 0, 0, 0
		for frameIndex, frame := range planned {
			if frame.HPStage != 0 {
				hpStage = frame.HPStage
			}
			pixels, err := ComposeNativeCommand8TargetFrame(in.TargetBases[targetIndex][hpStage], idle.Frames[idleFrame], in.Effect, frame)
			if err != nil {
				return NativeCommand8EffectSequence{}, fmt.Errorf("battlepresent: command8 target %d frame %d: %w", targetIndex, frameIndex, err)
			}
			out.Targets[targetIndex] = append(out.Targets[targetIndex], NativeCommand8RenderedFrame{Pixels: pixels, HPStage: frame.HPStage, PlaySub1: frame.PlaySub1, PlaySub2: frame.PlaySub2})
			if idle.Frames[idleFrame].Delay <= 0 {
				return NativeCommand8EffectSequence{}, fmt.Errorf("battlepresent: command8 target %d idle delay unavailable", targetIndex)
			}
			idleRepeat++
			if idleRepeat >= idle.Frames[idleFrame].Delay {
				idleRepeat = 0
				idleFrame = (idleFrame + 1) % len(idle.Frames)
			}
		}
		lastIdleFrame, lastIdleRepeat = idleFrame, idleRepeat
		state = next
		if targetIndex+1 < len(in.TargetIdle) {
			transition, after, err := figani.BuildNativeCommand8TransitionSequence(state, in.Schedule, in.RawSide)
			if err != nil {
				return NativeCommand8EffectSequence{}, err
			}
			frames := make([]NativeCommand8RenderedFrame, 0, len(transition))
			for frameIndex, frame := range transition {
				target := idle
				if frame.UseNextTarget {
					target = in.TargetIdle[targetIndex+1]
				}
				pixels, err := ComposeNativeCommand8TransitionFrame(in.TransitionBases[targetIndex], in.ActorEffect, target, in.Effect, frame)
				if err != nil {
					return NativeCommand8EffectSequence{}, fmt.Errorf("battlepresent: command8 transition %d frame %d: %w", targetIndex, frameIndex, err)
				}
				frames = append(frames, NativeCommand8RenderedFrame{Pixels: pixels, PlaySub1: frame.PlaySub1, PlaySub2: frame.PlaySub2})
			}
			out.Transitions = append(out.Transitions, frames)
			state = after
		}
	}
	tail, _, err := figani.BuildNativeCommand8TailSequence(state, in.Schedule)
	if err != nil {
		return NativeCommand8EffectSequence{}, err
	}
	lastIdle := in.TargetIdle[len(in.TargetIdle)-1]
	for i, frame := range tail {
		tailIdle := &figani.Animation{Frames: []figani.Frame{lastIdle.Frames[lastIdleFrame]}}
		pixels, err := ComposeNativeCommand8OrbitFrame(in.TailBase, in.ActorEffect, tailIdle, in.Effect, frame)
		if err != nil {
			return NativeCommand8EffectSequence{}, fmt.Errorf("battlepresent: command8 tail frame %d: %w", i, err)
		}
		out.Tail = append(out.Tail, NativeCommand8RenderedFrame{Pixels: pixels, PlaySub1: frame.PlaySub1, PlaySub2: frame.PlaySub2})
		if lastIdle.Frames[lastIdleFrame].Delay <= 0 {
			return NativeCommand8EffectSequence{}, errors.New("battlepresent: command8 tail idle delay unavailable")
		}
		lastIdleRepeat++
		if lastIdleRepeat >= lastIdle.Frames[lastIdleFrame].Delay {
			lastIdleRepeat = 0
			lastIdleFrame = (lastIdleFrame + 1) % len(lastIdle.Frames)
		}
	}
	out.NextIdleFrame, out.NextIdleRepeat = lastIdleFrame, lastIdleRepeat
	return out, nil
}
