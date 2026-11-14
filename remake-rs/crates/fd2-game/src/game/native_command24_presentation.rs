//! Go 來源：`remake/cmd/fd2/native_command24_presentation.go`（537 行）。
//!
//! 骨架：Go 檔的 10 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command24_presentation.go:16`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand24PresentationJob {
    pub(crate) hd: Option<ModernHDSplitStrike>,
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) target: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDerivedStrikePlan>,
    pub(crate) schedule: figani::NativeCommand24Schedule,
    pub(crate) effect_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) effect_positions: Vec<[i64; 2]>,
    pub(crate) target_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) target_positions: Vec<[i64; 2]>,
    pub(crate) target_delays: Vec<i64>,
    pub(crate) prelude_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) prelude_frame: i64,
    pub(crate) actor_base_before: Option<Arc<Picture>>,
    pub(crate) actor_base_after: Option<Arc<Picture>>,
    pub(crate) target_base_before: Option<Arc<Picture>>,
    pub(crate) target_base_after: Option<Arc<Picture>>,
    pub(crate) transition_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) transition_frame: i64,
    pub(crate) frame: i64,
    pub(crate) repeat: i64,
    pub(crate) target_frame: i64,
    pub(crate) target_repeat: i64,
    pub(crate) shake_counter: i64,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) damage_published: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: i64,
    pub(crate) target_raw_byte5_before: u8,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommand24Damage>) + Send>>,
}

/// Go：`native_command24_presentation.go:61`，`func nativeCommand24RuntimeUnitIndex(st *battle.State, unit *battle.Unit) (int, error)`
pub(crate) fn native_command24_runtime_unit_index(st: Option<&battle::State>, unit: Option<UnitIdx>) -> Result<i64> {
    todo!("native_command24_presentation.go:nativeCommand24RuntimeUnitIndex")
}

/// Go：`native_command24_presentation.go:89`，`func nativeCommand24PreludeImages(frames []battlepresent.NativeCommand24PreludeFrame) ([]*ebiten.Image, error)`
pub(crate) fn native_command24_prelude_images(
    frames: &[battlepresent::NativeCommand24PreludeFrame],
) -> Result<Vec<Option<Arc<Picture>>>> {
    todo!("native_command24_presentation.go:nativeCommand24PreludeImages")
}

impl Game {
    /// Go：`native_command24_presentation.go:378`，`func (g *Game) stepNativeCommand24Presentation()`
    pub(crate) fn step_native_command24_presentation(&mut self) {
        todo!("native_command24_presentation.go:stepNativeCommand24Presentation")
    }
}
