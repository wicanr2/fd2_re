// event.go — 可擴充事件系統(doc 29)。
//
// 設計三層:trigger(觸發時機)→ when(條件)→ do(動作序列)。原版 FD2 事件寫死在
// FDFIELD.DAT turn_events + EXE handler(doc 25);remake 全資料化成 scenario JSON,
// 引擎不為任何關卡寫死分支。新增條件/動作 = 在下方 switch 加一個 case。
package battle

import (
	"encoding/json"
	"fmt"
	"os"
)

// Scenario 一關的劇本(對映原版 FDFIELD turn_events + 青衫 ground truth)。
type Scenario struct {
	Chapter               int                    `json:"chapter"`
	Name                  string                 `json:"name"`
	Map                   int                    `json:"map"`
	RuntimeAppendGroups   bool                   `json:"runtime_append_groups,omitempty"`          // party first; FDFIELD groups append only when constructed
	NativeActingResources string                 `json:"native_acting_resources,omitempty"`        // asset-root-relative decoded 0x1366A resource set for native battle events
	InitialGroups         []int                  `json:"initial_groups"`                           // 開局即在場的 unit group;其餘待命
	InitialGroupsAbsent   []InitialGroupAbsent   `json:"initial_groups_if_party_absent,omitempty"` // native pre-handler conditional FDFIELD group
	Party                 []PartyMember          `json:"party"`                                    // 主角隊(不在 FDFIELD roster,on_battle_start 進場)
	DeployCells           [][2]int               `json:"deploy_cells"`                             // 主角隊進場目標格
	Events                []Event                `json:"events"`
	NativeFieldEventRules []NativeFieldEventRule `json:"native_field_event_rules,omitempty"`
	NativeTurnEvents      []NativeTurnEvent      `json:"native_turn_events,omitempty"`
	pendingJoins          []int
}

// NativeTurnEvent preserves one live FDFIELD three-byte row consumer without
// forcing it into the normalized on_turn_end trigger. RawCamp is deliberately
// not renamed to a faction: sub_1A813 compares the byte at row+5, and each
// caller owns a different phase boundary.
type NativeTurnEvent struct {
	EventID      int                     `json:"event_id"`
	RawCamp      int                     `json:"raw_camp"`
	Handler      string                  `json:"handler"`
	Staging      NativeTurnStaging       `json:"staging"`
	DynamicGroup *NativeTurnDynamicGroup `json:"dynamic_group,omitempty"`
	Progression  *NativeTurnProgression  `json:"progression,omitempty"`
	PairMutation *NativeTurnPairMutation `json:"pair_mutation,omitempty"`
}

// NativeTurnDynamicGroup 從 raw event-state table 解析一個 0x35822 group 參數，
// 再套用該 caller 已證實的 live-row／state writes；它與 event63 這類固定 staging
// calls 分開表示。
type NativeTurnDynamicGroup struct {
	StateIndex      int `json:"state_index"`
	Minimum         int `json:"minimum"`
	Maximum         int `json:"maximum"`
	ControlSlot     int `json:"control_slot"`
	RescheduleDelta int `json:"reschedule_delta"`
	StopValue       int `json:"stop_value"`
	Increment       int `json:"increment"`
}

// NativeTurnProgression 保存 event76 這類由 raw event-state byte 推進、最後才
// materialize group 並啟動另一 live row 的可編輯規則；欄位名只描述運算，不替
// state byte 或 RawCamp 猜測玩法語意。
type NativeTurnProgression struct {
	StateIndex       int                  `json:"state_index"`
	RepeatUntil      int                  `json:"repeat_until"`
	MarkUnitIndex    int                  `json:"mark_unit_index"`
	MarkHelper       string               `json:"mark_helper"`
	ControlSlot      int                  `json:"control_slot"`
	RescheduleDelta  int                  `json:"reschedule_delta"`
	FinalTextIndex   int                  `json:"final_text_index"`
	SpawnGroup       int                  `json:"spawn_group"`
	SpawnHelper      string               `json:"spawn_helper"`
	RawPlacementGate int                  `json:"raw_placement_gate"`
	SpawnCount       int                  `json:"spawn_count"`
	BaseStateIndex   int                  `json:"base_state_index"`
	FinalActivation  NativeTurnActivation `json:"final_activation"`
	PulseHandler     string               `json:"pulse_handler"`
	PulseCount       int                  `json:"pulse_count"`
	ExtraDelayMS     int                  `json:"extra_delay_ms"`
	TailTextIndices  []int                `json:"tail_text_indices"`
}

// NativeTurnPairMutation 保存 event79 由一個 raw base 與單一步 RNG 選出兩個
// 循環相鄰 runtime slots 的規則，不替 bit7 賦予 caller 之外的玩法名稱。
type NativeTurnPairMutation struct {
	BaseStateIndex  int    `json:"base_state_index"`
	Group           int    `json:"group"`
	Count           int    `json:"count"`
	Modulo          int    `json:"modulo"`
	SecondOffset    int    `json:"second_offset"`
	MarkHelper      string `json:"mark_helper"`
	ControlSlot     int    `json:"control_slot"`
	RescheduleDelta int    `json:"reschedule_delta"`
}

