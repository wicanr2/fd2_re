//! Go 來源：`remake/internal/battle/native_physical_damage.go`（150 行）。

use super::*;

// native_physical_damage.go — 原版 `sub_29F72`（0x29F72..0x2A289）的物理傷害。
//
// 這裡刻意只做「一次擲骰」，不碰單位狀態、不碰演出：原版把結果填進呼叫者的六格
// 陣列，`sub_2939D` 才依那六格決定演出與逐格扣血。證據見
// docs/knowledge-base/106-physical-attack-counterattack-20260910.md。
//
// **RNG 的消耗次數與順序也是契約。** 原版的 `sub_4E893` 是全域狀態，攻擊、演出
// 與 AI 共用同一條序列；少擲或多擲一次，之後每一次結算都會偏掉。所以未命中就
// 不再擲暴擊與隨機加成，武器分支該擲兩次的就擲兩次。

/// NativePhysicalWeapon 是攻方武器 record（`sub_4E56C` 的 0x17 位元組列）在傷害
/// 公式裡用到的三個欄位。`Reach` 不參與傷害，是反擊資格用的（見
/// NativeCounterattackEligible）。
///
/// Go：`native_physical_damage.go:22`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePhysicalWeapon {
    /// record +9：4＝暴擊加成、2＝狀態異常、3＝連擊
    pub effect: i64,
    /// record +10：Effect 4 是暴擊加成，Effect 2 是發動機率
    pub param: i64,
    /// record +11
    pub reach: i64,
}

/// NativePhysicalRoll 是一次物理擲骰的輸入。攻守雙方的數值都是**已生效值**，
/// 也就是原版 record 上的 `+0x48`／`+0x4A`／`+0x4C`／`+0x4E`；地形百分比由呼叫者
/// 依 `sub_1F183` 的飛行閘門決定，飛行時傳 0。
///
/// Go：`native_physical_damage.go:31`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePhysicalRoll {
    pub attacker_ap: i64,
    pub defender_dp: i64,
    pub attacker_hit: i64,
    pub defender_ev: i64,
    /// AttackerCritPct 是 0x5239B[職業−1]，見 native_combat_tables.json。
    pub attacker_crit_pct: i64,
    pub attacker_terrain_ap_pct: i64,
    pub defender_terrain_dp_pct: i64,
    pub weapon: NativePhysicalWeapon,
    pub rng_state: u16,
}

/// NativePhysicalRollResult 對應原版填進呼叫者的六格陣列。
///
/// Go：`native_physical_damage.go:48`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePhysicalRollResult {
    /// out[0]：原版 1 是未命中、0 是命中
    pub missed: bool,
    /// out[1]
    pub crit: bool,
    /// out[2]：武器 Effect 2 發動
    pub status: bool,
    /// out[4]：武器 Effect 3，同一次指令再打一次
    pub extra: bool,
    /// out[5]，也是原版的回傳值
    pub damage: i64,
    /// StatusValue 是 Status 為真時原版寫進守方 `+37` 的值（`rand()%4 + 2`）。
    pub status_value: i64,
    pub rng_state: u16,
}

