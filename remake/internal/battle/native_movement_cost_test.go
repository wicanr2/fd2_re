package battle

import (
	"testing"
)

func nativeMovementRowsForTest() [][]byte {
	rows := make([][]byte, NativeMovementCostRowCount)
	for selector := range rows {
		rows[selector] = make([]byte, NativeMovementCostRowSize)
	}
	return rows
}

// 0x115B6 mode 6 的 Enter 判準：目的格沒有別的活記錄，而且該職業的地形成本
// **不是 20**（20 = 不可通行，`0x11706 cmp eax,0x14` 相等就回輸入迴圈）。
func TestNativeRelocationDestinationAllowedMatchesMode6Gates(t *testing.T) {
	records := make([]byte, 3*nativeRecordSize)
	target := records[nativeRecordSize : 2*nativeRecordSize]
	target[0], target[1], target[0x20] = 4, 5, 9
	rows := nativeMovementRowsForTest()
	rows[9][3] = 1 // 可通行
	allowed, err := NativeRelocationDestinationAllowed(records, 3, 1, 8, 9, 3, rows)
	if err != nil || !allowed {
		t.Fatalf("allowed=%v err=%v", allowed, err)
	}

	occupant := records[2*nativeRecordSize:]
	occupant[0], occupant[1], occupant[5] = 8, 9, 0
	if allowed, err = NativeRelocationDestinationAllowed(records, 3, 1, 8, 9, 3, rows); err != nil || allowed {
		t.Fatalf("occupied allowed=%v err=%v", allowed, err)
	}
	occupant[5] = 1
	if allowed, err = NativeRelocationDestinationAllowed(records, 3, 1, 8, 9, 3, rows); err != nil || !allowed {
		t.Fatalf("bit0 occupant allowed=%v err=%v", allowed, err)
	}
	rows[9][3] = 20 // 不可通行 → 拒絕
	if allowed, err = NativeRelocationDestinationAllowed(records, 3, 1, 8, 9, 3, rows); err != nil || allowed {
		t.Fatalf("不可通行地形 allowed=%v err=%v", allowed, err)
	}
}

func TestNativeRelocationDestinationSelectorOverrides(t *testing.T) {
	records := make([]byte, nativeRecordSize)
	rows := nativeMovementRowsForTest()
	// 只有被選中的那一列在這個地形碼上是 20（不可通行）；選錯列就會放行。
	rows[1][2], rows[19][2] = 20, 20
	for _, tc := range []struct {
		name                 string
		unit7, race, classID byte
		wantSelector         int
	}{
		{"unit7", 0x1c, 4, 0x13, 1},
		{"class", 0, 0, 0x13, 19},
		{"race4", 0, 4, 3, 19},
		{"race5", 0, 5, 3, 19},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := records[:nativeRecordSize]
			for i := range record {
				record[i] = 0
			}
			record[0x07], record[0x1f], record[0x20] = tc.unit7, tc.race, tc.classID
			allowed, err := NativeRelocationDestinationAllowed(records, 1, 0, 1, 1, 2, rows)
			if err != nil || allowed || rows[tc.wantSelector][2] != 20 {
				t.Fatalf("allowed=%v err=%v", allowed, err)
			}
		})
	}
}

func TestLoadNativeMovementCostRowsFixture(t *testing.T) {
	rows, err := LoadNativeMovementCostRows("../../assets/data/native_movement_cost_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 29 || len(rows[0]) != 20 {
		t.Fatalf("shape=%d/%d", len(rows), len(rows[0]))
	}
	for column, value := range rows[0] {
		if value != 1 {
			t.Fatalf("row0[%d]=%d want1", column, value)
		}
	}
	// 錨點是地形碼語意本身：碼 1／5 不可移動（20），碼 0 正常（1），碼 4 沼澤
	// 步行 -2（2）。這組對照同時來自 modify2.md 的靜態定義與 notes.md 的玩家實測
	// 扣減值；早先的匯出整份早一個位元組，碼 1 變成 1、碼 2 變成 20，海可以走、
	// 森林反而是牆。
	if rows[1][1] != 20 || rows[1][5] != 20 {
		t.Fatalf("row1 不可移動錨點=%d/%d", rows[1][1], rows[1][5])
	}
	if rows[1][0] != 1 || rows[1][2] != 1 || rows[1][3] != 2 || rows[1][4] != 2 {
		t.Fatalf("row1 正常/森林/沼澤=%d/%d/%d/%d", rows[1][0], rows[1][2], rows[1][3], rows[1][4])
	}
	// 騎兵列：森林（碼 2）多扣 1、沼澤（碼 4）多扣 1，對應 notes.md 的騎兵 -2／-3。
	if rows[3][2] != 2 || rows[3][3] != 3 || rows[3][4] != 3 {
		t.Fatalf("騎兵列=%d/%d/%d", rows[3][2], rows[3][3], rows[3][4])
	}
	// 飛行列：碼 1..4 一律 1，只有碼 5 仍然擋。
	for _, code := range []int{0, 1, 2, 3, 4} {
		if rows[19][code] != 1 {
			t.Fatalf("飛行列 碼%d=%d，應為 1", code, rows[19][code])
		}
	}
	if rows[19][5] != 20 {
		t.Fatalf("飛行列 碼5=%d，應為 20", rows[19][5])
	}
}
