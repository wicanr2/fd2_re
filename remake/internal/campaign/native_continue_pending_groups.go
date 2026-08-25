package campaign

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

type nativeContinueTurnKey struct {
	turn    int
	eventID int
}

// MaterializeNativeContinuePendingGroups 將仍未觸發的可編輯回合增援，綁到
// CONTINUE 戰場的未物化 FDFIELD roster。原版存檔選單在 0x117E7 的玩家互動
// 階段呼叫 0x19DF7；只有 0x13565 判定玩家可行動列耗盡後才進 0x1A30B。
// 其中 raw selector 1/0 在增加回合前掃描，selector 2 則在增加後掃描；所以
// saved turn 的 selector 1/0 仍待執行，selector 2 已於上一輪尾端執行。
//
// 這個適配器只接受 live turn/event pair 與可編輯 scenario 完全相符的章節。
// 原版 handler 會改寫 live turn byte；尚未資料化的改寫一律拒絕，不用靜態
// FDFIELD 或舊 scenario 覆寫存檔。所有檢查與深複製成功後才提交 State。
func MaterializeNativeContinuePendingGroups(
	state *battle.State,
	input fdsave.ContinueRuntimeInput,
	assetChapter int,
	assetState *battle.State,
	scenario *battle.Scenario,
	itemRows []byte,
) error {
	if state == nil || assetState == nil || scenario == nil {
		return fmt.Errorf("native CONTINUE pending groups: missing state or scenario")
	}
	if !input.ValidatedForRuntimeBridge() {
		return fmt.Errorf("native CONTINUE pending groups: input did not pass preflight")
	}
	if assetChapter != input.Context.Chapter ||
		scenario.Chapter != assetChapter+1 || scenario.Map != assetChapter ||
		assetState.W != input.Context.FieldWidth ||
		assetState.H != input.Context.FieldHeight ||
		state.W != assetState.W || state.H != assetState.H {
		return fmt.Errorf("native CONTINUE pending groups: chapter asset mismatch")
	}
	if !state.HasNativeFieldControlState ||
		!state.HasNativeRuntimeUnitProjection ||
		state.NativeRoundCounter != int(input.Header.TurnCounter) {
		return fmt.Errorf("native CONTINUE pending groups: prior runtime boundary is incomplete")
	}
	if len(assetState.Units) != len(state.NativeFieldUnitControls) ||
		len(assetState.NativeCompositionEventBytes) != assetState.W*assetState.H ||
		len(state.NativeCompositionEventBytes) != state.W*state.H ||
		!bytes.Equal(
			state.NativeCompositionEventBytes,
			assetState.NativeCompositionEventBytes,
		) || !reflect.DeepEqual(
		state.NativeFieldEventSlots,
		assetState.NativeFieldEventSlots,
	) || !reflect.DeepEqual(
		state.NativeFieldEventRules,
		assetState.NativeFieldEventRules,
	) {
		return fmt.Errorf("native CONTINUE pending groups: FDFIELD asset topology mismatch")
	}

	schedule, err := nativeContinueSpawnSchedule(scenario)
	if err != nil {
		return fmt.Errorf("native CONTINUE pending groups: %w", err)
	}
	if !scenario.RuntimeAppendGroups {
		if len(schedule) != 0 {
			return fmt.Errorf("native CONTINUE pending groups: static scenario declares future groups")
		}
		for _, rule := range assetState.NativeFieldEventRules {
			if rule.SpawnGroup != nil {
				return fmt.Errorf(
					"native CONTINUE pending groups: static scenario field event %d declares a future group",
					rule.EventID,
				)
			}
		}
		candidate := *state
		candidate.Roster = make([]*battle.Unit, 0)
		candidate.PendingGroups = make(map[int]bool)
		if err := candidate.BindNativeFutureItemRows(itemRows); err != nil {
			return fmt.Errorf("native CONTINUE pending groups: %w", err)
		}
		candidate.HasNativePendingGroupBinding = true
		*state = candidate
		return nil
	}
	currentTurn := int(input.Header.TurnCounter)
	if currentTurn <= 0 {
		return fmt.Errorf("native CONTINUE pending groups: saved turn is zero")
	}
	pending := make(map[int]bool)
	for key, groups := range schedule {
		matches := 0
		var rawSelector byte
		for _, live := range state.NativeTurnEventControls {
			if int(live.Turn) == key.turn && int(live.EventID) == key.eventID {
				if live.RawCamp > 2 {
					return fmt.Errorf(
						"live turn %d event %d has raw selector %d",
						key.turn, key.eventID, live.RawCamp,
					)
				}
				rawSelector = live.RawCamp
				matches++
			}
		}
		if matches != 1 {
			return fmt.Errorf(
				"live turn %d event %d matched %d rows",
				key.turn, key.eventID, matches,
			)
		}
		// 先核對所有原始排程列，再依 saved turn 分類。若 handler 把一個
		// 已過回合的列延後，先略過舊 turn 會把仍待執行的 group 靜默遺失。
		if key.turn < currentTurn {
			continue
		}
		if key.turn == currentTurn && rawSelector == 2 {
			// 初始 turn1 沒有可編輯 selector2 spawn schedule；若未來資料
			// 出現此形狀，需要另證 battle bootstrap 是否先掃 selector2。
			if currentTurn == 1 {
				return fmt.Errorf(
					"live turn 1 event %d uses ambiguous raw selector 2",
					key.eventID,
				)
			}
			continue
		}
		for _, group := range groups {
			pending[group] = true
		}
	}
	// 已資料化的格子事件也可能建構 future group（目前為 map25/event61）。
	// 只有其 once-state raw byte 能證明是否已消費；缺少該 producer 時不能
	// 從 current runtime record 反猜 group identity。
	for _, rule := range assetState.NativeFieldEventRules {
		if rule.SpawnGroup == nil {
			continue
		}
		if !nativeContinueFieldSpawnRuleIsLive(state, assetState, rule) {
			return fmt.Errorf(
				"native CONTINUE pending groups: field event %d lacks live selector/slot provenance",
				rule.EventID,
			)
		}
		if rule.OnceState == nil || *rule.OnceState < 0 ||
			*rule.OnceState >= len(state.NativeEventState) {
			return fmt.Errorf(
				"native CONTINUE pending groups: field event %d lacks once-state provenance",
				rule.EventID,
			)
		}
		group := *rule.SpawnGroup
		if group < 0 || group > 0xfe {
			return fmt.Errorf(
				"native CONTINUE pending groups: field event %d has invalid group %d",
				rule.EventID, group,
			)
		}
		if state.NativeEventState[*rule.OnceState] == 0 {
			pending[group] = true
		}
	}

	roster := make([]*battle.Unit, 0)
	seenGroups := make(map[int]int)
	for _, unit := range assetState.Units {
		if unit == nil || !pending[unit.Group] {
			continue
		}
		clone, cloneErr := cloneNativeContinueFutureUnit(unit)
		if cloneErr != nil {
			return fmt.Errorf("native CONTINUE pending groups: %w", cloneErr)
		}
		roster = append(roster, clone)
		seenGroups[unit.Group]++
	}
	groups := make([]int, 0, len(pending))
	for group := range pending {
		groups = append(groups, group)
	}
	sort.Ints(groups)
	for _, group := range groups {
		if group < 0 || group > 0xfe || seenGroups[group] == 0 {
			return fmt.Errorf(
				"native CONTINUE pending groups: group %d has no FDFIELD rows",
				group,
			)
		}
	}

	candidate := *state
	candidate.Roster = roster
	candidate.PendingGroups = make(map[int]bool, len(pending))
	for group := range pending {
		candidate.PendingGroups[group] = true
	}
	if err := candidate.BindNativeFutureItemRows(itemRows); err != nil {
		return fmt.Errorf("native CONTINUE pending groups: %w", err)
	}
	candidate.HasNativePendingGroupBinding = true
	*state = candidate
	return nil
}

