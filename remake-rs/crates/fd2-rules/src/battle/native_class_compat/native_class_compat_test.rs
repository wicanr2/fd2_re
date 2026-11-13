//! 移植自 `remake/internal/battle/native_class_compat_test.go`。

use super::*;

/// Go：`TestNativeClassCompatibilityRowOffset`
#[test]
fn native_class_compatibility_row_offset_test() {
    for (class_id, want) in [(0, 0), (1, 7), (0xff, 0xff * 7)] {
        assert_eq!(native_class_compatibility_row_offset(class_id), Ok(want), "class {class_id:#x}");
    }
    for class_id in [-1, 0x100] {
        assert!(native_class_compatibility_row_offset(class_id).is_err(), "class {class_id:#x} unexpectedly accepted");
    }
}

/// Go：`TestNativeClassItemCompatibleUsesExactlySixBytes`
#[test]
fn native_class_item_compatible_uses_exactly_six_bytes() {
    let row = [2, 4, 8, 16, 32, 64, 0xff];
    for item in [2, 64] {
        assert_eq!(native_class_item_compatible(item, &row), Ok(true), "item {item:#x}");
    }
    assert_eq!(native_class_item_compatible(0xff, &row), Ok(false), "seventh byte must not participate");
    assert!(native_class_item_compatible(2, &row[..5]).is_err(), "short row unexpectedly accepted");
}
