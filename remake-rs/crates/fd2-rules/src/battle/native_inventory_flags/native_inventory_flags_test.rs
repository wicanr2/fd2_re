//! 移植自 `remake/internal/battle/native_inventory_flags_test.go`。

use super::*;

/// Go：`TestNativeInventoryFlagsFromConstructorSource`
#[test]
fn native_inventory_flags_from_constructor_source() {
    let flags = native_inventory_flags_from_source(&[0x44, 0x55, 0xff, 0x66, 0xff, 0xff, 0x77, 0xff]).unwrap();
    assert_eq!(flags, [0x40, 0x40, 0x80, 0x00, 0x80, 0x80, 0x00, 0x80]);
    let collapsed = native_inventory_flags_from_source(&[0xff, 0x55, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff]).unwrap();
    assert!(collapsed[0] == 0x40 && collapsed[1] == 0x80, "collapsed first pair={collapsed:?}");
}

/// Go：`TestNativeInventoryCompactEligibleUsesRawFlags`
#[test]
fn native_inventory_compact_eligible_uses_raw_flags() {
    let flags = [0x40, 0x40, 0x80, 0, 0x80, 0x80, 0, 0x80];
    let slots = [0x44, 0x55, 0xff, 0x66, 0xff, 0xff, 0x77, 0xff];
    assert_eq!(native_inventory_compact_eligible(&flags, &slots, 0), Ok(true), "signed-nonnegative equipped item gate");
    assert_eq!(native_inventory_compact_eligible(&flags, &slots, 1), Ok(true), "unequipped item gate");
    assert_eq!(native_inventory_compact_eligible(&flags, &slots, 2), Ok(true), "second unequipped item gate");
    assert!(
        native_inventory_compact_eligible(&flags[..7], &slots, 0).is_err(),
        "short raw flags unexpectedly accepted"
    );
}
