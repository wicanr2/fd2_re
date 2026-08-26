package battle

import (
	"encoding/binary"
	"fmt"
)

// nextNativeAIPhysicalPlan 是原始 0x14237 候選橋的第一個正式消費端。
// 範圍刻意縮窄，只接受 runtime mode 2，不猜測 caller 擁有的 0x14ef0 前置選擇。
// 因此，移動／物品／地形來源不完整時回傳錯誤，不退回正規化目標選擇。
func (s *State) nextNativeAIPhysicalPlan(u *Unit) (*AIPlan, bool, error) {
	if s == nil || u == nil || !u.HasNativeRecordByte34 {
		return nil, false, nil
	}
	mode := int(u.NativeRecordByte34 & 0x0f)
	if mode != 2 {
		return nil, false, nil
	}
	if !u.HasNativeRecordByte6 || len(s.Units) == 0 {
		return nil, true, fmt.Errorf("native AI mode 2 lacks selector/runtime roster provenance")
	}
	actor := -1
	for index, candidate := range s.Units {
		if candidate == u {
			actor = index
			break
		}
	}
	if actor < 0 {
		return nil, true, fmt.Errorf("native AI mode 2 actor is not in the runtime roster")
	}
	if len(s.nativeMovementCostRows) != NativeMovementCostRowCount {
		return nil, true, fmt.Errorf("native AI mode 2 movement table is unavailable")
	}
	if len(s.NativeTerrainMoveCodes) != s.W*s.H {
		return nil, true, fmt.Errorf("native AI mode 2 terrain move-code provenance is unavailable")
	}
	if len(s.NativeCompositionEventBytes) != s.W*s.H {
		return nil, true, fmt.Errorf("native AI mode 2 composition provenance is unavailable")
	}

	records, err := NativeAIScoringRecords(s.Units)
	if err != nil {
		return nil, true, err
	}
	actorRecord := records[actor*nativeRecordSize : (actor+1)*nativeRecordSize]
	item, found, err := ResolveNativeAIPhysicalItemSource(actorRecord, s.nativeFutureItemRows)
	if err != nil {
		return nil, true, err
	}
	if !found {
		// 0x1b83d 找不到已裝備低階物品時會進另一條原生收尾；該 recovery
		// consumer 尚未閉合，因此此處停止，不把「沒有物理候選」冒充成已完成待機。
		return nil, true, fmt.Errorf("native AI mode 2 equipped low-item source is unavailable")
	}
	selector := int(u.NativeRecordByte6)
	if selector != 0 && selector != 1 {
		return nil, true, fmt.Errorf("native AI mode 2 selector=%d is outside the verified 0/1 caller set", selector)
	}
	costRow, err := nativeAICostRowForRecord(actorRecord, s.nativeMovementCostRows)
	if err != nil {
		return nil, true, err
	}
	geometry := NativeAIPhysicalAttackGeometry{
		TargetMode:      int(item.RawGeometryByte0C),
		TargetInnerMark: int(item.RawGeometryByte0B),
		// 0x14237 sets this local to 1 for selector zero and 0 otherwise.
		TargetCode: nativeAIPhysicalTargetCode(selector),
	}
	baseFlags, err := NativeCompositionBaseFlags(s.W, s.H, s.NativeCompositionEventBytes)
	if err != nil {
		return nil, true, err
	}
	runtimeFlags, err := NativeCommandRuntimeFlags(
		s.W, s.H, s.NativeCompositionEventBytes, s.Units, byte(selector),
	)
	if err != nil {
		return nil, true, err
	}
	initialBudget := int(actorRecord[0x3b])
	candidates, err := BuildNativeAIPhysicalAttackCandidates(
		s.W, s.H, records, len(s.Units), actor, selector, initialBudget,
		// 0x14237 在每次 0x14818 目標掃描前以 0x4DBFC 重設 live grid；因此
		// 目標幾何看到的是不可變的 low5 base，而不是前段移動佔用寫入值。
		geometry, baseFlags, baseFlags, s.NativeTerrainMoveCodes, costRow,
		func(raw NativeAIPhysicalAttackRawCandidate) (NativePhysicalAttackScoreInput, error) {
			return s.nativeAIPhysicalScoreInput(raw, s.nativeFutureItemRows)
		},
	)
	if err != nil {
		return nil, true, err
	}
	selection, ok, err := SelectNativePhysicalAttackCandidate(candidates)
	if err != nil {
		return nil, true, err
	}
	if !ok {
		// 0x13C06/0x13C0F 在 0x14237 回傳零時消費 0x13FD4。接受恢復
		// 與合法的閘門拒絕都要保留，兩者皆不可掉入正規化規劃器（planner）。
		decision, err := PlanNativeAIIdleRecovery(records, len(s.Units), actor)
		if err != nil {
			return nil, true, fmt.Errorf("native AI mode 2 0x13fd4: %w", err)
		}
		plan := &AIPlan{
			U: u, SpellID: -1, NativeMode2Physical: true,
			NativeModeFallbackActive: true, NativeModeFallback: 2,
			NativeScoredCommands: s.nativeAIPlanScoredCommands(u),
		}
		if decision.Accepted {
			plan.NativeIdleRecovery = &decision
		}
		return plan, true, nil
	}
	selected := selection.Candidate
	pathDirections, reachable, err := NativePathDirections(
		s.W, s.H,
		Cell{X: int(actorRecord[0]), Y: int(actorRecord[1])},
		Cell{X: selected.DestinationX, Y: selected.DestinationY},
		initialBudget, 0, runtimeFlags, s.NativeTerrainMoveCodes, costRow,
	)
	if err != nil {
		return nil, true, err
	}
	if !reachable {
		return nil, true, fmt.Errorf("native AI mode 2 selected destination is not path-reachable")
	}
	path, err := nativeAIDirectionPath(
		Cell{X: int(actorRecord[0]), Y: int(actorRecord[1])}, pathDirections,
	)
	if err != nil {
		return nil, true, err
	}
	if selected.TargetIndex < 0 || selected.TargetIndex >= len(s.Units) || s.Units[selected.TargetIndex] == nil {
		return nil, true, fmt.Errorf("native AI mode 2 selected target index is invalid")
	}
	return &AIPlan{
		U:                    u,
		Path:                 path,
		Target:               s.Units[selected.TargetIndex],
		SpellID:              -1,
		NativeMode2Physical:  true,
		NativeScoredCommands: s.nativeAIPlanScoredCommands(u),
	}, true, nil
}

