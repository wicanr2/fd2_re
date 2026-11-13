//! Go 來源：`remake/internal/battle/native_ai_mode_record.go`（27 行）。

use super::*;

/// NativeAIModeRecordForUnit materializes only the raw fields consumed by
/// 0x13a9f / fdother.PlanNativeUnitMode. Every source must have explicit
/// provenance; normalized Camp, class, AI flags, or scenario strings are not
/// substitutes for the constructor bytes.
///
/// Go：`native_ai_mode_record.go:9`，`func NativeAIModeRecordForUnit(unit *Unit) ([]byte, error)`
pub fn native_ai_mode_record_for_unit(unit: Option<&Unit>) -> Result<Vec<u8>> {
    let Some(unit) = unit.filter(|u| {
        u.has_native_record_byte5
            && u.has_native_record_byte34
            && u.has_native_record_byte35
            && u.has_native_record_byte36
            && u.has_native_record_race
    }) else {
        return Err(battle_err!("native AI mode record lacks raw provenance"));
    };
    let mut record = vec![0u8; 0x3e];
    record[5] = unit.native_record_byte5;
    record[0x34] = unit.native_record_byte34;
    record[0x35] = unit.native_record_byte35;
    record[0x36] = unit.native_record_byte36;
    // 0x10fcb copies FDFIELD b2 to runtime +0x3d; the same source is already
    // preserved as NativeRecordRace by the constructor/export boundary.
    record[0x3d] = unit.native_record_race;
    Ok(record)
}

#[cfg(test)]
mod native_ai_mode_record_test;
