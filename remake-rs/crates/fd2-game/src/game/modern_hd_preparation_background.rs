//! Go 來源：`remake/cmd/fd2/modern_hd_preparation_background.go`（189 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

impl NativeTownFrameSource {
    /// Go：`modern_hd_preparation_background.go:11`，`func (d *nativeTownFrameSource) matches(source []byte) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn matches(&self, source: &[u8]) -> bool {
        todo!("modern_hd_preparation_background.go:matches")
    }
}
