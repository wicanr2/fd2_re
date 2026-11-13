//! 移植自 `remake/internal/battle/native_ai_runtime_test.go`。
//!
//! 除了 `nativeAIPhysicalHelper1DEBE` 的純函式測試，其餘都經由 E 組的 `State::next_ai_plan`
//! （`combat.go:NextAIPlan`），E 合併前停在 E 的 `todo!()`。Go 以 `*Unit` 指標比較目標身份，
//! Rust 比較 `state.units` 的索引。

use super::*;
use crate::battle::native_ai_runtime::d_test_support::*;

const ROW: usize = NATIVE_ITEM_EFFECT_ROW_SIZE as usize;

/// Go `nativeAIRuntimeUnit`。
pub(crate) fn native_ai_runtime_unit(x: i64, y: i64, selector: u8, mode: u8) -> Unit {
    Unit {
        camp: ENEMY,
        x,
        y,
        on_field: true,
        hp: 20,
        max_hp: 20,
        mp: 4,
        max_mp: 4,
        ap: 20,
        dp: 1,
        mv: 2,
        battle_fig: 1,
        has_battle_fig: true,
        native_record_byte5: 0,
        has_native_record_byte5: true,
        native_record_byte6: selector,
        has_native_record_byte6: true,
        native_record_byte34: mode,
        has_native_record_byte34: true,
        native_record_byte35: 0,
        has_native_record_byte35: true,
        native_record_byte36: 0,
        has_native_record_byte36: true,
        native_record_byte8: 1,
        has_native_record_byte8: true,
        native_record_race: 0,
        has_native_record_race: true,
        native_record_class: 0,
        has_native_record_class: true,
        native_record_word42: 20,
        has_native_record_word42: true,
        native_record_word46: 4,
        has_native_record_word46: true,
        native_map_presentation: NativeMapPresentationState { x: x as u8, y: y as u8, ..Default::default() },
        has_native_map_presentation: true,
        inventory_slots: vec![0xff; 8],
        native_inventory_flags: vec![0; 8],
        ..Default::default()
    }
}

/// Go `nativeAIRuntimeCostRows`。
pub(crate) fn native_ai_runtime_cost_rows() -> Vec<Vec<u8>> {
    vec![vec![1u8; NATIVE_MOVEMENT_COST_ROW_SIZE as usize]; NATIVE_MOVEMENT_COST_ROW_COUNT as usize]
}

/// Go `nativeAIActionCommandBook`。
fn native_ai_action_command_book() -> Vec<NativeCommandRecord> {
    let mut book = native_command_book_ids();
    // Synthetic raw tuple: the command producer can reach an adjacent target
    // from the selected destination, while its score remains visibly positive.
    book[0] = NativeCommandRecord {
        id: 0,
        damage: 50,
        hit: 90,
        selection_mode: 5,
        effect_mode: 1,
        mp_cost: 2,
        target_code: 0,
    };
    book
}

fn line_state(w: i64, h: i64, units: Vec<Unit>) -> State {
    let n = (w * h) as usize;
    State {
        w,
        h,
        units,
        native_composition_event_bytes: vec![0; n],
        native_terrain_move_codes: vec![0; n],
        ..Default::default()
    }
}

fn own(mut u: Unit) -> Unit {
    u.camp = OWN;
    u
}

/// Go：`TestNextAIPlanUsesVerifiedMode2PhysicalCandidate`
#[test]
fn next_ai_plan_uses_verified_mode2_physical_candidate() {
    let mut actor = native_ai_runtime_unit(0, 0, 1, 2);
    actor.inventory_slots[0] = 1;
    actor.native_inventory_flags[0] = 0x40;
    // Physical selector one targets the opposite raw +6 group (zero); Camp is
    // intentionally not used as a substitute for that provenance.
    let mut target = own(native_ai_runtime_unit(2, 0, 0, 0));
    (target.ap, target.dp) = (1, 1);
    let mut state = line_state(3, 1, vec![actor, target]);
    state.bind_native_future_item_rows(&[0u8; 2 * ROW]).unwrap();
    state.native_future_item_rows[ROW + 0x0b] = 0;
    state.native_future_item_rows[ROW + 0x0c] = 1;
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    let plan = state.next_ai_plan().expect("mode-2 plan is nil");
    assert!(plan.native_error.is_none(), "mode-2 plan failed: {:?}", plan.native_error);
    assert!(plan.native_mode2_physical, "plan did not retain native mode-2 provenance: {plan:?}");
    assert!(plan.target == Some(1), "plan={plan:?} target={:?} want 1", plan.target);
    assert!(
        plan.path.len() >= 2 && plan.path[0] == Cell { x: 0, y: 0 } && plan.path.last() == Some(&Cell { x: 1, y: 0 }),
        "path={:?} want (0,0) to selected destination (1,0)",
        plan.path
    );
}

