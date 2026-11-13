//! Go 來源：`remake/internal/battle/native_ai_action_runtime.go`（809 行）。

use super::*;

/// DebugAI 是對拍診斷用的可選記錄器；nil 時不輸出。由測試端設定。
///
/// Go：`native_ai_action_runtime.go:9`
///
/// Rust 骨架：Go 簽名是 `func(format string, args ...any)`。Rust 改收已格式化的
/// `fmt::Arguments`，呼叫端以 `format_args!` 取代 Go 的 format 字串與參數；`None` 對應 Go 的 nil。
pub static DEBUG_AI: RwLock<Option<Callback<dyn Fn(fmt::Arguments<'_>) + Send + Sync>>> = RwLock::new(None);

impl State {
    /// nextNativeAI14EF0Plan is the first runtime consumer of the three raw
    /// producers called by 0x14ef0.  It is intentionally limited to records whose
    /// mode reaches that helper and to a complete command/item/movement export.
    /// The selected route remains an address-level label; no command or item name
    /// is inferred here.
    ///
    /// Go：`native_ai_action_runtime.go:16`，`func (s *State) nextNativeAI14EF0Plan(u *Unit) (*AIPlan, bool, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn next_native_ai14_ef0_plan(&self, u: Option<UnitIdx>) -> Result<(Option<AIPlan>, bool)> {
        let Some(ui) = u.filter(|&i| self.units.get(i).is_some_and(|unit| unit.has_native_record_byte34)) else {
            return Ok((None, false));
        };
        let unit = &self.units[ui];
        let mode = (unit.native_record_byte34 & 0x0f) as i64;
        if !native_ai_mode_uses14_ef0(mode) {
            // Known modes that do not enter 0x14ef0 are handled by the separate raw
            // mode bridge below.  Returning unhandled here is intentional: the
            // caller must try that bridge before it is allowed to consider the
            // normalized planner.
            return Ok((None, false));
        }
        if mode == 9 {
            // 0x13DA9 resolves +0x35 through 0x12C60 before any scoring. A found
            // raw +8 target goes straight to the movement bridge below; only -1
            // reaches 0x13AF3 -> 0x14EF0. See fd2_ch18_mode9_dispatch_20261002.json.
            if !unit.has_native_record_byte35 {
                return Err(battle_err!("native AI mode 9 raw target provenance is unavailable"));
            }
            let records = native_ai_scoring_records(&self.units)?;
            if native_ai_raw_record8_index(&records, self.units.len() as i64, unit.native_record_byte35 as i64).1 {
                return Ok((None, false));
            }
        }
        if self.native_command_book.len() as i64 != NATIVE_COMMAND_RECORD_COUNT
            || self.native_future_item_rows.is_empty()
        {
            // Keep the previously proven mode-2 physical slice usable in small raw
            // fixtures that intentionally omit the command/item tables.  Production
            // states bind both tables before reaching this consumer.
            if mode == 2 {
                return Ok((None, false));
            }
            return Err(battle_err!("native AI 0x14ef0 command/item tables are unavailable"));
        }
        if !unit.has_native_record_byte6 || self.units.is_empty() {
            return Err(battle_err!("native AI 0x14ef0 selector/runtime roster provenance is unavailable"));
        }
        // Go 以指標相等在 `Units` 裡找 actor；Rust 的 `u` 已經是索引。
        let actor = ui as i64;
        let selector = unit.native_record_byte6 as i64;
        if selector != 0 && selector != 1 {
            return Err(battle_err!("native AI 0x14ef0 selector={selector} is outside the verified 0/1 caller set"));
        }
        let wh = self.w.wrapping_mul(self.h);
        if self.native_movement_cost_rows.len() as i64 != NATIVE_MOVEMENT_COST_ROW_COUNT
            || self.native_terrain_move_codes.len() as i64 != wh
            || self.native_composition_event_bytes.len() as i64 != wh
        {
            return Err(battle_err!("native AI 0x14ef0 movement/terrain/composition provenance is unavailable"));
        }
        let records = native_ai_scoring_records(&self.units)?;
        let rs = NATIVE_RECORD_SIZE as usize;
        let actor_record = &records[ui * rs..(ui + 1) * rs];
        let base_flags = native_composition_base_flags(self.w, self.h, &self.native_composition_event_bytes)?;
        let cost_row = native_ai_cost_row_for_record(actor_record, &self.native_movement_cost_rows)?;

        let (physical, physical_ok) =
            self.native_ai14_ef0_physical_selection(actor, selector, &records, actor_record, &base_flags, &cost_row)?;
        if let Some(debug) = debug_ai() {
            debug(format_args!(
                "0x14ef0 actor={} at ({},{}) physicalOK={} candidate={:?} priority={}",
                actor, actor_record[0], actor_record[1], physical_ok, physical.candidate, physical.ranking.priority
            ));
        }
        let mut gate = native_ai1_f183_target_gate(&records);
        let command = score_native_ai1598_a(
            self.w,
            self.h,
            &records,
            self.units.len() as i64,
            actor,
            selector,
            Some(unit),
            &self.native_command_book,
            &base_flags,
            &self.native_terrain_move_codes,
            &cost_row,
            Some(&mut *gate),
        )?;
        let item = score_native_ai1567_e(
            self.w,
            self.h,
            &records,
            self.units.len() as i64,
            actor,
            selector,
            &self.native_future_item_rows,
            &self.native_command_book,
            &base_flags,
        )?;

        // 0x14ef0 always reads the target word from the raw pair selected by
        // 0x14237 ([0x53c4b]), even when a later command/item route wins.  Keep that
        // pointer separate from each producer's own target; choosing a command
        // target here would silently change the tie predicate at 0x14f5d.
        let mut target_index = -1;
        if physical_ok {
            target_index = physical.candidate.target_index;
        } else if command.has_positive_winner && !command.positive_winner.target_indices.is_empty() {
            // A zero C4F producer still leaves the raw target word read by the
            // dispatcher; in a detached fixture the only available provenance is
            // the command candidate, so retain that limitation explicitly.
            target_index = command.positive_winner.target_indices[0] as i64;
        } else if item.has_positive_winner && !item.target_indices.is_empty() {
            target_index = item.target_indices[0] as i64;
        }
        if target_index < 0 || target_index >= self.units.len() as i64 {
            return Err(battle_err!("native AI 0x14ef0 selected target provenance is unavailable"));
        }
        let ti = target_index as usize;
        let target_record = &records[ti * rs..(ti + 1) * rs];
        let mut r#in = NativeAI14EF0Input {
            score_c4_f: 0,
            score_c23: command.max_score as i32,
            score_c33: item.max_score as i32,
            record34: unit.native_record_byte34,
            actor_word48: u16::from_le_bytes([actor_record[0x48], actor_record[0x49]]),
            target_word4_a: u16::from_le_bytes([target_record[0x4a], target_record[0x4b]]),
            has_raw_score_c4_f: true,
            has_raw_score_c23: true,
            has_raw_score_c33: true,
            has_raw_record34: true,
            has_raw_actor_word48: true,
            has_raw_target_word4_a: true,
            ..Default::default()
        };
        if physical_ok {
            // sub_14237 writes priority (8/18) to [0x53c4f]; its larger
            // per-candidate arithmetic remains the separate tie word [0x53c43].
            r#in.score_c4_f = physical.ranking.priority as i32;
        }
        if command.has_positive_winner {
            let id = command.positive_winner.command_id;
            if id < 0 || id >= self.native_command_book.len() as i64 {
                return Err(battle_err!("native AI 0x14ef0 selected command id={id} is unavailable"));
            }
            r#in.command_id = id as i32;
            r#in.command_word = self.native_command_book[id as usize].damage as u16;
            (r#in.has_raw_command_id, r#in.has_raw_command_word) = (true, true);
        }
        let decision = plan_native_ai14_ef0_dispatch(r#in)?;
        let route = decision.tail;
        if let Some(debug) = debug_ai() {
            debug(format_args!(
                "0x14ef0 dispatch actor={} C4F={} C23={} C33={} route={} accepted={}",
                actor, r#in.score_c4_f, r#in.score_c23, r#in.score_c33, route.0, decision.accepted
            ));
        }
        match route {
            NATIVE_AI14_EF0_CALL1548_E => {
                if !physical_ok {
                    return Err(battle_err!("native AI 0x14ef0 selected 0x1548e without physical candidate"));
                }
                let mut plan = self.native_ai_plan_for_destination(
                    u,
                    actor_record,
                    selector,
                    Cell { x: physical.candidate.destination_x, y: physical.candidate.destination_y },
                    physical.candidate.target_index,
                    &cost_row,
                )?;
                plan.native_action_kind = NATIVE_AI_ACTION_PHYSICAL;
                plan.native_mode2_physical = mode == 2;
                plan.native_ai14_ef0_route = route;
                plan.native_action_score = physical.ranking.priority;
                Ok((Some(plan), true))
            }
            NATIVE_AI14_EF0_CALL15311 => {
                if !command.has_positive_winner || command.positive_winner.target_indices.is_empty() {
                    return Err(battle_err!("native AI 0x14ef0 selected 0x15311 without command target"));
                }
                let mut plan = self.native_ai_plan_for_stationary_selection(
                    u,
                    actor_record,
                    Cell { x: command.positive_winner.x, y: command.positive_winner.y },
                    command.positive_winner.target_indices[0] as i64,
                )?;
                plan.native_action_kind = NATIVE_AI_ACTION_COMMAND;
                plan.native_command_id = command.positive_winner.command_id;
                plan.native_ai14_ef0_route = route;
                plan.native_action_score = command.max_score;
                Ok((Some(plan), true))
            }
            NATIVE_AI14_EF0_CALL15055 => {
                if !item.has_positive_winner || item.target_indices.is_empty() {
                    return Err(battle_err!("native AI 0x14ef0 selected 0x15055 without item target"));
                }
                // 0x15055 沒有 0x14B78：winner 的 [0x53C37/0x53C3B] 是 0x1567E 在 actor 原地的指令
                // 目標場內選出的效果格，0x150BA 聚焦自己、0x1515B 聚焦效果格後就原地用物品
                // （第十章 r4 seq 3614 record 54 在 (10,28) 對 (9,28) 的 record 2 用物品 194）。
                let mut plan = self.native_ai_plan_for_stationary_selection(
                    u,
                    actor_record,
                    Cell { x: item.x, y: item.y },
                    item.target_indices[0] as i64,
                )?;
                plan.native_action_kind = NATIVE_AI_ACTION_ITEM;
                (plan.native_item_slot, plan.native_item_id) = (item.inventory_slot, item.item_id);
                plan.native_item_target_indices = item.target_indices.clone();
                plan.native_ai14_ef0_route = route;
                plan.native_action_score = item.max_score;
                Ok((Some(plan), true))
            }
            _ => {
                if !decision.accepted {
                    // 0x14F7D回0才讓caller進入mode後備。
                    return Ok((None, false));
                }
                // 0x15041→0x1504B：沒有handler仍清range、回1。
                // 原地交共用成功收尾，不聚焦、不尋路、不呼叫0x13FD4。
                Ok((
                    Some(AIPlan {
                        u,
                        spell_id: -1,
                        native_mode_write_range_zero: true,
                        native_action_destination: Cell { x: actor_record[0] as i64, y: actor_record[1] as i64 },
                        ..Default::default()
                    }),
                    true,
                ))
            }
        }
    }
}

/// Go：`native_ai_action_runtime.go:231`，`func nativeAIModeUses14EF0(mode int) bool`
pub(crate) fn native_ai_mode_uses14_ef0(mode: i64) -> bool {
    matches!(mode, 0 | 1 | 2 | 3 | 5 | 9 | 10)
}

impl State {
    /// Go：`native_ai_action_runtime.go:240`，`func (s *State) nativeAI14EF0PhysicalSelection( actor, selector int, records, actorRecord, baseFlags, costRow []byte, ) (NativePhysicalAttackSelection, bool, error)`
    pub(crate) fn native_ai14_ef0_physical_selection(
        &self,
        actor: i64,
        selector: i64,
        records: &[u8],
        actor_record: &[u8],
        base_flags: &[u8],
        cost_row: &[u8],
    ) -> Result<(NativePhysicalAttackSelection, bool)> {
        let (item, found) = resolve_native_ai_physical_item_source(actor_record, &self.native_future_item_rows)?;
        if !found {
            if let Some(debug) = debug_ai() {
                debug(format_args!(
                    "0x14237 actor={}: no equipped low item (inventory={})",
                    actor,
                    go_hex(&actor_record[0x0a..0x1a])
                ));
            }
            return Ok((NativePhysicalAttackSelection::default(), false));
        }
        let geometry = NativeAIPhysicalAttackGeometry {
            target_mode: item.raw_geometry_byte0_c as i64,
            target_inner_mark: item.raw_geometry_byte0_b as i64,
            target_code: native_ai_physical_target_code(selector),
        };
        let flags = native_command_runtime_flags(
            self.w,
            self.h,
            &self.native_composition_event_bytes,
            &self.units,
            selector as u8,
        )?;
        let mut resolve = |raw: NativeAIPhysicalAttackRawCandidate| {
            self.native_ai_physical_score_input(&raw, &self.native_future_item_rows)
        };
        let candidates = build_native_ai_physical_attack_candidates(
            self.w,
            self.h,
            records,
            self.units.len() as i64,
            actor,
            selector,
            actor_record[0x3b] as i64,
            geometry,
            &flags,
            base_flags,
            &self.native_terrain_move_codes,
            cost_row,
            Some(&mut resolve),
        )?;
        if let Some(debug) = debug_ai() {
            // Go 以 %+v 印結構；Rust 印 Debug 格式，只供診斷（PORTING.md B 組完成紀錄同此）。
            debug(format_args!(
                "0x14237 actor={} item={:?} geometry={:?} candidates={}",
                actor,
                item,
                geometry,
                candidates.len()
            ));
            for c in &candidates {
                debug(format_args!(
                    "  candidate dest=({},{}) target={} inputs={:?}",
                    c.destination_x, c.destination_y, c.target_index, c.inputs
                ));
            }
        }
        select_native_physical_attack_candidate(&candidates)
    }

    /// nativeAIPlanForStationarySelection owns 0x15311's selected command result.
    /// Unlike 0x1548E, 0x15311 never calls 0x14B78: its winner coordinates feed
    /// target/effect presentation while the actor remains at the current cell.
    ///
    /// Go：`native_ai_action_runtime.go:287`，`func (s *State) nativeAIPlanForStationarySelection( u *Unit, actorRecord []byte, destination Cell, targetIndex int, ) (*AIPlan, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ai_plan_for_stationary_selection(
        &self,
        u: Option<UnitIdx>,
        actor_record: &[u8],
        destination: Cell,
        target_index: i64,
    ) -> Result<AIPlan> {
        let Some(ui) = u.filter(|_| {
            actor_record.len() as i64 == NATIVE_RECORD_SIZE
                && destination.x >= 0
                && destination.y >= 0
                && destination.x < self.w
                && destination.y < self.h
                && target_index >= 0
                && target_index < self.units.len() as i64
        }) else {
            return Err(battle_err!("native AI stationary destination/target is malformed"));
        };
        let origin = Cell { x: actor_record[0] as i64, y: actor_record[1] as i64 };
        if origin.x < 0 || origin.y < 0 || origin.x >= self.w || origin.y >= self.h {
            return Err(battle_err!("native AI stationary actor origin is malformed"));
        }
        Ok(AIPlan {
            u: Some(ui),
            path: vec![origin],
            target: Some(target_index as UnitIdx),
            spell_id: -1,
            native_scored_commands: self.native_ai_plan_scored_commands(u),
            native_action_destination: destination,
            ..Default::default()
        })
    }

    /// nativeAIPlanForDestination replays 0x14b78's movement transaction for a
    /// selected raw destination.  The caller still owns the effect/target stage;
    /// this helper only returns the detached path and verified target pointer.
    ///
    /// Go：`native_ai_action_runtime.go:309`，`func (s *State) nativeAIPlanForDestination( u *Unit, actorRecord []byte, selector int, destination Cell, targetIndex int, costRow []byte, ) (*AIPlan, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn native_ai_plan_for_destination(
        &self,
        u: Option<UnitIdx>,
        actor_record: &[u8],
        selector: i64,
        destination: Cell,
        target_index: i64,
        cost_row: &[u8],
    ) -> Result<AIPlan> {
        if u.is_none()
            || actor_record.len() as i64 != NATIVE_RECORD_SIZE
            || target_index < 0
            || target_index >= self.units.len() as i64
        {
            return Err(battle_err!("native AI selected destination/target is malformed"));
        }
        let runtime_flags = native_command_runtime_flags(
            self.w,
            self.h,
            &self.native_composition_event_bytes,
            &self.units,
            selector as u8,
        )?;
        let origin = Cell { x: actor_record[0] as i64, y: actor_record[1] as i64 };
        let (directions, reachable) = native_path_directions(
            self.w,
            self.h,
            origin,
            destination,
            actor_record[0x3b] as i64,
            0,
            &runtime_flags,
            &self.native_terrain_move_codes,
            cost_row,
        )?;
        if !reachable {
            return Err(battle_err!("native AI selected destination is not path-reachable"));
        }
        let path = native_ai_direction_path(origin, &directions)?;
        Ok(AIPlan {
            u,
            path,
            target: Some(target_index as UnitIdx),
            spell_id: -1,
            native_scored_commands: self.native_ai_plan_scored_commands(u),
            native_action_destination: destination,
            ..Default::default()
        })
    }

    /// nextNativeAIModeFallbackPlan consumes only the mode branches whose inputs
    /// are explicit in the 0x13a9f disassembly.  Mode 5 now owns the raw
    /// 0x15df3/event-row state tail; its raw 0x25b45 AIL sample cue remains a
    /// separate gate. Mode 11's two-action transaction and the 0x13fd4 raw HP
    /// write still have no complete runtime owner here.
    ///
    /// Go：`native_ai_action_runtime.go:350`，`func (s *State) nextNativeAIModeFallbackPlan(u *Unit) (*AIPlan, bool, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn next_native_ai_mode_fallback_plan(&self, u: Option<UnitIdx>) -> Result<(Option<AIPlan>, bool)> {
        let Some(ui) = u.filter(|&i| self.units.get(i).is_some_and(|unit| unit.has_native_record_byte34)) else {
            return Ok((None, false));
        };
        let unit = &self.units[ui];
        let mode = (unit.native_record_byte34 & 0x0f) as i64;
        match mode {
            // These low nibbles are present in the fixed FDFIELD inventory or in
            // an explicitly recovered writer.  They must not fall through to the
            // normalized planner when their raw consumer is incomplete.
            0 | 1 | 2 | 3 | 4 | 5 | 7 | 8 | 9 | 10 | 11 => {}
            _ => return Err(battle_err!("native AI mode {mode} has no recovered dispatcher branch")),
        }
        if mode == 8 {
            // 0x13D9A直接跳0x1317D epilogue，跳過0x13E80→0x13512成功收尾。
            // 不移動、不設已行動，第二遍仍可派送；見fd2_ch14_mode8_dispatch_20261002.json。
            return Ok((
                Some(AIPlan {
                    u,
                    spell_id: -1,
                    native_mode_fallback_active: true,
                    native_mode_fallback: mode as u8,
                    ..Default::default()
                }),
                true,
            ));
        }
        if mode == 2 || mode == 11 {
            return Err(battle_err!("native AI mode {mode} fallback consumer remains unresolved"));
        }
        let (actor, selector, records, actor_record, base_flags, cost_row) = self.native_ai_mode_runtime_context(u)?;
        let rs = NATIVE_RECORD_SIZE as usize;
        if mode == 5 {
            if !unit.has_native_record_byte3_d || unit.native_record_byte3_d as usize >= self.native_event_state.len() {
                return Err(battle_err!("native AI mode 5 event-state index provenance is unavailable"));
            }
            if !self.native_chest_control_row(unit.native_record_byte3_d).2 {
                return Err(battle_err!("native AI mode 5 event-control row provenance is unavailable"));
            }
            // 0x13C42：`[0x53AD5][+0x3D]` 非零就跳 0x13B05；0x13C59：`0x15DF3` 找不到格子
            // （回 −1）也跳 0x13B05。兩種都不是錯誤，是走 mode 0 的 0x14121／0x13E9C 尾段。
            let mut event_ready = self.native_event_state[unit.native_record_byte3_d as usize] == 0;
            let mut intended = Cell::default();
            if event_ready {
                match self.native_ai_mode5_event_cell(unit.native_record_byte3_d) {
                    Ok(cell) => intended = cell,
                    Err(_) => event_ready = false,
                }
            }
            if event_ready {
                let mut plan = self.native_ai_plan_toward_raw_destination(
                    u,
                    actor,
                    selector,
                    &records,
                    &actor_record,
                    &base_flags,
                    &cost_row,
                    intended,
                )?;
                plan.native_mode_fallback_active = true;
                plan.native_mode_fallback = mode as u8;
                plan.native_mode_write_range_zero = true;
                plan.native_mode_focus_actor = true; // 0x13C39
                // 0x13C8C..0x13CB9：0x14B78 之後比記錄的 x／y 與事件格，走到了才跑事件尾段，
                // 沒走到就直接進共用收尾（0x13E5A）；0x13C7F 回 0（沒走成）先進 0x13FD4。
                if plan.native_action_destination == intended {
                    plan.native_mode_event_active = true;
                    plan.native_mode_event_id = unit.native_record_byte3_d;
                    plan.native_mode_event_destination = intended;
                }
                if plan.path.len() <= 1 {
                    self.attach_native_fallback_idle_recovery(&mut plan, &records, actor)?;
                }
                return Ok((Some(plan), true));
            }
            // 落到下面 mode 0／1 的共用尾段（0x13B05）。
        }
        if mode == 3 || mode == 9 {
            // 0x12c60 receives raw record +0x35 and returns the first live record
            // whose raw +0x08 matches it.  It is not a nearest-unit search and the
            // returned record's coordinates remain the only accepted destination.
            let (target_index, ok) =
                native_ai_raw_record8_index(&records, self.units.len() as i64, unit.native_record_byte35 as i64);
            if !ok {
                return Err(battle_err!("native AI mode {mode} 0x12c60 found no live raw +0x08 target"));
            }
            let target_record = &records[target_index as usize * rs..];
            let intended = Cell { x: target_record[0] as i64, y: target_record[1] as i64 };
            if intended.x < 0 || intended.y < 0 || intended.x >= self.w || intended.y >= self.h {
                return Err(battle_err!("native AI mode {mode} 0x12c60 target is outside the map"));
            }
            let mut plan = self.native_ai_plan_toward_raw_destination(
                u,
                actor,
                selector,
                &records,
                &actor_record,
                &base_flags,
                &cost_row,
                intended,
            )?;
            plan.native_mode_fallback_active = true;
            plan.native_mode_fallback = mode as u8;
            // mode 3 explicitly clears [0x51a83] after 0x14b78.  Mode 9 jumps
            // through the shared 0x14b78 block without that explicit caller write.
            plan.native_mode_write_range_zero = mode == 3;
            plan.native_mode_focus_actor = true;
            if plan.path.len() <= 1 {
                // 0x13BC0（mode 3）／0x13C06（mode 9 經 0x13BFE）：0x14B78 沒走成回 0 → 0x13FD4。
                self.attach_native_fallback_idle_recovery(&mut plan, &records, actor)?;
            }
            return Ok((Some(plan), true));
        }
        if mode == 4 || mode == 7 || mode == 10 {
            let intended = Cell { x: unit.native_record_byte35 as i64, y: unit.native_record_byte36 as i64 };
            if intended.x < 0 || intended.y < 0 || intended.x >= self.w || intended.y >= self.h {
                return Err(battle_err!(
                    "native AI mode {mode} raw destination ({},{}) is outside the map",
                    intended.x,
                    intended.y
                ));
            }
            let mut plan = self.native_ai_plan_toward_raw_destination(
                u,
                actor,
                selector,
                &records,
                &actor_record,
                &base_flags,
                &cost_row,
                intended,
            )?;
            plan.native_mode_fallback_active = true;
            plan.native_mode_fallback = mode as u8;
            plan.native_mode_write_byte5 = mode == 7 && plan.native_action_destination == intended;
            plan.native_mode_write_range_zero = true;
            plan.native_mode_focus_actor = true;
            if plan.path.len() <= 1 {
                // 0x13C06（mode 4／10）／0x13D53（mode 7）：0x14B78 沒走成回 0 → 0x13FD4。
                self.attach_native_fallback_idle_recovery(&mut plan, &records, actor)?;
            }
            return Ok((Some(plan), true));
        }

        // Modes 0/1 first call 0x14121 with the raw opposite-group blocker grid.
        // A successful blocked coordinate is only useful when it maps back to one
        // and only one active raw record; otherwise the target identity would be a
        // normalized guess.
        let flags = native_ai_mode_blocked_search_flags(
            self.w,
            self.h,
            &records,
            self.units.len() as i64,
            selector,
            &base_flags,
        )?;
        let actor_cell = Cell { x: actor_record[0] as i64, y: actor_record[1] as i64 };
        let (blocked, found) = native_path_blocked_coordinate(
            self.w,
            self.h,
            actor_cell,
            28,
            &flags,
            &self.native_terrain_move_codes,
            &cost_row,
        )?;
        if found && native_ai_active_unit_at_raw_cell(&records, self.units.len() as i64, selector, blocked).1 {
            let mut plan = self.native_ai_plan_toward_raw_destination(
                u,
                actor,
                selector,
                &records,
                &actor_record,
                &base_flags,
                &cost_row,
                blocked,
            )?;
            plan.native_mode_fallback_active = true;
            plan.native_mode_fallback = mode as u8;
            // 0x14121/0x13e9c are movement-only callers; target is retained
            // only as raw provenance for tests and is not executed as an attack.
            plan.target = None;
            // 0x14121 只在 0x14B78 真的走了（0x4E1A6 回傳的路徑長度非 0，0x14EC4 呼叫
            // 0x13488）才回 1；落點就是原格時回 0，mode 0 的 0x13A9F 分派（0x13B0F）
            // 接著走 0x13E9C 以最近的對立單位再規劃一次。第七章 r6 第 7 回合記錄 23
            // 在 (13,11)：0x14121 的阻擋格是 (14,32)、長路徑第一步就被 (13,12) 的
            // 零預算格截住而留在原地，原版改以索爾 (12,12) 為目標走到 (12,11)。
            if plan.path.len() > 1 {
                return Ok((Some(plan), true));
            }
            if mode == 1 {
                // 0x13B3F 0x14121 → jmp 0x13B1E → 0x13C06：沒走成（回 0）就 0x13FD4。
                self.attach_native_fallback_idle_recovery(&mut plan, &records, actor)?;
                return Ok((Some(plan), true));
            }
        }
        if mode == 1 {
            if found {
                return Err(battle_err!("native AI mode 1 0x14121 blocked coordinate has no unique raw record"));
            }
            // 0x14121 沒有阻擋格就回 0，同樣經 0x13C06 呼叫 0x13FD4。
            let (plan, handled) = self.native_ai_idle_recovery_only_plan(u, &records, actor, mode)?;
            return Ok((Some(plan), handled));
        }
        let (nearest, found_nearest) =
            select_native_nearest_opposite_coordinate(&records, self.units.len() as i64, actor, selector)?;
        // 0x13F67..0x13F84：沒有對立單位、或最近的就在原格，0x13E9C 不聚焦不走行、回 0；
        // 0x13B21 jmp 0x13C06 → 0x13FD4。
        if !found_nearest || nearest == actor_cell {
            let (plan, handled) = self.native_ai_idle_recovery_only_plan(u, &records, actor, mode)?;
            return Ok((Some(plan), handled));
        }
        let mut plan = self.native_ai_plan_toward_raw_destination(
            u,
            actor,
            selector,
            &records,
            &actor_record,
            &base_flags,
            &cost_row,
            nearest,
        )?;
        if plan.path.len() <= 1 {
            // 0x13F86..0x13FBF：聚焦、0x14B78 沒走成回 0 → 0x13C06 → 0x13FD4。第八章 r1
            // 第 7 回合記錄 36 在 (15,21) 被圍住，原版 HP 7→43（180/5）。
            self.attach_native_fallback_idle_recovery(&mut plan, &records, actor)?;
        }
        plan.native_mode_fallback_active = true;
        plan.native_mode_fallback = mode as u8;
        plan.native_mode_focus_actor = true; // 0x13F86
        // Go `native_ai_action_runtime.go` 531 行以 `nearest, found, err :=` 在同一作用域重新賦值了
        // 487 行的 `found`，558 行寫進 `NativeModeBlockedFound` 的其實是「找到最近對立單位」，走到這裡
        // 恆為 true（阻擋格仍是 0x14121 的結果）。這一欄只供 `FD2_SHOT_AI` 診斷，照凍結 Go 版的
        // 實際值寫入，讓狀態軌跡與 Go 相同；`rules-int` 黃金涵蓋這條路徑。
        (plan.native_mode_blocked_cell, plan.native_mode_blocked_found) = (blocked, found_nearest);
        for index in 0..self.units.len() {
            let record = &records[index * rs..];
            if native_ai_opposite_selector_group(record[6], selector) {
                plan.native_mode_opposite.push(Cell { x: record[0] as i64, y: record[1] as i64 });
            }
        }
        plan.target = None;
        Ok((Some(plan), true))
    }

    /// nativeAIIdleRecoveryOnlyPlan 是移動後備完全沒有落點時的 0x13C06→0x13FD4：沒有聚焦、
    /// 沒有走行，只有回復（或閘門拒絕）與共用收尾，和 mode 2 無物理候選同一種計畫形狀。
    ///
    /// Go：`native_ai_action_runtime.go:571`，`func (s *State) nativeAIIdleRecoveryOnlyPlan(u *Unit, records []byte, actor, mode int) (*AIPlan, bool, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn native_ai_idle_recovery_only_plan(
        &self,
        u: Option<UnitIdx>,
        records: &[u8],
        actor: i64,
        mode: i64,
    ) -> Result<(AIPlan, bool)> {
        let decision = plan_native_ai_idle_recovery(records, self.units.len() as i64, actor)
            .map_err(|err| err.wrap(format!("native AI mode {mode} 0x13fd4: ")))?;
        let mut plan = AIPlan {
            u,
            spell_id: -1,
            native_mode_fallback_active: true,
            native_mode_fallback: mode as u8,
            native_scored_commands: self.native_ai_plan_scored_commands(u),
            ..Default::default()
        };
        if decision.accepted {
            plan.native_idle_recovery = Some(decision);
        }
        Ok((plan, true))
    }

    /// Go：`native_ai_action_runtime.go:587`，`func (s *State) attachNativeFallbackIdleRecovery(plan *AIPlan, records []byte, actor int) error`
    pub(crate) fn attach_native_fallback_idle_recovery(
        &self,
        plan: &mut AIPlan,
        records: &[u8],
        actor: i64,
    ) -> Result<()> {
        let decision = plan_native_ai_idle_recovery(records, self.units.len() as i64, actor)
            .map_err(|err| err.wrap(format!("native AI mode {} 0x13fd4: ", plan.native_mode_fallback)))?;
        plan.target = None;
        if decision.accepted {
            plan.native_fallback_idle_recovery = Some(decision);
        }
        Ok(())
    }

    /// nativeAIMovementApproachPoint 是 0x14B78 的第 1～3 段（0x14C16..0x14D43）：
    /// intended 直接走得到就原樣回傳；否則沿不看單位的長路徑（預算 0x1C、mode 1）
    /// 找 MV 預算場內走得到的最後一格。長路徑也找不到時同樣原樣回傳。
    ///
    /// Go：`native_ai_action_runtime.go:602`，`func (s *State) nativeAIMovementApproachPoint( origin, intended Cell, mv, actor, selector int, records, baseFlags, runtimeFlags, costRow []byte, ) (Cell, error)`
    pub(crate) fn native_ai_movement_approach_point(
        &self,
        origin: Cell,
        intended: Cell,
        mv: i64,
        actor: i64,
        selector: i64,
        records: &[u8],
        base_flags: &[u8],
        runtime_flags: &[u8],
        cost_row: &[u8],
    ) -> Result<Cell> {
        let (_, direct) = native_path_directions(
            self.w,
            self.h,
            origin,
            intended,
            mv,
            0,
            runtime_flags,
            &self.native_terrain_move_codes,
            cost_row,
        )?;
        if direct {
            return Ok(intended);
        }
        // 0x4DBFC 之後只剩 FDFIELD 基底旗標：長路徑可以穿過任何單位。
        let (long, found) = native_path_directions(
            self.w,
            self.h,
            origin,
            intended,
            0x1c,
            1,
            base_flags,
            &self.native_terrain_move_codes,
            cost_row,
        )?;
        if !found {
            return Ok(intended);
        }
        let field = native_movement_budget_field(
            self.w,
            self.h,
            records,
            self.units.len() as i64,
            actor,
            selector,
            mv,
            base_flags,
            &self.native_terrain_move_codes,
            cost_row,
        )?;
        let (mut cell, mut last) = (origin, intended);
        for direction in long {
            match direction {
                NATIVE_PATH_LEFT => cell.x -= 1,
                NATIVE_PATH_UP => cell.y -= 1,
                NATIVE_PATH_DOWN => cell.y += 1,
                _ => cell.x += 1,
            }
            if cell.x < 0 || cell.y < 0 || cell.x >= self.w || cell.y >= self.h {
                return Err(battle_err!("native AI long path leaves the grid at ({},{})", cell.x, cell.y));
            }
            if field[(cell.y * self.w + cell.x) as usize] != 0xff {
                last = cell;
            }
        }
        Ok(last)
    }
}

