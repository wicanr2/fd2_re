//! 移植自 `remake/internal/battle/native_command34_test.go`。

use super::*;

/// Go `nativeCompound34Fixture`：回傳的 actor、target 是 `st.units` 的 0、1。
fn native_compound34_fixture() -> State {
    let actor = Unit {
        camp: OWN,
        on_field: true,
        x: 0,
        y: 0,
        hp: 100,
        max_hp: 100,
        mp: 30,
        ap: 100,
        dp: 80,
        hit: 60,
        ev: 50,
        lv: 20,
        battle_fig: 4,
        has_battle_fig: true,
        native_record_class: 19,
        has_native_record_class: true,
        ..Default::default()
    };
    let target = Unit {
        camp: OWN,
        on_field: true,
        x: 1,
        y: 0,
        hp: 90,
        max_hp: 90,
        mp: 0,
        ap: 70,
        dp: 60,
        hit: 50,
        ev: 40,
        lv: 12,
        battle_fig: 5,
        has_battle_fig: true,
        native_record_class: 2,
        has_native_record_class: true,
        ..Default::default()
    };
    let mut book: Vec<NativeCommandRecord> =
        (0..NATIVE_COMMAND_RECORD_COUNT).map(|id| NativeCommandRecord { id, ..Default::default() }).collect();
    book[34] = NativeCommandRecord {
        id: 34,
        selection_mode: 5,
        effect_mode: 3,
        mp_cost: 28,
        target_code: 1,
        ..Default::default()
    };
    State {
        w: 2,
        h: 1,
        units: vec![actor, target],
        native_composition_event_bytes: vec![0; 2],
        native_command_book: book,
        ..Default::default()
    }
}

/// Go `nativeCompound34UnitState`：mp、ap、dp、hit、ev、acted、transient。
type NativeCompound34UnitState = (i64, i64, i64, i64, i64, bool, [u8; 6]);

fn native_compound34_state(unit: &Unit) -> NativeCompound34UnitState {
    (unit.mp, unit.ap, unit.dp, unit.hit, unit.ev, unit.acted, unit.native_transient)
}

/// Go：`TestExecuteNativeCompoundCommand34PublishesThreeStagesAtomically`
#[test]
fn execute_native_compound_command34_publishes_three_stages_atomically() {
    let mut st = native_compound34_fixture();
    let result = st.execute_native_compound_command34(Some(0), Some(1), 1).unwrap();
    assert!(
        result.stages.len() == 3
            && result.stages[0].command_id == 17
            && result.stages[1].command_id == 18
            && result.stages[2].command_id == 19
            && result.rng_state != 1,
        "compound stages={:?} rng={:#x}",
        result.stages,
        result.rng_state
    );
    let actor = &st.units[0];
    assert!(actor.mp == 30 && actor.acted, "reachable class-19 MP contract actor={actor:?}");
    for (index, unit) in st.units.iter().enumerate() {
        assert!(
            !(unit.native_transient[0] == 0
                || unit.native_transient[1] == 0
                || unit.native_transient[2] == 0
                || unit.ap <= [100, 70][index]
                || unit.dp <= [80, 60][index]
                || unit.hit <= [60, 50][index]
                || unit.ev <= [50, 40][index]),
            "target {index} did not receive all raw stages: {unit:?}"
        );
    }
}

/// Go：`TestExecuteNativeCompoundCommand34RejectsUnprovenSelectorWithoutMutation`
#[test]
fn execute_native_compound_command34_rejects_unproven_selector_without_mutation() {
    let mut st = native_compound34_fixture();
    st.units[0].battle_fig = 18;
    let (actor_before, target_before) = (native_compound34_state(&st.units[0]), native_compound34_state(&st.units[1]));
    assert!(
        st.execute_native_compound_command34(Some(0), Some(1), 1).is_err(),
        "unproven compound selector was accepted"
    );
    assert!(
        native_compound34_state(&st.units[0]) == actor_before && native_compound34_state(&st.units[1]) == target_before,
        "failed selector gate mutated state actor={:?} target={:?}",
        st.units[0],
        st.units[1]
    );
}

/// Go：`TestExecuteNativeCompoundCommand34PreflightsEveryTargetBeforeMutation`
#[test]
fn execute_native_compound_command34_preflights_every_target_before_mutation() {
    let mut st = native_compound34_fixture();
    st.units[1].has_native_record_class = false;
    let (actor_before, target_before) = (native_compound34_state(&st.units[0]), native_compound34_state(&st.units[1]));
    assert!(
        st.execute_native_compound_command34(Some(0), Some(1), 1).is_err(),
        "incomplete final target provenance was accepted"
    );
    assert!(
        native_compound34_state(&st.units[0]) == actor_before && native_compound34_state(&st.units[1]) == target_before,
        "failed target preflight mutated state actor={:?} target={:?}",
        st.units[0],
        st.units[1]
    );
}
