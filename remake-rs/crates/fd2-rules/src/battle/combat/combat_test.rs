//! 移植自 `remake/internal/battle/combat_test.go`。
//!
//! `NextAIPlan` 的測試會先走 D 組的 native 規劃器（`nextNativeAIMode11Plan` 等），D 合併前停在
//! 那裡的 `todo!()`，所以標 `#[ignore = "等 D 合併"]`。

use super::*;
use crate::battle::magic::magic_test::{mk_unit, new_test_state};

/// Go `mkFighter`：建一個位置固定、攻防命中閃避暴擊皆可控的測試單位(供 AttackWithRNG 系列測試)。
#[allow(clippy::too_many_arguments)]
fn mk_fighter(camp: Camp, x: i64, y: i64, hp: i64, ap: i64, dp: i64, hit: i64, ev: i64, crit: i64) -> Unit {
    let mut u = mk_unit(camp, x, y, hp, 0);
    (u.ap, u.dp, u.hit, u.ev, u.crit_pct) = (ap, dp, hit, ev, crit);
    u
}

/// Go：`TestAttackWithRNG_HitRateDistribution`——命中率=(攻方HIT-守方EV)%(doc02 §4.1)。
/// HIT90-EV40=50%,固定多個 seed 應同時出現命中與 Miss 兩種結果;Miss 必為 Amount=0
/// 且不扣血,命中必為 Amount>0 且確實扣血。
#[test]
fn attack_with_rng_hit_rate_distribution() {
    let mut st = new_test_state();
    let (mut saw_hit, mut saw_miss) = (false, false);
    for seed in 1..=50 {
        st.units = vec![mk_fighter(OWN, 0, 0, 100, 50, 10, 90, 40, 0), mk_fighter(ENEMY, 1, 0, 100, 0, 10, 0, 0, 0)];
        let r = st.attack_with_rng(Some(0), 1, &mut GoMathRand::new(seed));
        let d = &st.units[1];
        if r.missed {
            saw_miss = true;
            assert_eq!(r.amount, 0, "seed {seed}: Miss 應 Amount=0");
            assert_eq!(d.hp, 100, "seed {seed}: Miss 不應扣血");
        } else {
            saw_hit = true;
            assert!(r.amount > 0, "seed {seed}: 命中應 Amount>0,got {}", r.amount);
            assert!(d.hp < 100, "seed {seed}: 命中應扣血,HP={}", d.hp);
        }
        assert!(st.units[0].acted, "seed {seed}: 攻方應標記已行動(miss 也算耗用行動)");
    }
    assert!(saw_hit && saw_miss, "50% 命中率應同時出現命中與 Miss:sawHit={saw_hit} sawMiss={saw_miss}");
}

/// Go：`TestAttackWithRNG_AlwaysMiss`——HIT-EV<=0(命中率算出 <=0%)依公式視為必定 miss,
/// 與 magic.go rollsHit 的「hit<=0 必中」特例不同語意(見 combat.go rollsHitPct 註解)。
#[test]
fn attack_with_rng_always_miss() {
    let mut st = new_test_state();
    for seed in 1..=10 {
        st.units = vec![
            mk_fighter(OWN, 0, 0, 100, 50, 0, 10, 0, 0),   // 攻方 HIT=10
            mk_fighter(ENEMY, 1, 0, 100, 0, 10, 0, 50, 0), // 守方 EV=50 → 10-50=-40%
        ];
        let r = st.attack_with_rng(Some(0), 1, &mut GoMathRand::new(seed));
        assert!(
            r.missed && r.amount == 0,
            "seed {seed}: 命中率<=0 應必定 Miss,got Missed={} Amount={}",
            r.missed,
            r.amount
        );
    }
}

/// Go：`TestAttackWithRNG_AlwaysHit`——HIT-EV>=100% 視為必中,不擲骰也不受 seed 影響。
#[test]
fn attack_with_rng_always_hit() {
    let mut st = new_test_state();
    for seed in 1..=10 {
        st.units = vec![mk_fighter(OWN, 0, 0, 100, 50, 0, 100, 0, 0), mk_fighter(ENEMY, 1, 0, 100, 0, 10, 0, 0, 0)];
        let r = st.attack_with_rng(Some(0), 1, &mut GoMathRand::new(seed));
        assert!(!r.missed, "seed {seed}: 命中率>=100% 應必中,got Missed=true");
    }
}

