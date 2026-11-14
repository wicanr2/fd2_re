//! Go 來源：`remake/internal/battle/native_command_experience.go`（246 行，2026-11-14 凍結例外修正）。
//!
//! 玩家指令的經驗累計：原版每個效果寫入端直接累加 `[0x53EC8]`，玩家控制器 `0x117E7` 在行動結束後
//! 統一發放（`0x118EE` 清零、`0x11943` 執行行動、`0x11959` 壓到 99、`0x1196D` 呼叫 `0x1E292`）。
//! 寫入端清單由 IDA 9.4 對 FD2.EXE（SHA-256 222b7d06…）的資料 xref 取得
//! （`tools/ida_export_data_writers.py`、`tools/ida_export_function_listing.py`）。
//!
//! 所有權：Go 的加總函式收的結果帶 `*Unit`；Rust 的結果只存 `state.units` 的索引，所以加總函式
//! 多一個 `units: &[Unit]` 參數（同 `PORTING.md` 簽名變更紀錄的 B 組做法）。

use super::*;

/// 各寫入端共用的「經驗用等級」L：記錄 +0x21（`lv`），職業 +0x20 在 9..24 時加 30。
///
/// Go：`native_command_experience.go:23`，`func nativeExperienceLevel(u *Unit) int`
fn native_experience_level(u: &Unit) -> i64 {
    let mut level = u.lv;
    if u.has_native_record_class && u.native_record_class > 8 && u.native_record_class < 0x19 {
        level += 30;
    }
    level
}

/// `0x1C8B3..0x1C8BC` 的 `0x4E4FF(+7 − 0x44)` 第 9 byte，和物理經驗讀同一張高階職業表。來源不完整時
/// 失敗即關閉。
///
/// Go：`native_command_experience.go:33`，`func nativeHighClassExperienceRow9(target *Unit) (int, error)`
fn native_high_class_experience_row9(target: &Unit) -> Result<i64> {
    let ok = match target.native_constructor.as_ref() {
        Some(table) => {
            table.validate().is_ok()
                && table.branch == "high_class"
                && table.index == target.battle_fig - 0x44
                && (1..=255).contains(&target.lv)
                && (1..=65535).contains(&target.max_hp)
        }
        None => false,
    };
    if !ok {
        return Err(battle_err!("native command EXP: high-class table unavailable for fig {}", target.battle_fig));
    }
    Ok(i64::from(target.native_constructor.as_ref().map_or(0, |t| t.record[9])))
}

/// `0x1C81F` 扣血之後的經驗累加（`0x1C8A3..0x1C8E2`）：守方 +7 < 0x44 時是 0；否則
/// `e = 列表[9] × 守方 +0x21`，守方扣血後 HP 非 0 時 `e = e × 擲出傷害 / 守方 +0x42`（有號整數除法）。
/// 不除以施術者等級；沒有原版 BattleFig 的正規化單位回 0。
///
/// Go：`native_command_experience.go:52`，`func NativeDamageExperience(target *Unit, rolled, hpAfter int) (int, error)`
/// Go 版先檢查 nil 目標；Rust 收 `&Unit`，該分支不移植。
pub fn native_damage_experience(target: &Unit, rolled: i64, hp_after: i64) -> Result<i64> {
    if !target.has_battle_fig || target.battle_fig < 0x44 {
        return Ok(0);
    }
    let row9 = native_high_class_experience_row9(target)?;
    let mut exp = row9 * target.lv;
    if hp_after != 0 {
        exp = exp * rolled / target.max_hp;
    }
    Ok(exp)
}

/// `0x1C916` 回復 HP 之後的經驗累加（`0x1C99C..0x1C9CF`）：受術者 +7 < 0x4B 時
/// `40 × L × 實際回復量 / +0x42`；最大 HP 為 0 時回 0。
///
/// Go：`native_command_experience.go:73`，`func NativeRestoreExperience(target *Unit, actual int) int`
/// Go 版的 nil 目標回 0；Rust 收 `Option<&Unit>`。
pub fn native_restore_experience(target: Option<&Unit>, actual: i64) -> i64 {
    let Some(target) = target else {
        return 0;
    };
    if !target.has_battle_fig || target.battle_fig >= 0x4B || target.max_hp == 0 {
        return 0;
    }
    40 * native_experience_level(target) * actual / target.max_hp
}

