package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

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
	released := inpututil.AppendJustReleasedKeys(nil)
	interval := 120
	if value, err := strconv.Atoi(os.Getenv("FD2_INPUT_AUDIT_INTERVAL")); err == nil && value >= 1 && value <= 120 {
		interval = value
	}
	if len(keys) == 0 && len(released) == 0 && g.frame%interval != 0 {
		return
	}
	r := map[string]any{"frame": g.frame, "keys": keys, "cursor": []int{g.curX, g.curY}, "camera": []float64{g.camX, g.camY}, "ring": g.ring, "ring_sel": g.ringSel, "ring_phase": g.actionOverlayPhase, "ring_blink": g.actionOverlayBlink.Phase, "banner": g.banner, "banner_t": g.bannerT, "moved": g.moved, "walking": g.walk != nil, "error": g.loadErr, "ai_busy": g.aiBusy, "dialogues": len(g.dialog), "move_preview": g.nativeMovePlan != nil}
	r["released_keys"] = released
	r["modern_theme"] = g.modernStoryPortraits != nil
	r["hd_theme"] = g.modernStoryPortraits != nil && g.modernStoryPortraits.hasHD
	r["battle_scene"] = g.atk != nil
	r["battle_event"] = g.battleEvent != nil
	r["level_up_dialogue"] = g.nativeLevelUpDialogue != nil
	r["hd_timed_dialogue"] = g.modernHDTimedDialogueVisible
	if g.modernHDTimedDialogueVisible {
		r["hd_timed_phase"] = g.modernHDTimedDialoguePhase
		r["hd_timed_index"] = g.modernHDTimedDialogueIndex
	}
	if s := g.nativeLevelUpDialogue; s != nil {
		r["timed_dialogue_owner"] = fmt.Sprintf("%p", s)
		r["timed_dialogue_wait"] = g.nativeLevelUpDialogueAtWait()
	}
	r["death_reward_ui"] = g.nativeDeathRewardUI != nil
	r["transient_ui"] = g.transientUI
	r["class_ui_job"] = g.nativeClassUIJob != nil
	if g.nativeMovePlan != nil {
		cells := make([][]int, 0, len(g.reach))
		for cell := range g.reach {
			cells = append(cells, []int{cell.X, cell.Y})
		}
		r["move_reachable"] = cells
	}
	r["hd_world"] = g.modernHDWorldVisible && g.modernHDMapGPU != nil && g.modernHDMapGPU.Theme == g.modernStoryPortraits
	if r["hd_world"] == true && g.modernHDMapPresented != nil {
		r["hd_world_frame"] = g.modernHDMapPresentedFrame
		units := make([]map[string]any, 0, len(g.modernHDMapPresented.Units))
		for _, unit := range g.modernHDMapPresented.Units {
			units = append(units, map[string]any{"group": unit.Group, "frame": unit.Frame, "rect": []int{unit.Rect.Min.X, unit.Rect.Min.Y, unit.Rect.Max.X, unit.Rect.Max.Y}, "work_offset": unit.WorkOffset, "palette_band": unit.PaletteBand})
		}
		r["hd_world_units"] = units
		terrain := make([]map[string]any, 0, len(g.modernHDMapPresented.Terrain))
		for _, tile := range g.modernHDMapPresented.Terrain {
			hash := sha256.Sum256(tile.Image.Pix)
			terrain = append(terrain, map[string]any{"frame": tile.Frame,
				"rect":        []int{tile.Rect.Min.X, tile.Rect.Min.Y, tile.Rect.Max.X, tile.Rect.Max.Y},
				"rgba_sha256": fmt.Sprintf("%x", hash)})
		}
		r["hd_world_terrain"] = terrain
		foreground := 0
		foregroundDraws := make([]map[string]any, 0)
		for _, layer := range g.modernHDMapPresented.Foreground {
			if layer.HDImage != nil {
				foreground++
				hash := sha256.Sum256(layer.HDImage.Pix)
				foregroundDraws = append(foregroundDraws, map[string]any{"frame": layer.SourceFrame,
					"rect":        []int{layer.Rect.Min.X, layer.Rect.Min.Y, layer.Rect.Max.X, layer.Rect.Max.Y},
					"rgba_sha256": fmt.Sprintf("%x", hash)})
			}
		}
		r["hd_world_terrain_foreground"] = foreground
		r["hd_world_terrain_foreground_draws"] = foregroundDraws
	}
	r["locale"] = g.localeID
	r["gold"] = g.gold
	r["shop_mode"] = g.nativeShopMode
	r["shop_service"] = g.nativeShopServiceSel
	r["shop_ui_busy"] = g.nativeShopUIJob != nil
	if job := g.nativeShopUIJob; job != nil && job.goldText != nil && len(job.timeline) > 0 {
		index, _ := job.timelinePosition()
		r["gold_roll_job"] = fmt.Sprintf("%p", job)
		r["gold_roll_index"] = index
		r["gold_roll_rows"] = job.goldText.windows[index].Rows
		r["gold_roll_credit"] = job.goldText.credit
	}
	r["item_selection"] = g.itemSel
	r["item_anim_step"] = g.itemAnimStep
	r["item_closing"] = g.itemClosing
	if g.nativeItemPanelText != nil && (g.itemClosing || g.itemAnimStep < 11) {
		index := 11 - g.itemAnimStep
		if g.itemClosing {
			index = g.itemAnimStep
		}
		r["panel_transition"] = map[string]any{"domain": "item", "job": fmt.Sprintf("%p", g.nativeItemPanelText),
			"kind": "item", "index": index, "closing": g.itemClosing}
	}
	if job := g.nativeClassUIJob; job != nil && job.statusText != nil && job.frame < len(job.frames) {
		r["panel_transition"] = map[string]any{"domain": "church", "job": fmt.Sprintf("%p", job),
			"kind": int(job.statusText.kind), "index": job.frame, "identity": g.churchStatusID}
	}
	if s := g.nativePlayerStatus; s != nil && s.text != nil && len(s.frames) > 0 {
		r["panel_transition"] = map[string]any{"domain": "battle", "job": fmt.Sprintf("%p", s.text),
			"kind": int(s.text.kind), "index": s.frame}
	}
	if g.nativeShopMode == nativeShopEquipPanelMode {
		if id, unit, ok := g.nativeShopEquipUnit(); ok {
			r["equip_unit"] = map[string]any{"id": id, "ap": unit.AP, "dp": unit.DP,
				"equipped": unit.Equipped, "raw_flags": unit.NativeInventoryFlags, "inventory": unit.Inventory}
		}
	}
	r["hotel_mode"] = g.nativeHotelMode
	r["hotel_selection"] = g.hotelSel
	r["hotel_slot"] = g.nativeHotelSlotSel
	r["church_mode"] = g.churchMode
	r["church_selection"] = g.churchSel
	r["church_status_id"] = g.churchStatusID
	r["church_ui_busy"] = g.nativeChurchUIJob != nil || g.nativeClassUIJob != nil
	if g.modernHDText != nil {
		r["hd_text_runs"] = len(g.modernHDText.runs)
	}
	r["dialogue_progress"] = g.nativeDialogueProgress
	r["dialogue_phase"] = g.dlgPhase
	r["title_phase"] = g.titlePhase
	r["title_selection"] = g.titleSel
	r["title_slot"] = g.titleSlotSel
	r["preparation_selecting"] = g.prepSelecting
	r["preparation_confirm"] = g.prepConfirm
	r["preparation_required"] = g.prepRequiredMissing
	r["preparation_choice"] = g.prepConfirmSel
	r["preparation_slots"] = g.prepRecordSlots
	r["preparation_slot"] = g.prepRecordSlotSel
	r["preparation_selection"] = g.prepSel
	r["preparation_selected"] = g.preparationSelected()
	r["preparation_limit"] = g.prepLimit
	r["preparation_ui_busy"] = g.nativeClassUIJob != nil
	if job := g.nativeClassUIJob; job != nil && job.preparationText != nil {
		r["preparation_transition_job"] = fmt.Sprintf("%p", job)
		r["preparation_transition_frame"] = job.frame
		r["preparation_transition_text_start"] = job.preparationText.start
		r["preparation_transition_required"] = job.preparationText.kind == nativePreparationRequiredQuestion
		r["preparation_transition_closing"] = job.preparationText.closing
	}
	r["help_visible"] = g.helpVisible
	if len(g.dialog) != 0 {
		r["dialogue_speaker"] = g.dialog[len(g.dialog)-1].Speaker
		r["dialogue_native"] = g.dialog[len(g.dialog)-1].NativeDialogue != nil
	}
	if overlay := g.modernHDOverlay; overlay != nil {
		r["hd_overlay_rect"] = []int{overlay.rect.Min.X, overlay.rect.Min.Y, overlay.rect.Dx(), overlay.rect.Dy()}
		r["hd_overlay_mirrored"] = overlay.mirrorX
	}
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
			units = append(units, map[string]any{"name": u.Name, "x": u.X, "y": u.Y, "hp": u.HP, "mp": u.MP, "lv": u.Lv, "exp": u.Exp, "camp": u.Camp, "acted": u.Acted, "on_field": u.OnField, "raw": u.NativeMapPresentation})
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
	r["system_end_turn_confirm"] = g.nativeSystemEndTurnConfirm
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
