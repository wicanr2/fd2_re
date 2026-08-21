package battle

import "fmt"

type NativeFieldEvent75Plan struct {
	EventID   byte
	TextIndex int
	Activate  bool
	Noop      bool

	trigger *Unit
	rule    *NativeFieldEventRule
}

// PlanNativeFieldEvent75 驗證 map28 selector1 handler，不修改 event-state table
// 或 live turn row。raw +8 != 9 選 FDTXT_029 index0，執行期另以觸發 record +7
// 呈現肖像；raw +8 == 9 選 index1，並在對話完成後才可啟動 dormant rows。
func PlanNativeFieldEvent75(st *State, trigger *Unit, x, y int) (NativeFieldEvent75Plan, error) {
	if st == nil || trigger == nil || trigger.X != x || trigger.Y != y ||
		!st.HasNativeTurnEventControlState {
		return NativeFieldEvent75Plan{}, fmt.Errorf("event75: incomplete trigger or live turn controls")
	}
	eventID, ok := NativeFieldEventIDAt(st, x, y, 1)
	if !ok || eventID != 75 {
		return NativeFieldEvent75Plan{}, fmt.Errorf("event75: selector1 binding is absent")
	}
	var rule *NativeFieldEventRule
	for i := range st.NativeFieldEventRules {
		candidate := &st.NativeFieldEventRules[i]
		if candidate.EventID == 75 && candidate.Selector == 1 {
			rule = candidate
			break
		}
	}
	wantWrites := []NativeEventStateWrite{{Index: 17, Value: 1}, {Index: 16, Value: 4}}
	wantTurns := []NativeTurnActivation{
		{Slot: 1, EventID: 76, RawCamp: 2, TurnDelta: 1},
		{Slot: 0, EventID: 74, RawCamp: 0, TurnDelta: 0},
	}
	if rule == nil || rule.TriggerGate != "record_byte6_nonzero" || rule.TurnChain == nil ||
		rule.TurnChain.Handler != "0x35c79" || rule.TurnChain.TriggerRecordByte8 != 9 ||
		rule.TurnChain.MismatchTextIndex != 0 || rule.TurnChain.SuccessTextIndex != 1 ||
		len(rule.TurnChain.StateWrites) != len(wantWrites) ||
		len(rule.TurnChain.TurnActivations) != len(wantTurns) {
		return NativeFieldEvent75Plan{}, fmt.Errorf("event75: editable rule is incomplete")
	}
	for i := range wantWrites {
		if rule.TurnChain.StateWrites[i] != wantWrites[i] {
			return NativeFieldEvent75Plan{}, fmt.Errorf("event75: state write %d differs from handler", i)
		}
	}
	for i := range wantTurns {
		if rule.TurnChain.TurnActivations[i] != wantTurns[i] {
			return NativeFieldEvent75Plan{}, fmt.Errorf("event75: turn activation %d differs from handler", i)
		}
	}
	if !trigger.HasNativeRecordByte6 || !trigger.HasNativeRecordByte8 {
		return NativeFieldEvent75Plan{}, fmt.Errorf("event75: raw +6/+8 provenance is absent")
	}
	plan := NativeFieldEvent75Plan{EventID: eventID, trigger: trigger, rule: rule}
	if trigger.NativeRecordByte6 == 0 || st.NativeEventState[17] != 0 {
		plan.Noop = true
		return plan, nil
	}
	if trigger.NativeRecordByte8 != rule.TurnChain.TriggerRecordByte8 {
		plan.TextIndex = rule.TurnChain.MismatchTextIndex
		return plan, nil
	}
	if st.NativeRoundCounter <= 0 || st.NativeRoundCounter > 0xfe {
		return NativeFieldEvent75Plan{}, fmt.Errorf("event75: native round is unavailable")
	}
	for _, activation := range rule.TurnChain.TurnActivations {
		row := st.NativeTurnEventControls[activation.Slot]
		if row != (NativeTurnEventControl{Turn: 0xff, EventID: byte(activation.EventID), RawCamp: activation.RawCamp}) {
			return NativeFieldEvent75Plan{}, fmt.Errorf("event75: dormant row %d identity mismatch", activation.Slot)
		}
	}
	plan.TextIndex = rule.TurnChain.SuccessTextIndex
	plan.Activate = true
	return plan, nil
}

