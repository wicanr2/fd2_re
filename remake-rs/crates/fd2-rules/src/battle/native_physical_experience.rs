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
    todo!("battle/native_physical_experience.go:planNativePhysicalExperience")
}

impl NativePhysicalExperiencePlan {
    /// award 是 `0x2A209..0x2A27A` 的一擊經驗：守方倒下（`[esp+0x28]` 剩餘 HP 為 0）
    /// 就是整份 base，否則乘上「這一擊的傷害／守方最大 HP」——打空時傷害 0 就是 0。
    /// 99 的上限不在這裡：那是玩家路徑 `0x11959` 才有的，敵方路徑 `0x1566A` 沒有。
    ///
    /// Go：`native_physical_experience.go:61`，`func (p nativePhysicalExperiencePlan) award(damage int, killed bool) int`
    pub(crate) fn award(self, damage: i64, killed: bool) -> i64 {
        todo!("battle/native_physical_experience.go:nativePhysicalExperiencePlan.award")
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
        todo!("battle/native_physical_experience.go:State.AttackWithNativeExperience")
    }
}
