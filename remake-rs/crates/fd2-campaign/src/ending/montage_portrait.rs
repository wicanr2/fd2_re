//! Go 來源：`remake/internal/ending/montage_portrait.go`（162 行）。

use super::*;

pub const NATIVE_MONTAGE_PORTRAIT_OFFSET: i64 = 0xc88;

pub const NATIVE_MONTAGE_PORTRAIT_NORMAL_FRAME: i64 = 0;

pub const NATIVE_MONTAGE_PORTRAIT_MOUTH_FRAME: i64 = 3;

pub const NATIVE_MONTAGE_PORTRAIT_SPLIT_TICK: i64 = 0xdc;

pub const NATIVE_MONTAGE_PORTRAIT_REGULAR_TICKS: i64 = 0xdc;

pub const NATIVE_MONTAGE_PORTRAIT_FINAL_TICKS: i64 = 0x1b8;

/// MontagePortraitState is the exact local byte countdown at
/// 0x2c7ed..0x2c9a9. RandomByte is caller-supplied because native obtains it
/// from 0x4e893; this adapter does not invent or share a replacement RNG.
///
/// Go：`montage_portrait.go:22`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct MontagePortraitState {
    pub countdown: i64,
}

impl MontagePortraitState {
    /// Step returns the DATO frame selected for one native portrait-loop
    /// iteration. A zero countdown is replaced with (random&31)+40 without an
    /// immediate decrement; nonzero countdowns decrement first. Values 1 and 0
    /// select pointer-table offset 12, i.e. DATO frame 3.
    ///
    /// Go：`montage_portrait.go:30`，`func (s *MontagePortraitState) Step(randomByte byte) (int, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn step(&mut self, random_byte: u8) -> Result<i64> {
        todo!("ending/montage_portrait.go:MontagePortraitState.Step")
    }
}

/// NativeMontagePortraitIterations preserves the caller's asymmetric duration:
/// native loop index zero receives 440 ticks; every other party entry receives
/// 220. Since PlanPartyCycle swaps indexes zero and one, this is intentionally
/// keyed by LoopIndex rather than UnitSlot.
///
/// Go：`montage_portrait.go:49`，`func NativeMontagePortraitIterations(loopIndex int) (int, error)`
pub fn native_montage_portrait_iterations(loop_index: i64) -> Result<i64> {
    todo!("ending/montage_portrait.go:NativeMontagePortraitIterations")
}

/// ComposeMontagePortraitFrame executes the recovered static portion of one
/// 0x2c7ed portrait tick: restore the FDOTHER#5 dialogue-grid snapshot, paste
/// the selected DATO frame at 0x0c88, and render the five proven FDTXT fields.
/// It returns a fresh 320x200 indexed frame so malformed inputs never partially
/// mutate the caller's restore buffer.
///
/// Go：`montage_portrait.go:64`，`func ComposeMontagePortraitFrame( m Montage, restore []byte, unit []byte, edi, datoFrame int, portraitFrames []dato.Frame, current, permanent *fdtxt.Strings, font *fdtxt.Font, ) ([]byte, error)`
pub fn compose_montage_portrait_frame(
    m: &Montage,
    restore: &[u8],
    unit: &[u8],
    edi: i64,
    dato_frame: i64,
    portrait_frames: &[dato::Frame],
    current: Option<&fdtxt::Strings>,
    permanent: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
) -> Result<Vec<u8>> {
    todo!("ending/montage_portrait.go:ComposeMontagePortraitFrame")
}

/// Go：`montage_portrait.go:122`，`func renderMontageText( frame []byte, font *fdtxt.Font, words []uint16, origin, lineAdvanceRows int, style MontageGlyphStyle, ) error`
pub(crate) fn render_montage_text(
    frame: &mut [u8],
    font: &fdtxt::Font,
    words: &[u16],
    origin: i64,
    line_advance_rows: i64,
    style: MontageGlyphStyle,
) -> Result<()> {
    todo!("ending/montage_portrait.go:renderMontageText")
}