// NativeTurnStaging is the editable form of the recovered 0x35822 helper.
// Helper addresses remain provenance, not runtime dispatch keys: the UI
// adapter validates the complete known signature before executing it.
type NativeTurnStaging struct {
	Helper             string                  `json:"helper"`
	PanHelper          string                  `json:"pan_helper"`
	SpawnHelper        string                  `json:"spawn_helper"`
	DelayBeforeFlashMS int                     `json:"delay_before_flash_ms"`
	PaletteHelper      string                  `json:"palette_helper"`
	PaletteStart       int                     `json:"palette_start"`
	PaletteEnd         int                     `json:"palette_end"`
	FlashDelta         int                     `json:"flash_delta"`
	FlashHoldMS        int                     `json:"flash_hold_ms"`
	RestoreDelta       int                     `json:"restore_delta"`
	RedrawHelper       string                  `json:"redraw_helper"`
	RawPlacementGate   int                     `json:"raw_placement_gate"`
	Calls              []NativeTurnStagingCall `json:"calls"`
}

type NativeTurnStagingCall struct {
	Group  int    `json:"group"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Source string `json:"source"`
}

// InitialGroupAbsent is an evidence-backed pre-battle condition: materialize
// its FDFIELD group only when the permanent party lacks CharID. It is not a
// generic event expression and does not claim runtime-slot append identity.
type InitialGroupAbsent struct {
	CharID int `json:"char_id"`
	Group  int `json:"group"`
}

// PartyMember 主角隊成員(數值來自 characters.json / EXE 表)。
type PartyMember struct {
	Name string `json:"name"`
	Cls  string `json:"cls"`
	// Fig is the stable JOIN/roster identity used by the remake's persistent
	// party map. For a fresh native JOIN record it also seeds raw +7, but it is
	// not the mutable map/battle selector after class change.
	Fig int `json:"fig"`
	// NativeIdentity is the optional persistent-record +0x08 key used by
	// native 0x11506. It is intentionally separate from Fig/+7 selectors.
	NativeIdentity *int `json:"native_identity,omitempty"`
	// Raw JOIN-constructor bytes written to persistent +0x1f/+0x20.
	NativeRecordRace  *byte `json:"native_record_race,omitempty"`
	NativeRecordClass *byte `json:"native_record_class,omitempty"`
	Portrait          int   `json:"portrait"`
	HP                int   `json:"hp"`
	MP                int   `json:"mp"`
	AP                int   `json:"ap"`
	DP                int   `json:"dp"`
	DX                int   `json:"dx"`   // projected into +0x3e; direct-raw provenance must be stated separately
	HIT               int   `json:"hit"`  // 命中(doc32:DX+起始武器HIT增值,對照orig_07_unit_status.png逐位驗證)
	EV                int   `json:"ev"`   // 閃避(doc32:DX+起始防具EV增值;起始4件防具EV增值皆為0)
	CritPct           int   `json:"crit"` // 暴擊率(resist_crit.json 依角色職業)
	MV                int   `json:"mv"`
	AtkMin            int   `json:"atk_min"` // 攻擊距離下限(0=預設1;doc32 weapon_range.json)
	AtkMax            int   `json:"atk_max"` // 攻擊距離上限(0=預設1;如亞雷斯騎士槍type3=2)
	Lv                int   `json:"lv"`
	Spells            []int `json:"spells"` // 已習得法術 id(spell.json)
	// InitialCommandMask is the exact four-byte constructor source for
	// unit+0x1a..+0x1d. It is deliberately not derived from Spells.
	InitialCommandMask []byte `json:"initial_command_mask,omitempty"`
	Inventory          []int  `json:"inventory,omitempty"`
	InventorySlots     []int  `json:"inventory_slots,omitempty"`
}

// Event 一條事件規則。
type Event struct {
	ID      string   `json:"id"`
	Trigger string   `json:"trigger"` // on_battle_start / on_turn_end / on_unit_death
	When    *When    `json:"when,omitempty"`
	Do      []Action `json:"do"`
	Once    bool     `json:"once"`
	fired   bool
}

// When 條件(可擴充:加欄位 + Match 加判斷)。
type When struct {
	Turn                  int    `json:"turn,omitempty"`                     // turn == N(0=不限)
	UnitDead              string `json:"unit_dead,omitempty"`                // 某角色陣亡
	UnitSlotActive        *int   `json:"unit_slot_active,omitempty"`         // 原版 runtime slot 已登場且存活
	NativeEventStateIndex *int   `json:"native_event_state_index,omitempty"` // battle-local raw byte index
	NativeEventStateValue *int   `json:"native_event_state_value,omitempty"` // exact raw byte value
}

// Action 動作(可擴充:加 type + execAction 加 case)。
type Action struct {
	Type           string            `json:"type"`
	Groups         []int             `json:"groups,omitempty"`          // spawn_group 的波次
	NativeEventID  *int              `json:"native_event_id,omitempty"` // turn_events 的原版事件編號
	NativeSpawns   []NativeSpawnCall `json:"native_spawns,omitempty"`   // 原版逐呼叫配置資料
	Camp           string            `json:"camp,omitempty"`            // 增援陣營(改為)
	ActImmediately bool              `json:"act_immediately,omitempty"` // 增援當回合可動(青衫「立即行動」)
	Speaker        int               `json:"speaker"`                   // dialogue 說話者(DATO 肖像 id;-1=旁白)
	Text           string            `json:"text,omitempty"`            // dialogue 文本
	Flag           string            `json:"flag,omitempty"`            // set_flag
	Unit           string            `json:"unit,omitempty"`            // set_ai 目標
	Mode           string            `json:"mode,omitempty"`            // set_ai 模式(berserk…)
	CharID         int               `json:"char_id,omitempty"`         // join_party: permanent player identity
	Grid           *[2]int           `json:"grid,omitempty"`            // pan:原版 camera grid(col,row)，runtime 依地圖 tile 尺寸換 pixel
	Ms             int               `json:"ms,omitempty"`              // delay:原版毫秒數
	// NativeActing 保存戰鬥事件呼叫 0x1366A 的資源與原始呼叫點。
	// 它只描述已解碼的行為，不以位址作為重製端執行鍵。
	NativeActing *NativeFollowingActing `json:"native_acting,omitempty"`
	// NativeSource 保留無獨立結構可承載的原始寫入／對話呼叫點。
	NativeSource string `json:"native_source,omitempty"`
	// NativeTextIndex 是一次原版 0x15F84 呼叫的 FDTXT 索引；多句
	// editable dialogue 可共用同一索引，但不能由文字內容反推。
	NativeTextIndex *int `json:"native_text_index,omitempty"`
	// EventStateIndex/Value 僅供已證實的 battle-local raw byte 寫入。
	EventStateIndex *int `json:"event_state_index,omitempty"`
	EventStateValue *int `json:"event_state_value,omitempty"`
}

// NativeSpawnCall 保存全域事件處理器的一個確切呼叫點。Group 是該排程回合
// 解析出的值；Source、Via 與 RawPlacementGate 直接來自 EXE，不從陣營或群組推測。
type NativeSpawnCall struct {
	Group            int                    `json:"group"`
	Via              string                 `json:"via"`
	Source           string                 `json:"source"`
	RawPlacementGate *int                   `json:"raw_placement_gate"`
	FollowingActing  *NativeFollowingActing `json:"following_acting,omitempty"`
}

// NativeFollowingActing 保存 wrapper 返回後、由同一呼叫端明確執行的 ACTING。
// 它不是由 group 推導的，也不是 0x32999 函式本體的一部分。
type NativeFollowingActing struct {
	Resource int    `json:"resource"`
	Source   string `json:"source"`
}

// DialogLine 一句對話(說話者肖像 + 文本),供 UI 畫頭像+嘴型+文字。
type DialogLine struct {
	Speaker        int
	Text           string
	Upper          *bool // 對話框上下位置覆蓋(nil=沿用預設「id>=32 走上框」規則;見 campaign.Beat.Upper)
	NativeDialogue *NativeDialogueLayout
}

// NativeDialogueLayout 是 campaign 編輯資料中 FFFE／FFFD 頁面的執行期副本。
// 型別留在 battle，避免泛用戰鬥套件反向依賴 campaign 編譯器。
type NativeDialogueLayout struct {
	SourceDAT   string
	StringIndex int
	Utterance   int
	Control     string
	Operand     int
	Pages       [][]string
	// MotionTargetY 是 sub_15F84 var_20 的 caller-resolved runtime 值。
	// 它不屬於可編輯 FDTXT bytes；HasMotionTarget=false 時原生收框必須失敗即關閉。
	MotionTargetY    int
	HasMotionTargetY bool
}

// LoadScenario 讀 scenario JSON。
func LoadScenario(path string) (*Scenario, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sc Scenario
	if err := json.Unmarshal(raw, &sc); err != nil {
		return nil, err
	}
	for i, member := range sc.Party {
		// Validate at the editable boundary. PartyUnits has a legacy no-error
		// signature, so accepting malformed JSON there would otherwise silently
		// change the native command inventory to zero.
		if err := (&Unit{}).SetInitialCommandMask(member.InitialCommandMask); err != nil {
			return nil, fmt.Errorf("scenario party member %d (%s) initial_command_mask: %w", i, member.Name, err)
		}
		if member.NativeIdentity != nil && (*member.NativeIdentity < 0 || *member.NativeIdentity > 0xff) {
			return nil, fmt.Errorf("scenario party member %d (%s) native_identity %d out of byte range", i, member.Name, *member.NativeIdentity)
		}
	}
	for eventIndex, event := range sc.Events {
		if event.When != nil &&
			((event.When.NativeEventStateIndex == nil) != (event.When.NativeEventStateValue == nil) ||
				(event.When.NativeEventStateIndex != nil &&
					(*event.When.NativeEventStateIndex < 0 || *event.When.NativeEventStateIndex >= 0x20 ||
						*event.When.NativeEventStateValue < 0 || *event.When.NativeEventStateValue > 0xff))) {
			return nil, fmt.Errorf("scenario event %d has invalid native event-state condition", eventIndex)
		}
		for actionIndex, action := range event.Do {
			if action.Type == "native_acting" {
				if action.NativeActing == nil || action.NativeActing.Resource < 0 ||
					action.NativeActing.Resource >= 106 || action.NativeActing.Source == "" ||
					action.NativeEventID == nil || *action.NativeEventID < 0 ||
					*action.NativeEventID >= 90 || sc.NativeActingResources == "" {
					return nil, fmt.Errorf(
						"scenario event %d action %d has invalid native acting provenance",
						eventIndex, actionIndex,
					)
				}
			}
			if action.Type == "set_native_event_state" {
				if action.EventStateIndex == nil || *action.EventStateIndex < 0 ||
					*action.EventStateIndex >= 0x20 || action.EventStateValue == nil ||
					*action.EventStateValue < 0 || *action.EventStateValue > 0xff ||
					action.NativeSource == "" || action.NativeEventID == nil ||
					*action.NativeEventID < 0 || *action.NativeEventID >= 90 {
					return nil, fmt.Errorf(
						"scenario event %d action %d has invalid native event-state write",
						eventIndex, actionIndex,
					)
				}
			}
			if len(action.NativeSpawns) == 0 {
				continue
			}
			if action.Type != "spawn_group" || len(action.NativeSpawns) != len(action.Groups) {
				return nil, fmt.Errorf(
					"scenario event %d action %d has inconsistent native spawn calls",
					eventIndex, actionIndex,
				)
			}
			if action.NativeEventID == nil || *action.NativeEventID < 0 || *action.NativeEventID >= 90 {
				return nil, fmt.Errorf(
					"scenario event %d action %d lacks native event provenance",
					eventIndex, actionIndex,
				)
			}
			for i, call := range action.NativeSpawns {
				if call.Group != action.Groups[i] || call.Source == "" ||
					(call.Via != "spawn_group" && call.Via != "spawn_group_with_intro") ||
					call.RawPlacementGate == nil || *call.RawPlacementGate < 0 || *call.RawPlacementGate > 0xff {
					return nil, fmt.Errorf(
						"scenario event %d action %d native spawn %d is invalid",
						eventIndex, actionIndex, i,
					)
				}
				if call.Via == "spawn_group_with_intro" {
					if call.FollowingActing == nil || call.FollowingActing.Resource < 0 ||
						call.FollowingActing.Resource >= 106 || call.FollowingActing.Source == "" ||
						sc.NativeActingResources == "" {
						return nil, fmt.Errorf(
							"scenario event %d action %d native intro spawn %d lacks following acting provenance/resources",
							eventIndex, actionIndex, i,
						)
					}
				} else if call.FollowingActing != nil {
					return nil, fmt.Errorf(
						"scenario event %d action %d ordinary native spawn %d carries intro acting",
						eventIndex, actionIndex, i,
					)
				}
			}
		}
	}
	seenNativeField := map[[2]int]bool{}
	for ruleIndex, rule := range sc.NativeFieldEventRules {
		key := [2]int{rule.EventID, int(rule.Selector)}
		if rule.EventID < 0 || rule.EventID >= 90 || rule.Selector > 2 ||
			seenNativeField[key] || rule.TriggerGate == "" || rule.TurnChain == nil ||
			rule.TurnChain.Handler == "" || len(rule.TurnChain.StateWrites) == 0 ||
			len(rule.TurnChain.TurnActivations) == 0 {
			return nil, fmt.Errorf("scenario native field rule %d is invalid", ruleIndex)
		}
		seenNativeField[key] = true
		for _, write := range rule.TurnChain.StateWrites {
			if write.Index < 0 || write.Index >= 0x20 {
				return nil, fmt.Errorf("scenario native field rule %d state write is invalid", ruleIndex)
			}
		}
		for _, activation := range rule.TurnChain.TurnActivations {
			if activation.Slot < 0 || activation.Slot >= 16 || activation.EventID < 0 ||
				activation.EventID >= 90 || activation.RawCamp < 0 || activation.RawCamp > 0xff {
				return nil, fmt.Errorf("scenario native field rule %d turn activation is invalid", ruleIndex)
			}
		}
	}
	seenNativeTurn := map[[2]int]bool{}
	initialGroups := map[int]bool{}
	for _, group := range sc.InitialGroups {
		initialGroups[group] = true
	}
	for eventIndex, event := range sc.NativeTurnEvents {
		key := [2]int{event.EventID, event.RawCamp}
		staging := event.Staging
		if !sc.RuntimeAppendGroups || event.EventID < 0 || event.EventID >= 90 ||
			event.RawCamp < 0 || event.RawCamp > 0xff || event.Handler == "" ||
			seenNativeTurn[key] {
			return nil, fmt.Errorf("scenario native turn event %d is invalid", eventIndex)
		}
		if event.Progression == nil && event.PairMutation == nil && (staging.Helper == "" || staging.PanHelper == "" ||
			staging.SpawnHelper == "" || staging.PaletteHelper == "" || staging.RedrawHelper == "" ||
			staging.DelayBeforeFlashMS < 0 || staging.FlashHoldMS < 0 ||
			staging.PaletteStart < 0 || staging.PaletteEnd < staging.PaletteStart || staging.PaletteEnd > 255 ||
			staging.FlashDelta < 0 || staging.FlashDelta > 255 ||
			staging.RestoreDelta < 0 || staging.RestoreDelta > 255 ||
			staging.RawPlacementGate < 0 || staging.RawPlacementGate > 0xff || len(staging.Calls) == 0) {
			return nil, fmt.Errorf("scenario native turn event %d is invalid", eventIndex)
		}
		seenNativeTurn[key] = true
		if event.DynamicGroup != nil {
			dynamic := event.DynamicGroup
			if dynamic.StateIndex < 0 || dynamic.StateIndex >= 0x20 ||
				dynamic.Minimum < 0 || dynamic.Maximum < dynamic.Minimum || dynamic.Maximum > 0xfe ||
				dynamic.ControlSlot < 0 || dynamic.ControlSlot >= 16 ||
				dynamic.RescheduleDelta <= 0 || dynamic.StopValue < dynamic.Minimum ||
				dynamic.StopValue > dynamic.Maximum || dynamic.Increment <= 0 || len(staging.Calls) != 1 {
				return nil, fmt.Errorf("scenario native turn event %d dynamic group is invalid", eventIndex)
			}
		}
		if event.Progression != nil {
			progression := event.Progression
			if event.DynamicGroup != nil || len(staging.Calls) != 0 || progression.StateIndex < 0 || progression.StateIndex >= 0x20 ||
				progression.RepeatUntil <= 0 || progression.RepeatUntil > 0xff || progression.MarkUnitIndex < 0 ||
				progression.MarkHelper == "" || progression.ControlSlot < 0 || progression.ControlSlot >= 16 ||
				progression.RescheduleDelta <= 0 || progression.FinalTextIndex < 0 || progression.SpawnGroup < 0 ||
				progression.SpawnGroup > 0xff || progression.SpawnHelper == "" || progression.RawPlacementGate < 0 ||
				progression.RawPlacementGate > 0xff || progression.SpawnCount <= 0 || progression.BaseStateIndex < 0 ||
				progression.BaseStateIndex >= 0x20 || progression.FinalActivation.Slot < 0 ||
				progression.FinalActivation.Slot >= 16 || progression.FinalActivation.EventID < 0 ||
				progression.FinalActivation.EventID >= 90 || progression.FinalActivation.RawCamp < 0 ||
				progression.FinalActivation.RawCamp > 0xff || progression.PulseHandler == "" ||
				progression.PulseCount <= 0 || progression.ExtraDelayMS < 0 || len(progression.TailTextIndices) == 0 {
				return nil, fmt.Errorf("scenario native turn event %d progression is invalid", eventIndex)
			}
		}
		if event.PairMutation != nil {
			pair := event.PairMutation
			if event.DynamicGroup != nil || event.Progression != nil || len(staging.Calls) != 0 ||
				pair.BaseStateIndex < 0 || pair.BaseStateIndex >= 0x20 || pair.Group < 0 || pair.Group > 0xff ||
				pair.Count <= 1 || pair.Modulo <= 1 || pair.SecondOffset <= 0 || pair.MarkHelper == "" ||
				pair.ControlSlot < 0 || pair.ControlSlot >= 16 || pair.RescheduleDelta <= 0 {
				return nil, fmt.Errorf("scenario native turn event %d pair mutation is invalid", eventIndex)
			}
		}
		seenGroups := map[int]bool{}
		for callIndex, call := range staging.Calls {
			if (event.DynamicGroup == nil && (call.Group < 0 || call.Group > 0xff)) ||
				(event.DynamicGroup != nil && call.Group != -1) || call.X < 0 || call.Y < 0 ||
				call.Source == "" || seenGroups[call.Group] || initialGroups[call.Group] {
				return nil, fmt.Errorf("scenario native turn event %d staging call %d is invalid", eventIndex, callIndex)
			}
			seenGroups[call.Group] = true
		}
	}
	return &sc, nil
}

// Setup 是既有測試與無原生欄位規則劇本的相容入口。正式執行路徑必須使用
// SetupChecked，避免把資料綁定錯誤誤當成沒有開場對話。
func (sc *Scenario) Setup(st *State) []DialogLine {
	dialogue, _ := sc.SetupChecked(st)
	return dialogue
}

// SetupChecked 套用劇本初始狀態並回報原生規則綁定錯誤；正式執行期以此
// 入口維持失敗即關閉。
func (sc *Scenario) SetupChecked(st *State) ([]DialogLine, error) {
	if err := sc.bindNativeFieldEventRules(st); err != nil {
		return nil, err
	}
	if sc.RuntimeAppendGroups {
		// Keep FDFIELD records as immutable-order constructor inputs. The opening
		// event materializes the player party first; initial FDFIELD groups follow,
		// exactly matching the runtime slots addressed by handler bytecode.
		st.Roster = st.Units
		st.Units = nil
		sc.materializePendingGroups(st)
		dialogues := sc.Fire(st, "on_battle_start", "")
		for _, group := range sc.InitialGroups {
			st.AppendGroup(group)
		}
		return dialogues, nil
	}
	if len(sc.InitialGroups) > 0 {
		init := map[int]bool{}
		for _, g := range sc.InitialGroups {
			init[g] = true
		}
		present := map[int]bool{}
		for _, member := range sc.Party {
			present[member.Fig] = true
		}
		for _, conditional := range sc.InitialGroupsAbsent {
			if !present[conditional.CharID] {
				init[conditional.Group] = true
			}
		}
		for _, u := range st.Units {
			if !init[u.Group] {
				u.OnField = false // 待命,等事件放出
			}
		}
	}
	return sc.Fire(st, "on_battle_start", ""), nil
}

func (sc *Scenario) bindNativeFieldEventRules(st *State) error {
	if sc == nil || st == nil {
		return fmt.Errorf("scenario field rules: missing scenario or state")
	}
	seen := make(map[[2]int]bool)
	for _, rule := range st.NativeFieldEventRules {
		seen[[2]int{rule.EventID, int(rule.Selector)}] = true
	}
	for _, rule := range sc.NativeFieldEventRules {
		key := [2]int{rule.EventID, int(rule.Selector)}
		if rule.EventID < 0 || rule.EventID >= 90 || seen[key] {
			return fmt.Errorf("scenario field rule event%d selector%d is invalid or duplicated", rule.EventID, rule.Selector)
		}
		seen[key] = true
		st.NativeFieldEventRules = append(st.NativeFieldEventRules, rule)
	}
	return nil
}

// BindNativeFieldEventRules 只把劇本的可編輯格子規則綁到既有戰場狀態。
// 原版 CONTINUE 已保存 live runtime，不能呼叫 SetupChecked 重播開場或重建單位；
// 這個窄入口讓續戰交易仍使用和正常新戰鬥相同的驗證與重複鍵拒絕規則。
func (sc *Scenario) BindNativeFieldEventRules(st *State) error {
	return sc.bindNativeFieldEventRules(st)
}

func (sc *Scenario) materializePendingGroups(st *State) {
	st.PendingGroups = map[int]bool{}
	for _, event := range sc.Events {
		for _, action := range event.Do {
			if action.Type == "spawn_group" {
				for _, group := range action.Groups {
					st.PendingGroups[group] = true
				}
			}
		}
	}
	for _, event := range sc.NativeTurnEvents {
		if event.DynamicGroup != nil {
			for group := event.DynamicGroup.Minimum; group <= event.DynamicGroup.Maximum; group++ {
				st.PendingGroups[group] = true
			}
			continue
		}
		if event.Progression != nil {
			st.PendingGroups[event.Progression.SpawnGroup] = true
			continue
		}
		for _, call := range event.Staging.Calls {
			st.PendingGroups[call.Group] = true
		}
	}
}

// NativeTurnEventsAt returns the editable handlers selected by sub_1A813's
// exact live-row predicate, preserving the sixteen-row order. A live row with
// no unique editable consumer is an error rather than a skipped event.
func (sc *Scenario) NativeTurnEventsAt(st *State, rawCamp byte) ([]NativeTurnEvent, error) {
	if sc == nil || st == nil || !st.HasNativeTurnEventControlState ||
		st.NativeRoundCounter <= 0 || st.NativeRoundCounter > 0xfe {
		return nil, fmt.Errorf("native turn event: live control state unavailable")
	}
	var out []NativeTurnEvent
	for slot, row := range st.NativeTurnEventControls {
		if int(row.Turn) != st.NativeRoundCounter || row.RawCamp != rawCamp {
			continue
		}
		matches := 0
		var matched NativeTurnEvent
		for _, event := range sc.NativeTurnEvents {
			if event.EventID == int(row.EventID) && event.RawCamp == int(rawCamp) {
				matched = event
				matches++
			}
		}
		if matches != 1 {
			return nil, fmt.Errorf(
				"native turn event: slot %d event %d raw camp %d has %d editable consumers",
				slot, row.EventID, rawCamp, matches,
			)
		}
		out = append(out, matched)
	}
	return out, nil
}

// AdoptHandlerBattleState attaches turn/death events to a runtime roster that
// the immediately preceding native pre-handler already constructed and moved.
// It must not replay on_battle_start: LOADCH/SPAWN/ACT effects are already
// present in that roster. Callers are responsible for proving that handler and
// battle use the same roster/scenario sources.
func (sc *Scenario) AdoptHandlerBattleState(st *State) error {
	if sc == nil || st == nil || !sc.RuntimeAppendGroups || len(st.Units) == 0 {
		return fmt.Errorf("battle: handler state cannot satisfy runtime-append scenario")
	}
	if err := sc.bindNativeFieldEventRules(st); err != nil {
		return err
	}
	sc.materializePendingGroups(st)
	// LOADCH always constructs party first and may already have appended one or
	// more initial FDFIELD groups. The caller removes those source rows from
	// Roster before this boundary; append every declared initial group now so a
	// no-op preserves an already materialized group while remaining groups keep
	// the original selector-cache order.
	for _, group := range sc.InitialGroups {
		st.AppendGroup(group)
		if st.NativeMapSelectorError != nil {
			return fmt.Errorf("battle: adopted initial group %d: %w", group, st.NativeMapSelectorError)
		}
	}
	for i := range sc.Events {
		event := &sc.Events[i]
		if event.Trigger == "on_battle_start" {
			if !event.Once {
				return fmt.Errorf("battle: repeatable on_battle_start cannot be adopted")
			}
			event.fired = true
		}
	}
	return nil
}

// Fire 對某 trigger 評估所有事件,執行符合者的動作。回傳要播的對話(含說話者)。
// ctxUnit:on_unit_death 時傳陣亡者名。
func (sc *Scenario) Fire(st *State, trigger, ctxUnit string) []DialogLine {
	actions := sc.TriggerActions(st, trigger, ctxUnit)
	var dialogues []DialogLine
	for _, action := range actions {
		if dl, ok := sc.ExecuteAction(st, action); ok {
			dialogues = append(dialogues, dl)
		}
	}
	return dialogues
}

// TriggerActions evaluates one trigger and returns its ordered editable actions,
// marking matching once-events as fired without executing them. The UI runtime
// uses this to preserve blocking PAN/delay/dialogue order; Fire remains the
// synchronous compatibility path for setup, tests and triggers without staging.
func (sc *Scenario) TriggerActions(st *State, trigger, ctxUnit string) []Action {
	var actions []Action
	for i := range sc.Events {
		e := &sc.Events[i]
		if e.Trigger != trigger || (e.Once && e.fired) {
			continue
		}
		if !e.When.match(st, ctxUnit) {
			continue
		}
		e.fired = true
		actions = append(actions, e.Do...)
	}
	return actions
}

// match 條件判斷(可擴充)。nil = 無條件,恆真。
func (w *When) match(st *State, ctxUnit string) bool {
	if w == nil {
		return true
	}
	if w.Turn != 0 && st.Turn != w.Turn {
		return false
	}
	if w.UnitDead != "" && ctxUnit != w.UnitDead {
		return false
	}
	if w.UnitSlotActive != nil {
		slot := *w.UnitSlotActive
		if slot < 0 || slot >= len(st.Units) || st.Units[slot] == nil ||
			!st.Units[slot].OnField || !st.Units[slot].Alive() {
			return false
		}
	}
	if w.NativeEventStateIndex != nil || w.NativeEventStateValue != nil {
		if w.NativeEventStateIndex == nil || w.NativeEventStateValue == nil {
			return false
		}
		index := *w.NativeEventStateIndex
		if st == nil || index < 0 || index >= len(st.NativeEventState) ||
			int(st.NativeEventState[index]) != *w.NativeEventStateValue {
			return false
		}
	}
	return true
}

// ExecuteAction 執行單一狀態動作。pan/delay 由 UI runner 阻塞處理；
// 回傳 (對話, true) 表示 runner 應停下並播放這句。
func (sc *Scenario) ExecuteAction(st *State, a Action) (DialogLine, bool) {
	dialogue, isDialogue, _ := sc.ExecuteActionChecked(st, a)
	return dialogue, isDialogue
}

// ExecuteActionChecked 是正式執行路徑的錯誤回報邊界。上方相容包裝保留給舊的
// 同步呼叫者；介面執行器使用本函式，避免原版增援資料錯誤被靜默忽略。
func (sc *Scenario) ExecuteActionChecked(st *State, a Action) (DialogLine, bool, error) {
	switch a.Type {
	case "spawn_party": // 主角隊從隊伍名冊進場到部署格(doc 25 雙來源)
		// The party constructor itself places members directly on the deployment
		// cells. Chapter 0 then immediately plays decoded ACT(0), which moves all
		// four runtime slots six cells upward; the old "no movement animation"
		// conclusion confused construction with the following handler operation.
		st.AppendNativeMapSelectorBatchOrLegacy(sc.PartyUnits(st.OwnDeploy))
	case "spawn_group": // 增援登場(原版 turn_events;doc 25)
		camp := campFrom(a.Camp)
		if len(a.NativeSpawns) > 0 && sc.RuntimeAppendGroups {
			if st == nil || len(st.Roster) == 0 {
				return DialogLine{}, false, fmt.Errorf("native spawn requires a runtime roster")
			}
			// 0x32999 不只是建構單位：它含 FDOTHER #9 的 12 次索引呈現，
			// wrapper 返回後呼叫端還會執行另一個 ACTING。兩段尚未由正式介面
			// 執行器承接前，必須在任何 roster 變更之前停止。
			for _, call := range a.NativeSpawns {
				if call.Via == "spawn_group_with_intro" {
					return DialogLine{}, false, fmt.Errorf(
						"native intro spawn %s requires the 0x32999 transition and following acting adapter",
						call.Source,
					)
				}
			}
			for _, call := range a.NativeSpawns {
				if call.RawPlacementGate == nil {
					return DialogLine{}, false, fmt.Errorf("native spawn %s lacks raw placement gate", call.Source)
				}
				before := len(st.Units)
				if _, err := st.AppendGroupWithNativePlacement(
					call.Group, byte(*call.RawPlacementGate),
				); err != nil {
					return DialogLine{}, false, fmt.Errorf("native spawn %s: %w", call.Source, err)
				}
				for _, unit := range st.Units[before:] {
					if a.Camp != "" {
						unit.Camp = camp
					}
					unit.Acted = !a.ActImmediately
				}
			}
		} else {
			// 尚未遷移到 runtime_append_groups 的情境仍走明確標示的正規化
			// 相容路徑。資料即使已帶原版欄位，本分支也不能作為忠實度證據。
			for _, group := range a.Groups {
				st.SpawnGroup(group, camp, a.Camp != "", a.ActImmediately)
			}
		}
	case "join_party":
		sc.pendingJoins = append(sc.pendingJoins, a.CharID)
	case "dialogue":
		return DialogLine{Speaker: a.Speaker, Text: a.Text}, true, nil
	case "set_native_event_state":
		if st == nil || a.EventStateIndex == nil || *a.EventStateIndex < 0 ||
			*a.EventStateIndex >= len(st.NativeEventState) || a.EventStateValue == nil ||
			*a.EventStateValue < 0 || *a.EventStateValue > 0xff || a.NativeSource == "" ||
			a.NativeEventID == nil || *a.NativeEventID < 0 || *a.NativeEventID >= 90 {
			return DialogLine{}, false, fmt.Errorf("native event-state write lacks proven raw provenance")
		}
		st.NativeEventState[*a.EventStateIndex] = byte(*a.EventStateValue)
	case "set_flag":
		st.Flags[a.Flag] = true
	case "set_ai":
		// Editable scenario marker only. No planner currently consumes this
		// string, and "berserk" has not been mapped to native record +0x34.
		// Keep the inert marker visible instead of claiming an AI transition.
		st.Flags["ai_"+a.Unit+"_"+a.Mode] = true
	}
	return DialogLine{}, false, nil
}

// TakePartyJoins transfers JOIN effects from battle-script execution to the
// campaign-owned persistent roster. Scenario stays independent from the UI /
// save layer while preserving the original ordering (JOIN before SPAWN).
func (sc *Scenario) TakePartyJoins() []int {
	if sc == nil || len(sc.pendingJoins) == 0 {
		return nil
	}
	joins := append([]int(nil), sc.pendingJoins...)
	sc.pendingJoins = nil
	return joins
}

// PartyUnits materializes the persistent player roster in scenario order.
// Original chapter 0 constructs these units before any FDFIELD spawn, so this
// order is also their authoritative acting-slot order (slots 0..3).  The same
// constructor is shared by battle setup and handler cutscenes to prevent the
// two paths from drifting in stats, deployment cells, or identity.
func (sc *Scenario) PartyUnits(fallback []Cell) []*Unit {
	if sc == nil {
		return nil
	}
	units := make([]*Unit, 0, len(sc.Party))
	for i, pm := range sc.Party {
		x, y := 0, 0
		if i < len(sc.DeployCells) {
			x, y = sc.DeployCells[i][0], sc.DeployCells[i][1]
		} else if i < len(fallback) {
			x, y = fallback[i].X, fallback[i].Y
		}
		inventory, equipped, runtimeSlots := materializeInventory(pm.InventorySlots, pm.Inventory)
		u := &Unit{
			Camp: Own, Name: pm.Name, ClsName: pm.Cls, Lv: pm.Lv,
			HP: pm.HP, MaxHP: pm.HP, MP: pm.MP, MaxMP: pm.MP, AP: pm.AP, DP: pm.DP, DX: pm.DX, MV: pm.MV,
			HIT: pm.HIT, EV: pm.EV, CritPct: pm.CritPct,
			AtkMin: pm.AtkMin, AtkMax: pm.AtkMax,
			Portrait: pm.Portrait, Fig: pm.Fig,
			// Fresh native JOIN writes join_id to persistent +7/+8. Fig is the
			// authored JOIN identity here; class change later writes raw +7
			// explicitly and must not alter Fig/+8 identity.
			BattleFig: pm.Fig, HasBattleFig: true, MapSelectorKey: pm.Fig, HasMapSelectorKey: true,
			X: x, Y: y, OnField: true,
			Spells: append([]int(nil), pm.Spells...), Inventory: inventory, Equipped: equipped, InventorySlots: runtimeSlots,
			Dir: 0,
		}
		// The native constructor writes record byte +5 as zero for a newly
		// materialized unit. Keep that provenance explicit so handler predicates
		// can prefer the raw byte instead of inferring it from HP/OnField.
		if u.HP > 0 {
			u.NativeRecordByte5 = 0
			u.HasNativeRecordByte5 = true
		}
		// Party constructors use FDFIELD camp code 2 for native record +6.
		u.NativeRecordByte6 = 2
		u.HasNativeRecordByte6 = true
		if flags, flagErr := NativeInventoryFlagsFromSource(pm.InventorySlots); flagErr == nil {
			u.NativeInventoryFlags = flags
		}
		if pm.NativeIdentity != nil && *pm.NativeIdentity >= 0 && *pm.NativeIdentity <= 0xff {
			u.NativeIdentity = *pm.NativeIdentity
			u.HasNativeIdentity = true
		}
		if pm.NativeRecordRace != nil {
			u.NativeRecordRace, u.HasNativeRecordRace = *pm.NativeRecordRace, true
		}
		if pm.NativeRecordClass != nil {
			u.NativeRecordClass, u.HasNativeRecordClass = *pm.NativeRecordClass, true
		}
		// Editable scenario AP/DP/HIT/EV are already effective values (doc32),
		// so preserve them as the base for later shop purchases.
		u.BaseAP, u.BaseDP, u.BaseHIT, u.BaseEV, u.BaseMV = u.AP, u.DP, u.HIT, u.EV, u.MV
		u.BaseAtkMin, u.BaseAtkMax, u.EquipmentBaseSet = u.AtkMin, u.AtkMax, false
		// LoadScenario validates authored masks before this materialization. Keep
		// direct in-memory Scenario construction backward compatible: an invalid
		// field is not partially copied into a different command inventory.
		_ = u.SetInitialCommandMask(pm.InitialCommandMask)
		units = append(units, u)
	}
	return units
}
