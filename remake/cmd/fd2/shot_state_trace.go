package main

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// screenshotStateTrace 是截圖同狀態驗證的旁車紀錄。它只在明確指定
// FD2_SHOT_STATE 時寫出，不改變遊戲控制流或畫面；目的是讓 X11 輸入擷取可證明
// 自己實際抵達的節點、游標與操作介面，而不是以一張外觀相近的 PNG 冒充同狀態證據。
type screenshotStateTrace struct {
	Frame                  int                      `json:"frame"`
	TitlePhase             string                   `json:"title_phase,omitempty"`
	TitleSelection         int                      `json:"title_selection"`
	TitleSlotSelection     int                      `json:"title_slot_selection"`
	TitleCut               *screenshotTitleCutTrace `json:"title_cut,omitempty"`
	CampaignNode           string                   `json:"campaign_node,omitempty"`
	Result                 string                   `json:"result,omitempty"`
	Message                string                   `json:"message,omitempty"`
	Gold                   int                      `json:"gold"`
	PartyJoinOrder         []int                    `json:"party_join_order,omitempty"`
	PartyRoster            map[int]battle.Unit      `json:"party_roster,omitempty"`
	Cursor                 [2]int                   `json:"cursor"`
	HasSelection           bool                     `json:"has_selection"`
	Selection              *[2]int                  `json:"selection,omitempty"`
	ActionOverlayOpen      bool                     `json:"action_overlay_open"`
	NativeCommandOpen      bool                     `json:"native_command_open"`
	NativeCommandTargeting bool                     `json:"native_command_targeting"`
	NativeCommandTargetID  *int                     `json:"native_command_target_id,omitempty"`
	NativeItemTargeting    bool                     `json:"native_item_targeting"`
	NativeItemTargetID     *int                     `json:"native_item_target_id,omitempty"`
	NativeItemRelocating   bool                     `json:"native_item_relocating"`
	ItemOpen               bool                     `json:"item_open"`
	ItemSelection          int                      `json:"item_selection"`
	SpellOpen              bool                     `json:"spell_open"`
	// 這些欄位只描述輸入是否被既有 modal 阻擋；它們讓普通 X11 重播
	// 能區分「按鍵未到」和「遊戲刻意尚未接受按鍵」。
	NativeContinueOpeningConfirm bool                        `json:"native_continue_opening_confirm"`
	NativeContinueCursorOverlay  bool                        `json:"native_continue_cursor_overlay"`
	NativeEnding                 *screenshotEndingTrace      `json:"native_ending,omitempty"`
	Church                       *screenshotChurchTrace      `json:"church,omitempty"`
	Preparation                  *screenshotPreparationTrace `json:"preparation,omitempty"`
	DialogCount                  int                         `json:"dialog_count"`
	BattleEventActive            bool                        `json:"battle_event_active"`
	NativeTurnStagingActive      bool                        `json:"native_turn_staging_active"`
	CursorUnit                   *screenshotCursorUnitTrace  `json:"cursor_unit,omitempty"`
	LoadError                    string                      `json:"load_error,omitempty"`
	Battle                       *screenshotBattleTrace      `json:"battle,omitempty"`
}

// screenshotChurchTrace 固定教會主選單可見的原始選擇、脈衝與金幣狀態，
// 避免把動畫相位不同造成的像素差誤判為排版或 renderer 缺陷。
type screenshotChurchTrace struct {
	Mode      string `json:"mode"`
	Selection int    `json:"selection"`
	Pulse     int    `json:"pulse"`
	Gold      int    `json:"gold"`
	ShotHold  bool   `json:"shot_hold"`
}

type screenshotPreparationTrace struct {
	Selecting        bool `json:"selecting"`
	Confirming       bool `json:"confirming"`
	ConfirmSelection int  `json:"confirm_selection"`
	Selection        int  `json:"selection"`
	Limit            int  `json:"limit"`
	CandidateCount   int  `json:"candidate_count"`
	DeployedCount    int  `json:"deployed_count"`
	TownBacked       bool `json:"town_backed"`
}

// screenshotTitleCutTrace 讓有界 Xvfb 擷取能證明自己位於哪一個原版排程邊界；
// 它只旁車記錄既有狀態，不提供跳段或改寫正式開場的入口。
type screenshotTitleCutTrace struct {
	Step       int     `json:"step"`
	Kind       string  `json:"kind"`
	Resource   int     `json:"resource,omitempty"`
	Frame      int     `json:"frame,omitempty"`
	Tick       int     `json:"tick"`
	ScrollY    float64 `json:"scroll_y,omitempty"`
	ScrollFrom int     `json:"scroll_from,omitempty"`
	ScrollTo   int     `json:"scroll_to,omitempty"`
}

