//! Go 來源：`remake/cmd/fd2/modern_hd_movement_panel.go`（119 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_movement_panel.go:17`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernHDMovementPanel {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativePlayerMovement>,
    pub(crate) source_image: Option<Arc<Picture>>,
    pub(crate) record: Vec<u8>,
    pub(crate) raw: Vec<u8>,
    pub(crate) palette: Vec<u8>,
    pub(crate) x: i64,
    pub(crate) locale: String,
    pub(crate) primary: Option<Font>,
    pub(crate) font: Option<Font>,
    pub(crate) primary_sf: Option<Arc<SfntFont>>,
    pub(crate) font_sf: Option<Arc<SfntFont>>,
    pub(crate) recipe: Option<ModernBattlePanelText>,
    pub(crate) seal: [u8; 32],
}

/// Go：`modern_hd_movement_panel.go:32`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDMovementPanelDraw {
    pub(crate) rect: Rectangle,
    pub(crate) hd: bool,
    pub(crate) art_hd: bool,
    pub(crate) key: String,
}

/// Go：`modern_hd_movement_panel.go:39`，`func movementPanelSourceSeal(h *modernHDMovementPanel) [32]byte`
pub(crate) fn movement_panel_source_seal(h: Option<&mut ModernHDMovementPanel>) -> [u8; 32] {
    todo!("modern_hd_movement_panel.go:movementPanelSourceSeal")
}

impl Game {
    /// Go：`modern_hd_movement_panel.go:49`，`func (g *Game) freezeModernHDMovementPanel(actor *battle.Unit, plan *battle.NativePlayerMovement, source *ebiten.Image, record, raw []byte, x int, panels []*modernBattlePanelText) (*modernHDMovementPanel, error)`
    pub(crate) fn freeze_modern_hd_movement_panel(
        &mut self,
        actor: Option<UnitIdx>,
        plan: Option<&mut battle::NativePlayerMovement>,
        source: Option<&mut Picture>,
        record: &[u8],
        raw: &[u8],
        x: i64,
        panels: &mut [ModernBattlePanelText],
    ) -> Result<Option<ModernHDMovementPanel>> {
        todo!("modern_hd_movement_panel.go:freezeModernHDMovementPanel")
    }
}
