//! Go 來源：`remake/internal/battle/native_ai_runtime.go`（291 行）。

use super::*;

impl State {
    /// nextNativeAIPhysicalPlan 是原始 0x14237 候選橋的第一個正式消費端。
    /// 範圍刻意縮窄，只接受 runtime mode 2，不猜測 caller 擁有的 0x14ef0 前置選擇。
    /// 因此，移動／物品／地形來源不完整時回傳錯誤，不退回正規化目標選擇。
    ///
    /// Go：`native_ai_runtime.go:11`，`func (s *State) nextNativeAIPhysicalPlan(u *Unit) (*AIPlan, bool, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn next_native_ai_physical_plan(&self, u: Option<UnitIdx>) -> Result<(Option<AIPlan>, bool)> {
        let Some(ui) = u.filter(|&i| self.units.get(i).is_some_and(|unit| unit.has_native_record_byte34)) else {
            return Ok((None, false));
        };
        let unit = &self.units[ui];
        let mode = (unit.native_record_byte34 & 0x0f) as i64;
        if mode != 2 {
            return Ok((None, false));
        }
        if !unit.has_native_record_byte6 || self.units.is_empty() {
            return Err(battle_err!("native AI mode 2 lacks selector/runtime roster provenance"));
        }
        // Go 以指標相等在 `Units` 裡找 actor；Rust 的 `u` 已經是索引。
        let actor = ui as i64;
        if self.native_movement_cost_rows.len() as i64 != NATIVE_MOVEMENT_COST_ROW_COUNT {
            return Err(battle_err!("native AI mode 2 movement table is unavailable"));
        }
        if self.native_terrain_move_codes.len() as i64 != self.w.wrapping_mul(self.h) {
            return Err(battle_err!("native AI mode 2 terrain move-code provenance is unavailable"));
        }
        if self.native_composition_event_bytes.len() as i64 != self.w.wrapping_mul(self.h) {
            return Err(battle_err!("native AI mode 2 composition provenance is unavailable"));
        }

        let records = native_ai_scoring_records(&self.units)?;
        let rs = NATIVE_RECORD_SIZE as usize;
        let actor_record = &records[ui * rs..(ui + 1) * rs];
        let (item, found) = resolve_native_ai_physical_item_source(actor_record, &self.native_future_item_rows)?;
        if !found {
            // 0x1428B 0x1B83D(actor,0) 回 -1 → 0x14296 je 0x145C3 → xor eax,eax：0x14237 回傳 0，
            // 與沒有候選同一條收尾（0x13C06→0x13FD4）。第十章友軍記錄 50 身上沒有武器。
            let (plan, handled) = self.native_ai_mode2_idle_recovery_plan(u, &records, actor)?;
            return Ok((Some(plan), handled));
        }
        let selector = unit.native_record_byte6 as i64;
        if selector != 0 && selector != 1 {
            return Err(battle_err!("native AI mode 2 selector={selector} is outside the verified 0/1 caller set"));
        }
        let cost_row = native_ai_cost_row_for_record(actor_record, &self.native_movement_cost_rows)?;
        let geometry = NativeAIPhysicalAttackGeometry {
            target_mode: item.raw_geometry_byte0_c as i64,
            target_inner_mark: item.raw_geometry_byte0_b as i64,
            // 0x14237 sets this local to 1 for selector zero and 0 otherwise.
            target_code: native_ai_physical_target_code(selector),
        };
        let base_flags = native_composition_base_flags(self.w, self.h, &self.native_composition_event_bytes)?;
        let runtime_flags = native_command_runtime_flags(
            self.w,
            self.h,
            &self.native_composition_event_bytes,
            &self.units,
            selector as u8,
        )?;
        let initial_budget = actor_record[0x3b] as i64;
        let mut resolve = |raw: NativeAIPhysicalAttackRawCandidate| {
            self.native_ai_physical_score_input(&raw, &self.native_future_item_rows)
        };
        let candidates = build_native_ai_physical_attack_candidates(
            self.w,
            self.h,
            &records,
            self.units.len() as i64,
            actor,
            selector,
            initial_budget,
            // 0x14237 在每次 0x14818 目標掃描前以 0x4DBFC 重設 live grid；因此
            // 目標幾何看到的是不可變的 low5 base，而不是前段移動佔用寫入值。
            geometry,
            &base_flags,
            &base_flags,
            &self.native_terrain_move_codes,
            &cost_row,
            Some(&mut resolve),
        )?;
        let (selection, ok) = select_native_physical_attack_candidate(&candidates)?;
        if !ok {
            let (plan, handled) = self.native_ai_mode2_idle_recovery_plan(u, &records, actor)?;
            return Ok((Some(plan), handled));
        }
        let selected = selection.candidate;
        let origin = Cell { x: actor_record[0] as i64, y: actor_record[1] as i64 };
        let (path_directions, reachable) = native_path_directions(
            self.w,
            self.h,
            origin,
            Cell { x: selected.destination_x, y: selected.destination_y },
            initial_budget,
            0,
            &runtime_flags,
            &self.native_terrain_move_codes,
            &cost_row,
        )?;
        if !reachable {
            return Err(battle_err!("native AI mode 2 selected destination is not path-reachable"));
        }
        let path = native_ai_direction_path(origin, &path_directions)?;
        if selected.target_index < 0 || selected.target_index >= self.units.len() as i64 {
            return Err(battle_err!("native AI mode 2 selected target index is invalid"));
        }
        Ok((
            Some(AIPlan {
                u,
                path,
                target: Some(selected.target_index as UnitIdx),
                spell_id: -1,
                native_mode2_physical: true,
                native_scored_commands: self.native_ai_plan_scored_commands(u),
                ..Default::default()
            }),
            true,
        ))
    }
}

