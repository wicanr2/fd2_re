package fdother

import "testing"

func TestBuildNativeCommand2022TailSchedulePreservesRawRows(t *testing.T) {
	rows, err := BuildNativeCommand2022TailSchedule()
	if err != nil || len(rows) != 3 {
		t.Fatalf("schedule=%#v err=%v", rows, err)
	}
	want := [][6]int{{20, 0xcc, 13, 4, 0xc0, 0x69}, {21, 0xd9, 13, 4, 0xc0, 0x69}, {22, 0xaa, 13, 3, 0x23, 0x5e}}
	for index, row := range rows {
		got := [6]int{row.CommandID, row.EffectStart, row.EffectFrames, row.EffectSample, row.MaskIndex, row.DigitBias}
		if got != want[index] || row.MaskPairs != 5 || row.DigitFrames != 22 || row.DigitHoldMilliseconds != 500 {
			t.Fatalf("row %d=%#v", index, row)
		}
	}
	if len(rows[0].ExtraSampleFrameIndices) != 0 || len(rows[1].ExtraSampleFrameIndices) != 0 || len(rows[2].ExtraSampleFrameIndices) != 1 || rows[2].ExtraSampleFrameIndices[0] != 7 {
		t.Fatalf("extra sample markers=%#v %#v %#v", rows[0].ExtraSampleFrameIndices, rows[1].ExtraSampleFrameIndices, rows[2].ExtraSampleFrameIndices)
	}
}
