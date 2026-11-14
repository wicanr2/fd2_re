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