/// Go：`TestNextAIPlanPhysicalHelperUsesFullTargetItemTable`
#[test]
fn next_ai_plan_physical_helper_uses_full_target_item_table() {
    let mut actor = native_ai_runtime_unit(0, 0, 1, 2);
    actor.inventory_slots[0] = 1;
    actor.native_inventory_flags[0] = 0x40;
    let mut target = own(native_ai_runtime_unit(2, 0, 0, 0));
    (target.ap, target.dp) = (1, 1);
    target.inventory_slots[0] = 0x1f;
    target.native_inventory_flags[0] = 0x40;
    let mut state = line_state(3, 1, vec![actor, target]);
    let mut rows = vec![0u8; 0x20 * ROW];
    rows[ROW + 0x0c] = 1;
    rows[0x1f * ROW + 0x0b] = 1;
    state.bind_native_future_item_rows(&rows).unwrap();
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    let plan = state.next_ai_plan().expect("mode-2 plan is nil");
    assert!(plan.native_error.is_none(), "mode-2 target item 0x1f plan error: {:?}", plan.native_error);
    assert!(plan.target == Some(1), "plan target={:?} want 1", plan.target);
}

/// Go：`TestNextAIPlanMode11BuildsOrderedDirectStages`
#[test]
fn next_ai_plan_mode11_builds_ordered_direct_stages() {
    let mut actor = native_ai_runtime_unit(0, 0, 1, 11);
    actor.native_command_mask[0] = 1;
    actor.native_inventory_flags = vec![0x80; 8];
    actor.mp = 255;
    let mut target = own(native_ai_runtime_unit(2, 0, 0, 0));
    target.native_inventory_flags = vec![0x80; 8];
    let mut state = line_state(3, 1, vec![actor, target]);
    state.native_command_book = native_ai_action_command_book();
    state.bind_native_future_item_rows(&[0u8; 2 * ROW]).unwrap();
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    let plan = state.next_ai_plan().expect("mode11 plan is nil");
    assert!(plan.native_error.is_none(), "mode11 plan={plan:?} err={:?}", plan.native_error);
    let stages = &plan.native_mode11_stages;
    assert!(stages.len() == 2, "mode11 stages={stages:?}, want two direct stages");
    assert!(
        stages[0].stage == NativeAIMode11Stage { ordinal: 1, route: NATIVE_AI_MODE11_CALL15311 },
        "first mode11 stage={:?}",
        stages[0].stage
    );
    let action = stages[0].action.as_ref();
    assert!(
        action.is_some_and(|a| a.native_action_kind == NATIVE_AI_ACTION_COMMAND),
        "first mode11 action={action:?}, want command owner"
    );
    let path = &action.unwrap().path;
    assert!(
        path.len() == 1 && path[0] == Cell { x: 0, y: 0 },
        "first mode11 command path={path:?}, want stationary actor origin"
    );
    assert!(stages[1].stage.ordinal == 2, "second mode11 stage={:?}", stages[1].stage);
}

/// Go：`TestNextAIPlanMode2FailsClosedWithoutMovementRows`
#[test]
fn next_ai_plan_mode2_fails_closed_without_movement_rows() {
    let mut state = line_state(1, 1, vec![native_ai_runtime_unit(0, 0, 0, 2)]);
    let plan = state.next_ai_plan();
    assert!(plan.as_ref().is_some_and(|p| p.native_error.is_some()), "plan={plan:?} want fail-closed native error");
}

