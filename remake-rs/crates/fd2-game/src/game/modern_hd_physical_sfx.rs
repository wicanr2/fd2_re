//! Go 來源：`remake/cmd/fd2/modern_hd_physical_sfx.go`（113 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

impl Game {
    /// Go：`modern_hd_physical_sfx.go:90`，`func (g *Game) modernPhysicalSound(bank, cue int, original []byte) []byte`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn modern_physical_sound(&self, bank: i64, cue: i64, original: &[u8]) -> Vec<u8> {
        todo!("modern_hd_physical_sfx.go:modernPhysicalSound")
    }
}
