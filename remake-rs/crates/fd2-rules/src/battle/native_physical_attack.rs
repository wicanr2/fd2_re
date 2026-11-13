//! Go 來源：`remake/internal/battle/native_physical_attack.go`（377 行）。

use super::*;

// native_physical_attack.go — 把 `sub_28A6C` 的兩次 `sub_2939D` 接成一次完整的
// 物理攻擊：攻方先打，守方符合條件就還手。證據見
// docs/knowledge-base/106-physical-attack-counterattack-20260910.md。

/// NativePhysicalStrike 是一次揮擊：一次 `sub_29F72` 加上把傷害寫回守方。
/// 原版同一次 `sub_2939D` 可能揮兩次——開頭 3% 的機率，或武器 record `+9`＝3。
///
/// Go：`native_physical_attack.go:16`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePhysicalStrike {
    pub roll: NativePhysicalRollResult,
    /// 實際扣掉的 HP（守方剩餘不足時小於 Roll.Damage）
    pub damage: i64,
    /// 這一擊之後守方的 HP
    pub defender_hp: i64,
}

/// NativePhysicalAttackResult 是一次物理攻擊指令的完整結果。
///
/// Go：`native_physical_attack.go:23`
#[derive(Debug, Clone, Default, PartialEq)]
pub struct NativePhysicalAttackResult {
    /// 主攻：ExpGained／LevelUps 是攻方拿到的
    pub attack: AttackResult,
    pub attack_strikes: Vec<NativePhysicalStrike>,
    /// Counter 為 nil 表示沒有反擊。反擊的經驗給守方：`sub_29F72` 每一擊都以「揮擊
    /// 者 `+6` ＝ 2」決定要不要把經驗寫進 `[0x53EC8]`，敵方行動結束時 `0x1566A`
    /// 對被打的那個單位呼叫 `0x1E292`，所以我方單位反擊敵人一樣有經驗、一樣會
    /// 當場升級。CounterExpGained／CounterLevelUps 就是那一份。
    pub counter: Option<AttackResult>,
    pub counter_strikes: Vec<NativePhysicalStrike>,
    pub counter_exp_gained: i64,
    pub counter_level_ups: Vec<LevelUpEvent>,
    /// RNGState 是整次結算（含升級成長擲骰）之後的原版全域 RNG。
    pub rng_state: u16,
}

/// nativeIgnoresTerrainModifier 重現 `sub_1F183`：回真就不套地形攻防修正，也就是
/// 飛行。缺欄位時各欄位當 0，那正好落在「非飛行」那一邊（0 ≠ 28、0 ≠ 19、
/// 0 ∉ {4,5}），與重製端既有行為一致。
///
/// Go：`native_physical_attack.go:43`，`func nativeIgnoresTerrainModifier(u *Unit) bool`
pub(crate) fn native_ignores_terrain_modifier(u: Option<&Unit>) -> bool {
    todo!("battle/native_physical_attack.go:nativeIgnoresTerrainModifier")
}

/// nativeEquippedWeaponSlot 重現 `sub_1B83D(unit, 0)`：找第一個「已裝備且 ID 小於
/// 0x80」的槽。原版看的是 `unit+10+2i` 的 bit 0x40；有原始旗標就用原始旗標，沒有
/// 才退回正規化的 Equipped。
///
/// Go：`native_physical_attack.go:67`，`func nativeEquippedWeaponSlot(u *Unit) (int, bool)`
pub(crate) fn native_equipped_weapon_slot(u: Option<&Unit>) -> (i64, bool) {
    todo!("battle/native_physical_attack.go:nativeEquippedWeaponSlot")
}

