package figani

import "errors"

const (
	NativeCommand9AICommandID      = 9
	NativeCommand9AIEffectResource = 44
	NativeCommand9AISoundResource  = 90
	NativeCommand9AIFrontFrames    = 20
	NativeCommand9AITargetFrames   = 60
	NativeCommand9AITailFrames     = 20
	NativeCommand9AIDamageStages   = 20
	NativeCommand9AIActorSlideIn   = 11
	NativeCommand9AIActorSlideOut  = 8
	NativeCommand9AISlideInHoldMS  = 500
)

type NativeCommand9AISchedule struct {
	EffectResource, SoundResource             int
	FrontFrames, TargetFrames, TailFrames     int
	DamageStages, ActorSlideIn, ActorSlideOut int
	SlideInHoldMS                             int
}

type NativeCommand9AIState struct {
	Counter int
	Toggle  bool
}

type NativeCommand9AIFrame struct {
	EffectFrames       []int
	NumericMarker      bool
	HPStage            int
	PlaySub1, PlaySub2 bool
	Next               NativeCommand9AIState
}

// BuildNativeCommand9AISchedule 只接受 0x2A6BD 已複製完整十筆的
// raw-side-zero table。raw-side-nonzero table 只複製九筆，ID9 沒有可證實值。
func BuildNativeCommand9AISchedule(rawSide byte, effect *Animation) (NativeCommand9AISchedule, error) {
	if rawSide != 0 {
		return NativeCommand9AISchedule{}, errors.New("figani: command9 AI nonzero-side effect selector unavailable")
	}
	if effect == nil || len(effect.Frames) < 31 {
		return NativeCommand9AISchedule{}, errors.New("figani: command9 AI effect must contain at least 31 frames")
	}
	return NativeCommand9AISchedule{EffectResource: NativeCommand9AIEffectResource, SoundResource: NativeCommand9AISoundResource,
		FrontFrames: NativeCommand9AIFrontFrames, TargetFrames: NativeCommand9AITargetFrames,
		TailFrames: NativeCommand9AITailFrames, DamageStages: NativeCommand9AIDamageStages,
		ActorSlideIn: NativeCommand9AIActorSlideIn, ActorSlideOut: NativeCommand9AIActorSlideOut,
		SlideInHoldMS: NativeCommand9AISlideInHoldMS}, nil
}

func NewNativeCommand9AIState() NativeCommand9AIState { return NativeCommand9AIState{Counter: 1} }

func PlanNativeCommand9AIOrbitFrame(state NativeCommand9AIState, schedule NativeCommand9AISchedule) (NativeCommand9AIFrame, error) {
	if schedule.EffectResource != NativeCommand9AIEffectResource {
		return NativeCommand9AIFrame{}, errors.New("figani: command9 AI schedule unavailable")
	}
	frame := NativeCommand9AIFrame{Next: state}
	if !state.Toggle {
		frame.EffectFrames = []int{0}
	}
	frame.Next.Toggle = !state.Toggle
	return frame, nil
}

func PlanNativeCommand9AITargetFrame(state NativeCommand9AIState, schedule NativeCommand9AISchedule) (NativeCommand9AIFrame, error) {
	if schedule.EffectResource != NativeCommand9AIEffectResource || state.Counter < 1 || state.Counter > 60 {
		return NativeCommand9AIFrame{}, errors.New("figani: command9 AI target state unavailable")
	}
	frame := NativeCommand9AIFrame{EffectFrames: []int{0, state.Counter >> 1}, Next: state}
	frame.PlaySub1 = state.Counter == 6
	frame.PlaySub2 = state.Counter == 36
	frame.Next.Counter++
	frame.NumericMarker = frame.Next.Counter > 16 && frame.Next.Counter < 44
	return frame, nil
}

func BuildNativeCommand9AITargetSequence(state NativeCommand9AIState, schedule NativeCommand9AISchedule) ([]NativeCommand9AIFrame, NativeCommand9AIState, error) {
	frames := make([]NativeCommand9AIFrame, 0, schedule.TargetFrames)
	stage := 0
	for index := 0; index < schedule.TargetFrames; index++ {
		frame, err := PlanNativeCommand9AITargetFrame(state, schedule)
		if err != nil {
			return nil, state, err
		}
		if frame.NumericMarker && stage < schedule.DamageStages {
			stage++
			frame.HPStage = stage
		}
		frames = append(frames, frame)
		state = frame.Next
	}
	if stage != schedule.DamageStages {
		return nil, state, errors.New("figani: command9 AI HP marker count incomplete")
	}
	return frames, state, nil
}
