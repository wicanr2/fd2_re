package battle

import (
	"encoding/binary"
	"fmt"
)

// nextNativeAI14EF0Plan is the first runtime consumer of the three raw
// producers called by 0x14ef0.  It is intentionally limited to records whose
// mode reaches that helper and to a complete command/item/movement export.
// The selected route remains an address-level label; no command or item name
// is inferred here.
func (s *State) nextNativeAI14EF0Plan(u *Unit) (*AIPlan, bool, error) {
	if s == nil || u == nil || !u.HasNativeRecordByte34 {
		return nil, false, nil
	}
	mode := int(u.NativeRecordByte34 & 0x0f)
	if !nativeAIModeUses14EF0(mode) {
		// Known modes that do not enter 0x14ef0 are handled by the separate raw
		// mode bridge below.  Returning unhandled here is intentional: the
		// caller must try that bridge before it is allowed to consider the
		// normalized planner.
		return nil, false, nil
	}
	if len(s.NativeCommandBook) != NativeCommandRecordCount || len(s.nativeFutureItemRows) == 0 {
		// Keep the previously proven mode-2 physical slice usable in small raw
		// fixtures that intentionally omit the command/item tables.  Production
		// states bind both tables before reaching this consumer.
		if mode == 2 {
			return nil, false, nil
		}
		return nil, true, fmt.Errorf("native AI 0x14ef0 command/item tables are unavailable")
	}
	if !u.HasNativeRecordByte6 || len(s.Units) == 0 {
		return nil, true, fmt.Errorf("native AI 0x14ef0 selector/runtime roster provenance is unavailable")
	}
	actor := -1
	for index, candidate := range s.Units {
		if candidate == u {
			actor = index
			break
		}
	}
	if actor < 0 {
		return nil, true, fmt.Errorf("native AI 0x14ef0 actor is not in the runtime roster")
	}
	selector := int(u.NativeRecordByte6)
	if selector != 0 && selector != 1 {
		return nil, true, fmt.Errorf("native AI 0x14ef0 selector=%d is outside the verified 0/1 caller set", selector)
	}
	if len(s.nativeMovementCostRows) != NativeMovementCostRowCount ||
		len(s.NativeTerrainMoveCodes) != s.W*s.H ||
		len(s.NativeCompositionEventBytes) != s.W*s.H {
		return nil, true, fmt.Errorf("native AI 0x14ef0 movement/terrain/composition provenance is unavailable")
	}
	records, err := NativeAIScoringRecords(s.Units)
	if err != nil {
		return nil, true, err
	}
	actorRecord := records[actor*nativeRecordSize : (actor+1)*nativeRecordSize]
	baseFlags, err := NativeCompositionBaseFlags(s.W, s.H, s.NativeCompositionEventBytes)
	if err != nil {
		return nil, true, err
	}
	costRow, err := nativeAICostRowForRecord(actorRecord, s.nativeMovementCostRows)
	if err != nil {
		return nil, true, err
	}

	physical, physicalOK, err := s.nativeAI14EF0PhysicalSelection(
		actor, selector, records, actorRecord, baseFlags, costRow,
	)
	if err != nil {
		return nil, true, err
	}
	command, err := ScoreNativeAI1598A(
		s.W, s.H, records, len(s.Units), actor, selector, u,
		s.NativeCommandBook, baseFlags, s.NativeTerrainMoveCodes,
		costRow, nil,
	)
	if err != nil {
		return nil, true, err
	}
	item, err := ScoreNativeAI1567E(
		s.W, s.H, records, len(s.Units), actor, selector,
		s.nativeFutureItemRows, s.NativeCommandBook, baseFlags,
	)
	if err != nil {
		return nil, true, err
	}

	// 0x14ef0 always reads the target word from the raw pair selected by
	// 0x14237 ([0x53c4b]), even when a later command/item route wins.  Keep that
	// pointer separate from each producer's own target; choosing a command
	// target here would silently change the tie predicate at 0x14f5d.
	targetIndex := -1
	if physicalOK {
		targetIndex = physical.Candidate.TargetIndex
	} else if command.HasPositiveWinner && len(command.PositiveWinner.TargetIndices) > 0 {
		// A zero C4F producer still leaves the raw target word read by the
		// dispatcher; in a detached fixture the only available provenance is
		// the command candidate, so retain that limitation explicitly.
		targetIndex = int(command.PositiveWinner.TargetIndices[0])
	} else if item.HasPositiveWinner && len(item.TargetIndices) > 0 {
		targetIndex = int(item.TargetIndices[0])
	}
	if targetIndex < 0 || targetIndex >= len(s.Units) || s.Units[targetIndex] == nil {
		return nil, true, fmt.Errorf("native AI 0x14ef0 selected target provenance is unavailable")
	}
	targetRecord := records[targetIndex*nativeRecordSize : (targetIndex+1)*nativeRecordSize]
	in := NativeAI14EF0Input{
		ScoreC4F: int32(0), ScoreC23: int32(command.MaxScore), ScoreC33: int32(item.MaxScore),
		Record34:       u.NativeRecordByte34,
		ActorWord48:    binary.LittleEndian.Uint16(actorRecord[0x48:0x4a]),
		TargetWord4A:   binary.LittleEndian.Uint16(targetRecord[0x4a:0x4c]),
		HasRawScoreC4F: true, HasRawScoreC23: true, HasRawScoreC33: true,
		HasRawRecord34: true, HasRawActorWord48: true, HasRawTargetWord4A: true,
	}
	if physicalOK {
		// sub_14237 writes priority (8/18) to [0x53c4f]; its larger
		// per-candidate arithmetic remains the separate tie word [0x53c43].
		in.ScoreC4F = int32(physical.Ranking.Priority)
	}
	if command.HasPositiveWinner {
		id := command.PositiveWinner.CommandID
		if id < 0 || id >= len(s.NativeCommandBook) {
			return nil, true, fmt.Errorf("native AI 0x14ef0 selected command id=%d is unavailable", id)
		}
		in.CommandID = int32(id)
		in.CommandWord = uint16(s.NativeCommandBook[id].Damage)
		in.HasRawCommandID, in.HasRawCommandWord = true, true
	}
	route, err := SelectNativeAI14EF0Tail(in)
	if err != nil {
		return nil, true, err
	}
	switch route {
	case NativeAI14EF0Call1548E:
		if !physicalOK {
			return nil, true, fmt.Errorf("native AI 0x14ef0 selected 0x1548e without physical candidate")
		}
		plan, err := s.nativeAIPlanForDestination(
			u, actorRecord, selector,
			Cell{X: physical.Candidate.DestinationX, Y: physical.Candidate.DestinationY},
			physical.Candidate.TargetIndex, costRow,
		)
		if err != nil {
			return nil, true, err
		}
		plan.NativeActionKind = NativeAIActionPhysical
		plan.NativeMode2Physical = mode == 2
		plan.NativeAI14EF0Route = route
		plan.NativeActionScore = physical.Ranking.Priority
		return plan, true, nil
	case NativeAI14EF0Call15311:
		if !command.HasPositiveWinner || len(command.PositiveWinner.TargetIndices) == 0 {
			return nil, true, fmt.Errorf("native AI 0x14ef0 selected 0x15311 without command target")
		}
		plan, err := s.nativeAIPlanForDestination(
			u, actorRecord, selector,
			Cell{X: command.PositiveWinner.X, Y: command.PositiveWinner.Y},
			int(command.PositiveWinner.TargetIndices[0]), costRow,
		)
		if err != nil {
			return nil, true, err
		}
		plan.NativeActionKind = NativeAIActionCommand
		plan.NativeCommandID = command.PositiveWinner.CommandID
		plan.NativeAI14EF0Route = route
		plan.NativeActionScore = command.MaxScore
		return plan, true, nil
	case NativeAI14EF0Call15055:
		if !item.HasPositiveWinner || len(item.TargetIndices) == 0 {
			return nil, true, fmt.Errorf("native AI 0x14ef0 selected 0x15055 without item target")
		}
		plan, err := s.nativeAIPlanForDestination(
			u, actorRecord, selector,
			Cell{X: item.X, Y: item.Y}, int(item.TargetIndices[0]), costRow,
		)
		if err != nil {
			return nil, true, err
		}
		plan.NativeActionKind = NativeAIActionItem
		plan.NativeItemSlot, plan.NativeItemID = item.InventorySlot, item.ItemID
		plan.NativeAI14EF0Route = route
		plan.NativeActionScore = item.MaxScore
		return plan, true, nil
	default:
		// A no-tail result is the native mode's documented fallback boundary;
		// it is not evidence for an arbitrary physical attack.  Let the raw
		// mode bridge consume it, or fail closed if that mode is not closed.
		return nil, false, nil
	}
}

