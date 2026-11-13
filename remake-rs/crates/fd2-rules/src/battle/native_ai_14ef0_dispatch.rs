//! Go 來源：`remake/internal/battle/native_ai_14ef0_dispatch.go`（120 行）。

use super::*;

/// NativeAI14EF0Tail 是 0x14ef0 在三個 raw score producer 完成後的
/// 尾端路由。名稱只保留原始位址，不把 0x15311、0x1548e 或 0x15055
/// 猜成攻擊、物品、法術或其他玩法效果。
///
/// Go：`native_ai_14ef0_dispatch.go:8`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct NativeAI14EF0Tail(pub u8);

/// Go 原式：`iota`
pub const NATIVE_AI14_EF0_NO_TAIL: NativeAI14EF0Tail = NativeAI14EF0Tail(0);

pub const NATIVE_AI14_EF0_CALL1548_E: NativeAI14EF0Tail = NativeAI14EF0Tail(1);

pub const NATIVE_AI14_EF0_CALL15311: NativeAI14EF0Tail = NativeAI14EF0Tail(2);

pub const NATIVE_AI14_EF0_CALL15055: NativeAI14EF0Tail = NativeAI14EF0Tail(3);

/// NativeAI14EF0Input 保存 0x14ef0 直接讀取的 raw 值。HasRaw* 欄位是
/// provenance 閘門；normalized 分數、Camp、HP 或 command 名稱不可補入。
/// CommandID／CommandWord 只有在 c4f==c23>c33 的分支會被消費，但該分支
/// 仍要求兩者明示存在，避免把缺少 0x4e516 結果的資料誤派發。
///
/// Go：`native_ai_14ef0_dispatch.go:21`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeAI14EF0Input {
    pub score_c4_f: i32,
    pub score_c23: i32,
    pub score_c33: i32,
    pub command_id: i32,
    pub command_word: u16,
    pub record34: u8,
    pub actor_word48: u16,
    pub target_word4_a: u16,
    pub has_raw_score_c4_f: bool,
    pub has_raw_score_c23: bool,
    pub has_raw_score_c33: bool,
    pub has_raw_command_id: bool,
    pub has_raw_command_word: bool,
    pub has_raw_record34: bool,
    pub has_raw_actor_word48: bool,
    pub has_raw_target_word4_a: bool,
}

/// NativeAI14EF0Decision 分開保存尾端動作與 caller 消費的回傳值。
/// 三者同分且至少6時，0x1504B仍回1，不能因 NoTail 進入移動後備。
/// 證據：docs/data/ida/fd2_ai_14ef0_return_20261006.json。
///
/// Go：`native_ai_14ef0_dispatch.go:46`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeAI14EF0Decision {
    pub tail: NativeAI14EF0Tail,
    pub accepted: bool,
}

/// Go：`native_ai_14ef0_dispatch.go:51`，`func PlanNativeAI14EF0Dispatch(in NativeAI14EF0Input) (NativeAI14EF0Decision, error)`
pub fn plan_native_ai14_ef0_dispatch(r#in: NativeAI14EF0Input) -> Result<NativeAI14EF0Decision> {
    todo!("battle/native_ai_14ef0_dispatch.go:PlanNativeAI14EF0Dispatch")
}

/// SelectNativeAI14EF0Tail 只保存 0x14ef0 的 raw 尾端決策樹。呼叫端必須
/// 另行執行並驗證前置的 0x14237、0x1598a、0x1567e producer。本函式不修改
/// 狀態，刻意不接 NextAIPlan 或正式 AI 執行器。
///
/// Go：`native_ai_14ef0_dispatch.go:64`，`func SelectNativeAI14EF0Tail(in NativeAI14EF0Input) (NativeAI14EF0Tail, error)`
pub fn select_native_ai14_ef0_tail(r#in: NativeAI14EF0Input) -> Result<NativeAI14EF0Tail> {
    todo!("battle/native_ai_14ef0_dispatch.go:SelectNativeAI14EF0Tail")
}