/// nativeAIRawRecord8Index mirrors 0x12c60's first-match scan.  The caller
/// supplies the complete scoring-record export, so a missing +0x08 provenance
/// is rejected by NativeAIScoringRecords before this helper is reached.
///
/// Go：`native_ai_action_runtime.go:657`，`func nativeAIRawRecord8Index(records []byte, count, rawKey int) (int, bool)`
pub(crate) fn native_ai_raw_record8_index(records: &[u8], count: i64, raw_key: i64) -> (i64, bool) {
    if count < 0 || records.len() as i64 != count.wrapping_mul(NATIVE_RECORD_SIZE) || !(0..=0xff).contains(&raw_key) {
        return (-1, false);
    }
    for index in 0..count {
        let record = &records[(index * NATIVE_RECORD_SIZE) as usize..];
        if record[5] & 1 != 0 || record[8] as i64 != raw_key {
            continue;
        }
        return (index, true);
    }
    (-1, false)
}

impl State {
    /// Go：`native_ai_action_runtime.go:671`，`func (s *State) nativeAIModeRuntimeContext(u *Unit) (int, int, []byte, []byte, []byte, []byte, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ai_mode_runtime_context(
        &self,
        u: Option<UnitIdx>,
    ) -> Result<(i64, i64, Vec<u8>, Vec<u8>, Vec<u8>, Vec<u8>)> {
        let Some(ui) = u.filter(|&i| self.units.get(i).is_some_and(|unit| unit.has_native_record_byte6)) else {
            return Err(battle_err!("native AI mode lacks selector provenance"));
        };
        let selector = self.units[ui].native_record_byte6 as i64;
        if selector != 0 && selector != 1 {
            return Err(battle_err!("native AI mode selector={selector} is outside the verified 0/1 caller set"));
        }
        let wh = self.w.wrapping_mul(self.h);
        if self.native_movement_cost_rows.len() as i64 != NATIVE_MOVEMENT_COST_ROW_COUNT
            || self.native_terrain_move_codes.len() as i64 != wh
            || self.native_composition_event_bytes.len() as i64 != wh
        {
            return Err(battle_err!("native AI mode movement/terrain/composition provenance is unavailable"));
        }
        let records = native_ai_scoring_records(&self.units)?;
        // Go 以指標相等在 `Units` 裡找 actor；Rust 的 `u` 已經是索引。
        let rs = NATIVE_RECORD_SIZE as usize;
        let actor_record = records[ui * rs..(ui + 1) * rs].to_vec();
        let cost_row = native_ai_cost_row_for_record(&actor_record, &self.native_movement_cost_rows)?;
        let base_flags = native_composition_base_flags(self.w, self.h, &self.native_composition_event_bytes)?;
        Ok((ui as i64, selector, records, actor_record, base_flags, cost_row))
    }
}