/// Go：`TestNextAIPlanMode2WithoutEquippedLowItemTakesIdleRecoveryTail`
///
/// 0x1428B 0x1B83D(actor,0) 回 -1 → 0x14296 je 0x145C3 → xor eax,eax：0x14237 回傳 0，
/// 與沒有物理候選同一條 0x13C06→0x13FD4 收尾，不是失敗即關閉。
#[test]
fn next_ai_plan_mode2_without_equipped_low_item_takes_idle_recovery_tail() {
    let mut state = line_state(1, 1, vec![native_ai_runtime_unit(0, 0, 0, 2)]);
    state.bind_native_future_item_rows(&[0u8; 2 * ROW]).unwrap();
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    let plan = state.next_ai_plan();
    assert!(
        plan.as_ref().is_some_and(|p| p.native_error.is_none()
            && p.native_mode2_physical
            && p.native_mode_fallback_active
            && p.native_mode_fallback == 2
            && p.target.is_none()
            && p.path.is_empty()),
        "plan={plan:?} want 0x13C06 idle-recovery tail"
    );
}

/// Go：`TestNextAIPlanMode2NoPhysicalCandidateUsesAccepted13FD4Decision`
#[test]
fn next_ai_plan_mode2_no_physical_candidate_uses_accepted13_fd4_decision() {
    let mut actor = native_ai_runtime_unit(0, 0, 0, 2);
    actor.hp = 10;
    actor.inventory_slots[0] = 1;
    actor.native_inventory_flags[0] = 0x40;
    let mut state = line_state(1, 1, vec![actor]);
    state.bind_native_future_item_rows(&[0u8; 2 * ROW]).unwrap();
    state.native_future_item_rows[ROW + 0x0b] = 0;
    state.native_future_item_rows[ROW + 0x0c] = 1;
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    let plan = state.next_ai_plan();
    let recovery = plan.as_ref().and_then(|p| p.native_idle_recovery);
    assert!(
        plan.as_ref().is_some_and(|p| p.native_error.is_none()
            && p.native_mode_fallback_active
            && p.native_mode_fallback == 2
            && p.native_mode2_physical)
            && recovery.is_some(),
        "plan={plan:?} want accepted mode-2 0x13fd4 fallback"
    );
    let r = recovery.unwrap();
    assert!(r.current_hp == 10 && r.maximum_hp == 20 && r.next_hp == 14, "recovery={r:?} want 10/20 -> 14");
}

/// Go：`TestNextAIPlanMode2NoPhysicalCandidateCompletesRejected13FD4Tail`
#[test]
fn next_ai_plan_mode2_no_physical_candidate_completes_rejected13_fd4_tail() {
    let mut actor = native_ai_runtime_unit(0, 0, 0, 2);
    actor.inventory_slots[0] = 1;
    actor.native_inventory_flags[0] = 0x40;
    let mut state = line_state(1, 1, vec![actor]);
    state.bind_native_future_item_rows(&[0u8; 2 * ROW]).unwrap();
    state.native_future_item_rows[ROW + 0x0c] = 1;
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    let plan = state.next_ai_plan();
    assert!(
        plan.as_ref().is_some_and(|p| p.native_error.is_none()
            && p.native_mode_fallback_active
            && p.native_mode_fallback == 2
            && p.native_idle_recovery.is_none()
            && p.target.is_none()),
        "plan={plan:?} want rejected 0x13fd4 common-tail plan"
    );
}

