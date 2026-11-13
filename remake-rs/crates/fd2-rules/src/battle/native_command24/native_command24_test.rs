//! 移植自 `remake/internal/battle/native_command24_test.go`。
//!
//! Go 以 `*Unit` 指標比較目標身份（`Target == near`），Rust 比較 `state.units` 的索引。

use super::*;
use crate::battle::native_ai_runtime::d_test_support::*;

/// Go `nativeCommand24Book`。
fn native_command24_book() -> Vec<NativeCommandRecord> {
    let mut book: Vec<NativeCommandRecord> =
        (0..36).map(|id| NativeCommandRecord { id, ..Default::default() }).collect();
    // Raw record24 is 00 00 00 05 01 16 00: selection=5, effect=1,
    // MP=22, target-code=0.  Damage/hit are intentionally unused here.
    book[24] = NativeCommandRecord {
        id: 24,
        selection_mode: 5,
        effect_mode: 1,
        mp_cost: 22,
        target_code: 0,
        ..Default::default()
    };
    book
}

fn unit(camp: Camp, x: i64, y: i64) -> Unit {
    Unit { camp, on_field: true, x, y, ..Default::default() }
}

fn state(w: i64, h: i64, units: Vec<Unit>, book: Vec<NativeCommandRecord>) -> State {
    State {
        w,
        h,
        units,
        native_composition_event_bytes: vec![0; (w * h) as usize],
        native_command_book: book,
        ..Default::default()
    }
}

/// Go：`TestPlanNativeCommand30MatchesStateOnlyClampAndDeath`
#[test]
fn plan_native_command30_matches_state_only_clamp_and_death() {
    for (ap, dp, hp) in [(1, 100, 20), (1000, 0, 5)] {
        let make_state = || {
            let a = Unit { hp: 40, ap, mp: 29, ..unit(OWN, 0, 0) };
            let b = Unit { hp, max_hp: hp, dp, has_native_record_byte5: true, ..unit(ENEMY, 1, 0) };
            let mut book = native_command24_book();
            book[30] = NativeCommandRecord { id: 30, selection_mode: 0x14, mp_cost: 24, ..Default::default() };
            state(2, 1, vec![a, b], book)
        };
        let mut st = make_state();
        let mut control = make_state();
        let (mut rng, mut crng) = (GoMathRand::new(30), GoMathRand::new(30));
        let plan = st.plan_native_command30(Some(0), Cell::default(), Cell { x: 1, y: 0 }, Some(&mut rng));
        let plan = match plan {
            Ok(p) if p.results.len() == 1 && st.units[0].mp == 29 && !st.units[0].acted && st.units[1].hp == hp => p,
            other => panic!("{ap}/{dp}/{hp}: planning mutation: {other:?}"),
        };
        let want = control.execute_native_command30(Some(0), Cell::default(), Cell { x: 1, y: 0 }, Some(&mut crng));
        assert!(
            matches!(&want, Ok(w) if plan.results[0].damage == w[0].damage && plan.results[0].amount == w[0].amount),
            "{ap}/{dp}/{hp}: old state-only result changed: {want:?}"
        );
        apply_native_command_derived_strike_mp(&mut st.units, Some(&plan)).unwrap();
        apply_native_command_derived_strike_target(&mut st.units, Some(&plan), 0).unwrap();
        complete_native_command_derived_strike(&mut st.units, Some(&plan)).unwrap();
        assert!(
            st.units[0] == control.units[0] && st.units[1] == control.units[1] && rng.int63() == crng.int63(),
            "{ap}/{dp}/{hp}: planned final HP/MP/death or RNG differs from state-only control"
        );
    }
}

