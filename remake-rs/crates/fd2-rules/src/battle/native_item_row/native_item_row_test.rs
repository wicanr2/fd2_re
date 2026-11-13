//! 移植自 `remake/internal/battle/native_item_row_test.go`。

use super::*;

/// Go：`TestNativeItemEffectRowOffset`
#[test]
fn native_item_effect_row_offset_test() {
    for (item, want) in [(0, 0), (1, 0x17), (0xff, 0x17 * 0xff)] {
        assert_eq!(native_item_effect_row_offset(item), Ok(want), "item {item:#x}");
    }
    for item in [-1, 0x100] {
        assert!(native_item_effect_row_offset(item).is_err(), "item {item:#x} unexpectedly accepted");
    }
}

/// Go：`TestNativeItemWordDeltaRouteForTypePreservesRawDispatch`
#[test]
fn native_item_word_delta_route_for_type_preserves_raw_dispatch() {
    for (item_type, field, presentation, base_stat) in [
        (8, 0x37, 0x11, NATIVE_ITEM_BASE_AP),
        (9, 0x39, 0x12, NATIVE_ITEM_BASE_DP),
        (0xa, 0x3e, 0x13, NATIVE_ITEM_BASE_DX),
    ] {
        let (route, ok) = native_item_word_delta_route_for_type(item_type);
        assert!(
            ok && route.item_type == item_type
                && route.base_stat == base_stat
                && route.field_offset == field
                && route.presentation_code == presentation,
            "type {item_type:#x} route={route:?} ok={ok}"
        );
    }
    assert!(!native_item_word_delta_route_for_type(7).1, "unsupported item type became a word-delta route");
}
