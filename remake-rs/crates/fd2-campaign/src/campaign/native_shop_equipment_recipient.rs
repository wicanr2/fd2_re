//! Go 來源：`remake/internal/campaign/native_shop_equipment_recipient.go`（228 行）。

use super::*;

pub const NATIVE_SHOP_EQUIPMENT_RECIPIENT_VISIBLE: i64 = 3;

/// Go：`native_shop_equipment_recipient.go:14`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeShopEquipmentRecipientRow {
    pub sprite: fdicon::Sprite,
    pub name_text_index: i64,
    pub current: [i64; 4],
    pub candidate: [i64; 4],
}

/// NativeShopEquipmentRecordForUnit materializes the exact 0x50-byte fields
/// consumed by 0x2efb7. The general item-panel adapter intentionally omits
/// base AP/DP at +0x37/+0x39, so equipment preview must require independently
/// initialized equipment-base provenance rather than silently previewing zero.
///
/// Go：`native_shop_equipment_recipient.go:25`，`func NativeShopEquipmentRecordForUnit(unit *battle.Unit) ([]byte, error)`
pub fn native_shop_equipment_record_for_unit(unit: Option<&Unit>) -> Result<Vec<u8>> {
    let unit = match unit {
        Some(u)
            if u.equipment_base_set
                && (-0x8000..=0x7fff).contains(&u.base_ap)
                && (-0x8000..=0x7fff).contains(&u.base_dp) =>
        {
            u
        }
        _ => return Err(campaign_err!("campaign: native shop equipment record lacks base AP/DP provenance")),
    };
    let mut record = battle::native_item_panel_record_for_unit(Some(unit))?;
    record[0x37..0x39].copy_from_slice(&(unit.base_ap as i16 as u16).to_le_bytes());
    record[0x39..0x3b].copy_from_slice(&(unit.base_dp as i16 as u16).to_le_bytes());
    Ok(record)
}

/// NativeShopEquipmentCandidateStats reproduces 0x2efb7. It previews replacing
/// the equipped item in the selected item's <=0x14 or >0x14 category while
/// retaining equipped contributions from the opposite category.
///
/// Go：`native_shop_equipment_recipient.go:45`，`func NativeShopEquipmentCandidateStats( record []byte, itemID int, effectRows []byte, ) ([4]int, error)`
pub fn native_shop_equipment_candidate_stats(record: &[u8], item_id: i64, effect_rows: &[u8]) -> Result<[i64; 4]> {
    const RECORD_SIZE: usize = 0x50;
    if record.len() < RECORD_SIZE || item_id < 0 {
        return Err(campaign_err!("campaign: native shop equipment preview state is invalid"));
    }
    let item = native_shop_effect_row(effect_rows, item_id)?;
    let w = native_shop_signed_word;
    let mut result = [
        w(record, 0x37) + w(&item, 1),
        w(record, 0x39) + w(&item, 5),
        w(record, 0x3e) + w(&item, 3),
        w(record, 0x3e) + w(&item, 7),
    ];
    for slot in 0..8 {
        let cell = 0x0a + 2 * slot;
        if record[cell] & 0x40 == 0 {
            continue;
        }
        let equipped = native_shop_effect_row(effect_rows, record[cell + 1] as i64)?;
        if (item[0] <= 0x14) == (equipped[0] <= 0x14) {
            continue;
        }
        result[0] += w(&equipped, 1);
        result[1] += w(&equipped, 5);
        result[2] += w(&equipped, 3);
        result[3] += w(&equipped, 7);
    }
    Ok(result)
}

/// Go：`native_shop_equipment_recipient.go:87`，`func NativeShopEquipmentCurrentStats(record []byte) ([4]int, error)`
pub fn native_shop_equipment_current_stats(record: &[u8]) -> Result<[i64; 4]> {
    if record.len() < 0x50 {
        return Err(campaign_err!("campaign: native shop equipment record is shorter than 80 bytes"));
    }
    let w = native_shop_signed_word;
    Ok([w(record, 0x48), w(record, 0x4a), w(record, 0x4c), w(record, 0x4e)])
}

/// ComposeNativeShopEquipmentRecipientFrame reproduces the stable
/// 0x2e8cf→0x2ebe0 target: three rows, selected FDTXT name color, and AP/DP/
/// HIT/EV current→candidate comparisons. Opening/closing remain caller-owned.
///
/// Go：`native_shop_equipment_recipient.go:104`，`func ComposeNativeShopEquipmentRecipientFrame( stable []byte, shop *NativeShopAssets, itemAssets battle.NativeItemPanelDataAssets, rows []NativeShopEquipmentRecipientRow, selected int, ) ([]byte, error)`
pub fn compose_native_shop_equipment_recipient_frame(
    stable: &[u8],
    shop: Option<&NativeShopAssets>,
    item_assets: &battle::NativeItemPanelDataAssets,
    rows: &[NativeShopEquipmentRecipientRow],
    selected: i64,
) -> Result<Vec<u8>> {
    compose_native_shop_equipment_recipient_frame_inner(stable, shop, item_assets, rows, selected, true)
}

