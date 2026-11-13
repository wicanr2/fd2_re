//! 移植自 `remake/internal/battle/native_command32_test.go`。

use super::*;

/// Go `nativeCompound32Fixture`：回傳的 actor、target 是 `st.units` 的 0、1。
fn native_compound32_fixture() -> State {
    let actor = Unit {
        camp: OWN,
        on_field: true,
        x: 0,
        y: 0,
        hp: 100,
        max_hp: 100,
        mp: 80,
        lv: 20,
        class_id: 5,
        battle_fig: 4,
        has_battle_fig: true,
        native_record_class: 19,
        has_native_record_class: true,
        ..Default::default()
    };
    let target = Unit {
        camp: ENEMY,
        on_field: true,
        x: 1,
        y: 0,
        hp: 100,
        max_hp: 100,
        lv: 12,
        class_id: 5,
        battle_fig: 5,
        has_battle_fig: true,
        native_record_class: 2,
        has_native_record_class: true,
        ..Default::default()
    };
    let mut book: Vec<NativeCommandRecord> =
        (0..NATIVE_COMMAND_RECORD_COUNT).map(|id| NativeCommandRecord { id, ..Default::default() }).collect();
    book[32] = NativeCommandRecord {
        id: 32,
        damage: 50,
        hit: 100,
        selection_mode: 1,
        effect_mode: 0,
        mp_cost: 76,
        target_code: 0,
    };
    State {
        w: 2,
        h: 1,
        units: vec![actor, target],
        native_composition_event_bytes: vec![0; 2],
        native_command_book: book,
        native_command_resistances: BTreeMap::from([(2, 10)]),
        ..Default::default()
    }
}

const ACTOR: UnitIdx = 0;
const TARGET: UnitIdx = 1;

/// Go：`TestExecuteNativeCompoundCommand32PublishesDamageAtomically`
#[test]
fn execute_native_compound_command32_publishes_damage_atomically() {
    let mut st = native_compound32_fixture();
    let result = st.execute_native_compound_command32(Some(ACTOR), Some(TARGET), 7).unwrap();
    let (actor, target) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(
        result.targets.len() == 1
            && result.targets[0].damage.hit
            && result.targets[0].hp_before == 100
            && result.targets[0].hp_after < 100
            && target.hp == result.targets[0].hp_after
            && result.rng_state != 7
            && actor.mp == 80
            && actor.acted,
        "command32 result={result:?} actor={actor:?} target={target:?}"
    );
    assert!(target.native_transient == [0; 6], "command32 invented raw markers: {:?}", target.native_transient);
}

/// Go：`TestExecuteNativeCompoundCommand32MissConsumesOnlyHitRoll`
#[test]
fn execute_native_compound_command32_miss_consumes_only_hit_roll() {
    let mut st = native_compound32_fixture();
    st.native_command_book[32].hit = 0;
    let result = st.execute_native_compound_command32(Some(ACTOR), Some(TARGET), 1).unwrap();
    let want = resolve_native_command_damage(50, 0, 10, 1);
    let target = &st.units[TARGET];
    assert!(
        matches!(want, Ok((_, want_state)) if result.targets.len() == 1 && !result.targets[0].damage.hit
            && target.hp == 100 && result.rng_state == want_state),
        "command32 miss={result:?} target={target:?} want_rng={want:?}"
    );
}

/// Go：`TestExecuteNativeCompoundCommand32RejectsMissingProvenanceAtomically`
#[test]
fn execute_native_compound_command32_rejects_missing_provenance_atomically() {
    let mutations: [fn(&mut State); 4] = [
        |st| st.units[ACTOR].battle_fig = 18,
        |st| st.units[ACTOR].mp = 75,
        |st| {
            let class = st.units[TARGET].native_record_class as i64;
            st.native_command_resistances.remove(&class);
        },
        |st| st.units[TARGET].has_native_record_class = false,
    ];
    for mutate in mutations {
        let mut st = native_compound32_fixture();
        mutate(&mut st);
        let (a, t) = (&st.units[ACTOR], &st.units[TARGET]);
        let (actor_hp, actor_mp, actor_acted, target_hp) = (a.hp, a.mp, a.acted, t.hp);
        assert!(
            st.execute_native_compound_command32(Some(ACTOR), Some(TARGET), 1).is_err(),
            "command32 accepted an unproven input"
        );
        let (a, t) = (&st.units[ACTOR], &st.units[TARGET]);
        assert!(
            !(a.hp != actor_hp || a.mp != actor_mp || a.acted != actor_acted || t.hp != target_hp),
            "failed command32 mutated actor={a:?} target={t:?}"
        );
    }
}

