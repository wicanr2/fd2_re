//! Go 來源：`remake/cmd/fd2/assets_glob_native.go`（43 行）。
//!
//! 骨架：Go 檔的 1 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// assetGlob 同 assetPath 的五層查找,用於萬用字元批次載入(sprite/portrait/figani)。
/// 第一層有命中(非空)就整層採用,不同層的檔案不混拼。
///
/// Go：`assets_glob_native.go:12`，`func assetGlob(pattern string) []string`
pub(crate) fn asset_glob(pattern: &str) -> Vec<String> {
    todo!("assets_glob_native.go:assetGlob")
}
