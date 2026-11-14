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
