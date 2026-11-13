//! 移植自 `remake/internal/battle/magic_test.go`。
//!
//! `newTestState`、`mkUnit` 是 `combat_test.go`、`growth_test.go` 共用的測試輔助，這裡設為
//! `pub(crate)` 讓同組的其他測試模組引用。

use super::*;
use crate::test_support::remake_asset;

/// Go `newTestState`。
pub(crate) fn new_test_state() -> State {
    State { w: 20, h: 20, ..Default::default() }
}

/// Go `mkUnit`。
pub(crate) fn mk_unit(camp: Camp, x: i64, y: i64, hp: i64, mp: i64) -> Unit {
    Unit { camp, hp, max_hp: hp, mp, on_field: true, x, y, ..Default::default() }
}

/// Go：`TestCastArea_AoE_HitsAllInRange`——波及範圍(曼哈頓距離 ≤ Range)內的合法目標(同陣營篩選)
/// 全部各中一次,範圍外、陣營不合的單位不受影響。
#[test]
fn cast_area_aoe_hits_all_in_range() {
    let mut st = new_test_state();
    st.units = vec![
        mk_unit(OWN, 5, 5, 100, 50),
        mk_unit(ENEMY, 5, 5, 100, 0), // 中心,距離0
        mk_unit(ENEMY, 5, 6, 100, 0), // 距離1(範圍內)
        mk_unit(ENEMY, 4, 5, 100, 0), // 距離1(範圍內)
        mk_unit(ENEMY, 5, 8, 100, 0), // 距離3(範圍外)
        mk_unit(OWN, 6, 5, 100, 0),   // 距離1,但同陣營 → 攻擊法術不應命中
    ];
    let (caster, e4, friend) = (0, 4, 5);

    let sp = Spell { id: 999, dmg: 100, hit: 100, dist: 5, range: 1, mp: 20, target: 0, ..Default::default() };
    let results = st.cast_area(Some(caster), 5, 5, &sp, &mut GoMathRand::new(1));

    assert_eq!(results.len(), 3, "AoE 命中數,want 3(中心 + 2 個範圍內敵人)");
    for r in &results {
        assert!(
            !r.missed && r.amount > 0,
            "目標 {:?} 應命中造成傷害,got Missed={} Amount={}",
            r.target,
            r.missed,
            r.amount
        );
    }
    assert_eq!(st.units[e4].hp, 100, "範圍外的 e4 不應受影響");
    assert_eq!(st.units[friend].hp, 100, "同陣營的 friend 不應被攻擊法術命中");
}

/// Go：`TestCastArea_MPDeductedOnce`——AoE 命中多個目標時 MP 只扣一次(非每個目標各扣一次)。
#[test]
fn cast_area_mp_deducted_once() {
    let mut st = new_test_state();
    st.units = vec![mk_unit(OWN, 0, 0, 100, 20), mk_unit(ENEMY, 0, 0, 50, 0), mk_unit(ENEMY, 0, 1, 50, 0)];
    let sp = Spell { id: 998, dmg: 10, hit: 100, dist: 5, range: 1, mp: 20, target: 0, ..Default::default() };
    let results = st.cast_area(Some(0), 0, 0, &sp, &mut GoMathRand::new(2));
    assert_eq!(results.len(), 2, "命中數");
    assert_eq!(st.units[0].mp, 0, "MP 應只扣一次(20)");
}

/// Go：`TestCastArea_InsufficientMP`——MP 不足時不扣血、不套效果,回 nil。
#[test]
fn cast_area_insufficient_mp() {
    let mut st = new_test_state();
    st.units = vec![mk_unit(OWN, 0, 0, 100, 5)];
    let sp = Spell { id: 997, dmg: 10, hit: 100, dist: 5, range: 0, mp: 20, target: 0, ..Default::default() };
    let got = st.cast_area(Some(0), 0, 0, &sp, &mut GoMathRand::new(3));
    assert!(got.is_empty(), "MP 不足應回 nil,got {got:?}");
    assert_eq!(st.units[0].mp, 5, "MP 不足不應扣血");
}

