package fdother

import "fmt"

// BuildNativeCommand2022TailSchedule preserves the three command-indexed
// rows consumed by the shared player/enemy wrappers at 0x22A85, 0x22BC6 and
// 0x22BE1. The return type is shared with the byte-identical tail primitives;
// its historical name does not imply command-34 ownership.
func BuildNativeCommand2022TailSchedule() ([]NativeCommand34StageSchedule, error) {
	vertical := [25]int{15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15}
	rows := []NativeCommand34StageSchedule{
		{CommandID: 20, EffectStart: 0xcc, EffectFrames: 13, EffectSample: 4, MaskIndex: 0xc0, DigitBias: 0x69},
		{CommandID: 21, EffectStart: 0xd9, EffectFrames: 13, EffectSample: 4, MaskIndex: 0xc0, DigitBias: 0x69},
		{CommandID: 22, EffectStart: 0xaa, EffectFrames: 13, EffectSample: 3, ExtraSampleFrameIndices: []int{7}, MaskIndex: 0x23, DigitBias: 0x5e},
	}
	for index := range rows {
		rows[index].MaskPairs = 5
		rows[index].DigitFrames = 22
		rows[index].DigitHoldMilliseconds = 500
		rows[index].DigitVertical = vertical
		if rows[index].EffectStart < 0 || rows[index].EffectFrames <= 0 || rows[index].EffectSample <= 0 || rows[index].DigitBias < 0 {
			return nil, fmt.Errorf("native command20-22 malformed stage %d", index)
		}
	}
	return rows, nil
}
