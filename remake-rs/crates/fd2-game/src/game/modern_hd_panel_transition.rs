//! Go 來源：`remake/cmd/fd2/modern_hd_panel_transition.go`（209 行）。
//!
//! 骨架：Go 檔的 7 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_panel_transition.go:12`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeStatusTransitionKind(pub(crate) u8);

/// Go：`modern_hd_panel_transition.go:20`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativeStatusTransitionText {
    pub(crate) kind: NativeStatusTransitionKind,
    pub(crate) commands: bool,
}

/// Go：`modern_hd_panel_transition.go:14`..`17`
pub(crate) const NATIVE_STATUS_OPENING: NativeStatusTransitionKind = NativeStatusTransitionKind(0);
pub(crate) const NATIVE_STATUS_CLOSING: NativeStatusTransitionKind = NativeStatusTransitionKind(1);
pub(crate) const NATIVE_STATUS_TO_COMMANDS: NativeStatusTransitionKind = NativeStatusTransitionKind(2);

/// HD 狀態轉場文字的快照。只有 HD 繪製（R6）讀它；忠實模式只保留種類與頁別，`source`／`data` 的
/// 複本隨 HD 欄位在 R6 補上。輸入不完整時與 Go 相同回 `None`。
///
/// Go：`modern_hd_panel_transition.go:27`，`func snapshotNativeStatusTransition(source []byte, data *nativeUnitStatusData, kind nativeStatusTransitionKind, commands bool) *nativeStatusTransitionText`
pub(crate) fn snapshot_native_status_transition(
    source: &[u8],
    data: Option<&NativeUnitStatusData>,
    kind: NativeStatusTransitionKind,
    commands: bool,
) -> Option<NativeStatusTransitionText> {
    if source.len() != 64000 || data.is_none() {
        return None;
    }
    Some(NativeStatusTransitionText { kind, commands })
}
