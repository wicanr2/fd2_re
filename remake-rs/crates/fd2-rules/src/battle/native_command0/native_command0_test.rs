//! 移植自 `remake/internal/battle/native_command0_test.go`。
//!
//! Go 的 `NativeCommandDamageResult` 內嵌 `NativeCommandDamage`（`results[0].Hit`）；Rust 版是具名欄位
//! `native_command_damage`。

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

fn sequential_book() -> Vec<NativeCommandRecord> {
    (0..NATIVE_COMMAND_RECORD_COUNT).map(|id| NativeCommandRecord { id, ..Default::default() }).collect()
}

const ACTOR: UnitIdx = 0;
const TARGET: UnitIdx = 1;

/// Go `nativeCommand0TestState`：actor、target 是 `st.units` 的 0、1。
fn native_command0_test_state() -> State {
    let mut book = sequential_book();
    book[0] = NativeCommandRecord {
        id: 0,
        damage: 50,
        hit: 100,
        selection_mode: 1,
        effect_mode: 0,
        mp_cost: 2,
        target_code: 0,
    };
    State {
        w: 2,
        h: 1,
        units: vec![
            Unit { camp: OWN, x: 0, y: 0, hp: 20, mp: 3, on_field: true, ..Default::default() },
            Unit { camp: ENEMY, class_id: 5, x: 1, y: 0, hp: 100, on_field: true, ..Default::default() },
        ],
        native_composition_event_bytes: vec![0; 2],
        native_command_book: book,
        ..Default::default()
    }
}

/// Go：`TestNativeCommandDamageUsesRawClassBeforeAuthoredClass`
#[test]
fn native_command_damage_uses_raw_class_before_authored_class() {
    let mut st = native_command0_test_state();
    (st.units[ACTOR].camp, st.units[ACTOR].mp) = (ALLY, 188);
    let target = &mut st.units[TARGET];
    (target.class_id, target.native_record_class, target.has_native_record_class) = (1, 26, true);
    target.hp = 266;
    st.native_command_book[8] = NativeCommandRecord {
        id: 8,
        damage: 440,
        hit: 100,
        selection_mode: 1,
        effect_mode: 0,
        mp_cost: 24,
        target_code: 0,
    };
    let mut resistances = BTreeMap::from([(1, 10), (26, 4)]);
    let plan = st.plan_native_command_damage(Some(ACTOR), Some(TARGET), 8, &resistances, 33548);
    assert!(
        matches!(&plan, Ok(p) if p.results.len() == 1 && p.results[0].native_command_damage.damage == 174
            && p.results[0].hp_after == 92),
        "record+0x20 抗性未生效：plan={plan:?}"
    );
    let (a, t) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(!(a.mp != 188 || t.hp != 266 || a.acted), "預建傷害計畫不得發布交易");
    resistances.remove(&26);
    let err = st.plan_native_command_damage(Some(ACTOR), Some(TARGET), 8, &resistances, 33548);
    let (a, t) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(err.is_err() && a.mp == 188 && t.hp == 266, "原生職業缺抗性時不得回退或發布交易");
}

