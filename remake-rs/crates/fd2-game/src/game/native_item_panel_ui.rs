//! Go 來源：`remake/cmd/fd2/native_item_panel_ui.go`（985 行）。
//!
//! 骨架：Go 檔的 24 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    pub(crate) fn plan_native_ai_item_restore(&mut self, plan: &battle::AIPlan) -> Result<NativeAIItemTransaction> {
        if plan.native_action_kind != battle::NATIVE_AI_ACTION_ITEM || plan.native_item_target_indices.is_empty() {
            return Err(game_err!("敵方道具缺少原生目標名單"));
        }
        let state = self.st.as_ref().ok_or_else(|| game_err!("敵方道具缺少戰場狀態"))?;
        let actor_index = plan.u.ok_or_else(|| game_err!("敵方道具缺少使用者"))?;
        let actor = state.units.get(actor_index).ok_or_else(|| game_err!("敵方道具使用者不在名冊"))?;
        let slot = usize::try_from(plan.native_item_slot).map_err(|_| game_err!("敵方道具欄位無效"))?;
        if slot >= 8
            || actor.inventory_slots.len() != 8
            || actor.native_inventory_flags.len() != 8
            || actor.native_inventory_flags[slot] & 0x80 != 0
            || actor.inventory_slots[slot] != plan.native_item_id
        {
            return Err(game_err!("敵方道具欄位與原生記錄不符"));
        }
        let targets = plan.native_item_target_indices.iter().map(|&index| usize::from(index)).collect::<Vec<_>>();
        let mut seen = BTreeSet::new();
        if targets.first().copied() != plan.target
            || targets.iter().any(|&index| index >= state.units.len() || !seen.insert(index))
        {
            return Err(game_err!("敵方道具的原生目標名單無效"));
        }
        if self.native_item_effect_rows.is_empty() {
            self.native_item_effect_rows =
                battle::load_native_item_effect_row_prefix(&asset_path("assets/data/native_item_effect_rows.json"))?;
        }
        let offset = battle::native_item_effect_row_offset(plan.native_item_id)? as usize;
        let row = self
            .native_item_effect_rows
            .get(offset..offset + battle::NATIVE_ITEM_EFFECT_ROW_SIZE as usize)
            .ok_or_else(|| game_err!("缺少原生道具表第 {} 列", plan.native_item_id))?;
        let amount = u16::from_le_bytes([row[0x0e], row[0x0f]]);
        let (route, valid) = battle::native_item_hp_restore_route_for_type(row[0x0d], amount);
        if !valid {
            let (damage_route, damage_valid) = battle::native_item_command_damage_route_for_type(row[0x0d], amount);
            if !damage_valid {
                return Err(game_err!("敵方道具 {} 的原生效果類型 {} 尚無執行入口", plan.native_item_id, row[0x0d]));
            }
            // Go：`native_item_panel_ui.go:497` 的指令傷害分支。共用傷害 helper 會改 HP，先在複本上算，
            // 驗證失敗時不會發布半套多目標名單。
            let before = native_item_runtime_records(&state.units)?;
            let mut clones = targets.iter().map(|&index| state.units[index].clone()).collect::<Vec<_>>();
            let (damage, rng_after) = battle::apply_native_item_command_damage(
                &mut clones,
                damage_route,
                &self.native_command_book,
                &self.native_command_resistances,
                self.native_rng_state,
            )?;
            let mut after = before.clone();
            for (clone, &raw) in clones.iter().zip(&plan.native_item_target_indices) {
                let base = usize::from(raw) * 80;
                after[base + 5] = clone.native_record_byte5;
                after[base + 0x40..base + 0x42].copy_from_slice(&(clone.hp as i16).to_le_bytes());
            }
            return Ok(NativeAIItemTransaction {
                before,
                after,
                targets,
                rng_before: self.native_rng_state,
                rng_after,
                damage,
                damage_route: Some(damage_route),
                ..Default::default()
            });
        }
        let before = native_item_runtime_records(&state.units)?;
        let mut after = before.clone();
        let rng_before = self.native_rng_state;
        let restore = battle::apply_native_item_hp_restore(
            &mut after,
            &plan.native_item_target_indices,
            route,
            rng_before,
            actor_index as i64,
            plan.native_item_slot,
        )?;
        Ok(NativeAIItemTransaction {
            before,
            after,
            targets,
            rng_before,
            rng_after: restore.rng_state,
            restore: Some(restore),
            ..Default::default()
        })
    }

    /// Go：`native_item_panel_ui.go:156`，`func (g *Game) clearNativeItemPanel()`
    pub(crate) fn clear_native_item_panel(&mut self) {
        self.native_item_panel_text = None;
        self.native_item_panel = None;
        self.native_item_panel_base = Vec::new();
        self.native_item_panel_record = Vec::new();
        self.native_item_panel_assets = None;
        self.native_item_effect_rows = Vec::new();
        self.item_anim_step = 0;
        self.item_closing = false;
    }
}

