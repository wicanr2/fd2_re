//! Go 來源：`remake/internal/campaign/native_story_dialogue_coverage.go`（96 行）。

use super::*;

/// NativeStoryDialogueFrameBounds derives presentation bounds from the same
/// opaque writes as the native frame. The restored background and cursor tail
/// carry no dialogue border.
///
/// Go：`native_story_dialogue_coverage.go:14`，`func NativeStoryDialogueFrameBounds(cells []fdother.RawCell, control string, phase NativeDialogueFramePhase, index int) (image.Rectangle, error)`
pub fn native_story_dialogue_frame_bounds(
    cells: &[fdother::RawCell],
    control: &str,
    phase: NativeDialogueFramePhase,
    index: i64,
) -> Result<go_image::Rectangle> {
    todo!("campaign/native_story_dialogue_coverage.go:NativeStoryDialogueFrameBounds")
}

/// Go：`native_story_dialogue_coverage.go:33`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct NativeDialogueFramePhase(pub u8);

/// Go 原式：`iota`
pub const NATIVE_DIALOGUE_OPENING: NativeDialogueFramePhase = NativeDialogueFramePhase(0);

pub const NATIVE_DIALOGUE_STEADY: NativeDialogueFramePhase = NativeDialogueFramePhase(1);

pub const NATIVE_DIALOGUE_CLOSING: NativeDialogueFramePhase = NativeDialogueFramePhase(2);

/// NativeStoryDialogueCoverage follows the actual opaque raw-cell writes.
/// Closing covers the five snapshot restores, without the optional cursor tail.
///
/// Go：`native_story_dialogue_coverage.go:43`，`func NativeStoryDialogueCoverage(cells []fdother.RawCell, control string, phase NativeDialogueFramePhase, index int) ([]bool, error)`
pub fn native_story_dialogue_coverage(
    cells: &[fdother::RawCell],
    control: &str,
    phase: NativeDialogueFramePhase,
    index: i64,
) -> Result<Vec<bool>> {
    todo!("campaign/native_story_dialogue_coverage.go:NativeStoryDialogueCoverage")
}