/// Go：`native_ai_action_runtime.go:709`，`func nativeAIModeBlockedSearchFlags(w, h int, records []byte, count, selector int, baseFlags []byte) ([]byte, error)`
pub(crate) fn native_ai_mode_blocked_search_flags(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    selector: i64,
    base_flags: &[u8],
) -> Result<Vec<u8>> {
    if w <= 0
        || h <= 0
        || base_flags.len() as i64 != w.wrapping_mul(h)
        || count < 0
        || records.len() as i64 != count.wrapping_mul(NATIVE_RECORD_SIZE)
    {
        return Err(battle_err!("native AI mode blocker grid is malformed"));
    }
    let mut flags = base_flags.to_vec();
    for index in 0..count {
        let record = &records[(index * NATIVE_RECORD_SIZE) as usize..];
        if record[5] & 1 != 0 || !native_ai_opposite_selector_group(record[6], selector) {
            continue;
        }
        let (x, y) = (record[0] as i64, record[1] as i64);
        if x < 0 || y < 0 || x >= w || y >= h {
            return Err(battle_err!("native AI mode blocker unit {index} is outside the map"));
        }
        flags[(y * w + x) as usize] |= NATIVE_COMMAND_GRID_BLOCKED;
        for delta in NATIVE_AI_DESTINATION_NEIGHBOURS {
            let (nx, ny) = (x + delta[0], y + delta[1]);
            if nx >= 0 && ny >= 0 && nx < w && ny < h {
                flags[(ny * w + nx) as usize] |= NATIVE_COMMAND_GRID_ZERO_BUDGET;
            }
        }
    }
    Ok(flags)
}

