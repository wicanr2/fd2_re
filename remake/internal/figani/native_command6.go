package figani

import (
	"fmt"
	"math"
)

const (
	NativeCommand6EffectFrameCount = 10
	NativeCommand6ChannelCount     = 5
	NativeCommand6SoundResource    = 87
	NativeCommand6TargetFrames     = 12
	NativeCommand6DamageStages     = 5
)

var nativeCommand6DwordTable = [NativeCommand6ChannelCount]int{10, 8, 3, 0, 0}
var nativeCommand6ByteTable = [NativeCommand6ChannelCount]byte{10, 8, 3, 0, 0}

type NativeCommand6Point struct {
	X, Y int
}

type NativeCommand6TargetState struct {
	Counters  [NativeCommand6ChannelCount]int
	Secondary [NativeCommand6ChannelCount]byte
}

type NativeCommand6Layer struct {
	Mode, Channel, Frame int
	X, Y                 int
	Secondary            bool
}

type NativeCommand6TargetFrame struct {
	Mode4, Mode5  []NativeCommand6Layer
	Next          NativeCommand6TargetState
	NumericMarker bool
	HPStage       int
}

type NativeCommand6OrbitFrame struct {
	First, Second []NativeCommand6Layer
	NextRadius    int
	DrawTarget    bool
}

type NativeCommand6TransitionFrame struct {
	Mode4, Mode5  []NativeCommand6Layer
	UseNextTarget bool
	TargetOffsetX int
	Next          NativeCommand6TargetState
}

// NativeCommand6PresentationSchedule 保存 0x26E39 已由直接指令閉合的
// 區域表格與資源邊界。浮點座標和 mode4/5 compositor 不在此型別中猜測。
type NativeCommand6PresentationSchedule struct {
	EffectResource int
	SoundResource  int
	SoundIndices   [3]int
	BaseByte       byte
	DwordTable     [NativeCommand6ChannelCount]int
	ByteTable      [NativeCommand6ChannelCount]byte
}

// NativeCommand6Coordinates reproduces 0x26F99..0x2704C. The degree
// conversion remains the original single-precision constant; fistp uses the
// default nearest-even rounding mode.
func NativeCommand6Coordinates(radius int, baseByte byte) [NativeCommand6ChannelCount]NativeCommand6Point {
	const degree = float64(float32(0.017453292519943295))
	var points [NativeCommand6ChannelCount]NativeCommand6Point
	for channel := range points {
		angle := float64(channel*72) * degree
		points[channel] = NativeCommand6Point{
			X: int(math.RoundToEven(math.Cos(angle)*float64(radius) + float64(baseByte))),
			Y: int(math.RoundToEven(math.Sin(angle)*float64(radius)*1.2 + 30)),
		}
	}
	return points
}

func nativeCommand6OrbitLayers(mode, radius int, rawSide byte, schedule NativeCommand6PresentationSchedule) ([]NativeCommand6Layer, error) {
	if schedule.EffectResource != 32 && schedule.EffectResource != 33 {
		return nil, fmt.Errorf("figani: command6 orbit schedule unavailable")
	}
	points := NativeCommand6Coordinates(radius, schedule.BaseByte)
	layers := make([]NativeCommand6Layer, 0, NativeCommand6ChannelCount)
	for channel, point := range points {
		draw := false
		if rawSide == 0 {
			draw = mode == 2 || mode == 8
		} else if mode == 1 || mode == 7 {
			draw = channel < 2
		} else if mode == 2 || mode == 8 {
			draw = channel > 1
		}
		if draw {
			layers = append(layers, NativeCommand6Layer{Mode: mode, Channel: channel, Frame: 4, X: point.X, Y: point.Y})
		}
	}
	return layers, nil
}

func PlanNativeCommand6PreludeFrame(radius int, rawSide byte, schedule NativeCommand6PresentationSchedule) (NativeCommand6OrbitFrame, error) {
	first, err := nativeCommand6OrbitLayers(1, radius, rawSide, schedule)
	if err != nil {
		return NativeCommand6OrbitFrame{}, err
	}
	second, err := nativeCommand6OrbitLayers(2, radius, rawSide, schedule)
	if err != nil {
		return NativeCommand6OrbitFrame{}, err
	}
	return NativeCommand6OrbitFrame{First: first, Second: second, NextRadius: radius + 6}, nil
}

