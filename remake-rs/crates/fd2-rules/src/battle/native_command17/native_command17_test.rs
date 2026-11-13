//! 移植自 `remake/internal/battle/native_command17_test.go`。

use super::*;

fn native_command_modifier_book(id: i64) -> Vec<NativeCommandRecord> {
    let mut book: Vec<NativeCommandRecord> =
        (0..NATIVE_COMMAND_RECORD_COUNT).map(|i| NativeCommandRecord { id: i, ..Default::default() }).collect();
    let rec = |id, mp_cost| NativeCommandRecord {
        id,
        selection_mode: 1,
        effect_mode: 0,
        mp_cost,
        target_code: 1,
        ..Default::default()
    };
    book[id as usize] = rec(id, 99);
    book[18] = rec(18, 4);
    if id == 19 {
        book[19] = rec(19, 5);
    }
    book
}

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

fn modifier_actor() -> Unit {
    Unit { camp: OWN, on_field: true, hp: 10, max_hp: 10, mp: 10, x: 0, y: 0, ..Default::default() }
}

fn modifier_target(ap: i64, dp: i64, hit: i64, ev: i64) -> Unit {
    Unit {
        camp: OWN,
        on_field: true,
        hp: 10,
        max_hp: 10,
        ap,
        dp,
        hit,
        ev,
        x: 1,
        y: 0,
        lv: 2,
        native_record_class: 9,
        has_native_record_class: true,
        ..Default::default()
    }
}

fn modifier_state(target: Unit, book: Vec<NativeCommandRecord>) -> State {
    State {
        w: 2,
        h: 1,
        units: vec![modifier_actor(), target],
        native_composition_event_bytes: vec![0; 2],
        native_command_book: book,
        ..Default::default()
    }
}

/// Go 兩個 AI 測試共用的 actor／target 設定。
fn ai_modifier_state() -> State {
    let mut actor = complete_native_ai_scoring_unit();
    (actor.native_map_presentation.x, actor.native_map_presentation.y) = (0, 0);
    (actor.native_record_byte5, actor.native_record_byte6) = (0, 0);
    actor.native_transient = [0; 6];
    (actor.camp, actor.mp, actor.ap) = (ENEMY, 10, 100);
    let mut target = complete_native_ai_scoring_unit();
    (target.native_map_presentation.x, target.native_map_presentation.y) = (1, 0);
    (target.native_record_byte5, target.native_record_byte6) = (0, 0);
    target.native_transient = [0; 6];
    (target.camp, target.ap) = (ENEMY, 200);
    let mut book = native_command_modifier_book(17);
    book[17].effect_mode = 1;
    State {
        w: 2,
        h: 1,
        units: vec![actor, target],
        native_composition_event_bytes: vec![0; 2],
        native_command_book: book,
        ..Default::default()
    }
}

/// Go：`TestExecuteNativeCommand17UsesRecord18DebitAndPublishesAtomically`
#[test]
fn execute_native_command17_uses_record18_debit_and_publishes_atomically() {
    let mut st = modifier_state(modifier_target(100, 20, 30, 40), native_command_modifier_book(17));
    let got = st.execute_native_command_modifier(Some(0), Some(1), 17, 0).unwrap();
    assert!(
        got.rng_state == 0x80a4 && got.word_steps.len() == 1 && got.word_steps[0].processed,
        "modifier result={got:?}"
    );
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        target.native_transient[0] == 2 && target.ap == 116 && actor.mp == 6 && actor.acted,
        "actor={actor:?} target={target:?}"
    );
}

/// Go：`TestExecuteNativeCommand19PublishesPairAndOwnDebit`
#[test]
fn execute_native_command19_publishes_pair_and_own_debit() {
    let mut st = modifier_state(modifier_target(10, 20, -2, 1), native_command_modifier_book(19));
    let got = st.execute_native_command_modifier(Some(0), Some(1), 19, 0).unwrap();
    assert!(
        got.rng_state == 0x80a4 && got.pair_steps.len() == 1 && got.pair_steps[0].processed,
        "modifier result={got:?}"
    );
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        target.native_transient[2] == 2 && target.hit == 13 && target.ev == 16 && actor.mp == 5 && actor.acted,
        "actor={actor:?} target={target:?}"
    );
}