/// 結果裡的目標索引；Go 的 nil 目標在 Rust 是 `None`。
fn unit_at(units: &[Unit], idx: Option<UnitIdx>) -> Option<&Unit> {
    idx.and_then(|i| units.get(i))
}

fn require_unit(units: &[Unit], idx: Option<UnitIdx>) -> Result<&Unit> {
    unit_at(units, idx).ok_or_else(|| battle_err!("native command EXP: nil target"))
}

/// 加總指令 0～12（`0x2A6BD` → `0x1C75E` → `0x1C81F`）每個命中目標的經驗；打空不累加。
///
/// Go：`native_command_experience.go:82`，`func NativeCommandDamageExperience(results []NativeCommandDamageResult) (int, error)`
pub fn native_command_damage_experience(units: &[Unit], results: &[NativeCommandDamageResult]) -> Result<i64> {
    let mut total = 0;
    for result in results {
        if !result.native_command_damage.hit {
            continue;
        }
        let target = require_unit(units, result.target)?;
        total += native_damage_experience(target, result.native_command_damage.damage, result.hp_after)?;
    }
    Ok(total)
}

/// 加總指令 13～16 每個目標的 `0x1C916` 經驗。
///
/// Go：`native_command_experience.go:98`，`func NativeCommandHealExperience(results []NativeCommandHealResult) int`
pub fn native_command_heal_experience(units: &[Unit], results: &[NativeCommandHealResult]) -> i64 {
    results.iter().map(|r| native_restore_experience(unit_at(units, r.target), r.restore.actual)).sum()
}

/// 加總 `0x276EC` → `0x1C81F`（指令 24、28～31）每個目標的經驗；擲出值用 `rolled`（截斷前）。
///
/// Go：`native_command_experience.go:108`，`func NativeCommandDerivedStrikeExperience(results []NativeCommand24Damage) (int, error)`
pub fn native_command_derived_strike_experience(units: &[Unit], results: &[NativeCommand24Damage]) -> Result<i64> {
    let mut total = 0;
    for result in results {
        let target = require_unit(units, result.target)?;
        total += native_damage_experience(target, result.rolled, result.hp_after)?;
    }
    Ok(total)
}

/// 加總指令 20～22、25～27 尾段的經驗：20／21 → `0x22AF6`（4 × L 加上 `0x1C916` 的 score）；
/// 22／26／27 → `0x22D1B`（`0x1C81F(目標,10)` 的傷害經驗加 8 × +0x21，不加 30）；
/// 25 → `0x22C04`（+5 bit7 被清掉的目標 8 × L）。
///
/// Go：`native_command_experience.go:128`，`func NativeCommand2022Experience(plan *NativeAICommand2022Plan) (int, error)`
/// Go 版先檢查 nil plan；Rust 收 `&NativeAICommand2022Plan`，該分支不移植。
pub fn native_command2022_experience(units: &[Unit], plan: &NativeAICommand2022Plan) -> Result<i64> {
    let mut total = 0;
    for result in &plan.results {
        let target = unit_at(units, result.target);
        if result.command25 {
            if let (true, Some(target)) = (result.cleared_byte5, target) {
                total += 8 * native_experience_level(target);
            }
        } else if let Some(restore) = &result.restore {
            if let (true, Some(target)) = (restore.applied, target) {
                total += restore.restore.score + 4 * native_experience_level(target);
            }
        } else if let Some(apply) = &result.apply {
            if !apply.applied {
                continue;
            }
            let index = apply.target_index;
            let after = usize::try_from(index).ok().and_then(|i| plan.after.get(i));
            let (Some(after), Some(target)) = (after, target) else {
                return Err(battle_err!("native command EXP: command tail target state unavailable"));
            };
            if after.target != result.target {
                return Err(battle_err!("native command EXP: command tail target state unavailable"));
            }
            total += native_damage_experience(target, apply.damage.rolled, after.hp)? + 8 * target.lv;
        }
    }
    Ok(total)
}

