//! 移植自 `remake/internal/battle/native_physical_damage_test.go`。

use super::*;
use fd2_formats::native_rng::native_rng_step;

/// Go `rngSequence`：回傳從 state 起連續 n 次 NativeRNGStep 的值，用來預測原版擲骰。
fn rng_sequence(mut state: u16, n: usize) -> Vec<u16> {
    (0..n)
        .map(|_| {
            state = native_rng_step(state);
            state
        })
        .collect()
}

/// Go：`TestRollNativePhysicalDamageFollowsTheOriginalOrder` 釘住 sub_29F72 的順序：
/// 地形修正 →命中 →暴擊（防禦減半）→九成差額加隨機。順序若和重製端舊公式一樣
/// 反過來（先暴擊減半再套地形），整數除法會給出不同的傷害。
#[test]
fn roll_native_physical_damage_follows_the_original_order() {
    const STATE: u16 = 0x1234;
    let seq = rng_sequence(STATE, 3);
    let input = NativePhysicalRoll {
        attacker_ap: 40,
        defender_dp: 11,
        attacker_hit: 100,
        defender_ev: 0,
        attacker_crit_pct: 100, // 讓暴擊必定發生，好驗證減半的時機
        attacker_terrain_ap_pct: 10,
        defender_terrain_dp_pct: 10,
        rng_state: STATE,
        ..Default::default()
    };
    let got = roll_native_physical_damage(input).unwrap();
    assert!(!got.missed && got.crit, "命中 100%、暴擊 100% 卻得到 missed={} crit={}", got.missed, got.crit);
    // 原版順序：ap=40+4=44、dp=11+1=12、暴擊後 dp=6 → 9*(44-6)/10 = 34。
    let ap = 40 + 40 * 10 / 100;
    let mut dp = 11 + 11 * 10 / 100;
    dp /= 2;
    let mut want = 9 * (ap - dp) / 10;
    let span = want / 9;
    if span != 0 {
        want += i64::from(seq[2]) % span;
    }
    assert_eq!(got.damage, want, "傷害與原版順序算出來的不同");
    // 反過來（先減半再套地形）會是另一個值；不同才證明這支測試在看順序。
    #[allow(clippy::identity_op)] // 照 Go 原式保留「減半後再套地形」的寫法。
    let reversed = 9 * ((40 + 40 * 10 / 100) - (11 / 2 + (11 / 2) * 10 / 100)) / 10;
    assert_ne!(reversed, 9 * (ap - dp) / 10, "兩種順序算出同一個基礎傷害，這組數字驗不到順序");
    assert_eq!(got.rng_state, seq[2], "RNG 停在 {:#x}，應該是第三次擲骰後的 {:#x}", got.rng_state, seq[2]);
}

/// Go：`TestRollNativePhysicalDamageStopsRollingOnAMiss` 釘住未命中就不再擲暴擊與隨機
/// 加成。RNG 是全域共用的，多擲一次會讓之後每一次結算都偏掉。
#[test]
fn roll_native_physical_damage_stops_rolling_on_a_miss() {
    const STATE: u16 = 0x4321;
    let seq = rng_sequence(STATE, 1);
    let got = roll_native_physical_damage(NativePhysicalRoll {
        attacker_ap: 40,
        defender_dp: 5,
        attacker_hit: 0, // 命中率 0：`rand%100 >= 0` 恆成立
        defender_ev: 0,
        attacker_crit_pct: 100,
        rng_state: STATE,
        ..Default::default()
    })
    .unwrap();
    assert!(got.missed, "命中率 0 卻命中了");
    assert!(got.damage == 0 && !got.crit, "未命中卻有傷害 {}／暴擊 {}", got.damage, got.crit);
    assert_eq!(got.rng_state, seq[0], "未命中共擲了不只一次：停在 {:#x}，應為 {:#x}", got.rng_state, seq[0]);
}

/// Go：`TestRollNativePhysicalDamageWeaponBranches` 釘住武器 record +9 的三個分支，
/// 包含它們各自消耗幾次 RNG。
#[test]
fn roll_native_physical_damage_weapon_branches() {
    const STATE: u16 = 0x2468;
    // effect4 把 +10 加進暴擊率：職業暴擊率 0、武器加成 100 → 必定暴擊。
    let got = roll_native_physical_damage(NativePhysicalRoll {
        attacker_ap: 30,
        defender_dp: 10,
        attacker_hit: 100,
        attacker_crit_pct: 0,
        weapon: NativePhysicalWeapon { effect: 4, param: 100, ..Default::default() },
        rng_state: STATE,
        ..Default::default()
    })
    .unwrap();
    assert!(got.crit, "武器加成 100 卻沒有暴擊");

    // effect3 是連擊且不擲骰
    let seq = rng_sequence(STATE, 3);
    let got = roll_native_physical_damage(NativePhysicalRoll {
        attacker_ap: 30,
        defender_dp: 10,
        attacker_hit: 100,
        weapon: NativePhysicalWeapon { effect: 3, ..Default::default() },
        rng_state: STATE,
        ..Default::default()
    })
    .unwrap();
    assert!(got.extra, "effect 3 沒有標記連擊");
    assert_eq!(got.rng_state, seq[2], "effect 3 多擲或少擲了骰：停在 {:#x}，應為 {:#x}", got.rng_state, seq[2]);

    // effect2 發動時多擲一次算狀態值
    let seq = rng_sequence(STATE, 2);
    let got = roll_native_physical_damage(NativePhysicalRoll {
        attacker_ap: 30,
        defender_dp: 10,
        attacker_hit: 100,
        weapon: NativePhysicalWeapon { effect: 2, param: 100, ..Default::default() },
        rng_state: STATE,
        ..Default::default()
    })
    .unwrap();
    assert!(got.status, "發動機率 100 卻沒有狀態異常");
    let want = i64::from(seq[1]) % 4 + 2;
    assert_eq!(got.status_value, want, "狀態值，原版是第二次擲骰的 %4+2");
    assert!((2..=5).contains(&got.status_value), "狀態值 {} 落在 2..5 之外", got.status_value);

    // effect2 沒發動就只擲一次
    let got = roll_native_physical_damage(NativePhysicalRoll {
        attacker_ap: 30,
        defender_dp: 10,
        attacker_hit: 100,
        weapon: NativePhysicalWeapon { effect: 2, param: 0, ..Default::default() },
        rng_state: STATE,
        ..Default::default()
    })
    .unwrap();
    assert!(!got.status, "發動機率 0 卻發動了");
}

