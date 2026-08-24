package fdother

import "fmt"

// NativeCommand34StageSchedule preserves one of the three raw presentation
// rows selected by 0x22721, 0x22866 and 0x22997. Names remain deliberately raw.
type NativeCommand34StageSchedule struct {
	CommandID               int
	EffectStart             int
	EffectFrames            int
	EffectSample            int
	ExtraSampleFrameIndices []int
	MaskIndex               int
	MaskPairs               int
	DigitBias               int
	DigitFrames             int
	DigitHoldMilliseconds   int
	DigitVertical           [25]int
}

// BuildNativeCommand34TailSchedule returns the exact 17→18→19 order used by
// command 34 after the 0x27FC9 common segment.
func BuildNativeCommand34TailSchedule() ([]NativeCommand34StageSchedule, error) {
	vertical := [25]int{15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15}
	rows := []NativeCommand34StageSchedule{
		{CommandID: 17, EffectStart: 0xb7, EffectFrames: 8, EffectSample: 6, MaskIndex: 0x92},
		{CommandID: 18, EffectStart: 0x7e, EffectFrames: 12, EffectSample: 7, ExtraSampleFrameIndices: []int{4}, MaskIndex: 0x48},
		{CommandID: 19, EffectStart: 0x93, EffectFrames: 11, EffectSample: 8, ExtraSampleFrameIndices: []int{3, 6}, MaskIndex: 0xd8},
	}
	for index := range rows {
		rows[index].MaskPairs = 5
		rows[index].DigitBias = 0x69
		rows[index].DigitFrames = 22
		rows[index].DigitHoldMilliseconds = 500
		rows[index].DigitVertical = vertical
		if rows[index].EffectStart < 0 || rows[index].EffectFrames <= 0 || rows[index].EffectSample <= 0 {
			return nil, fmt.Errorf("native command34 malformed stage %d", index)
		}
	}
	return rows, nil
}
