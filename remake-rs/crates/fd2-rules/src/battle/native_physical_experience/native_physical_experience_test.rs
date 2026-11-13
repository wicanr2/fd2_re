//! 移植自 `remake/internal/battle/native_physical_experience_test.go`。

use super::*;
use crate::test_support::remake_asset;
use fd2_formats::native_rng::native_rng_step;

/// Go `completeNativeAIScoringUnit`（`native_ai_record_test.go`；測試輔助不跨模組共用，這裡複製一份）。
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

fn high_class_record() -> NativeConstructorTable {
    NativeConstructorTable {
        branch: "high_class".into(),
        index: 28,
        record: vec![1, 7, 14, 0, 0, 7, 1, 1, 4, 21],
        ..Default::default()
    }
}

/// Go `nativeExperiencePair`
fn native_experience_pair() -> (Unit, Unit) {
    let (mut a, mut d) = (complete_native_ai_scoring_unit(), complete_native_ai_scoring_unit());
    (a.camp, a.native_record_byte6, a.native_record_class) = (OWN, 2, 2);
    (a.native_record_byte5, a.native_record_byte8) = (0, 0);
    (a.lv, a.exp, a.hit, a.ap, a.crit_pct) = (4, 0.0, 100, 29, 0);
    (a.hp, a.max_hp, a.x, a.y) = (48, 48, 0, 0);
    (d.camp, d.native_record_byte6, d.native_record_byte5) = (ENEMY, 0, 0);
    (d.lv, d.hp, d.max_hp, d.dp, d.ev) = (2, 28, 28, 4, 0);
    (d.x, d.y, d.battle_fig) = (1, 0, 96);
    d.native_constructor = Some(high_class_record());
    // 刻意保留舊錯誤 ex 值，確保正式結算只消費原始建構列。
    d.exp_per_level = 110;
    (a, d)
}

/// Go：`TestNativePhysicalExperienceTwoIntegerDivisions`
#[test]
fn native_physical_experience_two_integer_divisions() {
    let (mut a, mut d) = native_experience_pair();
    let p = plan_native_physical_experience(Some(&a), Some(&d)).unwrap();
    assert!(p.award(22, false) == 7 && p.award(6, true) == 10, "未依原版分段取整或擊倒全額結算");
    a.native_record_class = 9;
    let p = plan_native_physical_experience(Some(&a), Some(&d));
    assert!(p.as_ref().is_ok_and(|p| p.base == 1), "轉職分母={p:?}");
    (a.native_record_class, a.native_record_byte8) = (2, 28);
    let p = plan_native_physical_experience(Some(&a), Some(&d));
    assert!(p.is_ok_and(|p| p.base == 1), "raw +8=28 的分母未加30");
    (a.native_record_byte8, a.lv, d.lv) = (0, 1, 20);
    let p = plan_native_physical_experience(Some(&a), Some(&d));
    // 99 的上限在玩家路徑 0x11959，不在 sub_29F72 的一擊計算裡。
    assert!(p.as_ref().is_ok_and(|p| p.award(28, true) == 420), "一擊經驗未依 base 全額回傳：{p:?}");
    assert_eq!(p.unwrap().award(0, false), 0, "打空的一擊應該把 [0x53EC8] 寫成 0");
}

/// Go：`TestNativePhysicalExperienceCarriesAcrossCappedCounterAndMiss`——滿級反擊的 0x1E292 早退不清
/// [0x53EC8]；下一次敵方隔空未命中，沒有反擊，0x1566A 仍把保留值交給被打的非滿級我方。
#[test]
fn native_physical_experience_carries_across_capped_counter_and_miss() {
    let (mut s, enemy, capped) = native_counter_experience_pair();
    (s.units[capped].lv, s.units[capped].exp) = (40, 0.0);
    s.units[enemy].lv = 15;
    s.units[enemy].native_constructor.as_mut().unwrap().record[9] = 40;
    let first =
        s.attack_native_physical_with_experience(Some(enemy), Some(capped), 0x1234, Some(&mut GoMathRand::new(4)));
    assert!(
        first.as_ref().is_ok_and(|r| r.counter.is_some() && r.counter_exp_gained == 0)
            && s.units[enemy].hp == 0
            && s.native_experience_accumulator == 15,
        "滿級反擊未保留15：result={first:?} carry={}",
        s.native_experience_accumulator
    );
    let (_, mut target) = native_experience_pair();
    (target.camp, target.native_record_byte6, target.native_record_class) = (OWN, 2, 2);
    (target.battle_fig, target.native_record_byte8) = (5, 5);
    (target.lv, target.exp, target.hp, target.max_hp, target.ev) = (26, 52.0, 129, 231, 100);
    (target.x, target.y) = (0, 0);
    (s.units[enemy].hp, s.units[enemy].x, s.units[enemy].y) = (28, 4, 0);
    // Go：`s.W, s.Units = 5, []*Unit{enemy, target}`。
    s.w = 5;
    s.units.truncate(1);
    s.units.push(target);
    let target = 1;
    let second =
        s.attack_native_physical_with_experience(Some(enemy), Some(target), 0x1234, Some(&mut GoMathRand::new(4)));
    assert!(
        second.as_ref().is_ok_and(|r| r.counter.is_none() && r.attack.missed && r.counter_exp_gained == 15)
            && s.units[target].exp == 67.0
            && s.native_experience_accumulator == 0,
        "隔空未命中未消費保留值：result={second:?} exp={} carry={}",
        s.units[target].exp,
        s.native_experience_accumulator
    );
    let third =
        s.attack_native_physical_with_experience(Some(enemy), Some(target), 0x1234, Some(&mut GoMathRand::new(4)));
    assert!(
        third.as_ref().is_ok_and(|r| r.counter_exp_gained == 0) && s.units[target].exp == 67.0,
        "正常清零後重複發經驗：result={third:?} exp={}",
        s.units[target].exp
    );
}

