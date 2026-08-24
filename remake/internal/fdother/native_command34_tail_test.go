package fdother

import "testing"

func TestBuildNativeCommand34TailSchedulePreservesRawRows(t *testing.T) {
	rows, err := BuildNativeCommand34TailSchedule()
	if err != nil || len(rows) != 3 {
		t.Fatalf("schedule=%#v err=%v", rows, err)
	}
	want := [][5]int{{17, 0xb7, 8, 6, 0x92}, {18, 0x7e, 12, 7, 0x48}, {19, 0x93, 11, 8, 0xd8}}
	for index, row := range rows {
		got := [5]int{row.CommandID, row.EffectStart, row.EffectFrames, row.EffectSample, row.MaskIndex}
		if got != want[index] || row.MaskPairs != 5 || row.DigitBias != 0x69 || row.DigitFrames != 22 || row.DigitHoldMilliseconds != 500 {
			t.Fatalf("row %d=%#v", index, row)
		}
	}
	if len(rows[0].ExtraSampleFrames) != 0 || len(rows[1].ExtraSampleFrames) != 1 || rows[1].ExtraSampleFrames[0] != 4 || len(rows[2].ExtraSampleFrames) != 2 || rows[2].ExtraSampleFrames[0] != 3 || rows[2].ExtraSampleFrames[1] != 6 {
		t.Fatalf("extra sample markers=%#v %#v %#v", rows[0].ExtraSampleFrames, rows[1].ExtraSampleFrames, rows[2].ExtraSampleFrames)
	}
}
