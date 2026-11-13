//! 移植自 `remake/internal/battle/native_command13_test.go`。

use super::*;

fn native_command_heal_book(id: i64) -> Vec<NativeCommandRecord> {
    let mut book: Vec<NativeCommandRecord> =
        (0..36).map(|i| NativeCommandRecord { id: i, ..Default::default() }).collect();
    book[id as usize] = NativeCommandRecord {
        id,
        damage: 70,
        selection_mode: 1,
        effect_mode: 0,
        mp_cost: 3,
        target_code: 1,
        ..Default::default()
    };
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

/// Go：`TestExecuteNativeCommandHealUsesSelectedRecordAndCapsHP`
#[test]
fn execute_native_command_heal_uses_selected_record_and_caps_hp() {
    let mut st = State {
        w: 2,
        h: 1,
        units: vec![
            Unit { camp: OWN, on_field: true, hp: 20, mp: 5, x: 0, y: 0, ..Default::default() },
            Unit { camp: OWN, on_field: true, hp: 40, max_hp: 100, x: 1, y: 0, ..Default::default() },
        ],
        native_composition_event_bytes: vec![0; 2],
        native_command_book: native_command_heal_book(13),
        ..Default::default()
    };
    let (actor, target) = (0, 1);
    let mut rng = GoMathRand::new(2);
    let got = st.execute_native_command_heal(Some(actor), Some(target), 13, Some(&mut rng));
    assert!(
        matches!(&got, Ok(g) if g.len() == 1 && g[0].target == Some(target) && (63..=69).contains(&g[0].restore.rolled)),
        "heal = {got:?}"
    );
    let got = got.unwrap();
    let (a, t) = (&st.units[actor], &st.units[target]);
    assert!(
        t.hp == 100 && got[0].restore.actual == 60 && a.mp == 2 && a.acted,
        "post heal actor={a:?} target={t:?} result={:?}",
        got[0]
    );
}

/// Go：`TestExecuteNativeCommandHealRejectsFamilyBoundaryBeforeMutation`
///
/// Go 以空 `State` 呼叫且 actor 不在 `Units`；Rust 依模組文件第 6 點先 push 再傳索引。
#[test]
fn execute_native_command_heal_rejects_family_boundary_before_mutation() {
    let mut st = State { units: vec![Unit { mp: 5, ..Default::default() }], ..Default::default() };
    let mut rng = GoMathRand::new(1);
    let err = st.execute_native_command_heal(Some(0), None, 12, Some(&mut rng));
    let actor = &st.units[0];
    assert!(err.is_err() && actor.mp == 5 && !actor.acted, "non-heal ID must fail closed: actor={actor:?} err={err:?}");
}

/// Go：`TestExecuteNativeAICommandHealRebuildsTargetsFromRawSelector`
#[test]
fn execute_native_ai_command_heal_rebuilds_targets_from_raw_selector() {
    let mut actor = complete_native_ai_scoring_unit();
    (actor.native_map_presentation.x, actor.native_map_presentation.y) = (0, 0);
    (actor.native_record_byte5, actor.native_record_byte6) = (0, 0);
    (actor.hp, actor.max_hp, actor.mp) = (100, 100, 5);
    let mut target = complete_native_ai_scoring_unit();
    (target.native_map_presentation.x, target.native_map_presentation.y) = (1, 0);
    (target.native_record_byte5, target.native_record_byte6) = (0, 0);
    (target.hp, target.max_hp) = (40, 100);
    let mut book = native_command_heal_book(13);
    book[13].effect_mode = 1;
    let mut st = State {
        w: 2,
        h: 1,
        units: vec![actor, target],
        native_composition_event_bytes: vec![0; 2],
        native_command_book: book,
        ..Default::default()
    };
    let (actor, target) = (0, 1);
    let mut rng = GoMathRand::new(2);
    let got = st.execute_native_ai_command_heal(Some(actor), Cell::default(), 13, Some(&mut rng)).unwrap();
    assert!(got.len() == 2 && got[0].target == Some(actor) && got[1].target == Some(target), "AI target array={got:?}");
    let (a, t) = (&st.units[actor], &st.units[target]);
    assert!(a.mp == 2 && t.hp == 100 && a.acted, "AI heal actor={a:?} target={t:?}");
}

/// Go：`TestExecuteNativeAICommandHealRejectsMissingSelectorBeforeMutation`
#[test]
fn execute_native_ai_command_heal_rejects_missing_selector_before_mutation() {
    let mut actor = complete_native_ai_scoring_unit();
    actor.has_native_record_byte6 = false;
    actor.mp = 5;
    let mut st = State { native_command_book: native_command_heal_book(13), units: vec![actor], ..Default::default() };
    let mut rng = GoMathRand::new(1);
    let err = st.execute_native_ai_command_heal(Some(0), Cell::default(), 13, Some(&mut rng));
    let actor = &st.units[0];
    assert!(
        err.is_err() && actor.mp == 5 && !actor.acted,
        "missing selector crossed AI heal gate: actor={actor:?} err={err:?}"
    );
}
