package fdother

import (
	"errors"
	"fmt"
)

const (
	NativeCommand32TailEffectResource = 6
	NativeCommand32TailEffectStart    = 0x40
	NativeCommand32TailEffectFrames   = 10
	NativeCommand32TailToggleA        = 0x4a
	NativeCommand32TailToggleB        = 0x4b
	NativeCommand32TailTogglePairs    = 4
	NativeCommand32TailToggleDelayMS  = 90
	NativeCommand32TailSoundResource  = 80
	NativeCommand32TailEffectSample   = 9
	NativeCommand32TailDigitResource  = 5
	NativeCommand32TailDigitBias      = 0x5e
	NativeCommand32TailDigitFrames    = 22
	NativeCommand32TailDigitDelayMS   = 2
	NativeCommand32TailDigitHoldMS    = 500
)

// NativeCommand32TailSchedule preserves the exact 0x2111A caller-specific
// 0x1C4CC -> 0x1CAC7 -> numeric-result contract. Descriptor values remain raw.
type NativeCommand32TailSchedule struct {
	EffectResource, EffectStart, EffectFrames     int
	SoundResource, EffectSample, EffectDelayTicks int
	ToggleA, ToggleB, TogglePairs, ToggleDelayMS  int
	DigitResource, DigitBias, DigitFrames         int
	DigitDelayMS, DigitHoldMS                     int
	DigitVertical                                 [25]int
}

func BuildNativeCommand32TailSchedule(effect, digits []LMI1Entry) (NativeCommand32TailSchedule, error) {
	if len(effect) <= NativeCommand32TailToggleB {
		return NativeCommand32TailSchedule{}, errors.New("native command32 tail effect descriptors unavailable")
	}
	for index := NativeCommand32TailEffectStart; index <= NativeCommand32TailToggleB; index++ {
		entry := effect[index]
		if entry.Width <= 0 || entry.Height <= 0 || len(entry.Pixels) != entry.Width*entry.Height {
			return NativeCommand32TailSchedule{}, fmt.Errorf("native command32 tail effect descriptor %#x invalid", index)
		}
	}
	if len(digits) <= 0x76 {
		return NativeCommand32TailSchedule{}, errors.New("native command32 tail digit descriptors unavailable")
	}
	return NativeCommand32TailSchedule{
		EffectResource: NativeCommand32TailEffectResource, EffectStart: NativeCommand32TailEffectStart,
		EffectFrames: NativeCommand32TailEffectFrames, SoundResource: NativeCommand32TailSoundResource,
		EffectSample: NativeCommand32TailEffectSample, EffectDelayTicks: 1,
		ToggleA: NativeCommand32TailToggleA, ToggleB: NativeCommand32TailToggleB,
		TogglePairs: NativeCommand32TailTogglePairs, ToggleDelayMS: NativeCommand32TailToggleDelayMS,
		DigitResource: NativeCommand32TailDigitResource, DigitBias: NativeCommand32TailDigitBias,
		DigitFrames: NativeCommand32TailDigitFrames, DigitDelayMS: NativeCommand32TailDigitDelayMS,
		DigitHoldMS:   NativeCommand32TailDigitHoldMS,
		DigitVertical: [25]int{15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15},
	}, nil
}