/// Go：`TestExecuteNativeCommand18PublishesDPAndOwnDebit`
#[test]
fn execute_native_command18_publishes_dp_and_own_debit() {
    let mut st = modifier_state(modifier_target(10, 100, 30, 40), native_command_modifier_book(18));
    let got = st.execute_native_command_modifier(Some(0), Some(1), 18, 0).unwrap();
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        got.word_steps.len() == 1
            && got.word_steps[0].processed
            && target.native_transient[1] == 2
            && target.dp == 116
            && actor.mp == 6
            && actor.acted,
        "result={got:?} actor={actor:?} target={target:?}"
    );
}

/// Go：`TestExecuteNativeCommandModifierRejectsBadTargetBeforeMutation`
///
/// Go 的 `good` 不在 `st.Units`；Rust 依模組文件第 6 點 push 到索引 2（不在場，`OnField` 為 false）
/// 再傳索引。被測方法只處理傳入的 targets，不掃整個 `units`。
#[test]
fn execute_native_command_modifier_rejects_bad_target_before_mutation() {
    let good = Unit {
        ap: 100,
        dp: 20,
        hit: 30,
        ev: 40,
        lv: 2,
        native_record_class: 9,
        has_native_record_class: true,
        ..Default::default()
    };
    let mut st = modifier_state(modifier_target(1 << 20, 20, 30, 40), native_command_modifier_book(17));
    st.units.push(good);
    let (actor, target, good) = (0, 1, 2);
    assert!(
        st.execute_native_command_modifier_targets(Some(actor), &[good, target], 17, 0).is_err(),
        "out-of-range target accepted"
    );
    let (a, g, t) = (&st.units[actor], &st.units[good], &st.units[target]);
    assert!(
        !(a.mp != 10
            || a.acted
            || g.native_transient != [0; 6]
            || g.ap != 100
            || t.native_transient != [0; 6]
            || t.ap != 1 << 20),
        "failed transaction mutated actor={a:?} good={g:?} target={t:?}"
    );
}

/// Go：`TestExecuteNativeAICommandModifierUsesRawSelectorTargets`
#[test]
fn execute_native_ai_command_modifier_uses_raw_selector_targets() {
    let mut st = ai_modifier_state();
    let got = st.execute_native_ai_command_modifier(Some(0), Cell::default(), 17, 0).unwrap();
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        got.word_steps.len() == 2 && actor.ap == 116 && target.ap == 231 && actor.mp == 6 && actor.acted,
        "result={got:?} actor={actor:?} target={target:?}"
    );
}

/// Go：`TestNativeAICommandModifierPlanPublishesAndRollsBackAtTailBoundary`
#[test]
fn native_ai_command_modifier_plan_publishes_and_rolls_back_at_tail_boundary() {
    let mut st = ai_modifier_state();
    let mut plan = st.plan_native_ai_command_modifier(Some(0), Cell::default(), 17, 0).unwrap();
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(!(actor.ap != 100 || target.ap != 200 || actor.mp != 10 || actor.acted), "plan mutated live state");
    publish_native_ai_command_modifier(&mut st.units, Some(&mut plan)).unwrap();
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        !(actor.ap != 116 || target.ap != 231 || actor.mp != 6 || actor.acted),
        "publish boundary actor={actor:?} target={target:?}"
    );
    abort_native_ai_command_modifier(&mut st.units, Some(&mut plan)).unwrap();
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        !(actor.ap != 100 || target.ap != 200 || actor.mp != 10 || actor.acted),
        "rollback actor={actor:?} target={target:?}"
    );
}
