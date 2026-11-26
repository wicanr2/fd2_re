//! Go 來源：`remake/cmd/fd2/native_palette_pulse.go`（170 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativePalettePulseJob presents baseline-relative full-range DAC writes,
/// including 0x35E5A's pulse and individual 0x11DF2 steps from 0x25052.
/// Every write must be drawn before advancing; indexed VGA bytes stay constant.
///
/// Go：`native_palette_pulse.go:17`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativePalettePulseJob {
    pub(crate) deltas: Vec<i64>,
    pub(crate) waits: Vec<i64>,
    pub(crate) step: i64,
    pub(crate) wait: i64,
    pub(crate) vga: Vec<u8>,
    pub(crate) dac: Vec<u8>,
    pub(crate) baseline: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) drawn: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

impl NativePalettePulseJob {
    /// R4 B 組隨呼叫端補移植（原標 `r8`）：Go `native_palette_pulse.go:114`，`func (j *nativePalettePulseJob) applyCurrent() error`。
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn apply_current(&mut self) -> Result<()> {
        if self.step < 0 || self.step >= self.deltas.len() as i64 || self.waits.len() != self.deltas.len() {
            return Err(GameError::new("native 0x35E5A palette step unavailable"));
        }
        fdother::apply_vga_palette_delta(&mut self.dac, &self.baseline, 0, 255, self.deltas[self.step as usize])?;
        let palette = fdother::vga_palette_from_dac(&self.dac)?;
        (self.palette, self.drawn) = (color_palette_from_nrgba(&palette), false);
        self.wait = self.waits[self.step as usize] - 1;
        if self.wait < 0 {
            self.wait = 0;
        }
        Ok(())
    }
}

/// Go：`native_palette_pulse.go:30`，`func validateNativePalettePulse(spec campaign.NativePalettePulse) bool`
pub(crate) fn validate_native_palette_pulse(spec: &campaign::NativePalettePulse) -> bool {
    spec.rise_start == 0
        && spec.rise_end == 63
        && spec.rise_delay_ms == 8
        && spec.hold_ms == 400
        && spec.fall_start == 62
        && spec.fall_end == 0
        && spec.fall_delay_ms == 8
}

impl Game {
    pub(super) fn native_palette_presenter_active(&self) -> bool {
        self.native_palette_pulse.is_some()
            || self.native_palette_ramp.is_some()
            || self.native_unit_present.is_some()
            || self.transition_reveal.is_some()
            || self.indexed_transition.is_some()
    }

    /// 呈現一次 0x11DF2(0,255,delta) 寫入；之後的等待由呼叫端下一個 delay beat 負責，本工作只承認 Draw。
    ///
    /// Go：`native_palette_pulse.go:37`，`func (g *Game) startNativePaletteUpdate(delta int, then func()) error`
    pub(crate) fn start_native_palette_update(
        &mut self,
        delta: i64,
        then: Box<dyn FnOnce(&mut Game) + Send>,
    ) -> Result<()> {
        if self.native_palette_presenter_active() {
            return Err(GameError::new("native indexed palette presenter is already active"));
        }
        if !(0..=63).contains(&delta) || !native_map_assets_available(self.native_map_assets.as_ref()) {
            return Err(GameError::new("native palette delta or indexed DAC baseline unavailable"));
        }
        self.ensure_native_palette_frame()?;
        let baseline = self.native_map_assets.as_ref().unwrap().palette_dac.clone();
        let mut job = NativePalettePulseJob {
            deltas: vec![delta],
            waits: vec![1],
            vga: self.native_map_vga.clone(),
            dac: baseline.clone(),
            baseline,
            then: Some(then),
            ..Default::default()
        };
        job.apply_current()?;
        self.native_full_dac_white = false;
        self.native_palette_pulse = Some(job);
        Ok(())
    }

