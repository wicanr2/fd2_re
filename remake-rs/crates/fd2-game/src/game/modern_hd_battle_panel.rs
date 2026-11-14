//! Go 來源：`remake/cmd/fd2/modern_hd_battle_panel.go`（246 行）。
//!
//! 骨架：Go 檔的 9 個函式中產生 6 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Frozen at the native panel write, before settlement or a later HP step.
/// The native image is also the exact source identity; no glyph recognition.
///
/// Go：`modern_hd_battle_panel.go:19`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernBattlePanelText {
    pub(crate) native: figani::Frame,
    pub(crate) name: String,
    pub(crate) values: battle::NativeBattlePanelValues,
    pub(crate) chrome: ModernBattlePanelChrome,
    pub(crate) background: u8,
    pub(crate) digit_ink: [u8; 2],
    pub(crate) font: Option<Font>,
    pub(crate) last_palette: [u8; 32],
    pub(crate) last_scale: f64,
    pub(crate) picture: Option<Rgba>,
    pub(crate) key: String,
}

impl Game {
    /// Go：`modern_hd_battle_panel.go:56`，`func (g *Game) renderPhysicalBattlePanel(scene *nativePhysicalScene, record, dst []byte, index int) error`
    pub(crate) fn render_physical_battle_panel(
        &mut self,
        scene: &mut NativePhysicalScene,
        record: &mut [u8],
        dst: &mut [u8],
        index: i64,
    ) -> Result<()> {
        todo!("modern_hd_battle_panel.go:renderPhysicalBattlePanel")
    }

    /// Go：`modern_hd_battle_panel.go:60`，`func (g *Game) renderBattlePanelText(assets battle.NativeItemPanelDataAssets, record, dst []byte, index, chapter int, collected *[]*modernBattlePanelText) error`
    pub(crate) fn render_battle_panel_text(
        &mut self,
        assets: &battle::NativeItemPanelDataAssets,
        record: &mut [u8],
        dst: &mut [u8],
        index: i64,
        chapter: i64,
        collected: Option<&mut Vec<ModernBattlePanelText>>,
    ) -> Result<()> {
        todo!("modern_hd_battle_panel.go:renderBattlePanelText")
    }
}

/// Bind only a complete, byte-identical panel at its proven native origin.
/// Partial scroll views inherit the explicit layer and clip from their producer.
///
/// Go：`modern_hd_battle_panel.go:123`，`func bindModernPhysicalPanelText(scene *nativePhysicalScene, sources *modernHDBattleSources)`
pub(crate) fn bind_modern_physical_panel_text(
    scene: Option<&mut NativePhysicalScene>,
    sources: &mut ModernHDBattleSources,
) {
    todo!("modern_hd_battle_panel.go:bindModernPhysicalPanelText")
}

/// Go：`modern_hd_battle_panel.go:129`，`func bindModernBattlePanelText(panels []*modernBattlePanelText, sources *modernHDBattleSources)`
pub(crate) fn bind_modern_battle_panel_text(
    panels: &mut [ModernBattlePanelText],
    sources: Option<&mut ModernHDBattleSources>,
) {
    todo!("modern_hd_battle_panel.go:bindModernBattlePanelText")
}

impl ModernBattlePanelText {
    /// Go：`modern_hd_battle_panel.go:163`，`func (p *modernBattlePanelText) raster(palette color.Palette) (image.Image, string, error)`
    pub(crate) fn raster(&mut self, palette: &ColorPalette) -> Result<(Option<Arc<Picture>>, String)> {
        todo!("modern_hd_battle_panel.go:raster")
    }
}

/// Go：`modern_hd_battle_panel.go:224`，`func drawModernBattlePanelInk(dst *image.RGBA, f *Font, value string, rect image.Rectangle, maximum int, ink color.RGBA) error`
pub(crate) fn draw_modern_battle_panel_ink(
    dst: &mut Rgba,
    f: &mut Font,
    value: &str,
    rect: Rectangle,
    maximum: i64,
    ink: ColorRgba,
) -> Result<()> {
    todo!("modern_hd_battle_panel.go:drawModernBattlePanelInk")
}
