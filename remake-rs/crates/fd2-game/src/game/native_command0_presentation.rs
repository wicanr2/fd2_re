//! Go 來源：`remake/cmd/fd2/native_command0_presentation.go`（546 行）。
//!
//! 骨架：Go 檔的 9 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command0_presentation.go:17`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand0PresentationPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND0_PRELUDE: NativeCommand0PresentationPhase = NativeCommand0PresentationPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND0_ACTOR: NativeCommand0PresentationPhase = NativeCommand0PresentationPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND0_TARGET: NativeCommand0PresentationPhase = NativeCommand0PresentationPhase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND0_TAIL: NativeCommand0PresentationPhase = NativeCommand0PresentationPhase(3);

/// Go：`native_command0_presentation.go:26`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand0PresentationJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) target: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) prelude: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_black: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_pulse: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_specs: Vec<battlepresent::NativeCommand0ActorFrame>,
    pub(crate) target_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) target_impact: Vec<bool>,
    pub(crate) tail: Vec<Option<Arc<Picture>>>,
    pub(crate) hd: Option<ModernHDCommand0Frames>,
    pub(crate) phase: NativeCommand0PresentationPhase,
    pub(crate) frame: i64,
    pub(crate) pulse_black: bool,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) damage_stages: i64,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: i64,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
}

impl Game {
    /// Go：`native_command0_presentation.go:420`，`func (g *Game) stepNativeCommand0Presentation()`
    pub(crate) fn step_native_command0_presentation(&mut self) {
        todo!("native_command0_presentation.go:stepNativeCommand0Presentation")
    }
}
