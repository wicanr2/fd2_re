//! Go 來源：`remake/cmd/fd2/modern_hd_facility_dialogue.go`（200 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_facility_dialogue.go:13`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernHDFacilityDialogue {
    pub(crate) key: [u8; 32],
    pub(crate) font: Option<Font>,
    pub(crate) chrome: Option<ModernHDDialogueChrome>,
    pub(crate) buttons: Option<ModernHDConfirmation>,
    pub(crate) borders: [Vec<u8>; 2],
}

impl Game {
    /// Go：`modern_hd_facility_dialogue.go:63`，`func (g *Game) prepareModernHDFacilityDialogue() (*modernHDFacilityDialogue, error)`
    /// Go 第 1 個回傳值是指向既有狀態的指標，Rust 回傳借用。
    pub(crate) fn prepare_modern_hd_facility_dialogue(&mut self) -> Result<Option<&mut ModernHDFacilityDialogue>> {
        todo!("modern_hd_facility_dialogue.go:prepareModernHDFacilityDialogue")
    }
}

/// Compare original indices with the actual writer result. Matching colors do
/// not establish ownership, and a covered border must not become a new frame.
///
/// Go：`modern_hd_facility_dialogue.go:128`，`func facilityDialogueBorderVisible(frame, expected []byte) bool`
pub(crate) fn facility_dialogue_border_visible(frame: &[u8], expected: &[u8]) -> bool {
    todo!("modern_hd_facility_dialogue.go:facilityDialogueBorderVisible")
}
