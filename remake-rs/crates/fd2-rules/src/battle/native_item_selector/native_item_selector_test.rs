//! 移植自 `remake/internal/battle/native_item_selector_test.go`。

use super::*;
use crate::test_support::remake_asset;

const RS: usize = NATIVE_RECORD_SIZE as usize;

fn tracked_table() -> Option<Vec<u8>> {
    let path = remake_asset("data/native_item_effect_rows.json")?;
    Some(load_native_item_effect_row_prefix(&path).unwrap())
}

/// Go：`TestAdvanceNativeItemSelectorMatchesTwoColumnInput`
#[test]
fn advance_native_item_selector_matches_two_column_input() {
    for (name, selection, count, key, want, result) in [
        ("up wrap", 0, 7, 72, 6, NATIVE_ITEM_SELECTOR_WAIT),
        ("down crosses column", 3, 7, 80, 4, NATIVE_ITEM_SELECTOR_WAIT),
        ("down wrap", 6, 7, 80, 0, NATIVE_ITEM_SELECTOR_WAIT),
        ("left", 6, 7, 75, 2, NATIVE_ITEM_SELECTOR_WAIT),
        ("left blocked", 2, 7, 75, 2, NATIVE_ITEM_SELECTOR_WAIT),
        ("right", 2, 7, 77, 6, NATIVE_ITEM_SELECTOR_WAIT),
        ("right missing", 3, 7, 77, 3, NATIVE_ITEM_SELECTOR_WAIT),
        ("cancel", 2, 7, 1, 2, NATIVE_ITEM_SELECTOR_CANCEL),
    ] {
        let got = advance_native_item_selector(selection, count, key, true, 5);
        assert!(matches!(got, Ok((g, r)) if g == want && r == result), "{name}: got={got:?}");
    }
}

/// Go：`TestAdvanceNativeItemSelectorBattleUseRejectsTypeZero`
#[test]
fn advance_native_item_selector_battle_use_rejects_type_zero() {
    let got = advance_native_item_selector(0, 1, 28, true, 0);
    assert!(matches!(got, Ok((_, r)) if r == NATIVE_ITEM_SELECTOR_WAIT), "battle type0 result={got:?}");
    let got = advance_native_item_selector(0, 1, 57, false, 0);
    assert!(matches!(got, Ok((_, r)) if r == NATIVE_ITEM_SELECTOR_CONFIRM), "nonbattle type0 result={got:?}");
}

/// Go：`TestNativeItemSelectorCellsCompactAndUseOriginalGeometry`
#[test]
fn native_item_selector_cells_compact_and_use_original_geometry() {
    let Some(table) = tracked_table() else { return };
    let mut records = vec![0u8; RS];
    for slot in 0..8 {
        records[0x0a + slot * 2] = 0x80;
    }
    (records[0x0a], records[0x0b]) = (0x40, 0);
    (records[0x0e], records[0x0f]) = (0, 79);
    let cells = native_item_selector_cells(&records, 0, 2, &table).unwrap();
    assert_eq!(cells.len(), 2, "cells");
    let c = cells[0];
    assert!(
        c.raw_slot == 0
            && c.display_index == 0
            && c.label_x == 42
            && c.label_y == 103
            && c.equipped
            && c.category_icon == 62,
        "cell0={c:?}"
    );
    let c = cells[1];
    assert!(
        c.raw_slot == 2 && c.display_index == 1 && c.label_x == 42 && c.label_y == 125 && c.selected,
        "cell1={c:?}"
    );
}

/// Go：`TestNativeItemSelectorFifthCellStartsRightColumn`
#[test]
fn native_item_selector_fifth_cell_starts_right_column() {
    let Some(table) = tracked_table() else { return };
    let mut records = vec![0u8; RS];
    for slot in 0..8 {
        (records[0x0a + slot * 2], records[0x0b + slot * 2]) = (0, slot as u8);
    }
    let cells = native_item_selector_cells(&records, 0, -1, &table).unwrap();
    assert!(cells[4].label_x == 192 && cells[4].label_y == 103, "fifth cell={:?}", cells[4]);
}

