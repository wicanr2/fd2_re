//! Go 來源：`remake/cmd/fd2/native_command7_presentation.go`（511 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command7_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand7PresentationPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND7_COMMON_PRELUDE: NativeCommand7PresentationPhase = NativeCommand7PresentationPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND7_ACTOR: NativeCommand7PresentationPhase = NativeCommand7PresentationPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND7_HANDLER: NativeCommand7PresentationPhase = NativeCommand7PresentationPhase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND7_COMMON_TAIL: NativeCommand7PresentationPhase = NativeCommand7PresentationPhase(3);

/// Go：`native_command7_presentation.go:24`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeCommand7HandlerFrame {
    pub(crate) image: Option<Arc<Picture>>,
    pub(crate) target_index: i64,
    pub(crate) hp_stage: i64,
    pub(crate) play_handle_a: bool,
    pub(crate) play_handle_b: bool,
}

/// Go：`native_command7_presentation.go:32`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand7PresentationJob {
    pub(crate) hd: Option<ModernHDOrdinaryFrames>,
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) prelude: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_black: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_pulse: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_specs: Vec<battlepresent::NativeCommand0ActorFrame>,
    pub(crate) handler: Vec<NativeCommand7HandlerFrame>,
    pub(crate) tail: Vec<Option<Arc<Picture>>>,
    pub(crate) phase: NativeCommand7PresentationPhase,
    pub(crate) frame: i64,
    pub(crate) pulse_black: bool,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: Vec<i64>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
}

impl Game {
    /// Go：`native_command7_presentation.go:386`，`func (g *Game) stepNativeCommand7Presentation()`
    pub(crate) fn step_native_command7_presentation(&mut self) {
        todo!("native_command7_presentation.go:stepNativeCommand7Presentation")
    }
}
