//! Go 來源：`remake/internal/battle/native_turn_progression.go`（109 行）。

use super::*;

/// Go：`native_turn_progression.go:8`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeTurnEvent76Plan {
    pub r#final: bool,
    pub(crate) event: NativeTurnEvent,
}

/// Go：`native_turn_progression.go:14`，`func validateNativeTurnEvent76(event NativeTurnEvent) error`
pub(crate) fn validate_native_turn_event76(event: &NativeTurnEvent) -> Result<()> {
    todo!("battle/native_turn_progression.go:validateNativeTurnEvent76")
}

/// PlanNativeTurnEvent76 驗證 raw-camp2 的 live row 與 state byte，不修改單位、
/// 排程或 roster。state17 小於4走 repeat branch；等於4才進 final branch。
///
/// Go：`native_turn_progression.go:32`，`func PlanNativeTurnEvent76(st *State, event NativeTurnEvent) (NativeTurnEvent76Plan, error)`
pub fn plan_native_turn_event76(st: Option<&State>, event: &NativeTurnEvent) -> Result<NativeTurnEvent76Plan> {
    todo!("battle/native_turn_progression.go:PlanNativeTurnEvent76")
}

/// CommitNativeTurnEvent76Repeat 重驗 repeat plan，並原子提交 slot1 raw +5 bit7、
/// state17 increment及row1下一回合；若存在CONTINUE raw views則同步雙寫。
///
/// Go：`native_turn_progression.go:66`，`func CommitNativeTurnEvent76Repeat(st *State, plan NativeTurnEvent76Plan) error`
pub fn commit_native_turn_event76_repeat(st: Option<&mut State>, plan: &NativeTurnEvent76Plan) -> Result<()> {
    todo!("battle/native_turn_progression.go:CommitNativeTurnEvent76Repeat")
}