/// Go `nativeCounterExperiencePair`：讓敵方（`+6`＝0）打我方（`+6`＝2）：敵人打不死我方，
/// 我方反擊一擊必殺。兩邊都帶原版建構列，所以反擊的經驗計畫和主攻一樣可算。
/// 回傳的兩個索引依序是攻方（敵人）與守方（我方）。
fn native_counter_experience_pair() -> (State, UnitIdx, UnitIdx) {
    let (mut a, mut d) = native_experience_pair();
    // 攻方換成敵人：+6＝0、命中極低所以永遠打空；守方是我方 lv4，反擊 AP 遠高於敵 DP。
    (a.camp, a.native_record_byte6, a.native_record_byte5) = (ENEMY, 0, 0);
    (a.lv, a.hp, a.max_hp, a.hit, a.ap, a.dp, a.ev) = (2, 28, 28, 0, 1, 0, 0);
    a.battle_fig = 96;
    a.native_constructor = Some(high_class_record());
    (d.camp, d.native_record_byte6, d.native_record_byte5, d.native_record_class) = (OWN, 2, 0, 2);
    d.native_record_byte8 = 0;
    (d.lv, d.exp, d.hp, d.max_hp, d.hit, d.ev, d.ap, d.dp) = (4, 95.0, 48, 48, 100, 100, 200, 4);
    d.battle_fig = 9;
    d.native_transient = [0; 6]; // +0x26 非零就不反擊
    (d.inventory, d.equipped) = (vec![1], vec![true]);
    (d.inventory_slots, d.native_inventory_flags) = (vec![1], vec![0x40]);
    let mut s = State { w: 2, h: 1, units: vec![a, d], ..Default::default() };
    s.bind_native_future_item_rows(&vec![0; 2 * NATIVE_ITEM_EFFECT_ROW_SIZE as usize]).unwrap();
    s.native_future_item_rows[NATIVE_ITEM_EFFECT_ROW_SIZE as usize + 11] = 1; // 武器射程 1 才能反擊
    // 對應 growth.json 的 [6,8)、[4,5)、[2,2)、[8,11)、[0,0)：DP 跨距 1 仍擲一次（恆 0），
    // DX 與 MP 跨距 0 不擲。
    s.native_growth_rows = Some(BTreeMap::from([(
        9,
        GrowthRow {
            ap: StatRange { min: 6, max: 7, native: true, fixed: false },
            dp: StatRange { min: 4, max: 4, native: true, fixed: false },
            dx: StatRange { min: 2, max: 2, native: true, fixed: true },
            hp: StatRange { min: 8, max: 10, native: true, fixed: false },
            mp: StatRange { min: 0, max: 0, native: true, fixed: true },
            balance_rates: None,
        },
    )]));
    (s, 0, 1)
}

