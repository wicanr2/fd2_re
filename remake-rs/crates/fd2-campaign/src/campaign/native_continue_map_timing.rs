//! Go 來源：`remake/internal/campaign/native_continue_map_timing.go`（73 行）。

use super::*;

/// MaterializeNativeContinueMapTiming installs the exact process-global seed
/// observed immediately before sub_10010's first 0x11CAC redraw. It does not
/// sample a clock or schedule either redraw; those are caller-owned operations.
///
/// Field and runtime-unit adapters must already have succeeded. This preserves
/// the original order and prevents a timing-only partial CONTINUE state.
///
/// Go：`native_continue_map_timing.go:18`，`func MaterializeNativeContinueMapTiming( state *battle.State, input fdsave.ContinueRuntimeInput, ) error`
pub fn materialize_native_continue_map_timing(
    state: Option<&mut battle::State>,
    input: &save::ContinueRuntimeInput,
) -> Result<()> {
    todo!("campaign/native_continue_map_timing.go:MaterializeNativeContinueMapTiming")
}
