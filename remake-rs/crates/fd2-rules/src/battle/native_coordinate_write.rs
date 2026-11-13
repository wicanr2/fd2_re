//! Go 來源：`remake/internal/battle/native_coordinate_write.go`（14 行）。

use super::*;

/// SetNativeUnitCoordinateBytes reproduces the state write in 0x22253 used by
/// command-23 relocation: the final native coordinates are written to raw
/// record bytes +0 and +1.  Indexed renderer/presentation work is outside this
/// byte-level adapter.
///
/// Go：`native_coordinate_write.go:7`，`func SetNativeUnitCoordinateBytes(records []byte, unitIndex int, x, y byte) error`
pub fn set_native_unit_coordinate_bytes(records: &mut [u8], unit_index: i64, x: u8, y: u8) -> Result<()> {
    let base = unit_index.wrapping_mul(NATIVE_RECORD_SIZE);
    if unit_index < 0 || base < 0 || base.wrapping_add(2) > records.len() as i64 {
        return Err(record_bounds_error(unit_index));
    }
    records[base as usize] = x;
    records[base as usize + 1] = y;
    Ok(())
}

#[cfg(test)]
mod native_coordinate_write_test;
