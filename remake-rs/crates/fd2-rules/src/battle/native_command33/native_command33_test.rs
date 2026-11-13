//! 移植自 `remake/internal/battle/native_command33_test.go`。

use super::*;

/// Go `nativeCompound33Fixture`：回傳的 actor、target 是 `st.units` 的 0、1。
fn native_compound33_fixture() -> State {
    let actor = Unit {
        camp: OWN,
        on_field: true,
        x: 0,
        y: 0,
        hp: 40,
        max_hp: 100,
        mp: 60,
        lv: 20,
        battle_fig: 4,
        has_battle_fig: true,
        native_record_class: 19,
        has_native_record_class: true,
        native_transient: [1, 2, 3, 4, 5, 6],
        ..Default::default()
    };
    let target = Unit {
        camp: OWN,
        on_field: true,
        x: 1,
        y: 0,
        hp: 10,
        max_hp: 120,
        lv: 12,
        battle_fig: 5,
        has_battle_fig: true,
        native_record_class: 2,
        has_native_record_class: true,
        native_transient: [7, 8, 9, 10, 11, 12],
        ..Default::default()
    };
    let mut book: Vec<NativeCommandRecord> =
        (0..NATIVE_COMMAND_RECORD_COUNT).map(|id| NativeCommandRecord { id, ..Default::default() }).collect();
    book[33] = NativeCommandRecord {
        id: 33,
        selection_mode: 5,
        effect_mode: 3,
        mp_cost: 52,
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

/// Go：`TestExecuteNativeCompoundCommand33ClearsRawMarkersAndRestoresAtomically`
#[test]
fn execute_native_compound_command33_clears_raw_markers_and_restores_atomically() {
    let mut st = native_compound33_fixture();
    let result = st.execute_native_compound_command33(Some(0), Some(1), 1).unwrap();
    let actor = &st.units[0];
    assert!(
        result.restore.results.len() == 2 && result.restore.rng_state != 1 && actor.mp == 60 && actor.acted,
        "command33 result={result:?} actor={actor:?}"
    );
    for (index, unit) in st.units.iter().enumerate() {
        assert!(
            !(unit.hp != unit.max_hp
                || unit.native_transient[3] != 0
                || unit.native_transient[4] != 0
                || unit.native_transient[5] != 0),
            "target {index} did not publish restore/clear: {unit:?}"
        );
    }
}

/// Go：`TestExecuteNativeCompoundCommand33RejectsIncompleteTargetWithoutMutation`
#[test]
fn execute_native_compound_command33_rejects_incomplete_target_without_mutation() {
    let mut st = native_compound33_fixture();
    st.units[1].has_battle_fig = false;
    let (a, t) = (&st.units[0], &st.units[1]);
    let (actor_hp, actor_mp, actor_acted, actor_raw) = (a.hp, a.mp, a.acted, a.native_transient);
    let (target_hp, target_raw) = (t.hp, t.native_transient);
    assert!(
        st.execute_native_compound_command33(Some(0), Some(1), 1).is_err(),
        "incomplete target provenance was accepted"
    );
    let (a, t) = (&st.units[0], &st.units[1]);
    assert!(
        !(a.hp != actor_hp
            || a.mp != actor_mp
            || a.acted != actor_acted
            || a.native_transient != actor_raw
            || t.hp != target_hp
            || t.native_transient != target_raw),
        "failed preflight mutated actor={a:?} target={t:?}"
    );
}

/// Go：`TestExecuteNativeCompoundCommand33RejectsUnprovenPlayerSelector`
#[test]
fn execute_native_compound_command33_rejects_unproven_player_selector() {
    let mut st = native_compound33_fixture();
    st.units[0].battle_fig = 18;
    let err = st.execute_native_compound_command33(Some(0), Some(1), 1);
    let actor = &st.units[0];
    assert!(
        err.is_err() && !actor.acted && actor.mp == 60,
        "unproven selector transaction actor={actor:?} err={err:?}"
    );
}

/// Go：`TestNativeCompoundCommand33PlanPublishesOnlyAtExplicitBoundary`
#[test]
fn native_compound_command33_plan_publishes_only_at_explicit_boundary() {
    let mut st = native_compound33_fixture();
    let (actor_hp, actor_raw) = (st.units[0].hp, st.units[0].native_transient);
    let (target_hp, target_raw) = (st.units[1].hp, st.units[1].native_transient);
    let mut plan = st.plan_native_compound_command33(Some(0), Some(1), 1).unwrap();
    let (a, t) = (&st.units[0], &st.units[1]);
    assert!(
        !(a.hp != actor_hp
            || a.native_transient != actor_raw
            || t.hp != target_hp
            || t.native_transient != target_raw
            || a.acted),
        "plan mutated live state actor={a:?} target={t:?}"
    );
    publish_native_compound_command33(&mut st.units, Some(&mut plan)).unwrap();
    let (a, t) = (&st.units[0], &st.units[1]);
    assert!(
        !(a.hp != a.max_hp || t.hp != t.max_hp || a.native_transient[3] != 0 || t.native_transient[5] != 0 || a.acted),
        "publication boundary mismatch actor={a:?} target={t:?}"
    );
    complete_native_compound_command33(&mut st.units, Some(&mut plan)).unwrap();
    assert!(st.units[0].acted, "completion did not publish Acted");
}

/// Go：`TestNativeCompoundCommand33AbortRestoresPublishedBatch`
#[test]
fn native_compound_command33_abort_restores_published_batch() {
    let mut st = native_compound33_fixture();
    let (actor_hp, actor_raw) = (st.units[0].hp, st.units[0].native_transient);
    let (target_hp, target_raw) = (st.units[1].hp, st.units[1].native_transient);
    let mut plan = st.plan_native_compound_command33(Some(0), Some(1), 7).unwrap();
    publish_native_compound_command33(&mut st.units, Some(&mut plan)).unwrap();
    abort_native_compound_command33(&mut st.units, Some(&mut plan)).unwrap();
    let (a, t) = (&st.units[0], &st.units[1]);
    assert!(
        !(a.hp != actor_hp
            || a.native_transient != actor_raw
            || t.hp != target_hp
            || t.native_transient != target_raw
            || a.acted),
        "abort did not restore batch actor={a:?} target={t:?}"
    );
}

/// Go：`TestNativeCompoundCommand33ChangedTargetFailsBeforePublication`
#[test]
fn native_compound_command33_changed_target_fails_before_publication() {
    let mut st = native_compound33_fixture();
    let mut plan = st.plan_native_compound_command33(Some(0), Some(1), 9).unwrap();
    st.units[1].hp += 1;
    let err = publish_native_compound_command33(&mut st.units, Some(&mut plan));
    let (a, t) = (&st.units[0], &st.units[1]);
    assert!(err.is_err() && a.hp == 40 && !a.acted, "changed target publication actor={a:?} target={t:?} err={err:?}");
}