// screenshotCursorUnitTrace 只輸出目前游標格的互動資格，避免把角色名稱或
// 推測性語意混進截圖證據。camp 保留 battle.Camp 的原始整數值。
type screenshotCursorUnitTrace struct {
	Camp                 int   `json:"camp"`
	OnField              bool  `json:"on_field"`
	Acted                bool  `json:"acted"`
	Paralyzed            bool  `json:"paralyzed"`
	NativeCommandCount   int   `json:"native_command_count"`
	HP                   int   `json:"hp"`
	MaxHP                int   `json:"max_hp"`
	MP                   int   `json:"mp"`
	MaxMP                int   `json:"max_mp"`
	InventorySlots       []int `json:"inventory_slots,omitempty"`
	NativeInventoryFlags []int `json:"native_inventory_flags,omitempty"`
}

type screenshotBattleTrace struct {
	Width              int                        `json:"width"`
	Height             int                        `json:"height"`
	Turn               int                        `json:"turn"`
	NativeRoundCounter int                        `json:"native_round_counter"`
	NativeMapView      *battle.NativeMapViewState `json:"native_map_view,omitempty"`
}

// screenshotEndingTrace 保存結局截圖所在的精確 raw 邊界。它刻意只記錄狀態，
// 不能把已還原前綴或來源約束 E1 升格成完整結局 E2 宣告。
type screenshotEndingTrace struct {
	PlaybackState       string `json:"playback_state"`
	BlockedOp           string `json:"blocked_op,omitempty"`
	BlockedSource       string `json:"blocked_source,omitempty"`
	CampaignSourceBound bool   `json:"campaign_source_bound"`
	AudioCueConsumed    bool   `json:"audio_cue_consumed"`
	MontagePhase        string `json:"montage_phase,omitempty"`
	MontagePlanIndex    *int   `json:"montage_plan_index,omitempty"`
	MontagePlanCount    int    `json:"montage_plan_count,omitempty"`
	MontageInputPending bool   `json:"montage_input_pending,omitempty"`
	MontageStartError   string `json:"montage_start_error,omitempty"`
	DialoguePhase       string `json:"dialogue_phase,omitempty"`
	DialogueBlock       *int   `json:"dialogue_block,omitempty"`
	DialogueBlockCount  int    `json:"dialogue_block_count,omitempty"`
	DialogueUtterance   *int   `json:"dialogue_utterance,omitempty"`
	DialoguePage        *int   `json:"dialogue_page,omitempty"`
	DialogueWaiting     bool   `json:"dialogue_waiting,omitempty"`
}

