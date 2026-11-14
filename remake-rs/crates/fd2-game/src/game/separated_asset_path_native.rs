//! Go 來源：`remake/cmd/fd2/separated_asset_path_native.go`（18 行）。
//!
//! 骨架：Go 檔的 1 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// separatedAssetPath 解析分離素材（原版衍生資產）。FD2_ASSET_PACK 指定素材包時
/// 一律以它為準；沒有指定就退回一般資產查找，讓「素材放在遊戲目錄旁」的安裝方式
/// 也能運作。
///
/// Go：`separated_asset_path_native.go:13`，`func separatedAssetPath(relative string) string`
pub(crate) fn separated_asset_path(relative: &str) -> String {
    todo!("separated_asset_path_native.go:separatedAssetPath")
}
