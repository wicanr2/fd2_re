//! Go 來源：`remake/internal/battle/native_coordinate_write.go`（14 行）。

use super::*;

/// SetNativeUnitCoordinateBytes reproduces the state write in 0x22253 used by
/// command-23 relocation: the final native coordinates are written to raw
/// record bytes +0 and +1.  Indexed renderer/presentation work is outside this
/// byte-level adapter.
///
/// Go：`native_coordinate_write.go:7`，`func SetNativeUnitCoordinateBytes(records []byte, unitIndex int, x, y byte) error`
pub fn set_native_unit_coordinate_bytes(records: &mut [u8], unit_index: i64, x: u8, y: u8) -> Result<()> {
    todo!("battle/native_coordinate_write.go:SetNativeUnitCoordinateBytes")
}
