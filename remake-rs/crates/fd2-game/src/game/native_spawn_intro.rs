//! Go 來源：`remake/cmd/fd2/native_spawn_intro.go`（372 行）。
//!
//! 骨架：Go 檔的 13 個函式中產生 12 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativeSpawnIntroJob owns sub_32999's twelve observable presentation passes.
/// The following 0x1366A call remains the next independent compiled Beat.
///
/// Go：`native_spawn_intro.go:19`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeSpawnIntroJob {
    pub(crate) schedule: Vec<fdother::NativeSpawnIntroStep>,
    pub(crate) work: Vec<u8>,
    pub(crate) vga: Vec<u8>,
    pub(crate) frames: Vec<Vec<u8>>,
    pub(crate) palette: ColorPalette,
    pub(crate) sound: Vec<u8>,
    pub(crate) frame: i64,
    pub(crate) drawn: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

/// Go：`native_spawn_intro.go:31`，`func cloneStoryUnitPointers(source []battle.Unit) []*battle.Unit`
pub(crate) fn clone_story_unit_pointers(source: &[battle::Unit]) -> Vec<UnitIdx> {
    todo!("native_spawn_intro.go:cloneStoryUnitPointers")
}

/// Go：`native_spawn_intro.go:40`，`func cloneBattleUnitPointers(source []*battle.Unit) []*battle.Unit`
pub(crate) fn clone_battle_unit_pointers(source: &[UnitIdx]) -> Vec<UnitIdx> {
    todo!("native_spawn_intro.go:cloneBattleUnitPointers")
}

/// Go：`native_spawn_intro.go:52`，`func cloneNativeBattleSpawnState(source *battle.State) (*battle.State, error)`
pub(crate) fn clone_native_battle_spawn_state(source: Option<&mut battle::State>) -> Result<Option<battle::State>> {
    todo!("native_spawn_intro.go:cloneNativeBattleSpawnState")
}

impl Game {
    /// Go：`native_spawn_intro.go:65`，`func (g *Game) bindNativeFutureItemRows(state *battle.State) error`
    pub(crate) fn bind_native_future_item_rows(&self, state: Option<&mut battle::State>) -> Result<()> {
        todo!("native_spawn_intro.go:bindNativeFutureItemRows")
    }

    /// prepareNativeStorySpawn executes the proven future-group placement against
    /// private clones. The caller commits the returned values only after every
    /// visual pass has preflighted successfully.
    ///
    /// Go：`native_spawn_intro.go:81`，`func (g *Game) prepareNativeStorySpawn(group int, rawGate byte) ([]battle.Unit, []battle.Unit, error)`
    pub(crate) fn prepare_native_story_spawn(
        &self,
        group: i64,
        raw_gate: u8,
    ) -> Result<(Vec<battle::Unit>, Vec<battle::Unit>)> {
        todo!("native_spawn_intro.go:prepareNativeStorySpawn")
    }

    /// Go：`native_spawn_intro.go:114`，`func (g *Game) buildNativeSpawnIntroJob( oldInput, candidateInput indexedmap.NativeTransitionFrameInput, oldCount int, then func(), ) (*nativeSpawnIntroJob, error)`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn build_native_spawn_intro_job(
        &self,
        old_input: &indexedmap::NativeTransitionFrameInput<'_>,
        candidate_input: &indexedmap::NativeTransitionFrameInput<'_>,
        old_count: i64,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<Option<NativeSpawnIntroJob>> {
        todo!("native_spawn_intro.go:buildNativeSpawnIntroJob")
    }

    /// Go：`native_spawn_intro.go:163`，`func (g *Game) startNativeSpawnIntro(group int, rawGate byte, then func()) error`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_native_spawn_intro(
        &mut self,
        group: i64,
        raw_gate: u8,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        todo!("native_spawn_intro.go:startNativeSpawnIntro")
    }
}

/// Go：`native_spawn_intro.go:197`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeBattleIntroEvidence {
    pub(crate) event_id: i64,
    pub(crate) group: i64,
    pub(crate) acting_resource: i64,
    pub(crate) spawn_source: String,
    pub(crate) acting_source: String,
}

/// Go：`native_spawn_intro.go:202`
pub(crate) static NATIVE_BATTLE_INTRO_EVIDENCE_TABLE: LazyLock<Vec<NativeBattleIntroEvidence>> = LazyLock::new(|| {
    vec![
        NativeBattleIntroEvidence {
            event_id: 1,
            group: 4,
            acting_resource: 3,
            spawn_source: String::from("0x342ce"),
            acting_source: String::from("0x342e7"),
        },
        NativeBattleIntroEvidence {
            event_id: 2,
            group: 5,
            acting_resource: 4,
            spawn_source: String::from("0x34336"),
            acting_source: String::from("0x3434f"),
        },
    ]
});

/// nativeBattleIntroCall admits only the two proven global callers. The raw
/// addresses remain authoritative; editable metadata may describe the calls,
/// but it cannot silently redefine faithful-mode executable provenance.
///
/// Go：`native_spawn_intro.go:210`，`func nativeBattleIntroCall(action battle.Action) (battle.NativeSpawnCall, bool, error)`
pub(crate) fn native_battle_intro_call(action: &battle::Action) -> Result<(battle::NativeSpawnCall, bool)> {
    todo!("native_spawn_intro.go:nativeBattleIntroCall")
}

impl Game {
    /// Go：`native_spawn_intro.go:240`，`func (g *Game) loadNativeBattleFollowingActing( meta *battle.NativeFollowingActing, unitCount int, ) ([]campaign.ActingFrame, error)`
    pub(crate) fn load_native_battle_following_acting(
        &self,
        meta: Option<&battle::NativeFollowingActing>,
        unit_count: i64,
    ) -> Result<Vec<campaign::ActingFrame>> {
        todo!("native_spawn_intro.go:loadNativeBattleFollowingActing")
    }

    /// Go：`native_spawn_intro.go:270`，`func (g *Game) startNativeBattleFollowingActing(frames []campaign.ActingFrame, then func())`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_native_battle_following_acting(
        &mut self,
        frames: &mut [campaign::ActingFrame],
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) {
        todo!("native_spawn_intro.go:startNativeBattleFollowingActing")
    }

    /// Go：`native_spawn_intro.go:276`，`func (g *Game) startNativeBattleSpawnIntro( action battle.Action, call battle.NativeSpawnCall, then func(), ) error`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_native_battle_spawn_intro(
        &mut self,
        action: &battle::Action,
        call: &battle::NativeSpawnCall,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        todo!("native_spawn_intro.go:startNativeBattleSpawnIntro")
    }

    /// Go：`native_spawn_intro.go:337`，`func (g *Game) stepNativeSpawnIntro()`
    pub(crate) fn step_native_spawn_intro(&mut self) {
        todo!("native_spawn_intro.go:stepNativeSpawnIntro")
    }
}
