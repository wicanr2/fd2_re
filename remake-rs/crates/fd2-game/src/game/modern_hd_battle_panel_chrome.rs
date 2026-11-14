//! Go 來源：`remake/cmd/fd2/modern_hd_battle_panel_chrome.go`（112 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Frozen with the panel, so later settlement cannot change a bar already shown.
/// Colors retain the source palette; geometry is the HD presentation contract.
///
/// Go：`modern_hd_battle_panel_chrome.go:14`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct ModernBattlePanelChrome {
    pub(crate) ready: bool,
    pub(crate) border: u8,
    pub(crate) track: u8,
    pub(crate) ink: [u8; 2],
    pub(crate) width: [i64; 2],
    pub(crate) frame: [u8; 6258],
}

impl Default for ModernBattlePanelChrome {
    fn default() -> Self {
        Self {
            ready: Default::default(),
            border: Default::default(),
            track: Default::default(),
            ink: Default::default(),
            width: Default::default(),
            frame: std::array::from_fn(|_| Default::default()),
        }
    }
}

/// Go：`modern_hd_battle_panel_chrome.go:22`，`func freezeModernBattlePanelChrome(assets battle.NativeItemPanelDataAssets, values battle.NativeBattlePanelValues) (modernBattlePanelChrome, error)`
pub(crate) fn freeze_modern_battle_panel_chrome(
    assets: &battle::NativeItemPanelDataAssets,
    values: battle::NativeBattlePanelValues,
) -> Result<ModernBattlePanelChrome> {
    todo!("modern_hd_battle_panel_chrome.go:freezeModernBattlePanelChrome")
}

impl ModernBattlePanelChrome {
    /// Go：`modern_hd_battle_panel_chrome.go:55`，`func (c modernBattlePanelChrome) paint(canvas *image.RGBA, palette color.Palette, background byte) error`
    /// Go 值接收者：本體對接收者的改寫只影響副本。
    pub(crate) fn paint(&self, canvas: &mut Rgba, palette: &ColorPalette, background: u8) -> Result<()> {
        todo!("modern_hd_battle_panel_chrome.go:paint")
    }
}