func nativeContinueSpawnSchedule(
	scenario *battle.Scenario,
) (map[nativeContinueTurnKey][]int, error) {
	schedule := make(map[nativeContinueTurnKey][]int)
	seenGroups := make(map[int]nativeContinueTurnKey)
	for eventIndex, event := range scenario.Events {
		for actionIndex, action := range event.Do {
			if action.Type != "spawn_group" {
				continue
			}
			if event.Trigger != "on_turn_end" || event.When == nil ||
				event.When.Turn <= 0 || !event.Once || action.NativeEventID == nil ||
				len(action.Groups) == 0 || len(action.NativeSpawns) != len(action.Groups) {
				return nil, fmt.Errorf(
					"scenario event %d action %d lacks exact native schedule provenance",
					eventIndex, actionIndex,
				)
			}
			key := nativeContinueTurnKey{
				turn: event.When.Turn, eventID: *action.NativeEventID,
			}
			for index, group := range action.Groups {
				call := action.NativeSpawns[index]
				if group < 0 || group > 0xfe || call.Group != group ||
					call.RawPlacementGate == nil || call.Source == "" ||
					(call.Via != "spawn_group" && call.Via != "spawn_group_with_intro") {
					return nil, fmt.Errorf(
						"scenario event %d action %d group %d lacks native call provenance",
						eventIndex, actionIndex, group,
					)
				}
				if prior, exists := seenGroups[group]; exists {
					return nil, fmt.Errorf(
						"group %d is scheduled by both turn %d/event %d and turn %d/event %d",
						group, prior.turn, prior.eventID, key.turn, key.eventID,
					)
				}
				seenGroups[group] = key
				schedule[key] = append(schedule[key], group)
			}
		}
	}
	return schedule, nil
}

