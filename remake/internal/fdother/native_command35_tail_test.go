package fdother

import "testing"

func TestBuildNativeCommand35TailSchedulePreservesRawRows(t *testing.T) {
	rows, err := BuildNativeCommand35TailSchedule()
	if err != nil || len(rows) != 3 {
		t.Fatalf("rows=%#v err=%v", rows, err)
	}
	want := [][5]int{{26, 0x8a, 9, 3, 0xc0}, {22, 0xaa, 13, 3, 0x23}, {27, 0x9e, 12, 2, 0xc0}}
	for index, row := range rows {
		if got := [5]int{row.CommandID, row.EffectStart, row.EffectFrames, row.EffectSample, row.MaskIndex}; got != want[index] {
			t.Fatalf("row %d=%#v", index, row)
		}
		if row.MaskPairs != 5 || row.DigitBias != 0x5e || row.DigitFrames != 22 || row.DigitHoldMilliseconds != 500 {
			t.Fatalf("tail row %d=%#v", index, row)
		}
	}
	if len(rows[1].ExtraSampleFrameIndices) != 1 || rows[1].ExtraSampleFrameIndices[0] != 7 || rows[1].ExtraSample != 3 {
		t.Fatalf("command22 extra marker=%#v", rows[1])
	}
}
