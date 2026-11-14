//! Go 來源：`remake/internal/campaign/native_shop_recipient.go`（29 行）。

use super::*;

/// ComposeNativeShopConsumableRecipientFrame reproduces the stable
/// 0x2f331→0x2e6b8 target used only when item type >= 0x20. Equipment takes
/// the distinct filtered 0x2e8cf→0x2ebe0 comparison panel and must not be
/// projected onto this two-column roster.
///
/// Go：`native_shop_recipient.go:13`，`func ComposeNativeShopConsumableRecipientFrame( stable []byte, shop *NativeShopAssets, rows []NativeRosterRow, selected, itemType int, strings *fdtxt.Strings, font *fdtxt.Font, ) ([]byte, error)`
pub fn compose_native_shop_consumable_recipient_frame(
    stable: &[u8],
    shop: Option<&NativeShopAssets>,
    rows: &[NativeRosterRow],
    selected: i64,
    item_type: i64,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
) -> Result<Vec<u8>> {
    todo!("campaign/native_shop_recipient.go:ComposeNativeShopConsumableRecipientFrame")
}
