//! Go 來源：`remake/internal/battle/native_ai_record.go`（44 行）。

use super::*;

/// NativeAIScoringRecords materializes the raw runtime fields consumed across
/// 0x1598a, 0x15b77 and their phase admission. It deliberately requires an
/// already-materialized native map position and every optional record byte
/// used by those paths; normalized X/Y, Acted, Camp or status fields are never
/// substituted. The returned buffer is a detached snapshot.
///
/// Go：`native_ai_record.go:13`，`func NativeAIScoringRecords(units []*Unit) ([]byte, error)`
pub fn native_ai_scoring_records(units: &[Unit]) -> Result<Vec<u8>> {
    todo!("battle/native_ai_record.go:NativeAIScoringRecords")
}