/// applyNativeAITargetItem consumes the complete 0x1567e winner list at the
/// 0x15055 boundary. It deliberately bypasses the player-owned cursor/modal:
/// the original AI owner rebuilds the whole list from its saved destination
/// and passes that list to 0x20c6f. All mutable records stay detached until
/// item, target and effect provenance have passed validation.
///
/// Go：`native_item_panel_ui.go:433`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeAIItemTransaction {
    pub(crate) before: Vec<u8>,
    pub(crate) after: Vec<u8>,
    pub(crate) targets: Vec<UnitIdx>,
    pub(crate) restore: Option<battle::NativeRawRestoreBatch>,
    pub(crate) damage: Vec<battle::NativeCommandDamage>,
    pub(crate) damage_route: Option<battle::NativeItemCommandDamageRoute>,
    pub(crate) rng_before: u16,
    pub(crate) rng_after: u16,
}

/// Go：`native_item_panel_ui.go:386`，`func nativeItemRuntimeRecords(units []*battle.Unit) ([]byte, error)`（`r8`，隨
/// `commitNativeAIItemTransaction` 補移植）。
pub(crate) fn native_item_runtime_records(units: &[battle::Unit]) -> Result<Vec<u8>> {
    let mut records = Vec::with_capacity(units.len() * 80);
    for (index, unit) in units.iter().enumerate() {
        let mut record = battle::native_item_panel_record_for_unit(Some(unit))
            .map_err(|e| GameError::from(e).wrap(format!("runtime unit {index} lacks native item record: ")))?;
        if unit.x < 0 || unit.x > 0xff || unit.y < 0 || unit.y > 0xff || !unit.has_native_record_byte5 {
            return Err(game_err!("runtime unit {index} lacks native coordinate/activity provenance"));
        }
        (record[0], record[1], record[5]) = (unit.x as u8, unit.y as u8, unit.native_record_byte5);
        records.extend_from_slice(&record);
    }
    Ok(records)
}

/// Go：`native_item_panel_ui.go:403`，`func syncNativeItemRuntimeRecord(unit *battle.Unit, record []byte)`（`r8`，隨
/// `cancelNativeAIItemPresentation` 補移植）。
pub(crate) fn sync_native_item_runtime_record(unit: &mut battle::Unit, record: &[u8]) {
    let word = |offset: usize| i64::from(i16::from_le_bytes([record[offset], record[offset + 1]]));
    unit.native_record_byte5 = record[5];
    unit.has_native_record_byte5 = true;
    unit.hp = word(0x40);
    unit.mp = word(0x44);
    let dir = unit.dir;
    unit.set_map_placement(i64::from(record[0]), i64::from(record[1]), dir);
    unit.ap = word(0x48);
    unit.dp = word(0x4a);
    unit.hit = word(0x4c);
    unit.ev = word(0x4e);
    unit.native_transient.copy_from_slice(&record[0x22..0x28]);
    unit.inventory_slots = vec![0; 8].into();
    unit.native_inventory_flags = vec![0; 8].into();
    unit.inventory.clear();
    unit.equipped.clear();
    for slot in 0..8 {
        let (flag, item) = (i64::from(record[0x0a + slot * 2]), i64::from(record[0x0b + slot * 2]));
        (unit.native_inventory_flags[slot], unit.inventory_slots[slot]) = (flag, item);
        if flag & 0x80 == 0 {
            unit.inventory.push(item);
            unit.equipped.push(flag & 0x40 != 0);
        }
    }
}