/// Go：`TestNextAIPlanMode2RoutesFailed14EF0ProducerTo13FD4`
#[test]
fn next_ai_plan_mode2_routes_failed14_ef0_producer_to13_fd4() {
    let mut actor = native_ai_runtime_unit(0, 0, 0, 2);
    actor.hp = 10;
    actor.inventory_slots[0] = 1;
    actor.native_inventory_flags[0] = 0x40;
    let mut state = line_state(1, 1, vec![actor]);
    state.native_command_book = vec![NativeCommandRecord::default(); NATIVE_COMMAND_RECORD_COUNT as usize];
    state.bind_native_future_item_rows(&[0u8; 2 * ROW]).unwrap();
    state.native_future_item_rows[ROW + 0x0c] = 1;
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    let plan = state.next_ai_plan();
    assert!(
        plan.as_ref().is_some_and(|p| p.native_error.is_none()
            && p.native_mode_fallback_active
            && p.native_mode_fallback == 2
            && p.native_mode2_physical
            && p.native_idle_recovery.is_some()),
        "plan={plan:?} want failed 0x14ef0 producer routed to mode-2 0x13fd4"
    );
}

/// Go：`TestNativeAIPhysicalHelper1DEBEKeepsRawPredicateBoundaries`
#[test]
fn native_ai_physical_helper1_debe_keeps_raw_predicate_boundaries() {
    let mut target = vec![0u8; RS];
    (target[0], target[1]) = (1, 0);
    target[0x26] = 0;
    target[0x0a] = 0x40;
    target[0x0b] = 1;
    let mut item_rows = vec![0u8; 2 * ROW];
    item_rows[ROW + 0x0b] = 1;

    let got = native_ai_physical_helper1_debe(&target, Cell { x: 0, y: 0 }, &item_rows);
    assert!(matches!(got, Ok(1)), "verified adjacent raw predicate = {got:?}; want 1");

    target[0x26] = 1;
    let got = native_ai_physical_helper1_debe(&target, Cell { x: 0, y: 0 }, &item_rows);
    assert!(matches!(got, Ok(-1)), "nonzero target +0x26 predicate = {got:?}; want -1");
    target[0x26] = 0;

    let got = native_ai_physical_helper1_debe(&target, Cell { x: 0, y: 2 }, &item_rows);
    assert!(matches!(got, Ok(-1)), "non-adjacent raw predicate = {got:?}; want -1");

    item_rows[ROW + 0x0b] = 2;
    let got = native_ai_physical_helper1_debe(&target, Cell { x: 0, y: 0 }, &item_rows);
    assert!(matches!(got, Ok(-1)), "raw item geometry >1 predicate = {got:?}; want -1");
}

/// Go：`TestNextAIPlanUses14EF0CommandWinnerAndRetainsRawTarget`
#[test]
fn next_ai_plan_uses14_ef0_command_winner_and_retains_raw_target() {
    let mut actor = native_ai_runtime_unit(0, 0, 1, 2);
    actor.native_command_mask[0] = 1;
    actor.native_inventory_flags = vec![0x80; 8];
    let mut target = own(native_ai_runtime_unit(2, 0, 0, 0));
    target.native_inventory_flags = vec![0x80; 8];
    let mut state = line_state(3, 1, vec![actor, target]);
    state.native_command_book = native_ai_action_command_book();
    state.bind_native_future_item_rows(&[0u8; 2 * ROW]).unwrap();
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    let plan = state.next_ai_plan().expect("command plan is nil");
    assert!(plan.native_error.is_none(), "command plan={plan:?} err={:?}", plan.native_error);
    assert!(
        plan.native_action_kind == NATIVE_AI_ACTION_COMMAND
            && plan.native_command_id == 0
            && plan.native_ai14_ef0_route == NATIVE_AI14_EF0_CALL15311
            && plan.target == Some(1),
        "command plan={plan:?}"
    );
    assert!(
        plan.path.len() == 1
            && plan.path[0] == Cell { x: 0, y: 0 }
            && plan.native_action_destination == Cell { x: 1, y: 0 },
        "command path={:?} destination={:?} want stationary origin and effect (1,0)",
        plan.path,
        plan.native_action_destination
    );
}

