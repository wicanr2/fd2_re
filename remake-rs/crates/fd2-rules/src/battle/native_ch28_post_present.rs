//! Go 來源：`remake/internal/battle/native_ch28_post_present.go`（153 行）。

use super::*;

pub const NATIVE_CH28_POST_VISIBLE_COLUMNS: i64 = 13;

pub const NATIVE_CH28_POST_VISIBLE_ROWS: i64 = 8;

/// NativeCh28PostOverlayTarget is one eligible record and the exact work-buffer
/// destination saved by 0x1DB65 before any pose or inactive-byte writes.
///
/// Go：`native_ch28_post_present.go:17`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCh28PostOverlayTarget {
    pub slot: i64,
    pub origin: i64,
}

/// NativeCh28PostPresentationPlan preserves the raw viewport and the stable
/// record-pointer set used by both six-frame LMI loops. It deliberately carries
/// no inferred effect name.
///
/// Go：`native_ch28_post_present.go:25`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCh28PostPresentationPlan {
    pub camera_x: i64,
    pub camera_y: i64,
    pub targets: Vec<NativeCh28PostOverlayTarget>,
}

/// Go：`native_ch28_post_present.go:30`，`func validateNativeCh28PostPresenterFields(st *State) error`
pub(crate) fn validate_native_ch28_post_presenter_fields(st: Option<&State>) -> Result<()> {
    todo!("battle/native_ch28_post_present.go:validateNativeCh28PostPresenterFields")
}

/// PlanNativeCh28PostPresentation reproduces 0x1DB7C..0x1DC2D. The native
/// upper bounds are cameraX+13 and cameraY+8+1; the lower bounds are camera-1.
/// Empty Targets is valid and selects the original no-presentation branch.
///
/// Go：`native_ch28_post_present.go:64`，`func PlanNativeCh28PostPresentation(st *State, cameraX, cameraY int) (NativeCh28PostPresentationPlan, error)`
pub fn plan_native_ch28_post_presentation(
    st: Option<&State>,
    camera_x: i64,
    camera_y: i64,
) -> Result<NativeCh28PostPresentationPlan> {
    todo!("battle/native_ch28_post_present.go:PlanNativeCh28PostPresentation")
}

/// ApplyNativeCh28PostPoseFrame reproduces the only record write in each of
/// 0x1DB65's thirteen first-phase frames: eligible active records with word40=0
/// receive raw +3 = frame mod 4. The renderer/present/tick boundary is owned by
/// the caller and must succeed before advancing to the next frame.
///
/// Go：`native_ch28_post_present.go:92`，`func ApplyNativeCh28PostPoseFrame(st *State, frame int) error`
pub fn apply_native_ch28_post_pose_frame(st: Option<&mut State>, frame: i64) -> Result<()> {
    todo!("battle/native_ch28_post_present.go:ApplyNativeCh28PostPoseFrame")
}

/// ApplyNativeCh28PostInactiveMark reproduces 0x1DD25..0x1DD50. Every record
/// whose raw word40 is zero receives raw +5=1, including off-viewport records.
///
/// Go：`native_ch28_post_present.go:127`，`func ApplyNativeCh28PostInactiveMark(st *State) error`
pub fn apply_native_ch28_post_inactive_mark(st: Option<&mut State>) -> Result<()> {
    todo!("battle/native_ch28_post_present.go:ApplyNativeCh28PostInactiveMark")
}
