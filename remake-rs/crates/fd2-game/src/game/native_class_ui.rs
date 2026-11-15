//! Go 來源：`remake/cmd/fd2/native_class_ui.go`（244 行）。
//!
//! 骨架：Go 檔的 10 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_class_ui.go:16`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeClassUIAssets {
    pub(crate) background: Vec<u8>,
    pub(crate) entries: Vec<fdother::LMI1Entry>,
    pub(crate) panel: fdother::LMI1Entry,
    pub(crate) price_cell: fdother::RawCell,
    pub(crate) choices: Vec<fdother::RawCell>,
    pub(crate) dialogue: Vec<fdother::RawCell>,
    pub(crate) digits: Vec<fdother::Frame>,
    pub(crate) portrait: dato::Frame,
    pub(crate) portraits: Vec<dato::Frame>,
    pub(crate) units: Option<fdicon::Bank>,
    pub(crate) strings: Option<fdtxt::Strings>,
    pub(crate) font: Option<fdtxt::Font>,
    pub(crate) palette: ColorPalette,
    pub(crate) palette_dac: Vec<u8>,
    pub(crate) revive_fx: Vec<fdother::Frame>,
}

/// Go：`native_class_ui.go:34`，`func loadNativeClassUIAssets() (*nativeClassUIAssets, error)`
pub(crate) fn load_native_class_ui_assets() -> Result<Option<NativeClassUIAssets>> {
    let church_assets = fdother::load_separated_church_ui_assets(separated_asset_path("ui"))?;
    let mut background = vec![0u8; 320 * 200];
    church_assets.background.blit_at(&mut background, 320, 0, -1)?;
    let strings = fdtxt::load_separated_resource(separated_asset_path("text"), 0)?;
    let font = fdtxt::load_separated_font(separated_asset_path("fonts"))?;
    let shared_ui = fdother::load_separated_item_panel_entries(separated_asset_path("ui"))?;
    let mut dialogue = Vec::with_capacity(20);
    for index in 0..=19 {
        let Some(cell) = shared_ui.raw.get(&index) else {
            return Err(game_err!("native class UI: separated FDOTHER #5 dialogue is incomplete"));
        };
        dialogue.push(cell.clone());
    }
    let mut digits = Vec::with_capacity(10);
    for digit in 0..10 {
        let Some(frame) = shared_ui.frames.get(&(31 + digit)) else {
            return Err(game_err!("native class UI: separated FDOTHER #5 digits are incomplete"));
        };
        digits.push(frame.clone());
    }
    let portraits = load_native_separated_portrait(131)?;
    if portraits.is_empty() {
        return Err(game_err!("native church UI: DATO#131 has no frames"));
    }
    let units = fdicon::load_separated_bank(separated_asset_path("sprites/fdicon"))?;
    let choices = fdother::load_separated_action_cells(separated_asset_path("ui"))?;
    let (palette_raw, mut palette) = load_native_battle_palette()?;
    // Go `palette[0] = color.NRGBA{A: 0xff}`：不透明黑。
    palette[0] = ColorRgba { r: 0, g: 0, b: 0, a: 0xff };
    let panel = church_assets.entries[16].clone();
    Ok(Some(NativeClassUIAssets {
        background,
        entries: church_assets.entries,
        panel,
        price_cell: church_assets.price_cell,
        choices,
        dialogue,
        digits,
        portrait: portraits[0].clone(),
        portraits,
        units: Some(units),
        strings: Some(strings),
        font: Some(font),
        palette,
        palette_dac: palette_raw,
        revive_fx: church_assets.revive_fx,
    }))
}

#[cfg(test)]
mod native_class_ui_test;

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// Go：`native_class_ui.go:206`，`func (g *Game) presentNativeClassFrame(screen *ebiten.Image, frame []byte)`
    pub(crate) fn present_native_class_frame(&mut self, screen: &mut Picture, frame: &mut [u8]) {
        todo!("r5:native_class_ui.go:presentNativeClassFrame")
    }

    /// Go：`native_class_ui.go:210`，`func (g *Game) presentNativeClassFrameWithPalette( screen *ebiten.Image, frame []byte, palette color.Palette, )`
    pub(crate) fn present_native_class_frame_with_palette(
        &mut self,
        screen: &mut Picture,
        frame: &mut [u8],
        palette: &ColorPalette,
    ) {
        todo!("r5:native_class_ui.go:presentNativeClassFrameWithPalette")
    }
}
