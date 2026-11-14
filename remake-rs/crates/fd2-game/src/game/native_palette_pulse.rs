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

impl Game {
    /// Go：`native_palette_pulse.go:133`，`func (g *Game) stepNativePalettePulse()`
    pub(crate) fn step_native_palette_pulse(&mut self) {
        todo!("native_palette_pulse.go:stepNativePalettePulse")
    }
}