/// Go：`TestNextAIPlanUses14EF0ItemWinnerAndRetainsRawTarget`
#[test]
fn next_ai_plan_uses14_ef0_item_winner_and_retains_raw_target() {
    let mut actor = native_ai_runtime_unit(0, 0, 0, 2);
    actor.native_inventory_flags = vec![0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80];
    actor.inventory_slots[0] = 1;
    let mut target = own(native_ai_runtime_unit(2, 0, 1, 0));
    target.hp = 1;
    target.native_inventory_flags = vec![0x80; 8];
    let mut state = line_state(3, 1, vec![actor, target]);
    state.native_command_book = native_ai_action_command_book();
    state.native_command_book[0].mp_cost = 99; // make the command unavailable
    let mut rows = vec![0u8; 2 * ROW];
    rows[ROW + 0x0d] = 5; // one of the recovered positive score families
    rows[ROW + 0x10] = 5; // destination selection mode
    rows[ROW + 0x11] = 0; // selector zero flips raw target code to 1
    rows[ROW + 0x12] = 1; // adjacent target stage
    state.bind_native_future_item_rows(&rows).unwrap();
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    let plan = state.next_ai_plan().expect("item plan is nil");
    assert!(plan.native_error.is_none(), "item plan={plan:?} err={:?}", plan.native_error);
    assert!(
        plan.native_action_kind == NATIVE_AI_ACTION_ITEM
            && plan.native_item_slot == 0
            && plan.native_item_id == 1
            && plan.native_ai14_ef0_route == NATIVE_AI14_EF0_CALL15055
            && plan.target == Some(1),
        "item plan={plan:?}"
    );
    assert!(
        plan.native_item_target_indices == [1],
        "item target list={:?} want detached raw [1]",
        plan.native_item_target_indices
    );
    // 0x15055 不呼叫 0x14B78：原地使用，(1,0) 只是效果格。
    assert!(
        plan.path.len() == 1
            && plan.path[0] == Cell { x: 0, y: 0 }
            && plan.native_action_destination == Cell { x: 1, y: 0 },
        "item path={:?} destination={:?} want stationary origin and effect (1,0)",
        plan.path,
        plan.native_action_destination
    );
}

/// Go：`TestNextAIPlanRejectsRawModeWithout14EF0Tables`
#[test]
fn next_ai_plan_rejects_raw_mode_without14_ef0_tables() {
    let actor = native_ai_runtime_unit(0, 0, 0, 1);
    let target = own(native_ai_runtime_unit(1, 0, 1, 0));
    let mut state = State { w: 2, h: 1, units: vec![actor, target], ..Default::default() };
    let plan = state.next_ai_plan();
    assert!(plan.as_ref().is_some_and(|p| p.native_error.is_some()), "plan={plan:?} want fail-closed raw mode error");
}

fn single_mode_plan(actor: Unit, w: i64) -> Option<AIPlan> {
    let mut state = line_state(w, 1, vec![actor]);
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    state.next_ai_plan()
}

/// Go：`TestNextAIPlanUsesMode4RawDestinationFallback`
#[test]
fn next_ai_plan_uses_mode4_raw_destination_fallback() {
    let mut actor = native_ai_runtime_unit(0, 0, 1, 4);
    (actor.native_record_byte35, actor.native_record_byte36) = (1, 0);
    let plan = single_mode_plan(actor, 3);
    let plan = match plan {
        Some(p) if p.native_error.is_none() => p,
        other => panic!("mode4 plan={other:?}"),
    };
    assert!(
        plan.native_mode_fallback == 4
            && plan.target.is_none()
            && plan.native_action_destination == Cell { x: 1, y: 0 },
        "mode4 raw fallback plan={plan:?}"
    );
    assert!(plan.path.len() == 2 && plan.path[1] == Cell { x: 1, y: 0 }, "mode4 path={:?}", plan.path);
}

/// Go：`TestNextAIPlanMode4AtDestinationFocusesActorAndTakesIdleRecovery`
///
/// 0x13BE1..0x13C0F：mode 4 無條件 0x12D7B 聚焦自己，0x14B78 沒走成回 0 → 0x13FD4。
/// 第十章 r2 每回合 record 12／13 已在 (8,10)／(22,10) 仍各聚焦一次。
#[test]
fn next_ai_plan_mode4_at_destination_focuses_actor_and_takes_idle_recovery() {
    let mut actor = native_ai_runtime_unit(1, 0, 1, 4);
    (actor.native_record_byte35, actor.native_record_byte36) = (1, 0);
    let plan = single_mode_plan(actor, 3);
    let plan = match plan {
        Some(p) if p.native_error.is_none() && p.native_mode_fallback == 4 => p,
        other => panic!("mode4 plan={other:?}"),
    };
    assert!(
        plan.path.len() <= 1
            && plan.native_mode_focus_actor
            && plan.native_mode_write_range_zero
            && plan.target.is_none(),
        "mode4 at destination plan={plan:?}"
    );
}

