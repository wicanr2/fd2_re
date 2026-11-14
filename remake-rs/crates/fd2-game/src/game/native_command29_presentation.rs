//! Go 來源：`remake/cmd/fd2/native_command29_presentation.go`（585 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command29_presentation.go:14`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeCommand29TargetPresentation {
    pub(crate) unit: Option<UnitIdx>,
    pub(crate) base_before: Option<Arc<Picture>>,
    pub(crate) base_after: Option<Arc<Picture>>,
    pub(crate) idle_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) idle_positions: Vec<[i64; 2]>,
    pub(crate) idle_delays: Vec<i64>,
    pub(crate) transition_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) hp_before: i64,
    pub(crate) raw_byte5_before: u8,
    pub(crate) damage_published: bool,
}

/// Go：`native_command29_presentation.go:26`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand29PresentationJob {
    pub(crate) command28: Option<NativeCommand28Sequence>,
    pub(crate) hd: Option<ModernHDSplitStrike>,
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDerivedStrikePlan>,
    pub(crate) schedule: figani::NativeCommandDerivedStrikeSchedule,
    pub(crate) effect_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) effect_positions: Vec<[i64; 2]>,
    pub(crate) prelude_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) prelude_frame: i64,
    pub(crate) actor_base_before: Option<Arc<Picture>>,
    pub(crate) actor_base_after: Option<Arc<Picture>>,
    pub(crate) targets: Vec<NativeCommand29TargetPresentation>,
    pub(crate) target_index: i64,
    pub(crate) transition_frame: i64,
    pub(crate) frame: i64,
    pub(crate) repeat: i64,
    pub(crate) idle_frame: i64,
    pub(crate) idle_repeat: i64,
    pub(crate) shake_counter: i64,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) sfx_actor: Vec<u8>,
    pub(crate) sfx_target: Vec<u8>,
    pub(crate) actor_cues: BTreeMap<i64, Vec<u8>>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommand24Damage>) + Send>>,
}

impl Game {
    /// Go：`native_command29_presentation.go:409`，`func (g *Game) stepNativeCommand29Presentation()`
    pub(crate) fn step_native_command29_presentation(&mut self) {
        todo!("native_command29_presentation.go:stepNativeCommand29Presentation")
    }
}
