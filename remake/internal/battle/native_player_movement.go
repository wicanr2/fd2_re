package battle

import "fmt"

// NativePlayerMovement 保存 0x18890 的玩家 selector 1 呼叫結果。
// 範圍與路徑都消費同一份已驗證的地形、角色與移動表，不讀正規化 BFS。
type NativePlayerMovement struct {
	Field                 []byte
	Reach                 map[Cell]bool
	w, h, budget          int
	origin                Cell
	flags, terrain, costs []byte
}

func (s *State) PlanNativePlayerMovement(u *Unit) (*NativePlayerMovement, error) {
	if s == nil || u == nil || !s.HasNativeMapViewState {
		return nil, fmt.Errorf("native player movement: runtime provenance unavailable")
	}
	actor := -1
	for i, candidate := range s.Units {
		if candidate == u {
			actor = i
			break
		}
	}
	if actor < 0 || len(s.nativeMovementCostRows) != NativeMovementCostRowCount {
		return nil, fmt.Errorf("native player movement: actor or cost rows unavailable")
	}
	records, err := NativeAIScoringRecords(s.Units)
	if err != nil {
		return nil, err
	}
	r := records[actor*nativeRecordSize : (actor+1)*nativeRecordSize]
	if r[6] != 2 {
		return nil, fmt.Errorf("native player movement: raw actor camp is not 2")
	}
	selector := int(r[0x20])
	// 0x188E6→0x1F183；+7==0x1c 明確排除特殊地形分支，另選表16。
	if r[7] != 0x1c && (r[0x20] == 19 || r[0x1f] == 4 || r[0x1f] == 5) {
		selector = 19
	} else if r[7] == 0x1c {
		selector = 16
	}
	if selector >= len(s.nativeMovementCostRows) {
		return nil, fmt.Errorf("native player movement: cost selector %d unavailable", selector)
	}
	base, err := NativeCompositionBaseFlags(s.W, s.H, s.NativeCompositionEventBytes)
	if err != nil {
		return nil, err
	}
	field, err := nativeMovementDestinationField(s.W, s.H, records, len(s.Units), actor, 1, int(r[0x3b]), base, s.NativeTerrainMoveCodes, s.nativeMovementCostRows[selector])
	if err != nil {
		return nil, err
	}
	flags, err := NativeCommandRuntimeFlags(s.W, s.H, s.NativeCompositionEventBytes, s.Units, 1)
	if err != nil {
		return nil, err
	}
	p := &NativePlayerMovement{Field: field, Reach: make(map[Cell]bool), w: s.W, h: s.H, budget: int(r[0x3b]), origin: Cell{int(r[0]), int(r[1])}, flags: flags, terrain: append([]byte(nil), s.NativeTerrainMoveCodes...), costs: append([]byte(nil), s.nativeMovementCostRows[selector]...)}
	for i, value := range field {
		if value != 0xff {
			p.Reach[Cell{i % s.W, i / s.W}] = true
		}
	}
	return p, nil
}

func (p *NativePlayerMovement) Path(destination Cell) ([]Cell, error) {
	if p == nil || !p.Reach[destination] {
		return nil, fmt.Errorf("native player movement: unreachable destination")
	}
	directions, reachable, err := NativePathDirections(p.w, p.h, p.origin, destination, p.budget, 0, p.flags, p.terrain, p.costs)
	if err != nil {
		return nil, err
	}
	if !reachable {
		return nil, fmt.Errorf("native player movement: destination has no path")
	}
	return nativeAIDirectionPath(p.origin, directions)
}
