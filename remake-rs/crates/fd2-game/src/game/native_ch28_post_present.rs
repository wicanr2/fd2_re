//! Go 來源：`remake/cmd/fd2/native_ch28_post_present.go`（363 行）。
//!
//! 骨架：Go 檔的 9 個函式中產生 5 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativeCh28PostPresentJob owns sub_1DB65's 13 pose frames, two six-frame
/// FDOTHER#5 overlay loops, raw sample index 3, and final 0x11CAC redraw. The
/// address-shaped name avoids inventing a player-facing effect name.
///
/// Go：`native_ch28_post_present.go:18`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCh28PostPresentJob {
    pub(crate) work_frames: Vec<Vec<u8>>,
    pub(crate) vga_frames: Vec<Vec<u8>>,
    pub(crate) state_frames: Vec<battle::State>,
    pub(crate) waits: Vec<i64>,
    pub(crate) palette: ColorPalette,
    pub(crate) frame: i64,
    pub(crate) wait: i64,
    pub(crate) drawn: bool,
    pub(crate) sfx_at: i64,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) rollback: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) hd_inputs: Vec<indexedmap::NativeFrameInput<'static>>,
    pub(crate) hd_stage: Option<ModernHDDeathStageSource>,
    pub(crate) hd_targets: Vec<battle::NativeCh28PostOverlayTarget>,
    pub(crate) hd_frames: Vec<ModernHDMapFrame>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
    pub(crate) presented_frame: i64,
    pub(crate) presented_hd: bool,
    /// ordinary settlement keeps its live State/Unit identities
    pub(crate) retained_state: Option<battle::State>,
    pub(crate) kind: String,
}

/// Go：`native_ch28_post_present.go:41`，`func cloneNativeCh28PostState(source *battle.State) (*battle.State, error)`
pub(crate) fn clone_native_ch28_post_state(source: &mut battle::State) -> Result<Option<battle::State>> {
    todo!("native_ch28_post_present.go:cloneNativeCh28PostState")
}

impl Game {
    /// Go：`native_ch28_post_present.go:45`，`func (g *Game) nativeCh28PostFrameInput(state *battle.State) (indexedmap.NativeFrameInput, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ch28_post_frame_input(
        &mut self,
        state: Option<&mut battle::State>,
    ) -> Result<indexedmap::NativeFrameInput<'static>> {
        todo!("native_ch28_post_present.go:nativeCh28PostFrameInput")
    }

    /// Go：`native_ch28_post_present.go:69`，`func (g *Game) startNativeUnitDeathPresentMode(prelude func(*battle.State) error, then func(), retainState bool) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_native_unit_death_present_mode(
        &mut self,
        prelude: &mut dyn FnMut(&battle::State) -> Result<()>,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
        retain_state: bool,
    ) -> Result<()> {
        todo!("native_ch28_post_present.go:startNativeUnitDeathPresentMode")
    }

    /// Go：`native_ch28_post_present.go:283`，`func (g *Game) publishNativeCh28PostFrame()`
    pub(crate) fn publish_native_ch28_post_frame(&mut self) {
        todo!("native_ch28_post_present.go:publishNativeCh28PostFrame")
    }

    /// Go：`native_ch28_post_present.go:305`，`func (g *Game) stepNativeCh28PostPresent()`
    pub(crate) fn step_native_ch28_post_present(&mut self) {
        todo!("native_ch28_post_present.go:stepNativeCh28PostPresent")
    }
}