/// Go：`native_ai_runtime.go:132`，`func nativeAIPhysicalTargetCode(selector int) int`
pub(crate) fn native_ai_physical_target_code(selector: i64) -> i64 {
    if selector == 0 { 1 } else { 0 }
}

/// Go：`native_ai_runtime.go:139`，`func nativeAICostRowForRecord(record []byte, rows [][]byte) ([]byte, error)`
pub(crate) fn native_ai_cost_row_for_record(record: &[u8], rows: &[Vec<u8>]) -> Result<Vec<u8>> {
    if (record.len() as i64) < NATIVE_RECORD_SIZE || rows.len() as i64 != NATIVE_MOVEMENT_COST_ROW_COUNT {
        return Err(battle_err!("native AI movement row provenance is malformed"));
    }
    let mut selector = record[0x20] as usize;
    if native_aiis_terrain_special_record(record) {
        selector = 19;
    }
    if record[8] == 0x1c {
        selector = 1;
    }
    if selector >= rows.len() || rows[selector].len() as i64 != NATIVE_MOVEMENT_COST_ROW_SIZE {
        return Err(battle_err!("native AI movement selector={selector} is unavailable"));
    }
    Ok(rows[selector].clone())
}

/// nativeAIIsTerrainSpecialRecord 重現 `0x1F183`：`+7 == 0x1C` 時**回 0**
/// （`0x1F1A7 cmp edx,0x1C` / `0x1F1AA je 0x1F1C9` → `xor eax,eax`），只有
/// `+0x20 == 0x13` 或 `+0x1F` 是 4／5 才回 1。AI 兩個呼叫端（`0x1417B`、
/// `0x14BDB`）拿它的回傳值把成本列換成 19；`+7 == 0x1C` 在那兩處是另一條規則
/// （`+8 == 0x1C` → 列 1），玩家側 `0x188F9` 則是列 16。
///
/// 早期把 `+7 == 0x1C` 也算成 true，等於讓那些單位在 AI 回合拿到「每種地形都
/// 只要 1」的列，走進自己走不出來的山地。
///
/// Go：`native_ai_runtime.go:164`，`func nativeAIIsTerrainSpecialRecord(record []byte) bool`
pub(crate) fn native_aiis_terrain_special_record(record: &[u8]) -> bool {
    record.len() as i64 >= NATIVE_RECORD_SIZE
        && record[7] != 0x1c
        && (record[0x20] == 0x13 || record[0x1f] == 4 || record[0x1f] == 5)
}

/// 0x15B77的指令10～12以目標runtime索引呼叫1F183。
/// 評分器先驗證candidate bounds，再消費這份同狀態raw record快照。
///
/// Go：`native_ai_runtime.go:171`，`func nativeAI1F183TargetGate(records []byte) func(int) bool`
pub(crate) fn native_ai1_f183_target_gate(records: &[u8]) -> Box<dyn FnMut(i64) -> bool + '_> {
    let rs = NATIVE_RECORD_SIZE as usize;
    Box::new(move |index: i64| {
        let index = index as usize;
        native_aiis_terrain_special_record(&records[index * rs..(index + 1) * rs])
    })
}

