package figani

import "fmt"

const (
	NativeCommand7EffectFrameCount = 5
	NativeCommand7InitChannelCount = 4
	NativeCommand7DrawChannelCount = 3
	NativeCommand7PositionCount    = 10
	NativeCommand7SoundResource    = 88
	NativeCommand7FrontFrames      = 2
	NativeCommand7TargetFrames     = 32
	NativeCommand7TransitionFrames = 9
	NativeCommand7TailFrames       = 16
	NativeCommand7DamageStages     = 5
)

var nativeCommand7XOffsets = [NativeCommand7PositionCount]int{30, -10, 70, 20, 100, 130, 40, 80, 110, 60}

type NativeCommand7PresentationSchedule struct {
	EffectResource int
	SoundResource  int
	SoundIndices   [2]int
	XOffsets       [NativeCommand7PositionCount]int
}

type NativeCommand7State struct {
	Counters     [NativeCommand7InitChannelCount]int
	Positions    [NativeCommand7InitChannelCount]int
	NextPosition int
	Stop         bool
	Toggle       byte
}

type NativeCommand7Layer struct {
	Channel int
	Frame   int
	X       int
}

type NativeCommand7Frame struct {
	Layers        []NativeCommand7Layer
	Next          NativeCommand7State
	NumericMarker bool
	HPStage       int
	PlayHandleA   bool
	PlayHandleB   bool
}

type NativeCommand7TransitionFrame struct {
	NativeCommand7Frame
	UseNextTarget bool
	TargetOffsetX int
}

func BuildNativeCommand7PresentationSchedule(rawSide byte, effect *Animation) (NativeCommand7PresentationSchedule, error) {
	resource := 37
	if rawSide == 0 {
		resource = 38
	}
	if effect == nil || effect.HeaderByte1 != 0 || effect.HeaderByte2 != NativeCommand7EffectFrameCount ||
		effect.HeaderByte4 != 0 || len(effect.Frames) != NativeCommand7EffectFrameCount {
		return NativeCommand7PresentationSchedule{}, fmt.Errorf("figani: command7 FDOTHER #%d signature mismatch", resource)
	}
	for index, frame := range effect.Frames {
		if frame.Width <= 0 || frame.Height <= 0 || len(frame.Pixels) != frame.Width*frame.Height || len(frame.Mask) != len(frame.Pixels) {
			return NativeCommand7PresentationSchedule{}, fmt.Errorf("figani: command7 FDOTHER #%d frame %d malformed", resource, index)
		}
	}
	return NativeCommand7PresentationSchedule{
		EffectResource: resource,
		SoundResource:  NativeCommand7SoundResource,
		SoundIndices:   [2]int{0, 1},
		XOffsets:       nativeCommand7XOffsets,
	}, nil
}

func NewNativeCommand7State() NativeCommand7State {
	state := NativeCommand7State{NextPosition: NativeCommand7InitChannelCount}
	for channel := 0; channel < NativeCommand7InitChannelCount; channel++ {
		state.Counters[channel] = -3 * channel
		state.Positions[channel] = channel
	}
	return state
}

// PlanNativeCommand7DrawFrame 重現mode2／5／8的toggle雙張與三通道loop。
// mode1／4／7不畫handler effect。
func PlanNativeCommand7DrawFrame(state NativeCommand7State, schedule NativeCommand7PresentationSchedule, rawSide byte) (NativeCommand7Frame, error) {
	wantResource, xShift := 37, 0
	if rawSide == 0 {
		wantResource, xShift = 38, 130
	}
	if schedule.EffectResource != wantResource {
		return NativeCommand7Frame{}, fmt.Errorf("figani: command7 side mismatch")
	}
	next := state
	next.Toggle = (next.Toggle + 1) % 2
	frame := NativeCommand7Frame{}
	for channel := 0; channel < NativeCommand7DrawChannelCount; channel++ {
		counter := next.Counters[channel]
		if counter >= 0 && counter < NativeCommand7EffectFrameCount {
			position := next.Positions[channel]
			if position < 0 || position >= len(schedule.XOffsets) {
				return NativeCommand7Frame{}, fmt.Errorf("figani: command7 position %d unavailable", position)
			}
			frame.Layers = append(frame.Layers, NativeCommand7Layer{Channel: channel, Frame: counter, X: schedule.XOffsets[position] + xShift})
		}
		if next.Toggle != 0 {
			continue
		}
		if counter == 1 && channel == 0 {
			frame.PlayHandleA = true
		}
		if counter == 1 && channel == 1 {
			frame.PlayHandleB = true
		}
		next.Counters[channel]++
		if next.Counters[channel] == 2 {
			frame.NumericMarker = true
		}
		if next.Counters[channel] == 7 && !next.Stop {
			next.NextPosition = (next.NextPosition + 1) % NativeCommand7PositionCount
			next.Positions[channel] = next.NextPosition
			next.Counters[channel] = 0
		}
	}
	frame.Next = next
	return frame, nil
}

func BuildNativeCommand7TargetSequence(state NativeCommand7State, schedule NativeCommand7PresentationSchedule, rawSide byte) ([]NativeCommand7Frame, NativeCommand7State, error) {
	frames := make([]NativeCommand7Frame, 0, NativeCommand7TargetFrames)
	hpStage := 0
	for index := 0; index < NativeCommand7TargetFrames; index++ {
		frame, err := PlanNativeCommand7DrawFrame(state, schedule, rawSide)
		if err != nil {
			return nil, state, err
		}
		if frame.NumericMarker && hpStage < NativeCommand7DamageStages {
			hpStage++
			frame.HPStage = hpStage
		}
		frames = append(frames, frame)
		state = frame.Next
	}
	if hpStage != NativeCommand7DamageStages {
		return nil, state, fmt.Errorf("figani: command7 HP marker count incomplete: %d", hpStage)
	}
	return frames, state, nil
}

func BuildNativeCommand7TransitionSequence(state NativeCommand7State, schedule NativeCommand7PresentationSchedule, rawSide byte) ([]NativeCommand7TransitionFrame, NativeCommand7State, error) {
	direction := -1
	if rawSide == 0 {
		direction = 1
	}
	frames := make([]NativeCommand7TransitionFrame, 0, NativeCommand7TransitionFrames)
	appendFrame := func(useNext bool, step int) error {
		planned, err := PlanNativeCommand7DrawFrame(state, schedule, rawSide)
		if err != nil {
			return err
		}
		frames = append(frames, NativeCommand7TransitionFrame{NativeCommand7Frame: planned, UseNextTarget: useNext, TargetOffsetX: direction * 35 * step})
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

func BuildNativeCommand7TailSequence(state NativeCommand7State, schedule NativeCommand7PresentationSchedule, rawSide byte) ([]NativeCommand7Frame, NativeCommand7State, error) {
	state.Stop = true
	frames := make([]NativeCommand7Frame, 0, NativeCommand7TailFrames)
	for index := 0; index < NativeCommand7TailFrames; index++ {
		frame, err := PlanNativeCommand7DrawFrame(state, schedule, rawSide)
		if err != nil {
			return nil, state, err
		}
		frames = append(frames, frame)
		state = frame.Next
	}
	return frames, state, nil
}
