//! 移植自 `remake/internal/battle/native_command9_ai_test.go`。

use super::*;

fn command9_book(record: NativeCommandRecord) -> Vec<NativeCommandRecord> {
    let mut book: Vec<NativeCommandRecord> =
        (0..NATIVE_COMMAND_RECORD_COUNT).map(|id| NativeCommandRecord { id, ..Default::default() }).collect();
    book[9] = record;
    book
}

fn command9_units() -> Vec<Unit> {
    vec![
        Unit { camp: ENEMY, hp: 50, mp: 40, on_field: true, ..Default::default() },
        Unit { camp: OWN, class_id: 5, hp: 500, on_field: true, ..Default::default() },
    ]
}

/// Go：`TestPlanNativeAICommand9UsesProducerTargetAndTwentyStages`
#[test]
fn plan_native_ai_command9_uses_producer_target_and_twenty_stages() {
    let book = command9_book(NativeCommandRecord {
        id: 9,
        damage: 999,
        hit: 100,
        selection_mode: 3,
        effect_mode: 0,
        mp_cost: 30,
        target_code: 0,
    });
    let st = State { native_command_book: book, units: command9_units(), ..Default::default() };
    let (actor, target) = (0, 1);
    let plan = st
        .plan_native_ai_command_damage_single_target(Some(actor), Some(target), 9, &BTreeMap::from([(5, 10)]), 3)
        .unwrap();
    assert!(
        plan.damage_stages == 20
            && plan.results.len() == 1
            && plan.results[0].target == Some(target)
            && st.units[actor].mp == 40
            && st.units[target].hp == 500,
        "AI command9 plan={plan:?} actorMP={} targetHP={}",
        st.units[actor].mp,
        st.units[target].hp
    );
}

/// Go：`TestPlanNativeAICommand9RejectsPlayerAndAreaRecords`
///
/// Go 的 State 沒有放入單位（`Units` 為 nil）；Rust 依模組文件第 6 點先 push 再傳索引。
#[test]
fn plan_native_ai_command9_rejects_player_and_area_records() {
    let mut st = State {
        native_command_book: command9_book(NativeCommandRecord {
            id: 9,
            damage: 999,
            hit: 100,
            effect_mode: 1,
            mp_cost: 30,
            target_code: 0,
            ..Default::default()
        }),
        units: command9_units(),
        ..Default::default()
    };
    let resist = BTreeMap::from([(5, 10)]);
    assert!(
        st.plan_native_ai_command_damage_single_target(Some(0), Some(1), 9, &resist, 3).is_err(),
        "unproven area record was accepted"
    );
    st.units[0].camp = OWN;
    // Go 的 book 與 st.NativeCommandBook 共用底層陣列；Rust 直接改 State 上那份。
    st.native_command_book[9].effect_mode = 0;
    assert!(
        st.plan_native_ai_command_damage_single_target(Some(0), Some(1), 9, &resist, 3).is_err(),
        "player actor was accepted by AI planner"
    );
}
