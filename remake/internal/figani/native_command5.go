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
	CommandID                                 int
	EffectResource                            int
	SoundResource                             int
	SoundIndices                              [2]int
	XOffsets                                  [NativeCommand5PositionCount]int
	EffectFrames, PhaseStride, ActiveCounters int
	FrontFrames, TargetFrames, TailFrames     int
	MarkerCounter, ResetCounter               int
	SampleEveryChannel                        bool
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
		CommandID:      5,
		EffectResource: resource,
		SoundResource:  NativeCommand5SoundResource,
		SoundIndices:   [2]int{0, 1},
		XOffsets:       nativeCommand5XOffsets,
		EffectFrames:   NativeCommand5EffectFrameCount, PhaseStride: 6, ActiveCounters: 6,
		FrontFrames: NativeCommand5FrontFrames, TargetFrames: NativeCommand5TargetFrames,
		TailFrames: NativeCommand5TailFrames, MarkerCounter: 2, ResetCounter: 7,
	}, nil
}

// BuildNativeCommand4PresentationSchedule 保存 0x269D3 已證實的六槽變體。
// 型別沿用相同的六槽狀態容器，但所有會不同的原版常數都留在 schedule，
// 不把第5號資源或 marker 偷渡給第4號。
func BuildNativeCommand4PresentationSchedule(rawSide byte, effect *Animation) (NativeCommand5PresentationSchedule, error) {
	resource := 22
	if rawSide == 0 {
		resource = 23
	}
	if effect == nil || effect.HeaderByte1 != 0 || effect.HeaderByte2 != 14 || effect.HeaderByte4 != 0 || len(effect.Frames) != 14 {
		return NativeCommand5PresentationSchedule{}, fmt.Errorf("figani: command4 FDOTHER #%d signature mismatch", resource)
	}
	for index, frame := range effect.Frames {
		if frame.Width <= 0 || frame.Height <= 0 || len(frame.Pixels) != frame.Width*frame.Height || len(frame.Mask) != len(frame.Pixels) {
			return NativeCommand5PresentationSchedule{}, fmt.Errorf("figani: command4 FDOTHER #%d frame %d malformed", resource, index)
		}
	}
	return NativeCommand5PresentationSchedule{CommandID: 4, EffectResource: resource, SoundResource: 85,
		SoundIndices: [2]int{0, 1}, XOffsets: nativeCommand5XOffsets, EffectFrames: 14,
		PhaseStride: 7, ActiveCounters: 7, FrontFrames: 2, TargetFrames: 12,
		TailFrames: 8, MarkerCounter: 3, ResetCounter: 8, SampleEveryChannel: true}, nil
}

// NewNativeCommand5State 重現 mode0：六個相位各消耗一次 0x4E893，
// 並保留下一個循環位置 6。raw side 只影響畫面水平錨點，不影響狀態。
func NewNativeCommand5StateForSchedule(rng uint16, schedule NativeCommand5PresentationSchedule) NativeCommand5State {
	state := NativeCommand5State{NextPosition: NativeCommand5ChannelCount, RNG: rng}
	for channel := 0; channel < NativeCommand5ChannelCount; channel++ {
		state.Counters[channel] = -2 * channel
		state.Positions[channel] = channel
		state.RNG = fdother.NativeRNGStep(state.RNG)
		state.Phases[channel] = int(state.RNG%2) * schedule.PhaseStride
	}
	return state
}

func NewNativeCommand5State(rng uint16) NativeCommand5State {
	return NewNativeCommand5StateForSchedule(rng, NativeCommand5PresentationSchedule{PhaseStride: 6})
}

// PlanNativeCommand5DrawFrame 重現 mode2／5／8 共用的六通道迴圈。
// mode1／4／7 不畫 handler effect，故不在此函式內表示。
func PlanNativeCommand5DrawFrame(state NativeCommand5State, schedule NativeCommand5PresentationSchedule, rawSide byte) (NativeCommand5Frame, error) {
	if schedule.CommandID != 4 && schedule.CommandID != 5 {
		return NativeCommand5Frame{}, fmt.Errorf("figani: command5 schedule unavailable")
	}
	wantResource := 24
	if schedule.CommandID == 4 {
		wantResource = 22
	}
	xShift := 0
	if rawSide == 0 {
		if schedule.CommandID == 4 {
			wantResource = 23
		} else {
			wantResource = 25
		}
		xShift = 143
	}
	if schedule.EffectResource != wantResource {
		return NativeCommand5Frame{}, fmt.Errorf("figani: command5 side mismatch")
	}

	next := state
	frame := NativeCommand5Frame{Next: next}
	for channel := 0; channel < NativeCommand5ChannelCount; channel++ {
		counter := next.Counters[channel]
		if counter >= 0 && counter < schedule.ActiveCounters {
			position := next.Positions[channel]
			if position < 0 || position >= len(schedule.XOffsets) {
				return NativeCommand5Frame{}, fmt.Errorf("figani: command5 position %d unavailable", position)
			}
			effectFrame := next.Phases[channel] + counter
			if effectFrame < 0 || effectFrame >= schedule.EffectFrames {
				return NativeCommand5Frame{}, fmt.Errorf("figani: command5 effect frame %d unavailable", effectFrame)
			}
			frame.Layers = append(frame.Layers, NativeCommand5Layer{Channel: channel, Frame: effectFrame, X: schedule.XOffsets[position] + xShift})
			if counter == 0 && (schedule.SampleEveryChannel || channel == 0) {
				frame.PlayPrimary = true
			}
			if counter == 0 && !schedule.SampleEveryChannel && channel == 3 {
				frame.PlaySecondary = true
			}
		}

		next.Counters[channel]++
		if next.Counters[channel] == schedule.MarkerCounter {
			frame.NumericMarker = true
		}
		if next.Counters[channel] == schedule.ResetCounter && !next.Stop {
			next.Positions[channel] = next.NextPosition
			next.NextPosition = (next.NextPosition + 1) % NativeCommand5PositionCount
			next.Counters[channel] = 0
			next.RNG = fdother.NativeRNGStep(next.RNG)
			next.Phases[channel] = int(next.RNG%2) * schedule.PhaseStride
		}
	}
	frame.Next = next
	return frame, nil
}

