//! 移植自 `remake/internal/battle/native_physical_parity_test.go`——拿原版收據的數字驗傷害公式。
//!
//! 輸入全部來自 dosgolem 的 checkpoint（攻擊前那一格，step 368,483,500 的 80 位元組
//! record）與地圖資料，不是挑出來湊答案的：
//!
//!     攻方 idx2 亞雷斯 (6,19)：HP 48、AP 26、DP 6、HIT 92、EV 2、職業 3（騎士）
//!     守方 idx11 盜賊  (5,19)：HP 28、AP 24、DP 4、HIT 97、EV 2、職業 7（盜賊）
//!     兩格的地形碼都是 0（map0 的 tiles×native_terrain_control 第 2 個位元組）
//!     職業暴擊率：騎士 3、盜賊 0（0x5239B，見 native_combat_tables.json）
//!
//! 原版實測結果（docs/data/ui-traces/fd2-counterattack-20260910.json）：
//! 守方 28→8（傷害 20），1,200,000 指令後攻方 48→31（反擊傷害 17）。

use super::*;
use fd2_formats::native_rng::native_rng_step;
use std::collections::BTreeMap;

const PARITY_ACTOR_AP: i64 = 26;
const PARITY_ACTOR_DP: i64 = 6;
const PARITY_ACTOR_HIT: i64 = 92;
const PARITY_ACTOR_EV: i64 = 2;
const PARITY_ACTOR_CRIT: i64 = 3; // 騎士
const PARITY_THIEF_AP: i64 = 24;
const PARITY_THIEF_DP: i64 = 4;
const PARITY_THIEF_HIT: i64 = 97;
const PARITY_THIEF_EV: i64 = 2;
const PARITY_THIEF_CRIT: i64 = 0; // 盜賊
const PARITY_TERRAIN_AP_PCT: i64 = 5; // 地形碼 0 的 0x51A12[0]
const PARITY_TERRAIN_DP_PCT: i64 = 0; // 地形碼 0 的 0x51A2A[0]

const PARITY_ORIGINAL_MAIN_DAMAGE: i64 = 20;
const PARITY_ORIGINAL_COUNTER_DAMAGE: i64 = 17;

fn parity_main_roll(state: u16) -> NativePhysicalRoll {
    NativePhysicalRoll {
        attacker_ap: PARITY_ACTOR_AP,
        defender_dp: PARITY_THIEF_DP,
        attacker_hit: PARITY_ACTOR_HIT,
        defender_ev: PARITY_THIEF_EV,
        attacker_crit_pct: PARITY_ACTOR_CRIT,
        attacker_terrain_ap_pct: PARITY_TERRAIN_AP_PCT,
        defender_terrain_dp_pct: PARITY_TERRAIN_DP_PCT,
        rng_state: state,
        ..Default::default()
    }
}

fn parity_counter_roll(state: u16) -> NativePhysicalRoll {
    // 反擊把攻守對調：盜賊打騎士，兩人站的格子不變，所以地形百分比也不變。
    NativePhysicalRoll {
        attacker_ap: PARITY_THIEF_AP,
        defender_dp: PARITY_ACTOR_DP,
        attacker_hit: PARITY_THIEF_HIT,
        defender_ev: PARITY_ACTOR_EV,
        attacker_crit_pct: PARITY_THIEF_CRIT,
        attacker_terrain_ap_pct: PARITY_TERRAIN_AP_PCT,
        defender_terrain_dp_pct: PARITY_TERRAIN_DP_PCT,
        rng_state: state,
        ..Default::default()
    }
}

