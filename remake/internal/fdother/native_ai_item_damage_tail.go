package fdother

import "fmt"

const (
	NativeAIItemDamageEffectResource = 6
	NativeAIItemDamageEffectStart    = 0x31
	NativeAIItemDamageEffectFrames   = 8
	NativeAIItemDamageEffectSample   = 6
	NativeAIItemDamageBlendFrames    = 10
	NativeAIItemDamageRawBase        = 0x20
	NativeAIItemDamageDigitResource  = 5
	NativeAIItemDamageDigitBias      = 0x5e
	NativeAIItemDamageDigitFrames    = 22
	NativeAIItemDamageDigitHoldMs    = 500
)

// NativeAIItemDamageTailSchedule is the caller-specific data contract for
// type 20/24's 0x1c4cc -> 0x1cd17 -> 0x1e0db/0x1e1dc -> 0x1df58 path.
// Raw descriptor and palette values deliberately remain unnamed.
type NativeAIItemDamageTailSchedule struct {
	EffectResource, EffectStart, EffectFrames int
	EffectSample, EffectFrameDelayTicks       int
	BlendFrames, RawBase, BlendDelayTicks     int
	Blend                                     [NativeAIItemDamageBlendFrames]int
	DigitResource, DigitBias, DigitFrames     int
	DigitFrameDelayTicks, DigitHoldMs         int
	DigitVertical                             [25]int
}

func BuildNativeAIItemDamageTailSchedule(commandID int) (NativeAIItemDamageTailSchedule, error) {
	if commandID != 0 && commandID != 2 && commandID != 3 {
		return NativeAIItemDamageTailSchedule{}, fmt.Errorf("native AI item damage tail: unsupported command %d", commandID)
	}
	return NativeAIItemDamageTailSchedule{
		EffectResource: NativeAIItemDamageEffectResource, EffectStart: NativeAIItemDamageEffectStart,
		EffectFrames: NativeAIItemDamageEffectFrames, EffectSample: NativeAIItemDamageEffectSample,
		EffectFrameDelayTicks: 1, BlendFrames: NativeAIItemDamageBlendFrames,
		RawBase: NativeAIItemDamageRawBase, BlendDelayTicks: 1,
		Blend:         [NativeAIItemDamageBlendFrames]int{7, 6, 5, 4, 3, 2, 1, 0, 7, 6},
		DigitResource: NativeAIItemDamageDigitResource, DigitBias: NativeAIItemDamageDigitBias,
		DigitFrames: NativeAIItemDamageDigitFrames, DigitFrameDelayTicks: 1,
		DigitHoldMs:   NativeAIItemDamageDigitHoldMs,
		DigitVertical: [25]int{15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15},
	}, nil
}
