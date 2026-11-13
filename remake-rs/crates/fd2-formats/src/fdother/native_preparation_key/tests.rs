//! 移植自 `native_preparation_key_test.go`。

use crate::fdother::*;

// Go: TestNormalizeNativePreparationKeyPreserves32004Branches
#[test]
fn normalize_native_preparation_key_preserves_32004_branches() {
    for (name, raw53a8d, raw53a8e, want) in [
        ("extended-e0", 0x00, 0xe0, 0x1c),
        ("extended-52", 0x00, 0x52, 0x1c),
        ("space-record", 0x20, 0x10, 0x1c),
        ("53-without-space", 0x00, 0x53, 1),
        ("space-precedes-53", 0x20, 0x53, 0x1c),
        ("seed-default", 0x00, 0x10, 0x10),
    ] {
        assert_eq!(normalize_native_preparation_key(raw53a8d, raw53a8e), want, "{name}");
    }
}
