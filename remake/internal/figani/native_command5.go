package figani

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

const (
	NativeCommand5EffectFrameCount = 12
	NativeCommand5ChannelCount     = 6
	NativeCommand5PositionCount    = 10
	NativeCommand5SoundResource    = 86
	NativeCommand5FrontFrames      = 1
	NativeCommand5TargetFrames     = 12
	NativeCommand5TransitionFrames = 9
	NativeCommand5TailFrames       = 8
	NativeCommand5DamageStages     = 6
)

var nativeCommand5XOffsets = [NativeCommand5PositionCount]int{30, 50, 70, 40, 80, 100, 70, 30, 60, 90}

type NativeCommand5PresentationSchedule struct {
	EffectResource int
	SoundResource  int
	SoundIndices   [2]int
	XOffsets       [NativeCommand5PositionCount]int
}

type NativeCommand5State struct {
	Counters     [NativeCommand5ChannelCount]int
	Positions    [NativeCommand5ChannelCount]int
	Phases       [NativeCommand5ChannelCount]int
	NextPosition int
	Stop         bool
	RNG          uint16
}

type NativeCommand5Layer struct {
	Channel int
	Frame   int
	X       int
}

type NativeCommand5Frame struct {
	Layers        []NativeCommand5Layer
	Next          NativeCommand5State
	NumericMarker bool
	HPStage       int
	PlayPrimary   bool
	PlaySecondary bool
}

type NativeCommand5TransitionFrame struct {
	NativeCommand5Frame
	UseNextTarget bool
	TargetOffsetX int
}

func BuildNativeCommand5PresentationSchedule(rawSide byte, effect *Animation) (NativeCommand5PresentationSchedule, error) {
	resource := 24
	if rawSide == 0 {
		resource = 25
	}
	if effect == nil || effect.HeaderByte1 != 0 || effect.HeaderByte2 != NativeCommand5EffectFrameCount ||
		effect.HeaderByte4 != 0 || len(effect.Frames) != NativeCommand5EffectFrameCount {
		return NativeCommand5PresentationSchedule{}, fmt.Errorf("figani: command5 FDOTHER #%d signature mismatch", resource)
	}
	for index, frame := range effect.Frames {
		if frame.Width <= 0 || frame.Height <= 0 || len(frame.Pixels) != frame.Width*frame.Height || len(frame.Mask) != len(frame.Pixels) {
			return NativeCommand5PresentationSchedule{}, fmt.Errorf("figani: command5 FDOTHER #%d frame %d malformed", resource, index)
		}
	}
	return NativeCommand5PresentationSchedule{
		EffectResource: resource,
		SoundResource:  NativeCommand5SoundResource,
		SoundIndices:   [2]int{0, 1},
		XOffsets:       nativeCommand5XOffsets,
	}, nil
}

// NewNativeCommand5State 重現 mode0：六個相位各消耗一次 0x4E893，
// 並保留下一個循環位置 6。raw side 只影響畫面水平錨點，不影響狀態。
func NewNativeCommand5State(rng uint16) NativeCommand5State {
	state := NativeCommand5State{NextPosition: NativeCommand5ChannelCount, RNG: rng}
	for channel := 0; channel < NativeCommand5ChannelCount; channel++ {
		state.Counters[channel] = -2 * channel
		state.Positions[channel] = channel
		state.RNG = fdother.NativeRNGStep(state.RNG)
		state.Phases[channel] = int(state.RNG%2) * 6
	}
	return state
}

