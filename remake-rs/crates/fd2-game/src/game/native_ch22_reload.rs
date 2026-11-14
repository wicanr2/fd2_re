//! Go 來源：`remake/cmd/fd2/native_ch22_reload.go`（165 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_ch22_reload.go:47`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeCh22ReloadState {
    pub(crate) phase: i64,
    pub(crate) field: Option<MapData>,
    pub(crate) assets: Option<NativeMapAssets>,
    pub(crate) state: Option<battle::State>,
    pub(crate) aux: Vec<u8>,
}