/// Go：`TestNativeCounterDamageIsAlwaysSeventeen` 釘住反擊那一擊：盜賊的職業暴擊率是 0，
/// 而 9*(25−6)/10 ＝ 17 讓隨機加成的跨距只有 1，所以只要命中，傷害在**任何** RNG
/// 狀態下都是 17——與原版實測的 48→31 完全相同。這是不靠挑 seed 的對拍。
#[test]
fn native_counter_damage_is_always_seventeen() {
    let mut hits = 0;
    for state in 0..=0xFFFFu16 {
        let got = roll_native_physical_damage(parity_counter_roll(state)).unwrap();
        if got.missed {
            continue;
        }
        hits += 1;
        assert!(!got.crit, "狀態 {state:#x} 暴擊了，但盜賊的職業暴擊率是 0");
        assert_eq!(got.damage, PARITY_ORIGINAL_COUNTER_DAMAGE, "狀態 {state:#x} 的反擊傷害與原版實測不同");
    }
    assert!(hits > 0, "六萬多個狀態沒有一次命中，這支測試等於沒驗到東西");
    // 命中率 97−2 ＝ 95%，允許 RNG 分布的偏差。
    let ratio = f64::from(hits) / 65536.0;
    assert!((0.9..=1.0).contains(&ratio), "命中比例 {ratio:.3} 偏離 95% 太多");
}

/// Go：`TestNativeMainDamageMatchesTheOriginalReceipt` 釘住主攻那一擊：未暴擊時
/// 9*(27−4)/10 ＝ 20，隨機加成跨距 2，所以傷害只可能是 20 或 21；原版實測的
/// 28→8 正是 20。暴擊會讓守方防禦減半而得到 22，與實測不符——也就是那一次沒暴擊。
#[test]
fn native_main_damage_matches_the_original_receipt() {
    let mut seen: BTreeMap<i64, i64> = BTreeMap::new();
    for state in 0..=0xFFFFu16 {
        let got = roll_native_physical_damage(parity_main_roll(state)).unwrap();
        if got.missed {
            continue;
        }
        *seen.entry(got.damage).or_default() += 1;
    }
    for damage in [PARITY_ORIGINAL_MAIN_DAMAGE, PARITY_ORIGINAL_MAIN_DAMAGE + 1] {
        assert!(seen.get(&damage).copied().unwrap_or(0) != 0, "沒有任何狀態算出傷害 {damage}");
    }
    for damage in seen.keys() {
        // 20/21 是未暴擊、22/23 是暴擊（守方防禦 4→2）。
        assert!(matches!(damage, 20..=23), "出現預期外的傷害 {damage}，公式或輸入有問題");
    }
    assert!(
        seen.get(&22).copied().unwrap_or(0) + seen.get(&23).copied().unwrap_or(0) != 0,
        "騎士暴擊率 3% 卻一次都沒暴擊，暴擊那條沒被驗到"
    );
}

/// Go：`TestNativeExchangeReproducesTheOriginalHPTimeline` 找出能同時重現原版兩次結算的
/// RNG 狀態：主攻 20、反擊 17，中間隔一次 sub_2939D 開頭的 3% 判定。找得到就表示
/// 公式與 RNG 的消耗順序都與原版一致。
#[test]
fn native_exchange_reproduces_the_original_hp_timeline() {
    let mut matches = 0;
    for state in 0..=0xFFFFu16 {
        // sub_2939D 開頭先擲 3% 決定要不要揮兩次；這裡只找單擊的那條。
        let mut rng = native_rng_step(state);
        if i64::from(rng) % 100 < 3 {
            continue;
        }
        let main = roll_native_physical_damage(parity_main_roll(rng)).unwrap();
        if main.missed || main.damage != PARITY_ORIGINAL_MAIN_DAMAGE {
            continue;
        }
        rng = native_rng_step(main.rng_state);
        if i64::from(rng) % 100 < 3 {
            continue;
        }
        let counter = roll_native_physical_damage(parity_counter_roll(rng)).unwrap();
        if counter.missed || counter.damage != PARITY_ORIGINAL_COUNTER_DAMAGE {
            continue;
        }
        matches += 1;
    }
    assert!(matches > 0, "找不到任何 RNG 狀態能同時給出原版的 20 與 17；公式或 RNG 的消耗順序與原版不一致");
    eprintln!("有 {matches} 個起始狀態重現原版的兩次結算（守方 28→8、攻方 48→31）");
}
