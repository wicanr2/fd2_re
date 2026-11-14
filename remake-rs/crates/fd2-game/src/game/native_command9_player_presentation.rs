//! Go 來源：`remake/cmd/fd2/native_command9_player_presentation.go`（300 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command9_player_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand9PlayerPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND9_PLAYER_PALETTE: NativeCommand9PlayerPhase = NativeCommand9PlayerPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND9_PLAYER_EFFECT: NativeCommand9PlayerPhase = NativeCommand9PlayerPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND9_PLAYER_RESULT: NativeCommand9PlayerPhase = NativeCommand9PlayerPhase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND9_PLAYER_HOLD: NativeCommand9PlayerPhase = NativeCommand9PlayerPhase(3);

/// Go：`native_command9_player_presentation.go:24`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand9PlayerJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) target: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) palettes: Vec<ColorPalette>,
    pub(crate) effect_frames: Vec<Vec<u8>>,
    pub(crate) result_frames: Vec<Vec<u8>>,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) phase: NativeCommand9PlayerPhase,
    pub(crate) frame: i64,
    pub(crate) hold: i64,
    pub(crate) drawn: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) target_hp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) rng_before: u16,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
    pub(crate) effect_sources: Vec<Vec<indexedmap::NativeCommandTailLayer>>,
    pub(crate) result_sources: Vec<Vec<indexedmap::NativeCommandTailLayer>>,
    pub(crate) hd_input: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_input_err: Option<GameError>,
    pub(crate) hd_scene: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_frames: Option<ModernHDPlayer9Frames>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
}

impl Game {
    /// Go：`native_command9_player_presentation.go:172`，`func (g *Game) stepNativeCommand9PlayerPresentation()`
    pub(crate) fn step_native_command9_player_presentation(&mut self) {
        todo!("native_command9_player_presentation.go:stepNativeCommand9PlayerPresentation")
    }
}
