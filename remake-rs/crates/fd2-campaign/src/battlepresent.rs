//! 對應凍結 Go 版 `remake/internal/battlepresent`：戰鬥演出的索引色畫格合成。
//!
//! 這個套件把 `figani` 已閉合的指令演出排程（`0x2A6BD` 各指令處理器、`0x29164`、`0x2B659`、
//! `0x29C90`、`0x2111A` 等）合成為 320×200 的索引色畫格，連同每張畫格消費過的來源，交給演出端
//! 在 Draw 確認後依序發布。它不推進戰鬥狀態、不發布數值，也不播放音效。
//!
//! 每個 Go 檔 `x.go` 對應子模組 `battlepresent/x.rs`，測試放在 `battlepresent/x/<t>_test.rs`。
//!
//! # 型別對應
//!
//! - Go `int` → `i64`；`byte` → `u8`。
//! - Go 的 `*figani.Animation` 參數可為 nil → `Option<&Animation>`；以值傳入的 `figani.Frame`、
//!   `fdother.Frame` → `&Frame`（Go 只讀或先複製再改，借用即可）。
//! - 輸入結構裡的 `*figani.Animation`、`[]*figani.Animation` → `Option<&'a Animation>`、
//!   `Vec<Option<&'a Animation>>`；`[]byte` 底圖 → `&'a [u8]` 或 `Vec<&'a [u8]>`。
//! - 來源紀錄（`NativeCommandSources.Base` 等）在 Go 與輸入共用同一個底圖切片；Rust 存一份
//!   `Vec<u8>` 複本，內容相同。
//! - Go 的 func 欄位（`ResolveTarget`）→ `Option<&'a mut dyn FnMut(..)>`，所以含它的輸入結構以
//!   `&mut` 傳入。
//! - 錯誤用 `fd2_rules::battle::BattleError`（只保存訊息，`Display` 與 Go `err.Error()` 逐字相同）；
//!   `fd2_rules` 沒有 `From<FiganiError>`，所以下層錯誤經 [`GoResult`] 轉成訊息（`%w` 接在前綴後）。

mod native_command0;
mod native_command0_actor;
mod native_command1;
mod native_command2;
mod native_command24_background;
mod native_command24_prelude;
mod native_command3;
mod native_command32_tail;
mod native_command3_sequence;
mod native_command5;
mod native_command5_sequence;
mod native_command6;
mod native_command6_sequence;
mod native_command7;
mod native_command7_sequence;
mod native_command8;
mod native_command8_sequence;
mod native_command9;
mod native_command_compound;
mod native_command_sources;
mod native_physical_body;

pub use native_command_compound::*;
pub use native_command_sources::*;
pub use native_command0::*;
pub use native_command0_actor::*;
pub use native_command1::*;
pub use native_command2::*;
pub use native_command3::*;
pub use native_command3_sequence::*;
pub use native_command5::*;
pub use native_command5_sequence::*;
pub use native_command6::*;
pub use native_command6_sequence::*;
pub use native_command7::*;
pub use native_command7_sequence::*;
pub use native_command8::*;
pub use native_command8_sequence::*;
pub use native_command9::*;
pub use native_command24_background::*;
pub use native_command24_prelude::*;
pub use native_command32_tail::*;
pub use native_physical_body::*;

pub use fd2_rules::battle::BattleError as Error;

/// `battlepresent` 的結果型別。
pub type Result<T, E = Error> = std::result::Result<T, E>;

/// Go `errors.New(msg)`。
fn err(message: impl Into<String>) -> Error {
    Error::new(message)
}

/// 下層（`figani`、`fdother`、`fdicon`、`indexedmap`、`battle`）錯誤轉成本套件錯誤。
///
/// - `go()`：Go 直接回傳下層 `err`，訊息原樣保留。
/// - `wrap(prefix)`：Go `fmt.Errorf("<prefix>%w", err)`，`prefix` 含結尾的 `": "`。
trait GoResult<T> {
    fn go(self) -> Result<T>;
    fn wrap(self, prefix: impl std::fmt::Display) -> Result<T>;
}

impl<T, E: std::fmt::Display> GoResult<T> for std::result::Result<T, E> {
    fn go(self) -> Result<T> {
        self.map_err(|e| Error::new(e.to_string()))
    }

    fn wrap(self, prefix: impl std::fmt::Display) -> Result<T> {
        self.map_err(|e| Error::new(format!("{prefix}{e}")))
    }
}

/// Go `fmt` 的 `%#x`：負數寫成 `-0x..`。
fn go_sharp_x(v: i64) -> String {
    if v < 0 { format!("-{:#x}", v.unsigned_abs()) } else { format!("{v:#x}") }
}

/// Go `len(s)` 轉成 `int`。
fn len64<T>(s: &[T]) -> i64 {
    s.len() as i64
}

/// 已檢查非負的 Go `int` 偏移轉成索引；負值轉成極大值，之後切片時 panic，與 Go 的越界 panic 對應。
fn at(i: i64) -> usize {
    i as usize
}

#[cfg(test)]
mod test_util;
