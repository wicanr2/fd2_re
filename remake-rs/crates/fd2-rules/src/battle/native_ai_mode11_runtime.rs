//! Go 來源：`remake/internal/battle/native_ai_mode11_runtime.go`（164 行）。

use super::*;

impl State {
    /// nextNativeAIMode11Plan lowers the direct 0x13A9F mode-11 branch without
    /// routing it through 0x14EF0.  The two producers are evaluated before any
    /// action is committed, then their raw gate order is preserved in
    /// NativeMode11Stages.  A missing producer/table/record is an error rather
    /// than permission to use the normalized planner.
    ///
    /// Go：`native_ai_mode11_runtime.go:10`，`func (s *State) nextNativeAIMode11Plan(u *Unit) (*AIPlan, bool, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn next_native_ai_mode11_plan(&self, u: Option<UnitIdx>) -> Result<(Option<AIPlan>, bool)> {
        let Some(ui) = u.filter(|&i| {
            self.units
                .get(i)
                .is_some_and(|unit| unit.has_native_record_byte34 && unit.native_record_byte34 & 0x0f == 11)
        }) else {
            return Ok((None, false));
        };
        if self.native_command_book.len() as i64 != NATIVE_COMMAND_RECORD_COUNT
            || self.native_future_item_rows.is_empty()
        {
            return Err(battle_err!("native AI mode 11 command/item producer tables are unavailable"));
        }
        let (actor, selector, records, actor_record, base_flags, cost_row) = self.native_ai_mode_runtime_context(u)?;
        let (physical, physical_ok) = self
            .native_ai14_ef0_physical_selection(actor, selector, &records, &actor_record, &base_flags, &cost_row)
            .map_err(|err| err.wrap("native AI mode 11 0x14237: "))?;
        let mut gate = native_ai1_f183_target_gate(&records);
        let command = score_native_ai1598_a(
            self.w,
            self.h,
            &records,
            self.units.len() as i64,
            actor,
            selector,
            Some(&self.units[ui]),
            &self.native_command_book,
            &base_flags,
            &self.native_terrain_move_codes,
            &cost_row,
            Some(&mut *gate),
        )
        .map_err(|err| err.wrap("native AI mode 11 0x1598A: "))?;
        let tx = select_native_ai_mode11_transaction(
            command.max_score as i32,
            native_ai_physical_gate(physical, physical_ok) as i32,
            true,
            true,
        )?;
        let stages = tx.stages()?;
        let mut plan =
            AIPlan { u, spell_id: -1, native_mode11_stages: Vec::with_capacity(stages.len()), ..Default::default() };
        for stage in stages {
            let mut stage_plan = NativeAIMode11StagePlan { stage, ..Default::default() };
            match stage.route {
                NATIVE_AI_MODE11_CALL15311 => {
                    if !command.has_positive_winner
                        || command.max_score < 6
                        || command.positive_winner.target_indices.is_empty()
                    {
                        return Err(battle_err!("native AI mode 11 0x15311 winner provenance is unavailable"));
                    }
                    let winner = &command.positive_winner;
                    if winner.command_id < 0 || winner.command_id >= self.native_command_book.len() as i64 {
                        return Err(battle_err!("native AI mode 11 command id={} is unavailable", winner.command_id));
                    }
                    let target_index = winner.target_indices[0] as i64;
                    if target_index >= self.units.len() as i64 {
                        return Err(battle_err!("native AI mode 11 command target provenance is unavailable"));
                    }
                    let mut action = self
                        .native_ai_plan_for_stationary_selection(
                            u,
                            &actor_record,
                            Cell { x: winner.x, y: winner.y },
                            target_index,
                        )
                        .map_err(|err| err.wrap("native AI mode 11 0x15311 path: "))?;
                    action.native_action_kind = NATIVE_AI_ACTION_COMMAND;
                    action.native_command_id = winner.command_id;
                    action.native_action_score = command.max_score;
                    action.native_scored_commands = self.native_ai_plan_scored_commands(u);
                    stage_plan.action = Some(action);
                }
                NATIVE_AI_MODE11_CALL1548_E => {
                    if !physical_ok || physical.ranking.priority < 6 {
                        return Err(battle_err!("native AI mode 11 0x1548e physical winner provenance is unavailable"));
                    }
                    let mut action = self
                        .native_ai_plan_for_destination(
                            u,
                            &actor_record,
                            selector,
                            Cell { x: physical.candidate.destination_x, y: physical.candidate.destination_y },
                            physical.candidate.target_index,
                            &cost_row,
                        )
                        .map_err(|err| err.wrap("native AI mode 11 0x1548e path: "))?;
                    action.native_action_kind = NATIVE_AI_ACTION_PHYSICAL;
                    action.native_action_score = physical.ranking.priority;
                    action.native_scored_commands = self.native_ai_plan_scored_commands(u);
                    stage_plan.action = Some(action);
                }
                NATIVE_AI_MODE11_CALL14121 => {
                    let (action, recovery) = self.native_ai_mode11_blocked_or_recovery(
                        u,
                        actor,
                        selector,
                        &records,
                        &actor_record,
                        &base_flags,
                        &cost_row,
                    )?;
                    (stage_plan.action, stage_plan.recovery) = (action, recovery);
                }
                _ => {
                    return Err(battle_err!("native AI mode 11 route {:#x} is not executable", stage.route.0));
                }
            }
            plan.native_mode11_stages.push(stage_plan);
        }
        if plan.native_mode11_stages.is_empty() {
            return Err(battle_err!("native AI mode 11 has no stages"));
        }
        Ok((Some(plan), true))
    }
}