impl Game {
    /// Go：`native_item_panel_ui.go:528`，`func (g *Game) commitNativeAIItemTransaction(tx *nativeAIItemTransaction) error`
    /// （`r8`，隨 `stepNativeAIItemPresentation` 補移植）。
    pub(crate) fn commit_native_ai_item_transaction(&mut self, tx: Option<&NativeAIItemTransaction>) -> Result<()> {
        let (Some(st), Some(tx)) = (self.st.as_mut(), tx) else {
            return Err(game_err!("native AI item transaction is unavailable"));
        };
        if tx.before.is_empty() || tx.after.len() != tx.before.len() {
            return Err(game_err!("native AI item transaction is unavailable"));
        }
        let current = native_item_runtime_records(&st.units)?;
        if current != tx.before || self.native_rng_state != tx.rng_before {
            return Err(game_err!("native AI item source state changed before publication"));
        }
        for (index, unit) in st.units.iter_mut().enumerate() {
            sync_native_item_runtime_record(unit, &tx.after[index * 80..(index + 1) * 80]);
        }
        self.native_rng_state = tx.rng_after;
        Ok(())
    }
}

/// 原版八格裡旗標 bit7 清除（有物品）的原始格索引。
///
/// Go：`native_item_panel_ui.go:15`，`func nativeItemRawSlots(unit *battle.Unit) []int`
pub(crate) fn native_item_raw_slots(unit: &battle::Unit) -> Vec<i64> {
    if unit.inventory_slots.len() != 8 || unit.native_inventory_flags.len() != 8 {
        return Vec::new();
    }
    (0..8).filter(|&slot| unit.native_inventory_flags[slot as usize] & 0x80 == 0).collect()
}

impl Game {
    fn native_item_row(&self, item_id: i64) -> Result<Vec<u8>> {
        let offset = battle::native_item_effect_row_offset(item_id).ok().filter(|&o| {
            o >= 0 && o + battle::NATIVE_ITEM_EFFECT_ROW_SIZE <= self.native_item_effect_rows.len() as i64
        });
        let Some(offset) = offset else {
            return Err(game_err!("native item row {item_id} is unavailable"));
        };
        Ok(self.native_item_effect_rows[offset as usize..(offset + battle::NATIVE_ITEM_EFFECT_ROW_SIZE) as usize]
            .to_vec())
    }

    /// Go：`native_item_panel_ui.go:28`，`func (g *Game) prepareNativeItemPanel(unit *battle.Unit) bool`
    pub(crate) fn prepare_native_item_panel(&mut self, unit: UnitIdx) -> bool {
        self.prepare_native_item_panel_mode(unit, false)
    }

    /// 原版索引色面板：底圖、資料與效果表都備齊才發布；HD 文字快照（`prepareNativeItemPanelText`）屬 R6。
    ///
    /// Go：`native_item_panel_ui.go:32`，`func (g *Game) prepareNativeItemPanelMode(unit *battle.Unit, allowEmpty bool) bool`
    pub(crate) fn prepare_native_item_panel_mode(&mut self, unit: UnitIdx, allow_empty: bool) -> bool {
        self.clear_native_item_panel();
        if self.modern_hd_interface_enabled() {
            todo!("R6: native_item_panel_ui.go:57 preflightModernHDItemPanel／prepareNativeItemPanelText");
        }
        let (asset_root, portrait_root) = (separated_asset_path(""), separated_asset_path("portraits"));
        if asset_root.is_empty() || self.native_ui_palette.len() < 256 {
            return false;
        }
        let Some(st) = self.st.as_ref() else {
            return false;
        };
        let Ok(record) = battle::native_item_panel_record_for_unit(Some(&st.units[unit])) else {
            return false;
        };
        let mut pixels = vec![0u8; 320 * 200];
        if battle::render_native_item_panel_resources(&asset_root, &portrait_root, &record, &mut pixels).is_err() {
            return false;
        }
        let Ok(assets) = battle::load_native_item_panel_data_assets(&asset_root) else {
            return false;
        };
        let Ok(rows) =
            battle::load_native_item_effect_row_prefix(&asset_path("assets/data/native_item_effect_rows.json"))
        else {
            return false;
        };
        self.native_item_panel_base = pixels;
        self.native_item_panel_record = record;
        self.native_item_panel_assets = Some(assets);
        self.native_item_effect_rows = rows;
        self.refresh_native_item_panel_mode(unit, allow_empty)
    }