/// Go：`TestCastArea_SealedCannotCast`——被封咒術禁止施法的單位無法再施法(doc02 §6.4)。
#[test]
fn cast_area_sealed_cannot_cast() {
    let mut st = new_test_state();
    let mut caster = mk_unit(OWN, 0, 0, 100, 50);
    caster.sealed = true;
    st.units = vec![caster];
    let sp = Spell { id: 996, dmg: 10, hit: 100, dist: 5, range: 0, mp: 5, target: 0, ..Default::default() };
    let got = st.cast_area(Some(0), 0, 0, &sp, &mut GoMathRand::new(4));
    assert!(got.is_empty(), "被封咒應無法施法,got {got:?}");
    assert_eq!(st.units[0].mp, 50, "施法失敗不應扣 MP");
}

/// Go：`TestApplySpell_HitRollBothOutcomes`——毒擊術(id26,hit=50)固定 seed 多次擲骰應同時出現命中與 Miss;
/// 命中時才施加中毒狀態,傷害套用 doc02 §4.3 隨機公式(dmg=10 恆得 9,對應攻略「附加 9 點傷害」)。
#[test]
fn apply_spell_hit_roll_both_outcomes() {
    let sp = Spell { id: 26, dmg: 10, hit: 50, dist: 4, range: 0, mp: 8, target: 0, ..Default::default() };
    let mut rng = GoMathRand::new(42);
    let (mut saw_hit, mut saw_miss) = (false, false);
    for i in 0..40 {
        let mut st = new_test_state();
        st.units = vec![mk_unit(OWN, 0, 0, 100, 100), mk_unit(ENEMY, 0, 0, 100, 0)];
        let results = st.cast_area(Some(0), 0, 0, &sp, &mut rng);
        assert_eq!(results.len(), 1, "第 {i} 次:預期 1 筆結果");
        let r = &results[0];
        let tgt = &st.units[1];
        if r.missed {
            saw_miss = true;
            assert!(!tgt.poisoned, "Miss 不應施加中毒");
        } else {
            saw_hit = true;
            assert!(
                tgt.poisoned && (2..=4).contains(&tgt.poison_turns),
                "命中應施加中毒且回合數在 2–4,got Poisoned={} Turns={}",
                tgt.poisoned,
                tgt.poison_turns
            );
            assert_eq!(r.amount, 9, "毒擊 dmg=10 經隨機公式應恆得 9");
        }
    }
    assert!(saw_hit && saw_miss, "40 次擲骰(hit=50%)應同時出現命中與 Miss,sawHit={saw_hit} sawMiss={saw_miss}");
}

/// Go：`TestApplySpell_BuffAP`——魔刃術(id17)AP +15%,持續 2–4 回合。
#[test]
fn apply_spell_buff_ap() {
    let mut st = new_test_state();
    // 離中心(0,0)有距離,避免施法者自己也落在 Range=0 內
    let mut tgt = mk_unit(OWN, 0, 0, 100, 0);
    tgt.ap = 100;
    st.units = vec![mk_unit(OWN, 5, 5, 100, 100), tgt];
    let sp = Spell { id: 17, dmg: 0, hit: 0, dist: 4, range: 0, mp: 5, target: 1, ..Default::default() };
    let results = st.cast_area(Some(0), 0, 0, &sp, &mut GoMathRand::new(9));
    assert!(results.len() == 1 && !results[0].missed, "魔刃術應命中我方目標,got {results:?}");
    let tgt = &st.units[1];
    assert_eq!(tgt.buff_ap_pct, 15, "BuffAPPct");
    assert!((2..=4).contains(&tgt.buff_turns), "BuffTurns = {}, want 2-4", tgt.buff_turns);
    assert_eq!(tgt.effective_ap(), 115, "EffectiveAP");
}