// CommitNativeFieldEvent75 在 FDTXT_029 index1 完成後重驗 plan，再原子提交
// event-state 與 typed live rows；只有 state 帶完整 raw field provenance 時才一併
// 核對並更新該 raw view。
func CommitNativeFieldEvent75(st *State, plan NativeFieldEvent75Plan) error {
	if st == nil || !plan.Activate || plan.trigger == nil || plan.rule == nil {
		return fmt.Errorf("event75: activation plan is unavailable")
	}
	recheck, err := PlanNativeFieldEvent75(st, plan.trigger, plan.trigger.X, plan.trigger.Y)
	if err != nil || !recheck.Activate || recheck.TextIndex != plan.TextIndex {
		return fmt.Errorf("event75: activation changed before commit: %v", err)
	}
	candidateState := st.NativeEventState
	candidateRows := st.NativeTurnEventControls
	candidateRaw := append([]byte(nil), st.NativeFieldControlRaw...)
	for _, write := range plan.rule.TurnChain.StateWrites {
		candidateState[write.Index] = write.Value
	}
	for _, activation := range plan.rule.TurnChain.TurnActivations {
		turn := st.NativeRoundCounter + activation.TurnDelta
		if turn < 0 || turn > 0xfe {
			return fmt.Errorf("event75: scheduled turn is outside byte range")
		}
		candidateRows[activation.Slot].Turn = byte(turn)
		if st.HasNativeFieldControlState {
			offset := 3 + activation.Slot*3
			if len(candidateRaw) <= offset+2 || candidateRaw[offset] != 0xff ||
				candidateRaw[offset+1] != byte(activation.EventID) ||
				candidateRaw[offset+2] != activation.RawCamp {
				return fmt.Errorf("event75: raw row %d disagrees with typed controls", activation.Slot)
			}
			candidateRaw[offset] = byte(turn)
		}
	}
	st.NativeEventState = candidateState
	st.NativeTurnEventControls = candidateRows
	if st.HasNativeFieldControlState {
		st.NativeFieldControlRaw = candidateRaw
	}
	return nil
}

// NativeFieldEventIDAt 保存 0x13a44 的格子事件 selector：
// 該格必須具有已驗證的 0..15 slot，event_id 不得為 0xff，且列內 selector
// 必須等於 caller selector。資料缺失時失敗即關閉。
func NativeFieldEventIDAt(
	st *State,
	x, y int,
	selector byte,
) (byte, bool) {
	if st == nil ||
		st.W <= 0 ||
		st.H <= 0 ||
		len(st.NativeFieldEventSlots) != st.W*st.H ||
		len(st.NativeFieldEvents) != 16 ||
		x < 0 || x >= st.W ||
		y < 0 || y >= st.H {
		return 0, false
	}
	slot := st.NativeFieldEventSlots[y*st.W+x]
	if slot < 0 || slot >= len(st.NativeFieldEvents) {
		return 0, false
	}
	event := st.NativeFieldEvents[slot]
	if event.EventID == 0xff || event.Selector != selector {
		return 0, false
	}
	return event.EventID, true
}

// ApplyNativeFieldModeEvent 執行目前已閉合的 mode-range 規則。
// selector 的呼叫時機仍由上層保存；非 mode 規則（包含 event61）不在此猜測執行。
func ApplyNativeFieldModeEvent(
	st *State,
	trigger *Unit,
	x, y int,
	selector byte,
) (byte, bool) {
	eventID, ok := NativeFieldEventIDAt(st, x, y, selector)
	if !ok || trigger == nil {
		return 0, false
	}
	var rule *NativeFieldEventRule
	for i := range st.NativeFieldEventRules {
		if st.NativeFieldEventRules[i].EventID == int(eventID) &&
			st.NativeFieldEventRules[i].Selector == selector {
			rule = &st.NativeFieldEventRules[i]
			break
		}
	}
	if rule == nil || len(rule.SetModeRanges) == 0 {
		return 0, false
	}
	if (rule.SetStateIndex == nil) != (rule.SetStateValue == nil) ||
		(rule.SetStateIndex != nil &&
			(*rule.SetStateIndex < 0 || *rule.SetStateIndex >= len(st.NativeEventState) ||
				*rule.SetStateValue < 0 || *rule.SetStateValue > 0xff)) {
		return 0, false
	}
	if rule.TriggerGate != "record_byte6_nonzero" ||
		!trigger.HasNativeRecordByte6 ||
		trigger.NativeRecordByte6 == 0 {
		return 0, false
	}
	for _, modeRange := range rule.SetModeRanges {
		if modeRange.Start < 0 || modeRange.End >= len(st.Units) {
			return 0, false
		}
		for index := modeRange.Start; index <= modeRange.End; index++ {
			if st.Units[index] == nil || !st.Units[index].HasNativeRecordByte34 {
				return 0, false
			}
		}
	}
	for _, modeRange := range rule.SetModeRanges {
		for index := modeRange.Start; index <= modeRange.End; index++ {
			u := st.Units[index]
			u.NativeRecordByte34 = (u.NativeRecordByte34 & 0xF0) | modeRange.Mode
		}
	}
	if rule.SetStateIndex != nil {
		st.NativeEventState[*rule.SetStateIndex] = byte(*rule.SetStateValue)
	}
	return eventID, true
}

