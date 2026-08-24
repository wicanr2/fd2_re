package fdother

import "fmt"

// NativeCommand35StageSchedule preserves one raw presentation row consumed
// by the three 0x22D1B calls. Frame indices are native zero-based counters.
type NativeCommand35StageSchedule struct {
	CommandID               int
	EffectStart             int
	EffectFrames            int
	EffectSample            int
	ExtraSampleFrameIndices []int
	ExtraSample             int
	MaskIndex               int
	MaskPairs               int
	DigitBias               int
	DigitFrames             int
	DigitHoldMilliseconds   int
	DigitVertical           [25]int
}

func BuildNativeCommand35TailSchedule() ([]NativeCommand35StageSchedule, error) {
	vertical := [25]int{15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15}
	rows := []NativeCommand35StageSchedule{
		{CommandID: 26, EffectStart: 0x8a, EffectFrames: 9, EffectSample: 3, MaskIndex: 0xc0},
		{CommandID: 22, EffectStart: 0xaa, EffectFrames: 13, EffectSample: 3, ExtraSampleFrameIndices: []int{7}, ExtraSample: 3, MaskIndex: 0x23},
		{CommandID: 27, EffectStart: 0x9e, EffectFrames: 12, EffectSample: 2, MaskIndex: 0xc0},
	}
	for index := range rows {
		rows[index].MaskPairs = 5
		rows[index].DigitBias = 0x5e
		rows[index].DigitFrames = 22
		rows[index].DigitHoldMilliseconds = 500
		rows[index].DigitVertical = vertical
		if rows[index].EffectFrames <= 0 || rows[index].EffectSample <= 0 || rows[index].EffectStart < 0 {
			return nil, fmt.Errorf("native command35 malformed stage %d", index)
		}
	}
	return rows, nil
}