func (g *Game) writeShotStateTrace(path string) error {
	if g == nil || path == "" {
		return errors.New("截圖狀態追蹤輸出路徑不可用")
	}
	trace := screenshotStateTrace{
		Frame:                        g.frame,
		TitlePhase:                   g.titlePhase,
		TitleSelection:               g.titleSel,
		TitleSlotSelection:           g.titleSlotSel,
		Result:                       g.result,
		Message:                      g.msg,
		Gold:                         g.gold,
		PartyJoinOrder:               append([]int(nil), g.partyJoinOrder...),
		PartyRoster:                  g.partyRoster,
		Cursor:                       [2]int{g.curX, g.curY},
		ActionOverlayOpen:            g.ring,
		NativeCommandOpen:            g.nativeCommandOpen,
		NativeCommandTargeting:       g.nativeCommand0Targeting,
		NativeItemTargeting:          g.nativeItemTargeting,
		NativeItemRelocating:         g.nativeItemRelocating,
		ItemOpen:                     g.itemOpen,
		ItemSelection:                g.itemSel,
		SpellOpen:                    g.spellOpen,
		NativeContinueOpeningConfirm: g.nativeContinueOpeningConfirm,
		// JSON field name is retained for compatibility with the 2026-08-11
		// chapter0 trace; runtime ownership is now the shared 0x117E7 path.
		NativeContinueCursorOverlay: g.nativeSystemCursorOverlay,
		DialogCount:                 len(g.dialog),
		BattleEventActive:           g.battleEvent != nil,
		NativeTurnStagingActive:     g.nativeTurnStaging != nil,
		LoadError:                   g.loadErr,
	}
	if g.titlePhase == "cutscene" && g.cutIdx >= 0 && g.cutIdx < len(cutScript) {
		step := cutScript[g.cutIdx]
		trace.TitleCut = &screenshotTitleCutTrace{
			Step:       g.cutIdx,
			Kind:       step.kind,
			Resource:   step.res,
			Frame:      g.cutFrame,
			Tick:       g.cutTick,
			ScrollY:    g.scrollY,
			ScrollFrom: step.scrollFrom,
			ScrollTo:   step.scrollTo,
		}
	}
	if g.camp != nil {
		if node := g.camp.Node(); node != nil && node.Type == "church" {
			trace.Church = &screenshotChurchTrace{
				Mode: g.churchMode, Selection: g.churchSel,
				Pulse: g.nativeChurchUIPulse, Gold: g.gold,
				ShotHold: g.nativeChurchUIShotHold,
			}
		} else if node != nil && node.Type == "preparation" {
			deployed := 0
			for _, selected := range g.partyDeploy {
				if selected {
					deployed++
				}
			}
			trace.Preparation = &screenshotPreparationTrace{
				Selecting: g.prepSelecting, Confirming: g.prepConfirm,
				ConfirmSelection: g.prepConfirmSel, Selection: g.prepSel,
				Limit: g.prepLimit, CandidateCount: len(g.prepIDs),
				DeployedCount: deployed, TownBacked: node.Cancel != "",
			}
		}
	}
	if g.nativeCommand0Targeting {
		commandID := g.nativeCommandTargetID
		trace.NativeCommandTargetID = &commandID
	}
	if g.nativeItemTargeting {
		itemID := g.nativeItemTargetID
		trace.NativeItemTargetID = &itemID
	}
	if g.camp != nil {
		trace.CampaignNode = g.camp.NodeID()
	}
	if p := g.nativeEnding; p != nil && p.player != nil {
		endingTrace := &screenshotEndingTrace{
			PlaybackState:       string(p.player.State),
			CampaignSourceBound: p.campaignSourceBound,
			AudioCueConsumed:    p.audioCueConsumed,
		}
		if p.player.Blocked != nil {
			endingTrace.BlockedOp = p.player.Blocked.Op
			endingTrace.BlockedSource = p.player.Blocked.Source
		}
		if p.montage != nil {
			endingTrace.MontagePhase = string(p.montage.Phase)
			index := p.montage.PlanIndex
			endingTrace.MontagePlanIndex = &index
			endingTrace.MontagePlanCount = len(p.montage.Plans)
			endingTrace.MontageInputPending = p.montageInputPending
		}
		if p.dialogue != nil {
			endingTrace.DialoguePhase = p.dialogue.Phase.String()
			block, utterance, page := p.dialogue.Block, p.dialogue.Utterance, p.dialogue.Page
			endingTrace.DialogueBlock = &block
			endingTrace.DialogueBlockCount = len(p.dialogue.Blocks)
			endingTrace.DialogueUtterance = &utterance
			endingTrace.DialoguePage = &page
			endingTrace.DialogueWaiting = p.dialogue.Waiting()
		}
		endingTrace.MontageStartError = p.montageStartError
		trace.NativeEnding = endingTrace
	}
	if g.sel != nil {
		trace.HasSelection = true
		selection := [2]int{g.sel.X, g.sel.Y}
		trace.Selection = &selection
	}
	if g.st != nil {
		state := &screenshotBattleTrace{
			Width:              g.st.W,
			Height:             g.st.H,
			Turn:               g.st.Turn,
			NativeRoundCounter: g.st.NativeRoundCounter,
		}
		if g.st.HasNativeMapViewState {
			view := g.st.NativeMapViewState
			state.NativeMapView = &view
		}
		trace.Battle = state
		if unit := g.st.UnitAt(g.curX, g.curY); unit != nil {
			trace.CursorUnit = &screenshotCursorUnitTrace{
				Camp:                 int(unit.Camp),
				OnField:              unit.OnField,
				Acted:                unit.Acted,
				Paralyzed:            unit.Paralyzed,
				NativeCommandCount:   len(unit.NativeCommandIDs()),
				HP:                   unit.HP,
				MaxHP:                unit.MaxHP,
				MP:                   unit.MP,
				MaxMP:                unit.MaxMP,
				InventorySlots:       append([]int(nil), unit.InventorySlots...),
				NativeInventoryFlags: append([]int(nil), unit.NativeInventoryFlags...),
			}
		}
	}
	encoded, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(encoded, '\n'), 0o644)
}
