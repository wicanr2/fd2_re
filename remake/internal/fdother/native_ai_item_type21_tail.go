package fdother

import (
	"errors"
	"fmt"
)

const (
	NativeAIItemType21EffectResource = 6
	NativeAIItemType21ToggleA        = 0x4a
	NativeAIItemType21ToggleB        = 0x4b
	NativeAIItemType21TogglePairs    = 4
	NativeAIItemType21ToggleDelayMS  = 90
	NativeAIItemType21SoundResource  = 80
	NativeAIItemType21DigitResource  = 5
	NativeAIItemType21DigitBias      = 0x5e
	NativeAIItemType21DigitFrames    = 22
	NativeAIItemType21DigitDelayMS   = 2
	NativeAIItemType21DigitHoldMS    = 500
)

// NativeAIItemType21TailSchedule preserves type21's caller-specific
// 0x1c4cc -> 0x1cac7 -> result queue contract.
type NativeAIItemType21TailSchedule struct {
	CommandID                                     int
	EffectResource, EffectStart, EffectFrames     int
	SoundResource, EffectSample, EffectDelayTicks int
	ToggleA, ToggleB, TogglePairs, ToggleDelayMS  int
	DigitResource, DigitBias, DigitFrames         int
	DigitDelayMS, DigitHoldMS                     int
	DigitVertical                                 [25]int
}

func BuildNativeAIItemType21TailSchedule(commandID int, effect, digits []LMI1Entry) (NativeAIItemType21TailSchedule, error) {
	start, frames, sample := 0, 0, 0
	switch commandID {
	case 1:
		start, frames, sample = 0x31, 8, 6
	case 6, 7:
		start, frames, sample = 0x40, 10, 9
	default:
		return NativeAIItemType21TailSchedule{}, fmt.Errorf("native AI item type21 tail: unsupported command %d", commandID)
	}
	if len(effect) <= NativeAIItemType21ToggleB || start+frames > len(effect) {
		return NativeAIItemType21TailSchedule{}, errors.New("native AI item type21 effect descriptors unavailable")
	}
	for index := start; index < start+frames; index++ {
		entry := effect[index]
		if entry.Width <= 0 || entry.Height <= 0 || len(entry.Pixels) != entry.Width*entry.Height {
			return NativeAIItemType21TailSchedule{}, fmt.Errorf("native AI item type21 effect descriptor %#x invalid", index)
		}
	}
	for _, index := range []int{NativeAIItemType21ToggleA, NativeAIItemType21ToggleB} {
		entry := effect[index]
		if entry.Width <= 0 || entry.Height <= 0 || len(entry.Pixels) != entry.Width*entry.Height {
			return NativeAIItemType21TailSchedule{}, fmt.Errorf("native AI item type21 toggle descriptor %#x invalid", index)
		}
	}
	if len(digits) <= NativeAIItemType21DigitBias+9 {
		return NativeAIItemType21TailSchedule{}, errors.New("native AI item type21 digit descriptors unavailable")
	}
	return NativeAIItemType21TailSchedule{
		CommandID: commandID, EffectResource: NativeAIItemType21EffectResource,
		EffectStart: start, EffectFrames: frames, SoundResource: NativeAIItemType21SoundResource,
		EffectSample: sample, EffectDelayTicks: 1,
		ToggleA: NativeAIItemType21ToggleA, ToggleB: NativeAIItemType21ToggleB,
		TogglePairs: NativeAIItemType21TogglePairs, ToggleDelayMS: NativeAIItemType21ToggleDelayMS,
		DigitResource: NativeAIItemType21DigitResource, DigitBias: NativeAIItemType21DigitBias,
		DigitFrames: NativeAIItemType21DigitFrames, DigitDelayMS: NativeAIItemType21DigitDelayMS,
		DigitHoldMS:   NativeAIItemType21DigitHoldMS,
		DigitVertical: [25]int{15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15},
	}, nil
}
