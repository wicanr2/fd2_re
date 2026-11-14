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
    todo!("campaign/native_shop_equipment_recipient.go:NativeShopEquipmentRecordForUnit")
}

/// NativeShopEquipmentCandidateStats reproduces 0x2efb7. It previews replacing
/// the equipped item in the selected item's <=0x14 or >0x14 category while
/// retaining equipped contributions from the opposite category.
///
/// Go：`native_shop_equipment_recipient.go:45`，`func NativeShopEquipmentCandidateStats( record []byte, itemID int, effectRows []byte, ) ([4]int, error)`
pub fn native_shop_equipment_candidate_stats(record: &[u8], item_id: i64, effect_rows: &[u8]) -> Result<[i64; 4]> {
    todo!("campaign/native_shop_equipment_recipient.go:NativeShopEquipmentCandidateStats")
}

/// Go：`native_shop_equipment_recipient.go:87`，`func NativeShopEquipmentCurrentStats(record []byte) ([4]int, error)`
pub fn native_shop_equipment_current_stats(record: &[u8]) -> Result<[i64; 4]> {
    todo!("campaign/native_shop_equipment_recipient.go:NativeShopEquipmentCurrentStats")
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
    todo!("campaign/native_shop_equipment_recipient.go:composeNativeShopEquipmentRecipientFrame")
}

/// Go：`native_shop_equipment_recipient.go:204`，`func nativeShopComparisonColor(current, candidate int) int`
pub(crate) fn native_shop_comparison_color(current: i64, candidate: i64) -> i64 {
    todo!("campaign/native_shop_equipment_recipient.go:nativeShopComparisonColor")
}

/// Go：`native_shop_equipment_recipient.go:215`，`func nativeShopEffectRow(effectRows []byte, itemID int) ([]byte, error)`
pub(crate) fn native_shop_effect_row(effect_rows: &[u8], item_id: i64) -> Result<Vec<u8>> {
    todo!("campaign/native_shop_equipment_recipient.go:nativeShopEffectRow")
}

/// Go：`native_shop_equipment_recipient.go:226`，`func nativeShopSignedWord(raw []byte, offset int) int`
pub(crate) fn native_shop_signed_word(raw: &[u8], offset: i64) -> i64 {
    todo!("campaign/native_shop_equipment_recipient.go:nativeShopSignedWord")
}
