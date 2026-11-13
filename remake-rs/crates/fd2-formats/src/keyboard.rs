//! FD2.EXE 鍵盤輔助程式存取的兩個 BIOS 資料區 word。
//!
//! 移植自 Go `remake/internal/fdsave/keyboard_buffer.go`。欄位名稱刻意保留位址：
//! 原版只證明了 word 操作，重製端不把它們升格為按鍵碼、事件佇列或遊戲語意。
//!
//! FD2.EXE 線性位址（固定參考雜湊）：
//! - `0x10620` 比較 word `0x41a` 與 word `0x41c`。
//! - `0x4e031` 把 word `0x41a` 複製到 word `0x41c`。

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeBiosKeyboardState {
    pub word_041a: u16,
    pub word_041c: u16,
}

impl NativeBiosKeyboardState {
    /// `0x10620` 產生的布林判斷；不解讀 word 的值或對應的實體按鍵。
    pub fn words_differ(&self) -> bool {
        self.word_041a != self.word_041c
    }

    /// 套用 `0x4e031` 的 word 寫入，回傳目的端是否改變。呼叫端不可把它解讀為
    /// 已消耗的按鍵或遊戲事件。
    pub fn copy_041a_to_041c(&mut self) -> bool {
        if self.word_041a == self.word_041c {
            return false;
        }
        self.word_041c = self.word_041a;
        true
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn words_differ() {
        let equal = NativeBiosKeyboardState { word_041a: 0x1234, word_041c: 0x1234 };
        assert!(!equal.words_differ(), "equal raw words reported as different");
        let different = NativeBiosKeyboardState { word_041a: 0x1234, word_041c: 0x5678 };
        assert!(different.words_differ(), "different raw words reported as equal");
    }

    #[test]
    fn copy_preserves_raw_word_contract() {
        let mut s = NativeBiosKeyboardState { word_041a: 0x1234, word_041c: 0x5678 };
        assert!(s.copy_041a_to_041c(), "different words did not report a write");
        assert!(s.word_041c == s.word_041a && !s.words_differ(), "copy did not close raw words: {s:?}");
        assert!(!s.copy_041a_to_041c(), "equal words reported a second write");
        // Go 版另測 nil 指標拒寫；Rust 的 &mut 不可能為空，該情況由型別排除。
    }
}