    /// Go：`native_item_panel_ui.go:70`，`func (g *Game) refreshNativeItemPanel(unit *battle.Unit) bool`
    pub(crate) fn refresh_native_item_panel(&mut self, unit: UnitIdx) -> bool {
        self.refresh_native_item_panel_mode(unit, false)
    }

    /// Go：`native_item_panel_ui.go:74`，`func (g *Game) refreshNativeItemPanelMode(unit *battle.Unit, allowEmpty bool) bool`
    pub(crate) fn refresh_native_item_panel_mode(&mut self, unit: UnitIdx, allow_empty: bool) -> bool {
        if self.native_item_panel_base.len() != 320 * 200
            || self.native_item_panel_record.len() != 80
            || self.native_item_effect_rows.is_empty()
        {
            return false;
        }
        let Some(assets) = self.native_item_panel_assets.as_ref() else {
            return false;
        };
        let raw_slots = native_item_raw_slots(&self.st.as_ref().unwrap().units[unit]);
        let selected = if raw_slots.is_empty() {
            if !allow_empty {
                return false;
            }
            -1
        } else {
            self.item_sel = self.item_sel.clamp(0, raw_slots.len() as i64 - 1);
            raw_slots[self.item_sel as usize]
        };
        let mut pixels = self.native_item_panel_base.clone();
        if battle::render_native_item_panel_rows(
            assets,
            &self.native_item_panel_record,
            selected,
            &self.native_item_effect_rows,
            &mut pixels,
        )
        .is_err()
        {
            return false;
        }
        self.set_native_item_panel_pixels(&pixels)
    }

    /// Go：`native_item_panel_ui.go:109`，`func (g *Game) setNativeItemPanelPixels(pixels []byte) bool`
    pub(crate) fn set_native_item_panel_pixels(&mut self, pixels: &[u8]) -> bool {
        if pixels.len() != 320 * 200 || self.native_ui_palette.len() < 256 {
            return false;
        }
        let mut palette = self.native_ui_palette.clone();
        palette[0] = ColorRgba { r: 0, g: 0, b: 0, a: 0xff };
        let frame = indexed_frame(320, 200, pixels, palette);
        self.native_item_panel = Some(Arc::new(Picture::from_indexed(&frame)));
        true
    }

    /// 回傳 true 表示輸入仍要擋住（開收動畫進行中）。
    ///
    /// Go：`native_item_panel_ui.go:168`，`func (g *Game) stepNativeItemPanelAnimation() bool`
    pub(crate) fn step_native_item_panel_animation(&mut self) -> bool {
        if self.native_item_panel.is_none() {
            return false;
        }
        if self.item_closing {
            if self.item_anim_step < 11 {
                self.item_anim_step += 1;
                return true;
            }
            if self.native_death_reward_ui.is_some() {
                todo!("R8: native_item_panel_ui.go:180 finishNativeDeathRewardItemClose");
            }
            self.item_open = false;
            self.begin_action_overlay_open(self.ring_sel);
            self.clear_native_item_panel();
            return true;
        }
        if self.item_anim_step < 11 {
            self.item_anim_step += 1;
            return true;
        }
        false
    }

    /// Go：`native_item_panel_ui.go:195`，`func (g *Game) beginNativeItemPanelClose()`
    pub(crate) fn begin_native_item_panel_close(&mut self) {
        if self.native_item_panel.is_none() {
            self.item_open = false;
            self.begin_action_overlay_open(self.ring_sel);
            return;
        }
        (self.item_closing, self.item_anim_step) = (true, 0);
    }

