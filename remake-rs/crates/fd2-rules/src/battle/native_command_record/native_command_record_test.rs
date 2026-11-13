//! 移植自 `remake/internal/battle/native_command_record_test.go`。

use super::*;
use crate::test_support::{remake_asset, temp_dir};

/// Go：`TestLoadNativeCommandRecordsRuntimeExport`
#[test]
fn load_native_command_records_runtime_export() {
    let Some(path) = remake_asset("spells.json") else { return };
    let records = load_native_command_records(&path).unwrap();
    assert_eq!(records.len(), 36, "records");
    let got = records[0];
    assert!(
        got.id == 0
            && got.damage == 50
            && got.hit == 90
            && got.selection_mode == 5
            && got.effect_mode == 0
            && got.mp_cost == 2
            && got.target_code == 0,
        "command0={got:?}"
    );
}

/// Go：`TestLoadNativeCommandRecordsRejectsEditedPresentationMismatch`
#[test]
fn load_native_command_records_rejects_edited_presentation_mismatch() {
    let Some(source) = remake_asset("spells.json") else { return };
    let path = temp_dir("commands").join("commands.json");
    let data = std::fs::read_to_string(source).unwrap().replacen(r#""dmg": 50"#, r#""dmg": 51"#, 1);
    std::fs::write(&path, data).unwrap();
    assert!(load_native_command_records(path.to_str().unwrap()).is_err(), "mismatched editable field must fail closed");
}
