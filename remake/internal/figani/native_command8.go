package figani

import (
	"errors"
	"fmt"
)

const (
	NativeCommand8EffectFrameCount = 32
	NativeCommand8ChannelCount     = 16
	NativeCommand8SoundResource    = 90
	NativeCommand8FrontFrames      = 3
	NativeCommand8TargetFrames     = 34
	NativeCommand8TransitionFrames = 9
	NativeCommand8TailFrames       = 2
	NativeCommand8DamageStages     = 16
)

var nativeCommand8FrameBases = [NativeCommand8ChannelCount]int{0, 8, 24, 16, 8, 0, 24, 16, 0, 8, 24, 16, 8, 0, 24, 16}

type NativeCommand8PresentationSchedule struct {
	EffectResource int
	SoundResource  int
	FrameBases     [NativeCommand8ChannelCount]int
}

type NativeCommand8State struct {
	Counters [NativeCommand8ChannelCount]int
}
type NativeCommand8Layer struct{ Channel, Frame int }
type NativeCommand8Frame struct {
	Layers                            []NativeCommand8Layer
	NumericMarker, PlaySub1, PlaySub2 bool
	HPStage                           int
	Next                              NativeCommand8State
}
type NativeCommand8TransitionFrame struct {
	NativeCommand8Frame
	UseNextTarget bool
	TargetOffsetX int
}

func BuildNativeCommand8PresentationSchedule(rawSide byte, effect *Animation) (NativeCommand8PresentationSchedule, error) {
	if effect == nil || len(effect.Frames) != NativeCommand8EffectFrameCount {
		return NativeCommand8PresentationSchedule{}, errors.New("figani: command8 effect must contain 32 frames")
	}
	resource := 28
	if rawSide == 0 {
		resource = 30
	}
	return NativeCommand8PresentationSchedule{EffectResource: resource, SoundResource: NativeCommand8SoundResource, FrameBases: nativeCommand8FrameBases}, nil
}

func NewNativeCommand8State() NativeCommand8State {
	var state NativeCommand8State
	for channel := range state.Counters {
		state.Counters[channel] = -2 * channel
	}
	return state
}

func PlanNativeCommand8DrawFrame(state NativeCommand8State, schedule NativeCommand8PresentationSchedule) (NativeCommand8Frame, error) {
	if schedule.EffectResource == 0 || schedule.SoundResource != NativeCommand8SoundResource {
		return NativeCommand8Frame{}, errors.New("figani: command8 schedule unavailable")
	}
	frame := NativeCommand8Frame{Next: state}
	for channel, counter := range state.Counters {
		if counter >= 0 && counter < 8 {
			index := schedule.FrameBases[channel] + counter
			if index < 0 || index >= NativeCommand8EffectFrameCount {
				return NativeCommand8Frame{}, fmt.Errorf("figani: command8 channel %d frame %d invalid", channel, index)
			}
			frame.Layers = append(frame.Layers, NativeCommand8Layer{Channel: channel, Frame: index})
		}
		if counter == 0 {
			frame.PlaySub1 = true
		}
		if counter == 4 {
			frame.PlaySub2 = true
		}
		counter++
		frame.Next.Counters[channel] = counter
		if counter == 4 {
			frame.NumericMarker = true
		}
	}
	return frame, nil
}

func BuildNativeCommand8TargetSequence(state NativeCommand8State, schedule NativeCommand8PresentationSchedule) ([]NativeCommand8Frame, NativeCommand8State, error) {
	frames := make([]NativeCommand8Frame, 0, NativeCommand8TargetFrames)
	stage := 0
	for i := 0; i < NativeCommand8TargetFrames; i++ {
		frame, err := PlanNativeCommand8DrawFrame(state, schedule)
		if err != nil {
			return nil, state, err
		}
		if frame.NumericMarker && stage < NativeCommand8DamageStages {
			stage++
			frame.HPStage = stage
		}
		frames = append(frames, frame)
		state = frame.Next
	}
	if stage != NativeCommand8DamageStages {
		return nil, state, fmt.Errorf("figani: command8 HP marker count incomplete: %d", stage)
	}
	return frames, state, nil
}

func BuildNativeCommand8TransitionSequence(state NativeCommand8State, schedule NativeCommand8PresentationSchedule, rawSide byte) ([]NativeCommand8TransitionFrame, NativeCommand8State, error) {
	direction := -1
	if rawSide == 0 {
		direction = 1
	}
	frames := make([]NativeCommand8TransitionFrame, 0, NativeCommand8TransitionFrames)
	appendFrame := func(next bool, step int) error {
		frame, err := PlanNativeCommand8DrawFrame(state, schedule)
		if err != nil {
			return err
		}
		frames = append(frames, NativeCommand8TransitionFrame{NativeCommand8Frame: frame, UseNextTarget: next, TargetOffsetX: direction * 35 * step})
		state = frame.Next
		return nil
	}
	for step := 1; step <= 4; step++ {
		if err := appendFrame(false, step); err != nil {
			return nil, state, err
		}
	}
	for step := 4; step >= 0; step-- {
		if err := appendFrame(true, step); err != nil {
			return nil, state, err
		}
	}
	return frames, state, nil
}

func BuildNativeCommand8TailSequence(state NativeCommand8State, schedule NativeCommand8PresentationSchedule) ([]NativeCommand8Frame, NativeCommand8State, error) {
	frames := make([]NativeCommand8Frame, 0, NativeCommand8TailFrames)
	for i := 0; i < NativeCommand8TailFrames; i++ {
		frame, err := PlanNativeCommand8DrawFrame(state, schedule)
		if err != nil {
			return nil, state, err
		}
		frames = append(frames, frame)
		state = frame.Next
	}
	return frames, state, nil
}