/// Go：`native_ai_action_runtime.go:734`，`func nativeAIActiveUnitAtRawCell(records []byte, count, selector int, cell Cell) (int, bool)`
pub(crate) fn native_ai_active_unit_at_raw_cell(records: &[u8], count: i64, selector: i64, cell: Cell) -> (i64, bool) {
    for index in 0..count {
        let record = &records[(index * NATIVE_RECORD_SIZE) as usize..];
        if record[5] & 1 != 0 || !native_ai_opposite_selector_group(record[6], selector) {
            continue;
        }
        if record[0] as i64 == cell.x && record[1] as i64 == cell.y {
            return (index, true);
        }
    }
    (-1, false)
}

impl State {
    /// nativeAIPlanTowardRawDestination 重現 0x14B78 的四段：
    ///
    ///  1. 0x145CD 標記後以 MV 預算、mode 0 直接找到 intended 的路徑；找到就跳到第 4 段。
    ///  2. 找不到就 0x4DBFC 清掉單位標記（只剩 FDFIELD 基底旗標），以預算 0x1C、mode 1
    ///     找一條穿過單位的長路徑；找不到也跳到第 4 段。
    ///  3. 重新標記、0x4E040 建 MV 預算場（不做 0x146D1 同組清除），沿長路徑逐格走，
    ///     取「預算場走得到的最後一格」當新的 intended。
    ///  4. 0x14B16 候選（含 0x146D1）依 距離→XY 軸差→列優先 選實際落點，再以 MV、mode 0
    ///     取最終路徑。
    ///
    /// 第四章原版 eip-trace（work/parity-slot-ch04/ai-trace-r1）證實每個敵人都是
    /// 0x4E1A6(MV,mode0)→0x4E1A6(0x1C,mode1)→0x4E040×2→0x14B16→0x4E1A6(MV,mode0) 這個序列。
    ///
    /// Go：`native_ai_action_runtime.go:759`，`func (s *State) nativeAIPlanTowardRawDestination( u *Unit, actor, selector int, records, actorRecord, baseFlags, costRow []byte, intended Cell, ) (*AIPlan, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ai_plan_toward_raw_destination(
        &self,
        u: Option<UnitIdx>,
        actor: i64,
        selector: i64,
        records: &[u8],
        actor_record: &[u8],
        base_flags: &[u8],
        cost_row: &[u8],
        intended: Cell,
    ) -> Result<AIPlan> {
        if u.is_none() || actor_record.len() as i64 != NATIVE_RECORD_SIZE {
            return Err(battle_err!("native AI mode destination context is malformed"));
        }
        let origin = Cell { x: actor_record[0] as i64, y: actor_record[1] as i64 };
        let mv = actor_record[0x3b] as i64;
        let runtime_flags = native_command_runtime_flags(
            self.w,
            self.h,
            &self.native_composition_event_bytes,
            &self.units,
            selector as u8,
        )?;
        let intended = self.native_ai_movement_approach_point(
            origin,
            intended,
            mv,
            actor,
            selector,
            records,
            base_flags,
            &runtime_flags,
            cost_row,
        )?;
        let candidates = native_ai_physical_destinations(
            self.w,
            self.h,
            records,
            self.units.len() as i64,
            actor,
            selector,
            mv,
            base_flags,
            &self.native_terrain_move_codes,
            cost_row,
        )?;
        let (destination, ok) = select_native_movement_destination(&candidates, intended);
        if !ok {
            return Err(battle_err!("native AI mode destination has no reachable cell"));
        }
        let (directions, reachable) = native_path_directions(
            self.w,
            self.h,
            origin,
            destination,
            mv,
            0,
            &runtime_flags,
            &self.native_terrain_move_codes,
            cost_row,
        )?;
        if !reachable {
            return Err(battle_err!("native AI mode destination is not path-reachable"));
        }
        let path = native_ai_direction_path(origin, &directions)?;
        Ok(AIPlan {
            u,
            path,
            spell_id: -1,
            native_action_destination: destination,
            native_scored_commands: self.native_ai_plan_scored_commands(u),
            native_mode_intended: intended,
            native_mode_candidates: candidates,
            ..Default::default()
        })
    }
}

/// 取出目前的 `DEBUG_AI` 記錄器（Go `if DebugAI != nil`）。鎖中毒時沿用內部值，與 B 組相同。
/// Rust 新增的小工具，沒有對應的 Go 函式。
fn debug_ai() -> Option<Callback<dyn Fn(fmt::Arguments<'_>) + Send + Sync>> {
    DEBUG_AI.read().unwrap_or_else(|e| e.into_inner()).clone()
}

/// Go `fmt` 對 `[]byte` 的 `%x`：小寫十六進位、不加分隔。只供 `DEBUG_AI` 診斷文字使用。
fn go_hex(bytes: &[u8]) -> String {
    bytes.iter().map(|b| format!("{b:02x}")).collect()
}
