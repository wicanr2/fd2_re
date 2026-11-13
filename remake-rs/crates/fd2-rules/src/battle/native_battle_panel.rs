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
    if dst.len() as i64 != NATIVE_ITEM_PANEL_BYTES
        || x < 0
        || y < 0
        || x + NATIVE_BATTLE_PANEL_WIDTH > 320
        || y + NATIVE_BATTLE_PANEL_HEIGHT > 200
        || assets.battle_panel.width != NATIVE_BATTLE_PANEL_WIDTH
        || assets.battle_panel.height != NATIVE_BATTLE_PANEL_HEIGHT
        || assets.raw_cells.is_none()
        || assets.frames.is_none()
    {
        return Err(battle_err!("battle: native battle panel value inputs are invalid"));
    }
    let mut staged = dst.to_vec();
    render_native_battle_panel_values_at_inner(assets, &mut staged, x, y, values)?;
    dst.copy_from_slice(&staged);
    Ok(())
}

/// Go：`native_battle_panel.go:48`，`func renderNativeBattlePanelValuesAt( assets NativeItemPanelDataAssets, dst []byte, x, y int, values NativeBattlePanelValues, ) error`
pub(crate) fn render_native_battle_panel_values_at_inner(
    assets: &NativeItemPanelDataAssets,
    dst: &mut [u8],
    x: i64,
    y: i64,
    values: NativeBattlePanelValues,
) -> Result<()> {
    assets
        .battle_panel
        .blit_opaque_at(dst, 320, x, y, false)
        .map_err(|e| battle_err!("battle: native battle panel base: {e}"))?;
    let (raw_cells, frames) = (assets.raw_cells_or_empty(), assets.frames_or_empty());
    let p = |x, y| NativeItemPanelPoint { x, y };
    for (point, current, max, base) in
        [(p(x + 21, y + 22), values.hp, values.max_hp, 23), (p(x + 21, y + 31), values.mp, values.max_mp, 26)]
    {
        let call = NativeItemPanelBarCall { destination: point, fdother_base_entry: base, ..Default::default() };
        blit_native_item_panel_bar(raw_cells, dst, call, current, max)?;
    }
    blit_native_item_panel_number(frames, dst, p(x + 132, y + 4), values.level, 31, 2)?;
    for (point, current, max) in
        [(p(x + 126, y + 21), values.hp, values.max_hp), (p(x + 126, y + 30), values.mp, values.max_mp)]
    {
        let base = if current == max { 31 } else { 42 };
        blit_native_item_panel_number(frames, dst, point, current, base, 3)?;
    }
    Ok(())
}

/// NativeBattlePanelRecordForUnit materializes only the raw fields consumed by
/// 0x2A289→0x18C6D. Raw +6 and +8 provenance remain mandatory; normalized Camp,
/// Name, Portrait, and BattleFig are not substitutes for either selector.
///
/// Go：`native_battle_panel.go:96`，`func NativeBattlePanelRecordForUnit(unit *Unit) ([]byte, error)`
pub fn native_battle_panel_record_for_unit(unit: Option<&Unit>) -> Result<Vec<u8>> {
    let Some(unit) = unit.filter(|u| u.has_native_record_byte6) else {
        return Err(battle_err!("native battle panel: unit lacks raw record byte +6"));
    };
    let (raw_byte8, ok) = native_record_byte8_for_unit(Some(unit));
    if !ok {
        return Err(battle_err!("native battle panel: unit lacks raw record byte +8"));
    }
    if unit.lv < 0 || unit.lv > 0xff {
        return Err(battle_err!("native battle panel: level is outside native byte range"));
    }
    for value in [unit.hp, unit.max_hp, unit.mp, unit.max_mp] {
        if value < i16::MIN as i64 || value > i16::MAX as i64 {
            return Err(battle_err!("native battle panel: HP/MP field is outside native word range"));
        }
    }
    let mut record = vec![0u8; NATIVE_RECORD_SIZE as usize];
    record[6] = unit.native_record_byte6;
    record[8] = raw_byte8;
    record[0x21] = unit.lv as u8;
    put_native_item_panel_word(&mut record, 0x40, unit.hp);
    put_native_item_panel_word(&mut record, 0x42, unit.max_hp);
    put_native_item_panel_word(&mut record, 0x44, unit.mp);
    put_native_item_panel_word(&mut record, 0x46, unit.max_mp);
    Ok(record)
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
    render_native_battle_panel_inner(assets, record, dst, unit_index, raw_chapter, true)
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
    render_native_battle_panel_inner(assets, record, dst, unit_index, raw_chapter, false)
}

/// NativeBattlePanelOrigin 回傳 0x2A289→0x18C6D 的原版面板原點。
///
/// Go：`native_battle_panel.go:145`，`func NativeBattlePanelOrigin(record []byte, unitIndex, rawChapter int) (int, int, error)`
pub fn native_battle_panel_origin(record: &[u8], unit_index: i64, raw_chapter: i64) -> Result<(i64, i64)> {
    if (record.len() as i64) < NATIVE_RECORD_SIZE || unit_index < 0 || !(0..=0xff).contains(&raw_chapter) {
        return Err(battle_err!("battle: native battle panel selectors are invalid"));
    }
    if record[6] == 0 || raw_chapter == 24 && unit_index == 17 {
        return Ok((0, 154));
    }
    Ok((171, 4))
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
    if (record.len() as i64) < NATIVE_RECORD_SIZE || dst.len() as i64 != NATIVE_ITEM_PANEL_BYTES {
        return Err(battle_err!("battle: native battle panel inputs are invalid"));
    }
    if unit_index < 0
        || !(0..=0xff).contains(&raw_chapter)
        || assets.battle_panel.width != NATIVE_BATTLE_PANEL_WIDTH
        || assets.battle_panel.height != NATIVE_BATTLE_PANEL_HEIGHT
        || assets.raw_cells.is_none()
        || assets.frames.is_none()
        || render_name && (assets.strings.is_none() || assets.font.is_none())
    {
        return Err(battle_err!("battle: native battle panel assets or selectors are invalid"));
    }
    let (x, y) = native_battle_panel_origin(record, unit_index, raw_chapter)?;
    let mut staged = dst.to_vec();
    let word = |off: usize| u16::from_le_bytes([record[off], record[off + 1]]) as i16 as i64;
    let values = NativeBattlePanelValues {
        level: record[0x21] as i64,
        hp: word(0x40),
        max_hp: word(0x42),
        mp: word(0x44),
        max_mp: word(0x46),
    };
    render_native_battle_panel_values_at_inner(assets, &mut staged, x, y, values)?;
    if render_name {
        blit_native_item_panel_text_opt(
            assets.strings.as_ref(),
            assets.font.as_ref(),
            &mut staged,
            NativeItemPanelPoint { x: x + 5, y: y + 4 },
            record[8] as i64 + 1,
            205,
        )?;
    }
    dst.copy_from_slice(&staged);
    Ok(())
}

#[cfg(test)]
mod native_battle_panel_test;
