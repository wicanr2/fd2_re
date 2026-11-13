//! 移植自 `remake/internal/battle/native_item_mp_restore_test.go`。

use super::*;
use crate::test_support::{get16, put16};

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestNativeType11MPRestoreSkipsZeroMaxWithoutRNGAndConsumesSource`
#[test]
fn native_type11_mp_restore_skips_zero_max_without_rng_and_consumes_source() {
    let mut records = vec![0u8; 3 * RS];
    put16(&mut records, 0x44, 0);
    put16(&mut records, 0x46, 0);
    put16(&mut records, RS + 0x44, 10);
    put16(&mut records, RS + 0x46, 100);
    for slot in 0..8 {
        (records[2 * RS + 0x0a + slot * 2], records[2 * RS + 0x0b + slot * 2]) = (0, 0xce + slot as u8);
    }
    let (route, ok) = native_item_mp_restore_route_for_type(11, 80);
    assert!(ok, "type 11 route missing");
    let result = apply_native_item_mp_restore(&mut records, &[0, 1], route, 0, 2, 0).unwrap();
    assert!(
        result.skipped_targets.len() == 1 && result.skipped_targets[0] == 0 && result.results.len() == 1,
        "result={result:?}"
    );
    assert!(
        result.rng_state == result.results[0].rng_state && result.rng_state != 0,
        "zero-max target consumed RNG: {result:?}"
    );
    assert!(get16(&records, RS + 0x44) > 10, "MP was not restored: {}", get16(&records, RS + 0x44));
    let source = &records[2 * RS..];
    assert!(source[0x0b] == 0xcf && source[0x18] == 0x80, "source slot not consumed: {:02x?}", &source[0x0a..0x1a]);
}

/// Go：`TestNativeType11MPRestorePreflightsSourceAtomically`
#[test]
fn native_type11_mp_restore_preflights_source_atomically() {
    let mut records = vec![0u8; RS];
    put16(&mut records, 0x44, 10);
    put16(&mut records, 0x46, 100);
    let before = records.clone();
    let (route, _) = native_item_mp_restore_route_for_type(11, 80);
    assert!(
        apply_native_item_mp_restore(&mut records, &[0], route, 0, 0, 8).is_err(),
        "invalid source slot unexpectedly accepted"
    );
    assert_eq!(records, before, "invalid source removal partially restored MP");
}