func nativeAIModeUses14EF0(mode int) bool {
	switch mode {
	case 0, 1, 2, 3, 5, 9, 10:
		return true
	default:
		return false
	}
}

func (s *State) nativeAI14EF0PhysicalSelection(
	actor, selector int, records, actorRecord, baseFlags, costRow []byte,
) (NativePhysicalAttackSelection, bool, error) {
	item, found, err := ResolveNativeAIPhysicalItemSource(actorRecord, s.nativeFutureItemRows)
	if err != nil {
		return NativePhysicalAttackSelection{}, false, err
	}
	if !found {
		return NativePhysicalAttackSelection{}, false, nil
	}
	geometry := NativeAIPhysicalAttackGeometry{
		TargetMode: int(item.RawGeometryByte0C), TargetInnerMark: int(item.RawGeometryByte0B),
		TargetCode: nativeAIPhysicalTargetCode(selector),
	}
	flags, err := NativeCommandRuntimeFlags(
		s.W, s.H, s.NativeCompositionEventBytes, s.Units, byte(selector),
	)
	if err != nil {
		return NativePhysicalAttackSelection{}, false, err
	}
	candidates, err := BuildNativeAIPhysicalAttackCandidates(
		s.W, s.H, records, len(s.Units), actor, selector,
		int(actorRecord[0x3b]), geometry, flags, baseFlags,
		s.NativeTerrainMoveCodes, costRow,
		func(raw NativeAIPhysicalAttackRawCandidate) (NativePhysicalAttackScoreInput, error) {
			return s.nativeAIPhysicalScoreInput(raw, item.Row)
		},
	)
	if err != nil {
		return NativePhysicalAttackSelection{}, false, err
	}
	selection, ok, err := SelectNativePhysicalAttackCandidate(candidates)
	return selection, ok, err
}