/// Go：`TestExecuteBoundNativeCommand0UsesTwoStageTargetsAndOneMPDebit`
#[test]
fn execute_bound_native_command0_uses_two_stage_targets_and_one_mp_debit() {
    // The executor requires the complete verified book rather than an invented
    // partial record; fill unused rows with exact sequential IDs for this unit
    // test, which only dispatches ID 0.
    let mut book = sequential_book();
    book[0] = NativeCommandRecord {
        id: 0,
        damage: 50,
        hit: 100,
        selection_mode: 1,
        effect_mode: 0,
        mp_cost: 2,
        target_code: 0,
    };
    let mut st = State {
        w: 3,
        h: 1,
        units: vec![
            Unit { camp: OWN, x: 0, y: 0, hp: 20, mp: 3, on_field: true, ..Default::default() },
            Unit { camp: ENEMY, class_id: 5, x: 1, y: 0, hp: 100, on_field: true, ..Default::default() },
            Unit { camp: ENEMY, class_id: 5, x: 2, y: 0, hp: 100, on_field: true, ..Default::default() },
        ],
        native_composition_event_bytes: vec![0; 3],
        native_command_book: book,
        native_command_resistances: BTreeMap::from([(5, 10)]),
        ..Default::default()
    };
    let (actor, confirmed, other) = (0, 1, 2);
    let got = st.execute_bound_native_command0(Some(actor), Some(confirmed), 3);
    assert!(
        matches!(&got, Ok((r, _)) if r.len() == 1 && r[0].target == Some(confirmed) && r[0].native_command_damage.hit),
        "results={got:?}"
    );
    let (a, c, o) = (&st.units[actor], &st.units[confirmed], &st.units[other]);
    assert!(
        a.mp == 1 && a.acted && c.hp < 100 && o.hp == 100,
        "mp/acted/hp actor={} acted={} confirmed={} other={}",
        a.mp,
        a.acted,
        c.hp,
        o.hp
    );
}

/// Go：`TestExecuteBoundNativeCommand0FailsBeforeMPOnMissingResistance`
#[test]
fn execute_bound_native_command0_fails_before_mp_on_missing_resistance() {
    let mut st = native_command0_test_state();
    st.units[TARGET].class_id = 99;
    let err = st.execute_bound_native_command0(Some(ACTOR), Some(TARGET), 1);
    let (a, t) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(
        err.is_err() && a.mp == 3 && !a.acted && t.hp == 100,
        "missing resistance mutated state: mp={} acted={} hp={} err={err:?}",
        a.mp,
        a.acted,
        t.hp
    );
}

/// Go：`TestPlanBoundNativeCommand0PublishesSevenRecoveredHPStages`
#[test]
fn plan_bound_native_command0_publishes_seven_recovered_hp_stages() {
    let mut st = native_command0_test_state();
    st.native_command_resistances = BTreeMap::from([(st.units[TARGET].class_id, 10)]);
    st.units[ACTOR].mp = 8;
    let (before_actor_mp, before_target_hp) = (st.units[ACTOR].mp, st.units[TARGET].hp);
    let mut plan = st.plan_bound_native_command0(Some(ACTOR), Some(TARGET), 3).unwrap();
    let (a, t) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(
        !(a.mp != before_actor_mp || a.acted || t.hp != before_target_hp),
        "plan mutated state mp={} acted={} hp={}",
        a.mp,
        a.acted,
        t.hp
    );
    apply_native_command_damage_mp(&mut st.units, Some(&mut plan)).unwrap();
    for stage in 1..=7 {
        apply_native_command_damage_stage(&mut st.units, Some(&mut plan), 0, stage)
            .unwrap_or_else(|err| panic!("stage {stage}: {err}"));
        let r = plan.results[0];
        let want = r.hp_before - (r.hp_before - r.hp_after) * stage / 7;
        assert!(st.units[TARGET].hp == want, "stage {stage} hp={} want={want}", st.units[TARGET].hp);
    }
    complete_native_command_damage(&mut st.units, Some(&mut plan)).unwrap();
    let (a, t) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(
        a.acted && t.hp == plan.results[0].hp_after && a.mp == plan.mp_after,
        "completion mp={} acted={} hp={} plan={plan:?}",
        a.mp,
        a.acted,
        t.hp
    );
}

/// Go：`TestPlanBoundNativeCommand0RejectsChangedStateBetweenMarkers`
#[test]
fn plan_bound_native_command0_rejects_changed_state_between_markers() {
    let mut st = native_command0_test_state();
    st.native_command_resistances = BTreeMap::from([(st.units[TARGET].class_id, 10)]);
    let mut plan = st.plan_bound_native_command0(Some(ACTOR), Some(TARGET), 1).unwrap();
    apply_native_command_damage_mp(&mut st.units, Some(&mut plan)).unwrap();
    st.units[TARGET].hp -= 1;
    assert!(
        apply_native_command_damage_stage(&mut st.units, Some(&mut plan), 0, 1).is_err(),
        "changed target state was accepted"
    );
}

