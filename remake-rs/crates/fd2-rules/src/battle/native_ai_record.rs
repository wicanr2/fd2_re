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
    let mut records = Vec::with_capacity(units.len() * NATIVE_RECORD_SIZE as usize);
    for (index, unit) in units.iter().enumerate() {
        if !unit.has_native_map_presentation
            || !unit.has_battle_fig
            || !unit.has_native_record_byte5
            || !unit.has_native_record_byte34
            || !unit.has_native_record_byte35
            || !unit.has_native_record_byte36
            || !unit.has_native_record_word42
            || !unit.has_native_record_word46
        {
            return Err(battle_err!("native AI record: unit {index} lacks raw provenance"));
        }
        let mut record = native_item_panel_record_for_unit(Some(unit))
            .map_err(|e| e.wrap(format!("native AI record: unit {index}: ")))?;
        let presentation = unit.native_map_presentation;
        record[0] = presentation.x;
        record[1] = presentation.y;
        record[3] = presentation.pose;
        record[4] = presentation.motion;
        record[5] = unit.native_record_byte5;
        record[0x34] = unit.native_record_byte34;
        record[0x35] = unit.native_record_byte35;
        record[0x36] = unit.native_record_byte36;
        record[0x42..0x44].copy_from_slice(&unit.native_record_word42.to_le_bytes());
        record[0x46..0x48].copy_from_slice(&unit.native_record_word46.to_le_bytes());
        records.extend_from_slice(&record);
    }
    Ok(records)
}

#[cfg(test)]
mod native_ai_record_test;
