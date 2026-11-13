//! 移植自 `remake/internal/battle/native_inventory_available_test.go`。

use super::*;

/// Go：`TestNativeInventoryAvailableCountUsesSignedRawFlags`
#[test]
fn native_inventory_available_count_uses_signed_raw_flags() {
    assert_eq!(native_inventory_available_count(&[0x40, 0x40, 0, 0x80, 0x80, 0, 0x80, 0x80]), Ok(4));
    assert!(native_inventory_available_count(&[0, 0, 0]).is_err(), "short raw flag input unexpectedly accepted");
}
