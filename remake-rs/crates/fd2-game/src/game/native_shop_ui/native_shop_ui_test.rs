//! Go 來源：`remake/cmd/fd2/native_shop_ui_test.go`（R4 E 組，期望值不改）。
//!
//! 未移植：`:21`（`TestNativeShopProductionOwnerDrawsOriginalMenuAndPurchaseList`，以 `Draw` 推進並走購買、
//! 販賣、裝備與轉移的 `r8` 函式）與四個截圖狀態測試（`:785`、`:862`、`:940`、`:1027`，`r8`）。

use super::*;
use crate::test_support;

/// Go：`native_shop_ui_test.go:697` `TestNativeShopAssetsDoNotRequireFDOTHERArchive`
///
/// Go 另把 `FD2_ORIGINAL_FDOTHER` 換成不存在的檔，證明商店載入器不開封存檔；Rust 的載入器只讀分離素材包。
#[test]
fn test_native_shop_assets_do_not_require_fdother_archive() {
    if test_support::input("FD2_ASSET_PACK", "shop/FDOTHER_012/resource.json").is_none() {
        return;
    }
    let shared = load_native_class_ui_assets().expect("loadNativeClassUIAssets").unwrap();
    let shops = load_native_shop_ui_assets(Some(&shared)).expect("loadNativeShopUIAssets").unwrap();
    for variant in [1, 3, 5] {
        assert!(shops.shops.contains_key(&variant), "variant {variant} was not preflighted");
    }
}

/// Go：`native_shop_ui_test.go:1100` `TestNativeShopProductionOwnerFailsClosedForCustomVariant`
#[test]
fn test_native_shop_production_owner_fails_closed_for_custom_variant() {
    let c = campaign::Campaign {
        start: String::from("shop"),
        nodes: BTreeMap::from([(
            String::from("shop"),
            campaign::Node { r#type: String::from("shop"), native_hub_variant: 0, ..Default::default() },
        )]),
        ..Default::default()
    };
    let mut g = Game { camp: Some(campaign::new_runner(&c)), ..Default::default() };
    assert!(
        !g.setup_native_shop() && g.native_shop_mode.is_empty(),
        "custom shop node was incorrectly claimed by original shop owner"
    );
}
