//! Go 來源：`remake/cmd/fd2/native_class_ui_test.go`（R4 E 組，期望值不改）。
//!
//! 未移植：`:13`（`TestDrawNativeClassListUsesPlayerOriginalAssets`，以 `Draw` 斷言畫面並走轉職、復活的 `r8` 函式）。

use super::*;
use crate::test_support;

/// Go：`native_class_ui_test.go:202` `TestLoadNativeClassUIUsesSeparatedFDTXT`
///
/// Go 把 `FD2_ORIGINAL_FDTXT` 換成不存在的檔，證明 FDTXT 來自分離素材包；Rust 的載入器只讀分離素材包。
#[test]
fn test_load_native_class_ui_uses_separated_fdtxt() {
    if test_support::input("FD2_ASSET_PACK", "text").is_none() {
        return;
    }
    let assets = load_native_class_ui_assets().expect("loadNativeClassUIAssets").unwrap();
    let count = assets.strings.as_ref().map(|s| s.count());
    assert_eq!(count, Some(661), "separated FDTXT_000 strings");
}