/// Go：`TestNativeCommand6PublishesRecoveredFiveHPStages`
#[test]
fn native_command6_publishes_recovered_five_hp_stages() {
    let mut book = sequential_book();
    book[6] = NativeCommandRecord {
        id: 6,
        damage: 90,
        hit: 100,
        selection_mode: 1,
        effect_mode: 0,
        mp_cost: 2,
        target_code: 0,
    };
    let mut st = State {
        w: 2,
        h: 1,
        units: vec![
            Unit { camp: OWN, x: 0, y: 0, hp: 20, mp: 5, on_field: true, ..Default::default() },
            Unit { camp: ENEMY, class_id: 5, x: 1, y: 0, hp: 103, on_field: true, ..Default::default() },
        ],
        native_composition_event_bytes: vec![0; 2],
        native_command_book: book,
        ..Default::default()
    };
    let mut plan = st.plan_native_command_damage(Some(ACTOR), Some(TARGET), 6, &BTreeMap::from([(5, 10)]), 3).unwrap();
    assert!(plan.damage_stages == 5, "command6 stages={} want=5", plan.damage_stages);
    apply_native_command_damage_mp(&mut st.units, Some(&mut plan)).unwrap();
    for stage in 1..=plan.damage_stages {
        apply_native_command_damage_stage(&mut st.units, Some(&mut plan), 0, stage)
            .unwrap_or_else(|err| panic!("stage {stage}: {err}"));
        let r = plan.results[0];
        let want = r.hp_before - (r.hp_before - r.hp_after) * stage / 5;
        assert!(st.units[TARGET].hp == want, "stage {stage} hp={} want={want}", st.units[TARGET].hp);
    }
    assert!(
        apply_native_command_damage_stage(&mut st.units, Some(&mut plan), 0, 6).is_err(),
        "command6 accepted sixth HP marker"
    );
    complete_native_command_damage(&mut st.units, Some(&mut plan)).unwrap();
}

/// Go：`TestExecuteNativeCommandDamageAcceptsRecoveredIDOne`
#[test]
fn execute_native_command_damage_accepts_recovered_id_one() {
    let mut book = sequential_book();
    book[1] = NativeCommandRecord {
        id: 1,
        damage: 120,
        hit: 100,
        selection_mode: 1,
        effect_mode: 0,
        mp_cost: 1,
        target_code: 0,
    };
    let mut st = State {
        w: 2,
        h: 1,
        units: vec![
            Unit { camp: OWN, x: 0, mp: 2, hp: 1, on_field: true, ..Default::default() },
            Unit { camp: ENEMY, class_id: 5, x: 1, hp: 200, on_field: true, ..Default::default() },
        ],
        native_composition_event_bytes: vec![0; 2],
        native_command_book: book,
        ..Default::default()
    };
    let got = st.execute_native_command_damage(Some(ACTOR), Some(TARGET), 1, &BTreeMap::from([(5, 10)]), 1);
    let (a, t) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(matches!(&got, Ok((r, _)) if r.len() == 1) && a.mp == 1 && a.acted && t.hp < 200, "got={got:?}");
}

/// Go：`TestExecuteNativeCommandDamageAcceptsRecoveredCompositorIDTen`
#[test]
fn execute_native_command_damage_accepts_recovered_compositor_id_ten() {
    let mut book = sequential_book();
    book[10] = NativeCommandRecord {
        id: 10,
        damage: 100,
        hit: 100,
        selection_mode: 1,
        effect_mode: 0,
        mp_cost: 1,
        target_code: 0,
    };
    let mut st = State {
        w: 2,
        h: 1,
        units: vec![
            Unit { camp: OWN, on_field: true, hp: 20, mp: 3, x: 0, y: 0, ..Default::default() },
            Unit { camp: ENEMY, class_id: 5, on_field: true, hp: 200, max_hp: 200, x: 1, y: 0, ..Default::default() },
        ],
        native_composition_event_bytes: vec![0; 2],
        native_command_book: book,
        ..Default::default()
    };
    let got = st.execute_native_command_damage(Some(ACTOR), Some(TARGET), 10, &BTreeMap::from([(5, 10)]), 1);
    let (a, t) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(
        matches!(&got, Ok((r, _)) if r.len() == 1 && r[0].native_command_damage.hit)
            && a.mp == 2
            && a.acted
            && t.hp < 200,
        "ID10 numeric route = {got:?} actor={a:?} target={t:?}"
    );
}

