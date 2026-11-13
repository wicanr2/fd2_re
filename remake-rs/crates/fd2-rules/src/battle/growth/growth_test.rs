//! 移植自 `remake/internal/battle/growth_test.go`。

use super::*;
use crate::battle::magic::magic_test::new_test_state;

// ---- 升級成長(growthTable × GainExp)----

/// Go：`TestGainExp_CrossesThreshold_MultiLevel`——一次給的經驗值跨兩個 100 門檻,應連續升 2 級,
/// 每級成長量落在 growthTable["索爾"]["劍士"] 的範圍內(doc02 §7.2:AP 6-7、DP 4-5、
/// DX 2-2、HP 8-11、MP 0-0),且屬性確實被加進 Unit。
#[test]
fn gain_exp_crosses_threshold_multi_level() {
    let mut u = Unit {
        camp: OWN,
        name: "索爾".into(),
        cls_name: "劍士".into(),
        lv: 1,
        ap: 6,
        dp: 4,
        dx: 2,
        hp: 42,
        max_hp: 42,
        mp: 0,
        ..Default::default()
    };
    let mut rng = GoMathRand::new(1);
    let events = gain_exp(Some(&mut u), 250.0, &mut rng); // 250 = 2 級整 + 50 剩餘

    assert_eq!(events.len(), 2, "升級事件數");
    assert_eq!(u.lv, 3, "Lv");
    assert_eq!(u.exp, 50.0, "剩餘經驗");
    let (mut want_ap_gain, mut want_dp_gain, mut want_hp_gain) = (0, 0, 0);
    for (i, ev) in events.iter().enumerate() {
        assert!(
            (6..=7).contains(&ev.ap_gain),
            "event[{i}].ApGain = {}, 應落在 [6,7](doc02 §7.2 索爾/劍士)",
            ev.ap_gain
        );
        assert!((4..=5).contains(&ev.dp_gain), "event[{i}].DpGain = {}, 應落在 [4,5]", ev.dp_gain);
        assert_eq!(ev.dx_gain, 2, "event[{i}].DxGain, want 2(固定值)");
        assert!((8..=11).contains(&ev.hp_gain), "event[{i}].HpGain = {}, 應落在 [8,11]", ev.hp_gain);
        assert_eq!(ev.mp_gain, 0, "event[{i}].MpGain, want 0(索爾劍士無 MP 成長)");
        want_ap_gain += ev.ap_gain;
        want_dp_gain += ev.dp_gain;
        want_hp_gain += ev.hp_gain;
    }
    assert_eq!(u.ap, 6 + want_ap_gain, "AP");
    assert_eq!(u.dp, 4 + want_dp_gain, "DP");
    // 原版升級只加 MaxHP（0x1E529 寫 +0x42），目前 HP +0x40 不變。
    assert!(
        u.max_hp == 42 + want_hp_gain && u.hp == 42,
        "HP/MaxHP = {}/{}, want 42/{}(升級不補目前 HP)",
        u.hp,
        u.max_hp,
        42 + want_hp_gain
    );
}

/// Go：`TestGainExp_DXUpdatesDerivedHitAndEV`——The EXE derives both HIT and EV from the same DX
/// accumulator, then adds equipped item contributions.  A level-up must therefore move both derived
/// values by the exact DX growth amount, without losing the equipment delta.
#[test]
fn gain_exp_dx_updates_derived_hit_and_ev() {
    let mut u = Unit {
        camp: OWN,
        name: "索爾".into(),
        cls_name: "劍士".into(),
        lv: 1,
        ap: 16,
        dp: 12,
        dx: 2,
        hit: 97,
        ev: 2,
        base_hit: 2,
        base_ev: 2,
        equipment_base_set: true,
        ..Default::default()
    };
    gain_exp(Some(&mut u), 100.0, &mut GoMathRand::new(1));
    assert_eq!(u.dx, 4, "DX");
    assert!(
        u.hit == 99 && u.ev == 4 && u.base_hit == 4 && u.base_ev == 4,
        "DX synthesis lost equipment/base: HIT={} EV={} base={}/{}",
        u.hit,
        u.ev,
        u.base_hit,
        u.base_ev
    );
}

