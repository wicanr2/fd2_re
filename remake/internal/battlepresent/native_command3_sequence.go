package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type NativeCommand3RenderedFrame struct {
	Pixels             []byte
	HPStage            int
	PlaySub1, PlaySub2 bool
}

type NativeCommand3EffectSequence struct {
	Front                         []NativeCommand3RenderedFrame
	Targets                       [][]NativeCommand3RenderedFrame
	Transitions                   [][]NativeCommand3RenderedFrame
	Tail                          []NativeCommand3RenderedFrame
	NextIdleFrame, NextIdleRepeat int
}

type NativeCommand3EffectInput struct {
	FrontBase, TailBase []byte
	TargetBases         [][][]byte
	TransitionBases     [][]byte
	ActorEffect         *figani.Animation
	TargetIdle          []*figani.Animation
	Effect              *figani.Animation
	Schedule            figani.NativeCommand3PresentationSchedule
	RawSide             byte
}

func BuildNativeCommand3EffectSequence(in NativeCommand3EffectInput) (NativeCommand3EffectSequence, error) {
	if len(in.FrontBase) != nativeCommand0SurfaceSize || len(in.TailBase) != nativeCommand0SurfaceSize ||
		in.ActorEffect == nil || len(in.ActorEffect.Frames) == 0 || in.Effect == nil ||
		len(in.TargetIdle) == 0 || len(in.TargetBases) != len(in.TargetIdle) ||
		len(in.TransitionBases) != len(in.TargetIdle)-1 || in.Schedule.EffectResource == 0 {
		return NativeCommand3EffectSequence{}, errors.New("battlepresent: incomplete command3 effect sequence input")
	}
	for targetIndex, idle := range in.TargetIdle {
		if idle == nil || len(idle.Frames) == 0 || len(in.TargetBases[targetIndex]) != figani.NativeCommand3DamageStages+1 {
			return NativeCommand3EffectSequence{}, fmt.Errorf("battlepresent: command3 target %d inputs unavailable", targetIndex)
		}
		for stage, base := range in.TargetBases[targetIndex] {
			if len(base) != nativeCommand0SurfaceSize {
				return NativeCommand3EffectSequence{}, fmt.Errorf("battlepresent: command3 target %d stage %d base unavailable", targetIndex, stage)
			}
		}
	}
	for index, base := range in.TransitionBases {
		if len(base) != nativeCommand0SurfaceSize {
			return NativeCommand3EffectSequence{}, fmt.Errorf("battlepresent: command3 transition %d base unavailable", index)
		}
	}

	out := NativeCommand3EffectSequence{Targets: make([][]NativeCommand3RenderedFrame, len(in.TargetIdle))}
	state := figani.NewNativeCommand3State()
	for frameIndex := 0; frameIndex < figani.NativeCommand3FrontFrames; frameIndex++ {
		planned, err := figani.PlanNativeCommand3DrawFrame(state, in.Schedule, in.RawSide)
		if err != nil {
			return NativeCommand3EffectSequence{}, err
		}
		pixels, err := ComposeNativeCommand3OrbitFrame(in.FrontBase, in.ActorEffect, in.TargetIdle[0], in.Effect, planned)
		if err != nil {
			return NativeCommand3EffectSequence{}, fmt.Errorf("battlepresent: command3 front frame %d: %w", frameIndex, err)
		}
		out.Front = append(out.Front, NativeCommand3RenderedFrame{Pixels: pixels, PlaySub1: planned.PlaySub1, PlaySub2: planned.PlaySub2})
		state = planned.Next
	}

	lastIdleFrame, lastIdleRepeat := 0, 0
	for targetIndex, idle := range in.TargetIdle {
		planned, next, err := figani.BuildNativeCommand3TargetSequence(state, in.Schedule, in.RawSide)
		if err != nil {
			return NativeCommand3EffectSequence{}, err
		}
		idleFrame, idleRepeat, hpStage := 0, 0, 0
		for frameIndex, frame := range planned {
			if frame.HPStage != 0 {
				hpStage = frame.HPStage
			}
			pixels, err := ComposeNativeCommand3TargetFrame(in.TargetBases[targetIndex][hpStage], idle.Frames[idleFrame], in.Effect, frame)
			if err != nil {
				return NativeCommand3EffectSequence{}, fmt.Errorf("battlepresent: command3 target %d frame %d: %w", targetIndex, frameIndex, err)
			}
			out.Targets[targetIndex] = append(out.Targets[targetIndex], NativeCommand3RenderedFrame{
				Pixels: pixels, HPStage: frame.HPStage, PlaySub1: frame.PlaySub1, PlaySub2: frame.PlaySub2,
			})
			if idle.Frames[idleFrame].Delay <= 0 {
				return NativeCommand3EffectSequence{}, fmt.Errorf("battlepresent: command3 target %d idle delay unavailable", targetIndex)
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
			transition, after, err := figani.BuildNativeCommand3TransitionSequence(state, in.Schedule, in.RawSide)
			if err != nil {
				return NativeCommand3EffectSequence{}, err
			}
			frames := make([]NativeCommand3RenderedFrame, 0, len(transition))
			for frameIndex, frame := range transition {
				target := idle
				if frame.UseNextTarget {
					target = in.TargetIdle[targetIndex+1]
				}
				pixels, err := ComposeNativeCommand3TransitionFrame(in.TransitionBases[targetIndex], in.ActorEffect, target, in.Effect, frame)
				if err != nil {
					return NativeCommand3EffectSequence{}, fmt.Errorf("battlepresent: command3 transition %d frame %d: %w", targetIndex, frameIndex, err)
				}
				frames = append(frames, NativeCommand3RenderedFrame{Pixels: pixels, PlaySub1: frame.PlaySub1, PlaySub2: frame.PlaySub2})
			}
			out.Transitions = append(out.Transitions, frames)
			state = after
		}
	}

	tail, _, err := figani.BuildNativeCommand3TailSequence(state, in.Schedule, in.RawSide)
	if err != nil {
		return NativeCommand3EffectSequence{}, err
	}
	lastIdle := in.TargetIdle[len(in.TargetIdle)-1]
	for frameIndex, frame := range tail {
		tailIdle := &figani.Animation{Frames: []figani.Frame{lastIdle.Frames[lastIdleFrame]}}
		pixels, err := ComposeNativeCommand3OrbitFrame(in.TailBase, in.ActorEffect, tailIdle, in.Effect, frame)
		if err != nil {
			return NativeCommand3EffectSequence{}, fmt.Errorf("battlepresent: command3 tail frame %d: %w", frameIndex, err)
		}
		out.Tail = append(out.Tail, NativeCommand3RenderedFrame{Pixels: pixels, PlaySub1: frame.PlaySub1, PlaySub2: frame.PlaySub2})
		if lastIdle.Frames[lastIdleFrame].Delay <= 0 {
			return NativeCommand3EffectSequence{}, errors.New("battlepresent: command3 tail idle delay unavailable")
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
