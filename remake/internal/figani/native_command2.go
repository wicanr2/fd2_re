package figani

import "fmt"

const (
	NativeCommand2EffectFrameCount = 18
	NativeCommand2FrontFrames      = 29
	NativeCommand2TargetFrames     = 12
	NativeCommand2DamageStages     = 6
	NativeCommand2TransitionFrames = 9
	NativeCommand2TailFrames       = 10
	NativeCommand2SoundResource    = 83
	NativeCommand2SoundMode2Index  = 1
	NativeCommand2SoundMode5Index  = 2
	NativeCommand2SoundMode6Index  = 3
)

type NativeCommand2HelperState struct {
	Frame, Repeat byte
}

type NativeCommand2HelperFrame struct {
	EffectFrame   int
	Next          NativeCommand2HelperState
	Sample1       bool
	Sample2       bool
	NumericMarker bool
	HPStage       int
}

// NativeCommand2PresentationSchedule 保存 0x26528 已閉合的資源契約；
// 各 phase 的 frame/repeat 排程由同檔 typed builder 建立。
type NativeCommand2PresentationSchedule struct {
	EffectResource int
	SoundResource  int
	SoundIndices   [3]int
}

func advanceNativeCommand2Helper(state NativeCommand2HelperState, effect *Animation) (NativeCommand2HelperFrame, error) {
	if effect == nil || int(state.Frame) >= len(effect.Frames) || effect.Frames[state.Frame].Delay <= 0 {
		return NativeCommand2HelperFrame{}, fmt.Errorf("figani: command2 helper frame %d unavailable", state.Frame)
	}
	next := state
	next.Repeat++
	if int(next.Repeat) >= effect.Frames[state.Frame].Delay {
		next.Repeat = 0
		next.Frame++
	}
	return NativeCommand2HelperFrame{EffectFrame: int(state.Frame), Next: next}, nil
}

func BuildNativeCommand2FrontSequence(rawSide byte, effect *Animation) ([]NativeCommand2HelperFrame, NativeCommand2HelperState, error) {
	state := NativeCommand2HelperState{}
	frames := make([]NativeCommand2HelperFrame, 0, NativeCommand2FrontFrames)
	for step := 0; step < NativeCommand2FrontFrames; step++ {
		if state.Frame == 10 {
			state.Frame = 15
		}
		frame, err := advanceNativeCommand2Helper(state, effect)
		if err != nil {
			return nil, NativeCommand2HelperState{}, err
		}
		frame.Sample1 = rawSide == 0 && state.Frame == 7
		frames = append(frames, frame)
		state = frame.Next
	}
	return frames, state, nil
}

func planNativeCommand2TargetFrame(state NativeCommand2HelperState) NativeCommand2HelperFrame {
	frame := NativeCommand2HelperFrame{EffectFrame: int(state.Frame)}
	next := state
	next.Frame++
	if next.Frame == 17 {
		frame.Sample2, frame.NumericMarker = true, true
	} else if next.Frame == 18 {
		next.Frame = 16
	}
	frame.Next = next
	return frame
}

func BuildNativeCommand2TargetSequence() ([]NativeCommand2HelperFrame, NativeCommand2HelperState, error) {
	state := NativeCommand2HelperState{Frame: 16}
	frames := make([]NativeCommand2HelperFrame, 0, NativeCommand2TargetFrames)
	hpStage := 0
	for step := 0; step < NativeCommand2TargetFrames; step++ {
		frame := planNativeCommand2TargetFrame(state)
		if frame.NumericMarker {
			hpStage++
			frame.HPStage = hpStage
		}
		frames = append(frames, frame)
		state = frame.Next
	}
	if hpStage != NativeCommand2DamageStages {
		return nil, NativeCommand2HelperState{}, fmt.Errorf("figani: command2 HP marker count=%d", hpStage)
	}
	return frames, state, nil
}

func BuildNativeCommand2TransitionSequence(state NativeCommand2HelperState) []NativeCommand2HelperFrame {
	frames := make([]NativeCommand2HelperFrame, 0, NativeCommand2TransitionFrames)
	for step := 0; step < NativeCommand2TransitionFrames; step++ {
		frame := planNativeCommand2TargetFrame(state)
		frame.NumericMarker, frame.HPStage = false, 0
		frames = append(frames, frame)
		state = frame.Next
	}
	return frames
}

func BuildNativeCommand2TailSequence(rawSide byte, repeat byte, effect *Animation) ([]NativeCommand2HelperFrame, error) {
	state := NativeCommand2HelperState{Frame: 10, Repeat: repeat}
	frames := make([]NativeCommand2HelperFrame, 0, NativeCommand2TailFrames)
	for step := 0; step < NativeCommand2TailFrames; step++ {
		frame, err := advanceNativeCommand2Helper(state, effect)
		if err != nil {
			return nil, err
		}
		frame.Sample1 = rawSide == 0 && state.Frame == 7
		frames = append(frames, frame)
		state = frame.Next
	}
	return frames, nil
}

func BuildNativeCommand2PresentationSchedule(rawSide byte, effect *Animation) (NativeCommand2PresentationSchedule, error) {
	resource := 26
	if rawSide == 0 {
		resource = 27
	}
	if effect == nil || effect.HeaderByte1 != 0 || effect.HeaderByte2 != NativeCommand2EffectFrameCount ||
		effect.HeaderByte4 != 0 || len(effect.Frames) != NativeCommand2EffectFrameCount {
		return NativeCommand2PresentationSchedule{}, fmt.Errorf("figani: command2 FDOTHER #%d signature mismatch", resource)
	}
	for index, frame := range effect.Frames {
		if frame.Width <= 0 || frame.Height <= 0 || len(frame.Pixels) != frame.Width*frame.Height || len(frame.Mask) != len(frame.Pixels) {
			return NativeCommand2PresentationSchedule{}, fmt.Errorf("figani: command2 FDOTHER #%d frame %d malformed", resource, index)
		}
	}
	return NativeCommand2PresentationSchedule{
		EffectResource: resource,
		SoundResource:  NativeCommand2SoundResource,
		SoundIndices:   [3]int{NativeCommand2SoundMode2Index, NativeCommand2SoundMode5Index, NativeCommand2SoundMode6Index},
	}, nil
}