/// Go：`TestNativeItemPanelFrameClipping`
#[test]
fn native_item_panel_frame_clipping() {
    let closed = native_item_panel_frame_for(11).unwrap();
    assert!(
        closed.left.source_x == 80
            && closed.left.dest_x == 0
            && closed.left.width == 11
            && !closed.upper.enabled
            && !closed.bottom.enabled,
        "frame11={closed:?}"
    );
    let middle = native_item_panel_frame_for(5).unwrap();
    assert!(
        middle.left.dest_x == 5
            && middle.left.width == 86
            && middle.upper.enabled
            && middle.upper.source_y == 32
            && middle.upper.height == 61
            && middle.bottom.enabled
            && middle.bottom.dest_y == 174
            && middle.bottom.height == 26,
        "frame5={middle:?}"
    );
    let open = native_item_panel_frame_for(0).unwrap();
    assert!(
        open.left.dest_x == 5 && open.upper.dest_y == 7 && open.bottom.dest_y == 94 && open.bottom.height == 102,
        "frame0={open:?}"
    );
}

/// Go：`TestNativeItemPanelSchedulesReverseExactly`
#[test]
fn native_item_panel_schedules_reverse_exactly() {
    let opening = native_item_panel_schedule(true).unwrap();
    let closing = native_item_panel_schedule(false).unwrap();
    assert!(
        opening.len() == 12
            && closing.len() == 12
            && opening[0].frame == 11
            && opening[11].frame == 0
            && closing[0].frame == 0
            && closing[11].frame == 11,
        "opening={opening:?} closing={closing:?}"
    );
    for i in 0..opening.len() {
        assert_eq!(opening[i], closing[11 - i], "step {i} does not reverse");
    }
}

/// Go：`TestNativeItemPanelBaseLayoutMatches17EEF`
#[test]
fn native_item_panel_base_layout_matches17_eef() {
    let got = native_item_panel_base_layout_for();
    let p = |x, y| NativeItemPanelPoint { x, y };
    assert!(
        got.stride == 320
            && got.fdother_resource == 5
            && got.grid_origin == p(5, 7)
            && got.grid_columns == 5
            && got.grid_rows == 5
            && got.portrait_record_offset == 7
            && got.portrait_destination == p(8, 10)
            && got.upper_directory_entry == 20
            && got.upper_destination == p(92, 7)
            && got.bottom_directory_entry == 21
            && got.bottom_destination == p(5, 94),
        "base layout={got:?}"
    );
}

/// Go：`TestNativeItemPanelDataPlanMatches17FC0`
#[test]
fn native_item_panel_data_plan_matches17_fc0() {
    let got = native_item_panel_data_plan_for();
    let p = |x, y| NativeItemPanelPoint { x, y };
    assert!(
        got.bars.len() == 2
            && got.compared_numbers.len() == 4
            && got.raw_numbers.len() == 8
            && got.text.len() == 3
            && got.flag_icons.len() == 3,
        "plan cardinality={got:?}"
    );
    assert!(
        got.bars[0]
            == NativeItemPanelBarCall {
                destination: p(198, 33),
                fdother_base_entry: 23,
                current_offset: 64,
                maximum_offset: 66
            }
            && got.bars[1].destination == p(198, 52),
        "bars={:?}",
        got.bars
    );
    assert!(
        got.compared_numbers[0].destination == p(267, 41) && got.compared_numbers[3].destination == p(293, 59),
        "compared numbers={:?}",
        got.compared_numbers
    );
    let r = &got.raw_numbers;
    assert!(
        r[0].value_bytes == 1
            && r[3].value_bytes == 2
            && r[4].value_offset == 72
            && r[4].alternate_flag_offset == 34
            && r[7].value_offset == 78
            && r[7].alternate_flag_offset == 36,
        "raw numbers={r:?}"
    );
    assert!(
        got.text[0] == NativeItemPanelTextCall { destination: p(99, 13), record_offset: 8, fdtxt_base: 1 }
            && got.text[2].fdtxt_base == 150,
        "text={:?}",
        got.text
    );
    assert!(
        got.base_icon.fdother_entry == 53
            && got.base_icon.alternate_fdother_entry == 54
            && got.base_icon.use_alternate_when_flag_zero
            && got.base_icon.flag_offset == 6
            && got.flag_icons[0].fdother_entry == 55
            && got.flag_icons[0].draw_when_flag_nonzero
            && got.flag_icons[2].flag_offset == 39,
        "icons={:?} / {:?}",
        got.base_icon,
        got.flag_icons
    );
}
