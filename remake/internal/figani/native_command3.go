package figani

import (
	"errors"
	"fmt"
)

const (
	NativeCommand3EffectFrameCount = 33
	NativeCommand3ChannelCount     = 12
	NativeCommand3PositionCount    = 12
	NativeCommand3SoundResource    = 84
	NativeCommand3FrontFrames      = 2
	NativeCommand3TargetFrames     = 40
	NativeCommand3TransitionFrames = 9
	NativeCommand3TailFrames       = 20
	NativeCommand3DamageStages     = 13
)

var nativeCommand3XOffsets = [NativeCommand3PositionCount]int{30, 0, 70, 40, 130, 70, -30, 30, 110, 80, -10, 30}
var nativeCommand3YOffsets = [NativeCommand3PositionCount]int{0, 10, 0, 20, 5, 20, 0, 10, 0, 18, 0, 10}
var nativeCommand3FrameBases = [NativeCommand3PositionCount]int{22, 0, 0, 0, 0, 0, 11, 0, 22, 0, 0, 0}

type NativeCommand3PresentationSchedule struct {
	EffectResource int
	SoundResource  int
	XOffsets       [NativeCommand3PositionCount]int
	YOffsets       [NativeCommand3PositionCount]int
	FrameBases     [NativeCommand3PositionCount]int
}

type NativeCommand3State struct {
	Counters     [NativeCommand3ChannelCount]int
	Positions    [NativeCommand3ChannelCount]int
	NextPosition int
	Stop         bool
	Toggle       int
}

type NativeCommand3Layer struct {
	Channel, Frame, X, Y int
}

type NativeCommand3Frame struct {
	Layers                  []NativeCommand3Layer
	NumericMarker, PlaySub1 bool
	PlaySub2                bool
	HPStage                 int
	Next                    NativeCommand3State
}

type NativeCommand3TransitionFrame struct {
	NativeCommand3Frame
	UseNextTarget bool
	TargetOffsetX int
}

func BuildNativeCommand3PresentationSchedule(rawSide byte, effect *Animation) (NativeCommand3PresentationSchedule, error) {
	if effect == nil || len(effect.Frames) != NativeCommand3EffectFrameCount {
		return NativeCommand3PresentationSchedule{}, errors.New("figani: command3 effect must contain 33 frames")
	}
	resource := 39
	if rawSide == 0 {
		resource = 43
	}
	return NativeCommand3PresentationSchedule{
		EffectResource: resource, SoundResource: NativeCommand3SoundResource,
		XOffsets: nativeCommand3XOffsets, YOffsets: nativeCommand3YOffsets, FrameBases: nativeCommand3FrameBases,
	}, nil
}

func NewNativeCommand3State() NativeCommand3State {
	state := NativeCommand3State{NextPosition: NativeCommand3PositionCount}
	for channel := 0; channel < NativeCommand3ChannelCount; channel++ {
		state.Counters[channel] = -2 * channel
		state.Positions[channel] = channel
	}
	return state
}

func PlanNativeCommand3DrawFrame(state NativeCommand3State, schedule NativeCommand3PresentationSchedule, rawSide byte) (NativeCommand3Frame, error) {
	if schedule.EffectResource == 0 || schedule.SoundResource != NativeCommand3SoundResource {
		return NativeCommand3Frame{}, errors.New("figani: command3 schedule unavailable")
	}
	next := state
	next.Toggle = (next.Toggle + 1) % 2
	frame := NativeCommand3Frame{Next: next}
	for channel := 0; channel < NativeCommand3ChannelCount; channel++ {
		position := state.Positions[channel]
		if position < 0 || position >= NativeCommand3PositionCount {
			return NativeCommand3Frame{}, fmt.Errorf("figani: command3 channel %d position %d invalid", channel, position)
		}
		counter := state.Counters[channel]
		if counter >= 0 && counter < 11 {
			x := schedule.XOffsets[position]
			if rawSide == 0 {
				x += 20
			}
			frame.Layers = append(frame.Layers, NativeCommand3Layer{
				Channel: channel, Frame: schedule.FrameBases[position] + counter,
				X: x, Y: -schedule.YOffsets[position],
			})
		}
	}
	if next.Toggle != 0 {
		frame.Next = next
		return frame, nil
	}
	for channel := 0; channel < NativeCommand3ChannelCount; channel++ {
		position := next.Positions[channel]
		counter := next.Counters[channel]
		if counter == 0 && schedule.FrameBases[position] != 0 {
			frame.PlaySub2 = true
		}
		counter++
		next.Counters[channel] = counter
		if counter == 3 {
			frame.NumericMarker = true
			if schedule.FrameBases[position] == 0 {
				frame.PlaySub1 = true
			}
		}
		if counter == 11 && !next.Stop {
			next.NextPosition = (next.NextPosition + 1) % NativeCommand3PositionCount
			next.Positions[channel] = next.NextPosition
			next.Counters[channel] = 0
		}
	}
	frame.Next = next
	return frame, nil
}

func BuildNativeCommand3TargetSequence(state NativeCommand3State, schedule NativeCommand3PresentationSchedule, rawSide byte) ([]NativeCommand3Frame, NativeCommand3State, error) {
	frames := make([]NativeCommand3Frame, 0, NativeCommand3TargetFrames)
	hpStage := 0
	for index := 0; index < NativeCommand3TargetFrames; index++ {
		frame, err := PlanNativeCommand3DrawFrame(state, schedule, rawSide)
		if err != nil {
			return nil, state, err
		}
		if frame.NumericMarker && hpStage < NativeCommand3DamageStages {
			hpStage++
			frame.HPStage = hpStage
		}
		frames = append(frames, frame)
		state = frame.Next
	}
	if hpStage != NativeCommand3DamageStages {
		return nil, state, fmt.Errorf("figani: command3 HP marker count incomplete: %d", hpStage)
	}
	return frames, state, nil
}

func BuildNativeCommand3TransitionSequence(state NativeCommand3State, schedule NativeCommand3PresentationSchedule, rawSide byte) ([]NativeCommand3TransitionFrame, NativeCommand3State, error) {
	direction := -1
	if rawSide == 0 {
		direction = 1
	}
	frames := make([]NativeCommand3TransitionFrame, 0, NativeCommand3TransitionFrames)
	appendFrame := func(useNext bool, step int) error {
		planned, err := PlanNativeCommand3DrawFrame(state, schedule, rawSide)
		if err != nil {
			return err
		}
		frames = append(frames, NativeCommand3TransitionFrame{NativeCommand3Frame: planned, UseNextTarget: useNext, TargetOffsetX: direction * 35 * step})
		state = planned.Next
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

func BuildNativeCommand3TailSequence(state NativeCommand3State, schedule NativeCommand3PresentationSchedule, rawSide byte) ([]NativeCommand3Frame, NativeCommand3State, error) {
	state.Stop = true
	frames := make([]NativeCommand3Frame, 0, NativeCommand3TailFrames)
	for index := 0; index < NativeCommand3TailFrames; index++ {
		frame, err := PlanNativeCommand3DrawFrame(state, schedule, rawSide)
		if err != nil {
			return nil, state, err
		}
		frames = append(frames, frame)
		state = frame.Next
	}
	return frames, state, nil
}
