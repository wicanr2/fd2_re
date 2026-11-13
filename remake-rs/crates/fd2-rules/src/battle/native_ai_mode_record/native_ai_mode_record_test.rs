//! 移植自 `remake/internal/battle/native_ai_mode_record_test.go`。

use super::*;

/// Go：`TestNativeAIModeRecordForUnitPreservesConstructorBytes`
#[test]
fn native_ai_mode_record_for_unit_preserves_constructor_bytes() {
    let unit = Unit {
        native_record_byte5: 0x80,
        has_native_record_byte5: true,
        native_record_byte34: 0xc2,
        has_native_record_byte34: true,
        native_record_byte35: 0x12,
        has_native_record_byte35: true,
        native_record_byte36: 0x07,
        has_native_record_byte36: true,
        native_record_race: 4,
        has_native_record_race: true,
        ..Default::default()
    };
    let mut record = native_ai_mode_record_for_unit(Some(&unit)).unwrap();
    assert!(
        record.len() == 0x3e
            && record[5] == 0x80
            && record[0x34] == 0xc2
            && record[0x35] == 0x12
            && record[0x36] == 7
            && record[0x3d] == 4,
        "native AI mode record={record:?}"
    );
    record[5] = 0;
    let plan = fdother::plan_native_unit_mode(&record, 3, 1);
    assert!(
        matches!(&plan, Ok(Some(p)) if p.mode == 2 && p.raw_arg35 == 0x12 && p.raw_arg36 == 7 && p.raw_counter == 4),
        "native AI mode plan={plan:?}"
    );
}

/// Go：`TestNativeAIModeRecordForUnitFailsClosed`
#[test]
fn native_ai_mode_record_for_unit_fails_closed() {
    assert!(native_ai_mode_record_for_unit(Some(&Unit::default())).is_err(), "missing mode provenance accepted");
}