/// nativeAIPhysicalGate is the raw [0x53C4F] threshold producer.  The helper
/// keeps the no-candidate value below the signed >=6 gate; it never invents a
/// target when 0x14237 did not return one.
///
/// Go：`native_ai_mode11_runtime.go:114`，`func nativeAIPhysicalGate(selection NativePhysicalAttackSelection, ok bool) int`
pub(crate) fn native_ai_physical_gate(selection: NativePhysicalAttackSelection, ok: bool) -> i64 {
    if !ok {
        return 0;
    }
    selection.ranking.priority
}

impl State {
    /// nativeAIMode11BlockedOrRecovery is the direct 0x14121 -> 0x13FD4 edge.  A
    /// found raw blocked cell becomes movement-only.  If no cell is found, the
    /// raw HP/gate decision is retained; an unaccepted decision is a legitimate
    /// common-tail no-op, not a normalized wait or attack.
    ///
    /// Go：`native_ai_mode11_runtime.go:125`，`func (s *State) nativeAIMode11BlockedOrRecovery( u *Unit, actor, selector int, records, actorRecord, baseFlags, costRow []byte, ) (*AIPlan, *NativeAIIdleRecoveryDecision, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn native_ai_mode11_blocked_or_recovery(
        &self,
        u: Option<UnitIdx>,
        actor: i64,
        selector: i64,
        records: &[u8],
        actor_record: &[u8],
        base_flags: &[u8],
        cost_row: &[u8],
    ) -> Result<(Option<AIPlan>, Option<NativeAIIdleRecoveryDecision>)> {
        let flags = native_ai_mode_blocked_search_flags(
            self.w,
            self.h,
            records,
            self.units.len() as i64,
            selector,
            base_flags,
        )?;
        let (blocked, found) = native_path_blocked_coordinate(
            self.w,
            self.h,
            Cell { x: actor_record[0] as i64, y: actor_record[1] as i64 },
            28,
            &flags,
            &self.native_terrain_move_codes,
            cost_row,
        )?;
        if found {
            let mut action = self
                .native_ai_plan_toward_raw_destination(
                    u,
                    actor,
                    selector,
                    records,
                    actor_record,
                    base_flags,
                    cost_row,
                    blocked,
                )
                .map_err(|err| err.wrap("native AI mode 11 0x14121 path: "))?;
            action.native_mode_fallback_active = true;
            action.native_mode_fallback = 11;
            action.native_mode_write_range_zero = true;
            action.target = None;
            // 14121返回14B78的成功旗標；找到raw格但零步移動仍回零。
            // 13E52因此繼續13FD4，不把blocked-cell found當成移動完成。
            if action.path.len() > 1 {
                return Ok((Some(action), None));
            }
        }
        let decision = plan_native_ai_idle_recovery(records, self.units.len() as i64, actor)
            .map_err(|err| err.wrap("native AI mode 11 0x13fd4: "))?;
        if !decision.accepted {
            return Ok((None, None));
        }
        Ok((None, Some(decision)))
    }
}