/// Go：`native_shop_equipment_recipient.go:114`，`func ComposeNativeShopEquipmentRecipientFrameWithoutNames( stable []byte, shop *NativeShopAssets, itemAssets battle.NativeItemPanelDataAssets, rows []NativeShopEquipmentRecipientRow, selected int, ) ([]byte, error)`
pub fn compose_native_shop_equipment_recipient_frame_without_names(
    stable: &[u8],
    shop: Option<&NativeShopAssets>,
    item_assets: &battle::NativeItemPanelDataAssets,
    rows: &[NativeShopEquipmentRecipientRow],
    selected: i64,
) -> Result<Vec<u8>> {
    compose_native_shop_equipment_recipient_frame_inner(stable, shop, item_assets, rows, selected, false)
}

/// Go：`native_shop_equipment_recipient.go:124`，`func composeNativeShopEquipmentRecipientFrame( stable []byte, shop *NativeShopAssets, itemAssets battle.NativeItemPanelDataAssets, rows []NativeShopEquipmentRecipientRow, selected int, renderNames bool, ) ([]byte, error)`
pub(crate) fn compose_native_shop_equipment_recipient_frame_inner(
    stable: &[u8],
    shop: Option<&NativeShopAssets>,
    item_assets: &battle::NativeItemPanelDataAssets,
    rows: &[NativeShopEquipmentRecipientRow],
    selected: i64,
    render_names: bool,
) -> Result<Vec<u8>> {
    let shop = match shop {
        Some(s)
            if stable.len() as i64 == NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT
                && !rows.is_empty()
                && rows.len() as i64 <= NATIVE_SHOP_EQUIPMENT_RECIPIENT_VISIBLE
                && selected >= 0
                && selected < rows.len() as i64
                && !(render_names && (item_assets.strings.is_none() || item_assets.font.is_none())) =>
        {
            s
        }
        _ => return Err(campaign_err!("campaign: native shop equipment recipient state is invalid")),
    };
    let mut frame = stable.to_vec();
    shop.panel.blit_opaque_at(&mut frame, NATIVE_SHOP_WIDTH, 5, 112, false)?;
    // (cell, x, dy, currentOffset, arrowOffset, resultOffset)
    const POSITIONS: [(usize, i64, i64, i64, i64, i64); 4] =
        [(0, 122, 3, 15, 35, 43), (1, 122, 12, 15, 35, 43), (2, 196, 3, 18, 38, 46), (3, 196, 12, 18, 38, 46)];
    for (i, row) in rows.iter().enumerate() {
        let i = i as i64;
        let y = 117 + 26 * i;
        row.sprite
            .blit_at(&mut frame, NATIVE_SHOP_WIDTH, 14, y)
            .map_err(|e| CampaignError::from(e).wrap(format!("campaign: equipment recipient row {i} sprite: ")))?;
        let foreground: u8 = if i == selected { 201 } else { 205 };
        if render_names && let (Some(strings), Some(font)) = (item_assets.strings.as_ref(), item_assets.font.as_ref()) {
            blit_native_class_list_text(&mut frame, strings, font, 40, y + 4, row.name_text_index, foreground)
                .map_err(|e| e.wrap(format!("campaign: equipment recipient row {i} name: ")))?;
        }
        for (stat, &(cell, x, dy, current_offset, arrow_offset, result_offset)) in POSITIONS.iter().enumerate() {
            shop.compare_cells[cell].blit_opaque_at(&mut frame, NATIVE_SHOP_WIDTH, x, y + dy, false)?;
            let color_base = native_shop_comparison_color(row.current[stat], row.candidate[stat]);
            battle::render_native_facility_number(
                item_assets,
                &mut frame,
                x + current_offset,
                y + dy,
                row.current[stat],
                color_base,
                3,
            )?;
            shop.compare_cells[4].blit_opaque_at(&mut frame, NATIVE_SHOP_WIDTH, x + arrow_offset, y + dy + 1, false)?;
            battle::render_native_facility_number(
                item_assets,
                &mut frame,
                x + result_offset,
                y + dy,
                row.candidate[stat],
                color_base,
                3,
            )?;
        }
    }
    Ok(frame)
}

/// Go：`native_shop_equipment_recipient.go:204`，`func nativeShopComparisonColor(current, candidate int) int`
pub(crate) fn native_shop_comparison_color(current: i64, candidate: i64) -> i64 {
    match current.cmp(&candidate) {
        std::cmp::Ordering::Equal => 31,
        std::cmp::Ordering::Less => 42,
        std::cmp::Ordering::Greater => 119,
    }
}

/// Go：`native_shop_equipment_recipient.go:215`，`func nativeShopEffectRow(effectRows []byte, itemID int) ([]byte, error)`
pub(crate) fn native_shop_effect_row(effect_rows: &[u8], item_id: i64) -> Result<Vec<u8>> {
    let start = item_id.wrapping_mul(battle::NATIVE_ITEM_EFFECT_ROW_SIZE);
    if item_id < 0 || start < 0 || start + battle::NATIVE_ITEM_EFFECT_ROW_SIZE > effect_rows.len() as i64 {
        return Err(campaign_err!("campaign: native shop item row {item_id} is unavailable"));
    }
    Ok(effect_rows[start as usize..(start + battle::NATIVE_ITEM_EFFECT_ROW_SIZE) as usize].to_vec())
}

/// Go：`native_shop_equipment_recipient.go:226`，`func nativeShopSignedWord(raw []byte, offset int) int`
pub(crate) fn native_shop_signed_word(raw: &[u8], offset: i64) -> i64 {
    let at = offset as usize;
    i16::from_le_bytes([raw[at], raw[at + 1]]) as i64
}
