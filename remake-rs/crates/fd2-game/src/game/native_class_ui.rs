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
    ///
    /// R5 階段 2 r5d：`drawNativeStoryDialogue` 經 `presentNativeStoryDialogueFrame` 呼叫這裡（分組表列在「城鎮與設施」，
    /// 本批沒有別組移植）。`frame` 改收 `&[u8]`：Go 只讀不寫。`g.nativeClassUI` 為 nil 時 Go 解參考 panic，Rust 同樣 panic。
    pub(crate) fn present_native_class_frame(&mut self, screen: &mut Picture, frame: &[u8]) {
        let palette =
            self.native_class_ui.as_ref().expect("presentNativeClassFrame：nativeClassUI 為 nil").palette.clone();
        self.present_native_class_frame_with_palette(screen, frame, &palette);
    }

    /// Go：`native_class_ui.go:210`，`func (g *Game) presentNativeClassFrameWithPalette( screen *ebiten.Image, frame []byte, palette color.Palette, )`
    ///
    /// R5 階段 2 r5d：`frame` 改收 `&[u8]`。`clearModernHDText`、`appendModernHDFacilityMoney` 在 `modernHDText` 為 nil
    /// （忠實模式）時都不做事；HD 時 `todo!("r6:…")`。
    pub(crate) fn present_native_class_frame_with_palette(
        &mut self,
        screen: &mut Picture,
        frame: &[u8],
        palette: &ColorPalette,
    ) {
        self.native_interface_hud_visible = false;
        self.native_map_presented = false;
        if self.modern_hd_text.is_some() {
            todo!("r6:modern_hd_text.go:clearModernHDText、modern_hd_facility_money.go:appendModernHDFacilityMoney");
        }
        // Go `image.NewPaletted(320×200)` 後 `copy(paletted.Pix, frame)`：只複製兩者較短的長度，其餘為 0。
        let mut pix = vec![0u8; 320 * 200];
        let n = pix.len().min(frame.len());
        pix[..n].copy_from_slice(&frame[..n]);
        let paletted = IndexedFrame { width: 320, height: 200, pix, palette: palette.clone() };
        let native = Picture::from_indexed(&paletted);
        let mut op = DrawImageOptions::default();
        op.geo_m.scale(2.0, 2.0);
        screen.draw_image(&native, &op);
    }
}
