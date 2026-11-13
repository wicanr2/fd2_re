//! 移植自 `remake/internal/battle/native_item_dispatch_test.go`。

use super::*;

/// Go：`TestNativeItemEffectRouteForTypePreservesRawBranches`
#[test]
fn native_item_effect_route_for_type_preserves_raw_branches() {
    for (item_type, primary, secondary, arg2, arg5, arg7) in [
        (5u8, 0x211a4u32, 0u32, 0i64, 0i64, 0i64),
        (6, 0x22af6, 0, 20, 37, 0),
        (8, 0x21082, 0, 55, 0, 17),
        (11, 0x1c4cc, 0x1c2da, 13, 0, 0),
        (20, 0x1c4cc, 0x1cd17, 0, 0, 0),
        (22, 0x22d1b, 0, 22, 39, 0),
        (23, 0x2218a, 0, 0, 0, 0),
    ] {
        let (route, ok) = native_item_effect_route_for_type(item_type, 0x1234);
        assert!(
            ok && route.primary == primary
                && route.secondary == secondary
                && route.arg2 == arg2
                && route.arg5 == arg5
                && route.arg7 == arg7
                && route.word == 0x1234,
            "type {item_type} route = {route:?}, want primary={primary:#x} secondary={secondary:#x} args=({arg2},{arg5},{arg7})"
        );
    }
}

/// Go：`TestNativeItemEffectRouteRejectsUnknownType`
#[test]
fn native_item_effect_route_rejects_unknown_type() {
    let (route, ok) = native_item_effect_route_for_type(0, 0);
    assert!(!ok && route.primary == 0, "unknown item type must fail closed: {route:?}, {ok}");
}
