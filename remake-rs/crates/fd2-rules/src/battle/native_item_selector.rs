//! Go 來源：`remake/internal/battle/native_item_selector.go`（342 行）。

use super::*;

/// Go：`native_item_selector.go:5`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct NativeItemSelectorResult(pub i64);

/// Go 原式：`iota`
pub const NATIVE_ITEM_SELECTOR_WAIT: NativeItemSelectorResult = NativeItemSelectorResult(0);

pub const NATIVE_ITEM_SELECTOR_CONFIRM: NativeItemSelectorResult = NativeItemSelectorResult(1);

pub const NATIVE_ITEM_SELECTOR_CANCEL: NativeItemSelectorResult = NativeItemSelectorResult(2);

/// AdvanceNativeItemSelector reproduces 0x1b9de's two-column/four-row input
/// state. Up/down walk the compact occupied prefix linearly with wrap; left and
/// right move by four. Battle-use confirmation rejects effect type zero.
///
/// Go：`native_item_selector.go:16`，`func AdvanceNativeItemSelector( selection, occupiedCount, scanCode int, battleUse bool, selectedEffectType byte, ) (int, NativeItemSelectorResult, error)`
pub fn advance_native_item_selector(
    selection: i64,
    occupied_count: i64,
    scan_code: i64,
    battle_use: bool,
    selected_effect_type: u8,
) -> Result<(i64, NativeItemSelectorResult)> {
    if occupied_count <= 0 || occupied_count > 8 || selection < 0 || selection >= occupied_count {
        return Err(battle_err!("native item selector selection/count={selection}/{occupied_count} is invalid"));
    }
    match scan_code {
        72 => {
            if selection == 0 {
                return Ok((occupied_count - 1, NATIVE_ITEM_SELECTOR_WAIT));
            }
            return Ok((selection - 1, NATIVE_ITEM_SELECTOR_WAIT));
        }
        80 => {
            if selection == occupied_count - 1 {
                return Ok((0, NATIVE_ITEM_SELECTOR_WAIT));
            }
            return Ok((selection + 1, NATIVE_ITEM_SELECTOR_WAIT));
        }
        75 if selection >= 4 => return Ok((selection - 4, NATIVE_ITEM_SELECTOR_WAIT)),
        77 if selection <= 3 && selection + 4 < occupied_count => {
            return Ok((selection + 4, NATIVE_ITEM_SELECTOR_WAIT));
        }
        28 | 57 if !battle_use || selected_effect_type != 0 => return Ok((selection, NATIVE_ITEM_SELECTOR_CONFIRM)),
        1 => return Ok((selection, NATIVE_ITEM_SELECTOR_CANCEL)),
        _ => {}
    }
    Ok((selection, NATIVE_ITEM_SELECTOR_WAIT))
}

/// Go：`native_item_selector.go:55`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemSelectorCell {
    pub raw_slot: i64,
    pub display_index: i64,
    pub item_id: u8,
    pub equipped: bool,
    pub selected: bool,
    pub label_x: i64,
    pub label_y: i64,
    pub category_icon: i64,
    pub stat_icon: i64,
    pub stat_value: i64,
    pub has_stat_value: bool,
}