/// Go：`TestAttackWithRNG_Crit`——暴擊率 100% 時必定觸發,DP 先減半再套地形%(notes.md 公式順序)。
/// st.Cost 為 nil → TerrainAPDPPct 對任何座標回「正常地形」(AP+5%,DP-5%,與 MoveCost
/// 對 nil 的預設一致,見 terrain.go 註解),下列數字已把這個預設地形加成算進去:
///
///     AP=100 → 有效 100(無 buff)→ 地形 ×105/100 = 105
///     DP=100 → 暴擊減半 50 → 地形 ×95/100 = 47(50*95/100=47.5,整數除法取 47)
///     max = 105-47 = 58;實際傷害 = 58*0.9(=52.2→52) ~ 58-1(=57)
#[test]
fn attack_with_rng_crit() {
    let mut st = new_test_state();
    for seed in 1..=20 {
        st.units = vec![
            mk_fighter(OWN, 0, 0, 100, 100, 0, 100, 0, 100), // HIT100 必中,Crit100% 必暴擊
            mk_fighter(ENEMY, 1, 0, 200, 0, 100, 0, 0, 0),
        ];
        let r = st.attack_with_rng(Some(0), 1, &mut GoMathRand::new(seed));
        assert!(r.crit, "seed {seed}: CritPct=100 應必定暴擊");
        assert!((52..=57).contains(&r.amount), "seed {seed}: 暴擊傷害 = {}, want [52,57](見函式註解手算)", r.amount);
    }
}

/// Go：`TestAttackWithRNG_TerrainLowersAttackerDamage`——doc02 §3.2 沼澤 AP-5%(相對正常地形 AP+5%),
/// 攻方站在沼澤應比站在正常地形造成更低傷害(同一組固定 seed、相同攻防雙方數值,只變攻方座標
/// 對應的地形)。用 CritPct=0、HIT=100 排除暴擊/命中率的隨機干擾,只看地形這一個變數。
#[test]
fn attack_with_rng_terrain_lowers_attacker_damage() {
    for seed in 1..=10 {
        let mut normal_cost = vec![1; 25];
        let mut st_normal = State { w: 5, h: 5, cost: Some(normal_cost.clone()), ..Default::default() };
        normal_cost[0] = 2;
        let mut st_swamp = State { w: 5, h: 5, cost: Some(normal_cost), ..Default::default() };

        st_normal.units =
            vec![mk_fighter(OWN, 0, 0, 100, 100, 0, 100, 0, 0), mk_fighter(ENEMY, 1, 0, 300, 0, 20, 0, 0, 0)];
        let r_n = st_normal.attack_with_rng(Some(0), 1, &mut GoMathRand::new(seed));

        // 站 (0,0) = cost 2 = 沼澤
        st_swamp.units =
            vec![mk_fighter(OWN, 0, 0, 100, 100, 0, 100, 0, 0), mk_fighter(ENEMY, 1, 0, 300, 0, 20, 0, 0, 0)];
        let r_s = st_swamp.attack_with_rng(Some(0), 1, &mut GoMathRand::new(seed));

        assert!(r_s.amount < r_n.amount, "seed {seed}: 攻方站沼澤傷害({})應低於站正常地形({})", r_s.amount, r_n.amount);
    }
}

/// Go：`TestAttackWithRNG_DamageRandomizationRange`——實際傷害 = 最大傷害*0.9 ~ 最大傷害-1(doc02 §4.1),
/// AP=100/DP=20,套預設(nil Cost=正常地形 AP+5%/DP-5%)後 max=86,期望區間 [77,85]
/// (86*9/10=77.4→77,86-1=85);多 seed 跑,驗證都落在區間內、且低高兩端都會出現(非退化成固定值)。
#[test]
fn attack_with_rng_damage_randomization_range() {
    let mut st = new_test_state();
    let (mut seen_low, mut seen_high) = (false, false);
    for seed in 1..=100 {
        st.units = vec![mk_fighter(OWN, 0, 0, 100, 100, 0, 100, 0, 0), mk_fighter(ENEMY, 1, 0, 300, 0, 20, 0, 0, 0)];
        let r = st.attack_with_rng(Some(0), 1, &mut GoMathRand::new(seed));
        assert!((77..=85).contains(&r.amount), "seed {seed}: 傷害 = {}, want [77,85]", r.amount);
        if r.amount <= 79 {
            seen_low = true;
        }
        if r.amount >= 83 {
            seen_high = true;
        }
    }
    assert!(seen_low && seen_high, "100 個 seed 的傷害應涵蓋區間低端與高端,seenLow={seen_low} seenHigh={seen_high}");
}

