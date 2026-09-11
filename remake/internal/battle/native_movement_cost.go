package battle

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

const (
	NativeMovementCostRowCount = 29
	NativeMovementCostRowSize  = 20
)

type nativeMovementCostRowJSON struct {
	Selector int    `json:"selector"`
	Raw      string `json:"raw"`
}

func LoadNativeMovementCostRows(path string) ([][]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rows []nativeMovementCostRowJSON
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	if len(rows) != NativeMovementCostRowCount {
		return nil, fmt.Errorf("native movement rows=%d want %d", len(rows), NativeMovementCostRowCount)
	}
	result := make([][]byte, len(rows))
	for index, row := range rows {
		if row.Selector != index {
			return nil, fmt.Errorf("native movement selector=%d at index=%d", row.Selector, index)
		}
		decoded, err := hex.DecodeString(row.Raw)
		if err != nil || len(decoded) != NativeMovementCostRowSize {
			return nil, fmt.Errorf("native movement selector=%d has invalid raw row", index)
		}
		result[index] = decoded
	}
	return result, nil
}

// NativeRelocationDestinationAllowed reproduces 0x115b6 mode 6's Enter
// predicate. targetUnit is excluded from occupancy; any other record on the
// destination with raw +5 bit0 clear blocks it. The target's raw fields select
// a 0x4e555 row; the destination is refused when that row's terrain entry is
// 20（不可通行）。
//
// 原版：`0x11702 movzx eax,[ebx+eax]` 取 `costRow[地形碼]`，`0x11706 cmp eax,0x14`
// 相等就 `je 0x117A9` 回到輸入迴圈（不接受），否則 `0x1170F mov eax,1` 接受。
// 早先寫成「等於 20 才允許」是把極性寫反了；那時成本表本身也錯位（碼 2 變成 20），
// 兩個錯誤在森林格上互相抵消，所以測不出來。
func NativeRelocationDestinationAllowed(
	records []byte,
	count, targetUnit int,
	destinationX, destinationY byte,
	terrainIndex int,
	costRows [][]byte,
) (bool, error) {
	if count < 0 || count > len(records)/nativeRecordSize {
		return false, fmt.Errorf("native relocation count=%d is out of bounds", count)
	}
	if targetUnit < 0 || targetUnit >= count {
		return false, recordBoundsError(targetUnit)
	}
	if len(costRows) != NativeMovementCostRowCount ||
		terrainIndex < 0 || terrainIndex >= NativeMovementCostRowSize {
		return false, fmt.Errorf("native relocation movement table/index is invalid")
	}
	for selector, row := range costRows {
		if len(row) != NativeMovementCostRowSize {
			return false, fmt.Errorf("native movement selector=%d row is malformed", selector)
		}
	}
	for unit := 0; unit < count; unit++ {
		if unit == targetUnit {
			continue
		}
		record := records[unit*nativeRecordSize:]
		if record[0] == destinationX && record[1] == destinationY && record[5]&1 == 0 {
			return false, nil
		}
	}
	target := records[targetUnit*nativeRecordSize:]
	selector := int(target[0x20])
	if target[0x07] == 0x1c {
		selector = 1
	} else if target[0x20] == 0x13 || target[0x1f] == 4 || target[0x1f] == 5 {
		selector = 19
	}
	if selector < 0 || selector >= len(costRows) {
		return false, fmt.Errorf("native relocation selector=%d is out of bounds", selector)
	}
	return costRows[selector][terrainIndex] != 20, nil
}