impl State {
    /// nativeEquippedWeapon 取出攻方武器 record 的 `+9`／`+10`／`+11`。拿不到原版物品表
    /// 時回零值，也就是沒有武器特效——那與重製端接線前的行為相同，不是新的猜測。
    ///
    /// Go：`native_physical_attack.go:90`，`func (s *State) nativeEquippedWeapon(u *Unit) (NativePhysicalWeapon, bool)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn native_equipped_weapon(&self, u: Option<UnitIdx>) -> (NativePhysicalWeapon, bool) {
        todo!("battle/native_physical_attack.go:State.nativeEquippedWeapon")
    }

    /// buildNativePhysicalRoll 組出一次擲骰的輸入。
    ///
    /// Go：`native_physical_attack.go:106`，`func (s *State) buildNativePhysicalRoll(a, d *Unit, rngState uint16) NativePhysicalRoll`
    /// 所有權：`a`、`d` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn build_native_physical_roll(
        &self,
        a: Option<UnitIdx>,
        d: Option<UnitIdx>,
        rng_state: u16,
    ) -> NativePhysicalRoll {
        todo!("battle/native_physical_attack.go:State.buildNativePhysicalRoll")
    }

    /// nativePhysicalExchange 重現一次 `sub_2939D`：開頭 3% 決定要不要揮兩次，之後每
    /// 一輪擲一次 `sub_29F72` 並把傷害寫回；武器 record `+9`＝3 會再多一輪，但只多一次
    /// （原版的 `v40` 旗標）。每一輪都重讀守方當下的 HP，所以第二擊打的是已扣血的目標。
    ///
    /// Go：`native_physical_attack.go:128`，`func (s *State) nativePhysicalExchange(a, d *Unit, rngState uint16) ([]NativePhysicalStrike, uint16, error)`
    /// 所有權：`a`、`d` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn native_physical_exchange(
        &mut self,
        a: Option<UnitIdx>,
        d: Option<UnitIdx>,
        rng_state: u16,
    ) -> Result<(Vec<NativePhysicalStrike>, u16)> {
        todo!("battle/native_physical_attack.go:State.nativePhysicalExchange")
    }

    /// counterattackDefender 把守方的狀態整理成反擊資格判定的輸入。
    ///
    /// 原版 `sub_1F0DC` 讀的 `unit+38` 就是 `+0x26`——六個暫時狀態剩餘回合數
    /// （`+0x22..+0x27`）中的一個，由 `0x1A866` 的 camp-phase 掃描逐一遞減。非零代表
    /// 那個狀態還在生效，同一組條件也擋掉回合開始的自動回復（`sub_1A30B` 的
    /// `[ebx+25h]`／`[ebx+26h]` 兩個閘門）。
    ///
    /// Go：`native_physical_attack.go:178`，`func (s *State) counterattackDefender(d *Unit) NativeCounterattackDefender`
    /// 所有權：`d` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn counterattack_defender(&self, d: Option<UnitIdx>) -> NativeCounterattackDefender {
        todo!("battle/native_physical_attack.go:State.counterattackDefender")
    }

    /// NativePhysicalCounterattackEligible 供呈現預檢沿用結算的資格來源。
    /// 這裡使用攻擊前HP；主攻結束後仍由結算端重新檢查生存條件。
    ///
    /// Go：`native_physical_attack.go:189`，`func (s *State) NativePhysicalCounterattackEligible(actor, target *Unit) bool`
    /// 所有權：`actor`、`target` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_physical_counterattack_eligible(&self, actor: Option<UnitIdx>, target: Option<UnitIdx>) -> bool {
        todo!("battle/native_physical_attack.go:State.NativePhysicalCounterattackEligible")
    }

    /// AttackNativePhysical 依原版結算一次物理攻擊，含反擊。rngState 是原版的全域
    /// `0x627B8`；呼叫者要把回傳的新狀態存回去，否則之後每一次結算都會偏掉。
    ///
    /// rng 只用在經驗值與升級（那條仍是重製端既有的規則），傷害本身完全走原版 RNG。
    ///
    /// Go：`native_physical_attack.go:198`，`func (s *State) AttackNativePhysical(a, d *Unit, rngState uint16, rng *rand.Rand) (NativePhysicalAttackResult, error)`
    /// 所有權：`a`、`d` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn attack_native_physical(
        &mut self,
        a: Option<UnitIdx>,
        d: Option<UnitIdx>,
        rng_state: u16,
        rng: &mut dyn GoRand,
    ) -> Result<NativePhysicalAttackResult> {
        todo!("battle/native_physical_attack.go:State.AttackNativePhysical")
    }

    /// Go：`native_physical_attack.go:202`，`func (s *State) attackNativePhysical( a, d *Unit, rngState uint16, rng *rand.Rand, nativeEXP *nativePhysicalExperiencePlan, ) (NativePhysicalAttackResult, error)`
    /// 所有權：`a`、`d` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn attack_native_physical_inner(
        &mut self,
        a: Option<UnitIdx>,
        d: Option<UnitIdx>,
        rng_state: u16,
        rng: &mut dyn GoRand,
        native_exp: Option<&NativePhysicalExperiencePlan>,
    ) -> Result<NativePhysicalAttackResult> {
        todo!("battle/native_physical_attack.go:State.attackNativePhysical")
    }
}