/// NativeItemSelectorCells reproduces 0x184c0's compact display and geometry.
/// Empty/reserved raw slots are skipped; native writers maintain an occupied
/// prefix, but keeping RawSlot explicit exposes malformed/editable holes.
///
/// Go：`native_item_selector.go:72`，`func NativeItemSelectorCells( records []byte, unitIndex, selectedRawSlot int, effectRows []byte, ) ([]NativeItemSelectorCell, error)`
pub fn native_item_selector_cells(
    records: &[u8],
    unit_index: i64,
    selected_raw_slot: i64,
    effect_rows: &[u8],
) -> Result<Vec<NativeItemSelectorCell>> {
    let base = unit_index.wrapping_mul(NATIVE_RECORD_SIZE);
    if unit_index < 0 || base < 0 || base.wrapping_add(NATIVE_RECORD_SIZE) > records.len() as i64 {
        return Err(record_bounds_error(unit_index));
    }
    if !(-1..8).contains(&selected_raw_slot) {
        return Err(battle_err!("native item selector raw slot={selected_raw_slot} is invalid"));
    }
    let record = &records[base as usize..(base + NATIVE_RECORD_SIZE) as usize];
    let row_size = NATIVE_ITEM_EFFECT_ROW_SIZE as usize;
    let word = |row: &[u8], off: usize| u16::from_le_bytes([row[off], row[off + 1]]) as i16 as i64;
    let mut cells: Vec<NativeItemSelectorCell> = Vec::with_capacity(8);
    for raw_slot in 0..8usize {
        let flag = record[0x0a + raw_slot * 2];
        if flag & 0x80 != 0 {
            continue;
        }
        let item_id = record[0x0b + raw_slot * 2];
        let row_base = item_id as usize * row_size;
        if row_base + row_size > effect_rows.len() {
            return Err(battle_err!("native item selector row {item_id} is unavailable"));
        }
        let row = &effect_rows[row_base..row_base + row_size];
        let display = cells.len() as i64;
        let mut cell = NativeItemSelectorCell {
            raw_slot: raw_slot as i64,
            display_index: display,
            item_id,
            equipped: flag & 0x40 != 0,
            selected: raw_slot as i64 == selected_raw_slot,
            label_x: 42 + 150 * (display / 4),
            label_y: 103 + 22 * (display % 4),
            ..Default::default()
        };
        if row[0] < 0x15 {
            cell.category_icon = 59;
            cell.stat_icon = 64;
            cell.stat_value = word(row, 1);
            cell.has_stat_value = true;
        } else if row[0] < 0x20 {
            cell.category_icon = 60;
            cell.stat_icon = 65;
            cell.stat_value = word(row, 5);
            cell.has_stat_value = true;
        } else {
            cell.category_icon = 61;
            cell.stat_icon = match (row[0], row[0x0d]) {
                (0x20, 5) => 66,
                (0x20, 11) => 67,
                _ => 41,
            };
            if cell.stat_icon == 66 || cell.stat_icon == 67 {
                cell.stat_value = word(row, 0x0e);
                cell.has_stat_value = true;
            }
        }
        if cell.equipped {
            cell.category_icon += 3;
        }
        cells.push(cell);
    }
    Ok(cells)
}

/// Go：`native_item_selector.go:137`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemPanelRegion {
    pub enabled: bool,
    pub source_x: i64,
    pub source_y: i64,
    pub dest_x: i64,
    pub dest_y: i64,
    pub width: i64,
    pub height: i64,
}

/// Go：`native_item_selector.go:144`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemPanelFrame {
    pub frame: i64,
    pub left: NativeItemPanelRegion,
    pub upper: NativeItemPanelRegion,
    pub bottom: NativeItemPanelRegion,
}

/// Go：`native_item_selector.go:151`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemPanelPoint {
    pub x: i64,
    pub y: i64,
}

/// Go：`native_item_selector.go:155`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemPanelBaseLayout {
    pub stride: i64,
    pub fdother_resource: i64,
    pub grid_origin: NativeItemPanelPoint,
    pub grid_columns: i64,
    pub grid_rows: i64,
    pub portrait_record_offset: i64,
    pub portrait_destination: NativeItemPanelPoint,
    pub upper_directory_entry: i64,
    pub upper_destination: NativeItemPanelPoint,
    pub bottom_directory_entry: i64,
    pub bottom_destination: NativeItemPanelPoint,
}

