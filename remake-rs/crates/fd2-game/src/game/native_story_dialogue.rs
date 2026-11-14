//! Go 來源：`remake/cmd/fd2/native_story_dialogue.go`（586 行）。
//!
//! 骨架：Go 檔的 12 個函式中產生 9 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// startNativeDialogueFrames 接回 0x165D6 的開框前聚焦；背景須在聚焦後保存。
    ///
    /// Go：`native_story_dialogue.go:16`，`func (g *Game) startNativeDialogueFrames() error`
    pub(crate) fn start_native_dialogue_frames(&mut self) -> Result<()> {
        todo!("native_story_dialogue.go:startNativeDialogueFrames")
    }

    /// Go：`native_story_dialogue.go:93`，`func (g *Game) prepareNativeDialogueFrames() error`
    pub(crate) fn prepare_native_dialogue_frames(&mut self) -> Result<()> {
        todo!("native_story_dialogue.go:prepareNativeDialogueFrames")
    }
}

/// Go：`native_story_dialogue.go:309`，`func loadNativeSeparatedPortrait(speaker int) ([]dato.Frame, error)`
pub(crate) fn load_native_separated_portrait(speaker: i64) -> Result<Vec<dato::Frame>> {
    todo!("native_story_dialogue.go:loadNativeSeparatedPortrait")
}

impl Game {
    /// Go：`native_story_dialogue.go:353`，`func (g *Game) stepNativeStoryDialogueProgress()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn step_native_story_dialogue_progress(&mut self) {
        todo!("native_story_dialogue.go:stepNativeStoryDialogueProgress")
    }

    /// Go：`native_story_dialogue.go:386`，`func (g *Game) beginNativeStoryDialogueClosing() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn begin_native_story_dialogue_closing(&mut self) -> bool {
        todo!("native_story_dialogue.go:beginNativeStoryDialogueClosing")
    }

    /// Go：`native_story_dialogue.go:398`，`func (g *Game) resetNativeStoryDialogueMouth()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn reset_native_story_dialogue_mouth(&mut self) {
        todo!("native_story_dialogue.go:resetNativeStoryDialogueMouth")
    }

    /// Go：`native_story_dialogue.go:408`，`func (g *Game) nativeStoryDialogueAtInputWait() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_story_dialogue_at_input_wait(&self) -> bool {
        todo!("native_story_dialogue.go:nativeStoryDialogueAtInputWait")
    }

    /// Go：`native_story_dialogue.go:419`，`func (g *Game) stepDialogueMouth()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn step_dialogue_mouth(&mut self) {
        todo!("native_story_dialogue.go:stepDialogueMouth")
    }

    /// Go：`native_story_dialogue.go:460`，`func (g *Game) finishNativeStoryDialogueClosing()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn finish_native_story_dialogue_closing(&mut self) {
        todo!("native_story_dialogue.go:finishNativeStoryDialogueClosing")
    }
}