// ApplyNativeFieldTurnActivationEvent 執行已閉合的 event62 mutation：selector0
// 對到 event62 時，把休眠 row0（event63/raw camp0）排到 native round+1，並
// 設定 battle-local state byte17。editable rule 與完整 raw row 必須先一致，
// 才能改動任一值。
func ApplyNativeFieldTurnActivationEvent(
	st *State,
	x, y int,
	selector byte,
) (byte, error) {
	if st == nil || !st.HasNativeTurnEventControlState {
		return 0, fmt.Errorf("turn activation: complete native turn controls are absent")
	}
	eventID, ok := NativeFieldEventIDAt(st, x, y, selector)
	if !ok || eventID != 62 || selector != 0 {
		return 0, fmt.Errorf("turn activation: event62 selector0 binding is absent")
	}
	var rule *NativeFieldEventRule
	for i := range st.NativeFieldEventRules {
		if st.NativeFieldEventRules[i].EventID == int(eventID) &&
			st.NativeFieldEventRules[i].Selector == selector {
			rule = &st.NativeFieldEventRules[i]
			break
		}
	}
	if rule == nil || rule.OnceState == nil || *rule.OnceState != 17 ||
		rule.TurnActivation == nil ||
		*rule.TurnActivation != (NativeTurnActivation{
			Slot: 0, EventID: 63, RawCamp: 0, TurnDelta: 1,
		}) {
		return 0, fmt.Errorf("turn activation: editable event62 rule is incomplete")
	}
	if st.NativeEventState[*rule.OnceState] != 0 ||
		st.NativeRoundCounter <= 0 || st.NativeRoundCounter > 0xfe {
		return 0, fmt.Errorf("turn activation: state or native round is not activatable")
	}
	activation := *rule.TurnActivation
	if activation.Slot < 0 || activation.Slot >= len(st.NativeTurnEventControls) {
		return 0, fmt.Errorf("turn activation: row slot is out of range")
	}
	row := st.NativeTurnEventControls[activation.Slot]
	if row != (NativeTurnEventControl{
		Turn: 0xff, EventID: byte(activation.EventID), RawCamp: activation.RawCamp,
	}) {
		return 0, fmt.Errorf("turn activation: dormant row identity mismatch")
	}
	rawOffset := 3 + activation.Slot*3
	if st.HasNativeFieldControlState {
		if len(st.NativeFieldControlRaw) <= rawOffset+2 ||
			st.NativeFieldControlRaw[rawOffset] != row.Turn ||
			st.NativeFieldControlRaw[rawOffset+1] != row.EventID ||
			st.NativeFieldControlRaw[rawOffset+2] != row.RawCamp {
			return 0, fmt.Errorf("turn activation: live raw row disagrees with typed controls")
		}
	}
	turn := byte(st.NativeRoundCounter + activation.TurnDelta)
	st.NativeTurnEventControls[activation.Slot].Turn = turn
	if st.HasNativeFieldControlState {
		st.NativeFieldControlRaw[rawOffset] = turn
	}
	st.NativeEventState[*rule.OnceState] = 1
	return eventID, nil
}

type NativeFieldEvent61Plan struct {
	EventID       byte
	MissingItem   bool
	TextIndex     int
	FinalText     int
	Presentation  NativeFieldPresentation
	JoinCharacter int

	trigger        *Unit
	itemIndex      int
	stateIndex     int
	spawnGroup     int
	requiredFrames int
}

