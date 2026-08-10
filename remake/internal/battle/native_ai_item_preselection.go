package battle

import "fmt"

// NativeAI1567EScoreResult preserves the positive best tuple written by
// 0x1567e to [0x53c33/37/3b/3f]. InventorySlot is the raw slot index saved in
// [0x53c3f], not the item command byte.
type NativeAI1567EScoreResult struct {
	MaxScore      int
	X             int
	Y             int
	InventorySlot int
	ItemID        int
	// TargetIndices is the detached raw target list for the winning
	// destination.  It is retained so the execution owner can re-run the
	// second target/effect stage instead of inventing a target from normalized
	// party data.
	TargetIndices     []byte
	HasPositiveWinner bool
}

// ScoreNativeAI1567E connects the proven item-command preselection slices:
// 0x1b8a6 count-sized slot scan, caller-specific 0x14818/0x149f8 geometry,
// 0x15880 scoring, and strict greater-than winner replacement. It does not
// execute the selected item or infer an effect name.
func ScoreNativeAI1567E(
	w, h int,
	records []byte,
	count, actor, selector int,
	itemRows []byte,
	book []NativeCommandRecord,
	baseFlags []byte,
) (NativeAI1567EScoreResult, error) {
	var result NativeAI1567EScoreResult
	if w <= 0 || h <= 0 || count <= 0 || count > 0x100 ||
		actor < 0 || actor >= count || (selector != 0 && selector != 1) ||
		len(records) != count*nativeRecordSize || len(baseFlags) != w*h ||
		len(itemRows) == 0 || len(itemRows)%NativeItemEffectRowSize != 0 {
		return result, fmt.Errorf("native AI 0x1567e inputs are malformed")
	}
	actorRecord := records[actor*nativeRecordSize:]
	actorCell := Cell{X: int(actorRecord[0]), Y: int(actorRecord[1])}
	if actorCell.X < 0 || actorCell.Y < 0 || actorCell.X >= w || actorCell.Y >= h {
		return result, fmt.Errorf("native AI 0x1567e actor is outside the grid")
	}
	occupied, err := NativeInventoryOccupiedCount(records, actor)
	if err != nil {
		return result, err
	}
	for slot := 0; slot < occupied; slot++ {
		itemID := int(actorRecord[0x0b+slot*2])
		rowOffset := itemID * NativeItemEffectRowSize
		if rowOffset < 0 || rowOffset+NativeItemEffectRowSize > len(itemRows) {
			return NativeAI1567EScoreResult{}, fmt.Errorf(
				"native AI 0x1567e item %d row is unavailable", itemID,
			)
		}
		row := itemRows[rowOffset : rowOffset+NativeItemEffectRowSize]
		itemType := int(row[0x0d])
		if itemType == 0 {
			continue
		}
		command := int(row[0x10])
		inner := 0
		if command > 0x0f {
			inner = 1
		}
		field, err := NativeCommandTargetFieldBytes(
			w, h, actorCell, command, inner, baseFlags,
		)
		if err != nil {
			return NativeAI1567EScoreResult{}, err
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if field[y*w+x] == 0xff {
					continue
				}
				destination := Cell{X: x, Y: y}
				var targets []byte
				if command > 0x0f {
					targets, err = nativeAI1567ECardinalTargets(
						w, h, records, count, actorCell, destination, command-0x10,
					)
				} else {
					targetCode := int(row[0x11])
					if selector == 0 {
						if targetCode == 0 {
							targetCode = 1
						} else {
							targetCode = 0
						}
					}
					targets, err = nativeAI1567EAreaTargets(
						w, h, records, count, destination,
						int(row[0x12]), targetCode, baseFlags,
					)
				}
				if err != nil {
					return NativeAI1567EScoreResult{}, err
				}
				if len(targets) == 0 {
					continue
				}
				score, err := ScoreNativeAIItemCommandTargets(
					records, targets, itemType,
					int(row[0x0e])|int(row[0x0f])<<8, book,
				)
				if err != nil {
					return NativeAI1567EScoreResult{}, err
				}
				if score > result.MaxScore {
					result = NativeAI1567EScoreResult{
						MaxScore: score, X: x, Y: y,
						InventorySlot: slot, ItemID: itemID,
						TargetIndices:     append([]byte(nil), targets...),
						HasPositiveWinner: true,
					}
				}
			}
		}
	}
	return result, nil
}

func nativeAI1567EAreaTargets(
	w, h int,
	records []byte,
	count int,
	origin Cell,
	mode, targetCode int,
	baseFlags []byte,
) ([]byte, error) {
	field, err := NativeCommandTargetFieldBytes(
		w, h, origin, mode, 0, baseFlags,
	)
	if err != nil {
		return nil, err
	}
	targets := make([]byte, 0)
	for index := 0; index < count; index++ {
		record := records[index*nativeRecordSize:]
		if record[5]&1 != 0 ||
			!nativeAIScoredRawTargetMatches(targetCode, record[6]) {
			continue
		}
		cell := Cell{X: int(record[0]), Y: int(record[1])}
		if cell.X < 0 || cell.Y < 0 || cell.X >= w || cell.Y >= h {
			return nil, fmt.Errorf("native AI 0x1567e target is outside the grid")
		}
		if field[cell.Y*w+cell.X] != 0xff {
			targets = append(targets, byte(index))
		}
	}
	return targets, nil
}

func nativeAI1567ECardinalTargets(
	w, h int,
	records []byte,
	count int,
	start, destination Cell,
	steps int,
) ([]byte, error) {
	if steps < 0 || start.X < 0 || start.Y < 0 || start.X >= w || start.Y >= h ||
		destination.X < 0 || destination.Y < 0 || destination.X >= w || destination.Y >= h {
		return nil, fmt.Errorf("native AI 0x1567e cardinal inputs are malformed")
	}
	dx, dy := 0, 0
	if start.X != destination.X {
		if start.X > destination.X {
			dx = -1
		} else {
			dx = 1
		}
	} else if start.Y > destination.Y {
		dy = -1
	} else {
		dy = 1
	}
	cell := start
	targets := make([]byte, 0)
	for step := 0; step < steps; step++ {
		cell.X += dx
		cell.Y += dy
		if cell.X < 0 || cell.Y < 0 || cell.X >= w || cell.Y >= h {
			continue
		}
		for index := 0; index < count; index++ {
			record := records[index*nativeRecordSize:]
			if record[5]&1 != 0 ||
				int(record[0]) != cell.X || int(record[1]) != cell.Y {
				continue
			}
			if record[6] == 0 {
				targets = append(targets, byte(index))
			}
			break
		}
	}
	return targets, nil
}