/// Go：`TestAttackWithRNG_MinDamageFloor`——即使公式算出 max<=0(守方 DP 過高),命中的攻擊仍至少造成 1
/// (青衫「dmg<=2」是 AI 選標門檻,非玩家攻擊下限,見 combat.go 註解)。
#[test]
fn attack_with_rng_min_damage_floor() {
    let mut st = new_test_state();
    st.units = vec![mk_fighter(OWN, 0, 0, 100, 5, 0, 100, 0, 0), mk_fighter(ENEMY, 1, 0, 100, 0, 999, 0, 0, 0)];
    let r = st.attack_with_rng(Some(0), 1, &mut GoMathRand::new(1));
    assert!(!r.missed, "HIT=100 應必中");
    assert_eq!(r.amount, 1, "max<=0 時命中應保底 1");
}

/// Go：`TestAttack_BackwardCompat`——main.go 呼叫的舊介面 Attack(a,d) 回傳與 AttackWithRNG 一致的
/// Amount 語意(用必中/無暴擊/無地形干擾的簡單案例驗證扣血與回傳值一致)。
#[test]
fn attack_backward_compat() {
    let mut st = new_test_state();
    st.units = vec![mk_fighter(OWN, 0, 0, 100, 50, 10, 100, 0, 0), mk_fighter(ENEMY, 1, 0, 200, 0, 10, 0, 0, 0)];
    let before = st.units[1].hp;
    let amt = st.attack(Some(0), 1);
    assert!(amt > 0, "Attack 應回傳正傷害,got {amt}");
    assert_eq!(before - st.units[1].hp, amt, "扣血量應等於回傳傷害");
}

/// Go：`TestEstDamage_UsesTerrain`——AI 評分公式(doc11)套地形% 後,攻方站沼澤估算傷害應低於站正常地形
/// (呼應 doc42 gap-audit 第 5 項「AI 評分公式未套地形%」)。
///
/// Go 的 `u`、`t2` 不在 `State.Units`；Rust 依模組文件第 6 點先 push 再傳索引（`estDamage` 只讀兩個單位）。
#[test]
fn est_damage_uses_terrain() {
    let u = Unit { x: 0, y: 0, ap: 100, ..Default::default() };
    let t2 = Unit { x: 1, y: 0, dp: 20, ..Default::default() };
    let st_normal =
        State { w: 3, h: 1, cost: Some(vec![1, 1, 1]), units: vec![u.clone(), t2.clone()], ..Default::default() };
    let st_swamp = State { w: 3, h: 1, cost: Some(vec![2, 1, 1]), units: vec![u, t2], ..Default::default() };
    let dmg_normal = st_normal.est_damage(0, 1);
    let dmg_swamp = st_swamp.est_damage(0, 1);
    assert!(dmg_swamp < dmg_normal, "攻方站沼澤的 AI 估計傷害({dmg_swamp})應低於站正常地形({dmg_normal})");
}

/// Go：`TestNextAIPlan_SkipsLowDamageTarget`
#[test]
#[ignore = "等 D 合併"]
fn next_ai_plan_skips_low_damage_target() {
    let mut ai = mk_fighter(ENEMY, 0, 0, 100, 5, 0, 100, 0, 0);
    let mut target = mk_fighter(OWN, 1, 0, 100, 0, 20, 0, 0, 0);
    ai.mv = 3;
    (ai.on_field, target.on_field) = (true, true);
    let mut st = State { w: 5, h: 1, units: vec![ai, target], ..Default::default() };

    let plan = st.next_ai_plan();
    let Some(plan) = plan.filter(|p| p.u == Some(0)) else {
        panic!("expected deterministic plan for AI unit");
    };
    assert!(plan.target.is_none(), "estimated damage <=2 must skip attack target, got {:?}", plan.target);
    let got = st.est_damage(0, 1);
    assert!(got <= 2, "fixture damage must be at most two, got {got}");
}