// PlanNativeFieldEvent61 validates the complete editable/native boundary
// before the UI starts text #3 and the 59-frame presentation. Missing item is
// a valid, non-mutating native outcome; malformed provenance fails closed.
func PlanNativeFieldEvent61(
	st *State,
	trigger *Unit,
	x, y int,
) (NativeFieldEvent61Plan, error) {
	if st == nil || trigger == nil || trigger.X != x || trigger.Y != y {
		return NativeFieldEvent61Plan{}, fmt.Errorf("event61: invalid trigger")
	}
	eventID, ok := NativeFieldEventIDAt(st, x, y, 1)
	if !ok || eventID != 61 {
		return NativeFieldEvent61Plan{}, fmt.Errorf("event61: selector1 binding is absent")
	}
	var rule *NativeFieldEventRule
	for i := range st.NativeFieldEventRules {
		if st.NativeFieldEventRules[i].EventID == 61 &&
			st.NativeFieldEventRules[i].Selector == 1 {
			rule = &st.NativeFieldEventRules[i]
			break
		}
	}
	if rule == nil || rule.OnceState == nil || *rule.OnceState != 12 ||
		rule.RequiredItem == nil || *rule.RequiredItem != 0xD0 ||
		!rule.ConsumeItem || rule.SpawnGroup == nil || *rule.SpawnGroup != 1 ||
		rule.JoinCharacter == nil || *rule.JoinCharacter != 31 ||
		rule.TextIndices == nil || rule.Presentation == nil ||
		*rule.TextIndices != (NativeFieldTextIndices{MissingItem: 2, Success: 3, Final: 4}) ||
		*rule.Presentation != (NativeFieldPresentation{
			Archive: "FDOTHER.DAT", Resource: 45, Frames: 59,
			Helper: "0x2935b", DestinationOffset: 48356, Stride: 320,
			Transparent: -1, DelayHelper: "0x17aa9", DelayTicks: 2,
		}) {
		return NativeFieldEvent61Plan{}, fmt.Errorf("event61: editable rule is incomplete")
	}
	if st.NativeEventState[*rule.OnceState] != 0 {
		return NativeFieldEvent61Plan{}, fmt.Errorf("event61: once state is already set")
	}
	if err := ValidateNativeInventoryProjection(trigger); err != nil {
		return NativeFieldEvent61Plan{}, fmt.Errorf("event61: %w", err)
	}
	itemIndex := -1
	for i, item := range trigger.Inventory {
		if item == *rule.RequiredItem {
			itemIndex = i
			break
		}
	}
	plan := NativeFieldEvent61Plan{
		EventID: 61, MissingItem: itemIndex < 0,
		TextIndex: rule.TextIndices.Success, FinalText: rule.TextIndices.Final,
		Presentation: *rule.Presentation, JoinCharacter: *rule.JoinCharacter,
		trigger: trigger, itemIndex: itemIndex, stateIndex: *rule.OnceState,
		spawnGroup: *rule.SpawnGroup, requiredFrames: rule.Presentation.Frames,
	}
	if itemIndex < 0 {
		plan.TextIndex = rule.TextIndices.MissingItem
		return plan, nil
	}
	var pending int
	for _, unit := range st.Roster {
		if unit != nil && unit.Group == plan.spawnGroup {
			if unit.Fig != plan.JoinCharacter || unit.Camp != Own {
				return NativeFieldEvent61Plan{}, fmt.Errorf("event61: pending group identity mismatch")
			}
			pending++
		}
	}
	if pending != 1 {
		return NativeFieldEvent61Plan{}, fmt.Errorf("event61: pending group cardinality is %d", pending)
	}
	return plan, nil
}

// CommitNativeFieldEvent61 applies only the mutations after the caller has
// presented all native frames. It revalidates the plan so cancellation or an
// intervening inventory change cannot consume an item or partially set state.
// Persistent JOIN is intentionally returned to the campaign owner.
func CommitNativeFieldEvent61(
	st *State,
	plan NativeFieldEvent61Plan,
	presentedFrames int,
) (int, error) {
	if st == nil || plan.MissingItem || plan.trigger == nil ||
		presentedFrames != plan.requiredFrames ||
		plan.stateIndex != 12 || st.NativeEventState[plan.stateIndex] != 0 {
		return 0, fmt.Errorf("event61: plan is not committable")
	}
	if err := ValidateNativeInventoryProjection(plan.trigger); err != nil ||
		plan.itemIndex < 0 || plan.itemIndex >= len(plan.trigger.Inventory) ||
		plan.trigger.Inventory[plan.itemIndex] != 0xD0 {
		return 0, fmt.Errorf("event61: trigger inventory changed")
	}
	var pending int
	for _, unit := range st.Roster {
		if unit != nil && unit.Group == plan.spawnGroup &&
			unit.Fig == plan.JoinCharacter && unit.Camp == Own {
			pending++
		}
	}
	if pending != 1 {
		return 0, fmt.Errorf("event61: pending group changed")
	}
	if err := RemoveNativeCompactInventory(plan.trigger, plan.itemIndex); err != nil {
		return 0, err
	}
	st.NativeEventState[plan.stateIndex] = 1
	if appended := st.AppendGroup(plan.spawnGroup); appended != 1 {
		// Preflight made this unreachable for the current representation. Keep
		// the failure explicit instead of claiming campaign JOIN succeeded.
		return 0, fmt.Errorf("event61: group append changed after preflight")
	}
	return plan.JoinCharacter, nil
}