/// Go：`TestGainExp_EnemyNoop`——Enemy 不會累積經驗值/升級(doc42 gap 範圍只涵蓋玩家停滯問題,
/// 敵方無成長曲線資料,見 growth.go 檔頭說明)。
#[test]
fn gain_exp_enemy_noop() {
    let mut u = Unit { camp: ENEMY, cls_name: "戰士".into(), lv: 5, hp: 50, max_hp: 50, ..Default::default() };
    let events = gain_exp(Some(&mut u), 999.0, &mut GoMathRand::new(1));
    assert!(events.is_empty(), "Enemy 不應升級,got {} 筆事件", events.len());
    assert!(u.lv == 5 && u.exp == 0.0, "Enemy 狀態不應被改動,Lv={} Exp={}", u.lv, u.exp);
}

/// Go：`TestGainExp_UnknownGrowth_StillLevelsNoStatGain`——growthTable 查無資料(如友軍雜兵無名單位)
/// 時等級仍照門檻演進(避免經驗卡死),但不套用屬性成長(誠實反映缺資料,不是靜默丟棄)。
#[test]
fn gain_exp_unknown_growth_still_levels_no_stat_gain() {
    let mut u = Unit {
        camp: ALLY,
        name: "無名雜兵".into(),
        cls_name: "士兵".into(),
        lv: 1,
        ap: 10,
        dp: 10,
        hp: 50,
        max_hp: 50,
        ..Default::default()
    };
    let events = gain_exp(Some(&mut u), 100.0, &mut GoMathRand::new(1));
    assert!(events.is_empty(), "查無成長資料不應有 LevelUpEvent,got {}", events.len());
    assert_eq!(u.lv, 2, "Lv 仍應演進為 2");
    assert!(
        u.ap == 10 && u.dp == 10 && u.max_hp == 50,
        "查無成長資料不應改動屬性,got AP={} DP={} MaxHP={}",
        u.ap,
        u.dp,
        u.max_hp
    );
}

/// Go：`TestGainExp_MultiClassPromotion` 驗證 growthTable 的同角色多職業列。正式教會轉職
/// 已由 campaign／UI owner 實作；本測試只確認索爾「英雄」能查到獨立於「劍士」的成長列。
#[test]
fn gain_exp_multi_class_promotion() {
    let mut u = Unit {
        camp: OWN,
        name: "索爾".into(),
        cls_name: "英雄".into(),
        lv: 20,
        ap: 100,
        hp: 200,
        max_hp: 200,
        ..Default::default()
    };
    let events = gain_exp(Some(&mut u), 100.0, &mut GoMathRand::new(7));
    assert_eq!(events.len(), 1, "升級事件數");
    let ev = &events[0];
    // doc02 §7.2 索爾/英雄 AP 10-14
    assert!((10..=14).contains(&ev.ap_gain), "英雄職業 ApGain = {}, 應落在 [10,14]", ev.ap_gain);
}

// ---- 經驗值公式(doc02 §4.5)手算案例 ----

/// Go：`TestAttackExp_HandCalc`——攻方Lv10 打 守方Lv20(每級經驗30),造成守方總HP的一半傷害。
/// 手算:ratio=0.5;(20×30)=600;(20/10)=2 → 0.5×600×2=600。
#[test]
fn attack_exp_hand_calc() {
    assert_eq!(attack_exp(10, 20, 50, 100, 30), 600.0, "AttackExp");
}

/// Go：`TestAttackExp_LethalCapsAtTotalHP`——doc02「致死視同傷害HP=總HP」——即使呼叫端傳入的
/// dmgHP 超過總HP(overkill),也應等效於總HP,不會算出超過應有值的經驗。
#[test]
fn attack_exp_lethal_caps_at_total_hp() {
    let got = attack_exp(10, 20, 150, 100, 30); // dmgHP(150) > defTotalHP(100)
    let want = attack_exp(10, 20, 100, 100, 30);
    assert_eq!(got, want, "overkill 應等同 dmgHP=defTotalHP");
}