    /// 0x35E5A 全 DAC 脈衝：0..63 上升（63 多停 400ms）、62..0 下降；全部 127 步先預檢再發布第一步。
    ///
    /// Go：`native_palette_pulse.go:62`，`func (g *Game) startNativePalettePulse(spec campaign.NativePalettePulse, then func()) error`
    pub(crate) fn start_native_palette_pulse(
        &mut self,
        spec: &campaign::NativePalettePulse,
        then: Box<dyn FnOnce(&mut Game) + Send>,
    ) -> Result<()> {
        if self.native_palette_presenter_active() {
            return Err(GameError::new("native 0x35E5A palette presenter is already active"));
        }
        if !validate_native_palette_pulse(spec) || !native_map_assets_available(self.native_map_assets.as_ref()) {
            return Err(GameError::new("native indexed framebuffer or DAC baseline unavailable"));
        }
        self.ensure_native_palette_frame()?;
        let baseline = self.native_map_assets.as_ref().unwrap().palette_dac.clone();
        let mut dac = baseline.clone();
        if self.native_map_dac.len() == baseline.len() {
            dac.copy_from_slice(&self.native_map_dac);
        }
        let mut job = NativePalettePulseJob {
            vga: self.native_map_vga.clone(),
            dac: dac.clone(),
            baseline: baseline.clone(),
            then: Some(then),
            ..Default::default()
        };
        for delta in spec.rise_start..=spec.rise_end {
            job.deltas.push(delta);
            let mut wait = native_delay_ticks(spec.rise_delay_ms);
            if delta == spec.rise_end {
                wait += native_delay_ticks(spec.hold_ms);
            }
            job.waits.push(wait);
        }
        let mut delta = spec.fall_start;
        while delta >= spec.fall_end {
            job.deltas.push(delta);
            job.waits.push(native_delay_ticks(spec.fall_delay_ms));
            delta -= 1;
        }
        if job.deltas.len() != 127 || job.waits.len() != job.deltas.len() {
            return Err(GameError::new("native 0x35E5A schedule is incomplete"));
        }
        let mut probe = dac;
        for &delta in &job.deltas {
            fdother::apply_vga_palette_delta(&mut probe, &baseline, 0, 255, delta)?;
            fdother::vga_palette_from_dac(&probe)?;
        }
        job.apply_current()?;
        self.native_palette_pulse = Some(job);
        Ok(())
    }

    /// Go：`native_palette_pulse.go:158`，`func (g *Game) drawNativePalettePulse(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_palette_pulse(&mut self, screen: &mut Picture) -> bool {
        let Some(j) = self.native_palette_pulse.as_mut() else {
            return false;
        };
        if j.vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE || j.palette.len() != 256 {
            return false;
        }
        let img = indexed_frame(320, 200, &j.vga, j.palette.clone());
        let mut options = DrawImageOptions::default();
        options.geo_m.scale(2.0, 2.0);
        screen.draw_image(&Picture::from_indexed(&img), &options);
        j.drawn = true;
        true
    }
}

impl Game {
    /// Go：`native_palette_pulse.go:133`，`func (g *Game) stepNativePalettePulse()`
    ///
    /// R4 B 組：`drawn` 由呈現回報設定（Go 在 `drawNativePalettePulse` 設）。
    pub(crate) fn step_native_palette_pulse(&mut self) {
        let Some(j) = self.native_palette_pulse.as_mut().filter(|j| j.drawn) else {
            return;
        };
        if j.wait > 0 {
            j.wait -= 1;
            return;
        }
        j.step += 1;
        if j.step >= j.deltas.len() as i64 {
            let mut j = self.native_palette_pulse.take().unwrap();
            let then = j.then.take();
            self.native_map_dac.clear();
            self.native_map_dac.extend_from_slice(&j.dac);
            if let Some(then) = then {
                then(self);
            }
            return;
        }
        if let Err(err) = j.apply_current() {
            self.load_err = format!("native 0x35E5A palette pulse: {err}");
            self.native_palette_pulse = None;
        }
    }
}