/// Go：`TestPlanNativeCompoundCommand32DoesNotPublishState`
#[test]
fn plan_native_compound_command32_does_not_publish_state() {
    let mut st = native_compound32_fixture();
    let mut plan = st.plan_native_compound_command32(Some(ACTOR), Some(TARGET), 7).unwrap();
    let (a, t) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(
        plan.result.targets.len() == 1 && t.hp == 100 && !a.acted && a.mp == 80,
        "command32 plan published state: plan={plan:?} actor={a:?} target={t:?}"
    );
    apply_native_compound_command32_target(&mut st.units, Some(&mut plan), 0).unwrap();
    let (a, t) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(
        t.hp == plan.result.targets[0].hp_after && !a.acted,
        "command32 target boundary mismatch: plan={plan:?} actor={a:?} target={t:?}"
    );
    complete_native_compound_command32(&mut st.units, Some(&mut plan)).unwrap();
    let a = &st.units[ACTOR];
    assert!(a.acted && a.mp == 80, "command32 completion mismatch: actor={a:?}");
}

/// Go：`TestNativeCompoundCommand32PlanFailsClosedAcrossBoundaries`／`complete before target`
#[test]
fn native_compound_command32_plan_fails_closed_complete_before_target() {
    let mut st = native_compound32_fixture();
    let mut plan = st.plan_native_compound_command32(Some(ACTOR), Some(TARGET), 7).unwrap();
    assert!(
        complete_native_compound_command32(&mut st.units, Some(&mut plan)).is_err(),
        "command32 completed before publishing its target"
    );
    let (a, t) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(t.hp == 100 && !a.acted, "failed completion mutated actor={a:?} target={t:?}");
}

/// Go：`TestNativeCompoundCommand32PlanFailsClosedAcrossBoundaries`／`target changed`
#[test]
fn native_compound_command32_plan_fails_closed_target_changed() {
    let mut st = native_compound32_fixture();
    let mut plan = st.plan_native_compound_command32(Some(ACTOR), Some(TARGET), 7).unwrap();
    st.units[TARGET].hp -= 1;
    assert!(
        apply_native_compound_command32_target(&mut st.units, Some(&mut plan), 0).is_err(),
        "command32 published over changed target state"
    );
    let (a, t) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(t.hp == 99 && !a.acted, "failed publication changed actor={a:?} target={t:?}");
}

/// Go：`TestNativeCompoundCommand32PlanFailsClosedAcrossBoundaries`／`abort restores published target`
#[test]
fn native_compound_command32_plan_fails_closed_abort_restores_published_target() {
    let mut st = native_compound32_fixture();
    let mut plan = st.plan_native_compound_command32(Some(ACTOR), Some(TARGET), 7).unwrap();
    plan.result.targets[0].hp_after = 1; // Public report is not transaction authority.
    apply_native_compound_command32_target(&mut st.units, Some(&mut plan), 0).unwrap();
    assert!(st.units[TARGET].hp != 1, "command32 trusted a caller-mutated public result");
    abort_native_compound_command32(&mut st.units, Some(&mut plan)).unwrap();
    let (a, t) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(t.hp == 100 && !a.acted, "command32 rollback mismatch: actor={a:?} target={t:?}");
}

/// Go：`TestNativeCompoundCommand32PlanFailsClosedAcrossBoundaries`／`repeat and out of range publication`
#[test]
fn native_compound_command32_plan_fails_closed_repeat_and_out_of_range_publication() {
    let mut st = native_compound32_fixture();
    let mut plan = st.plan_native_compound_command32(Some(ACTOR), Some(TARGET), 7).unwrap();
    assert!(
        apply_native_compound_command32_target(&mut st.units, Some(&mut plan), -1).is_err(),
        "command32 accepted negative target index"
    );
    apply_native_compound_command32_target(&mut st.units, Some(&mut plan), 0).unwrap();
    let published_hp = st.units[TARGET].hp;
    assert!(
        apply_native_compound_command32_target(&mut st.units, Some(&mut plan), 0).is_err(),
        "command32 published a target twice"
    );
    let (a, t) = (&st.units[ACTOR], &st.units[TARGET]);
    assert!(t.hp == published_hp && !a.acted, "repeated publication changed actor={a:?} target={t:?}");
}
