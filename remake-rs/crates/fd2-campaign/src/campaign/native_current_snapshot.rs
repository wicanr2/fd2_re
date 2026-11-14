//! Go 來源：`remake/internal/campaign/native_current_snapshot.go`（135 行）。

use super::*;

/// BuildNativeCurrentRuntimeRecords overlays the proven live Unit fields on an
/// exact current-runtime raw baseline. Unknown bytes remain byte-identical.
/// This is intentionally narrower than a general Unit serializer: it accepts
/// only a stable CONTINUE-projected roster in original record order.
///
/// Go：`native_current_snapshot.go:15`，`func BuildNativeCurrentRuntimeRecords( state *battle.State, ) ([]fdsave.PersistentRecord, error)`
pub fn build_native_current_runtime_records(state: Option<&battle::State>) -> Result<Vec<save::PersistentRecord>> {
    todo!("campaign/native_current_snapshot.go:BuildNativeCurrentRuntimeRecords")
}

/// BuildNativeCurrentPersistentRecords extends the exact persistent baseline
/// only for newly appended own-camp JOIN records. Enemy/ally reinforcements are
/// battle-local and do not consume persistent slots. Every new player record is
/// rebuilt by the proven JOIN constructor; all pre-existing and unused bytes
/// remain byte-identical to the loaded snapshot.
///
/// Go：`native_current_snapshot.go:80`，`func BuildNativeCurrentPersistentRecords( state *battle.State, baseline fdsave.CurrentSnapshot, table NativeJoinConstructorTable, itemRows []byte, ) ([fdsave.RosterUnits]fdsave.PersistentRecord, byte, error)`
pub fn build_native_current_persistent_records(
    state: Option<&battle::State>,
    baseline: &save::CurrentSnapshot,
    table: &NativeJoinConstructorTable,
    item_rows: &[u8],
) -> Result<([save::PersistentRecord; 32], u8)> {
    todo!("campaign/native_current_snapshot.go:BuildNativeCurrentPersistentRecords")
}
