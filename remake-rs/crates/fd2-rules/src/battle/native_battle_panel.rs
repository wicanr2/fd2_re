//! Go 來源：`remake/internal/battle/native_battle_panel.go`（193 行）。

use super::*;

pub(crate) const NATIVE_BATTLE_PANEL_WIDTH: i64 = 149;

pub(crate) const NATIVE_BATTLE_PANEL_HEIGHT: i64 = 42;

/// NativeBattlePanelValues 是 0x18C6D 直接交給 bar／digit consumers 的可見值。
/// 它不包含 raw +6／+8 selector；caller 若缺原始 selector，不得藉此宣稱完整
/// RenderNativeBattlePanel parity。
///
/// Go：`native_battle_panel.go:18`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeBattlePanelValues {
    pub level: i64,
    pub hp: i64,
    pub max_hp: i64,
    pub mp: i64,
    pub max_mp: i64,
}

/// RenderNativeBattlePanelValuesAt 重用 0x18C6D 的框、bar與digit核心，但不畫姓名。
/// 這讓已有可見值與明確panel原點的E1 caller避免另寫一套RGBA近似；完整raw caller
/// 仍應使用 RenderNativeBattlePanel。
///
/// Go：`native_battle_panel.go:27`，`func RenderNativeBattlePanelValuesAt( assets NativeItemPanelDataAssets, dst []byte, x, y int, values NativeBattlePanelValues, ) error`
pub fn render_native_battle_panel_values_at(
    assets: &NativeItemPanelDataAssets,
    dst: &mut [u8],
    x: i64,
    y: i64,
    values: NativeBattlePanelValues,
) -> Result<()> {
    todo!("battle/native_battle_panel.go:RenderNativeBattlePanelValuesAt")
}

/// Go：`native_battle_panel.go:48`，`func renderNativeBattlePanelValuesAt( assets NativeItemPanelDataAssets, dst []byte, x, y int, values NativeBattlePanelValues, ) error`
pub(crate) fn render_native_battle_panel_values_at_inner(
    assets: &NativeItemPanelDataAssets,
    dst: &mut [u8],
    x: i64,
    y: i64,
    values: NativeBattlePanelValues,
) -> Result<()> {
    todo!("battle/native_battle_panel.go:renderNativeBattlePanelValuesAt")
}

/// NativeBattlePanelRecordForUnit materializes only the raw fields consumed by
/// 0x2A289→0x18C6D. Raw +6 and +8 provenance remain mandatory; normalized Camp,
/// Name, Portrait, and BattleFig are not substitutes for either selector.
///
/// Go：`native_battle_panel.go:96`，`func NativeBattlePanelRecordForUnit(unit *Unit) ([]byte, error)`
pub fn native_battle_panel_record_for_unit(unit: Option<&Unit>) -> Result<Vec<u8>> {
    todo!("battle/native_battle_panel.go:NativeBattlePanelRecordForUnit")
}

/// RenderNativeBattlePanel reproduces the complete narrow 0x2A289→0x18C6D
/// indexed compositor. It stages the full 320x200 destination so missing raw
/// provenance, assets, text, or geometry never leaves a partially drawn panel.
///
/// Go：`native_battle_panel.go:126`，`func RenderNativeBattlePanel( assets NativeItemPanelDataAssets, record, dst []byte, unitIndex, rawChapter int, ) error`
pub fn render_native_battle_panel(
    assets: &NativeItemPanelDataAssets,
    record: &[u8],
    dst: &mut [u8],
    unit_index: i64,
    raw_chapter: i64,
) -> Result<()> {
    todo!("battle/native_battle_panel.go:RenderNativeBattlePanel")
}

/// RenderNativeBattlePanelWithoutName 保留原版面板、數值及定位，只省略姓名。
/// 多語 owner 必須以同一 raw +8 selector 查詢語系目錄後，在原姓名矩形重繪。
///
/// Go：`native_battle_panel.go:136`，`func RenderNativeBattlePanelWithoutName( assets NativeItemPanelDataAssets, record, dst []byte, unitIndex, rawChapter int, ) error`
pub fn render_native_battle_panel_without_name(
    assets: &NativeItemPanelDataAssets,
    record: &[u8],
    dst: &mut [u8],
    unit_index: i64,
    raw_chapter: i64,
) -> Result<()> {
    todo!("battle/native_battle_panel.go:RenderNativeBattlePanelWithoutName")
}

/// NativeBattlePanelOrigin 回傳 0x2A289→0x18C6D 的原版面板原點。
///
/// Go：`native_battle_panel.go:145`，`func NativeBattlePanelOrigin(record []byte, unitIndex, rawChapter int) (int, int, error)`
pub fn native_battle_panel_origin(record: &[u8], unit_index: i64, raw_chapter: i64) -> Result<(i64, i64)> {
    todo!("battle/native_battle_panel.go:NativeBattlePanelOrigin")
}

/// Go：`native_battle_panel.go:156`，`func renderNativeBattlePanel( assets NativeItemPanelDataAssets, record, dst []byte, unitIndex, rawChapter int, renderName bool, ) error`
pub(crate) fn render_native_battle_panel_inner(
    assets: &NativeItemPanelDataAssets,
    record: &[u8],
    dst: &mut [u8],
    unit_index: i64,
    raw_chapter: i64,
    render_name: bool,
) -> Result<()> {
    todo!("battle/native_battle_panel.go:renderNativeBattlePanel")
}
