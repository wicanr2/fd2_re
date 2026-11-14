//! Go 來源：`remake/internal/campaign/native_continue_pending_groups.go`（320 行）。

use super::*;

/// Go：`native_continue_pending_groups.go:13`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeContinueTurnKey {
    pub(crate) turn: i64,
    pub(crate) event_id: i64,
}

/// MaterializeNativeContinuePendingGroups 將仍未觸發的可編輯回合增援，綁到
/// CONTINUE 戰場的未物化 FDFIELD roster。原版存檔選單在 0x117E7 的玩家互動
/// 階段呼叫 0x19DF7；只有 0x13565 判定玩家可行動列耗盡後才進 0x1A30B。
/// 其中 raw selector 1/0 在增加回合前掃描，selector 2 則在增加後掃描；所以
/// saved turn 的 selector 1/0 仍待執行，selector 2 已於上一輪尾端執行。
///
/// 這個適配器只接受 live turn/event pair 與可編輯 scenario 完全相符的章節。
/// 原版 handler 會改寫 live turn byte；尚未資料化的改寫一律拒絕，不用靜態
/// FDFIELD 或舊 scenario 覆寫存檔。所有檢查與深複製成功後才提交 State。
///
/// Go：`native_continue_pending_groups.go:27`，`func MaterializeNativeContinuePendingGroups( state *battle.State, input fdsave.ContinueRuntimeInput, assetChapter int, assetState *battle.State, scenario *battle.Scenario, itemRows []byte, ) error`
pub fn materialize_native_continue_pending_groups(
    state: Option<&mut battle::State>,
    input: &save::ContinueRuntimeInput,
    asset_chapter: i64,
    asset_state: Option<&battle::State>,
    scenario: Option<&battle::Scenario>,
    item_rows: &[u8],
) -> Result<()> {
    todo!("campaign/native_continue_pending_groups.go:MaterializeNativeContinuePendingGroups")
}

/// Go：`native_continue_pending_groups.go:214`，`func nativeContinueSpawnSchedule( scenario *battle.Scenario, ) (map[nativeContinueTurnKey][]int, error)`
pub(crate) fn native_continue_spawn_schedule(
    scenario: &battle::Scenario,
) -> Result<BTreeMap<NativeContinueTurnKey, Vec<i64>>> {
    todo!("campaign/native_continue_pending_groups.go:nativeContinueSpawnSchedule")
}

/// Go：`native_continue_pending_groups.go:259`，`func nativeContinueFieldSpawnRuleIsLive( state *battle.State, assetState *battle.State, rule battle.NativeFieldEventRule, ) bool`
pub(crate) fn native_continue_field_spawn_rule_is_live(
    state: Option<&battle::State>,
    asset_state: Option<&battle::State>,
    rule: &battle::NativeFieldEventRule,
) -> bool {
    todo!("campaign/native_continue_pending_groups.go:nativeContinueFieldSpawnRuleIsLive")
}

/// Go：`native_continue_pending_groups.go:289`，`func cloneNativeContinueFutureUnit(source *battle.Unit) (*battle.Unit, error)`
pub(crate) fn clone_native_continue_future_unit(source: Option<&Unit>) -> Result<Unit> {
    todo!("campaign/native_continue_pending_groups.go:cloneNativeContinueFutureUnit")
}
