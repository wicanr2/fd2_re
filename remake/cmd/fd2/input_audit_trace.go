package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
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
	g.recordNativeDeathAuditState(r)
	r["command_scene"] = g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil || g.nativeCmd2Presentation != nil ||
		g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil || g.nativeCmd6Presentation != nil || g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil ||
		g.nativeCmd24Presentation != nil || g.nativeCmd29Presentation != nil
	r["command_targeting"] = g.nativeCommand0Targeting
	r["command_grid"], r["command_selection"] = g.nativeCommandOpen, g.nativeCommandSel
	r["command_target_id"] = g.nativeCommandTargetID
	r["heal_scene"] = g.nativeHealPresentation != nil
	r["system_cursor_overlay"] = g.nativeSystemCursorOverlay
	r["system_nested_open"] = g.nativeSystemNestedOpen
	if j := g.nativeHealPresentation; j != nil {
		r["heal_command_id"], r["heal_owner"] = j.commandID, fmt.Sprintf("%p", j)
		r["heal_phase"], r["heal_frame"] = int(j.phase), j.frame
		targets := make([]int, len(j.targets))
		for i, target := range j.targets {
			targets[i] = target.RecordIndex
		}
		r["heal_target_indices"] = targets
		results := make([]map[string]int, len(j.results))
		for i, result := range j.results {
			results[i] = map[string]int{"rolled": result.Restore.Rolled, "actual": result.Restore.Actual}
		}
		r["heal_results"] = results
	}
	if g.nativeCommand0Targeting && g.st != nil && g.st.W > 0 {
		cells := make([][]int, 0)
		for index, value := range g.st.NativeTileBlitModes {
			if value != 0xff {
				cells = append(cells, []int{index % g.st.W, index / g.st.W})
			}
		}
		r["command_target_cells"] = cells
	}
	if j := g.nativeCmd0Presentation; j != nil {
		r["command_id"], r["command_owner"] = 0, fmt.Sprintf("%p", j)
		r["command_actor_fig"], r["command_target_fig"] = j.actor.BattleFig, j.target.BattleFig
		r["command_frame"], r["command_pulse_black"] = j.frame, j.pulseBlack
		if frame := j.modernFrame(); frame != nil {
			r["command_phase"] = frame.hdPhase
			r["command_source_layers"] = modernBattleAuditLayers(frame.hdSources)
		}
	}
	if j := g.nativeCmd1Presentation; j != nil {
		r["command_id"], r["command_owner"] = 1, fmt.Sprintf("%p", j)
		r["command_actor_fig"] = j.actor.BattleFig
		if j.plan != nil && len(j.plan.Results) > 0 && j.plan.Results[0].Target != nil {
			r["command_target_fig"] = j.plan.Results[0].Target.BattleFig
		}
		r["command_frame"], r["command_pulse_black"] = j.frame, j.pulseBlack
		if frame := j.modernFrame(); frame != nil {
			r["command_phase"] = frame.hdPhase
			r["command_source_layers"] = modernBattleAuditLayers(frame.hdSources)
		}
	}
	if j := g.nativeCmd2Presentation; j != nil {
		r["command_id"], r["command_owner"] = 2, fmt.Sprintf("%p", j)
		r["command_actor_fig"] = j.actor.BattleFig
		if j.plan != nil && len(j.plan.Results) > 0 && j.plan.Results[0].Target != nil {
			r["command_target_fig"] = j.plan.Results[0].Target.BattleFig
		}
		r["command_frame"], r["command_pulse_black"] = j.frame, j.pulseBlack
		if frame := j.modernFrame(); frame != nil {
			r["command_phase"] = frame.hdPhase
			r["command_source_layers"] = modernBattleAuditLayers(frame.hdSources)
		}
	}
	if j := g.nativeCmd3Presentation; j != nil {
		r["command_id"], r["command_owner"] = 3, fmt.Sprintf("%p", j)
		r["command_actor_fig"] = j.actor.BattleFig
		if j.plan != nil && len(j.plan.Results) > 0 && j.plan.Results[0].Target != nil {
			r["command_target_fig"] = j.plan.Results[0].Target.BattleFig
		}
		r["command_frame"], r["command_pulse_black"] = j.frame, j.pulseBlack
		if frame := j.modernFrame(); frame != nil {
			r["command_phase"] = frame.hdPhase
			r["command_source_layers"] = modernBattleAuditLayers(frame.hdSources)
		}
	}
	if j := g.nativeCmd6Presentation; j != nil {
		r["command_id"], r["command_owner"] = 6, fmt.Sprintf("%p", j)
		r["command_actor_fig"] = j.actor.BattleFig
		if j.plan != nil && len(j.plan.Results) > 0 && j.plan.Results[0].Target != nil {
			r["command_target_fig"] = j.plan.Results[0].Target.BattleFig
		}
		r["command_frame"], r["command_pulse_black"] = j.frame, j.pulseBlack
		if frame := j.modernFrame(); frame != nil {
			r["command_phase"] = frame.hdPhase
			r["command_source_layers"] = modernBattleAuditLayers(frame.hdSources)
		}
	}
	if j := g.nativeCmd7Presentation; j != nil {
		r["command_id"], r["command_owner"] = 7, fmt.Sprintf("%p", j)
		r["command_actor_fig"] = j.actor.BattleFig
		if j.plan != nil && len(j.plan.Results) > 0 && j.plan.Results[0].Target != nil {
			r["command_target_fig"] = j.plan.Results[0].Target.BattleFig
		}
		r["command_frame"], r["command_pulse_black"] = j.frame, j.pulseBlack
		if frame := j.modernFrame(); frame != nil {
			r["command_phase"] = frame.hdPhase
			r["command_source_layers"] = modernBattleAuditLayers(frame.hdSources)
		}
	}
	if j := g.nativeCmd8Presentation; j != nil {
		r["command_id"], r["command_owner"] = 8, fmt.Sprintf("%p", j)
		r["command_actor_fig"] = j.actor.BattleFig
		if j.plan != nil && len(j.plan.Results) > 0 && j.plan.Results[0].Target != nil {
			r["command_target_fig"] = j.plan.Results[0].Target.BattleFig
		}
		r["command_frame"], r["command_pulse_black"] = j.frame, j.pulseBlack
		if frame := j.modernFrame(); frame != nil {
			r["command_phase"] = frame.hdPhase
			r["command_source_layers"] = modernBattleAuditLayers(frame.hdSources)
		}
	}

	if j := g.nativeCmd24Presentation; j != nil {
		r["command_id"], r["command_owner"] = 24, fmt.Sprintf("%p", j)
		r["command_actor_fig"], r["command_target_fig"] = j.actor.BattleFig, j.target.BattleFig
		r["command_frame"] = j.frame
		if frame := j.modernFrame(); frame != nil {
			r["command_phase"] = frame.hdPhase
			r["command_source_layers"] = modernBattleAuditLayers(frame.hdSources)
		}
	}
	if j := g.nativeCmd29Presentation; j != nil {
		r["command_id"], r["command_owner"] = j.schedule.CommandID, fmt.Sprintf("%p", j)
		r["command_mp_published"], r["command_target_count"] = j.mpPublished, len(j.targets)
		r["command_actor_fig"], r["command_target_index"] = j.actor.BattleFig, j.targetIndex
		if j.targetIndex >= 0 && j.targetIndex < len(j.targets) && j.targets[j.targetIndex].unit != nil {
			r["command_target_fig"] = j.targets[j.targetIndex].unit.BattleFig
		}
		r["command_frame"] = j.frame
		if frame := j.modernFrame(); frame != nil {
			r["command_phase"] = frame.hdPhase
			r["command_source_layers"] = modernBattleAuditLayers(frame.hdSources)
		}
	}
	r["battle_event"] = g.battleEvent != nil
	if j := g.nativeCmd5Presentation; j != nil {
		r["command_id"], r["command_owner"] = j.commandID, fmt.Sprintf("%p", j)
		r["command_actor_fig"] = j.actor.BattleFig
		if j.plan != nil && len(j.plan.Results) > 0 && j.plan.Results[0].Target != nil {
			r["command_target_fig"] = j.plan.Results[0].Target.BattleFig
		}
		r["command_frame"], r["command_pulse_black"] = j.frame, j.pulseBlack
		if frame := j.modernFrame(); frame != nil {
			r["command_phase"] = frame.hdPhase
			r["command_source_layers"] = modernBattleAuditLayers(frame.hdSources)
		}
	}
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
	r["death_reward_message"] = g.nativeSystemEndTurnUI != nil && g.nativeSystemEndTurnUI.rewardMessage != nil
	r["death_reward_message_wait"] = g.nativeDeathRewardMessageAwaitingKey()
	r["transient_ui"] = g.transientUI
	r["class_ui_job"] = g.nativeClassUIJob != nil
	r["system_end_turn_ui"] = g.nativeSystemEndTurnUI != nil
	r["native_map_dac_black"] = nativeDACIsBlack(g.nativeMapDAC)
	r["native_full_dac_black"] = g.nativeFullDACBlack
	if job := g.nativeClassUIJob; job != nil {
		r["class_ui_frame"] = job.frame
		r["class_ui_frames"] = len(job.frames)
		r["class_ui_restore_length"] = len(job.restore)
		r["class_ui_drawn"] = job.drawn
		r["class_ui_timeline_steps"] = len(job.timeline)
		r["class_ui_elapsed_ns"] = int64(job.elapsed)
	}
	if g.nativeMovePlan != nil {
		cells := make([][]int, 0, len(g.reach))
		for cell := range g.reach {
			cells = append(cells, []int{cell.X, cell.Y})
		}
		r["move_reachable"] = cells
	}
	r["hd_battle"] = g.modernHDBattleVisible && g.modernHDBattleGPU != nil && g.modernHDBattleGPU.theme == g.modernStoryPortraits
	if g.atk != nil && g.atk.nativeScene != nil && g.nativePaletteRamp == nil {
		scene := g.atk.nativeScene
		r["battle_owner"] = fmt.Sprintf("%p", scene)
		job, phase, layers := physicalBattleAuditSources(scene)
		r["battle_actor_fig"], r["battle_target_fig"] = scene.actorBattleFig, scene.targetBattleFig
		if job >= 0 {
			r["battle_body_job"], r["battle_phase"], r["battle_source_layers"] = job, phase, layers
		}
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
		if g.nativeHealPresentation != nil {
			overlays := make([]map[string]any, 0, len(g.modernHDMapPresented.Overlays))
			for _, layer := range g.modernHDMapPresented.Overlays {
				hash := sha256.Sum256(layer.Image.Pix)
				overlays = append(overlays, map[string]any{"entry": layer.Entry,
					"rect":        []int{layer.Rect.Min.X, layer.Rect.Min.Y, layer.Rect.Max.X, layer.Rect.Max.Y},
					"rgba_sha256": fmt.Sprintf("%x", hash)})
			}
			r["hd_heal_overlays"] = overlays
		}
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
			units = append(units, map[string]any{"name": u.Name, "x": u.X, "y": u.Y, "hp": u.HP, "mp": u.MP, "lv": u.Lv, "exp": u.Exp, "camp": u.Camp, "acted": u.Acted, "on_field": u.OnField, "raw": u.NativeMapPresentation,
				"native_identity": u.NativeIdentity, "battle_fig": u.BattleFig, "attack_min": u.AtkMin, "attack_max": u.AtkMax,
				"command_ids": u.NativeCommandIDs()})
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
		r["system_info_owner"] = fmt.Sprintf("%p", s)
		r["system_info_hd"] = s.hdPresented
		r["system_info_values"] = s.input
		if regions, err := fdother.NativeSystemInfoRegions(s.phase, s.frame); err == nil {
			r["system_info_regions"] = regions
		}
		if s.hd != nil {
			r["system_info_picture_sha256"] = fmt.Sprintf("%x", s.hd.digest)
			r["system_info_captions"] = s.hd.captions
			r["system_info_lines"] = s.hd.lines
		}
		r["system_info_text_index"] = s.textIndex
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

// physicalBattleAuditSources只讀當前job，不表示GPU已呈現，也不推進時鐘或預載圖源。
func physicalBattleAuditSources(scene *nativePhysicalScene) (int, string, []map[string]any) {
	if scene == nil || scene.body == nil || scene.preludeFrame < len(scene.preludeImages) ||
		scene.body.index < 0 || scene.body.index >= len(scene.body.jobs) {
		return -1, "", nil
	}
	job := scene.body.jobs[scene.body.index]
	if job.hdSources == nil {
		return scene.body.index, job.hdPhase, nil
	}
	return scene.body.index, job.hdPhase, modernBattleAuditLayers(job.hdSources)
}

func modernBattleAuditLayers(sources *modernHDBattleSources) []map[string]any {
	if sources == nil {
		return nil
	}
	layers := make([]map[string]any, 0, len(sources.Layers))
	for _, layer := range sources.Layers {
		entry := map[string]any{"archive": layer.Archive, "resource": layer.Resource, "frame": layer.Index,
			"x": layer.Frame.X + layer.DX, "y": layer.Frame.Y + layer.DY,
			"width": layer.Frame.Width, "height": layer.Frame.Height, "fill": layer.Fill, "clip": layer.Clip}
		if layer.PackedShader != 0 {
			colours := make([]int, len(layer.PackedColours))
			for index, value := range layer.PackedColours {
				colours[index] = int(value)
			}
			entry["packed_shader"], entry["packed_colours"] = layer.PackedShader, colours
		}
		layers = append(layers, entry)
	}
	return layers
}
