package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type NativeCommand7RenderedFrame struct {
	Pixels      []byte
	HPStage     int
	PlayHandleA bool
	PlayHandleB bool
}

type NativeCommand7EffectSequence struct {
	Front                         []NativeCommand7RenderedFrame
	Targets                       [][]NativeCommand7RenderedFrame
	Transitions                   [][]NativeCommand7RenderedFrame
	Tail                          []NativeCommand7RenderedFrame
	NextIdleFrame, NextIdleRepeat int
}

type NativeCommand7EffectInput struct {
	FrontBase, TailBase []byte
	TargetBases         [][][]byte
	TransitionBases     [][]byte
	ActorEffect         *figani.Animation
	TargetIdle          []*figani.Animation
	Effect              *figani.Animation
	Schedule            figani.NativeCommand7PresentationSchedule
	RawSide             byte
}

func BuildNativeCommand7EffectSequence(in NativeCommand7EffectInput) (NativeCommand7EffectSequence, error) {
	if len(in.FrontBase) != nativeCommand0SurfaceSize || len(in.TailBase) != nativeCommand0SurfaceSize ||
		in.ActorEffect == nil || len(in.ActorEffect.Frames) == 0 || in.Effect == nil ||
		len(in.TargetIdle) == 0 || len(in.TargetBases) != len(in.TargetIdle) ||
		len(in.TransitionBases) != len(in.TargetIdle)-1 || in.Schedule.EffectResource == 0 {
		return NativeCommand7EffectSequence{}, errors.New("battlepresent: incomplete command7 effect sequence input")
	}
	for targetIndex, idle := range in.TargetIdle {
		if idle == nil || len(idle.Frames) == 0 || len(in.TargetBases[targetIndex]) != figani.NativeCommand7DamageStages+1 {
			return NativeCommand7EffectSequence{}, fmt.Errorf("battlepresent: command7 target %d inputs unavailable", targetIndex)
		}
		for stage, base := range in.TargetBases[targetIndex] {
			if len(base) != nativeCommand0SurfaceSize {
				return NativeCommand7EffectSequence{}, fmt.Errorf("battlepresent: command7 target %d stage %d base unavailable", targetIndex, stage)
			}
		}
	}
	for index, base := range in.TransitionBases {
		if len(base) != nativeCommand0SurfaceSize {
			return NativeCommand7EffectSequence{}, fmt.Errorf("battlepresent: command7 transition %d base unavailable", index)
		}
	}

	out := NativeCommand7EffectSequence{Targets: make([][]NativeCommand7RenderedFrame, len(in.TargetIdle))}
	state := figani.NewNativeCommand7State()
	for frameIndex := 0; frameIndex < figani.NativeCommand7FrontFrames; frameIndex++ {
		planned, err := figani.PlanNativeCommand7DrawFrame(state, in.Schedule, in.RawSide)
		if err != nil {
			return NativeCommand7EffectSequence{}, err
		}
		pixels, err := ComposeNativeCommand7OrbitFrame(in.FrontBase, in.ActorEffect, in.TargetIdle[0], in.Effect, planned)
		if err != nil {
			return NativeCommand7EffectSequence{}, fmt.Errorf("battlepresent: command7 front frame %d: %w", frameIndex, err)
		}
		out.Front = append(out.Front, NativeCommand7RenderedFrame{Pixels: pixels, PlayHandleA: planned.PlayHandleA, PlayHandleB: planned.PlayHandleB})
		state = planned.Next
	}

	lastIdleFrame, lastIdleRepeat := 0, 0
	for targetIndex, idle := range in.TargetIdle {
		planned, next, err := figani.BuildNativeCommand7TargetSequence(state, in.Schedule, in.RawSide)
		if err != nil {
			return NativeCommand7EffectSequence{}, err
		}
		idleFrame, idleRepeat, hpStage := 0, 0, 0
		for frameIndex, frame := range planned {
			if frame.HPStage != 0 {
				hpStage = frame.HPStage
			}
			pixels, err := ComposeNativeCommand7TargetFrame(in.TargetBases[targetIndex][hpStage], idle.Frames[idleFrame], in.Effect, frame)
			if err != nil {
				return NativeCommand7EffectSequence{}, fmt.Errorf("battlepresent: command7 target %d frame %d: %w", targetIndex, frameIndex, err)
			}
			out.Targets[targetIndex] = append(out.Targets[targetIndex], NativeCommand7RenderedFrame{
				Pixels: pixels, HPStage: frame.HPStage, PlayHandleA: frame.PlayHandleA, PlayHandleB: frame.PlayHandleB,
			})
			if idle.Frames[idleFrame].Delay <= 0 {
				return NativeCommand7EffectSequence{}, fmt.Errorf("battlepresent: command7 target %d idle delay unavailable", targetIndex)
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
			transition, after, err := figani.BuildNativeCommand7TransitionSequence(state, in.Schedule, in.RawSide)
			if err != nil {
				return NativeCommand7EffectSequence{}, err
			}
			frames := make([]NativeCommand7RenderedFrame, 0, len(transition))
			for frameIndex, frame := range transition {
				target := idle
				if frame.UseNextTarget {
					target = in.TargetIdle[targetIndex+1]
				}
				pixels, err := ComposeNativeCommand7TransitionFrame(in.TransitionBases[targetIndex], in.ActorEffect, target, in.Effect, frame)
				if err != nil {
					return NativeCommand7EffectSequence{}, fmt.Errorf("battlepresent: command7 transition %d frame %d: %w", targetIndex, frameIndex, err)
				}
				frames = append(frames, NativeCommand7RenderedFrame{Pixels: pixels, PlayHandleA: frame.PlayHandleA, PlayHandleB: frame.PlayHandleB})
			}
			out.Transitions = append(out.Transitions, frames)
			state = after
		}
	}

	tail, _, err := figani.BuildNativeCommand7TailSequence(state, in.Schedule, in.RawSide)
	if err != nil {
		return NativeCommand7EffectSequence{}, err
	}
	lastIdle := in.TargetIdle[len(in.TargetIdle)-1]
	for frameIndex, frame := range tail {
		tailIdle := &figani.Animation{Frames: []figani.Frame{lastIdle.Frames[lastIdleFrame]}}
		pixels, err := ComposeNativeCommand7OrbitFrame(in.TailBase, in.ActorEffect, tailIdle, in.Effect, frame)
		if err != nil {
			return NativeCommand7EffectSequence{}, fmt.Errorf("battlepresent: command7 tail frame %d: %w", frameIndex, err)
		}
		out.Tail = append(out.Tail, NativeCommand7RenderedFrame{Pixels: pixels, PlayHandleA: frame.PlayHandleA, PlayHandleB: frame.PlayHandleB})
		if lastIdle.Frames[lastIdleFrame].Delay <= 0 {
			return NativeCommand7EffectSequence{}, errors.New("battlepresent: command7 tail idle delay unavailable")
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