/// Go：`TestRollNativePhysicalDamageHasNoFloor` 釘住原版沒有「至少 1 點」的下限：
/// 防禦高於攻擊時傷害就是 0。重製端舊公式對玩家攻擊硬給 1 點，那不是原版行為。
#[test]
fn roll_native_physical_damage_has_no_floor() {
    let got = roll_native_physical_damage(NativePhysicalRoll {
        attacker_ap: 5,
        defender_dp: 50,
        attacker_hit: 100,
        rng_state: 0x1111,
        ..Default::default()
    })
    .unwrap();
    assert!(!got.missed, "命中率 100 卻沒命中");
    assert_eq!(got.damage, 0, "防禦遠高於攻擊時傷害應為 0（原版）");
}

/// Go：`TestNativeCounterattackEligibleChecksEveryCondition` 逐條驗反擊資格，每一條都
/// 要單獨拿掉一次——否則「全都成立時回真」證明不了任何一條真的被檢查。
#[test]
fn native_counterattack_eligible_checks_every_condition() {
    let base = NativeCounterattackDefender {
        x: 5,
        y: 19,
        alive_hp: 8,
        has_weapon: true,
        weapon_reach: 1,
        ..Default::default()
    };
    const AX: i64 = 6;
    const AY: i64 = 19;
    assert!(native_counterattack_eligible(AX, AY, base), "四項條件全中卻判成不能反擊");
    type Mutate = fn(&mut NativeCounterattackDefender);
    let cases: [(&str, Mutate); 7] = [
        ("守方被打死", |d| d.alive_hp = 0),
        ("守方 +38 非零", |d| d.byte38 = 1),
        ("沒有已裝備的第 0 類", |d| d.has_weapon = false),
        ("武器 +11 不是 1", |d| d.weapon_reach = 2),
        ("隔一格", |d| d.x = 4),
        ("斜角", |d| (d.x, d.y) = (5, 20)),
        ("同一格", |d| (d.x, d.y) = (AX, AY)),
    ];
    for (name, mutate) in cases {
        let mut defender = base;
        mutate(&mut defender);
        assert!(!native_counterattack_eligible(AX, AY, defender), "{name} 仍判成可以反擊");
    }
}

/// Go：`TestCounterattackReadsTheRealTransientByte` 釘住反擊判定讀的是真正的
/// `+0x26`（六個暫時狀態剩餘回合數之一），不是恆為 0 的佔位值。原版
/// `sub_1F0DC` 讀 `unit+38`，同一個閘門也擋掉回合開始的自動回復。
///
/// Go 的 `defender` 不在 `State.Units`；Rust 依模組文件第 6 點先 push 再傳索引。
#[test]
fn counterattack_reads_the_real_transient_byte() {
    let mut state = State::default();
    state.units.push(Unit { x: 5, y: 19, hp: 8, ..Default::default() });
    let defender = Some(0);
    assert_eq!(state.counterattack_defender(defender).byte38, 0, "沒有狀態時 Byte38 應為 0");
    assert!(state.units[0].set_native_transient_duration(0x26, 3), "寫不進 +0x26");
    let got = state.counterattack_defender(defender);
    assert_eq!(got.byte38, 3, "+0x26 設成 3 之後 Byte38");
    assert!(!native_counterattack_eligible(6, 19, got), "+0x26 生效中仍判成可以反擊");
    // 其他五個 transient 不該影響反擊——原版只讀 +0x26。
    state.units[0].set_native_transient_duration(0x26, 0);
    for offset in [0x22, 0x23, 0x24, 0x25, 0x27] {
        state.units[0].set_native_transient_duration(offset, 4);
    }
    let mut other = state.counterattack_defender(defender);
    (other.has_weapon, other.weapon_reach) = (true, 1);
    assert!(native_counterattack_eligible(6, 19, other), "只有其他 transient 生效時就不能反擊了，原版只看 +0x26");
}