// nativeAIPlanForDestination replays 0x14b78's movement transaction for a
// selected raw destination.  The caller still owns the effect/target stage;
// this helper only returns the detached path and verified target pointer.
func (s *State) nativeAIPlanForDestination(
	u *Unit, actorRecord []byte, selector int, destination Cell,
	targetIndex int, costRow []byte,
) (*AIPlan, error) {
	if u == nil || len(actorRecord) != nativeRecordSize || targetIndex < 0 || targetIndex >= len(s.Units) ||
		s.Units[targetIndex] == nil {
		return nil, fmt.Errorf("native AI selected destination/target is malformed")
	}
	runtimeFlags, err := NativeCommandRuntimeFlags(
		s.W, s.H, s.NativeCompositionEventBytes, s.Units, byte(selector),
	)
	if err != nil {
		return nil, err
	}
	directions, reachable, err := NativePathDirections(
		s.W, s.H,
		Cell{X: int(actorRecord[0]), Y: int(actorRecord[1])}, destination,
		int(actorRecord[0x3b]), 0, runtimeFlags, s.NativeTerrainMoveCodes, costRow,
	)
	if err != nil {
		return nil, err
	}
	if !reachable {
		return nil, fmt.Errorf("native AI selected destination is not path-reachable")
	}
	path, err := nativeAIDirectionPath(Cell{X: int(actorRecord[0]), Y: int(actorRecord[1])}, directions)
	if err != nil {
		return nil, err
	}
	return &AIPlan{
		U: u, Path: path, Target: s.Units[targetIndex], SpellID: -1,
		NativeScoredCommands:    s.nativeAIPlanScoredCommands(u),
		NativeActionDestination: destination,
	}, nil
}