/// Go：`TestAttackExp_MissingExpPerLevel_ZerosOut`——defExpPerLv<=0(資料尚未接線,見 Unit.ExpPerLevel
/// 註解)時誠實回 0,不假造經驗值。
#[test]
fn attack_exp_missing_exp_per_level_zeros_out() {
    assert_eq!(attack_exp(10, 20, 50, 100, 0), 0.0, "AttackExp with defExpPerLv=0");
}

/// Go：`TestHealExpTerm_HandCalc`——恢復 20/40 HP、受法者 Lv15 → term = 0.5×15 = 7.5。
/// 疊加另一位恢復 10/50、Lv5 → term = 0.2×5 = 1.0。Σ=8.5,套用「40/施法者Lv(10)」→ 34.0。
#[test]
fn heal_exp_term_hand_calc() {
    let sum = heal_exp_term(20, 40, 15) + heal_exp_term(10, 50, 5);
    assert_eq!(sum, 8.5, "Σterm");
    let total = 40.0 / 10.0 * sum;
    assert_eq!(total, 34.0, "恢復法術經驗值");
}

/// Go：`TestTeleportExp_HandCalc`——傳送術 10×(受法者Lv/施法者Lv)。Lv20 施法者對 Lv10 受法者。
#[test]
fn teleport_exp_hand_calc() {
    assert_eq!(teleport_exp(20, 10), 5.0, "TeleportExp(20,10)");
}

/// Go：`TestActionExp_HandCalc`——行動術 8×(受法者Lv/施法者Lv)。同Lv(10,10)→ 8。
#[test]
fn action_exp_hand_calc() {
    assert_eq!(action_exp(10, 10), 8.0, "ActionExp(10,10)");
}

/// Go：`TestBuffExpTerm_HandCalc`——魔刃/魔鎧/風行 2×Σ(受法者Lv/施法者Lv)。施法者Lv10,兩位受法者
/// Lv20、Lv5 → Σ=2.0+0.5=2.5,×2=5.0。
#[test]
fn buff_exp_term_hand_calc() {
    let sum = buff_exp_term(10, 20) + buff_exp_term(10, 5);
    assert_eq!(sum, 2.5, "Σterm");
    assert_eq!(2.0 * sum, 5.0, "buff 經驗值");
}

/// Go：`TestStatusExpTerm_HandCalc`——麻痺/毒擊(及解毒/祛麻共用同式)Σ(40×9/受法者總HP)×(受法者Lv/施法者Lv)。
/// 受法者總HP=90 → 360/90=4;Lv20/施法者Lv10=2 → 4×2=8。
#[test]
fn status_exp_term_hand_calc() {
    assert_eq!(status_exp_term(10, 20, 90), 8.0, "statusExpTerm(10,20,90)");
}

// ---- 端對端接線(combat.go / magic.go 呼叫路徑)----

/// Go：`TestAttackWithRNG_GrantsExpAndLevelsUp`——近戰攻擊命中後,Own 攻方應取得經驗值;經驗值
/// 夠大時連續升級,AttackResult 帶回 LevelUps。
#[test]
fn attack_with_rng_grants_exp_and_levels_up() {
    let mut st = new_test_state();
    let a = Unit {
        camp: OWN,
        name: "索爾".into(),
        cls_name: "劍士".into(),
        lv: 1,
        ap: 400,
        dp: 4,
        hp: 42,
        max_hp: 42,
        hit: 100,
        ev: 5,
        on_field: true,
        x: 0,
        y: 0,
        ..Default::default()
    };
    let d = Unit {
        camp: ENEMY,
        lv: 30,
        dp: 0,
        hp: 500,
        max_hp: 500,
        ev: 0,
        exp_per_level: 100,
        on_field: true,
        x: 0,
        y: 1,
        ..Default::default()
    };
    st.units = vec![a, d];
    let res = st.attack_with_rng(Some(0), 1, &mut GoMathRand::new(3));
    assert!(!res.missed, "命中率 100% 不應 Miss");
    assert!(res.exp_gained > 0.0, "ExpGained = {}, 應 > 0(高等級敵方、每級經驗100)", res.exp_gained);
    assert!(!res.level_ups.is_empty(), "經驗值夠大應觸發至少一次升級,LevelUps 為空;ExpGained={}", res.exp_gained);
    assert!(st.units[0].lv > 1, "攻方 Lv 應已提升,got {}", st.units[0].lv);
}

