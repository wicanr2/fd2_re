//! 移植自 `remake/internal/battle/native_record_byte5_test.go`。

use super::*;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestSetNativeRecordByte5OneOverwritesWholeByte`
#[test]
fn set_native_record_byte5_one_overwrites_whole_byte() {
    let mut records = vec![0u8; 2 * RS];
    records[5] = 0x80;
    records[RS + 5] = 0xff;
    set_native_record_byte5_one(&mut records, 1).unwrap();
    assert!(records[5] == 0x80 && records[RS + 5] == 1, "bytes={:#x}/{:#x}", records[5], records[RS + 5]);
}

/// Go：`TestSetNativeRecordByte5OneRejectsBounds`
#[test]
fn set_native_record_byte5_one_rejects_bounds() {
    assert!(set_native_record_byte5_one(&mut [0u8; RS], 1).is_err(), "out-of-range record unexpectedly accepted");
    assert!(set_native_record_byte5_one(&mut [0u8; RS], -1).is_err(), "negative record unexpectedly accepted");
}

/// Go：`TestHPWritersMirrorRawByte5OnlyWhenProvenanceExists`
#[test]
fn hp_writers_mirror_raw_byte5_only_when_provenance_exists() {
    let mut dead =
        Unit { hp: 5, max_hp: 5, has_native_record_byte5: true, native_record_byte5: 0x80, ..Default::default() };
    dead.apply_hp_damage(9);
    assert!(
        dead.hp == 0 && dead.native_record_byte5 == 1,
        "native death writer hp={} byte5={:#x}",
        dead.hp,
        dead.native_record_byte5
    );
    dead.restore_native_hp();
    assert!(
        dead.hp == 5 && dead.native_record_byte5 == 0,
        "native revive writer hp={} byte5={:#x}",
        dead.hp,
        dead.native_record_byte5
    );
    let mut legacy = Unit { hp: 5, max_hp: 5, native_record_byte5: 0x80, ..Default::default() };
    legacy.apply_hp_damage(9);
    assert!(
        legacy.hp == 0 && legacy.native_record_byte5 == 0x80,
        "legacy path fabricated raw provenance hp={} byte5={:#x}",
        legacy.hp,
        legacy.native_record_byte5
    );
}
