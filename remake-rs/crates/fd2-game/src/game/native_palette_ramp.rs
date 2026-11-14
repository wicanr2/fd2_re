//! Go 來源：`remake/cmd/fd2/native_palette_ramp.go`（281 行）。
//!
//! 骨架：Go 檔的 10 個函式中產生 8 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativePaletteRampJob owns the exact whole-DAC 0x11d40 writes used by
/// 0x1f882 and 0x1f525. Each step must be presented before it advances; this
/// intentionally stretches sub-frame DOS waits on a 60Hz host rather than
/// collapsing the complete effect into one unobservable update.
///
/// Go：`native_palette_ramp.go:20`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativePaletteRampJob {
    pub(crate) hd_battle_sources: Option<ModernHDBattleSources>,
    pub(crate) deltas: Vec<i64>,
    pub(crate) delay_ms: i64,
    pub(crate) step: i64,
    pub(crate) vga: Vec<u8>,
    pub(crate) dac: Vec<u8>,
    pub(crate) baseline: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) drawn: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

/// Go：`native_palette_ramp.go:33`，`func buildInclusivePaletteDeltas(start, end int) ([]int, error)`
pub(crate) fn build_inclusive_palette_deltas(start: i64, end: i64) -> Result<Vec<i64>> {
    todo!("native_palette_ramp.go:buildInclusivePaletteDeltas")
}

impl Game {
    /// Go：`native_palette_ramp.go:51`，`func (g *Game) startNativePaletteRamp(start, end, delayMs int, then func()) error`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_native_palette_ramp(
        &mut self,
        start: i64,
        end: i64,
        delay_ms: i64,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        todo!("native_palette_ramp.go:startNativePaletteRamp")
    }

    /// startNativePaletteRampOnFrame accepts the caller's current indexed VGA.
    /// Normal physical returns fade their last body present, before the map VGA
    /// is cleared. Contract: physical_owner_return_dac_spec in the canonical evidence.
    ///
    /// Go：`native_palette_ramp.go:67`，`func (g *Game) startNativePaletteRampOnFrame(vga []byte, start, end, delayMs int, then func()) error`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_native_palette_ramp_on_frame(
        &mut self,
        vga: &[u8],
        start: i64,
        end: i64,
        delay_ms: i64,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        todo!("native_palette_ramp.go:startNativePaletteRampOnFrame")
    }

    /// ensureNativePaletteFrame uses the already published steady battle frame
    /// when available. A handler cutscene has no battle State/HUD; its acting
    /// redraw path is the separately recovered terrain → unit/foreground →
    /// 0x11eb0 compositor, represented by the three strict indexedmap operations
    /// below. No normalized PNG pixels are converted into fabricated palette
    /// indices.
    ///
    /// Go：`native_palette_ramp.go:101`，`func (g *Game) ensureNativePaletteFrame() error`
    pub(crate) fn ensure_native_palette_frame(&mut self) -> Result<()> {
        todo!("native_palette_ramp.go:ensureNativePaletteFrame")
    }
}

impl NativePaletteRampJob {
    /// Go：`native_palette_ramp.go:128`，`func (j *nativePaletteRampJob) applyCurrent() error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn apply_current(&mut self) -> Result<()> {
        todo!("native_palette_ramp.go:applyCurrent")
    }
}

impl Game {
    /// Go：`native_palette_ramp.go:143`，`func (g *Game) stepNativePaletteRamp()`
    pub(crate) fn step_native_palette_ramp(&mut self) {
        todo!("native_palette_ramp.go:stepNativePaletteRamp")
    }

    /// Go：`native_palette_ramp.go:164`，`func (g *Game) drawNativePaletteRamp(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_palette_ramp(&mut self, screen: &mut Picture) -> bool {
        todo!("native_palette_ramp.go:drawNativePaletteRamp")
    }
}

/// Go：`native_palette_ramp.go:190`，`func nativeDACIsBlack(dac []byte) bool`
pub(crate) fn native_dacis_black(dac: &[u8]) -> bool {
    todo!("native_palette_ramp.go:nativeDACIsBlack")
}
