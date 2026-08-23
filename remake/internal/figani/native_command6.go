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
