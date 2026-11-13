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
    todo!("battle/native_item_selector.go:AdvanceNativeItemSelector")
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
    todo!("battle/native_item_selector.go:NativeItemSelectorCells")
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
    todo!("battle/native_item_selector.go:NativeItemPanelBaseLayoutFor")
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
    todo!("battle/native_item_selector.go:NativeItemPanelDataPlanFor")
}

/// NativeItemPanelFrameFor reproduces 0x18409's three clipped 320-stride
/// memmove regions. Frame 11 is the smallest state; frame 0 is fully open.
///
/// Go：`native_item_selector.go:278`，`func NativeItemPanelFrameFor(frame int) (NativeItemPanelFrame, error)`
pub fn native_item_panel_frame_for(frame: i64) -> Result<NativeItemPanelFrame> {
    todo!("battle/native_item_selector.go:NativeItemPanelFrameFor")
}

/// Go：`native_item_selector.go:328`，`func NativeItemPanelSchedule(opening bool) ([]NativeItemPanelFrame, error)`
pub fn native_item_panel_schedule(opening: bool) -> Result<Vec<NativeItemPanelFrame>> {
    todo!("battle/native_item_selector.go:NativeItemPanelSchedule")
}
