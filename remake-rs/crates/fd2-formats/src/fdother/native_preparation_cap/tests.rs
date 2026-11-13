//! 移植自 `native_preparation_cap_test.go`。

use crate::fdother::*;

// Go: TestNativePreparationPartyLimitUsesNativeIndexBoundary
#[test]
fn native_preparation_party_limit_uses_native_index_boundary() {
    for (index, want) in [(-1, 15), (0, 15), (0x1a, 15), (0x1b, 19), (29, 19)] {
        assert_eq!(native_preparation_party_limit(index), want, "limit({index})");
    }
}
