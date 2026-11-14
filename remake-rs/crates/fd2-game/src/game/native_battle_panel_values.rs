//! Go 來源：`remake/cmd/fd2/native_battle_panel_values.go`（54 行）。
//!
//! 骨架：Go 檔的 1 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_battle_panel_values.go:10`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeBattlePanelValueKey {
    pub(crate) level: i64,
    pub(crate) hp: i64,
    pub(crate) max_hp: i64,
    pub(crate) mp: i64,
    pub(crate) max_mp: i64,
}
