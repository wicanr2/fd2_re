package figani

import "fmt"

const (
	NativeCommand2EffectFrameCount = 18
	NativeCommand2SoundResource    = 83
	NativeCommand2SoundMode2Index  = 1
	NativeCommand2SoundMode5Index  = 2
	NativeCommand2SoundMode6Index  = 3
)

// NativeCommand2PresentationSchedule 保存 0x26528 已閉合的資源與
// target mode契約；mode1/2/7/8 helper仍須另一份座標證據。
type NativeCommand2PresentationSchedule struct {
	EffectResource int
	SoundResource  int
	SoundIndices   [3]int
}

func BuildNativeCommand2PresentationSchedule(rawSide byte, effect *Animation) (NativeCommand2PresentationSchedule, error) {
	resource := 26
	if rawSide == 0 {
		resource = 27
	}
	if effect == nil || effect.HeaderByte1 != 0 || effect.HeaderByte2 != NativeCommand2EffectFrameCount ||
		effect.HeaderByte4 != 0 || len(effect.Frames) != NativeCommand2EffectFrameCount {
		return NativeCommand2PresentationSchedule{}, fmt.Errorf("figani: command2 FDOTHER #%d signature mismatch", resource)
	}
	for index, frame := range effect.Frames {
		if frame.Width <= 0 || frame.Height <= 0 || len(frame.Pixels) != frame.Width*frame.Height || len(frame.Mask) != len(frame.Pixels) {
			return NativeCommand2PresentationSchedule{}, fmt.Errorf("figani: command2 FDOTHER #%d frame %d malformed", resource, index)
		}
	}
	return NativeCommand2PresentationSchedule{
		EffectResource: resource,
		SoundResource:  NativeCommand2SoundResource,
		SoundIndices:   [3]int{NativeCommand2SoundMode2Index, NativeCommand2SoundMode5Index, NativeCommand2SoundMode6Index},
	}, nil
}
