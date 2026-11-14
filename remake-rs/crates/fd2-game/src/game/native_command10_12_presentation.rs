//! Go 來源：`remake/cmd/fd2/native_command10_12_presentation.go`（346 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command10_12_presentation.go:16`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand1012Phase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND1012_PRELUDE: NativeCommand1012Phase = NativeCommand1012Phase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND1012_MAIN: NativeCommand1012Phase = NativeCommand1012Phase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND1012_RESULT: NativeCommand1012Phase = NativeCommand1012Phase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND1012_HOLD: NativeCommand1012Phase = NativeCommand1012Phase(3);

/// Go：`native_command10_12_presentation.go:25`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand1012Job {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) schedule: fdother::NativeCommand10To12Schedule,
    pub(crate) main_frames: Vec<Vec<u8>>,
    pub(crate) result_frames: Vec<Vec<u8>>,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) phase: NativeCommand1012Phase,
    pub(crate) frame: i64,
    pub(crate) hold: i64,
    pub(crate) drawn: bool,
    pub(crate) prelude_done: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: Vec<i64>,
    pub(crate) rng_before: u16,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
    pub(crate) result_sources: Vec<Vec<indexedmap::NativeCommandTailLayer>>,
    pub(crate) hd_input: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_baseline: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_input_err: Option<GameError>,
    pub(crate) hd_scene: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_signature: [u8; 32],
    pub(crate) hd_frames: Option<ModernHDCommand1012Frames>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
}

impl Game {
    /// Go：`native_command10_12_presentation.go:231`，`func (g *Game) stepNativeCommand1012Presentation()`
    pub(crate) fn step_native_command1012_presentation(&mut self) {
        todo!("native_command10_12_presentation.go:stepNativeCommand1012Presentation")
    }
}
