//! Go 來源：`remake/internal/battle/native_physical_experience.go`（85 行）。

use super::*;

/// 原版普通攻擊的兩階段整數除法與玩家交易上限。
/// 證據：docs/knowledge-base/94-ch01-town-parity-20260908.md。
/// 只閉合 EXP 邊界，傷害與成長仍沿用既有執行期的證據限制。
///
/// Go：`native_physical_experience.go:12`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativePhysicalExperiencePlan {
    pub(crate) base: i64,
    pub(crate) target_max_hp: i64,
    pub(crate) writes_accumulator: bool,
}

/// Go：`native_physical_experience.go:17`，`func planNativePhysicalExperience(actor, target *Unit) (nativePhysicalExperiencePlan, error)`
pub(crate) fn plan_native_physical_experience(
    actor: Option<&Unit>,
    target: Option<&Unit>,
) -> Result<NativePhysicalExperiencePlan> {
    let fail = || Err(battle_err!("native physical EXP: raw source or integer state unavailable"));
    let (Some(actor), Some(target)) = (actor, target) else {
        return fail();
    };
    if !actor.has_native_record_byte6 || !actor.has_native_record_byte5 {
        return fail();
    }
    if actor.native_record_byte6 != 2 {
        return Ok(NativePhysicalExperiencePlan::default());
    }
    if actor.camp != OWN
        || !actor.has_native_record_class
        || actor.lv < 1
        || actor.lv > 255
        || actor.exp < 0.0
        || actor.exp >= 100.0
        || actor.exp.is_nan()
        || actor.exp != actor.exp.trunc()
        || !target.has_battle_fig
    {
        return fail();
    }
    let (identity, ok) = native_record_byte8_for_unit(Some(actor));
    if !ok || target.battle_fig < 0 || target.battle_fig > 255 {
        return fail();
    }
    if target.battle_fig < 68 {
        return Ok(NativePhysicalExperiencePlan::default());
    }
    let Some(table) = target.native_constructor.as_ref() else {
        return fail();
    };
    if table.validate().is_err()
        || table.branch != "high_class"
        || table.index != target.battle_fig - 68
        || target.lv < 1
        || target.lv > 255
        || target.max_hp < 1
        || target.max_hp > 65535
    {
        return fail();
    }
    let mut level = actor.lv as u8;
    if actor.native_record_class > 8 && actor.native_record_class < 25 || identity == 28 {
        level = level.wrapping_add(30);
    }
    if level == 0 {
        return fail();
    }
    Ok(NativePhysicalExperiencePlan {
        base: i64::from(table.record[9]) * target.lv / i64::from(level),
        target_max_hp: target.max_hp,
        writes_accumulator: true,
    })
}

impl NativePhysicalExperiencePlan {
    /// award 是 `0x2A209..0x2A27A` 的一擊經驗：守方倒下（`[esp+0x28]` 剩餘 HP 為 0）
    /// 就是整份 base，否則乘上「這一擊的傷害／守方最大 HP」——打空時傷害 0 就是 0。
    /// 99 的上限不在這裡：那是玩家路徑 `0x11959` 才有的，敵方路徑 `0x1566A` 沒有。
    ///
    /// Go：`native_physical_experience.go:61`，`func (p nativePhysicalExperiencePlan) award(damage int, killed bool) int`
    pub(crate) fn award(self, damage: i64, killed: bool) -> i64 {
        if self.base == 0 {
            return 0;
        }
        if killed {
            return self.base;
        }
        if damage <= 0 {
            return 0;
        }
        damage.wrapping_mul(self.base) / self.target_max_hp
    }
}
impl State {
    /// AttackWithNativeExperience 先預檢所有 EXP 來源，再消耗 RNG 與提交攻擊。
    /// 玩家與敵方正式物理入口共用；不在 raw 資料缺少時猜用相容 ex 欄。
    ///
    /// Go：`native_physical_experience.go:76`，`func (s *State) AttackWithNativeExperience(actor, target *Unit, rng *rand.Rand) (AttackResult, error)`
    /// 所有權：`actor`、`target` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn attack_with_native_experience(
        &mut self,
        actor: Option<UnitIdx>,
        target: Option<UnitIdx>,
        rng: Option<&mut dyn GoRand>,
    ) -> Result<AttackResult> {
        let Some(rng) = rng else {
            return Err(battle_err!("native physical EXP: state or RNG unavailable"));
        };
        let plan = plan_native_physical_experience(actor.map(|i| &self.units[i]), target.map(|i| &self.units[i]))?;
        // plan 成功代表兩者都不是 nil（Go 的 nil 檢查在 planNativePhysicalExperience 裡）。
        let target = target.expect("planNativePhysicalExperience 已確認 target 存在");
        Ok(self.attack_with_experience(actor, target, rng, Some(&plan)))
    }
}

#[cfg(test)]
mod native_physical_experience_test;