impl State {
    /// Go：`native_ai_runtime.go:177`，`func (s *State) nativeAIPhysicalScoreInput( raw NativeAIPhysicalAttackRawCandidate, itemRows []byte, ) (NativePhysicalAttackScoreInput, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ai_physical_score_input(
        &self,
        raw: &NativeAIPhysicalAttackRawCandidate,
        item_rows: &[u8],
    ) -> Result<NativePhysicalAttackScoreInput> {
        if raw.actor_record.len() as i64 != NATIVE_RECORD_SIZE || raw.target_record.len() as i64 != NATIVE_RECORD_SIZE {
            return Err(battle_err!("native AI physical score record snapshot is malformed"));
        }
        let word = |record: &[u8], offset: usize| u16::from_le_bytes([record[offset], record[offset + 1]]) as i64;
        let (actor48, actor4a) = self.native_ai_terrain_adjusted_words(
            &raw.actor_record,
            raw.destination,
            word(&raw.actor_record, 0x48),
            word(&raw.actor_record, 0x4a),
        )?;
        // 0x14545..0x1454E 讀 target +0/+1；只有 actor 使用候選目的地。
        // 兩側均保留 sub_1F183 非零才調整的 caller-specific 閘門。
        let target_cell = Cell { x: raw.target_record[0] as i64, y: raw.target_record[1] as i64 };
        let (target48, target4a) = self.native_ai_terrain_adjusted_words(
            &raw.target_record,
            target_cell,
            word(&raw.target_record, 0x48),
            word(&raw.target_record, 0x4a),
        )?;
        let helper = native_ai_physical_helper1_debe(&raw.target_record, raw.destination, item_rows)?;
        Ok(NativePhysicalAttackScoreInput {
            actor_word48: actor48,
            actor_word4_a: actor4a,
            target_word48: target48,
            target_word4_a: target4a,
            target_word40: word(&raw.target_record, 0x40),
            raw_target_byte8: raw.target_record[8],
            raw_helper1_debe_result: helper,
        })
    }

    /// Go：`native_ai_runtime.go:214`，`func (s *State) nativeAITerrainAdjustedWords(record []byte, destination Cell, word48, word4a int) (int, int, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ai_terrain_adjusted_words(
        &self,
        record: &[u8],
        destination: Cell,
        word48: i64,
        word4a: i64,
    ) -> Result<(i64, i64)> {
        if !native_aiis_terrain_special_record(record) {
            return Ok((word48, word4a));
        }
        if destination.x < 0
            || destination.y < 0
            || destination.x >= self.w
            || destination.y >= self.h
            || self.native_terrain_move_codes.len() as i64 != self.w.wrapping_mul(self.h)
        {
            return Err(battle_err!("native AI terrain score provenance is unavailable"));
        }
        let code = self.native_terrain_move_codes[(destination.y * self.w + destination.x) as usize];
        let (ap_pct, dp_pct, ok) = native_terrain_apdp_pct(code);
        if !ok {
            return Err(battle_err!("native AI terrain control byte {code} is unknown"));
        }
        Ok((word48 + word48 * ap_pct / 100, word4a + word4a * dp_pct / 100))
    }
}

/// Go：`native_ai_runtime.go:232`，`func nativeAIPhysicalHelper1DEBE(targetRecord []byte, destination Cell, itemRows []byte) (int, error)`
pub(crate) fn native_ai_physical_helper1_debe(
    target_record: &[u8],
    destination: Cell,
    item_rows: &[u8],
) -> Result<i64> {
    if target_record.len() as i64 != NATIVE_RECORD_SIZE {
        return Err(battle_err!("native AI 0x1debe target record is malformed"));
    }
    if target_record[0x26] != 0 {
        return Ok(-1);
    }
    let target = Cell { x: target_record[0] as i64, y: target_record[1] as i64 };
    if abs_int(target.x - destination.x) + abs_int(target.y - destination.y) != 1 {
        return Ok(-1);
    }
    let (source, found) = resolve_native_ai_physical_item_source(target_record, item_rows)?;
    if !found || source.raw_geometry_byte0_b > 1 {
        return Ok(-1);
    }
    Ok(1)
}

/// Go：`native_ai_runtime.go:253`，`func nativeAIDirectionPath(start Cell, directions []byte) ([]Cell, error)`
pub(crate) fn native_ai_direction_path(start: Cell, directions: &[u8]) -> Result<Vec<Cell>> {
    let mut path = Vec::with_capacity(directions.len() + 1);
    path.push(start);
    let mut current = start;
    for (index, &direction) in directions.iter().enumerate() {
        match direction {
            NATIVE_PATH_DOWN => current.y += 1,
            NATIVE_PATH_LEFT => current.x -= 1,
            NATIVE_PATH_UP => current.y -= 1,
            NATIVE_PATH_RIGHT => current.x += 1,
            _ => return Err(battle_err!("native AI path direction {direction} at step {index} is unknown")),
        }
        path.push(current);
    }
    Ok(path)
}

impl State {
    /// nativeAIMode2IdleRecoveryPlan 是 0x14237 回傳零之後的收尾：0x13C06/0x13C0F 消費 0x13FD4。
    /// 接受恢復與合法的閘門拒絕都要保留，兩者皆不可掉入正規化規劃器（planner）。
    ///
    /// Go：`native_ai_runtime.go:277`，`func (s *State) nativeAIMode2IdleRecoveryPlan(u *Unit, records []byte, actor int) (*AIPlan, bool, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn native_ai_mode2_idle_recovery_plan(
        &self,
        u: Option<UnitIdx>,
        records: &[u8],
        actor: i64,
    ) -> Result<(AIPlan, bool)> {
        let decision = plan_native_ai_idle_recovery(records, self.units.len() as i64, actor)
            .map_err(|err| err.wrap("native AI mode 2 0x13fd4: "))?;
        let mut plan = AIPlan {
            u,
            spell_id: -1,
            native_mode2_physical: true,
            native_mode_fallback_active: true,
            native_mode_fallback: 2,
            native_scored_commands: self.native_ai_plan_scored_commands(u),
            ..Default::default()
        };
        if decision.accepted {
            plan.native_idle_recovery = Some(decision);
        }
        Ok((plan, true))
    }
}

#[cfg(test)]
pub(crate) mod d_test_support;

#[cfg(test)]
pub(crate) mod native_ai_runtime_test;

#[cfg(test)]
mod native_ai_target_terrain_test;
