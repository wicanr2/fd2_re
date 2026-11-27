//! Go 來源：`remake/cmd/fd2/native_command_modifier_presentation.go`（138 行）。
//!
//! 三個函式都已移植；HD 綁定（`bindModernHDModifierPalette`、`stageModernHDCompoundMap`）與其他指令的 Rust 版相同
//! 另期，這裡只畫原版索引色畫面。

use super::*;

/// Go：`native_command_modifier_presentation.go:13`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommandModifierPresentationJob {
    pub(crate) command_id: i64,
    pub(crate) baseline: Vec<u8>,
    pub(crate) palettes: Vec<ColorPalette>,
    pub(crate) phase: i64,
    pub(crate) drawn: bool,
    pub(crate) transaction: Option<Box<dyn FnMut(&mut Game) -> Result<()> + Send>>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) hd_frames: Vec<NativeCompoundPresentedFrame>,
    pub(crate) unit_present: Option<NativeUnitPresentJob>,
}

impl Game {
    /// 玩家專屬的 0x1D6C8 前奏，指令 17～23、25～27 共用：FDOTHER #80 selector0，接著四次指令色／黑色輪替 DAC
    /// 第 0 色。呼叫端的完整交易預檢在音效與調色盤輸出之前跑，八個階段都被畫面承認之前不寫狀態。
    ///
    /// Go：`native_command_modifier_presentation.go:30`，`func (g *Game) startNativeCommandPalettePresentation(commandID int, preflight func() error, transaction func() error, then func()) error`
    pub(crate) fn start_native_command_palette_presentation(
        &mut self,
        command_id: i64,
        preflight: impl FnOnce(&Game) -> Result<()>,
        transaction: Box<dyn FnMut(&mut Game) -> Result<()> + Send>,
        then: Box<dyn FnOnce(&mut Game) + Send>,
    ) -> Result<()> {
        if !self.current_native_system_options().full_presentation_enabled() {
            return Err(game_err!("native abbreviated presentation owner unavailable"));
        }
        let valid_command = (17..=23).contains(&command_id) || (25..=27).contains(&command_id);
        if self.st.is_none() || !valid_command {
            return Err(game_err!("native command modifier presentation context unavailable"));
        }
        if self.native_modifier_presentation.is_some() || self.native_heal_presentation.is_some() {
            return Err(game_err!("native command modifier presentation already active"));
        }
        if self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
            || self.native_map_dac.len() != 256 * 3
            || self.native_command_palette_flash.is_none()
        {
            return Err(game_err!("native command modifier framebuffer/DAC table unavailable"));
        }
        if self.native_map_dac[..3] != [0, 0, 0] {
            return Err(game_err!("native command modifier baseline DAC entry zero is not black"));
        }
        preflight(self)?;
        if !os_mute_or_shot(Some(self)) && self.sfx_command_modifier.is_empty() {
            return Err(game_err!("native command modifier FDOTHER #80 selector0 unavailable"));
        }
        let phases =
            self.native_command_palette_flash.as_ref().unwrap().native_command_palette_flash_phases(command_id)?;
        let mut palettes = Vec::with_capacity(phases.len());
        for rgb in phases {
            let mut dac = self.native_map_dac.clone();
            dac[..3].copy_from_slice(&rgb);
            palettes.push(color_palette_from_nrgba(&fdother::vga_palette_from_dac(&dac)?));
        }
        self.native_modifier_presentation = Some(NativeCommandModifierPresentationJob {
            command_id,
            baseline: self.native_map_vga.clone(),
            palettes,
            transaction: Some(transaction),
            then: Some(then),
            ..Default::default()
        });
        let sample = self.sfx_command_modifier.clone();
        self.play_raw(&sample);
        Ok(())
    }

    /// 原版索引色分支：以目前階段的調色盤重畫進場時的畫面。
    ///
    /// Go：`native_command_modifier_presentation.go:117`，`func (g *Game) drawNativeCommandModifierPresentation(screen *ebiten.Image) bool`
    pub(crate) fn present_native_command_modifier(&mut self, screen: Option<&mut Picture>) -> bool {
        let Some(j) = self.native_modifier_presentation.as_mut() else { return false };
        let phase = j.phase;
        if phase < 0
            || phase >= j.palettes.len() as i64
            || j.baseline.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
            || j.palettes[phase as usize].len() != 256
        {
            return false;
        }
        if let Some(screen) = screen {
            let image = Picture::from_indexed(&IndexedFrame {
                width: 320,
                height: 200,
                pix: j.baseline.clone(),
                palette: j.palettes[phase as usize].clone(),
            });
            let mut draw = DrawImageOptions::default();
            draw.geo_m.scale(2.0, 2.0);
            screen.draw_image(&image, &draw);
        }
        j.drawn = true;
        true
    }

    /// Go：`native_command_modifier_presentation.go:95`，`func (g *Game) stepNativeCommandModifierPresentation()`
    pub(crate) fn step_native_command_modifier_presentation(&mut self) {
        let Some(j) = self.native_modifier_presentation.as_mut() else {
            return;
        };
        if !j.drawn {
            return;
        }
        j.drawn = false;
        j.phase += 1;
        if j.phase < j.palettes.len() as i64 {
            return;
        }
        // Go 在工作仍掛在 `g.nativeModifierPresentation` 時呼叫交易，之後才清掉；nil func 解參考 panic。
        // Go 的 `then := j.then` 讀的是同一個工作，交易改不到它，所以先取出等價。
        let mut transaction = j.transaction.take().expect("Go：j.transaction 為 nil 時呼叫 panic");
        let then = j.then.take();
        let err = transaction(self);
        self.native_modifier_presentation = None;
        if let Err(err) = err {
            self.msg = format!("原始指令 modifier transaction 失敗: {}", err.message());
            return;
        }
        if let Some(then) = then {
            then(self);
        }
    }
}
