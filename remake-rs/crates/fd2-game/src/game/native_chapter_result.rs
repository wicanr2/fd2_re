//! Go 來源：`remake/cmd/fd2/native_chapter_result.go`（82 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativeChapterResultJob 保留原版同步處理器在非同步介面裡的擁有權：
/// 當前列對白完全收框之後，才讀取下一列的raw條件。
///
/// Go：`native_chapter_result.go:7`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeChapterResultJob {
    pub(crate) next_rule: i64,
    pub(crate) code: i64,
    pub(crate) matched: Vec<String>,
}
