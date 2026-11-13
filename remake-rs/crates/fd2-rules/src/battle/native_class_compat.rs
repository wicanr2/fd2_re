//! Go 來源：`remake/internal/battle/native_class_compat.go`（36 行）。

use super::*;

/// NativeClassCompatibilityTableBase is the relocated EXE table used by
/// 0x4e53e.  The six-byte compatibility predicate consumes the first six
/// bytes of each seven-byte row; the seventh byte remains opaque.
pub const NATIVE_CLASS_COMPATIBILITY_TABLE_BASE: i64 = 0x6188a;

pub const NATIVE_CLASS_COMPATIBILITY_ROW_SIZE: i64 = 7;

/// NativeClassCompatibilityRowOffset preserves the proven address arithmetic
/// of 0x4e53e(class): table base + class*7.  It intentionally returns only a
/// table-relative offset; row bytes are not assigned gameplay names.
///
/// Go：`native_class_compat.go:16`，`func NativeClassCompatibilityRowOffset(classID int) (int, error)`
pub fn native_class_compatibility_row_offset(class_id: i64) -> Result<i64> {
    todo!("battle/native_class_compat.go:NativeClassCompatibilityRowOffset")
}

/// NativeClassItemCompatible reproduces 0x1c1c3's six-byte comparison.  The
/// caller must provide the exact row returned by 0x4e53e; short rows fail
/// closed rather than treating missing bytes as compatible.
///
/// Go：`native_class_compat.go:26`，`func NativeClassItemCompatible(itemRowByte0 byte, classRow []byte) (bool, error)`
pub fn native_class_item_compatible(item_row_byte0: u8, class_row: &[u8]) -> Result<bool> {
    todo!("battle/native_class_compat.go:NativeClassItemCompatible")
}
