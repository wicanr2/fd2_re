package fdother

import "fmt"

// BuildNativeCommand33TailSchedule preserves sub_211A4's hard-coded command
// 13 presentation row without importing command 13's separate 0x21EB1 front.
func BuildNativeCommand33TailSchedule() (NativeCommandHealTailSchedule, error) {
	schedule, err := BuildNativeCommandHealTailSchedule(13)
	if err != nil {
		return NativeCommandHealTailSchedule{}, err
	}
	if schedule.EffectResource != 6 || schedule.EffectStart != 0x39 || schedule.EffectFrames != 7 ||
		schedule.EffectSample != 12 || schedule.MaskSample != 1 || schedule.MaskIndex != 0xc0 ||
		schedule.MaskPairs != 5 || schedule.DigitResource != 5 || schedule.DigitBias != 0x69 ||
		schedule.DigitFrames != 22 || schedule.DigitHoldMs != 500 {
		return NativeCommandHealTailSchedule{}, fmt.Errorf("native command33 hard-coded command13 tail unavailable")
	}
	return schedule, nil
}