func PlanNativeCommand6TailFrame(radius int, rawSide byte, schedule NativeCommand6PresentationSchedule) (NativeCommand6OrbitFrame, error) {
	first, err := nativeCommand6OrbitLayers(7, radius, rawSide, schedule)
	if err != nil {
		return NativeCommand6OrbitFrame{}, err
	}
	next := radius - 6
	second, err := nativeCommand6OrbitLayers(8, radius, rawSide, schedule)
	if err != nil {
		return NativeCommand6OrbitFrame{}, err
	}
	return NativeCommand6OrbitFrame{First: first, Second: second, NextRadius: next, DrawTarget: true}, nil
}

func NewNativeCommand6TargetState() NativeCommand6TargetState {
	var state NativeCommand6TargetState
	for channel := range state.Counters {
		state.Counters[channel] = -channel
	}
	return state
}

// PlanNativeCommand6TargetFrame reproduces one mode4→target→mode5 iteration.
// The returned layers contain no sound or numeric publication semantics.
func PlanNativeCommand6TargetFrame(state NativeCommand6TargetState, schedule NativeCommand6PresentationSchedule, points [NativeCommand6ChannelCount]NativeCommand6Point, rawSide byte) (NativeCommand6TargetFrame, error) {
	if schedule.EffectResource != 32 && schedule.EffectResource != 33 {
		return NativeCommand6TargetFrame{}, fmt.Errorf("figani: command6 target schedule unavailable")
	}
	wantResource := 32
	if rawSide == 0 {
		wantResource = 33
	}
	if schedule.EffectResource != wantResource {
		return NativeCommand6TargetFrame{}, fmt.Errorf("figani: command6 target side mismatch")
	}
	next := state
	mainLayers := func(mode int) ([]NativeCommand6Layer, error) {
		layers := make([]NativeCommand6Layer, 0, NativeCommand6ChannelCount)
		for channel, counter := range next.Counters {
			frame := counter
			if frame < 0 {
				frame = 4
			}
			if frame < 0 || frame >= NativeCommand6EffectFrameCount {
				return nil, fmt.Errorf("figani: command6 main frame %d unavailable", frame)
			}
			if frame == 1 {
				next.Secondary[channel] = 5
			}
			draw := rawSide == 0 && mode == 5
			if rawSide != 0 && channel < 2 {
				draw = true
			}
			if draw {
				layers = append(layers, NativeCommand6Layer{
					Mode: mode, Channel: channel, Frame: frame,
					X: points[channel].X + schedule.DwordTable[frame],
					Y: points[channel].Y + int(schedule.ByteTable[frame]),
				})
			}
		}
		return layers, nil
	}
	mode4, err := mainLayers(4)
	if err != nil {
		return NativeCommand6TargetFrame{}, err
	}
	mode5, err := mainLayers(5)
	if err != nil {
		return NativeCommand6TargetFrame{}, err
	}
	numericMarker := false
	for channel := range next.Counters {
		next.Counters[channel]++
		if next.Counters[channel] == 5 {
			next.Counters[channel] = 0
		}
		if next.Counters[channel] == 2 {
			numericMarker = true
		}
		if secondary := next.Secondary[channel]; secondary != 0 {
			if secondary >= NativeCommand6EffectFrameCount {
				return NativeCommand6TargetFrame{}, fmt.Errorf("figani: command6 secondary frame %d unavailable", secondary)
			}
			mode5 = append(mode5, NativeCommand6Layer{
				Mode: 5, Channel: channel, Frame: int(secondary), Secondary: true,
				X: points[channel].X - 60, Y: points[channel].Y - 20,
			})
			next.Secondary[channel]++
			if next.Secondary[channel] == 10 {
				next.Secondary[channel] = 0
			}
		}
	}
	return NativeCommand6TargetFrame{Mode4: mode4, Mode5: mode5, Next: next, NumericMarker: numericMarker}, nil
}