/// Go：`TestNextAIPlan_PrefersDamageAboveThreshold`
#[test]
#[ignore = "等 D 合併"]
fn next_ai_plan_prefers_damage_above_threshold() {
    let mut ai = mk_fighter(ENEMY, 0, 0, 100, 50, 0, 100, 0, 0);
    let mut weak = mk_fighter(OWN, 0, 1, 100, 0, 100, 0, 0, 0); // low damage, nearer
    let mut good = mk_fighter(OWN, 4, 0, 100, 0, 10, 0, 0, 0); // viable damage
    ai.mv = 3;
    (ai.on_field, weak.on_field, good.on_field) = (true, true, true);
    let mut st = State { w: 6, h: 2, units: vec![ai, weak, good], ..Default::default() };
    let plan = st.next_ai_plan();
    assert!(
        plan.as_ref().is_some_and(|p| p.target == Some(2)),
        "AI should choose viable target over dmg<=2 target, got {plan:?}"
    );
}

/// Go：`TestNextAIPlan_DefaultSpellCommandIsDisabled`
#[test]
#[ignore = "等 D 合併"]
fn next_ai_plan_default_spell_command_is_disabled() {
    let mut ai = mk_fighter(ENEMY, 0, 0, 100, 50, 0, 100, 0, 0);
    let mut target = mk_fighter(OWN, 1, 0, 100, 0, 1, 0, 0, 0);
    (ai.on_field, target.on_field) = (true, true);
    let mut st = State {
        w: 2,
        h: 1,
        spell_book: vec![Spell { id: 7, name: "神雷".into(), ..Default::default() }],
        units: vec![ai, target],
        ..Default::default()
    };
    let plan = st.next_ai_plan();
    assert!(
        plan.as_ref().is_some_and(|p| p.spell_id == -1),
        "physical AI plan must not invent spell command: {plan:?}"
    );
    assert!(
        st.spell_book.len() == 1 && st.spell_book[0].id == 7,
        "injected spell book was mutated: {:?}",
        st.spell_book
    );
}

/// Go：`TestNextAIPlanSelectsEditableHealSpellBeforePhysicalFallback`
#[test]
#[ignore = "等 D 合併"]
fn next_ai_plan_selects_editable_heal_spell_before_physical_fallback() {
    let mut caster = mk_fighter(ENEMY, 0, 0, 100, 40, 0, 100, 0, 0);
    (caster.mp, caster.spells, caster.on_field) = (10, vec![13], true);
    let mut ally = mk_fighter(ENEMY, 1, 0, 30, 10, 20, 100, 0, 0);
    (ally.hp, ally.on_field) = (10, true);
    let mut target = mk_fighter(OWN, 2, 0, 30, 30, 0, 0, 0, 0);
    target.on_field = true;
    let mut st = State {
        w: 3,
        h: 1,
        spell_book: vec![Spell { id: 13, dmg: 70, dist: 3, mp: 5, target: 1, ..Default::default() }],
        units: vec![caster, ally, target],
        ..Default::default()
    };
    let plan = st.next_ai_plan();
    assert!(
        plan.as_ref().is_some_and(|p| p.spell_id == 13 && p.target == Some(1) && p.path.len() == 1),
        "spell plan={plan:?}, want heal ally without physical fallback"
    );
}

