package fdother

import (
	"errors"
	"fmt"
)

const (
	NativeCommand9PlayerEffectResource = 6
	NativeCommand9PlayerEffectStart    = 87
	NativeCommand9PlayerEffectFrames   = 27
	NativeCommand9PlayerSoundResource  = 88
	NativeCommand9PlayerInitialSample  = 14
	NativeCommand9PlayerRepeatSample   = 15
	NativeCommand9ResultResource       = 5
	NativeCommand9ResultDigitBias      = 94
	NativeCommand9ResultFrames         = 22
	NativeCommand9ResultDelayMS        = 2
	NativeCommand9ResultHoldMS         = 500
)

// NativeCommand9PlayerSchedule 保存玩家路徑
// 0x1D6C8 -> 0x214AD -> 0x1C4CC -> 0x1DF58 的原始表值。
// 它不描述敵方 0x2A6BD -> 0x275D6 的另一套畫面排程。
type NativeCommand9PlayerSchedule struct {
	EffectResource, EffectStart, EffectFrames  int
	SoundResource, InitialSample, RepeatSample int
	RepeatSampleFrames                         [2]int
	EffectFrameDelayTicks                      int
	ResultResource, ResultDigitBias            int
	ResultMissDescriptors                      [4]int
	ResultFrames, ResultDelayMS, ResultHoldMS  int
	ResultVertical                             [25]int
}

func BuildNativeCommand9PlayerSchedule(effect, result []LMI1Entry) (NativeCommand9PlayerSchedule, error) {
	if len(effect) < NativeCommand9PlayerEffectStart+NativeCommand9PlayerEffectFrames {
		return NativeCommand9PlayerSchedule{}, errors.New("native command9 player effect descriptors unavailable")
	}
	for index := NativeCommand9PlayerEffectStart; index < NativeCommand9PlayerEffectStart+NativeCommand9PlayerEffectFrames; index++ {
		entry := effect[index]
		if entry.Width <= 0 || entry.Height <= 0 || len(entry.Pixels) != entry.Width*entry.Height {
			return NativeCommand9PlayerSchedule{}, fmt.Errorf("native command9 player effect descriptor %d invalid", index)
		}
	}
	required := [...]int{94, 95, 96, 97, 98, 99, 100, 101, 102, 103, 116, 117, 118}
	for _, index := range required {
		if index >= len(result) || result[index].Width <= 0 || result[index].Height <= 0 || len(result[index].Pixels) != result[index].Width*result[index].Height {
			return NativeCommand9PlayerSchedule{}, fmt.Errorf("native command9 result descriptor %d unavailable", index)
		}
	}
	return NativeCommand9PlayerSchedule{
		EffectResource: NativeCommand9PlayerEffectResource, EffectStart: NativeCommand9PlayerEffectStart,
		EffectFrames: NativeCommand9PlayerEffectFrames, SoundResource: NativeCommand9PlayerSoundResource,
		InitialSample: NativeCommand9PlayerInitialSample, RepeatSample: NativeCommand9PlayerRepeatSample,
		RepeatSampleFrames: [2]int{15, 19}, EffectFrameDelayTicks: 1,
		ResultResource: NativeCommand9ResultResource, ResultDigitBias: NativeCommand9ResultDigitBias,
		ResultMissDescriptors: [4]int{116, 117, 118, 118}, ResultFrames: NativeCommand9ResultFrames,
		ResultDelayMS: NativeCommand9ResultDelayMS, ResultHoldMS: NativeCommand9ResultHoldMS,
		ResultVertical: [25]int{15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15},
	}, nil
}