    /// 只執行完全閉合、不用亂數的 type 8／9／10（基礎值）與 17／18／19（上限）即時物品交易。
    ///
    /// Go：`native_item_panel_ui.go:209`，`func (g *Game) applyNativeImmediateItem(rawSlot, itemID int) (bool, error)`
    pub(crate) fn apply_native_immediate_item(&mut self, raw_slot: i64, item_id: i64) -> Result<bool> {
        let (Some(_), Some(actor)) = (self.st.as_ref(), self.sel) else {
            return Err(GameError::new("native item transaction context is unavailable"));
        };
        let row = self.native_item_row(item_id)?;
        let (word_route, word_supported) = battle::native_item_word_delta_route_for_type(i64::from(row[0x0d]));
        let delta = u16::from_le_bytes([row[0x0e], row[0x0f]]);
        let (capacity_route, capacity_supported) = battle::native_item_capacity_step_route_for_type(row[0x0d], delta);
        if !word_supported && !capacity_supported {
            return Ok(false);
        }
        let plan = battle::native_item_target_plan_from_row(&row)?;
        let st = self.st.as_ref().unwrap();
        let flags = st.native_command_base_flags()?;
        let targets =
            battle::native_item_effect_targets(st.w, st.h, Some(actor), Some(actor), plan, &flags, &st.units)?;
        if targets != [actor] {
            return Err(GameError::new("native immediate item target list is not actor-only"));
        }
        if word_supported {
            if self.shop_item_stats.is_empty() {
                return Err(GameError::new("native equipment recomputation table is unavailable"));
            }
            let u = &st.units[actor];
            for (index, &equipped) in u.equipped.iter().enumerate() {
                if equipped && index < u.inventory.len() && !self.shop_item_stats.contains_key(&u.inventory[index]) {
                    return Err(game_err!(
                        "native equipped item {} is absent from recomputation table",
                        u.inventory[index]
                    ));
                }
            }
            let units = &mut self.st.as_mut().unwrap().units;
            battle::apply_native_item_base_stat_delta_to_unit(
                units,
                Some(actor),
                Some(actor),
                word_route,
                delta,
                raw_slot,
            )?;
            campaign::recompute_equipment(Some(&mut units[actor]), &self.shop_item_stats);
        } else {
            let units = &mut self.st.as_mut().unwrap().units;
            battle::apply_native_item_capacity_to_unit(units, Some(actor), Some(actor), capacity_route, raw_slot)?;
        }
        // 0x1bbdc calls 0x13512 immediately after successful 0x20c6f.
        let u = &mut self.st.as_mut().unwrap().units[actor];
        (u.native_record_byte5, u.has_native_record_byte5) = (u.native_record_byte5 | 0x80, true);
        self.finish_successful_unit_action(
            Some(actor),
            Some(Box::new(|g: &mut Game| {
                (g.item_open, g.ring) = (false, false);
                g.clear_native_item_panel();
                (g.sel, g.reach, g.moved) = (None, Default::default(), false);
            })),
        );
        Ok(true)
    }