/// RollNativePhysicalDamage 依 `sub_29F72` 的順序擲一次物理傷害。
///
/// Go：`native_physical_damage.go:62`，`func RollNativePhysicalDamage(in NativePhysicalRoll) (NativePhysicalRollResult, error)`
pub fn roll_native_physical_damage(r#in: NativePhysicalRoll) -> Result<NativePhysicalRollResult> {
    if r#in.attacker_ap < 0 || r#in.defender_dp < 0 {
        return Err(battle_err!(
            "native physical roll has negative stats: ap={} dp={}",
            r#in.attacker_ap,
            r#in.defender_dp
        ));
    }
    let mut rng = r#in.rng_state;
    let mut next = || {
        rng = fd2_formats::native_rng::native_rng_step(rng);
        i64::from(rng)
    };

    // 1. 地形修正。原版是 `v30 += v30 * 表值 / 100`，所以負百分比也走同一式。
    let ap = r#in.attacker_ap.wrapping_add(r#in.attacker_ap.wrapping_mul(r#in.attacker_terrain_ap_pct) / 100);
    let mut dp = r#in.defender_dp.wrapping_add(r#in.defender_dp.wrapping_mul(r#in.defender_terrain_dp_pct) / 100);

    // 2. 武器分支（0x29FE6 的 switch），在命中判定之前。
    let mut result = NativePhysicalRollResult { missed: true, ..Default::default() };
    let mut crit_pct = r#in.attacker_crit_pct;
    match r#in.weapon.effect {
        4 => crit_pct = crit_pct.wrapping_add(r#in.weapon.param),
        2 => {
            if next() % 100 < r#in.weapon.param {
                result.status = true;
                result.status_value = next() % 4 + 2;
            }
        }
        3 => result.extra = true,
        _ => {}
    }

    // 3. 命中：`rand()%100 < 命中 − 迴避`。未命中就到此為止，不再擲。
    if next() % 100 >= r#in.attacker_hit.wrapping_sub(r#in.defender_ev) {
        result.rng_state = rng;
        return Ok(result);
    }
    result.missed = false;

    // 4. 暴擊讓守方防禦減半——**在地形修正之後**。
    if next() % 100 < crit_pct {
        result.crit = true;
        dp /= 2;
    }

    // 5. 傷害：九成的差額，再加最多約一成的隨機。
    let mut damage = ap.wrapping_sub(dp).wrapping_mul(9) / 10;
    if damage < 0 {
        damage = 0;
    }
    let span = damage / 9;
    if span != 0 {
        damage += next() % span;
    }
    result.damage = damage;
    result.rng_state = rng;
    Ok(result)
}

/// NativeCounterattackDefender 是反擊資格判定需要的守方資訊。`WeaponReach` 是守方
/// 已裝備第 0 類裝備的 record `+11`；沒有已裝備的第 0 類時 HasWeapon 為 false。
///
/// Go：`native_physical_damage.go:120`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCounterattackDefender {
    pub x: i64,
    pub y: i64,
    /// record +38 ＝ +0x26，六個暫時狀態之一；非零就不反擊
    pub byte38: i64,
    pub has_weapon: bool,
    pub weapon_reach: i64,
    pub alive_hp: i64,
}

/// NativeCounterattackEligible 重現 `sub_1F0DC(攻方, 守方) == 1` 加上呼叫端的
/// 「守方還活著」判斷。原版的完整條件是：
///
///     sub_2939D(攻方, 守方, …) != 0   守方打完 HP 還大於 0
///     sub_1F0DC(攻方, 守方) == 1      守方 +38 為 0、兩格相鄰、已裝備第 0 類且 +11 為 1
///     dword_540FF == 0                不在抑制模式（該旗標的語意尚未解，重製端一律視為 0）
///
/// 相鄰是曼哈頓距離剛好 1，所以隔空攻擊不會被反擊。
///
/// Go：`native_physical_damage.go:136`，`func NativeCounterattackEligible(attackerX, attackerY int, defender NativeCounterattackDefender) bool`
pub fn native_counterattack_eligible(attacker_x: i64, attacker_y: i64, defender: NativeCounterattackDefender) -> bool {
    if defender.alive_hp <= 0 {
        return false;
    }
    if defender.byte38 != 0 {
        return false;
    }
    if abs(attacker_x.wrapping_sub(defender.x)).wrapping_add(abs(attacker_y.wrapping_sub(defender.y))) != 1 {
        return false;
    }
    if !defender.has_weapon {
        return false;
    }
    defender.weapon_reach == 1
}

#[cfg(test)]
mod native_physical_damage_test;

#[cfg(test)]
mod native_physical_parity_test;