// nextNativeAIModeFallbackPlan consumes only the mode branches whose inputs
// are explicit in the 0x13a9f disassembly.  Mode 5 now owns the raw
// 0x15df3/event-row state tail; its indexed presentation calls remain a
// separate gate. Mode 11's two-action transaction and the 0x13fd4 raw HP
// write still have no complete runtime owner here.
func (s *State) nextNativeAIModeFallbackPlan(u *Unit) (*AIPlan, bool, error) {
	if s == nil || u == nil || !u.HasNativeRecordByte34 {
		return nil, false, nil
	}
	mode := int(u.NativeRecordByte34 & 0x0f)
	switch mode {
	case 0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11:
		// These low nibbles are present in the fixed FDFIELD inventory or in
		// an explicitly recovered writer.  They must not fall through to the
		// normalized planner when their raw consumer is incomplete.
	default:
		return nil, true, fmt.Errorf("native AI mode %d has no recovered dispatcher branch", mode)
	}
	if mode == 8 {
		// 0x13a9f branches directly to its common completion path for mode 8;
		// no movement or effect is emitted by that branch.
		return &AIPlan{U: u, SpellID: -1, NativeModeFallbackActive: true, NativeModeFallback: byte(mode)}, true, nil
	}
	if mode == 2 || mode == 11 {
		return nil, true, fmt.Errorf("native AI mode %d fallback consumer remains unresolved", mode)
	}
	actor, selector, records, actorRecord, baseFlags, costRow, err := s.nativeAIModeRuntimeContext(u)
	if err != nil {
		return nil, true, err
	}
	if mode == 5 {
		if !u.HasNativeRecordByte3D || int(u.NativeRecordByte3D) >= len(s.NativeEventState) {
			return nil, true, fmt.Errorf("native AI mode 5 event-state index provenance is unavailable")
		}
		if !s.HasNativeFieldControlState || len(s.NativeFieldControlRaw) < 0x56+3*int(u.NativeRecordByte3D) {
			return nil, true, fmt.Errorf("native AI mode 5 event-control row provenance is unavailable")
		}
		if s.NativeEventState[u.NativeRecordByte3D] != 0 {
			return nil, true, fmt.Errorf("native AI mode 5 event state is already set; mode-0 recovery owner is unavailable")
		}
		intended, err := s.NativeAIMode5EventCell(u.NativeRecordByte3D)
		if err != nil {
			return nil, true, err
		}
		plan, err := s.nativeAIPlanTowardRawDestination(
			u, actor, selector, records, actorRecord, baseFlags, costRow, intended,
		)
		if err != nil {
			return nil, true, err
		}
		if plan.NativeActionDestination != intended {
			return nil, true, fmt.Errorf("native AI mode 5 event destination was substituted")
		}
		plan.NativeModeFallbackActive = true
		plan.NativeModeFallback = byte(mode)
		plan.NativeModeWriteRangeZero = true
		plan.NativeModeEventActive = true
		plan.NativeModeEventID = u.NativeRecordByte3D
		plan.NativeModeEventDestination = intended
		return plan, true, nil
	}
	if mode == 3 || mode == 9 {
		// 0x12c60 receives raw record +0x35 and returns the first live record
		// whose raw +0x08 matches it.  It is not a nearest-unit search and the
		// returned record's coordinates remain the only accepted destination.
		targetIndex, ok := nativeAIRawRecord8Index(records, len(s.Units), int(u.NativeRecordByte35))
		if !ok {
			return nil, true, fmt.Errorf("native AI mode %d 0x12c60 found no live raw +0x08 target", mode)
		}
		targetRecord := records[targetIndex*nativeRecordSize:]
		intended := Cell{X: int(targetRecord[0]), Y: int(targetRecord[1])}
		if intended.X < 0 || intended.Y < 0 || intended.X >= s.W || intended.Y >= s.H {
			return nil, true, fmt.Errorf("native AI mode %d 0x12c60 target is outside the map", mode)
		}
		plan, err := s.nativeAIPlanTowardRawDestination(
			u, actor, selector, records, actorRecord, baseFlags, costRow, intended,
		)
		if err != nil {
			return nil, true, err
		}
		plan.NativeModeFallbackActive = true
		plan.NativeModeFallback = byte(mode)
		// mode 3 explicitly clears [0x51a83] after 0x14b78.  Mode 9 jumps
		// through the shared 0x14b78 block without that explicit caller write.
		plan.NativeModeWriteRangeZero = mode == 3
		return plan, true, nil
	}
	if mode == 4 || mode == 7 || mode == 10 {
		intended := Cell{X: int(u.NativeRecordByte35), Y: int(u.NativeRecordByte36)}
		if intended.X < 0 || intended.Y < 0 || intended.X >= s.W || intended.Y >= s.H {
			return nil, true, fmt.Errorf("native AI mode %d raw destination (%d,%d) is outside the map", mode, intended.X, intended.Y)
		}
		plan, err := s.nativeAIPlanTowardRawDestination(
			u, actor, selector, records, actorRecord, baseFlags, costRow, intended,
		)
		if err != nil {
			return nil, true, err
		}
		plan.NativeModeFallbackActive = true
		plan.NativeModeFallback = byte(mode)
		plan.NativeModeWriteByte5 = mode == 7 && plan.NativeActionDestination == intended
		plan.NativeModeWriteRangeZero = true
		return plan, true, nil
	}

	// Modes 0/1 first call 0x14121 with the raw opposite-group blocker grid.
	// A successful blocked coordinate is only useful when it maps back to one
	// and only one active raw record; otherwise the target identity would be a
	// normalized guess.
	flags, err := nativeAIModeBlockedSearchFlags(s.W, s.H, records, len(s.Units), selector, baseFlags)
	if err != nil {
		return nil, true, err
	}
	blocked, found, err := NativePathBlockedCoordinate(
		s.W, s.H, Cell{X: int(actorRecord[0]), Y: int(actorRecord[1])}, 28,
		flags, s.NativeTerrainMoveCodes, costRow,
	)
	if err != nil {
		return nil, true, err
	}
	if found {
		if _, ok := nativeAIActiveUnitAtRawCell(records, len(s.Units), selector, blocked); ok {
			plan, err := s.nativeAIPlanTowardRawDestination(
				u, actor, selector, records, actorRecord, baseFlags, costRow, blocked,
			)
			if err != nil {
				return nil, true, err
			}
			plan.NativeModeFallbackActive = true
			plan.NativeModeFallback = byte(mode)
			// 0x14121/0x13e9c are movement-only callers; target is retained
			// only as raw provenance for tests and is not executed as an attack.
			plan.Target = nil
			return plan, true, nil
		}
	}
	if mode == 1 {
		return nil, true, fmt.Errorf("native AI mode 1 0x14121 produced no raw target; 0x13fd4 recovery is unavailable")
	}
	nearest, found, err := SelectNativeNearestOppositeCoordinate(
		records, len(s.Units), actor, selector,
	)
	if err != nil {
		return nil, true, err
	}
	if !found {
		return nil, true, fmt.Errorf("native AI mode 0 0x13e9c produced no raw target; 0x13fd4 recovery is unavailable")
	}
	plan, err := s.nativeAIPlanTowardRawDestination(
		u, actor, selector, records, actorRecord, baseFlags, costRow, nearest,
	)
	if err != nil {
		return nil, true, err
	}
	plan.NativeModeFallbackActive = true
	plan.NativeModeFallback = byte(mode)
	plan.Target = nil
	return plan, true, nil
}

