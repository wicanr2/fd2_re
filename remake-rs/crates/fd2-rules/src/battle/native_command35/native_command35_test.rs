//! 移植自 `remake/internal/battle/native_command35_test.go`。

use super::*;

/// Go `nativeCompound35Fixture`：回傳的 actor、target 是 `st.units` 的 0、1。
fn native_compound35_fixture() -> State {
    let actor = Unit {
        camp: OWN,
        on_field: true,
        x: 0,
        y: 0,
        hp: 100,
        max_hp: 100,
        mp: 40,
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
        lv: 12,
        battle_fig: 5,
        has_battle_fig: true,
        native_record_class: 2,
        has_native_record_class: true,
        ..Default::default()
    };
    let mut book: Vec<NativeCommandRecord> =
        (0..NATIVE_COMMAND_RECORD_COUNT).map(|id| NativeCommandRecord { id, ..Default::default() }).collect();
    book[35] = NativeCommandRecord {
        id: 35,
        selection_mode: 5,
        effect_mode: 3,
        mp_cost: 36,
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

/// Go `nativeCompound35UnitState`：hp、mp、acted、raw。
type NativeCompound35UnitState = (i64, i64, bool, [u8; 6]);

fn native_compound35_state(unit: &Unit) -> NativeCompound35UnitState {
    (unit.hp, unit.mp, unit.acted, unit.native_transient)
}

/// Go：`TestExecuteNativeCompoundCommand35PublishesThreeApplicationsAtomically`
#[test]
fn execute_native_compound_command35_publishes_three_applications_atomically() {
    let mut st = native_compound35_fixture();
    let result = st.execute_native_compound_command35(Some(0), Some(1), 1).unwrap();
    let want_ids = [26, 22, 27];
    let want_offsets = [0x25, 0x27, 0x26];
    assert!(
        result.stages.len() == 3 && result.rng_state == 0x5541,
        "command35 stages={:?} rng={:#x}",
        result.stages,
        result.rng_state
    );
    let (actor, target) = (&st.units[0], &st.units[1]);
    for (stage_index, stage) in result.stages.iter().enumerate() {
        assert!(
            !(stage.command_id != want_ids[stage_index]
                || stage.marker_offset != want_offsets[stage_index]
                || stage.results.len() != 2
                || stage.accumulator != 8 * (actor.lv + target.lv)),
            "command35 stage {stage_index}={stage:?}"
        );
        for application in &stage.results {
            assert!(
                application.applied && (2..=5).contains(&application.marker),
                "command35 stage {stage_index} application={application:?}"
            );
        }
    }
    assert!(
        actor.mp == 40 && actor.acted && actor.hp < 100 && target.hp < 90,
        "command35 published actor={actor:?} target={target:?}"
    );
    for (index, unit) in st.units.iter().enumerate() {
        assert!(
            !(unit.native_transient[3] == 0 || unit.native_transient[4] == 0 || unit.native_transient[5] == 0),
            "command35 target {index} markers={:?}",
            unit.native_transient
        );
    }
}

/// Go：`TestExecuteNativeCompoundCommand35RejectsIncompleteTargetAtomically`
#[test]
fn execute_native_compound_command35_rejects_incomplete_target_atomically() {
    let mut st = native_compound35_fixture();
    st.units[1].has_battle_fig = false;
    let (actor_before, target_before) = (native_compound35_state(&st.units[0]), native_compound35_state(&st.units[1]));
    assert!(
        st.execute_native_compound_command35(Some(0), Some(1), 1).is_err(),
        "command35 accepted incomplete target provenance"
    );
    assert!(
        native_compound35_state(&st.units[0]) == actor_before && native_compound35_state(&st.units[1]) == target_before,
        "failed command35 mutated actor={:?} target={:?}",
        st.units[0],
        st.units[1]
    );
}

/// Go：`TestExecuteNativeCompoundCommand35RejectsUnprovenSelectorAndMPGate`
#[test]
fn execute_native_compound_command35_rejects_unproven_selector_and_mp_gate() {
    let mutations: [fn(&mut Unit); 2] = [|actor| actor.battle_fig = 18, |actor| actor.mp = 35];
    for mutate in mutations {
        let mut st = native_compound35_fixture();
        mutate(&mut st.units[0]);
        let (actor_before, target_before) =
            (native_compound35_state(&st.units[0]), native_compound35_state(&st.units[1]));
        assert!(
            st.execute_native_compound_command35(Some(0), Some(1), 1).is_err(),
            "command35 accepted an unproven actor gate"
        );
        assert!(
            native_compound35_state(&st.units[0]) == actor_before
                && native_compound35_state(&st.units[1]) == target_before,
            "failed command35 gate mutated actor={:?} target={:?}",
            st.units[0],
            st.units[1]
        );
    }
}
