//! Go 來源：`remake/cmd/fd2/native_command5_presentation.go`（577 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command5_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand5PresentationPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND5_COMMON_PRELUDE: NativeCommand5PresentationPhase = NativeCommand5PresentationPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND5_ACTOR: NativeCommand5PresentationPhase = NativeCommand5PresentationPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND5_HANDLER: NativeCommand5PresentationPhase = NativeCommand5PresentationPhase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND5_COMMON_TAIL: NativeCommand5PresentationPhase = NativeCommand5PresentationPhase(3);

/// Go：`native_command5_presentation.go:24`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeCommand5HandlerFrame {
    pub(crate) image: Option<Arc<Picture>>,
    pub(crate) target_index: i64,
    pub(crate) hp_stage: i64,
    pub(crate) play_primary: bool,
    pub(crate) play_secondary: bool,
}

/// Go：`native_command5_presentation.go:32`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand5PresentationJob {
    pub(crate) command_id: i64,
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) prelude: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_black: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_pulse: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_specs: Vec<battlepresent::NativeCommand0ActorFrame>,
    pub(crate) handler: Vec<NativeCommand5HandlerFrame>,
    pub(crate) tail: Vec<Option<Arc<Picture>>>,
    pub(crate) hd: Option<ModernHDCommand5Frames>,
    pub(crate) phase: NativeCommand5PresentationPhase,
    pub(crate) frame: i64,
    pub(crate) pulse_black: bool,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: Vec<i64>,
    pub(crate) actor_sfx: Vec<u8>,
    pub(crate) target_sfx: Vec<u8>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
}

impl Game {
    /// Go：`native_command5_presentation.go:441`，`func (g *Game) stepNativeCommand5Presentation()`
    pub(crate) fn step_native_command5_presentation(&mut self) {
        todo!("native_command5_presentation.go:stepNativeCommand5Presentation")
    }
}
