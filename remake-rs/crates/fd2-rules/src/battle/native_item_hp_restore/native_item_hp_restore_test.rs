//! 移植自 `remake/internal/battle/native_item_hp_restore_test.go`。

use super::*;
use crate::test_support::{get16, put16};

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestNativeType5HPRestoreConsumesSourceAfterMutation`
#[test]
fn native_type5_hp_restore_consumes_source_after_mutation() {
    let mut records = vec![0u8; 2 * RS];
    put16(&mut records, 0x40, 10);
    put16(&mut records, 0x42, 100);
    for slot in 0..8 {
        let off = RS + 0x0a + slot * 2;
        (records[off], records[off + 1]) = (0, 0xc0 + slot as u8);
    }
    let (route, ok) = native_item_hp_restore_route_for_type(5, 40);
    assert!(ok, "type 5 route missing");
    let result = apply_native_item_hp_restore(&mut records, &[0], route, 0, 1, 1).unwrap();
    assert!(
        result.results.len() == 1 && get16(&records, 0x40) > 10,
        "restore result={result:?} hp={}",
        get16(&records, 0x40)
    );
    let source = &records[RS..];
    assert!(
        source[0x0c + 1] == 0xc2 && source[0x18] == 0x80,
        "type 5 source slot not consumed: {:02x?}",
        &source[0x0a..0x1a]
    );
}

/// Go：`TestNativeType13HPRestoreRetainsSource`
#[test]
fn native_type13_hp_restore_retains_source() {
    let mut records = vec![0u8; RS];
    put16(&mut records, 0x40, 10);
    put16(&mut records, 0x42, 100);
    (records[0x0a], records[0x0b]) = (0x40, 0xd3);
    let before_inventory = records[0x0a..0x1a].to_vec();
    let (route, ok) = native_item_hp_restore_route_for_type(13, 200);
    assert!(ok, "type 13 route missing");
    apply_native_item_hp_restore(&mut records, &[0], route, 0, -1, -1).unwrap();
    assert_eq!(records[0x0a..0x1a], before_inventory[..], "type 13 unexpectedly consumed source");
}

/// Go：`TestNativeItemHPRestorePreflightsRemovalAtomically`
#[test]
fn native_item_hp_restore_preflights_removal_atomically() {
    let mut records = vec![0u8; RS];
    put16(&mut records, 0x40, 10);
    put16(&mut records, 0x42, 100);
    let before = records.clone();
    let (route, _) = native_item_hp_restore_route_for_type(5, 40);
    assert!(
        apply_native_item_hp_restore(&mut records, &[0], route, 0, 0, 8).is_err(),
        "invalid source slot unexpectedly accepted"
    );
    assert_eq!(records, before, "invalid source removal partially restored HP");
}
