package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type NativeCommand6TargetSequence struct {
	Frames   [][]byte
	HPStages []int
}

type NativeCommand6EffectSequence struct {
	RNGAfter    uint16
	Front       [][]byte
	Targets     []NativeCommand6TargetSequence
	Transitions [][][]byte
	Tail        [][]byte
}

type NativeCommand6EffectInput struct {
	TargetHits          []bool
	TargetNumericRNG    []uint16
	FrontBase, TailBase []byte
	TargetBases         [][][]byte
	TransitionBases     [][]byte
	ActorEffect         *figani.Animation
	TargetIdle          []*figani.Animation
	Effect              *figani.Animation
	Schedule            figani.NativeCommand6PresentationSchedule
	RawSide             byte
}

// BuildNativeCommand6EffectSequence preflights every handler-owned command6
// frame before the caller publishes MP. Common 0x29164 and 0x2B659 phases are
// intentionally supplied by their existing scene owner.
func BuildNativeCommand6EffectSequence(in NativeCommand6EffectInput) (NativeCommand6EffectSequence, error) {
	if len(in.FrontBase) != nativeCommand0SurfaceSize || len(in.TailBase) != nativeCommand0SurfaceSize ||
		in.ActorEffect == nil || len(in.ActorEffect.Frames) == 0 || in.Effect == nil ||
		len(in.TargetIdle) == 0 || len(in.TargetBases) != len(in.TargetIdle) ||
		len(in.TargetHits) != len(in.TargetIdle) || len(in.TargetNumericRNG) != len(in.TargetIdle) ||
		len(in.TransitionBases) != len(in.TargetIdle)-1 || in.Schedule.EffectResource == 0 {
		return NativeCommand6EffectSequence{}, errors.New("battlepresent: incomplete command6 effect sequence input")
	}
	for index, idle := range in.TargetIdle {
		if idle == nil || len(idle.Frames) == 0 || len(in.TargetBases[index]) != figani.NativeCommand6DamageStages+1 {
			return NativeCommand6EffectSequence{}, fmt.Errorf("battlepresent: command6 target %d inputs unavailable", index)
		}
		for stage, base := range in.TargetBases[index] {
			if len(base) != nativeCommand0SurfaceSize {
				return NativeCommand6EffectSequence{}, fmt.Errorf("battlepresent: command6 target %d stage %d base unavailable", index, stage)
			}
		}
	}
	for index, base := range in.TransitionBases {
		if len(base) != nativeCommand0SurfaceSize {
			return NativeCommand6EffectSequence{}, fmt.Errorf("battlepresent: command6 transition %d base unavailable", index)
		}
	}
	out := NativeCommand6EffectSequence{Targets: make([]NativeCommand6TargetSequence, len(in.TargetIdle))}

	frontBase := append([]byte(nil), in.FrontBase...)
	if err := in.TargetIdle[0].Frames[0].BlitAt(frontBase, 320); err != nil {
		return NativeCommand6EffectSequence{}, fmt.Errorf("battlepresent: command6 front target: %w", err)
	}
	radius := 0
	for frameIndex := 0; frameIndex < 7; frameIndex++ {
		planned, err := figani.PlanNativeCommand6PreludeFrame(radius, in.RawSide, in.Schedule)
		if err != nil {
			return NativeCommand6EffectSequence{}, err
		}
		pixels, err := ComposeNativeCommand6OrbitFrame(frontBase, in.ActorEffect, nil, in.Effect, planned)
		if err != nil {
			return NativeCommand6EffectSequence{}, err
		}
		out.Front = append(out.Front, pixels)
		radius = planned.NextRadius
	}
	if radius != 42 {
		return NativeCommand6EffectSequence{}, fmt.Errorf("battlepresent: command6 front radius=%d", radius)
	}
	points := figani.NativeCommand6TargetCoordinates(figani.NativeCommand6Coordinates(36, in.Schedule.BaseByte), radius)
	state := figani.NewNativeCommand6TargetState()

	lastTargetFrame := 0
	displayState := figani.NewNativeCommand6DisplayState()
	for targetIndex, idle := range in.TargetIdle {
		planned, err := figani.BuildNativeCommand6TargetSequenceFromState(state, in.Schedule, points, in.RawSide)
		if err != nil {
			return NativeCommand6EffectSequence{}, err
		}

		displays, err := figani.BuildNativeCommand6TargetDisplayFrames(displayState, planned, in.TargetHits[targetIndex], in.TargetNumericRNG[targetIndex])
		if err != nil {
			return NativeCommand6EffectSequence{}, err
		}
		displayState = displays[len(displays)-1].Next
		out.RNGAfter = displays[len(displays)-1].RNGAfter
		idleFrame, idleRepeat, hpStage := 0, 0, 0
		for frameIndex, frame := range planned {
			pixels, err := ComposeNativeCommand6TargetFrame(in.TargetBases[targetIndex][hpStage], in.ActorEffect.Frames[len(in.ActorEffect.Frames)-1], idle.Frames[idleFrame], in.Effect, frame, displays[frameIndex])
			if err != nil {
				return NativeCommand6EffectSequence{}, fmt.Errorf("battlepresent: command6 target %d frame %d: %w", targetIndex, frameIndex, err)
			}
			out.Targets[targetIndex].Frames = append(out.Targets[targetIndex].Frames, pixels)
			out.Targets[targetIndex].HPStages = append(out.Targets[targetIndex].HPStages, frame.HPStage)
			if frame.HPStage != 0 {
				hpStage = frame.HPStage
			}
			idleRepeat++
			if idle.Frames[idleFrame].Delay <= 0 {
				return NativeCommand6EffectSequence{}, fmt.Errorf("battlepresent: command6 target %d idle delay unavailable", targetIndex)
			}
			if idleRepeat >= idle.Frames[idleFrame].Delay {
				idleRepeat = 0
				idleFrame = (idleFrame + 1) % len(idle.Frames)
			}
		}
		lastTargetFrame = idleFrame
		state = planned[len(planned)-1].Next
		if targetIndex+1 < len(in.TargetIdle) {
			transitionPlan, err := figani.BuildNativeCommand6TransitionSequence(state, in.Schedule, points, in.RawSide)
			if err != nil {
				return NativeCommand6EffectSequence{}, err
			}
			frames := make([][]byte, 0, len(transitionPlan))
			for frameIndex, frame := range transitionPlan {
				target := idle
				if frame.UseNextTarget {
					target = in.TargetIdle[targetIndex+1]
				}
				pixels, err := ComposeNativeCommand6TransitionFrame(in.TransitionBases[targetIndex], in.ActorEffect, target, in.Effect, frame)
				if err != nil {
					return NativeCommand6EffectSequence{}, fmt.Errorf("battlepresent: command6 transition %d frame %d: %w", targetIndex, frameIndex, err)
				}
				frames = append(frames, pixels)
			}
			out.Transitions = append(out.Transitions, frames)
			state = transitionPlan[len(transitionPlan)-1].Next
		}
	}

	radius = 42
	lastIdle := in.TargetIdle[len(in.TargetIdle)-1]
	for frameIndex := 0; frameIndex < 7; frameIndex++ {
		planned, err := figani.PlanNativeCommand6TailFrame(radius, in.RawSide, in.Schedule)
		if err != nil {
			return NativeCommand6EffectSequence{}, err
		}
		pixels, err := ComposeNativeCommand6OrbitFrame(in.TailBase, in.ActorEffect, &lastIdle.Frames[lastTargetFrame], in.Effect, planned)
		if err != nil {
			return NativeCommand6EffectSequence{}, fmt.Errorf("battlepresent: command6 tail frame %d: %w", frameIndex, err)
		}
		out.Tail = append(out.Tail, pixels)
		radius = planned.NextRadius
	}
	if radius != 0 {
		return NativeCommand6EffectSequence{}, fmt.Errorf("battlepresent: command6 tail radius=%d", radius)
	}
	return out, nil
}