/// Go：`TestNativeCounterKillAwardsDefenderExperienceAndLevelsUpNatively` 釘住 0x1566A：敵方
/// 行動結束後對被打的我方單位呼叫 0x1E292，[0x53EC8] 是我方反擊最後一擊寫的值。
/// 反擊擊倒 lv2 敵人：base = 21×2/4 = 10，全額；我方 95+10 → 升級，成長擲原版 RNG。
#[test]
fn native_counter_kill_awards_defender_experience_and_levels_up_natively() {
    let (mut s, a, d) = native_counter_experience_pair();
    let (ap0, dp0, hp0, max_hp0) = (s.units[d].ap, s.units[d].dp, s.units[d].hp, s.units[d].max_hp);
    let r = s.attack_native_physical_with_experience(Some(a), Some(d), 0x1234, Some(&mut GoMathRand::new(1))).unwrap();
    assert!(r.counter.is_some() && s.units[a].hp == 0, "反擊未擊倒敵人：{r:?} a.HP={}", s.units[a].hp);
    assert!(r.attack.exp_gained == 0.0 && r.attack.level_ups.is_empty(), "敵方攻擊不該拿經驗：{:?}", r.attack);
    let du = &s.units[d];
    assert!(
        r.counter_exp_gained == 10 && du.exp == 5.0 && du.lv == 5 && r.counter_level_ups.len() == 1,
        "反擊經驗／升級：exp={} d.Exp={} d.Lv={} ups={:?}",
        r.counter_exp_gained,
        du.exp,
        du.lv,
        r.counter_level_ups
    );
    // 成長要接在反擊之後的 RNG 上：AP 6..7、DP 恆 4 但擲一次、DX 固定 2（不擲）、
    // HP 8..10、MP 固定 0（不擲）——第四章 r9 收據 0x1E54A 對 id 4 正好四筆。
    let mut rng_after_strikes = r.counter_strikes.last().unwrap().roll.rng_state;
    let (want_ap, want_dp, want_hp) = {
        let dummy = Unit::default();
        let mut roll = native_stat_roll(&mut rng_after_strikes);
        let ap = 6 + roll(&dummy, 2);
        let dp = 4 + roll(&dummy, 1);
        let hp = 8 + roll(&dummy, 3);
        (ap, dp, hp)
    };
    assert!(
        du.ap == ap0 + want_ap && du.dp == dp0 + want_dp && du.max_hp == max_hp0 + want_hp && du.hp == hp0,
        "成長未依原版 RNG：AP {ap0}→{} DP {dp0}→{} MaxHP {max_hp0}→{} HP={} want +{want_ap}/+{want_dp}/+{want_hp}",
        du.ap,
        du.dp,
        du.max_hp,
        du.hp
    );
    assert_eq!(r.rng_state, rng_after_strikes, "結算後 RNG 應為成長擲骰之後的值");
}

/// Go：`TestNativeExchangeStopsWhenDefenderFalls` 釘住 0x29B4C：守方 HP 歸零就不再揮第二擊，
/// 也不再消耗 RNG。挑一個開頭 3% 會給兩擊的狀態，第一擊就打死。
#[test]
fn native_exchange_stops_when_defender_falls() {
    let (mut s, a, d) = native_counter_experience_pair();
    (s.units[a].hp, s.units[a].max_hp) = (1, 1);
    let mut found = false;
    for state in 0..=0xFFFFu16 {
        if i64::from(native_rng_step(state)) % 100 >= 3 {
            continue;
        }
        s.units[a].hp = 1;
        let (strikes, _) = s.native_physical_exchange(Some(d), Some(a), state).unwrap();
        assert!(
            strikes.len() == 1 && strikes[0].defender_hp == 0,
            "state={state:#x} strikes={strikes:?}：守方倒下後仍揮了第二擊"
        );
        found = true;
        break;
    }
    assert!(found, "找不到二連擊的起始狀態");
}

/// Go：`TestNativePhysicalAttackThenMovementRecordsRemainSerializable`
#[test]
fn native_physical_attack_then_movement_records_remain_serializable() {
    let (a, d) = native_experience_pair();
    let mut s = State { w: 2, h: 1, units: vec![a, d], ..Default::default() };
    let r = s.attack_with_native_experience(Some(0), Some(1), Some(&mut GoMathRand::new(3))).unwrap();
    let want = r.amount * 10 / 28;
    let a = &s.units[0];
    assert!(
        !r.missed
            && r.exp_gained == want as f64
            && a.exp == want as f64
            && a.acted
            && a.native_record_byte5 & 0x80 != 0,
        "攻擊／EXP／行動={r:?} actor={a:?}"
    );
    let records = native_ai_scoring_records(&s.units).unwrap_or_else(|e| panic!("攻擊後下一個移動的記錄建構失敗：{e}"));
    assert!(records[0x3c] == want as u8 && records[5] & 0x80 != 0, "原生記錄未同步 EXP／行動");
    native_item_panel_record_for_unit(Some(&s.units[0])).unwrap();
}

/// Go：`TestNativePhysicalExperienceMissingSourceRejectsBeforeMutation`
#[test]
fn native_physical_experience_missing_source_rejects_before_mutation() {
    let (a, mut d) = native_experience_pair();
    d.native_constructor = None;
    let mut s = State { w: 2, h: 1, units: vec![a, d], ..Default::default() };
    let (mut rng, mut control) = (GoMathRand::new(1), GoMathRand::new(1));
    assert!(s.attack_with_native_experience(Some(0), Some(1), Some(&mut rng)).is_err(), "缺建構列未拒絕");
    let a = &s.units[0];
    assert!(
        !(a.acted
            || a.exp != 0.0
            || a.native_record_byte5 != 0
            || s.units[1].hp != 28
            || rng.int63() != control.int63()),
        "預檢失敗仍變更交易或RNG"
    );
}

/// Go：`TestMap0PhysicalExperienceUsesRawConstructorInsteadOfLegacyEX`
#[test]
fn map0_physical_experience_uses_raw_constructor_instead_of_legacy_ex() {
    let Some(path) = remake_asset("maps/map0/map0_units.json") else { return };
    let s = load(&path).unwrap();
    let (a, _) = native_experience_pair();
    let p = plan_native_physical_experience(Some(&a), Some(&s.units[0]));
    assert!(p.as_ref().is_ok_and(|p| p.base == 10), "map0原始經驗基數={p:?}");
}