    /// 只對有閉合原始交易的效果族進入第一段 0x14818 選擇器；狀態改寫延到具體目標通過兩段規劃。
    ///
    /// Go：`native_item_panel_ui.go:281`，`func (g *Game) beginNativeTargetItem(rawSlot, itemID int) (bool, error)`
    pub(crate) fn begin_native_target_item(&mut self, raw_slot: i64, item_id: i64) -> Result<bool> {
        let row = self.native_item_row(item_id)?;
        let (item_type, amount) = (row[0x0d], u16::from_le_bytes([row[0x0e], row[0x0f]]));
        let hp = battle::native_item_hp_restore_route_for_type(item_type, amount).1;
        let supported = hp
            || battle::native_item_mp_restore_route_for_type(item_type, amount).1
            || battle::native_item_marker_clear_restore_route_for_type(item_type).1
            || battle::native_item_hitev_step_route_for_type(item_type).1
            || battle::native_item_apdp_step_route_for_type(item_type).1
            || battle::native_item_marker_application_route_for_type(item_type).1
            || battle::native_item_command_damage_route_for_type(item_type, amount).1
            || battle::native_item_relocation_route_for_type(item_type, amount).1;
        if !supported {
            return Ok(false);
        }
        let plan = battle::native_item_target_plan_from_row(&row)?;
        let actor = self.sel.ok_or_else(|| GameError::new("native item target lacks selection"))?;
        let st = self.st.as_ref().unwrap();
        let flags = st.native_command_base_flags()?;
        let origin = battle::Cell { x: st.units[actor].x, y: st.units[actor].y };
        let field = battle::native_command_target_field_bytes(
            st.w,
            st.h,
            origin,
            plan.selection_mode,
            plan.selection_inner_mark,
            &flags,
        )
        .ok()
        .filter(|field| field.len() == st.native_tile_blit_modes.len())
        .ok_or_else(|| GameError::new("native item target field is unavailable"))?;
        // 第一戰一般玩家證據：四名我方都滿 HP 時 type 5 草藥確認後仍留在物品面板，不發布目標框。
        // 只對 type 5 套用這個窄 gate；type 13 與其他物品沒有同等動態證據。
        if hp && item_type == 5 {
            let targets = battle::native_attack_candidates(
                st.w,
                st.h,
                origin,
                plan.selection_mode,
                plan.selection_inner_mark,
                plan.target_code,
                &flags,
                &st.units,
            )?;
            if !targets.iter().any(|&t| st.units[t].hp < st.units[t].max_hp) {
                return Ok(false);
            }
        }
        let selector = u8::try_from(plan.effect_mode)
            .map(battle::native_map_overlay_selector_from_record_byte)
            .map_err(|_| GameError::new("native item overlay selector is invalid"))?;
        let st = self.st.as_mut().unwrap();
        if !st.materialize_native_map_range_mode(selector) {
            return Err(GameError::new("native item overlay selector is invalid"));
        }
        st.native_tile_blit_modes.copy_from_slice(&field);
        (self.native_item_targeting, self.native_item_target_id, self.native_item_target_raw_slot) =
            (true, item_id, raw_slot);
        self.item_open = false;
        let rows = std::mem::take(&mut self.native_item_effect_rows);
        self.clear_native_item_panel();
        self.native_item_effect_rows = rows;
        (self.cur_x, self.cur_y) = (origin.x, origin.y);
        self.reach = Default::default();
        Ok(true)
    }

    /// 第二段：依確認目標算出最終目標名單，套用 HP／MP／旗標／能力值的原始記錄交易。
    /// 指令傷害物品與傳送物品的演出另期，遇到時失敗即關閉。
    ///
    /// Go：`native_item_panel_ui.go:556`，`func (g *Game) applyNativeTargetItem(confirmed *battle.Unit) (bool, error)`
    pub(crate) fn apply_native_target_item(&mut self, confirmed: UnitIdx) -> Result<bool> {
        let Some(actor) = self.sel.filter(|_| self.st.is_some() && self.native_item_targeting) else {
            return Ok(false);
        };
        let row = self.native_item_row(self.native_item_target_id)?;
        let plan = battle::native_item_target_plan_from_row(&row)?;
        let st = self.st.as_ref().unwrap();
        let flags = st.native_command_base_flags()?;
        let Ok(targets) =
            battle::native_item_effect_targets(st.w, st.h, Some(actor), Some(confirmed), plan, &flags, &st.units)
        else {
            return Ok(false);
        };
        // 0x1bd5e..0x1bd6c 在第一個選擇器回來後立刻重設 byte+3，並把 overlay selector 回 1。
        self.reset_native_target_field();
        self.st.as_mut().unwrap().materialize_native_map_range_mode(1);
        let target_indices = targets.iter().map(|&t| t as u8).collect::<Vec<_>>();
        let (item_type, amount) = (row[0x0d], u16::from_le_bytes([row[0x0e], row[0x0f]]));
        if battle::native_item_command_damage_route_for_type(item_type, amount).1 {
            todo!("R8: native_item_panel_ui.go:603 startNativeTargetItemPresentation（指令傷害物品）");
        }
        let units = &self.st.as_ref().unwrap().units;
        let mut records = native_item_runtime_records(units)?;
        let (source, slot, rng) = (actor as i64, self.native_item_target_raw_slot, self.native_rng_state);
        let next_rng;
        if let (route, true) = battle::native_item_hp_restore_route_for_type(item_type, amount) {
            next_rng = battle::apply_native_item_hp_restore(&mut records, &target_indices, route, rng, source, slot)?
                .rng_state;
        } else if let (route, true) = battle::native_item_mp_restore_route_for_type(item_type, amount) {
            next_rng = battle::apply_native_item_mp_restore(&mut records, &target_indices, route, rng, source, slot)?
                .rng_state;
        } else if let (route, true) = battle::native_item_marker_clear_restore_route_for_type(item_type) {
            next_rng = battle::apply_native_item_marker_clear_restore(
                &mut records,
                &target_indices,
                route,
                rng,
                source,
                slot,
            )?
            .1;
        } else if let (route, true) = battle::native_item_hitev_step_route_for_type(item_type) {
            next_rng = battle::apply_native_item_hitev_step(&mut records, &target_indices, route, rng)?.1;
        } else if let (route, true) = battle::native_item_apdp_step_route_for_type(item_type) {
            next_rng = battle::apply_native_item_apdp_step(&mut records, &target_indices, route, rng)?.1;
        } else if let (route, true) = battle::native_item_marker_application_route_for_type(item_type) {
            next_rng = battle::apply_native_item_marker_application(&mut records, &target_indices, route, rng)?.1;
        } else if battle::native_item_relocation_route_for_type(item_type, amount).1 {
            todo!("R8: native_item_panel_ui.go:686 傳送物品目的地（applyNativeRelocationDestination）");
        } else {
            return Ok(false);
        }
        let units = &mut self.st.as_mut().unwrap().units;
        for (index, unit) in units.iter_mut().enumerate() {
            sync_native_item_runtime_record(unit, &records[index * 80..(index + 1) * 80]);
        }
        self.native_rng_state = next_rng;
        let u = &mut units[actor];
        (u.native_record_byte5, u.has_native_record_byte5) = (u.native_record_byte5 | 0x80, true);
        self.finish_successful_unit_action(
            Some(actor),
            Some(Box::new(|g: &mut Game| {
                g.native_item_targeting = false;
                g.native_item_effect_rows = Vec::new();
                (g.sel, g.reach, g.moved) = (None, Default::default(), false);
            })),
        );
        Ok(true)
    }