// nativeAIRawRecord8Index mirrors 0x12c60's first-match scan.  The caller
// supplies the complete scoring-record export, so a missing +0x08 provenance
// is rejected by NativeAIScoringRecords before this helper is reached.
func nativeAIRawRecord8Index(records []byte, count, rawKey int) (int, bool) {
	if count < 0 || len(records) != count*nativeRecordSize || rawKey < 0 || rawKey > 0xff {
		return -1, false
	}
	for index := 0; index < count; index++ {
		record := records[index*nativeRecordSize:]
		if record[5]&1 != 0 || int(record[8]) != rawKey {
			continue
		}
		return index, true
	}
	return -1, false
}

func (s *State) nativeAIModeRuntimeContext(u *Unit) (int, int, []byte, []byte, []byte, []byte, error) {
	if s == nil || u == nil || !u.HasNativeRecordByte6 {
		return 0, 0, nil, nil, nil, nil, fmt.Errorf("native AI mode lacks selector provenance")
	}
	selector := int(u.NativeRecordByte6)
	if selector != 0 && selector != 1 {
		return 0, 0, nil, nil, nil, nil, fmt.Errorf("native AI mode selector=%d is outside the verified 0/1 caller set", selector)
	}
	if len(s.nativeMovementCostRows) != NativeMovementCostRowCount ||
		len(s.NativeTerrainMoveCodes) != s.W*s.H || len(s.NativeCompositionEventBytes) != s.W*s.H {
		return 0, 0, nil, nil, nil, nil, fmt.Errorf("native AI mode movement/terrain/composition provenance is unavailable")
	}
	records, err := NativeAIScoringRecords(s.Units)
	if err != nil {
		return 0, 0, nil, nil, nil, nil, err
	}
	actor := -1
	for index, candidate := range s.Units {
		if candidate == u {
			actor = index
			break
		}
	}
	if actor < 0 {
		return 0, 0, nil, nil, nil, nil, fmt.Errorf("native AI mode actor is not in the runtime roster")
	}
	actorRecord := records[actor*nativeRecordSize : (actor+1)*nativeRecordSize]
	costRow, err := nativeAICostRowForRecord(actorRecord, s.nativeMovementCostRows)
	if err != nil {
		return 0, 0, nil, nil, nil, nil, err
	}
	baseFlags, err := NativeCompositionBaseFlags(s.W, s.H, s.NativeCompositionEventBytes)
	if err != nil {
		return 0, 0, nil, nil, nil, nil, err
	}
	return actor, selector, records, actorRecord, baseFlags, costRow, nil
}