/// NativeItemPanelBaseLayoutFor transcribes 0x17eef without assigning guessed
/// meanings to the two large FDOTHER cells. 0x168b6 builds the 5x5 frame at
/// (5,7); DATO is selected by unit-record byte +7 and pasted at (8,10).
/// FDOTHER#5 LMI1 directory offsets +86/+90 are entries 20/21.
///
/// Go：`native_item_selector.go:173`，`func NativeItemPanelBaseLayoutFor() NativeItemPanelBaseLayout`
pub fn native_item_panel_base_layout_for() -> NativeItemPanelBaseLayout {
    NativeItemPanelBaseLayout {
        stride: 320,
        fdother_resource: 5,
        grid_origin: NativeItemPanelPoint { x: 5, y: 7 },
        grid_columns: 5,
        grid_rows: 5,
        portrait_record_offset: 7,
        portrait_destination: NativeItemPanelPoint { x: 8, y: 10 },
        upper_directory_entry: 20,
        upper_destination: NativeItemPanelPoint { x: 92, y: 7 },
        bottom_directory_entry: 21,
        bottom_destination: NativeItemPanelPoint { x: 5, y: 94 },
    }
}

/// Go：`native_item_selector.go:189`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemPanelBarCall {
    pub destination: NativeItemPanelPoint,
    pub fdother_base_entry: i64,
    pub current_offset: i64,
    pub maximum_offset: i64,
}

/// Go：`native_item_selector.go:195`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemPanelComparedNumberCall {
    pub destination: NativeItemPanelPoint,
    pub value_offset: i64,
    pub compare_offset: i64,
    pub width: i64,
}

/// Go：`native_item_selector.go:201`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemPanelRawNumberCall {
    pub destination: NativeItemPanelPoint,
    pub value_offset: i64,
    pub value_bytes: i64,
    pub color: i64,
    pub alternate_color: i64,
    pub alternate_flag_offset: i64,
    pub width: i64,
}

/// Go：`native_item_selector.go:211`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemPanelTextCall {
    pub destination: NativeItemPanelPoint,
    pub record_offset: i64,
    pub fdtxt_base: i64,
}

/// Go：`native_item_selector.go:216`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemPanelIconCall {
    pub destination: NativeItemPanelPoint,
    pub fdother_entry: i64,
    pub alternate_fdother_entry: i64,
    pub flag_offset: i64,
    pub use_alternate_when_flag_zero: bool,
    pub draw_when_flag_nonzero: bool,
}

/// Go：`native_item_selector.go:225`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeItemPanelDataPlan {
    pub bars: Vec<NativeItemPanelBarCall>,
    pub compared_numbers: Vec<NativeItemPanelComparedNumberCall>,
    pub raw_numbers: Vec<NativeItemPanelRawNumberCall>,
    pub text: Vec<NativeItemPanelTextCall>,
    pub base_icon: NativeItemPanelIconCall,
    pub flag_icons: Vec<NativeItemPanelIconCall>,
}

/// NativeItemPanelDataPlanFor is the exact 0x17fc0 destination/record schedule.
/// Record offsets and primitive inputs stay raw where their game meaning has
/// not been independently established.
///
/// Go：`native_item_selector.go:237`，`func NativeItemPanelDataPlanFor() NativeItemPanelDataPlan`
pub fn native_item_panel_data_plan_for() -> NativeItemPanelDataPlan {
    let p = |x, y| NativeItemPanelPoint { x, y };
    let bar = |x, y, fdother_base_entry, current_offset, maximum_offset| NativeItemPanelBarCall {
        destination: p(x, y),
        fdother_base_entry,
        current_offset,
        maximum_offset,
    };
    let cmp = |x, y, value_offset, compare_offset| NativeItemPanelComparedNumberCall {
        destination: p(x, y),
        value_offset,
        compare_offset,
        width: 3,
    };
    let raw =
        |x, y, value_offset, value_bytes, alternate_color, alternate_flag_offset, width| NativeItemPanelRawNumberCall {
            destination: p(x, y),
            value_offset,
            value_bytes,
            color: 42,
            alternate_color,
            alternate_flag_offset,
            width,
        };
    let text =
        |x, y, record_offset, fdtxt_base| NativeItemPanelTextCall { destination: p(x, y), record_offset, fdtxt_base };
    let flag = |x, y, fdother_entry, flag_offset| NativeItemPanelIconCall {
        destination: p(x, y),
        fdother_entry,
        flag_offset,
        draw_when_flag_nonzero: true,
        ..Default::default()
    };
    NativeItemPanelDataPlan {
        bars: vec![bar(198, 33, 23, 64, 66), bar(198, 52, 26, 68, 70)],
        compared_numbers: vec![cmp(267, 41, 64, 66), cmp(293, 41, 66, 66), cmp(267, 59, 68, 70), cmp(293, 59, 70, 70)],
        raw_numbers: vec![
            raw(157, 33, 33, 1, 0, 0, 2),
            raw(157, 44, 60, 1, 0, 0, 2),
            raw(157, 55, 59, 1, 0, 0, 2),
            raw(117, 55, 62, 2, 0, 0, 3),
            raw(157, 67, 72, 2, 119, 34, 3),
            raw(157, 79, 74, 2, 119, 35, 3),
            raw(117, 67, 76, 2, 119, 36, 3),
            raw(117, 79, 78, 2, 119, 36, 3),
        ],
        text: vec![text(99, 13, 8, 1), text(211, 13, 31, 140), text(251, 13, 32, 150)],
        base_icon: NativeItemPanelIconCall {
            destination: p(101, 30),
            fdother_entry: 53,
            alternate_fdother_entry: 54,
            flag_offset: 6,
            use_alternate_when_flag_zero: true,
            draw_when_flag_nonzero: false,
        },
        flag_icons: vec![flag(194, 68, 55, 37), flag(229, 68, 56, 38), flag(264, 68, 57, 39)],
    }
}

