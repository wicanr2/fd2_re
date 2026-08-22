package battle

import "fmt"

// NativeSystemGroupMarchStep 保存 sub_16F55 selector1 對一筆合格 runtime
// record 的可重播結果。Path 含起點與終點；本切片只接受不觸發全域事件的路徑。
type NativeSystemGroupMarchStep struct {
	UnitIndex int
	Path      []Cell
	Events    []NativeSystemGroupMarchEvent
}

// NativeSystemGroupMarchEvent 保存一個已在私有投影驗證的 selector1 中途事件。
// PathIndex 指向 Path 內事件格；TextIndex／Presentation 只供正式 UI owner 在
// 關閉確認框前驗證既有可編輯素材，不替 handler 新增語意。
type NativeSystemGroupMarchEvent struct {
	PathIndex    int
	EventID      byte
	TextIndex    int
	Presentation bool
}

// NativeSystemGroupMarchPlan 是確認視窗關閉前完成的私有預演結果。
type NativeSystemGroupMarchPlan struct {
	Destination Cell
	Steps       []NativeSystemGroupMarchStep
}

// PlanNativeSystemGroupMarch 重現 0x16FF4..0x1716A 的無事件產品子集：依
// runtime record 順序選 (raw+5)&0x85==0、raw+6==2 的單位，逐筆以 selector1
// 執行已閉合的 0x14B78 尋路。占位會在私有投影逐筆更新，任一未知事件或來源
// 缺失都讓整批失敗，不發布部分移動。
func (s *State) PlanNativeSystemGroupMarch(destination Cell) (NativeSystemGroupMarchPlan, error) {
	plan := NativeSystemGroupMarchPlan{Destination: destination}
	if s == nil || s.W <= 0 || s.H <= 0 || destination.X < 0 || destination.X >= s.W ||
		destination.Y < 0 || destination.Y >= s.H || len(s.Units) == 0 ||
		len(s.nativeMovementCostRows) != NativeMovementCostRowCount ||
		len(s.NativeTerrainMoveCodes) != s.W*s.H || len(s.NativeCompositionEventBytes) != s.W*s.H {
		return plan, fmt.Errorf("battle: native system group-march provenance is unavailable")
	}
	working, err := cloneNativeSystemGroupMarchState(s)
	if err != nil {
		return plan, err
	}
	// dword_53BEB 是每輪重讀的動態上限；event61 追加的 group1 record
	// 因此也會在同一批後續接受 raw gate，而不是被確認時的 len 快照截斷。
	for index := 0; index < len(working.Units); index++ {
		unit := working.Units[index]
		if !unit.HasNativeRecordByte5 || !unit.HasNativeRecordByte6 {
			return NativeSystemGroupMarchPlan{Destination: destination}, fmt.Errorf(
				"battle: native system group-march unit %d lacks raw gate provenance", index,
			)
		}
		if unit.NativeRecordByte5&0x85 != 0 || unit.NativeRecordByte6 != 2 {
			continue
		}
		records, err := NativeAIScoringRecords(working.Units)
		if err != nil {
			return NativeSystemGroupMarchPlan{Destination: destination}, err
		}
		actorRecord := records[index*nativeRecordSize : (index+1)*nativeRecordSize]
		costRow, err := nativeAICostRowForRecord(actorRecord, working.nativeMovementCostRows)
		if err != nil {
			return NativeSystemGroupMarchPlan{Destination: destination}, err
		}
		baseFlags, err := NativeCompositionBaseFlags(
			working.W, working.H, working.NativeCompositionEventBytes,
		)
		if err != nil {
			return NativeSystemGroupMarchPlan{Destination: destination}, err
		}
		movement, err := working.nativeAIPlanTowardRawDestination(
			unit, index, 1, records, actorRecord, baseFlags, costRow, destination,
		)
		if err != nil {
			return NativeSystemGroupMarchPlan{Destination: destination}, err
		}
		if len(movement.Path) == 0 {
			return NativeSystemGroupMarchPlan{Destination: destination}, fmt.Errorf(
				"battle: native system group-march unit %d produced no path", index,
			)
		}
		step := NativeSystemGroupMarchStep{
			UnitIndex: index, Path: append([]Cell(nil), movement.Path...),
		}
		for segment := 0; segment+1 < len(movement.Path); segment++ {
			a, b := movement.Path[segment], movement.Path[segment+1]
			unit.X, unit.Y = b.X, b.Y
			unit.NativeMapPresentation.X = byte(b.X)
			unit.NativeMapPresentation.Y = byte(b.Y)
			if b.X != a.X-1 || b.Y != a.Y {
				continue
			}
			if eventID, ok := NativeFieldEventIDAt(&working, b.X, b.Y, 1); ok {
				event, err := planNativeSystemGroupMarchEvent(&working, unit, segment+1, eventID)
				if err != nil {
					return NativeSystemGroupMarchPlan{Destination: destination}, err
				}
				step.Events = append(step.Events, event)
			}
		}
		plan.Steps = append(plan.Steps, step)
		unit.NativeRecordByte5 |= 0x80
		unit.Acted = true
	}
	if len(plan.Steps) == 0 {
		return plan, fmt.Errorf("battle: native system group-march has no eligible unit")
	}
	return plan, nil
}