/// Go：`TestNextAIPlanApproachesEditableAttackSpellTarget`
#[test]
#[ignore = "等 D 合併"]
fn next_ai_plan_approaches_editable_attack_spell_target() {
    let mut caster = mk_fighter(ENEMY, 0, 0, 100, 40, 0, 100, 0, 0);
    (caster.mp, caster.mv, caster.spells, caster.on_field) = (5, 2, vec![0], true);
    let mut target = mk_fighter(OWN, 3, 0, 30, 30, 0, 0, 0, 0);
    target.on_field = true;
    let mut st = State {
        w: 4,
        h: 1,
        spell_book: vec![Spell { id: 0, dmg: 50, dist: 1, mp: 5, target: 0, ..Default::default() }],
        units: vec![caster, target],
        ..Default::default()
    };
    let plan = st.next_ai_plan();
    assert!(
        plan.as_ref().is_some_and(|p| p.spell_id == 0
            && p.target == Some(1)
            && p.path.len() == 3
            && p.path.last() == Some(&Cell { x: 2, y: 0 })),
        "spell approach plan={plan:?}, want path to cast range"
    );
}

/// Go：`TestNextAIPlanAllyTargetsEnemyWithEditableAttackSpell`
#[test]
#[ignore = "等 D 合併"]
fn next_ai_plan_ally_targets_enemy_with_editable_attack_spell() {
    let mut caster = mk_fighter(ALLY, 0, 0, 100, 40, 0, 100, 0, 0);
    (caster.mp, caster.spells, caster.on_field) = (5, vec![0], true);
    let mut friend = mk_fighter(OWN, 1, 0, 30, 30, 0, 0, 0, 0);
    friend.on_field = true;
    let mut target = mk_fighter(ENEMY, 2, 0, 30, 30, 0, 0, 0, 0);
    target.on_field = true;
    let mut st = State {
        w: 3,
        h: 1,
        spell_book: vec![Spell { id: 0, dmg: 50, dist: 2, mp: 5, target: 0, ..Default::default() }],
        units: vec![caster, friend, target],
        ..Default::default()
    };
    let plan = st.next_ai_plan();
    assert!(
        plan.as_ref().is_some_and(|p| p.spell_id == 0 && p.target == Some(2)),
        "ally spell plan={plan:?}, want enemy target"
    );
}

/// Go：`TestNextAIPlanUsesEditableInventorySpellMapping`
#[test]
#[ignore = "等 D 合併"]
fn next_ai_plan_uses_editable_inventory_spell_mapping() {
    let mut caster = mk_fighter(ENEMY, 0, 0, 100, 40, 0, 100, 0, 0);
    (caster.mp, caster.inventory, caster.on_field) = (5, vec![0x61], true);
    let mut target = mk_fighter(OWN, 1, 0, 30, 30, 0, 0, 0, 0);
    target.on_field = true;
    let mut st = State {
        w: 2,
        h: 1,
        spell_book: vec![Spell { id: 0, dmg: 50, dist: 1, mp: 5, target: 0, ..Default::default() }],
        ai_command_spell: BTreeMap::from([(0x61, 0)]),
        units: vec![caster, target],
        ..Default::default()
    };
    let plan = st.next_ai_plan();
    assert!(
        plan.as_ref().is_some_and(|p| p.spell_id == 0 && p.target == Some(1)),
        "inventory spell plan={plan:?}, want mapped editable spell"
    );
}

/// Go：`TestNextAIPlanSpellPathUsesStableCellTieBreak`
#[test]
#[ignore = "等 D 合併"]
fn next_ai_plan_spell_path_uses_stable_cell_tie_break() {
    let mut caster = mk_fighter(ENEMY, 0, 0, 100, 40, 0, 100, 0, 0);
    (caster.mp, caster.mv, caster.spells, caster.on_field) = (5, 2, vec![0], true);
    let mut target = mk_fighter(OWN, 3, 1, 30, 30, 0, 0, 0, 0);
    target.on_field = true;
    let mut st = State {
        w: 4,
        h: 2,
        spell_book: vec![Spell { id: 0, dmg: 50, dist: 2, mp: 5, target: 0, ..Default::default() }],
        units: vec![caster, target],
        ..Default::default()
    };
    let plan = st.next_ai_plan();
    assert!(
        plan.as_ref().is_some_and(|p| p.path.last() == Some(&Cell { x: 2, y: 0 })),
        "spell path={plan:?}, want stable endpoint (2,0)"
    );
}

