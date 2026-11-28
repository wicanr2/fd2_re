//! Go 來源：`remake/cmd/fd2/modern_hd_status_text.go`（50 行）。
//!
//! 骨架：Go 檔的 2 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// This snapshot belongs to the open panel. Live unit state, display themes,
/// and the indexed animation buffers have independent lifetimes.
///
/// Go：`modern_hd_status_text.go:15`
///
/// HD 型別：保留忠實模式用到的欄位；HD 的 `itemChrome`、`itemValues`、`rowLabels`、`itemIcons` 在 R6 補上。
/// 不 derive `Debug`／`PartialEq`：`battle::NativeItemPanelDataAssets` 沒有實作。
#[derive(Clone, Default)]
pub(crate) struct NativeUnitStatusData {
    pub(crate) record: Vec<u8>,
    pub(crate) base: Vec<u8>,
    pub(crate) item_rows: Vec<u8>,
    pub(crate) assets: battle::NativeItemPanelDataAssets,
    pub(crate) command_ids: Vec<i64>,
    pub(crate) command_book: Vec<battle::NativeCommandRecord>,
}

/// Go：`modern_hd_status_text.go:11`
pub(crate) const NATIVE_PLAYER_STATUS_COMMANDS: &str = "commands";

impl NativeUnitStatusData {
    /// 在底圖上畫能力值，再畫物品列（`commands=false`）或指令列（`commands=true`）。
    ///
    /// Go：`modern_hd_status_text.go:26`，`func (d *nativeUnitStatusData) compose(assets battle.NativeItemPanelDataAssets, commands bool) ([]byte, bool)`
    pub(crate) fn compose(&self, assets: &battle::NativeItemPanelDataAssets, commands: bool) -> Option<Vec<u8>> {
        if self.base.len() != 320 * 200 {
            return None;
        }
        let mut frame = self.base.clone();
        battle::render_native_item_panel_data(assets, &self.record, &mut frame).ok()?;
        if commands {
            if self.command_ids.is_empty() {
                return None;
            }
            battle::render_native_command_overlay(assets, &self.command_ids, &self.command_book, -1, &mut frame)
                .ok()?;
        } else {
            battle::render_native_item_panel_rows(assets, &self.record, -1, &self.item_rows, &mut frame).ok()?;
        }
        Some(frame)
    }
}
