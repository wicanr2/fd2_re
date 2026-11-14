//! Go 來源：`remake/internal/campaign/native_continue_field_boundary.go`（108 行）。

use super::*;

/// MaterializeNativeContinueFieldBoundary installs only the live FDFIELD
/// control/view/HUD boundary which is already closed for title CONTINUE.
///
/// It deliberately does not materialize saved runtime 0x50 records, schedule
/// the two opening redraws, or start the battle driver. The caller must select
/// the chapter asset before calling and pass its raw chapter explicitly; a
/// mismatched asset fails before State is changed.
///
/// Go：`native_continue_field_boundary.go:17`，`func MaterializeNativeContinueFieldBoundary( state *battle.State, input fdsave.ContinueRuntimeInput, assetChapter int, ) error`
pub fn materialize_native_continue_field_boundary(
    state: Option<&mut battle::State>,
    input: &save::ContinueRuntimeInput,
    asset_chapter: i64,
) -> Result<()> {
    todo!("campaign/native_continue_field_boundary.go:MaterializeNativeContinueFieldBoundary")
}
