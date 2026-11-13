//! 移植自 `remake/internal/battle/native_command20_ai_plan_test.go`。

use super::*;

/// Go `completeNativeAIScoringUnit`（`native_ai_record_test.go`；A 組的 Rust 版是私有測試輔助，這裡複製一份）。
fn complete_native_ai_scoring_unit() -> Unit {
    Unit {
        x: 99,
        y: 98,
        native_map_presentation: NativeMapPresentationState { x: 7, y: 8, pose: 3, motion: 4 },
        has_native_map_presentation: true,
        battle_fig: 9,
        has_battle_fig: true,
        native_identity: 10,
        has_native_identity: true,
        native_record_byte8: 0x60,
        has_native_record_byte8: true,
        native_record_race: 11,
        has_native_record_race: true,
        native_record_class: 12,
        has_native_record_class: true,
        native_record_byte5: 0x20,
        has_native_record_byte5: true,
        native_record_byte6: 1,
        has_native_record_byte6: true,
        native_record_byte34: 0x81,
        has_native_record_byte34: true,
        native_record_byte35: 13,
        has_native_record_byte35: true,
        native_record_byte36: 14,
        has_native_record_byte36: true,
        native_record_word42: 234,
        has_native_record_word42: true,
        native_record_word46: 18,
        has_native_record_word46: true,
        inventory_slots: vec![1, 2, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0x40, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
        native_command_mask: [1, 2, 3, 4, 5],
        native_transient: [6, 7, 8, 9, 10, 0],
        lv: 15,
        mv: 6,
        hp: 123,
        max_hp: 234,
        mp: 17,
        max_mp: 18,
        ap: 19,
        dp: 20,
        hit: 21,
        ev: 22,
        ..Default::default()
    }
}

fn native_ai2022_book(id: i64) -> Vec<NativeCommandRecord> {
    let mut book: Vec<NativeCommandRecord> =
        (0..NATIVE_COMMAND_RECORD_COUNT).map(|i| NativeCommandRecord { id: i, ..Default::default() }).collect();
    book[10] = NativeCommandRecord { id: 10, damage: 10, ..Default::default() };
    book[id as usize] = NativeCommandRecord { id, effect_mode: 1, mp_cost: 2, target_code: 1, ..Default::default() };
    book
}

const ACTOR: UnitIdx = 0;
const TARGET: UnitIdx = 1;

/// Go `nativeAI2022State`：actor、target 是 `st.units` 的 0、1。
fn native_ai2022_state(id: i64) -> State {
    let mut actor = complete_native_ai_scoring_unit();
    (actor.camp, actor.on_field, actor.mp, actor.native_record_byte6) = (ENEMY, true, 8, 1);
    (actor.x, actor.y) = (0, 0);
    (actor.native_map_presentation.x, actor.native_map_presentation.y) = (0, 0);
    let mut target = complete_native_ai_scoring_unit();
    (target.camp, target.on_field, target.hp, target.max_hp) = (ENEMY, true, 20, 100);
    (target.x, target.y) = (1, 0);
    (target.native_map_presentation.x, target.native_map_presentation.y) = (1, 0);
    State {
        w: 2,
        h: 1,
        units: vec![actor, target],
        native_composition_event_bytes: vec![0; 2],
        native_command_book: native_ai2022_book(id),
        ..Default::default()
    }
}

fn set_record(st: &mut State, id: usize, selection_mode: i64, effect_mode: i64, target_code: i64) {
    let rec = &mut st.native_command_book[id];
    (rec.selection_mode, rec.effect_mode, rec.target_code) = (selection_mode, effect_mode, target_code);
}

/// Go：`TestPlanNativeAICommand20UsesRawSelectorAndRollback`
#[test]
fn plan_native_ai_command20_uses_raw_selector_and_rollback() {
    let mut st = native_ai2022_state(20);
    st.units[TARGET].native_transient[3] = 3;
    let plan = st.plan_native_ai_command2022(Some(ACTOR), Cell::default(), 20, 0);
    assert!(
        matches!(&plan, Ok(p) if p.targets.len() == 2 && p.results.len() == 2 && p.results[1].restore.is_some()),
        "plan={plan:?}"
    );
    let mut plan = plan.unwrap();
    assert!(
        !(st.units[TARGET].hp != 20 || st.units[TARGET].native_transient[3] != 3 || st.units[ACTOR].mp != 8),
        "planning mutated live state"
    );
    publish_native_ai_command2022(&mut st.units, Some(&mut plan)).unwrap();
    let (actor, target) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(
        !(target.native_transient[3] != 0 || actor.mp != 6 || actor.acted),
        "publish actor={actor:?} target={target:?}"
    );
    abort_native_ai_command2022(&mut st.units, Some(&mut plan)).unwrap();
    assert!(
        !(st.units[TARGET].hp != 20 || st.units[TARGET].native_transient[3] != 3 || st.units[ACTOR].mp != 8),
        "rollback failed"
    );
}

/// Go：`TestPlanNativeAICommand22UsesNativeApplicationAndCompletes`
#[test]
fn plan_native_ai_command22_uses_native_application_and_completes() {
    let mut st = native_ai2022_state(22);
    let plan = st.plan_native_ai_command2022(Some(ACTOR), Cell::default(), 22, 0);
    assert!(matches!(&plan, Ok(p) if !p.results.is_empty() && p.results[0].apply.is_some()), "plan={plan:?}");
    let mut plan = plan.unwrap();
    publish_native_ai_command2022(&mut st.units, Some(&mut plan)).unwrap();
    complete_native_ai_command2022(&mut st.units, Some(&mut plan)).unwrap();
    let actor = &st.units[ACTOR];
    assert!(actor.mp == 6 && actor.acted, "actor={actor:?}");
    let applied = plan.results.iter().any(|result| result.apply.is_some_and(|a| a.applied));
    let target = &st.units[TARGET];
    assert!(applied && target.hp <= 20, "application did not publish plan={plan:?} target={target:?}");
}

/// Go：`TestPlanNativePlayerCommand20UsesConfirmedCursorTargets`
#[test]
fn plan_native_player_command20_uses_confirmed_cursor_targets() {
    let mut st = native_ai2022_state(20);
    set_record(&mut st, 20, 1, 0, 1);
    st.units[ACTOR].camp = OWN;
    st.units[TARGET].camp = OWN;
    st.units[TARGET].native_transient[3] = 3;
    let mut plan = st.plan_native_player_command2022(Some(ACTOR), Some(TARGET), 20, 0).unwrap();
    assert!(
        !(plan.targets.is_empty() || plan.targets[0] != TARGET || plan.results.is_empty()),
        "unexpected player targets/results: {plan:?}"
    );
    assert!(
        !(st.units[TARGET].native_transient[3] != 3 || st.units[ACTOR].mp != 8),
        "player planning mutated live state"
    );
    publish_native_ai_command2022(&mut st.units, Some(&mut plan)).unwrap();
    let (actor, target) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(
        !(target.native_transient[3] != 0 || actor.mp != 6 || actor.acted),
        "publish actor={actor:?} target={target:?}"
    );
}

/// Go：`TestPlanNativePlayerCommand22UsesConfirmedCursorTargets`
#[test]
fn plan_native_player_command22_uses_confirmed_cursor_targets() {
    let mut st = native_ai2022_state(22);
    set_record(&mut st, 22, 1, 0, 0);
    st.units[ACTOR].camp = OWN;
    st.units[TARGET].camp = ENEMY;
    let mut plan = st.plan_native_player_command2022(Some(ACTOR), Some(TARGET), 22, 0).unwrap();
    assert!(
        !(plan.targets.is_empty()
            || plan.targets[0] != TARGET
            || plan.results.is_empty()
            || plan.results[0].apply.is_none()),
        "unexpected player targets/results: {plan:?}"
    );
    assert!(!(st.units[TARGET].hp != 20 || st.units[ACTOR].mp != 8), "player planning mutated live state");
    publish_native_ai_command2022(&mut st.units, Some(&mut plan)).unwrap();
    let actor = &st.units[ACTOR];
    assert!(!(actor.mp != 6 || actor.acted), "publish actor={actor:?}");
}

/// Go：`TestPlanNativePlayerCommand25ClearsOnlyRawByte5AndRollsBack`
#[test]
fn plan_native_player_command25_clears_only_raw_byte5_and_rolls_back() {
    let mut st = native_ai2022_state(25);
    set_record(&mut st, 25, 1, 0, 1);
    (st.units[ACTOR].camp, st.units[TARGET].camp) = (OWN, OWN);
    (st.units[TARGET].native_record_byte5, st.units[TARGET].has_native_record_byte5) = (0x80, true);
    let plan = st.plan_native_player_command2022(Some(ACTOR), Some(TARGET), 25, 0x1234);
    assert!(
        matches!(&plan, Ok(p) if p.results.len() == 1 && p.results[0].command25 && p.results[0].cleared_byte5),
        "plan={plan:?}"
    );
    let mut plan = plan.unwrap();
    let (actor, target) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(
        !(target.native_record_byte5 != 0x80 || actor.mp != 8 || actor.acted),
        "command25 planning mutated live state"
    );
    publish_native_ai_command2022(&mut st.units, Some(&mut plan)).unwrap();
    let (actor, target) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(
        !(target.native_record_byte5 != 0 || actor.mp != 6 || actor.acted || plan.rng_state != 0x1234),
        "published actor={actor:?} target={target:?} plan={plan:?}"
    );
    abort_native_ai_command2022(&mut st.units, Some(&mut plan)).unwrap();
    let (actor, target) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(!(target.native_record_byte5 != 0x80 || actor.mp != 8 || actor.acted), "command25 rollback failed");
}

/// Go：`TestPlanNativeAICommand26UsesRawSelectorApplication`
#[test]
fn plan_native_ai_command26_uses_raw_selector_application() {
    let mut st = native_ai2022_state(26);
    (st.units[ACTOR].native_transient[3], st.units[TARGET].native_transient[3]) = (0, 0);
    let plan = st.plan_native_ai_command2022(Some(ACTOR), Cell::default(), 26, 0);
    assert!(matches!(&plan, Ok(p) if !p.results.is_empty()), "plan={plan:?}");
    let plan = plan.unwrap();
    let found = plan
        .results
        .iter()
        .any(|result| result.target == Some(TARGET) && result.apply.is_some() && result.offset == 0x25);
    let (actor, target) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(
        found && target.native_transient[3] == 0 && target.hp == 20 && actor.mp == 8,
        "AI26 plan did not preserve raw-selector transaction: {plan:?}"
    );
}