// BuildNativeCommand6TargetSequence binds the twelve mode3 frames to the
// outer 0x2A6BD consumer. All eleven raw markers remain visible, but only the
// first five publish HP because 0x525AF[6] is five.
func BuildNativeCommand6TargetSequence(schedule NativeCommand6PresentationSchedule, points [NativeCommand6ChannelCount]NativeCommand6Point, rawSide byte) ([]NativeCommand6TargetFrame, error) {
	state := NewNativeCommand6TargetState()
	frames := make([]NativeCommand6TargetFrame, 0, NativeCommand6TargetFrames)
	hpStage := 0
	for index := 0; index < NativeCommand6TargetFrames; index++ {
		frame, err := PlanNativeCommand6TargetFrame(state, schedule, points, rawSide)
		if err != nil {
			return nil, err
		}
		if frame.NumericMarker && hpStage < NativeCommand6DamageStages {
			hpStage++
			frame.HPStage = hpStage
		}
		frames = append(frames, frame)
		state = frame.Next
	}
	if hpStage != NativeCommand6DamageStages {
		return nil, fmt.Errorf("figani: command6 HP marker count incomplete: %d", hpStage)
	}
	return frames, nil
}

// BuildNativeCommand6TransitionSequence reproduces the command6 branch of
// sub_2BA22. It preserves the target-effect counter state left by the current
// target while generating four outgoing and five incoming frames.
func BuildNativeCommand6TransitionSequence(state NativeCommand6TargetState, schedule NativeCommand6PresentationSchedule, points [NativeCommand6ChannelCount]NativeCommand6Point, rawSide byte) ([]NativeCommand6TransitionFrame, error) {
	direction := -1
	if rawSide == 0 {
		direction = 1
	}
	frames := make([]NativeCommand6TransitionFrame, 0, 9)
	appendFrame := func(useNext bool, step int) error {
		planned, err := PlanNativeCommand6TargetFrame(state, schedule, points, rawSide)
		if err != nil {
			return err
		}
		frames = append(frames, NativeCommand6TransitionFrame{
			Mode4: planned.Mode4, Mode5: planned.Mode5, UseNextTarget: useNext,
			TargetOffsetX: direction * 35 * step, Next: planned.Next,
		})
		state = planned.Next
		return nil
	}
	for step := 1; step <= 4; step++ {
		if err := appendFrame(false, step); err != nil {
			return nil, err
		}
	}
	for step := 4; step >= 0; step-- {
		if err := appendFrame(true, step); err != nil {
			return nil, err
		}
	}
	return frames, nil
}

func BuildNativeCommand6PresentationSchedule(rawSide byte, effect *Animation) (NativeCommand6PresentationSchedule, error) {
	resource, base := 32, byte(30)
	dwords, bytes := nativeCommand6DwordTable, nativeCommand6ByteTable
	if rawSide == 0 {
		resource, base = 33, 90
		for index := 0; index < 3; index++ {
			dwords[index] = -dwords[index]
			bytes[index] = 0
		}
	}
	if effect == nil || effect.HeaderByte1 != 0 || effect.HeaderByte2 != NativeCommand6EffectFrameCount ||
		effect.HeaderByte4 != 0 || len(effect.Frames) != NativeCommand6EffectFrameCount {
		return NativeCommand6PresentationSchedule{}, fmt.Errorf("figani: command6 FDOTHER #%d signature mismatch", resource)
	}
	for index, frame := range effect.Frames {
		if frame.Width <= 0 || frame.Height <= 0 || len(frame.Pixels) != frame.Width*frame.Height || len(frame.Mask) != len(frame.Pixels) {
			return NativeCommand6PresentationSchedule{}, fmt.Errorf("figani: command6 FDOTHER #%d frame %d malformed", resource, index)
		}
	}
	return NativeCommand6PresentationSchedule{
		EffectResource: resource,
		SoundResource:  NativeCommand6SoundResource,
		SoundIndices:   [3]int{1, 2, 3},
		BaseByte:       base,
		DwordTable:     dwords,
		ByteTable:      bytes,
	}, nil
}