/// Go：`TestApplySpell_BuffWindWalk`——風行術(id19)HIT+15、EV+15(doc02 §6.4;非先前規格草案猜測的 MV+2)。
#[test]
fn apply_spell_buff_wind_walk() {
    let mut st = new_test_state();
    st.units = vec![mk_unit(OWN, 0, 0, 100, 100), mk_unit(OWN, 0, 0, 100, 0)];
    let sp = Spell { id: 19, dmg: 0, hit: 0, dist: 4, range: 0, mp: 8, target: 1, ..Default::default() };
    st.cast_area(Some(0), 0, 0, &sp, &mut GoMathRand::new(10));
    let tgt = &st.units[1];
    assert!(
        tgt.buff_hit == 15 && tgt.buff_ev == 15,
        "風行術應 HIT+15/EV+15,got BuffHit={} BuffEV={}",
        tgt.buff_hit,
        tgt.buff_ev
    );
    assert!(
        tgt.buff_dp_pct == 0 && tgt.buff_ap_pct == 0,
        "風行術不應動到 AP/DP,got AP%={} DP%={}",
        tgt.buff_ap_pct,
        tgt.buff_dp_pct
    );
}

/// Go：`TestApplySpell_ComboSpells`——破壞神(魔刃+魔鎧+風行)、暗邪鬼(麻痺+封咒+毒擊)組合技(doc02 §6.4)。
#[test]
fn apply_spell_combo_spells() {
    let mut st = new_test_state();
    st.units = vec![mk_unit(OWN, 0, 0, 100, 200), mk_unit(OWN, 0, 0, 100, 0)];
    let sp_break = Spell { id: 34, dmg: 0, hit: 0, dist: 5, range: 0, mp: 28, target: 1, ..Default::default() };
    st.cast_area(Some(0), 0, 0, &sp_break, &mut GoMathRand::new(11));
    let ally = &st.units[1];
    assert!(
        ally.buff_ap_pct == 15 && ally.buff_dp_pct == 15 && ally.buff_hit == 15 && ally.buff_ev == 15,
        "破壞神應同時套用魔刃+魔鎧+風行,got AP%={} DP%={} Hit={} EV={}",
        ally.buff_ap_pct,
        ally.buff_dp_pct,
        ally.buff_hit,
        ally.buff_ev
    );

    let mut st2 = new_test_state();
    st2.units = vec![mk_unit(OWN, 0, 0, 100, 200), mk_unit(ENEMY, 0, 0, 100, 0)];
    let sp_dark = Spell { id: 35, dmg: 0, hit: 0, dist: 4, range: 0, mp: 36, target: 0, ..Default::default() };
    st2.cast_area(Some(0), 0, 0, &sp_dark, &mut GoMathRand::new(12));
    let enemy = &st2.units[1];
    assert!(
        enemy.paralyzed && enemy.sealed && enemy.poisoned,
        "暗邪鬼應同時施麻痺+封咒+毒擊,got Paralyzed={} Sealed={} Poisoned={}",
        enemy.paralyzed,
        enemy.sealed,
        enemy.poisoned
    );
}

/// Go：`TestApplySpell_ActionSpellResetsActed`——行動術(id25)讓已行動單位本回合可再次行動。
#[test]
fn apply_spell_action_spell_resets_acted() {
    let mut st = new_test_state();
    let mut tgt = mk_unit(OWN, 0, 0, 100, 0);
    tgt.acted = true;
    st.units = vec![mk_unit(OWN, 0, 0, 100, 100), tgt];
    let sp = Spell { id: 25, dmg: 0, hit: 0, dist: 3, range: 1, mp: 24, target: 1, ..Default::default() };
    st.cast_area(Some(0), 0, 0, &sp, &mut GoMathRand::new(14));
    assert!(!st.units[1].acted, "行動術應重置 Acted=false");
}