/// Go 兩個 AI 測試共用的 actor／target 設定（target 的呈現 X 由參數決定）。
fn ai_command_units(target_x: u8) -> Vec<Unit> {
    let mut actor = complete_native_ai_scoring_unit();
    (actor.camp, actor.on_field, actor.acted) = (ENEMY, true, false);
    (actor.native_map_presentation.x, actor.native_map_presentation.y) = (0, 0);
    (actor.native_record_byte6, actor.mp) = (0, 10);
    let mut target = complete_native_ai_scoring_unit();
    (target.camp, target.on_field, target.class_id) = (OWN, true, 5);
    target.native_record_class = 5;
    (target.native_map_presentation.x, target.native_map_presentation.y) = (target_x, 0);
    (target.native_record_byte6, target.hp) = (1, 100);
    vec![actor, target]
}

/// Go：`TestPlanNativeAICommand4UsesRawSelectorTargetArray`
#[test]
fn plan_native_ai_command4_uses_raw_selector_target_array() {
    let mut book = sequential_book();
    book[4] = NativeCommandRecord {
        id: 4,
        damage: 40,
        hit: 100,
        selection_mode: 4,
        effect_mode: 1,
        mp_cost: 4,
        target_code: 0,
    };
    let st = State {
        w: 2,
        h: 1,
        units: ai_command_units(1),
        native_composition_event_bytes: vec![0, 0],
        native_command_book: book,
        ..Default::default()
    };
    let plan =
        st.plan_native_ai_command_damage(Some(ACTOR), Cell::default(), 4, &BTreeMap::from([(5, 10)]), 3).unwrap();
    let (a, t) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(
        plan.results.len() == 1
            && plan.results[0].target == Some(TARGET)
            && plan.damage_stages == 6
            && a.mp == 10
            && t.hp == 100,
        "AI command4 plan={plan:?} actorMP={} targetHP={}",
        a.mp,
        t.hp
    );
}

/// Go：`TestPlanNativeAICommandDamageUsesWinnerCellNotActorCell`
///
/// 0x15311 keeps the actor on its cell and builds the 0x14818 geometry at the
/// winner pair [0x53C27]/[0x53C2B]; a single-cell (effect_mode 0) command
/// therefore hits the unit standing on the winner cell, not on the actor's.
#[test]
fn plan_native_ai_command_damage_uses_winner_cell_not_actor_cell() {
    let mut book = sequential_book();
    book[0] = NativeCommandRecord {
        id: 0,
        damage: 40,
        hit: 100,
        selection_mode: 3,
        effect_mode: 0,
        mp_cost: 0,
        target_code: 0,
    };
    let st = State {
        w: 3,
        h: 1,
        units: ai_command_units(2),
        native_composition_event_bytes: vec![0, 0, 0],
        native_command_book: book,
        ..Default::default()
    };
    let resist = BTreeMap::from([(5, 10)]);
    assert!(
        st.plan_native_ai_command_damage(Some(ACTOR), Cell { x: 0, y: 0 }, 0, &resist, 3).is_err(),
        "actor cell as origin must not find the target on the winner cell"
    );
    let plan = st.plan_native_ai_command_damage(Some(ACTOR), Cell { x: 2, y: 0 }, 0, &resist, 3).unwrap();
    assert!(plan.results.len() == 1 && plan.results[0].target == Some(TARGET), "winner-cell plan={plan:?}");
    assert!(
        st.plan_native_ai_command_damage(Some(ACTOR), Cell { x: 3, y: 0 }, 0, &resist, 3).is_err(),
        "origin outside the grid must fail closed"
    );
}