/// Go：`TestNativeCommand29FormalSelfCenterKeepsEnemyListAndTransaction`
#[test]
fn native_command29_formal_self_center_keeps_enemy_list_and_transaction() {
    let Some(spells) = asset_path("spells.json") else { return };
    let book = load_native_command_records(&spells).unwrap();
    assert!(
        book[29].selection_mode == 0 && book[29].effect_mode == 2 && book[29].mp_cost == 26,
        "formal command29 record changed"
    );
    let raw = |u: Unit| Unit { has_native_record_byte5: true, ..u };
    let actor = raw(Unit { hp: 180, mp: 40, ap: 120, ..unit(OWN, 2, 2) });
    let near = raw(Unit { hp: 500, dp: 20, ..unit(ENEMY, 3, 2) });
    let edge = raw(Unit { hp: 500, dp: 25, ..unit(ENEMY, 4, 2) });
    let far = raw(Unit { hp: 500, ..unit(ENEMY, 5, 2) });
    let friend = raw(Unit { hp: 180, ..unit(OWN, 2, 3) });
    let inactive = raw(Unit { hp: 500, native_record_byte5: 1, ..unit(ENEMY, 2, 1) });
    let mut st = state(7, 5, vec![actor, near, edge, far, friend, inactive], book);
    let (near, edge) = (1usize, 2usize);
    let mut plan = None;
    for cursor in [Cell { x: 2, y: 2 }, Cell { x: 3, y: 2 }, Cell { x: 2, y: 3 }] {
        let p = st.plan_native_command29_at_cursor(Some(0), cursor, Some(&mut GoMathRand::new(7))).unwrap();
        assert!(
            p.results.len() == 2
                && p.results[0].target == Some(near)
                && p.results[1].target == Some(edge)
                && p.mp_after == 14,
            "confirmation cursor{cursor:?} changed fixed list={:?} MP={}",
            p.results,
            p.mp_after
        );
        plan = Some(p);
    }
    let plan = plan.unwrap();
    let u = &st.units;
    assert!(u[0].mp == 40 && !u[0].acted && u[near].hp == 500 && u[edge].hp == 500, "planning changed live units");
    apply_native_command_derived_strike_mp(&mut st.units, Some(&plan)).unwrap();
    for i in 0..plan.results.len() as i64 {
        apply_native_command_derived_strike_target(&mut st.units, Some(&plan), i).unwrap();
    }
    complete_native_command_derived_strike(&mut st.units, Some(&plan)).unwrap();
    let u = &st.units;
    assert!(
        u[0].mp == 14
            && u[0].acted
            && u[near].hp < 500
            && u[edge].hp < 500
            && u[3].hp == 500
            && u[4].hp == 180
            && u[5].hp == 500,
        "self-center publication changed excluded units or missed final transaction"
    );
}

/// Go：`TestNativeCommand29InvalidCursorKeepsUnitsAndRNG`
#[test]
fn native_command29_invalid_cursor_keeps_units_and_rng() {
    let Some(spells) = asset_path("spells.json") else { return };
    let book = load_native_command_records(&spells).unwrap();
    let cases = [
        ("outside-effect-field", Cell { x: 3, y: 0 }, false, false, false),
        ("outside-grid", Cell { x: -1, y: 0 }, false, false, false),
        ("missing-raw", Cell::default(), true, false, false),
        ("inactive-enemy", Cell::default(), false, true, false),
        ("insufficient-mp", Cell::default(), false, false, true),
    ];
    for (name, cursor, missing_raw, inactive, insufficient) in cases {
        let mut actor = Unit { hp: 180, mp: 40, ap: 120, has_native_record_byte5: true, ..unit(OWN, 0, 0) };
        let mut target = Unit { hp: 500, has_native_record_byte5: !missing_raw, ..unit(ENEMY, 1, 0) };
        if inactive {
            target.native_record_byte5 = 1;
        }
        if insufficient {
            actor.mp = 25;
        }
        let (before_actor, before_target) = (actor.clone(), target.clone());
        let st = state(4, 1, vec![actor, target], book.clone());
        let (mut rng, mut control) = (GoMathRand::new(7), GoMathRand::new(7));
        assert!(
            st.plan_native_command29_at_cursor(Some(0), cursor, Some(&mut rng)).is_err(),
            "{name}: invalid self-center admission accepted"
        );
        assert!(
            st.units[0] == before_actor && st.units[1] == before_target && rng.int63() == control.int63(),
            "{name}: rejected admission changed units or RNG"
        );
    }
    for id in [-1, 36, 255] {
        let st = State { native_command_book: book.clone(), ..Default::default() };
        assert!(
            st.plan_native_command_derived_strike(None, None, id, Some(&mut GoMathRand::new(7))).is_err(),
            "invalid command accepted: {id}"
        );
    }
}

