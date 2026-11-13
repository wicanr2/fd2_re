//! 移植自 `remake/internal/battle/native_item_marker_clear_restore_test.go`。

use super::*;
use crate::test_support::{get16, put16};

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestNativeType6And7ClearRecordMarkerRestoreHPConsumeSource`
#[test]
fn native_type6_and7_clear_record_marker_restore_hp_consume_source() {
    for (typ, code, marker) in [(6u8, 20i64, 0x25i64), (7, 21, 0x26)] {
        let mut records = vec![0u8; 2 * RS];
        put16(&mut records, 0x40, 10);
        put16(&mut records, 0x42, 100);
        records[marker as usize] = 3;
        for slot in 0..8 {
            (records[RS + 0x0a + slot * 2], records[RS + 0x0b + slot * 2]) = (0, 0xc4 + slot as u8);
        }
        let (route, ok) = native_item_marker_clear_restore_route_for_type(typ);
        assert!(
            ok && route.presentation_code == code
                && route.marker_offset == marker
                && route.hp_restore_base == 10
                && route.consumes_source,
            "type {typ:#x} route = {route:?}, {ok}"
        );
        let (result, next_rng, _) = apply_native_item_marker_clear_restore(&mut records, &[0], route, 0, 1, 0).unwrap();
        assert!(
            result.len() == 1
                && result[0].applied
                && next_rng != 0
                && records[marker as usize] == 0
                && get16(&records, 0x40) == 19,
            "type {typ:#x} result={result:?} rng={next_rng:#x} marker={} hp={}",
            records[marker as usize],
            get16(&records, 0x40)
        );
        let source = &records[RS..];
        assert!(
            source[0x0b] == 0xc5 && source[0x18] == 0x80,
            "type {typ:#x} source not consumed: {:02x?}",
            &source[0x0a..0x1a]
        );
    }
}

/// Go：`TestNativeMarkerClearRestorePreflightsSourceBeforeHP`
#[test]
fn native_marker_clear_restore_preflights_source_before_hp() {
    let mut records = vec![0u8; RS];
    put16(&mut records, 0x40, 10);
    put16(&mut records, 0x42, 100);
    records[0x25] = 2;
    let before = records.clone();
    let (route, _) = native_item_marker_clear_restore_route_for_type(6);
    assert!(
        apply_native_item_marker_clear_restore(&mut records, &[0], route, 0, 0, 8).is_err(),
        "invalid source slot unexpectedly accepted"
    );
    assert_eq!(records, before, "invalid source removal partially mutated target");
}