/// Go：`TestAttackWithRNG_EnemyAttacker_NoExp`——Enemy 當攻方時不應取得經驗值(doc42 gap 範圍僅玩家)。
#[test]
fn attack_with_rng_enemy_attacker_no_exp() {
    let mut st = new_test_state();
    let a = Unit { camp: ENEMY, lv: 1, ap: 400, hp: 50, max_hp: 50, hit: 100, on_field: true, ..Default::default() };
    let d = Unit {
        camp: OWN,
        lv: 30,
        dp: 0,
        hp: 500,
        max_hp: 500,
        exp_per_level: 100,
        on_field: true,
        x: 0,
        y: 1,
        ..Default::default()
    };
    st.units = vec![a, d];
    let res = st.attack_with_rng(Some(0), 1, &mut GoMathRand::new(3));
    assert!(
        res.exp_gained == 0.0 && res.level_ups.is_empty(),
        "Enemy 攻方不應取得經驗值,got ExpGained={} LevelUps={:?}",
        res.exp_gained,
        res.level_ups
    );
}

/// Go：`TestCastArea_HealGrantsExp`——治療法術命中後,Own 施法者應依恢復法術公式取得經驗值,
/// CastResult 帶回 ExpGained/LevelUps(彙總值,每筆結果相同)。
#[test]
fn cast_area_heal_grants_exp() {
    let mut st = new_test_state();
    let caster = Unit {
        camp: OWN,
        name: "瑪琳".into(),
        cls_name: "僧侶".into(),
        lv: 1,
        mp: 50,
        hp: 30,
        max_hp: 30,
        on_field: true,
        ..Default::default()
    };
    let tgt = Unit { camp: OWN, lv: 40, hp: 10, max_hp: 200, on_field: true, x: 0, y: 1, ..Default::default() };
    st.units = vec![caster, tgt];
    let sp = Spell { id: 13, dmg: 70, hit: 0, dist: 3, range: 0, mp: 5, target: 1, name: "治療".into() };
    let results = st.cast_area(Some(0), 0, 1, &sp, &mut GoMathRand::new(9));
    assert_eq!(results.len(), 1, "結果數");
    assert!(results[0].exp_gained > 0.0, "ExpGained = {}, 應 > 0(高Lv受法者、大量恢復)", results[0].exp_gained);
    assert!(
        st.units[0].exp > 0.0 || !results[0].level_ups.is_empty(),
        "施法者應累積到經驗值,caster.Exp={} LevelUps={:?}",
        st.units[0].exp,
        results[0].level_ups
    );
}

/// Go：`TestCastArea_SealSpell_NoExp`——封咒術(22)doc02 §4.5 未列經驗公式,施放不應給經驗
/// (誠實的「文件未涵蓋」,不是遺漏)。
#[test]
fn cast_area_seal_spell_no_exp() {
    let mut st = new_test_state();
    let caster = Unit { camp: OWN, lv: 10, mp: 50, hp: 30, max_hp: 30, on_field: true, ..Default::default() };
    let tgt = Unit { camp: ENEMY, lv: 10, hp: 100, max_hp: 100, on_field: true, x: 0, y: 1, ..Default::default() };
    st.units = vec![caster, tgt];
    let sp = Spell { id: 22, dmg: 0, hit: 0, dist: 3, range: 0, mp: 5, target: 0, name: "封咒".into() };
    let results = st.cast_area(Some(0), 0, 1, &sp, &mut GoMathRand::new(1));
    assert_eq!(results.len(), 1, "結果數");
    assert_eq!(results[0].exp_gained, 0.0, "封咒術不應給經驗值");
}
