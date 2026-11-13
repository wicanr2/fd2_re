//! 移植自 `remake/internal/battle/native_transfer_item_render_test.go`。

use super::*;
use crate::battle::native_item_panel_data_render::native_item_panel_data_render_test::native_item_panel_test_assets;
use crate::test_support::put16;

const PANEL: usize = NATIVE_ITEM_PANEL_BYTES as usize;
const ROW: usize = NATIVE_ITEM_EFFECT_ROW_SIZE as usize;

fn facility() -> fdother::RawCell {
    fdother::RawCell { width: 1, height: 1, pixels: vec![88] }
}

fn one_row_1000() -> Vec<u8> {
    let mut rows = vec![0u8; ROW];
    rows[0] = 1;
    put16(&mut rows, 19, 1000);
    rows
}

/// Go：`TestRenderNativeTransferItemRowsMatches2DC55Mode1`
#[test]
fn render_native_transfer_item_rows_matches2_dc55_mode1() {
    let assets = native_item_panel_test_assets(0);
    let mut rows = vec![0u8; 2 * ROW];
    rows[0] = 1;
    put16(&mut rows, 1, 123);
    put16(&mut rows, 19, 1000);
    rows[ROW] = 0x15;
    put16(&mut rows, ROW + 5, 45);
    put16(&mut rows, ROW + 19, 200);
    let mut dst = vec![0u8; PANEL];
    render_native_transfer_item_rows(&assets, &facility(), &[0, 1], 0, 1, &rows, &mut dst).unwrap();
    assert_eq!(dst[119 * 320 + 10], 59, "item0 category");
    assert_eq!(dst[119 * 320 + 158], 60, "item1 category");
    assert_eq!(dst[122 * 320 + 38], 205, "item0 label");
    assert_eq!(dst[122 * 320 + 186], 201, "selected item1 label");
    assert_eq!(dst[131 * 320 + 105], 88, "facility cell");
    // Mode1 displays 3/4 of row+19: 1000 -> 750 with digit base119.
    assert_eq!(dst[131 * 320 + 114], 119, "price leading digit");
}

/// Go：`TestRenderNativeFacilityItemRowsModeZeroUsesFullPrice`
#[test]
fn render_native_facility_item_rows_mode_zero_uses_full_price() {
    let assets = native_item_panel_test_assets(0);
    let rows = one_row_1000();
    let mut full = vec![0u8; PANEL];
    let mut sale = vec![0u8; PANEL];
    render_native_facility_item_rows(&assets, &facility(), &[0], 0, 0, &rows, NATIVE_FACILITY_FULL_PRICE, &mut full)
        .unwrap();
    render_native_facility_item_rows(
        &assets,
        &facility(),
        &[0],
        0,
        0,
        &rows,
        NATIVE_FACILITY_THREE_QUARTER_PRICE,
        &mut sale,
    )
    .unwrap();
    // Five digits start at x=114 with six-pixel advances. 1000 renders
    // 01000 while 750 renders 00750; the middle digit therefore differs.
    assert_eq!(full[131 * 320 + 126], 119, "full-price hundreds digit");
    assert_eq!(sale[131 * 320 + 126], 126, "three-quarter hundreds digit");
}

/// Go：`TestRenderNativeFacilityItemRowsWithoutNamesLeavesSafeRectangleBlank`
#[test]
fn render_native_facility_item_rows_without_names_leaves_safe_rectangle_blank() {
    let assets = native_item_panel_test_assets(0);
    let rows = one_row_1000();
    let mut dst = vec![0u8; PANEL];
    render_native_facility_item_rows_without_names(
        &assets,
        &facility(),
        &[0],
        0,
        0,
        &rows,
        NATIVE_FACILITY_FULL_PRICE,
        &mut dst,
    )
    .unwrap();
    for y in 122..138 {
        for x in 38..105 {
            assert_eq!(dst[y * 320 + x], 0, "name rectangle changed at {x},{y}");
        }
    }
    assert!(
        dst[119 * 320 + 10] == 59 && dst[131 * 320 + 114] == 119,
        "suppressing names also removed category or price"
    );
}

/// Go：`TestNativeFacilityItemListPriceMatchesRendererModes`
#[test]
fn native_facility_item_list_price_matches_renderer_modes() {
    let mut rows = vec![0u8; ROW];
    put16(&mut rows, 19, 101);
    let full = native_facility_item_list_price(&rows, 0, NATIVE_FACILITY_FULL_PRICE);
    assert!(matches!(full, Ok(101)), "full price={full:?}");
    let sale = native_facility_item_list_price(&rows, 0, NATIVE_FACILITY_THREE_QUARTER_PRICE);
    assert!(matches!(sale, Ok(75)), "sale price={sale:?}");
}
