//! Go 來源：`remake/cmd/fd2/save.go`（422 行）。
//!
//! save.go — 存檔/讀檔(自有 JSON 格式,doc19;非破解原版 FD2.SAV)。
//! 語意:存「campaign 節點邊界」進度(目前節點/旗標/金幣/道具)。戰鬥中我方可操作時另附
//! battle_snapshot(battle_snapshot.go),讀檔回到同一回合;其他時刻戰鬥中拒絕存檔。
//!
//! 骨架：Go 檔的 13 個函式中產生 7 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`save.go:21`，`func saveSlotPath(slot int) string`
pub(crate) fn save_slot_path(slot: i64) -> String {
    todo!("save.go:saveSlotPath")
}

/// writeSaveFile replaces the save in one rename. Campaign progress is only
/// persisted at node boundaries, so a truncated JSON file must never turn a
/// valid town/preparation save into an unreadable slot after a process stop.
///
/// Go：`save.go:35`，`func writeSaveFile(path string, raw []byte) error`
pub(crate) fn write_save_file(path: &str, raw: &[u8]) -> Result<()> {
    todo!("save.go:writeSaveFile")
}

impl Game {
    /// Go：`save.go:133`，`func (g *Game) saveGameToSlot(slot int) error`
    pub(crate) fn save_game_to_slot(&mut self, slot: i64) -> Result<()> {
        todo!("save.go:saveGameToSlot")
    }

    /// saveGameToSlotFrom 的 fromSystemMenu 由戰鬥系統選單的存檔入口使用；見
    /// battleSnapshotBlockerFor。
    ///
    /// Go：`save.go:137`，`func (g *Game) saveGameToSlotFrom(slot int, fromSystemMenu bool) error`
    pub(crate) fn save_game_to_slot_from(&mut self, slot: i64, from_system_menu: bool) -> Result<()> {
        todo!("save.go:saveGameToSlotFrom")
    }

    /// Go：`save.go:348`，`func (g *Game) clearShopTransientStateForLoad()`
    pub(crate) fn clear_shop_transient_state_for_load(&mut self) {
        todo!("save.go:clearShopTransientStateForLoad")
    }

    /// Go：`save.go:392`，`func (g *Game) clearChurchTransientStateForLoad()`
    pub(crate) fn clear_church_transient_state_for_load(&mut self) {
        todo!("save.go:clearChurchTransientStateForLoad")
    }

    /// Go：`save.go:417`，`func (g *Game) restoreNativeMapHUDGateA(gateA byte)`
    pub(crate) fn restore_native_map_hud_gate_a(&mut self, gate_a: u8) {
        todo!("save.go:restoreNativeMapHUDGateA")
    }
}