    /// 原版十二格開收動畫：每格依 `native_item_panel_frame_for` 畫三個區塊。
    ///
    /// Go：`native_item_panel_ui.go:935`，`func (g *Game) drawNativeItemPanel(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_item_panel(&mut self, screen: &mut Picture) -> bool {
        let Some(panel) = self.native_item_panel.clone() else {
            return false;
        };
        let frame = if self.item_closing { self.item_anim_step } else { 11 - self.item_anim_step }.clamp(0, 11);
        let Ok(pass) = battle::native_item_panel_frame_for(frame) else {
            return false;
        };
        for region in [pass.left, pass.upper, pass.bottom] {
            if !region.enabled || region.width <= 0 || region.height <= 0 {
                continue;
            }
            let source = panel.sub_image(rect(
                region.source_x,
                region.source_y,
                region.source_x + region.width,
                region.source_y + region.height,
            ));
            let mut options = DrawImageOptions::default();
            options.geo_m.scale(2.0, 2.0);
            options.geo_m.translate((region.dest_x * 2) as f64, (region.dest_y * 2) as f64);
            screen.draw_sub_image(source, &options);
        }
        true
    }
}

impl Game {
    /// 原版八格選擇器邊界（0x1b932）：有原始八格時以旗標 bit7 判斷有無物品，舊版可編輯 JSON 只保留顯示用的後備。
    ///
    /// Go：`main.go:10406`，`func (g *Game) nativeItemMenuSlot(u *battle.Unit, slot int) (occupied bool, itemID int)`
    pub(crate) fn native_item_menu_slot(&self, unit: UnitIdx, slot: i64) -> (bool, i64) {
        let Some(u) = self.st.as_ref().map(|st| &st.units[unit]) else {
            return (false, 0);
        };
        if !(0..8).contains(&slot) {
            return (false, 0);
        }
        if u.inventory_slots.len() == 8 && u.native_inventory_flags.len() == 8 {
            return (u.native_inventory_flags[slot as usize] & 0x80 == 0, u.inventory_slots[slot as usize]);
        }
        match u.inventory.get(slot as usize) {
            Some(&item) => (true, item),
            None => (false, 0),
        }
    }