func nativeAIPhysicalTargetCode(selector int) int {
	if selector == 0 {
		return 1
	}
	return 0
}

func nativeAICostRowForRecord(record []byte, rows [][]byte) ([]byte, error) {
	if len(record) < nativeRecordSize || len(rows) != NativeMovementCostRowCount {
		return nil, fmt.Errorf("native AI movement row provenance is malformed")
	}
	selector := int(record[0x20])
	if nativeAIIsTerrainSpecialRecord(record) {
		selector = 19
	}
	if record[8] == 0x1c {
		selector = 1
	}
	if selector < 0 || selector >= len(rows) || len(rows[selector]) != NativeMovementCostRowSize {
		return nil, fmt.Errorf("native AI movement selector=%d is unavailable", selector)
	}
	return rows[selector], nil
}

func nativeAIIsTerrainSpecialRecord(record []byte) bool {
	return len(record) >= nativeRecordSize &&
		(record[7] == 0x1c || record[0x20] == 0x13 || record[0x1f] == 4 || record[0x1f] == 5)
}

func (s *State) nativeAIPhysicalScoreInput(
	raw NativeAIPhysicalAttackRawCandidate,
	itemRows []byte,
) (NativePhysicalAttackScoreInput, error) {
	if s == nil || len(raw.ActorRecord) != nativeRecordSize || len(raw.TargetRecord) != nativeRecordSize {
		return NativePhysicalAttackScoreInput{}, fmt.Errorf("native AI physical score record snapshot is malformed")
	}
	actor48 := int(binary.LittleEndian.Uint16(raw.ActorRecord[0x48:0x4a]))
	actor4a := int(binary.LittleEndian.Uint16(raw.ActorRecord[0x4a:0x4c]))
	target48 := int(binary.LittleEndian.Uint16(raw.TargetRecord[0x48:0x4a]))
	target4a := int(binary.LittleEndian.Uint16(raw.TargetRecord[0x4a:0x4c]))
	actor48, actor4a, err := s.nativeAITerrainAdjustedWords(raw.ActorRecord, raw.Destination, actor48, actor4a)
	if err != nil {
		return NativePhysicalAttackScoreInput{}, err
	}
	target48, target4a, err = s.nativeAITerrainAdjustedWords(raw.TargetRecord, raw.Destination, target48, target4a)
	if err != nil {
		return NativePhysicalAttackScoreInput{}, err
	}
	helper, err := nativeAIPhysicalHelper1DEBE(raw.TargetRecord, raw.Destination, itemRows)
	if err != nil {
		return NativePhysicalAttackScoreInput{}, err
	}
	return NativePhysicalAttackScoreInput{
		ActorWord48:          actor48,
		ActorWord4A:          actor4a,
		TargetWord48:         target48,
		TargetWord4A:         target4a,
		TargetWord40:         int(binary.LittleEndian.Uint16(raw.TargetRecord[0x40:0x42])),
		RawTargetByte8:       raw.TargetRecord[8],
		RawHelper1DEBEResult: helper,
	}, nil
}

