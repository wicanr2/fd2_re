//! Go 來源：`remake/cmd/fd2/native_system_info_ui.go`（233 行）。
//!
//! 骨架：Go 檔的 7 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_system_info_ui.go:15`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeSystemInfoUIState {
    pub(crate) opening: Vec<Vec<u8>>,
    pub(crate) steady: Vec<u8>,
    pub(crate) closing: Vec<Vec<u8>>,
    pub(crate) phase: String,
    pub(crate) frame: i64,
    pub(crate) drawn: bool,
    pub(crate) dac: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) input: campaign::NativeSystemInfoInput,
    pub(crate) text_index: i64,
    pub(crate) hd: Option<ModernHDSystemInfo>,
    pub(crate) hd_presented: bool,
}

/// Go：`native_system_info_ui.go:30`，`func loadNativeSystemInfoAssets() (*campaign.NativeSystemInfoAssets, error)`
pub(crate) fn load_native_system_info_assets() -> Result<Option<campaign::NativeSystemInfoAssets>> {
    let panels = fdother::load_separated_system_info_panels(separated_asset_path("ui"))?;
    let numbers = battle::load_native_item_panel_data_assets(&separated_asset_path(""))?;
    let font = numbers.font.clone();
    Ok(Some(campaign::NativeSystemInfoAssets { panels, numbers, font }))
}

impl Game {
    /// Go：`native_system_info_ui.go:140`，`func (g *Game) stepNativeSystemInfoUI()`
    pub(crate) fn step_native_system_info_ui(&mut self) {
        let Some(state) = self.native_system_info_ui.as_mut() else {
            return;
        };
        if !state.drawn {
            return;
        }
        state.drawn = false;
        if state.phase == "steady" {
            self.native_fdother_palette_phase = (self.native_fdother_palette_phase + 1) & 15;
            if let Err(err) =
                fdother::apply_native_dac_palette_cycle_e0ef(&mut state.dac, self.native_fdother_palette_phase)
            {
                self.load_err = format!("native system info palette cycle: {err}");
                self.native_system_info_ui = None;
                return;
            }
            match fdother::vga_palette_from_dac(&state.dac) {
                Ok(palette) => state.palette = color_palette_from_nrgba(&palette),
                Err(err) => {
                    self.load_err = format!("native system info palette: {err}");
                    self.native_system_info_ui = None;
                }
            }
            return;
        }
        state.frame += 1;
        let frames = if state.phase == "closing" { &state.closing } else { &state.opening };
        if state.frame < frames.len() as i64 {
            return;
        }
        if state.phase == "opening" {
            (state.phase, state.frame) = (String::from("steady"), 0);
            return;
        }
        self.native_system_info_ui = None;
    }
}

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// Go：`native_system_info_ui.go:186`，`func (g *Game) drawNativeSystemInfoUI(screen *ebiten.Image) bool`
    ///
    /// R5 階段 2 r5h 逐行移植到 `presentNativeClassFrameWithPalette` 與 `state.drawn = true`。開場與第一章只走到
    /// `state == nil` 的守門。HD 介面時 `drawModernHDSystemInfoUI` 是 `todo!("r6:…")`。
    ///
    /// 刻意不照 Go（使用者 2026-11-15 定案，121「貼圖與語系分開」；KNOWN_DIFFERENCES
    /// `system-info-locale-hd-overlay`）：Go 在非繁中語系不管 F2／F11，以 6 倍解析度合成含當地文字的 HD 圖，再縮成
    /// 1/3 疊在原版外框上（`native_system_info_ui.go:208`..`229`）。Rust 在原版美術時所有語系都只畫原版外框的
    /// 索引畫格，當地文字由畫格合成端（`prepareNativeSystemInfoUI`，`r8`）走該語系的一般文字路徑；HD 美術由 R6
    /// 對所有語系一致處理。
    pub(crate) fn draw_native_system_info_ui(&mut self, screen: &mut Picture) -> bool {
        let Some(state) = self.native_system_info_ui.as_mut() else {
            return false;
        };
        state.hd_presented = false;
        if self.modern_hd_interface_enabled() {
            todo!("r6:modern_hd_system_info.go:drawModernHDSystemInfoUI");
        }
        let state = self.native_system_info_ui.as_ref().unwrap();
        let mut frame = &state.steady;
        if state.phase == "opening" && state.frame < state.opening.len() as i64 {
            frame = &state.opening[state.frame as usize];
        } else if state.phase == "closing" && state.frame < state.closing.len() as i64 {
            frame = &state.closing[state.frame as usize];
        }
        if frame.len() != fdother::NATIVE_SYSTEM_INFO_BYTES {
            return false;
        }
        if state.palette.len() != 256 {
            return false;
        }
        let (frame, palette) = (frame.clone(), state.palette.clone());
        self.present_native_class_frame_with_palette(screen, &frame, &palette);
        self.native_system_info_ui.as_mut().unwrap().drawn = true;
        true
    }
}