    /// 物品面板開啟時的輸入：動畫中擋輸入、Esc 收合、方向鍵走原版選擇器、Enter 先試即時物品再試目標物品。
    ///
    /// Go：`main.go:5797..5917` 的 `itemOpen` 分支。
    pub(crate) fn native_item_panel_input(&mut self, enter: bool, esc: bool) -> bool {
        if self.step_native_item_panel_animation() {
            return true;
        }
        if self.native_death_reward_ui.is_some() {
            todo!("R8: main.go:5806 handleNativeDeathRewardItemInput（死亡獎勵物品）");
        }
        if esc {
            self.begin_native_item_panel_close();
            return true;
        }
        let Some(actor) = self.sel else {
            return false;
        };
        if self.native_item_panel.is_none() {
            todo!("R8: main.go:5896 缺原版素材時的後備物品清單");
        }
        let raw_slots = native_item_raw_slots(&self.st.as_ref().unwrap().units[actor]);
        if raw_slots.is_empty() {
            self.item_open = false;
            self.begin_action_overlay_open(self.ring_sel);
            self.clear_native_item_panel();
            return true;
        }
        let input = self.tick_input.clone();
        let key = [(Key::ArrowUp, 72), (Key::ArrowDown, 80), (Key::ArrowLeft, 75), (Key::ArrowRight, 77)]
            .into_iter()
            .find(|&(k, _)| input.is_just_pressed(k))
            .map_or(0, |(_, code)| code);
        if key != 0
            && let Ok((selected, _)) =
                battle::advance_native_item_selector(self.item_sel, raw_slots.len() as i64, key, true, 1)
            && selected != self.item_sel
        {
            self.item_sel = selected;
            self.refresh_native_item_panel(actor);
        }
        if !enter {
            return true;
        }
        let raw_slot = raw_slots[self.item_sel as usize];
        let (occupied, item_id) = self.native_item_menu_slot(actor, raw_slot);
        let message = |g: &mut Game, key: &str| {
            if let (message, true) = g.locale_message(key, &[localization::FormatArg::Int(item_id)]) {
                g.msg = message;
            }
        };
        if !occupied {
            message(self, "battle.item.empty_slot");
            return true;
        }
        let row = item_id * battle::NATIVE_ITEM_EFFECT_ROW_SIZE;
        if row < 0 || row + battle::NATIVE_ITEM_EFFECT_ROW_SIZE > self.native_item_effect_rows.len() as i64 {
            message(self, "battle.item.data_incomplete");
            return true;
        }
        let effect = self.native_item_effect_rows[(row + 0x0d) as usize];
        let confirmed = battle::advance_native_item_selector(self.item_sel, raw_slots.len() as i64, 28, true, effect)
            .is_ok_and(|(_, result)| result == battle::NATIVE_ITEM_SELECTOR_CONFIRM);
        if !confirmed {
            message(self, "battle.item.unusable");
            return true;
        }
        match self.apply_native_immediate_item(raw_slot, item_id) {
            Err(err) => {
                self.msg = format!("物品 {item_id:02X}h：{}", err.message());
                return true;
            }
            Ok(true) => {
                message(self, "battle.item.effect_complete");
                return true;
            }
            Ok(false) => {}
        }
        match self.begin_native_target_item(raw_slot, item_id) {
            Err(err) => {
                self.msg = format!("物品 {item_id:02X}h：{}", err.message());
                return true;
            }
            Ok(true) => {
                message(self, "battle.item.choose_target");
                return true;
            }
            Ok(false) => {}
        }
        message(self, "battle.item.effect_unverified");
        true
    }

    /// 目標選擇中按 Esc：重設目標標記、overlay selector 回 1，回到保留的物品面板（不重新準備，避免清掉效果表）。
    ///
    /// Go：`main.go:6222`，`func (g *Game) cancelNativeItemTargetModal() bool`
    pub(crate) fn cancel_native_item_target_modal(&mut self) -> bool {
        if self.st.is_none() || (!self.native_item_targeting && !self.native_item_relocating) {
            return false;
        }
        self.reset_native_target_field();
        self.st.as_mut().unwrap().materialize_native_map_range_mode(1);
        (self.native_item_targeting, self.native_item_relocating) = (false, false);
        self.native_movement_cost_rows = Default::default();
        if self.native_command23_relocating {
            self.native_command23_relocating = false;
            self.native_command_open = true;
            return true;
        }
        self.item_open = true;
        if let Some(actor) = self.sel {
            self.refresh_native_item_panel(actor);
        }
        true
    }
}
