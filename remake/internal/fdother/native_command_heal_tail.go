package fdother

import "fmt"

const (
	NativeCommandHealTailEffectResource = 6
	NativeCommandHealTailDigitResource  = 5
	NativeCommandHealTailEffectStart    = 0x39
	NativeCommandHealTailEffectFrames   = 7
	NativeCommandHealTailEffectSample   = 12
	NativeCommandHealTailMaskSample     = 1
	NativeCommandHealTailMaskIndex      = 0xc0
	NativeCommandHealTailMaskPairs      = 5
	NativeCommandHealTailDigitBias      = 0x69
	NativeCommandHealTailDigitFrames    = 22
	NativeCommandHealTailDigitDelayMs   = 2
	NativeCommandHealTailDigitHoldMs    = 500
)

// NativeCommandHealTailSchedule is the typed, caller-specific contract for
// 0x1c4cc -> 0x1c2da -> 0x1e0db/0x1df58. Raw palette and descriptor values
// deliberately remain unnamed.
type NativeCommandHealTailSchedule struct {
	EffectResource, EffectStart, EffectFrames int
	EffectSample, EffectFrameDelayTicks       int
	MaskSample, MaskIndex, MaskPairs          int
	DigitResource, DigitBias, DigitFrames     int
	DigitFrameDelayMs, DigitHoldMs            int
	DigitVertical                             [25]int
}

func BuildNativeCommandHealTailSchedule(commandID int) (NativeCommandHealTailSchedule, error) {
	if commandID < 13 || commandID > 16 {
		return NativeCommandHealTailSchedule{}, fmt.Errorf("native command heal tail: unsupported id %d", commandID)
	}
	return NativeCommandHealTailSchedule{
		EffectResource:        NativeCommandHealTailEffectResource,
		EffectStart:           NativeCommandHealTailEffectStart,
		EffectFrames:          NativeCommandHealTailEffectFrames,
		EffectSample:          NativeCommandHealTailEffectSample,
		EffectFrameDelayTicks: 1,
		MaskSample:            NativeCommandHealTailMaskSample,
		MaskIndex:             NativeCommandHealTailMaskIndex,
		MaskPairs:             NativeCommandHealTailMaskPairs,
		DigitResource:         NativeCommandHealTailDigitResource,
		DigitBias:             NativeCommandHealTailDigitBias,
		DigitFrames:           NativeCommandHealTailDigitFrames,
		DigitFrameDelayMs:     NativeCommandHealTailDigitDelayMs,
		DigitHoldMs:           NativeCommandHealTailDigitHoldMs,
		DigitVertical:         [25]int{15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15},
	}, nil
}