/// Go：`TestNextAIPlanPreservesRawNativeScoredCommandsWithoutExecutingThem`
#[test]
#[ignore = "等 D 合併"]
fn next_ai_plan_preserves_raw_native_scored_commands_without_executing_them() {
    let actor = Unit {
        camp: ENEMY,
        on_field: true,
        hp: 10,
        max_hp: 10,
        mp: 5,
        x: 0,
        y: 0,
        native_command_mask: [1 << 2, 0, 0, 1 << 4, 0],
        ..Default::default()
    };
    let target = Unit { camp: OWN, on_field: true, hp: 10, max_hp: 10, x: 1, y: 0, ..Default::default() };
    let book: Vec<NativeCommandRecord> =
        (0..36).map(|id| NativeCommandRecord { id, mp_cost: 0, ..Default::default() }).collect();
    let mut st = State { w: 2, h: 1, units: vec![actor, target], native_command_book: book, ..Default::default() };
    let plan = st.next_ai_plan();
    assert!(
        plan.as_ref().is_some_and(|p| p.native_scored_commands == [2, 28]),
        "native scored commands = {plan:?}, want [2 28]"
    );
    assert_eq!(plan.unwrap().spell_id, -1, "raw command bridge must not execute or select spell");
}

/// Go：`TestNextAIPlanOmitsRawScoredCommandsWithoutVerifiedBook`
#[test]
#[ignore = "等 D 合併"]
fn next_ai_plan_omits_raw_scored_commands_without_verified_book() {
    let actor = Unit { camp: ENEMY, on_field: true, hp: 10, max_hp: 10, x: 0, y: 0, ..Default::default() };
    let target = Unit { camp: OWN, on_field: true, hp: 10, max_hp: 10, x: 1, y: 0, ..Default::default() };
    let plan = State { w: 2, h: 1, units: vec![actor, target], ..Default::default() }.next_ai_plan();
    assert!(
        plan.as_ref().is_some_and(|p| p.native_scored_commands.is_empty()),
        "missing native book must fail closed: {plan:?}"
    );
}

/// Go：`TestAIAvailableSpellsResolvesInventoryCommandsWithoutScoring`
///
/// Go 的 `u` 不在 `State.Units`；Rust 依模組文件第 6 點先 push 再傳索引（只讀該單位的物品欄）。
#[test]
fn ai_available_spells_resolves_inventory_commands_without_scoring() {
    let u = Unit { inventory: vec![79, 80, 79], ..Default::default() };
    let st = State {
        ai_command_spell: BTreeMap::from([(79, 15), (80, 99)]),
        spell_book: vec![Spell { id: 15, name: "回復".into(), ..Default::default() }],
        units: vec![u],
        ..Default::default()
    };
    let got = st.ai_available_spells(Some(0));
    assert!(got.len() == 1 && got[0].id == 15, "available spells={got:?}, want one resolved spell 15");
}

/// Go：`TestAISpellCandidatesUsesOriginalFamilyCampFilters`
#[test]
fn ai_spell_candidates_uses_original_family_camp_filters() {
    let caster = Unit { camp: ENEMY, on_field: true, hp: 10, max_hp: 10, ..Default::default() };
    let ally = Unit { camp: ENEMY, on_field: true, hp: 4, max_hp: 10, ..Default::default() };
    let dead_enemy = Unit { camp: OWN, on_field: true, hp: 0, max_hp: 10, ..Default::default() };
    let target = Unit { camp: OWN, on_field: true, hp: 10, max_hp: 10, ..Default::default() };
    let mut st = State { units: vec![caster, ally, dead_enemy, target], ..Default::default() };
    let (caster, ally, target) = (0, 1, 3);
    let cand = |st: &State, id| st.ai_spell_candidates(Some(caster), &Spell { id, ..Default::default() });
    assert_eq!(cand(&st, 0), vec![target], "attack candidates, want live opposing target");
    assert_eq!(cand(&st, 13), vec![ally], "heal candidates, want injured same-camp ally");
    assert_eq!(cand(&st, 17), vec![caster, ally], "buff candidates, want same-camp live units");
    st.units[ally].poisoned = true;
    assert_eq!(cand(&st, 20), vec![ally], "cure candidates, want poisoned same-camp ally");
    assert_eq!(cand(&st, 22), vec![target], "status candidates, want opposing target");
}
