package battle

import "fmt"

// NativeAIPhysicalDestinations reproduces the destination-grid portion of
// FD2.EXE 0x14237:
//
//	0x145cd -> 0x4e040 -> 0x146d1 -> 0x14b16
//
// selector is kept raw.  Zero and nonzero select opposite record+6 groups;
// this function does not rename that caller-owned distinction as a camp or
// phase.  The returned cells follow 0x14b16's stable row-major order.
func NativeAIPhysicalDestinations(
	w, h int,
	records []byte,
	count, actor, selector, initialBudget int,
	baseFlags, terrainMoveCodes, costRow []byte,
) ([]Cell, error) {
	field, err := nativeMovementDestinationField(w, h, records, count, actor, selector, initialBudget, baseFlags, terrainMoveCodes, costRow)
	if err != nil {
		return nil, err
	}
	result := make([]Cell, 0)
	for index, value := range field {
		if value != 0xff {
			result = append(result, Cell{X: index % w, Y: index / w})
		}
	}
	return result, nil
}

// nativeMovementDestinationField 保留 0x4E040 的剩餘預算，供玩家地形繪圖與 AI 共用。
func nativeMovementDestinationField(
	w, h int, records []byte, count, actor, selector, initialBudget int,
	baseFlags, terrainMoveCodes, costRow []byte,
) ([]byte, error) {
	if w <= 0 || h <= 0 || len(baseFlags) != w*h || len(terrainMoveCodes) != w*h {
		return nil, fmt.Errorf("native AI destination grid is malformed")
	}
	if count < 0 || count > len(records)/nativeRecordSize || actor < 0 || actor >= count {
		return nil, fmt.Errorf("native AI destination roster/actor is malformed")
	}
	if initialBudget < 0 || initialBudget > 0xff || len(costRow) != NativeMovementCostRowSize {
		return nil, fmt.Errorf("native AI destination movement inputs are malformed")
	}
	for cell, code := range terrainMoveCodes {
		if int(code) >= len(costRow) {
			return nil, fmt.Errorf("native AI destination terrain code %d at cell %d is out of bounds", code, cell)
		}
	}

	flags := append([]byte(nil), baseFlags...)
	for unit := 0; unit < count; unit++ {
		record := records[unit*nativeRecordSize:]
		if record[5]&1 != 0 || !nativeAIOppositeSelectorGroup(record[6], selector) {
			continue
		}
		x, y := int(record[0]), int(record[1])
		if x < 0 || y < 0 || x >= w || y >= h {
			return nil, fmt.Errorf("native AI destination unit %d is outside the grid", unit)
		}
		flags[y*w+x] |= NativeCommandGridBlocked
		for _, delta := range nativeAIDestinationNeighbours {
			nx, ny := x+delta[0], y+delta[1]
			if nx >= 0 && ny >= 0 && nx < w && ny < h {
				flags[ny*w+nx] |= NativeCommandGridZeroBudget
			}
		}
	}

	field := make([]byte, w*h)
	for index := range field {
		field[index] = 0xff
	}
	actorRecord := records[actor*nativeRecordSize:]
	origin := Cell{X: int(actorRecord[0]), Y: int(actorRecord[1])}
	if origin.X < 0 || origin.Y < 0 || origin.X >= w || origin.Y >= h {
		return nil, fmt.Errorf("native AI destination actor is outside the grid")
	}
	field[origin.Y*w+origin.X] = byte(initialBudget)
	nativeAIPropagateDestinationBudget(
		w, h, origin, byte(initialBudget), flags, terrainMoveCodes, costRow, field,
	)

	// 0x146d1 removes occupied cells in the selector's own group after the
	// flood-fill.  The actor itself is explicitly skipped.
	for unit := 0; unit < count; unit++ {
		if unit == actor {
			continue
		}
		record := records[unit*nativeRecordSize:]
		if record[5]&1 != 0 || !nativeAISameSelectorGroup(record[6], selector) {
			continue
		}
		x, y := int(record[0]), int(record[1])
		if x < 0 || y < 0 || x >= w || y >= h {
			return nil, fmt.Errorf("native AI destination unit %d is outside the grid", unit)
		}
		field[y*w+x] = 0xff
	}

	return field, nil
}

var nativeAIDestinationNeighbours = [][2]int{
	{1, 0},
	{-1, 0},
	{0, 1},
	{0, -1},
}

func nativeAIOppositeSelectorGroup(recordByte6 byte, selector int) bool {
	if selector == 0 {
		return recordByte6 != 0
	}
	return recordByte6 == 0
}

func nativeAISameSelectorGroup(recordByte6 byte, selector int) bool {
	if selector == 0 {
		return recordByte6 == 0
	}
	return recordByte6 != 0
}

func nativeAIPropagateDestinationBudget(
	w, h int,
	cell Cell,
	budget byte,
	flags, terrainMoveCodes, costRow, field []byte,
) {
	for _, delta := range nativeAIDestinationNeighbours {
		next := Cell{X: cell.X + delta[0], Y: cell.Y + delta[1]}
		if next.X < 0 || next.Y < 0 || next.X >= w || next.Y >= h {
			continue
		}
		index := next.Y*w + next.X
		cost := costRow[terrainMoveCodes[index]]
		if cost > budget {
			continue
		}
		nextBudget := budget - cost
		// 0x4e18f uses signed JLE.  Initial 0xff therefore behaves as -1,
		// while ordinary nonnegative budgets retain only strict improvements.
		if int8(nextBudget) <= int8(field[index]) || flags[index]&NativeCommandGridBlocked != 0 {
			continue
		}
		if flags[index]&NativeCommandGridZeroBudget != 0 {
			nextBudget = 0
		}
		field[index] = nextBudget
		nativeAIPropagateDestinationBudget(
			w, h, next, nextBudget, flags, terrainMoveCodes, costRow, field,
		)
	}
}