func (s *State) nativeAITerrainAdjustedWords(record []byte, destination Cell, word48, word4a int) (int, int, error) {
	if !nativeAIIsTerrainSpecialRecord(record) {
		return word48, word4a, nil
	}
	if s == nil || destination.X < 0 || destination.Y < 0 || destination.X >= s.W || destination.Y >= s.H ||
		len(s.NativeTerrainMoveCodes) != s.W*s.H {
		return 0, 0, fmt.Errorf("native AI terrain score provenance is unavailable")
	}
	code := s.NativeTerrainMoveCodes[destination.Y*s.W+destination.X]
	apPct, dpPct, ok := NativeTerrainAPDPPct(code)
	if !ok {
		return 0, 0, fmt.Errorf("native AI terrain control byte %d is unknown", code)
	}
	word48 += word48 * apPct / 100
	word4a += word4a * dpPct / 100
	return word48, word4a, nil
}

func nativeAIPhysicalHelper1DEBE(targetRecord []byte, destination Cell, itemRows []byte) (int, error) {
	if len(targetRecord) != nativeRecordSize {
		return -1, fmt.Errorf("native AI 0x1debe target record is malformed")
	}
	if targetRecord[0x26] != 0 {
		return -1, nil
	}
	target := Cell{X: int(targetRecord[0]), Y: int(targetRecord[1])}
	if absInt(target.X-destination.X)+absInt(target.Y-destination.Y) != 1 {
		return -1, nil
	}
	source, found, err := ResolveNativeAIPhysicalItemSource(targetRecord, itemRows)
	if err != nil {
		return -1, err
	}
	if !found || source.RawGeometryByte0B > 1 {
		return -1, nil
	}
	return 1, nil
}

func nativeAIDirectionPath(start Cell, directions []byte) ([]Cell, error) {
	path := make([]Cell, 1, len(directions)+1)
	path[0] = start
	current := start
	for index, direction := range directions {
		switch direction {
		case NativePathDown:
			current.Y++
		case NativePathLeft:
			current.X--
		case NativePathUp:
			current.Y--
		case NativePathRight:
			current.X++
		default:
			return nil, fmt.Errorf("native AI path direction %d at step %d is unknown", direction, index)
		}
		path = append(path, current)
	}
	return path, nil
}
