//! Go 來源：`remake/cmd/fd2/native_indexed_transition.go`（361 行）。
//!
//! 骨架：Go 檔的 11 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_indexed_transition.go:17`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeIndexedTransitionPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_TRANSITION_PASS: NativeIndexedTransitionPhase = NativeIndexedTransitionPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_TRANSITION_TAIL: NativeIndexedTransitionPhase = NativeIndexedTransitionPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_TRANSITION_PALETTE: NativeIndexedTransitionPhase = NativeIndexedTransitionPhase(2);

/// nativeIndexedTransitionJob owns one complete 0x24618 presentation. Indexed
/// pixels and the six-bit DAC remain separate, just as they do in mode 13h.
/// Draw acknowledges every native present so multiple 5/4ms operations cannot
/// collapse into an unobservable final frame on the host's 60Hz update clock.
///
/// Go：`native_indexed_transition.go:29`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeIndexedTransitionJob {
    pub(crate) input: indexedmap::NativeTransitionFrameInput<'static>,
    pub(crate) schedule: fdother::NativeIndexedTransitionSchedule,
    pub(crate) spec: campaign::HandlerIndexedTransition,
    pub(crate) work: Vec<u8>,
    pub(crate) vga: Vec<u8>,
    pub(crate) frame_vga: Vec<Vec<u8>>,
    pub(crate) frame_work: Vec<Vec<u8>>,
    pub(crate) dac: Vec<u8>,
    pub(crate) baseline: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) phase: NativeIndexedTransitionPhase,
    pub(crate) frame: i64,
    pub(crate) palette_step: i64,
    pub(crate) tail_ticks: i64,
    pub(crate) drawn: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) hd_source: Option<ModernHDIndexedTransitionSource>,
    pub(crate) hd_signature: [u8; 32],
    pub(crate) hd_frames: Vec<ModernHDMapFrame>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
    pub(crate) hd_cached: Vec<ModernHDMapFrame>,
    pub(crate) hd_cached_theme: Option<ModernStoryPortraitSet>,
}

impl Game {
    /// Go：`native_indexed_transition.go:126`，`func (g *Game) buildNativeIndexedTransitionInputForActors(actorsSource []battle.Unit) (indexedmap.NativeTransitionFrameInput, error)`
    pub(crate) fn build_native_indexed_transition_input_for_actors(
        &mut self,
        actors_source: &[battle::Unit],
    ) -> Result<indexedmap::NativeTransitionFrameInput<'static>> {
        todo!("native_indexed_transition.go:buildNativeIndexedTransitionInputForActors")
    }

    /// buildNativeIndexedTransitionInputForState preserves the battle-session
    /// selector cache and animation/timing globals.  Reconstructing those from a
    /// flat unit slice would reset a turn-four transition to the chapter-opening
    /// phase even when every unit record was otherwise correct.
    ///
    /// Go：`native_indexed_transition.go:147`，`func (g *Game) buildNativeIndexedTransitionInputForState(state *battle.State) (indexedmap.NativeTransitionFrameInput, error)`
    pub(crate) fn build_native_indexed_transition_input_for_state(
        &mut self,
        state: Option<&mut battle::State>,
    ) -> Result<indexedmap::NativeTransitionFrameInput<'static>> {
        todo!("native_indexed_transition.go:buildNativeIndexedTransitionInputForState")
    }

    /// Go：`native_indexed_transition.go:260`，`func (g *Game) stepNativeIndexedTransition()`
    pub(crate) fn step_native_indexed_transition(&mut self) {
        todo!("native_indexed_transition.go:stepNativeIndexedTransition")
    }
}
