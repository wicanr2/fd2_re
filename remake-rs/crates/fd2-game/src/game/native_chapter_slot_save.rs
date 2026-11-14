//! Go 來源：`remake/cmd/fd2/native_chapter_slot_save.go`（212 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// saveNativeChapterSlot 是酒店存檔（0x30012）的原版槽寫回：以標題 LOAD 讀進來的
    /// FD2.SAV plaintext 為底；正常 START 首次保存按已確認政策建立新名冊。
    /// 把目前持續隊伍與金幣寫進 slot，重算校驗和後原子取代
    /// FD2_NATIVE_SAVE。寫成功後同一份 plaintext 成為下一次存檔的基底。
    ///
    /// Go：`native_chapter_slot_save.go:17`，`func (g *Game) saveNativeChapterSlot(slot int) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn save_native_chapter_slot(&mut self, slot: i64) -> Result<()> {
        todo!("native_chapter_slot_save.go:saveNativeChapterSlot")
    }
}
