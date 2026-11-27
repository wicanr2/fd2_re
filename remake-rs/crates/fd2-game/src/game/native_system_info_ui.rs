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

/// Go：`native_system_info_ui.go:43`，`func nativeCurrentRuntimeSaveExists() bool`
pub(crate) fn native_current_runtime_save_exists() -> bool {
    let path = native_current_save_path();
    !path.is_empty() && std::fs::metadata(&path).is_ok_and(|info| !info.is_dir())
}

impl Game {
    /// 巢狀系統選單（0x19DF7）的四格：存檔閘門由任一已離場我方記錄（+5 bit0 清、bit7 設）決定，讀檔閘門看
    /// 目前戰況存檔是否存在；四格圖都要在 FDOTHER #2，否則不開。
    ///
    /// Go：`native_system_info_ui.go:52`，`func (g *Game) nativeNestedSystemOverlayState() (fdother.ActionOverlayState, bool)`
    pub(crate) fn native_nested_system_overlay_state(&self) -> Option<fdother::ActionOverlayState> {
        let st = self.st.as_ref()?;
        let mut save_gate_set = false;
        for unit in &st.units {
            if !unit.has_native_record_byte5 {
                return None;
            }
            if unit.native_record_byte5 & 1 == 0 && unit.native_record_byte5 & 0x80 != 0 {
                save_gate_set = true;
            }
        }
        let state =
            fdother::native_nested_system_action_overlay_state(save_gate_set, !native_current_runtime_save_exists());
        for direction in 0..4 {
            match state.cell_index(direction) {
                Ok(index)
                    if index >= 0 && self.native_action_cells.get(index as usize).is_some_and(Option::is_some) => {}
                _ => return None,
            }
        }
        Some(state)
    }

    /// 戰場資訊（0x1B41D）：章節、回合、金錢與三方人數，加上本章兩行目標文字。
    ///
    /// Go：`native_system_info_ui.go:77`，`func (g *Game) prepareNativeSystemInfoUI() (*nativeSystemInfoUIState, error)`
    ///
    /// Go 在 HD 介面或非繁中語系時另外預備 HD 合成（`prepareModernHDSystemInfo`，R6）；Rust 的 HD 版尚未移植，
    /// 一律只準備原版索引畫格（見 `draw_native_system_info_ui`）。
    pub(crate) fn prepare_native_system_info_ui(&self) -> Result<NativeSystemInfoUIState> {
        let incomplete = || game_err!("native system info: runtime source is incomplete");
        let (Some(assets), Some(st)) = (self.native_system_info_assets.as_ref(), self.st.as_ref()) else {
            return Err(incomplete());
        };
        if self.native_map_vga.len() != fdother::NATIVE_SYSTEM_INFO_BYTES
            || self.native_map_dac.len() != 256 * 3
            || !(0..30).contains(&self.handler_chapter)
            || !(0..=0xff).contains(&st.native_round_counter)
            || !(0..=99_999_999).contains(&self.gold)
        {
            return Err(incomplete());
        }
        let counts = campaign::native_system_info_camp_counts(&st.units)?;
        let mut text_index = 2 * self.handler_chapter + 0x255;
        if self.handler_chapter == 16 && !self.party_members.get(&18).copied().unwrap_or(false) {
            text_index -= 2;
        }
        let strings = assets
            .numbers
            .strings
            .as_ref()
            .ok_or_else(|| game_err!("native system info: FDTXT strings unavailable"))?;
        let lines = [strings.words(text_index)?, strings.words(text_index + 1)?];
        let input = campaign::NativeSystemInfoInput {
            raw_chapter: self.handler_chapter as u8,
            raw_round: st.native_round_counter as u8,
            currency: self.gold as u32,
            raw_camp_counts: counts,
            lines,
        };
        let information = campaign::compose_native_system_info_surface(assets, &input)?;
        let (opening, closing) = fdother::native_system_info_transition_frames(&self.native_map_vga, &information)?;
        let steady = fdother::native_system_info_steady_frame(&self.native_map_vga, &information)?;
        let dac = self.native_map_dac.clone();
        let palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&dac)?);
        Ok(NativeSystemInfoUIState {
            opening,
            steady,
            closing,
            phase: String::from("opening"),
            dac,
            palette,
            input,
            text_index,
            ..Default::default()
        })
    }

    /// Go：`native_system_info_ui.go:177`，`func (g *Game) closeNativeSystemInfoUI()`
    pub(crate) fn close_native_system_info_ui(&mut self) {
        if let Some(state) = self.native_system_info_ui.as_mut().filter(|state| state.phase == "steady") {
            (state.phase, state.frame, state.drawn) = (String::from("closing"), 0, false);
        }
    }

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
        // HD 介面的 `drawModernHDSystemInfoUI` 屬 R6，尚未移植；接上前先畫原版索引畫格。
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
