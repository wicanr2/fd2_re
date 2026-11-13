//! 移植自 `remake/internal/battle/native_command_overlay_render_test.go`。

use super::*;
use crate::battle::native_item_panel_data_render::native_item_panel_data_render_test::native_item_panel_test_assets;

/// Go：`TestRenderNativeCommandOverlayMatches1CEEDCoordinates`
#[test]
fn render_native_command_overlay_matches1_ceed_coordinates() {
    let assets = native_item_panel_test_assets(0);
    let book: Vec<NativeCommandRecord> =
        (0..5).map(|id| NativeCommandRecord { id, mp_cost: id + 1, ..Default::default() }).collect();
    let mut dst = vec![0u8; NATIVE_ITEM_PANEL_BYTES as usize];
    render_native_command_overlay(&assets, &[0, 1, 2, 3, 4], &book, 4, &mut dst).unwrap();
    // Synthetic glyph 441 starts at index 441 and is drawn at (18,103).
    assert_eq!(dst[103 * 320 + 18], 205, "command0 label foreground");
    // Fifth command begins the second column at (118,103).
    assert_eq!(dst[103 * 320 + 118], 201, "selected command4 foreground");
    assert_ne!(dst[108 * 320 + 68], 0, "command0 MP icon did not render at (68,108)");
}

/// Go：`TestNativeItemPanelRecordIncludesCommandMaskAt1A`
#[test]
fn native_item_panel_record_includes_command_mask_at1_a() {
    let unit = Unit {
        battle_fig: 0,
        native_identity: 0,
        has_native_identity: true,
        native_record_byte6: 1,
        has_native_record_byte6: true,
        native_record_race: 1,
        has_native_record_race: true,
        native_record_class: 1,
        has_native_record_class: true,
        lv: 1,
        mv: 1,
        inventory_slots: vec![0xff; 8],
        native_inventory_flags: vec![0x80; 8],
        native_command_mask: [1, 2, 3, 4, 5],
        ..Default::default()
    };
    let record = native_item_panel_record_for_unit(Some(&unit)).unwrap();
    for (i, &want) in unit.native_command_mask.iter().enumerate() {
        assert_eq!(record[0x1a + i], want, "record command byte{i}");
    }
}