/// Go：`TestExecuteNativeCommand24UsesDerivedStatsAndOneMPDebit`
#[test]
fn execute_native_command24_uses_derived_stats_and_one_mp_debit() {
    let actor = Unit { ap: 100, mp: 30, ..unit(OWN, 0, 0) };
    let target = Unit { dp: 20, hp: 200, ..unit(ENEMY, 1, 0) };
    let mut st = state(2, 1, vec![actor, target], native_command24_book());
    let got = st.execute_native_command24(Some(0), Some(1), Some(&mut GoMathRand::new(2)));
    let got = match got {
        Ok(g) if g.len() == 1 && g[0].target == Some(1) && g[0].amount == 130 && (117..=129).contains(&g[0].damage) => {
            g
        }
        other => panic!("ID24 result={other:?}"),
    };
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        target.hp == 200 - got[0].damage && actor.mp == 8 && actor.acted,
        "ID24 state actor={actor:?} target={target:?} result={:?}",
        got[0]
    );
}

/// Go：`TestExecuteNativeCommand24RejectsInvalidBookBeforeMutation`
#[test]
fn execute_native_command24_rejects_invalid_book_before_mutation() {
    let actor = Unit { camp: OWN, on_field: true, mp: 30, ..Default::default() };
    let target = Unit { camp: ENEMY, on_field: true, hp: 100, ..Default::default() };
    let mut book = native_command24_book();
    book[24].id = 0;
    let mut st = state(1, 1, vec![actor, target], book);
    let err = st.execute_native_command24(Some(0), Some(1), Some(&mut GoMathRand::new(1)));
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        err.is_err() && actor.mp == 30 && !actor.acted && target.hp == 100,
        "invalid ID24 book mutated state actor={actor:?} target={target:?} err={err:?}"
    );
}

/// Go：`TestPlanNativeCommand24PublishesMPDamageAndCompletionAtSeparateMarkers`
#[test]
fn plan_native_command24_publishes_mp_damage_and_completion_at_separate_markers() {
    let actor = Unit { ap: 100, mp: 30, ..unit(OWN, 0, 0) };
    let target = Unit { dp: 20, hp: 200, ..unit(ENEMY, 1, 0) };
    let mut st = state(2, 1, vec![actor, target], native_command24_book());
    let plan = st.plan_native_command_derived_strike(Some(0), Some(1), 24, Some(&mut GoMathRand::new(2)));
    let plan = match plan {
        Ok(p) if p.results.len() == 1 => p,
        other => panic!("ID24 plan={other:?}"),
    };
    let check = |st: &State, mp: i64, acted: bool, hp: i64, what: &str| {
        let (actor, target) = (&st.units[0], &st.units[1]);
        assert!(actor.mp == mp && actor.acted == acted && target.hp == hp, "{what} actor={actor:?} target={target:?}");
    };
    check(&st, 30, false, 200, "planning mutated public state");
    apply_native_command_derived_strike_mp(&mut st.units, Some(&plan)).unwrap();
    check(&st, 8, false, 200, "actor marker published wrong state");
    apply_native_command_derived_strike_target(&mut st.units, Some(&plan), 0).unwrap();
    check(&st, 8, false, plan.results[0].hp_after, "target marker published wrong state");
    let err = complete_native_command_derived_strike(&mut st.units, Some(&plan));
    assert!(err.is_ok() && st.units[0].acted, "completion err={err:?} actor={:?}", st.units[0]);
}

