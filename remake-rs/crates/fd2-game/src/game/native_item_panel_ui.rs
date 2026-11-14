//! Go 來源：`remake/cmd/fd2/native_item_panel_ui.go`（985 行）。
//!
//! 骨架：Go 檔的 24 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// Go：`native_item_panel_ui.go:156`，`func (g *Game) clearNativeItemPanel()`
    pub(crate) fn clear_native_item_panel(&mut self) {
        todo!("native_item_panel_ui.go:clearNativeItemPanel")
    }
}

/// applyNativeAITargetItem consumes the complete 0x1567e winner list at the
/// 0x15055 boundary. It deliberately bypasses the player-owned cursor/modal:
/// the original AI owner rebuilds the whole list from its saved destination
/// and passes that list to 0x20c6f. All mutable records stay detached until
/// item, target and effect provenance have passed validation.
///
/// Go：`native_item_panel_ui.go:433`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeAIItemTransaction {
    pub(crate) before: Vec<u8>,
    pub(crate) after: Vec<u8>,
    pub(crate) targets: Vec<UnitIdx>,
    pub(crate) restore: Option<battle::NativeRawRestoreBatch>,
    pub(crate) damage: Vec<battle::NativeCommandDamage>,
    pub(crate) damage_route: Option<battle::NativeItemCommandDamageRoute>,
    pub(crate) rng_before: u16,
    pub(crate) rng_after: u16,
}
