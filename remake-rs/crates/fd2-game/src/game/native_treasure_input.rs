//! Go 來源：`remake/cmd/fd2/native_treasure_input.go`（121 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_treasure_input.go:10`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeTreasurePrompt {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) reward: battle::Treasure,
    pub(crate) x: i64,
    pub(crate) y: i64,
    pub(crate) await_ack: bool,
    /// ClaimTreasure 已成立（YES 且交易通過）
    pub(crate) claimed: bool,
}