/// Go：`TestPlanNativeCommand24PreservesDamageClampAndDeathProjection`
#[test]
fn plan_native_command24_preserves_damage_clamp_and_death_projection() {
    let actor = Unit { ap: 100, mp: 30, ..unit(OWN, 0, 0) };
    let target = Unit { hp: 5, max_hp: 5, has_native_record_byte5: true, native_record_byte5: 0, ..unit(ENEMY, 1, 0) };
    let mut st = state(2, 1, vec![actor, target], native_command24_book());
    let plan = st.plan_native_command_derived_strike(Some(0), Some(1), 24, Some(&mut GoMathRand::new(1))).unwrap();
    let got = plan.results[0].damage;
    assert!(got == 5, "clamped damage={got} want 5");
    apply_native_command_derived_strike_mp(&mut st.units, Some(&plan)).unwrap();
    apply_native_command_derived_strike_target(&mut st.units, Some(&plan), 0).unwrap();
    let target = &st.units[1];
    assert!(
        target.hp == 0 && target.native_record_byte5 == 1,
        "death projection HP={} raw+5={:02x}",
        target.hp,
        target.native_record_byte5
    );
}

/// Go：`TestPlanNativeCommand24DoesNotTurnNegativeDamageIntoHealing`
#[test]
fn plan_native_command24_does_not_turn_negative_damage_into_healing() {
    let actor = Unit { ap: 1, mp: 30, ..unit(OWN, 0, 0) };
    let target = Unit { dp: 100, hp: 20, max_hp: 20, ..unit(ENEMY, 1, 0) };
    let st = state(2, 1, vec![actor, target], native_command24_book());
    let plan = st.plan_native_command_derived_strike(Some(0), Some(1), 24, Some(&mut GoMathRand::new(1))).unwrap();
    let result = plan.results[0];
    assert!(result.damage == 0 && result.hp_after == 20, "negative raw damage changed HP: {result:?}");
}

/// Go：`TestExecuteNativeCommandDerivedStrikeUsesRecoveredID28MultiplierAndDenominator`
#[test]
fn execute_native_command_derived_strike_uses_recovered_id28_multiplier_and_denominator() {
    let actor = Unit { ap: 100, mp: 30, battle_fig: 19, has_battle_fig: true, ..unit(OWN, 0, 0) };
    let target = Unit { dp: 20, hp: 300, ..unit(ENEMY, 1, 0) };
    let mut book = native_command24_book();
    book[28] = NativeCommandRecord {
        id: 28,
        selection_mode: 1,
        effect_mode: 0,
        mp_cost: 22,
        target_code: 0,
        ..Default::default()
    };
    let mut st = state(2, 1, vec![actor, target], book);
    let got = st.execute_native_command_derived_strike(Some(0), Some(1), 28, Some(&mut GoMathRand::new(2)));
    let got = match got {
        Ok(g) if g.len() == 1 && g[0].amount == 180 && (162..=179).contains(&g[0].damage) => g,
        other => panic!("ID28 result={other:?}"),
    };
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        actor.mp == 8 && actor.acted && target.hp == 300 - got[0].damage,
        "ID28 state actor={actor:?} target={target:?} result={:?}",
        got[0]
    );
}

/// Go：`TestNativeCommand28EightStagesClampBeforeDivision`
#[test]
fn native_command28_eight_stages_clamp_before_division() {
    let actor = Unit {
        ap: 100,
        mp: 30,
        battle_fig: 19,
        has_battle_fig: true,
        has_native_record_byte5: true,
        ..unit(OWN, 0, 0)
    };
    let target = Unit { dp: 20, hp: 7, has_native_record_byte5: true, ..unit(ENEMY, 1, 0) };
    let mut book = native_command24_book();
    book[28] = NativeCommandRecord {
        id: 28,
        selection_mode: 1,
        effect_mode: 0,
        mp_cost: 22,
        target_code: 0,
        ..Default::default()
    };
    let mut st = state(2, 1, vec![actor, target], book);
    let plan = st.plan_native_command_derived_strike(Some(0), Some(1), 28, Some(&mut GoMathRand::new(2)));
    let mut plan = match plan {
        Ok(p)
            if p.target_impact_count == 8
                && p.results[0].damage == 7
                && st.units[1].hp == 7
                && st.units[0].mp == 30 =>
        {
            p
        }
        other => panic!("capped stage plan={other:?}"),
    };
    assert!(
        apply_native_command28_target_stage(&mut st.units, Some(&mut plan), 0, 1).is_err() && st.units[1].hp == 7,
        "target impact preceded MP"
    );
    apply_native_command_derived_strike_mp(&mut st.units, Some(&plan)).unwrap();
    assert!(
        apply_native_command28_target_stage(&mut st.units, Some(&mut plan), 0, 2).is_err() && st.units[1].hp == 7,
        "target impact skipped a stage"
    );
    for stage in 1..=8 {
        apply_native_command28_target_stage(&mut st.units, Some(&mut plan), 0, stage).unwrap();
        let (actor, target) = (&st.units[0], &st.units[1]);
        assert!(
            target.hp == 7 - 7 * stage / 8 && actor.mp == 8 && !actor.acted,
            "stage{stage}: HP{} MP{} acted{}",
            target.hp,
            actor.mp,
            actor.acted
        );
        assert!(
            apply_native_command28_target_stage(&mut st.units, Some(&mut plan), 0, stage).is_err(),
            "duplicate marker published twice"
        );
        if stage < 8 {
            assert!(
                st.units[1].native_record_byte5 == 0
                    && complete_native_command_derived_strike(&mut st.units, Some(&plan)).is_err(),
                "command28 completed or killed target before eight impacts"
            );
        }
    }
    let err = complete_native_command_derived_strike(&mut st.units, Some(&plan));
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        err.is_ok() && actor.acted && target.hp == 0 && target.native_record_byte5 == 1,
        "final stage did not complete: {err:?}"
    );
}