func nativeAIModeBlockedSearchFlags(w, h int, records []byte, count, selector int, baseFlags []byte) ([]byte, error) {
	if w <= 0 || h <= 0 || len(baseFlags) != w*h || count < 0 || len(records) != count*nativeRecordSize {
		return nil, fmt.Errorf("native AI mode blocker grid is malformed")
	}
	flags := append([]byte(nil), baseFlags...)
	for index := 0; index < count; index++ {
		record := records[index*nativeRecordSize:]
		if record[5]&1 != 0 || !nativeAIOppositeSelectorGroup(record[6], selector) {
			continue
		}
		x, y := int(record[0]), int(record[1])
		if x < 0 || y < 0 || x >= w || y >= h {
			return nil, fmt.Errorf("native AI mode blocker unit %d is outside the map", index)
		}
		flags[y*w+x] |= NativeCommandGridBlocked
		for _, delta := range nativeAIDestinationNeighbours {
			nx, ny := x+delta[0], y+delta[1]
			if nx >= 0 && ny >= 0 && nx < w && ny < h {
				flags[ny*w+nx] |= NativeCommandGridZeroBudget
			}
		}
	}
	return flags, nil
}

func nativeAIActiveUnitAtRawCell(records []byte, count, selector int, cell Cell) (int, bool) {
	for index := 0; index < count; index++ {
		record := records[index*nativeRecordSize:]
		if record[5]&1 != 0 || !nativeAIOppositeSelectorGroup(record[6], selector) {
			continue
		}
		if int(record[0]) == cell.X && int(record[1]) == cell.Y {
			return index, true
		}
	}
	return -1, false
}

func (s *State) nativeAIPlanTowardRawDestination(
	u *Unit, actor, selector int, records, actorRecord, baseFlags, costRow []byte, intended Cell,
) (*AIPlan, error) {
	if s == nil || u == nil || len(actorRecord) != nativeRecordSize {
		return nil, fmt.Errorf("native AI mode destination context is malformed")
	}
	candidates, err := NativeAIPhysicalDestinations(
		s.W, s.H, records, len(s.Units), actor, selector, int(actorRecord[0x3b]),
		baseFlags, s.NativeTerrainMoveCodes, costRow,
	)
	if err != nil {
		return nil, err
	}
	destination, ok := SelectNativeMovementDestination(candidates, intended)
	if !ok {
		return nil, fmt.Errorf("native AI mode destination has no reachable cell")
	}
	runtimeFlags, err := NativeCommandRuntimeFlags(
		s.W, s.H, s.NativeCompositionEventBytes, s.Units, byte(selector),
	)
	if err != nil {
		return nil, err
	}
	directions, reachable, err := NativePathDirections(
		s.W, s.H, Cell{X: int(actorRecord[0]), Y: int(actorRecord[1])}, destination,
		int(actorRecord[0x3b]), 0, runtimeFlags, s.NativeTerrainMoveCodes, costRow,
	)
	if err != nil {
		return nil, err
	}
	if !reachable {
		return nil, fmt.Errorf("native AI mode destination is not path-reachable")
	}
	path, err := nativeAIDirectionPath(Cell{X: int(actorRecord[0]), Y: int(actorRecord[1])}, directions)
	if err != nil {
		return nil, err
	}
	return &AIPlan{
		U: u, Path: path, SpellID: -1, NativeActionDestination: destination,
		NativeScoredCommands: s.nativeAIPlanScoredCommands(u),
	}, nil
}
