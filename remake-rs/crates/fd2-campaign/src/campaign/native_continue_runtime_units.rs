//! Go 來源：`remake/internal/campaign/native_continue_runtime_units.go`（210 行）。

use super::*;

/// MaterializeNativeContinueRuntimeUnits projects the saved 0x50-byte runtime
/// roster fields needed by the current battle into battle.Units without
/// consulting FDFIELD constructor
/// rows. The saved list order is authoritative. Raw +7 is consumed separately
/// as the battle-figure selector and as the CONTINUE selector-cache rebuild key;
/// raw +8 is promoted to persistent identity only for native camp 2.
///
/// This adapter deliberately does not seed map timing, switch to interactive
/// range mode, bind future FDFIELD groups, or publish the Game/controller
/// handoff.
///
/// Go：`native_continue_runtime_units.go:22`，`func MaterializeNativeContinueRuntimeUnits( state *battle.State, input fdsave.ContinueRuntimeInput, catalog *NativeCharacterCatalog, ) error`
pub fn materialize_native_continue_runtime_units(
    state: Option<&mut battle::State>,
    input: &save::ContinueRuntimeInput,
    catalog: &mut NativeCharacterCatalog,
) -> Result<()> {
    todo!("campaign/native_continue_runtime_units.go:MaterializeNativeContinueRuntimeUnits")
}

/// Go：`native_continue_runtime_units.go:94`，`func materializeNativeContinueRuntimeUnit( record fdsave.ContinueRuntimeRecord, fieldWidth, fieldHeight int, catalog *NativeCharacterCatalog, ) (*battle.Unit, error)`
pub(crate) fn materialize_native_continue_runtime_unit(
    record: save::ContinueRuntimeRecord,
    field_width: i64,
    field_height: i64,
    catalog: &NativeCharacterCatalog,
) -> Result<Unit> {
    todo!("campaign/native_continue_runtime_units.go:materializeNativeContinueRuntimeUnit")
}