func nativeContinueFieldSpawnRuleIsLive(
	state *battle.State,
	assetState *battle.State,
	rule battle.NativeFieldEventRule,
) bool {
	if state == nil || assetState == nil ||
		len(state.NativeFieldEvents) != len(assetState.NativeFieldEvents) {
		return false
	}
	matchedSlot := -1
	for slot, event := range assetState.NativeFieldEvents {
		if int(event.EventID) != rule.EventID || event.Selector != rule.Selector {
			continue
		}
		if matchedSlot >= 0 || state.NativeFieldEvents[slot] != event {
			return false
		}
		matchedSlot = slot
	}
	if matchedSlot < 0 {
		return false
	}
	for _, slot := range assetState.NativeFieldEventSlots {
		if slot == matchedSlot {
			return true
		}
	}
	return false
}

func cloneNativeContinueFutureUnit(source *battle.Unit) (*battle.Unit, error) {
	if source == nil {
		return nil, fmt.Errorf("nil FDFIELD roster row")
	}
	clone := *source
	clone.Spells = append([]int(nil), source.Spells...)
	clone.Inventory = append([]int(nil), source.Inventory...)
	clone.Equipped = append([]bool(nil), source.Equipped...)
	clone.InventorySlots = append([]int(nil), source.InventorySlots...)
	clone.NativeInventoryFlags = append([]int(nil), source.NativeInventoryFlags...)
	if source.NativeConstructor != nil {
		constructor := *source.NativeConstructor
		constructor.Record = append([]byte(nil), source.NativeConstructor.Record...)
		constructor.AuxRecord = append([]byte(nil), source.NativeConstructor.AuxRecord...)
		clone.NativeConstructor = &constructor
	}
	if source.DeathEffect != nil {
		effect := *source.DeathEffect
		clone.DeathEffect = &effect
	}
	if source.DeathReward != nil {
		reward := *source.DeathReward
		clone.DeathReward = &reward
	}
	if !clone.HasNativePositionRecord || clone.NativeConstructor == nil ||
		!clone.HasMapSelectorKey || !clone.HasBattleFig ||
		!clone.HasNativeRecordByte6 || !clone.HasNativeRecordByte34 ||
		!clone.HasNativeRecordByte35 || !clone.HasNativeRecordByte36 {
		return nil, fmt.Errorf("group %d FDFIELD row lacks native constructor provenance", clone.Group)
	}
	return &clone, nil
}