/// Go：`TestNativeCommand28UnprovenSelectorRejectsBeforeRNG`
#[test]
fn native_command28_unproven_selector_rejects_before_rng() {
    let actor =
        Unit { camp: OWN, on_field: true, ap: 100, mp: 30, battle_fig: 20, has_battle_fig: true, ..Default::default() };
    let target = Unit { hp: 300, ..unit(ENEMY, 1, 0) };
    let mut book = native_command24_book();
    book[28] = NativeCommandRecord { id: 28, selection_mode: 1, mp_cost: 22, ..Default::default() };
    let st = state(2, 1, vec![actor, target], book);
    let (mut rng, mut control) = (GoMathRand::new(2), GoMathRand::new(2));
    let err = st.plan_native_command_derived_strike(Some(0), Some(1), 28, Some(&mut rng));
    assert!(
        err.is_err()
            && st.units[0].mp == 30
            && !st.units[0].acted
            && st.units[1].hp == 300
            && rng.int63() == control.int63(),
        "unproven command28 source reached RNG or state publication"
    );
}

/// Go：`TestExecuteNativeCommandDerivedStrikeUsesRecoveredID29And31Multipliers`
#[test]
fn execute_native_command_derived_strike_uses_recovered_id29_and31_multipliers() {
    for (id, multiplier) in [(29, 12), (31, 18)] {
        let actor = Unit { ap: 100, mp: 40, ..unit(OWN, 0, 0) };
        let target = Unit { dp: 20, hp: 300, ..unit(ENEMY, 1, 0) };
        let mut book = native_command24_book();
        book[id as usize] = NativeCommandRecord {
            id,
            selection_mode: 1,
            effect_mode: 0,
            mp_cost: 26,
            target_code: 0,
            ..Default::default()
        };
        let mut st = state(2, 1, vec![actor, target], book);
        let got = st.execute_native_command_derived_strike(Some(0), Some(1), id, Some(&mut GoMathRand::new(2)));
        let want_amount = st.units[0].ap * multiplier / 10 - st.units[1].dp;
        let got = match got {
            Ok(g) if g.len() == 1 && g[0].amount == want_amount && g[0].damage > 0 => g,
            other => panic!("id{id}: result={other:?} want amount={want_amount}"),
        };
        let (actor, target) = (&st.units[0], &st.units[1]);
        assert!(
            actor.mp == 14 && actor.acted && target.hp == 300 - got[0].damage,
            "id{id}: state actor={actor:?} target={target:?} result={:?}",
            got[0]
        );
    }
}