/// NativeItemPanelFrameFor reproduces 0x18409's three clipped 320-stride
/// memmove regions. Frame 11 is the smallest state; frame 0 is fully open.
///
/// Go：`native_item_selector.go:278`，`func NativeItemPanelFrameFor(frame int) (NativeItemPanelFrame, error)`
pub fn native_item_panel_frame_for(frame: i64) -> Result<NativeItemPanelFrame> {
    if !(0..=11).contains(&frame) {
        return Err(battle_err!("native item panel frame={frame} is invalid"));
    }
    let mut result = NativeItemPanelFrame { frame, ..Default::default() };

    let left_arg = if frame >= 6 { 101 - 16 * frame } else { 5 };
    result.left = NativeItemPanelRegion {
        enabled: true,
        source_x: 5,
        source_y: 7,
        dest_x: left_arg,
        dest_y: 7,
        width: 86,
        height: 86,
    };
    if left_arg < 0 {
        result.left.source_x = 5 - left_arg;
        result.left.dest_x = 0;
        result.left.width = 86 + left_arg;
    }

    if frame <= 8 {
        let upper_arg = if frame >= 3 { 55 - 16 * frame } else { 7 };
        result.upper = NativeItemPanelRegion {
            enabled: true,
            source_x: 92,
            source_y: 7,
            dest_x: 92,
            dest_y: upper_arg,
            width: 223,
            height: 86,
        };
        if upper_arg < 0 {
            result.upper.source_y = 7 - upper_arg;
            result.upper.dest_y = 0;
            result.upper.height = 86 + upper_arg;
        }
    }

    if frame < 6 {
        let bottom_y = 94 + 16 * frame;
        let height = if bottom_y + 102 > 200 { 200 - bottom_y } else { 102 };
        result.bottom = NativeItemPanelRegion {
            enabled: true,
            source_x: 5,
            source_y: 94,
            dest_x: 5,
            dest_y: bottom_y,
            width: 310,
            height,
        };
    }
    Ok(result)
}

/// Go：`native_item_selector.go:328`，`func NativeItemPanelSchedule(opening bool) ([]NativeItemPanelFrame, error)`
pub fn native_item_panel_schedule(opening: bool) -> Result<Vec<NativeItemPanelFrame>> {
    let mut result = Vec::with_capacity(12);
    for step in 0..12 {
        let frame = if opening { 11 - step } else { step };
        result.push(native_item_panel_frame_for(frame)?);
    }
    Ok(result)
}

#[cfg(test)]
mod native_item_selector_test;