func cloneNativeSystemGroupMarchUnit(unit *Unit) *Unit {
	if unit == nil {
		return nil
	}
	clone := *unit
	clone.Inventory = append([]int(nil), unit.Inventory...)
	clone.Equipped = append([]bool(nil), unit.Equipped...)
	clone.InventorySlots = append([]int(nil), unit.InventorySlots...)
	clone.NativeInventoryFlags = append([]int(nil), unit.NativeInventoryFlags...)
	clone.Spells = append([]int(nil), unit.Spells...)
	return &clone
}

func cloneNativeSystemGroupMarchState(source *State) (State, error) {
	working := *source
	if source.NativeMapSelectorCache != nil {
		working.NativeMapSelectorCache = source.NativeMapSelectorCache.Clone()
	}
	working.Units = make([]*Unit, len(source.Units))
	for index, unit := range source.Units {
		if unit == nil {
			return State{}, fmt.Errorf("battle: native system group-march unit %d is nil", index)
		}
		working.Units[index] = cloneNativeSystemGroupMarchUnit(unit)
	}
	working.Roster = make([]*Unit, len(source.Roster))
	for index, unit := range source.Roster {
		if unit == nil {
			return State{}, fmt.Errorf("battle: native system group-march roster %d is nil", index)
		}
		working.Roster[index] = cloneNativeSystemGroupMarchUnit(unit)
	}
	working.NativeFieldControlRaw = append([]byte(nil), source.NativeFieldControlRaw...)
	return working, nil
}

func planNativeSystemGroupMarchEvent(
	working *State,
	trigger *Unit,
	pathIndex int,
	eventID byte,
) (NativeSystemGroupMarchEvent, error) {
	event := NativeSystemGroupMarchEvent{PathIndex: pathIndex, EventID: eventID}
	switch eventID {
	case 61:
		plan, err := PlanNativeFieldEvent61(working, trigger, trigger.X, trigger.Y)
		if err != nil {
			return event, fmt.Errorf("battle: native system group-march event61: %w", err)
		}
		event.TextIndex = plan.TextIndex
		event.Presentation = !plan.MissingItem
		if !plan.MissingItem {
			if _, err := CommitNativeFieldEvent61(working, plan, plan.requiredFrames); err != nil {
				return event, fmt.Errorf("battle: native system group-march event61 projection: %w", err)
			}
		}
	case 75:
		plan, err := PlanNativeFieldEvent75(working, trigger, trigger.X, trigger.Y)
		if err != nil {
			return event, fmt.Errorf("battle: native system group-march event75: %w", err)
		}
		event.TextIndex = plan.TextIndex
		if plan.Activate {
			if err := CommitNativeFieldEvent75(working, plan); err != nil {
				return event, fmt.Errorf("battle: native system group-march event75 projection: %w", err)
			}
		}
	default:
		return event, fmt.Errorf(
			"battle: native system group-march event %d has no atomic runtime owner", eventID,
		)
	}
	return event, nil
}

// CommitNativeSystemGroupMarchStep 原子發布一筆已預演移動；動畫 owner 必須在
// path 完整呈現後呼叫。它對應 selector1 每筆的姿態清零與 0x13512 bit7 writer。
func (s *State) CommitNativeSystemGroupMarchStep(step NativeSystemGroupMarchStep) error {
	if s == nil || step.UnitIndex < 0 || step.UnitIndex >= len(s.Units) ||
		s.Units[step.UnitIndex] == nil || len(step.Path) == 0 {
		return fmt.Errorf("battle: native system group-march step is malformed")
	}
	unit := s.Units[step.UnitIndex]
	start, end := step.Path[0], step.Path[len(step.Path)-1]
	if !unit.HasNativeMapPresentation || !unit.HasNativeRecordByte5 || !unit.HasNativeRecordByte6 ||
		unit.NativeRecordByte5&0x85 != 0 || unit.NativeRecordByte6 != 2 {
		return fmt.Errorf("battle: native system group-march step no longer matches runtime state")
	}
	atStart := unit.X == start.X && unit.Y == start.Y &&
		int(unit.NativeMapPresentation.X) == start.X && int(unit.NativeMapPresentation.Y) == start.Y
	atEnd := unit.X == end.X && unit.Y == end.Y &&
		int(unit.NativeMapPresentation.X) == end.X && int(unit.NativeMapPresentation.Y) == end.Y
	if !atStart && !atEnd {
		return fmt.Errorf("battle: native system group-march step no longer matches runtime position")
	}
	unit.X, unit.Y = end.X, end.Y
	unit.NativeMapPresentation.X = byte(end.X)
	unit.NativeMapPresentation.Y = byte(end.Y)
	unit.NativeMapPresentation.Pose = 0
	unit.NativeRecordByte5 |= 0x80
	unit.Acted = true
	return nil
}
