//! Go 來源：`remake/internal/battle/native_ai_mode_record.go`（27 行）。

use super::*;

/// NativeAIModeRecordForUnit materializes only the raw fields consumed by
/// 0x13a9f / fdother.PlanNativeUnitMode. Every source must have explicit
/// provenance; normalized Camp, class, AI flags, or scenario strings are not
/// substitutes for the constructor bytes.
///
/// Go：`native_ai_mode_record.go:9`，`func NativeAIModeRecordForUnit(unit *Unit) ([]byte, error)`
pub fn native_ai_mode_record_for_unit(unit: Option<&Unit>) -> Result<Vec<u8>> {
    todo!("battle/native_ai_mode_record.go:NativeAIModeRecordForUnit")
}
