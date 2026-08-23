package figani

import (
	"fmt"
	"math"
)

const (
	NativeCommand6EffectFrameCount = 10
	NativeCommand6ChannelCount     = 5
	NativeCommand6SoundResource    = 87
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
	Mode4, Mode5 []NativeCommand6Layer
	Next         NativeCommand6TargetState
	Complete     bool
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
	complete := false
	for channel := range next.Counters {
		next.Counters[channel]++
		if next.Counters[channel] == 5 {
			next.Counters[channel] = 0
		}
		if next.Counters[channel] == 2 {
			complete = true
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
	return NativeCommand6TargetFrame{Mode4: mode4, Mode5: mode5, Next: next, Complete: complete}, nil
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
