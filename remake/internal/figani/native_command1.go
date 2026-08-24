package figani

import "fmt"

const (
	NativeCommand1EffectFrameCount   = 30
	NativeCommand1ChannelCount       = 8
	NativeCommand1AnchorX            = 0x50
	NativeCommand1PresentationFrames = 31
)

var nativeCommand1XOffsets = [NativeCommand1ChannelCount]int{-59, -39, 0, 39, 55, 39, 0, -39}
var nativeCommand1YOffsets = [NativeCommand1ChannelCount]int{-10, -24, -30, -24, -10, 4, 10, 4}

// NativeCommand1PresentationSchedule 保存 0x262EF 的 mode 3/4/5 純繪製契約。
// 直接 sample1 與 numeric marker 由 NativeCommand1TargetMarkers 分開保存。
type NativeCommand1PresentationSchedule struct {
	EffectResource int
	SideXShift     int
	AnchorX        int
	XOffsets       [NativeCommand1ChannelCount]int
	YOffsets       [NativeCommand1ChannelCount]int
}

func BuildNativeCommand1PresentationSchedule(rawSide byte, effect *Animation) (NativeCommand1PresentationSchedule, error) {
	resource, shift := 19, 0
	if rawSide == 0 {
		resource, shift = 21, 148
	}
	if effect == nil || effect.HeaderByte1 != 0 || effect.HeaderByte2 != NativeCommand1EffectFrameCount ||
		effect.HeaderByte4 != 0 || len(effect.Frames) != NativeCommand1EffectFrameCount {
		return NativeCommand1PresentationSchedule{}, fmt.Errorf("figani: command1 FDOTHER #%d signature mismatch", resource)
	}
	for index, frame := range effect.Frames {
		if frame.Width <= 0 || frame.Height <= 0 || len(frame.Pixels) != frame.Width*frame.Height ||
			len(frame.Mask) != len(frame.Pixels) {
			return NativeCommand1PresentationSchedule{}, fmt.Errorf("figani: command1 FDOTHER #%d frame %d malformed", resource, index)
		}
	}
	return NativeCommand1PresentationSchedule{
		EffectResource: resource,
		SideXShift:     shift,
		AnchorX:        NativeCommand1AnchorX,
		XOffsets:       nativeCommand1XOffsets,
		YOffsets:       nativeCommand1YOffsets,
	}, nil
}

// NativeCommand1Counters 回傳 mode 3 寫入的八個 staggered counters。
func NativeCommand1Counters(step int) ([NativeCommand1ChannelCount]int, error) {
	if step < 0 || step >= NativeCommand1PresentationFrames {
		return [NativeCommand1ChannelCount]int{}, fmt.Errorf("figani: command1 step %d is unavailable", step)
	}
	var counters [NativeCommand1ChannelCount]int
	for slot := range counters {
		counters[slot] = step - 2*slot
	}
	return counters, nil
}

// NativeCommand1ModeFrame 保存 mode 4/5 的兩組 frame/index 交換規則。
// ok=false 表示 counter 不在原版的 [0,15) 繪製範圍。
func NativeCommand1ModeFrame(mode, slot, counter int) (frame, offsetIndex int, ok bool) {
	if slot < 0 || slot >= NativeCommand1ChannelCount || counter < 0 || counter >= 15 {
		return 0, 0, false
	}
	switch mode {
	case 4:
		if slot < 4 {
			return counter, slot, true
		}
		return counter + 15, slot - 4, true
	case 5:
		if slot < 4 {
			return counter + 15, slot + 4, true
		}
		return counter, slot, true
	default:
		return 0, 0, false
	}
}

// NativeCommand1TargetMarkers 保存 mode 5 遞增八個 staggered counter 後的
// 直接消費端：等於9時回傳 numeric marker，等於5時播放 sample1。
func NativeCommand1TargetMarkers(step int) (numeric, sample bool, err error) {
	counters, err := NativeCommand1Counters(step)
	if err != nil {
		return false, false, err
	}
	for _, counter := range counters {
		if counter+1 == 9 {
			numeric = true
		}
		if counter+1 == 5 {
			sample = true
		}
	}
	return numeric, sample, nil
}