/// Go：`TestExecuteNativeCommandDerivedStrikeRejectsSpecialSelectorID30`
///
/// Go 的 actor 不在 `State.Units` 裡；Rust 依 `battle.rs` 第 6 點先放進 `units` 再傳索引。
#[test]
fn execute_native_command_derived_strike_rejects_special_selector_id30() {
    let mut st = State { units: vec![Unit { mp: 30, ..Default::default() }], ..Default::default() };
    let err = st.execute_native_command_derived_strike(Some(0), None, 30, Some(&mut GoMathRand::new(1)));
    let actor = &st.units[0];
    assert!(
        err.is_err() && actor.mp == 30 && !actor.acted,
        "generic derived-strike executor must reject ID30 special selector: actor={actor:?} err={err:?}"
    );
}

/// Go：`TestNativeCommand30TargetsUsesSavedCursorAndXPriority`
#[test]
fn native_command30_targets_uses_saved_cursor_and_x_priority() {
    let x_line = Unit { hp: 1, ..unit(ENEMY, 1, 0) };
    let y_diagonal = Unit { hp: 1, ..unit(ENEMY, 0, 1) };
    let got = native_command30_targets(5, 3, Cell { x: 0, y: 0 }, Cell { x: 2, y: 2 }, 4, &[x_line, y_diagonal]);
    assert!(matches!(&got, Ok(g) if g == &[0]), "ID30 X-priority targets={got:?}");
    let plus_y = Unit { hp: 1, ..unit(ENEMY, 1, 2) };
    let got = native_command30_targets(3, 4, Cell { x: 1, y: 1 }, Cell { x: 1, y: 1 }, 1, &[plus_y]);
    assert!(matches!(&got, Ok(g) if g == &[0]), "ID30 equal-cursor must follow native +Y branch: targets={got:?}");
}

/// Go：`TestExecuteNativeCommand30UsesSpecialLineAndOneMPDebit`
#[test]
fn execute_native_command30_uses_special_line_and_one_mp_debit() {
    let actor = Unit { ap: 100, mp: 30, ..unit(OWN, 0, 0) };
    let between = Unit { dp: 20, hp: 300, ..unit(ENEMY, 2, 0) };
    let confirmed = Unit { dp: 20, hp: 300, ..unit(ENEMY, 4, 0) };
    let friendly = Unit { dp: 20, hp: 300, ..unit(OWN, 3, 0) };
    let mut book = native_command24_book();
    book[30] = NativeCommandRecord {
        id: 30,
        selection_mode: 20,
        effect_mode: 0,
        mp_cost: 24,
        target_code: 0,
        ..Default::default()
    };
    let mut st = state(5, 1, vec![actor, between, confirmed, friendly], book);
    let got =
        st.execute_native_command30(Some(0), Cell { x: 0, y: 0 }, Cell { x: 4, y: 0 }, Some(&mut GoMathRand::new(2)));
    assert!(
        matches!(&got, Ok(g) if g.len() == 2
            && g[0].target == Some(1)
            && g[1].target == Some(2)
            && g[0].amount == 160
            && g[1].amount == 160),
        "ID30 result={got:?}"
    );
    let u = &st.units;
    assert!(
        u[0].mp == 6 && u[0].acted && u[3].hp == 300 && u[1].hp < 300 && u[2].hp < 300,
        "ID30 mutation actor={:?} between={:?} confirmed={:?} friendly={:?}",
        u[0],
        u[1],
        u[2],
        u[3]
    );
}

/// Go：`TestExecuteNativeCommand30RejectsNonCandidateBeforeMP`
#[test]
fn execute_native_command30_rejects_non_candidate_before_mp() {
    let actor = Unit { ap: 100, mp: 30, ..unit(OWN, 0, 0) };
    let target = Unit { dp: 20, hp: 300, ..unit(ENEMY, 4, 0) };
    let mut book = native_command24_book();
    book[30] = NativeCommandRecord { id: 30, selection_mode: 20, mp_cost: 24, target_code: 0, ..Default::default() };
    let mut st = state(5, 1, vec![actor, target], book);
    let err =
        st.execute_native_command30(Some(0), Cell { x: 0, y: 0 }, Cell { x: 1, y: 0 }, Some(&mut GoMathRand::new(2)));
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        err.is_err() && actor.mp == 30 && !actor.acted && target.hp == 300,
        "invalid ID30 confirmation mutated state actor={actor:?} target={target:?} err={err:?}"
    );
}