/// nativeLastStrikeExperience 重現 `[0x53EC8]` 的寫法：`sub_29F72` 每一擊都整個覆寫，
/// 不累加，所以留下來的是最後一擊算出的值——最後一擊打空就是 0，最後一擊把守方
/// 打倒就是整份 base。
///
/// Go：`native_physical_attack.go:283`，`func nativeLastStrikeExperience(plan *nativePhysicalExperiencePlan, strikes []NativePhysicalStrike) int`
pub(crate) fn native_last_strike_experience(
    plan: Option<&NativePhysicalExperiencePlan>,
    strikes: &[NativePhysicalStrike],
) -> i64 {
    todo!("battle/native_physical_attack.go:nativeLastStrikeExperience")
}

/// summarizeNativeStrikes 把一次交鋒的每一擊併成既有的 AttackResult 形狀：傷害相加，
/// 全部落空才算 Missed，任一擊暴擊就算 Crit。
///
/// Go：`native_physical_attack.go:293`，`func summarizeNativeStrikes(strikes []NativePhysicalStrike) AttackResult`
pub(crate) fn summarize_native_strikes(strikes: &[NativePhysicalStrike]) -> AttackResult {
    todo!("battle/native_physical_attack.go:summarizeNativeStrikes")
}

impl State {
    /// awardNativePhysicalExperience 是沒有原版 `+6` 來源的舊可編輯單位走的經驗規則
    /// （重製端既有公式、Go RNG 升級）。原生單位改走 nativeLastStrikeExperience 加
    /// AwardExpNative。
    ///
    /// Go：`native_physical_attack.go:310`，`func (s *State) awardNativePhysicalExperience( a, d *Unit, damage int, rng *rand.Rand, nativeEXP *nativePhysicalExperiencePlan, ) (float64, []LevelUpEvent)`
    /// 所有權：`a`、`d` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn award_native_physical_experience(
        &mut self,
        a: Option<UnitIdx>,
        d: UnitIdx,
        damage: i64,
        rng: &mut dyn GoRand,
        native_exp: Option<&NativePhysicalExperiencePlan>,
    ) -> (f64, Vec<LevelUpEvent>) {
        todo!("battle/native_physical_attack.go:State.awardNativePhysicalExperience")
    }

    /// AttackNativePhysicalWithExperience 是正式物理入口：傷害與反擊照原版，經驗值在
    /// 攻方帶有原版 `+6` 來源時走 planNativePhysicalExperience，否則走重製端既有規則。
    ///
    /// 預檢在改動任何 HP 之前完成：raw 資料缺項時整筆拒絕，不猜用相容欄位。
    ///
    /// Go：`native_physical_attack.go:334`，`func (s *State) AttackNativePhysicalWithExperience( actor, target *Unit, rngState uint16, rng *rand.Rand, ) (NativePhysicalAttackResult, error)`
    /// 所有權：`actor`、`target` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn attack_native_physical_with_experience(
        &mut self,
        actor: Option<UnitIdx>,
        target: Option<UnitIdx>,
        rng_state: u16,
        rng: Option<&mut dyn GoRand>,
    ) -> Result<NativePhysicalAttackResult> {
        todo!("battle/native_physical_attack.go:State.AttackNativePhysicalWithExperience")
    }
}

/// nativeClassCritPctTable 是 FD2.EXE 0x5239B 的 24 個職業暴擊率（索引 = record
/// +0x20 職業減一；docs/data/exe_tables/native_combat_tables.json 釘住同一組位元組）。
/// `sub_29F72` 以 `[ebp+0x20]-1` 直接索引這張表，不讀 unit 記錄裡的任何欄位。
///
/// Go：`native_physical_attack.go:358`
pub(crate) static NATIVE_CLASS_CRIT_PCT_TABLE: [i64; 24] =
    [5, 3, 3, 5, 3, 3, 0, 18, 5, 3, 3, 12, 3, 3, 12, 10, 6, 3, 3, 7, 3, 3, 30, 18];

/// nativeClassCritPct 依原版以職業查暴擊率；沒有 raw +0x20 的舊可編輯單位沿用
/// authored CritPct。
///
/// Go：`native_physical_attack.go:362`，`func nativeClassCritPct(u *Unit) int`
pub(crate) fn native_class_crit_pct(u: Option<&Unit>) -> i64 {
    todo!("battle/native_physical_attack.go:nativeClassCritPct")
}
