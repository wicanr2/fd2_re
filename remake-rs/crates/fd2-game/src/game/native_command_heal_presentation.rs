//! Go 來源：`remake/cmd/fd2/native_command_heal_presentation.go`（509 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command_heal_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommandHealPresentationPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND_HEAL_FRAMES: NativeCommandHealPresentationPhase = NativeCommandHealPresentationPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_MID_HOLD: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_FRONT_HOLD: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_EFFECT_FRAMES: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(3);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_MASK_FRAMES: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(4);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_DIGIT_FRAMES: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(5);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_DIGIT_HOLD: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(6);

/// Go：`native_command_heal_presentation.go:27`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommandHealPresentationJob {
    /// Original wrapper identity, also used by read-only input audit.
    pub(crate) command_id: i64,
    pub(crate) front_schedule: fdother::NativeCommandHealPresentationSchedule,
    pub(crate) tail_schedule: fdother::NativeCommandHealTailSchedule,
    pub(crate) front_frames: Vec<Vec<u8>>,
    pub(crate) effect_frames: Vec<Vec<u8>>,
    pub(crate) mask_frame: Vec<u8>,
    pub(crate) digit_frames: Vec<Vec<u8>>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) digits: Vec<fdother::LMI1Entry>,
    pub(crate) targets: Vec<indexedmap::NativeCommandHealTailTarget>,
    pub(crate) palette: ColorPalette,
    pub(crate) frame: i64,
    pub(crate) phase: NativeCommandHealPresentationPhase,
    pub(crate) hold: i64,
    pub(crate) drawn: bool,
    pub(crate) transaction: Option<Box<dyn FnMut(&mut Game) -> Result<Vec<battle::NativeCommandHealResult>> + Send>>,
    pub(crate) results: Vec<battle::NativeCommandHealResult>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandHealResult>) + Send>>,
    pub(crate) map_stage: Option<NativeCh23AdapterState>,
    pub(crate) hd_before: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_scene: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_signature: [u8; 32],
    pub(crate) hd_input: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_baseline: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_input_err: Option<GameError>,
    pub(crate) hd_passes: Vec<fdother::IndexedTransitionPass>,
    pub(crate) hd_luts: Vec<Vec<u8>>,
    pub(crate) effect_sources: Vec<Vec<indexedmap::NativeCommandTailLayer>>,
    pub(crate) digit_sources: Vec<Vec<indexedmap::NativeCommandTailLayer>>,
    pub(crate) effect_targets: Vec<indexedmap::NativeCommandHealTailTarget>,
    pub(crate) hd_frames: Option<ModernHDHealFrames>,
    pub(crate) hd_cached: Option<ModernHDHealFrames>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
    pub(crate) hd_cached_theme: Option<ModernStoryPortraitSet>,
}

impl Game {
    /// Go：`native_command_heal_presentation.go:341`，`func (g *Game) stepNativeCommandHealPresentation()`
    pub(crate) fn step_native_command_heal_presentation(&mut self) {
        todo!("native_command_heal_presentation.go:stepNativeCommandHealPresentation")
    }
}
