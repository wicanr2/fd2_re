//! 移植自 `remake/internal/battle/balance_test.go`。

use super::*;
use crate::test_support::docs_file;

/// Go：`TestBalanceGrowthKeepsOriginalRollSequenceAndZeroGrowth`
///
/// Go 的 a～e 是不在 `st.Units` 的區域單位；Rust 依模組文件第 6 點先 push 再傳索引
/// （`AwardExpNative`／`GainExp` 只動傳入的單位）。
#[test]
fn balance_growth_keeps_original_roll_sequence_and_zero_growth() {
    let Some(path) = docs_file("data/exe_tables/native_growth_slots.json") else { return };
    let rows = load_native_growth_slots(&path).unwrap();
    let base = Unit {
        camp: OWN,
        native_identity: 4,
        has_native_identity: true,
        battle_fig: 4,
        has_battle_fig: true,
        lv: 2,
        hp: 10,
        max_hp: 40,
        ap: 20,
        dp: 10,
        dx: 3,
        ..Default::default()
    };
    let mut st = State {
        native_growth_rows: Some(rows),
        units: vec![base.clone(), base.clone(), base.clone()],
        ..Default::default()
    };
    let (a, b, c) = (0, 1, 2);
    let (_, old, seed_a) = st.award_exp_native(Some(a), 100, 7);
    st.balance = Some(new_balance_options());
    let (_, same, seed_b) = st.award_exp_native(Some(b), 100, 7);
    assert!(st.units[a] == st.units[b] && old == same && seed_a == seed_b, "100% changed native growth or RNG");
    st.balance.as_mut().unwrap().growth = GrowthRates { hp: 200, mp: 500, ap: 0, dp: 150, dx: 300 };
    let (_, changed, seed_c) = st.award_exp_native(Some(c), 100, 7);
    assert!(
        !(seed_c != seed_a
            || changed.len() != 1
            || changed[0].ap_gain != 0
            || changed[0].hp_gain != old[0].hp_gain * 2
            || changed[0].mp_gain != 0
            || st.units[c].hp != 10),
        "growth/RNG/heal changed: {old:?} {changed:?}"
    );
    let mut d = base;
    d.battle_fig = 36; // Promotion changes the growth selector, not the identity.
    st.balance.as_mut().unwrap().characters = BTreeMap::from([(4, original_growth_rates())]);
    st.units.push(d.clone());
    st.units.push(d);
    let (d, e) = (3, 4);
    st.gain_exp(Some(d), 100.0, &mut GoMathRand::new(7));
    st.balance = None;
    st.gain_exp(Some(e), 100.0, &mut GoMathRand::new(7));
    assert!(st.units[d] == st.units[e], "character override lost after promotion");
}

/// Go `nativeCommand0TestState`（`native_command0_test.go`；測試輔助不跨模組共用，這裡複製一份）：
/// actor、target 是 `st.units` 的 0、1。
fn native_command0_test_state() -> State {
    let mut book: Vec<NativeCommandRecord> =
        (0..36).map(|id| NativeCommandRecord { id, ..Default::default() }).collect();
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

/// Go：`TestBalanceHitChangesActualPlanOnlyForSelectedCharacter`
#[test]
fn balance_hit_changes_actual_plan_only_for_selected_character() {
    let mut st = native_command0_test_state();
    let (actor, target) = (0, 1);
    st.units[actor].native_identity = 9;
    st.units[actor].has_native_identity = true;
    st.native_command_book[0].hit = 0;
    let mut balance = new_balance_options();
    balance.skill_hit = BTreeMap::from([(9, BTreeMap::from([(0, 100)]))]);
    st.balance = Some(balance);
    let resist = BTreeMap::from([(5, 10)]);
    let plan = st.plan_native_command_damage(Some(actor), Some(target), 0, &resist, 7);
    assert!(
        matches!(&plan, Ok(p) if p.results[0].native_command_damage.hit && p.results[0].native_command_damage.damage > 0)
            && st.units[target].hp == 100
            && st.units[actor].mp == 3,
        "custom hit plan {plan:?}"
    );
    st.units[actor].native_identity = 10;
    let plan = st.plan_native_command_damage(Some(actor), Some(target), 0, &resist, 7);
    assert!(
        matches!(&plan, Ok(p) if !p.results[0].native_command_damage.hit),
        "hit override leaked to another character"
    );
    assert!(!native_command_has_hit_roll(30), "音速刃 acquired invented hit roll");
    let balance = st.balance.as_mut().unwrap();
    balance.skill_hit.get_mut(&9).unwrap().insert(30, 100);
    assert!(balance.validate().is_err(), "accepted fabricated command30 hit setting");
}

/// Go：`TestBalanceEnemyScalingIsIdempotentAndPreservesWounds`
#[test]
fn balance_enemy_scaling_is_idempotent_and_preserves_wounds() {
    let u = Unit {
        camp: ENEMY,
        hp: 30,
        max_hp: 100,
        mp: 10,
        max_mp: 20,
        ap: 65,
        dp: 40,
        dx: 10,
        hit: 20,
        ev: 15,
        equipment_base_set: true,
        base_ap: 50,
        base_dp: 30,
        base_hit: 20,
        base_ev: 15,
        ..Default::default()
    };
    let mut st = State { units: vec![u], balance: Some(new_balance_options()), ..Default::default() };
    st.balance.as_mut().unwrap().enemy_percent = 200;
    st.apply_enemy_strength();
    let u = &st.units[0];
    assert!(
        !(u.max_hp != 200
            || u.hp != 60
            || u.max_mp != 40
            || u.mp != 20
            || u.ap != 115
            || u.dp != 70
            || u.dx != 20
            || u.hit != 30),
        "scaled unit {u:?}"
    );
    let copy = st.units[0].clone();
    st.apply_enemy_strength();
    assert!(copy == st.units[0], "strength applied twice");
    st.units[0].hp = 0;
    st.balance.as_mut().unwrap().enemy_percent = 50;
    st.apply_enemy_strength();
    let u = &st.units[0];
    assert!(!(u.hp != 0 || u.max_hp != 50 || u.ap != 40 || u.dx != 5), "strength revived or compounded");
    st.units.push(Unit { camp: ENEMY, hp: 40, max_hp: 40, ap: 10, dp: 8, dx: 6, ..Default::default() });
    let spawn = 1;
    st.apply_enemy_strength();
    assert!(!(st.units[spawn].max_hp != 20 || st.units[spawn].hp != 20), "new spawn omitted");
    st.balance = None;
    st.apply_enemy_strength();
    assert!(
        !(st.units[spawn].max_hp != 40 || st.units[0].ap != 65 || st.units[0].hp != 0),
        "reset did not restore baseline"
    );
}