// PlanNativeCommand5DrawFrame 重現 mode2／5／8 共用的六通道迴圈。
// mode1／4／7 不畫 handler effect，故不在此函式內表示。
func PlanNativeCommand5DrawFrame(state NativeCommand5State, schedule NativeCommand5PresentationSchedule, rawSide byte) (NativeCommand5Frame, error) {
	if schedule.EffectResource != 24 && schedule.EffectResource != 25 {
		return NativeCommand5Frame{}, fmt.Errorf("figani: command5 schedule unavailable")
	}
	wantResource := 24
	xShift := 0
	if rawSide == 0 {
		wantResource = 25
		xShift = 143
	}
	if schedule.EffectResource != wantResource {
		return NativeCommand5Frame{}, fmt.Errorf("figani: command5 side mismatch")
	}

	next := state
	frame := NativeCommand5Frame{Next: next}
	for channel := 0; channel < NativeCommand5ChannelCount; channel++ {
		counter := next.Counters[channel]
		if counter >= 0 && counter < 6 {
			position := next.Positions[channel]
			if position < 0 || position >= len(schedule.XOffsets) {
				return NativeCommand5Frame{}, fmt.Errorf("figani: command5 position %d unavailable", position)
			}
			effectFrame := next.Phases[channel] + counter
			if effectFrame < 0 || effectFrame >= NativeCommand5EffectFrameCount {
				return NativeCommand5Frame{}, fmt.Errorf("figani: command5 effect frame %d unavailable", effectFrame)
			}
			frame.Layers = append(frame.Layers, NativeCommand5Layer{Channel: channel, Frame: effectFrame, X: schedule.XOffsets[position] + xShift})
			if counter == 0 && channel == 0 {
				frame.PlayPrimary = true
			}
			if counter == 0 && channel == 3 {
				frame.PlaySecondary = true
			}
		}

		next.Counters[channel]++
		if next.Counters[channel] == 2 {
			frame.NumericMarker = true
		}
		if next.Counters[channel] == 7 && !next.Stop {
			next.Positions[channel] = next.NextPosition
			next.NextPosition = (next.NextPosition + 1) % NativeCommand5PositionCount
			next.Counters[channel] = 0
			next.RNG = fdother.NativeRNGStep(next.RNG)
			next.Phases[channel] = int(next.RNG%2) * 6
		}
	}
	frame.Next = next
	return frame, nil
}

func BuildNativeCommand5TargetSequence(state NativeCommand5State, schedule NativeCommand5PresentationSchedule, rawSide byte) ([]NativeCommand5Frame, NativeCommand5State, error) {
	frames := make([]NativeCommand5Frame, 0, NativeCommand5TargetFrames)
	hpStage := 0
	for index := 0; index < NativeCommand5TargetFrames; index++ {
		frame, err := PlanNativeCommand5DrawFrame(state, schedule, rawSide)
		if err != nil {
			return nil, state, err
		}
		if frame.NumericMarker && hpStage < NativeCommand5DamageStages {
			hpStage++
			frame.HPStage = hpStage
		}
		frames = append(frames, frame)
		state = frame.Next
	}
	if hpStage != NativeCommand5DamageStages {
		return nil, state, fmt.Errorf("figani: command5 HP marker count incomplete: %d", hpStage)
	}
	return frames, state, nil
}

func BuildNativeCommand5TransitionSequence(state NativeCommand5State, schedule NativeCommand5PresentationSchedule, rawSide byte) ([]NativeCommand5TransitionFrame, NativeCommand5State, error) {
	direction := -1
	if rawSide == 0 {
		direction = 1
	}
	frames := make([]NativeCommand5TransitionFrame, 0, NativeCommand5TransitionFrames)
	appendFrame := func(useNext bool, step int) error {
		planned, err := PlanNativeCommand5DrawFrame(state, schedule, rawSide)
		if err != nil {
			return err
		}
		frames = append(frames, NativeCommand5TransitionFrame{NativeCommand5Frame: planned, UseNextTarget: useNext, TargetOffsetX: direction * 35 * step})
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

func BuildNativeCommand5TailSequence(state NativeCommand5State, schedule NativeCommand5PresentationSchedule, rawSide byte) ([]NativeCommand5Frame, NativeCommand5State, error) {
	state.Stop = true
	frames := make([]NativeCommand5Frame, 0, NativeCommand5TailFrames)
	for index := 0; index < NativeCommand5TailFrames; index++ {
		frame, err := PlanNativeCommand5DrawFrame(state, schedule, rawSide)
		if err != nil {
			return nil, state, err
		}
		frames = append(frames, frame)
		state = frame.Next
	}
	return frames, state, nil
}