func BuildNativeCommand5TargetSequence(state NativeCommand5State, schedule NativeCommand5PresentationSchedule, rawSide byte) ([]NativeCommand5Frame, NativeCommand5State, error) {
	return BuildNativeCommand5TargetSequenceHit(state, schedule, rawSide, true)
}

// BuildNativeCommand5TargetSequenceHit 是目標畫格序列加上 0x2A6BD 的命中分支：
// 0x1C75E 回 0（未命中）時 0x2ADFE 走 0x2AF61，mode 5 照樣呼叫（六槽重置照樣吃
// 亂數）但不看回傳值，沒有 HP 分段也沒有 0x2AF40 抖動；命中才走 0x2AEB6 的
// marker 分支（r4 收據 seq 1013：指令 0 未命中只吃 0x1C7F2 一步）。
func BuildNativeCommand5TargetSequenceHit(state NativeCommand5State, schedule NativeCommand5PresentationSchedule, rawSide byte, hit bool) ([]NativeCommand5Frame, NativeCommand5State, error) {
	frames := make([]NativeCommand5Frame, 0, schedule.TargetFrames)
	hpStage := 0
	for index := 0; index < schedule.TargetFrames; index++ {
		frame, err := PlanNativeCommand5DrawFrame(state, schedule, rawSide)
		if err != nil {
			return nil, state, err
		}
		if hit {
			if frame.NumericMarker && hpStage < NativeCommand5DamageStages {
				hpStage++
				frame.HPStage = hpStage
			}
			// 0x2AF40：mode 5 回傳 1 的每一張目標畫格都在 handler 之後再吃一次亂數。
			frame = NativeCommand5MarkerShake(frame)
		}
		frames = append(frames, frame)
		state = frame.Next
	}
	if hit && hpStage != NativeCommand5DamageStages {
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
	frames := make([]NativeCommand5Frame, 0, schedule.TailFrames)
	for index := 0; index < schedule.TailFrames; index++ {
		frame, err := PlanNativeCommand5DrawFrame(state, schedule, rawSide)
		if err != nil {
			return nil, state, err
		}
		frames = append(frames, frame)
		state = frame.Next
	}
	return frames, state, nil
}

// NativeCommand5MarkerShake 重現 0x2A6BD 目標迴圈在 mode 5 回傳 1 之後的
// `0x2AF40 call 0x4E893`：每個回傳 numeric marker 的目標畫格都再吃一次全域
// 亂數（餘數只用於畫面抖動位移 `1 - rng%3`）。前段 mode 2 與尾段 mode 8 的
// 呼叫端（0x2AC8E／0x2B1B6）不看回傳值，所以那兩段沒有這一步。
func NativeCommand5MarkerShake(frame NativeCommand5Frame) NativeCommand5Frame {
	if frame.NumericMarker {
		frame.Next.RNG = fdother.NativeRNGStep(frame.Next.RNG)
	}
	return frame
}

// WalkNativeCommand5RNG 只走 0x269D3 六槽狀態機與 0x2A6BD 目標迴圈的亂數
// 序列，不合成任何畫格：mode 0 初始化（六次）→ 前段畫格（可能的重置）→
// 每個目標：`resolve` 在 mode 3 之後、第一張目標畫格之前擲命中／傷害
// （0x2B114 call 0x1C75E），接著目標畫格裡 mode 5 的重置與 marker 抖動，
// 目標之間是過場畫格 → 尾段畫格（stop=1，沒有重置與抖動）。回傳整段結束時
// 的亂數，與 BuildNativeCommand5EffectSequence 走同一條序列（r4 收據
// seq 1451：6 次 0x26A5A、0x1C7F2、0x1C86E、0x2AF45／0x26BCD 交錯共 18 步）。
func WalkNativeCommand5RNG(rng uint16, schedule NativeCommand5PresentationSchedule, rawSide byte, targetCount int, resolve func(index int, rng uint16) (uint16, bool, error)) (uint16, error) {
	if targetCount <= 0 || resolve == nil {
		return rng, fmt.Errorf("figani: command5 RNG walk needs targets and a resolver")
	}
	state := NewNativeCommand5StateForSchedule(rng, schedule)
	for index := 0; index < schedule.FrontFrames; index++ {
		frame, err := PlanNativeCommand5DrawFrame(state, schedule, rawSide)
		if err != nil {
			return rng, err
		}
		state = frame.Next
	}
	for target := 0; target < targetCount; target++ {
		next, hit, err := resolve(target, state.RNG)
		if err != nil {
			return rng, err
		}
		state.RNG = next
		_, after, err := BuildNativeCommand5TargetSequenceHit(state, schedule, rawSide, hit)
		if err != nil {
			return rng, err
		}
		state = after
		if target+1 < targetCount {
			_, after, err := BuildNativeCommand5TransitionSequence(state, schedule, rawSide)
			if err != nil {
				return rng, err
			}
			state = after
		}
	}
	_, final, err := BuildNativeCommand5TailSequence(state, schedule, rawSide)
	if err != nil {
		return rng, err
	}
	return final.RNG, nil
}
