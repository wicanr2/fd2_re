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
	Frame             int     `json:"frame"`
	TitlePhase        string  `json:"title_phase,omitempty"`
	CampaignNode      string  `json:"campaign_node,omitempty"`
	Result            string  `json:"result,omitempty"`
	Cursor            [2]int  `json:"cursor"`
	HasSelection      bool    `json:"has_selection"`
	Selection         *[2]int `json:"selection,omitempty"`
	ActionOverlayOpen bool    `json:"action_overlay_open"`
	NativeCommandOpen bool    `json:"native_command_open"`
	SpellOpen         bool    `json:"spell_open"`
	// 這些欄位只描述輸入是否被既有 modal 阻擋；它們讓普通 X11 重播
	// 能區分「按鍵未到」和「遊戲刻意尚未接受按鍵」。
	NativeContinueOpeningConfirm bool                       `json:"native_continue_opening_confirm"`
	NativeContinueCursorOverlay  bool                       `json:"native_continue_cursor_overlay"`
	NativeEnding                 *screenshotEndingTrace     `json:"native_ending,omitempty"`
	DialogCount                  int                        `json:"dialog_count"`
	BattleEventActive            bool                       `json:"battle_event_active"`
	NativeTurnStagingActive      bool                       `json:"native_turn_staging_active"`
	CursorUnit                   *screenshotCursorUnitTrace `json:"cursor_unit,omitempty"`
	LoadError                    string                     `json:"load_error,omitempty"`
	Battle                       *screenshotBattleTrace     `json:"battle,omitempty"`
}

// screenshotCursorUnitTrace 只輸出目前游標格的互動資格，避免把角色名稱或
// 推測性語意混進截圖證據。camp 保留 battle.Camp 的原始整數值。
type screenshotCursorUnitTrace struct {
	Camp               int  `json:"camp"`
	OnField            bool `json:"on_field"`
	Acted              bool `json:"acted"`
	Paralyzed          bool `json:"paralyzed"`
	NativeCommandCount int  `json:"native_command_count"`
}

type screenshotBattleTrace struct {
	Width              int                        `json:"width"`
	Height             int                        `json:"height"`
	Turn               int                        `json:"turn"`
	NativeRoundCounter int                        `json:"native_round_counter"`
	NativeMapView      *battle.NativeMapViewState `json:"native_map_view,omitempty"`
}

// screenshotEndingTrace 保存結局截圖所在的精確 raw 邊界。它刻意只記錄狀態，
// 不能把已還原前綴或近似戰役回退升格成完整結局宣告。
type screenshotEndingTrace struct {
	PlaybackState       string `json:"playback_state"`
	BlockedOp           string `json:"blocked_op,omitempty"`
	BlockedSource       string `json:"blocked_source,omitempty"`
	CampaignApproximate bool   `json:"campaign_approximate"`
	AudioCueConsumed    bool   `json:"audio_cue_consumed"`
}

func (g *Game) writeShotStateTrace(path string) error {
	if g == nil || path == "" {
		return errors.New("截圖狀態追蹤輸出路徑不可用")
	}
	trace := screenshotStateTrace{
		Frame:                        g.frame,
		TitlePhase:                   g.titlePhase,
		Result:                       g.result,
		Cursor:                       [2]int{g.curX, g.curY},
		ActionOverlayOpen:            g.ring,
		NativeCommandOpen:            g.nativeCommandOpen,
		SpellOpen:                    g.spellOpen,
		NativeContinueOpeningConfirm: g.nativeContinueOpeningConfirm,
		NativeContinueCursorOverlay:  g.nativeContinueCursorOverlay,
		DialogCount:                  len(g.dialog),
		BattleEventActive:            g.battleEvent != nil,
		NativeTurnStagingActive:      g.nativeTurnStaging != nil,
		LoadError:                    g.loadErr,
	}
	if g.camp != nil {
		trace.CampaignNode = g.camp.NodeID()
	}
	if p := g.nativeEnding; p != nil && p.player != nil {
		endingTrace := &screenshotEndingTrace{
			PlaybackState:       string(p.player.State),
			CampaignApproximate: p.campaignApproximate,
			AudioCueConsumed:    p.audioCueConsumed,
		}
		if p.player.Blocked != nil {
			endingTrace.BlockedOp = p.player.Blocked.Op
			endingTrace.BlockedSource = p.player.Blocked.Source
		}
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
				Camp:               int(unit.Camp),
				OnField:            unit.OnField,
				Acted:              unit.Acted,
				Paralyzed:          unit.Paralyzed,
				NativeCommandCount: len(unit.NativeCommandIDs()),
			}
		}
	}
	encoded, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(encoded, '\n'), 0o644)
}