/// Go：`TestNextAIPlanMode7RetainsRawByte5ArrivalWrite`
#[test]
fn next_ai_plan_mode7_retains_raw_byte5_arrival_write() {
    let mut actor = native_ai_runtime_unit(0, 0, 1, 7);
    (actor.native_record_byte35, actor.native_record_byte36) = (1, 0);
    let plan = single_mode_plan(actor, 3);
    assert!(
        plan.as_ref()
            .is_some_and(|p| p.native_error.is_none() && p.native_mode_fallback == 7 && p.native_mode_write_byte5),
        "mode7 plan={plan:?}"
    );
}

/// Go：`TestNextAIPlanMode0UsesRawBlockedCoordinateBeforeNearestFallback`
#[test]
fn next_ai_plan_mode0_uses_raw_blocked_coordinate_before_nearest_fallback() {
    let actor = native_ai_runtime_unit(0, 0, 1, 0);
    let target = own(native_ai_runtime_unit(2, 0, 0, 0));
    let mut state = line_state(3, 1, vec![actor, target]);
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    let plan = state.next_ai_plan();
    let plan = match plan {
        Some(p) if p.native_error.is_none() && p.native_mode_fallback == 0 && p.target.is_none() => p,
        other => panic!("mode0 plan={other:?}"),
    };
    assert!(plan.path.len() == 2 && plan.path[1] == Cell { x: 1, y: 0 }, "mode0 path={:?}", plan.path);
}

/// Go `mustNativeAIModeBlockedSearchFlags`。
fn must_native_ai_mode_blocked_search_flags(state: &State, selector: i64) -> Vec<u8> {
    let records = native_ai_scoring_records(&state.units).unwrap();
    native_ai_mode_blocked_search_flags(
        state.w,
        state.h,
        &records,
        state.units.len() as i64,
        selector,
        &state.native_composition_event_bytes,
    )
    .unwrap()
}

/// Go：`TestNextAIPlanMode0FallsBackToNearestWhenBlockedPlanStays`
///
/// 0x14121 只在 0x14B78 真的走了才回 1；阻擋格的規劃落在原格時，0x13A9F 的 mode 0
/// 分派（0x13B0F）接著走 0x13E9C 以最近的對立單位再規劃。第七章 r6 第 7 回合記錄 23
/// 的形狀：mode 2 搜尋最後接受的阻擋格是遠處的 T2，往它的長路徑第一步就被同組單位
/// 佔住的零預算格截住；最近的 T1 才走得到。
#[test]
fn next_ai_plan_mode0_falls_back_to_nearest_when_blocked_plan_stays() {
    let mut actor = native_ai_runtime_unit(0, 0, 1, 0);
    actor.mv = 1;
    let mut ally = native_ai_runtime_unit(0, 1, 1, 0);
    ally.mv = 1;
    let near = own(native_ai_runtime_unit(2, 0, 0, 0));
    let far = own(native_ai_runtime_unit(0, 3, 0, 0));
    let mut state = line_state(4, 4, vec![actor, ally, near, far]);
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    let got = native_path_blocked_coordinate(
        4,
        4,
        Cell::default(),
        28,
        &must_native_ai_mode_blocked_search_flags(&state, 1),
        &state.native_terrain_move_codes,
        &native_ai_runtime_cost_rows()[0],
    );
    assert!(matches!(got, Ok((c, true)) if c == Cell { x: 0, y: 3 }), "0x14121 blocked cell={got:?} want (0,3)");
    let plan = state.next_ai_plan();
    let plan = match plan {
        Some(p) if p.native_error.is_none() && p.native_mode_fallback == 0 && p.target.is_none() => p,
        other => panic!("mode0 plan={other:?}"),
    };
    assert!(
        plan.path.len() == 2 && plan.path[1] == Cell { x: 1, y: 0 },
        "mode0 path={:?} want the 0x13E9C route toward (2,0) stopping at (1,0)",
        plan.path
    );
    assert!(
        plan.native_mode_blocked_found && plan.native_mode_blocked_cell == Cell { x: 0, y: 3 },
        "mode0 plan should record the abandoned 0x14121 cell: {plan:?}"
    );
}