/// 指令 32：`0x2111A` → `0x1C75E` → `0x1C81F`，同指令 0～12。
///
/// Go：`native_command_experience.go:162`，`func NativeCompoundCommand32Experience(result NativeCompoundCommand32Result) (int, error)`
pub fn native_compound_command32_experience(units: &[Unit], result: &NativeCompoundCommand32Result) -> Result<i64> {
    let mut total = 0;
    for target in &result.targets {
        if !target.damage.hit {
            continue;
        }
        let unit = require_unit(units, target.target)?;
        total += native_damage_experience(unit, target.damage.damage, target.hp_after)?;
    }
    Ok(total)
}

/// 指令 33：`0x211A4` 對每個目標呼叫 `0x1C916`，累加量是各次的 score。
///
/// Go：`native_command_experience.go:179`，`func NativeCompoundCommand33Experience(result NativeCompoundCommand33Result) int`
pub fn native_compound_command33_experience(result: &NativeCompoundCommand33Result) -> i64 {
    result.restore.results.iter().map(|r| r.score).sum()
}

/// 指令 34：`0x27FC9` 依序呼叫 `0x22721`／`0x22866`／`0x22997`，每段的累加量在 `accumulator`。
///
/// Go：`native_command_experience.go:189`，`func NativeCompoundCommand34Experience(result NativeCompoundCommand34Result) int`
pub fn native_compound_command34_experience(result: &NativeCompoundCommand34Result) -> i64 {
    result.stages.iter().map(|s| s.accumulator).sum()
}

/// 指令 35：`0x27FC9` 呼叫三次 `0x22D1B`；每段的 8 × +0x21 在 `accumulator`，命中目標的
/// `0x1C81F(目標,10)` 傷害經驗依該段寫回後的 HP 另算。
///
/// Go：`native_command_experience.go:199`，`func NativeCompoundCommand35Experience(result NativeCompoundCommand35Result) (int, error)`
pub fn native_compound_command35_experience(units: &[Unit], result: &NativeCompoundCommand35Result) -> Result<i64> {
    if result.stage_states.len() != result.stages.len() {
        return Err(battle_err!("native command EXP: command35 stage states unavailable"));
    }
    let mut total = 0;
    for (stage, states) in result.stages.iter().zip(&result.stage_states) {
        total += stage.accumulator;
        for application in &stage.results {
            if !application.applied {
                continue;
            }
            let state = usize::try_from(application.target_index).ok().and_then(|i| states.get(i));
            let Some(target) = state.and_then(|s| unit_at(units, s.target)) else {
                return Err(battle_err!("native command EXP: command35 target state unavailable"));
            };
            total += native_damage_experience(target, application.damage.rolled, state.map_or(0, |s| s.hp))?;
        }
    }
    Ok(total)
}

impl State {
    /// 玩家路徑的行動收尾：把這次行動的 delta 加進 `[0x53EC8]`；擊倒的單位帶無條件 exp_cancel 時整筆
    /// 清零（`0x34B48`），超過 99 壓到 99（`0x11959`），再 `0x1E292` 發給行動者；有收下才清零
    /// （`0x1E51A`）。`targets` 只用來查 exp_cancel。
    ///
    /// Go：`native_command_experience.go:230`，`func (s *State) AwardNativePlayerActionExperience(actor *Unit, delta int, rngState uint16, targets ...*Unit) (int, []LevelUpEvent, uint16)`
    /// 所有權：`actor` 與 `targets` 是 `self.units` 的索引；Go 的 nil actor 在 Rust 是 `None`。
    pub fn award_native_player_action_experience(
        &mut self,
        actor: Option<UnitIdx>,
        delta: i64,
        rng_state: u16,
        targets: &[UnitIdx],
    ) -> (i64, Vec<LevelUpEvent>, u16) {
        if actor.is_none() {
            return (0, Vec::new(), rng_state);
        }
        self.native_experience_accumulator += delta;
        if self.kill_cancels_exp(targets) {
            self.native_experience_accumulator = 0;
        }
        if self.native_experience_accumulator > 99 {
            self.native_experience_accumulator = 99;
        }
        let (got, ups, next) = self.award_exp_native(actor, self.native_experience_accumulator, rng_state);
        if got > 0 {
            self.native_experience_accumulator = 0;
        }
        (got, ups, next)
    }
}

#[cfg(test)]
mod native_command_experience_test;
