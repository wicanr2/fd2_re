package main

import (
	"encoding/json"
	"os"

	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// recordInputAuditState 是唯讀的普通輸入觀測器；不指定關卡、亂數或遊戲狀態。
// 回放工具可等待真實 node／turn 邊界，避免固定延遲把按鍵送進下一個選單。
func (g *Game) recordInputAuditState() {
	path := os.Getenv("FD2_INPUT_AUDIT_LOG")
	if path == "" {
		return
	}
	keys := inpututil.AppendJustPressedKeys(nil)
	if len(keys) == 0 && g.frame%120 != 0 {
		return
	}
	r := map[string]any{"frame": g.frame, "keys": keys, "cursor": []int{g.curX, g.curY}, "camera": []float64{g.camX, g.camY}, "ring": g.ring, "moved": g.moved, "walking": g.walk != nil, "error": g.loadErr, "ai_busy": g.aiBusy, "dialogues": len(g.dialog), "move_preview": g.nativeMovePlan != nil}
	if g.camp != nil {
		r["node"] = g.camp.NodeID()
	}
	if g.st != nil {
		r["turn"] = g.st.Turn
		r["native_round"] = g.st.NativeRoundCounter
		r["view"] = g.st.NativeMapViewState
		units := make([]map[string]any, 0, len(g.st.Units))
		for _, u := range g.st.Units {
			if u == nil {
				continue
			}
			units = append(units, map[string]any{"name": u.Name, "x": u.X, "y": u.Y, "hp": u.HP, "mp": u.MP, "camp": u.Camp, "acted": u.Acted, "on_field": u.OnField, "raw": u.NativeMapPresentation})
		}
		r["units"] = units
	}
	if g.sel != nil {
		r["selected"] = g.sel.Name
		r["selection_origin"] = []int{g.selOrigX, g.selOrigY}
	}
	if s := g.nativeSystemInfoUI; s != nil {
		r["system_info_phase"] = s.phase
		r["system_info_frame"] = s.frame
	}
	r["next_player_index"] = g.nativeNextPlayerIndex
	r["player_focus"] = g.nativePlayerFocus
	if s := g.nativeSystemEndTurnUI; s != nil && s.treasure != nil {
		r["treasure_prompt"] = true
		r["treasure_choice"] = g.nativeSystemEndTurnConfirm
		r["treasure_ack"] = s.treasure.awaitAck
	}
	if s := g.nativePlayerStatus; s != nil {
		r["player_status_phase"] = s.phase
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	_ = json.NewEncoder(f).Encode(r)
}
