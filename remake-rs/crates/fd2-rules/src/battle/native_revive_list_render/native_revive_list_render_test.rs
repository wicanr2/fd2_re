//! 移植自 `remake/internal/battle/native_revive_list_render_test.go`。

use super::*;
use crate::battle::native_item_panel_data_render::native_item_panel_data_render_test::native_item_panel_test_assets;

/// Go：`TestRenderNativeReviveRowsMatches30A47Geometry`
#[test]
fn render_native_revive_rows_matches30_a47_geometry() {
    let assets = native_item_panel_test_assets(0);
    let n = (fdicon::NATIVE_SIZE * fdicon::NATIVE_SIZE) as usize;
    let mut sprite = fdicon::Sprite { pixels: vec![0; n], mask: vec![0; n], ..Default::default() };
    (sprite.pixels[0], sprite.mask[0]) = (99, 1);
    let currency = fdother::RawCell { width: 1, height: 1, pixels: vec![88] };
    let row = |fee| NativeReviveRow {
        sprite: sprite.clone(),
        name_text_index: 1,
        race_text_index: 140,
        class_text_index: 150,
        fee,
    };
    let rows = [row(28), row(321)];
    let mut dst = vec![0u8; NATIVE_ITEM_PANEL_BYTES as usize];
    render_native_revive_rows(&assets, &currency, &rows, 1, &mut dst).unwrap();
    assert_eq!(dst[117 * 320 + 14], 99, "row0 sprite");
    assert_eq!(dst[143 * 320 + 14], 99, "row1 sprite");
    assert_eq!(dst[121 * 320 + 40], 205, "row0 name");
    assert_eq!(dst[147 * 320 + 40], 201, "selected row1 name");
    assert_eq!(dst[125 * 320 + 220], 88, "currency");
    assert_eq!(dst[125 * 320 + 228], 119, "fee leading digit");
}
