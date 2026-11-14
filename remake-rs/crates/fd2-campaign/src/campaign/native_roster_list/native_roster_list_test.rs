//! 移植自 `remake/internal/campaign/native_roster_list_test.go`。

use super::*;
use crate::campaign::native_class_list::native_class_list_test::{
    native_class_list_font, native_class_list_strings, native_test_sprite,
};

fn panel33() -> fdother::LMI1Entry {
    fdother::LMI1Entry {
        width: NATIVE_CLASS_PANEL_W,
        height: NATIVE_CLASS_PANEL_H,
        pixels: vec![33; (NATIVE_CLASS_PANEL_W * NATIVE_CLASS_PANEL_H) as usize],
    }
}

/// Go：`TestNativeTwoColumnWindowPreservesStatefulEvenScrollOrigin`
#[test]
fn native_two_column_window_preserves_stateful_even_scroll_origin() {
    let mut start = 0;
    for (selected, want_start, want_visible) in [
        (0, 0, 6),
        (5, 0, 6),
        (6, 2, 6),
        (8, 4, 6),
        (7, 4, 6), // still inside the current window: do not jump back
        (3, 2, 6), // crossed above start: scroll up exactly one row
        (1, 0, 6),
    ] {
        let visible;
        (start, visible) = native_two_column_window(10, selected, start);
        assert!(
            start == want_start && visible == want_visible,
            "selection {selected} window=({start},{visible}), want ({want_start},{want_visible})"
        );
    }
    assert_eq!(native_two_column_window(1, 0, 0), (0, 1), "single-entry window");
}

/// Go：`TestRenderNativeRosterRowsMatches2EA90Geometry`
#[test]
fn render_native_roster_rows_matches_2ea90_geometry() {
    let rows = vec![NativeRosterRow { sprite: native_test_sprite(99), name_text_index: 1 }; 6];
    let mut dst = vec![0u8; (NATIVE_CLASS_LIST_STRIDE * NATIVE_CLASS_LIST_HEIGHT) as usize];
    let (strings, font) = (native_class_list_strings(), native_class_list_font());
    render_native_roster_rows(&mut dst, &rows, 3, Some(&strings), Some(&font)).unwrap();
    for (i, (x, y)) in [(14, 117), (146, 117), (14, 143), (146, 143), (14, 169), (146, 169)].into_iter().enumerate() {
        assert_eq!(dst[y * 320 + x], 99, "row{i} sprite");
    }
    assert_eq!(dst[121 * 320 + 40], 205, "row0 foreground");
    assert_eq!(dst[147 * 320 + 172], 201, "selected row3 foreground");
    assert_eq!(dst[148 * 320 + 171], 76, "selected row3 shadow");
    assert_eq!(dst[147 * 320 + 171], 0, "selected row3 same-row shadow");
}

/// Go：`TestComposeNativeRosterFrameUsesEntry16Panel`
#[test]
fn compose_native_roster_frame_uses_entry16_panel() {
    let (strings, font) = (native_class_list_strings(), native_class_list_font());
    let rows = [NativeRosterRow { sprite: native_test_sprite(0), name_text_index: 1 }];
    let frame =
        compose_native_roster_frame(&vec![0u8; 320 * 200], &panel33(), &rows, 0, Some(&strings), Some(&font)).unwrap();
    assert_eq!(frame[(NATIVE_CLASS_PANEL_Y * 320 + NATIVE_CLASS_PANEL_X) as usize], 33, "panel origin");
}

/// Go：`TestComposeNativeRosterFrameWithoutNamesKeepsSpriteAndNameRectangleBlank`
#[test]
fn compose_native_roster_frame_without_names_keeps_sprite_and_name_rectangle_blank() {
    let rows = [NativeRosterRow { sprite: native_test_sprite(99), name_text_index: 1 }];
    let frame = compose_native_roster_frame_without_names(&vec![0u8; 320 * 200], &panel33(), &rows, 0).unwrap();
    assert_eq!(frame[117 * 320 + 14], 99, "sprite origin");
    assert_eq!(frame[121 * 320 + 40], 33, "blank name rectangle want panel 33");
}