/// Go：`TestNextAIPlanMode0StuckFallbackCarriesIdleRecovery`
///
/// 0x13E9C 聚焦後 0x14B78 沒走成就回 0，0x13B21 jmp 0x13C06 → 0x13FD4：第八章 r1 第 7 回合
/// 記錄 36 被圍住，原版 HP 7→43（max 180 的五分之一）。計畫不移動，但要帶回復決策。
#[test]
fn next_ai_plan_mode0_stuck_fallback_carries_idle_recovery() {
    let mut actor = native_ai_runtime_unit(0, 0, 1, 0);
    actor.mv = 1;
    actor.hp = 7;
    actor.max_hp = 180;
    let blocker = native_ai_runtime_unit(1, 0, 1, 0);
    let target = own(native_ai_runtime_unit(4, 0, 0, 0));
    let mut state = line_state(5, 1, vec![actor, blocker, target]);
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    let plan = state.next_ai_plan();
    let plan = match plan {
        Some(p) if p.native_error.is_none() && p.native_mode_fallback == 0 && p.target.is_none() => p,
        other => panic!("mode0 plan={other:?}"),
    };
    let moved = plan.path.len() > 1;
    let recovery = plan.native_idle_recovery.or(plan.native_fallback_idle_recovery);
    assert!(
        !moved && recovery.is_some(),
        "被圍住的 mode 0 應該不動並帶 0x13FD4 回復：path={:?} recovery={recovery:?}",
        plan.path
    );
}

/// Go：`TestNextAIPlanMode3Uses12C60RawRecord8Lookup`
#[test]
fn next_ai_plan_mode3_uses12_c60_raw_record8_lookup() {
    let mut actor = native_ai_runtime_unit(0, 0, 1, 3);
    actor.native_record_byte35 = 7;
    let mut target = own(native_ai_runtime_unit(2, 0, 0, 0));
    target.native_record_byte8 = 7;
    let mut state = line_state(3, 1, vec![actor, target]);
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    let plan = state.next_ai_plan();
    let plan = match plan {
        Some(p)
            if p.native_error.is_none()
                && p.native_mode_fallback_active
                && p.native_mode_fallback == 3
                && p.native_mode_write_range_zero =>
        {
            p
        }
        other => panic!("mode3 plan={other:?}"),
    };
    assert!(
        plan.path.len() == 2 && plan.path[1] == Cell { x: 1, y: 0 } && plan.target.is_none(),
        "mode3 path={:?} target={:?}",
        plan.path,
        plan.target
    );
}

/// Go：`TestNextAIPlanMode9Uses12C60RawRecord8Lookup`
#[test]
fn next_ai_plan_mode9_uses12_c60_raw_record8_lookup() {
    let mut actor = native_ai_runtime_unit(0, 0, 1, 9);
    actor.native_record_byte35 = 7;
    let mut target = own(native_ai_runtime_unit(2, 0, 0, 0));
    target.native_record_byte8 = 7;
    let mut state = line_state(3, 1, vec![actor, target]);
    state.bind_native_movement_cost_rows(&native_ai_runtime_cost_rows()).unwrap();
    let plan = state.next_ai_plan();
    let plan = match plan {
        Some(p) if p.native_error.is_none() && p.native_mode_fallback_active && p.native_mode_fallback == 9 => p,
        other => panic!("mode9 plan={other:?}"),
    };
    assert!(
        !plan.native_mode_write_range_zero && plan.path.len() == 2 && plan.path[1] == Cell { x: 1, y: 0 },
        "mode9 path/range={:?}/{}",
        plan.path,
        plan.native_mode_write_range_zero
    );
}