/// Go：`TestCureAndTickStatus`——解毒/祛麻術清除對應異常;TickStatus 每回合遞減、到期自動解除、
/// 中毒每回合扣 MaxHP 的 10%。
#[test]
fn cure_and_tick_status() {
    let mut st = new_test_state();
    let mut tgt = mk_unit(OWN, 0, 0, 100, 0);
    (tgt.poisoned, tgt.poison_turns) = (true, 3);
    (tgt.paralyzed, tgt.paralyze_turns) = (true, 3);
    st.units = vec![mk_unit(OWN, 0, 0, 100, 100), tgt];

    let sp_cure = Spell { id: 20, dmg: 0, hit: 0, dist: 4, range: 0, mp: 5, target: 1, ..Default::default() };
    st.cast_area(Some(0), 0, 0, &sp_cure, &mut GoMathRand::new(13));
    let tgt = &st.units[1];
    assert!(
        !tgt.poisoned && tgt.poison_turns == 0,
        "解毒術後應清除中毒,got Poisoned={} Turns={}",
        tgt.poisoned,
        tgt.poison_turns
    );
    assert!(tgt.paralyzed, "解毒術不應影響麻痺狀態");

    let mut poisoned = mk_unit(ENEMY, 1, 1, 100, 0);
    (poisoned.poisoned, poisoned.poison_turns) = (true, 1);
    poisoned.tick_status();
    assert_eq!(poisoned.hp, 90, "中毒應扣 MaxHP 的 10%");
    assert!(!poisoned.poisoned, "PoisonTurns 歸零後應自動解除中毒狀態");

    for _ in 0..3 {
        st.units[1].tick_status();
    }
    assert!(!st.units[1].paralyzed, "ParalyzeTurns 遞減 3 次後應自動解除麻痺");
}

/// Go：`TestCast_BackwardCompat`——舊版單體施法介面(引擎目前呼叫的簽名)維持可用。
///
/// Go 的 `caster2` 不在 `st.Units`；Rust 依模組文件第 6 點先 push 再傳索引（MP 不足時 `CastArea`
/// 在掃描單位之前就返回，多出的單位不影響結果）。
#[test]
fn cast_backward_compat() {
    let mut st = new_test_state();
    st.units = vec![mk_unit(OWN, 0, 0, 100, 100), mk_unit(ENEMY, 0, 0, 50, 0)];
    let (caster, tgt) = (0, 1);
    let sp = Spell { id: 995, dmg: 20, hit: 100, dist: 5, range: 0, mp: 10, target: 0, ..Default::default() };
    let amt = st.cast(Some(caster), tgt, &sp);
    assert!(amt > 0, "Cast 舊介面應回傳正傷害,got {amt}");
    assert_eq!(st.units[caster].mp, 90, "MP 應扣 10");

    st.units.push(mk_unit(OWN, 0, 0, 100, 5));
    let caster2 = 2;
    assert_eq!(st.cast(Some(caster2), tgt, &sp), -1, "MP 不足應回 -1");
}

/// Go：`TestLoadSpells_Integration`——實際 spells.json(EXE dump)載入 + 施放,交叉驗證 doc02 §6.1 傷害區間。
#[test]
fn load_spells_integration() {
    let Some(path) = remake_asset("spells.json") else { return };
    let spells = load_spells(&path).unwrap();
    assert_eq!(spells.len(), 36, "法術數");
    let fireball = spells.iter().find(|sp| sp.id == 0).cloned();
    let Some(fireball) = fireball.filter(|sp| sp.name == "火炎") else {
        panic!("id0 法術名應為 火炎");
    };

    let mut st = new_test_state();
    st.units = vec![mk_unit(OWN, 0, 0, 100, 100), mk_unit(ENEMY, 0, 0, 100, 0)];
    let results = st.cast_area(Some(0), 0, 0, &fireball, &mut GoMathRand::new(20));
    assert_eq!(results.len(), 1, "results");
    assert!(
        results[0].missed || (45..=49).contains(&results[0].amount),
        "火炎術命中傷害 = {}, want 45–49(doc02 §6.1)",
        results[0].amount
    );
}
