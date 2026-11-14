//! 對應凍結 Go 版 `remake/internal/indexedmap`：組合已驗證的原版戰術地圖穩態繪製步驟。
//!
//! Go 套件註解：Package indexedmap composes the verified steady native tactical-map passes. It
//! intentionally has no Ebiten dependency: callers supply indexed buffers and the still-separate
//! native HUD pass.
//!
//! 每個 Go 檔 `x.go` 對應子模組 `indexedmap/x.rs`，測試放在 `indexedmap/x/<t>_test.rs`。
//!
//! # 型別對應
//!
//! - Go `int` → `i64`；位址與偏移的乘加照 Go 以 64 位元計算，檢查過範圍才轉成 `usize` 索引。
//! - Go 輸入結構裡的 `*fdicon.Bank`、`*fdicon.NativeSelectorCache` 等指標 → `Option<&'a T>`（呼叫端
//!   擁有資源，nil 檢查照 Go 保留）；`[]byte` 輸入 → `&'a [u8]`；單位清單與格子 → `Vec`。
//! - `FrameInput.Cells` 在 Go 與呼叫端共用底層陣列，`ComposeFrame` 成功時把 mode 6 改寫的格子寫回
//!   那個陣列。Rust 的 [`FrameInput::cells`] 是擁有值，[`compose_frame`] 收 `&mut FrameInput`，成功時
//!   寫回 `in_.cells`；呼叫端從同一個欄位讀回結果。
//! - Go func 參數 → `Option<&mut dyn FnMut(..)>`（nil 檢查照 Go 保留）；`FrameInput.SelectionOverlay`
//!   這種結構欄位 → `Option<&'a dyn Fn(..)>`，需要改寫外部狀態時由呼叫端用 `Cell`／`RefCell`。
//! - 錯誤用 `fd2_rules::battle::BattleError`（只保存訊息，`Display` 與 Go `err.Error()` 逐字相同）；
//!   `%w` 包裝格式層錯誤時串接 Go 前綴與格式層訊息。
//! - Go 切片越界會 panic 的地方（例如工作緩衝短於 `0x8088` 時切 `frame[workBase:]`），Rust 照樣
//!   panic，不另加檢查。

mod frame;
mod hud;
mod mosaic;
mod native_command10_12;
mod native_command_heal_tail;
mod spawn_intro;

pub use frame::*;
pub use hud::*;
pub use mosaic::*;
pub use native_command_heal_tail::*;
pub use native_command10_12::*;
pub use spawn_intro::*;

pub use fd2_rules::battle::BattleError as Error;

/// `indexedmap` 的結果型別。
pub type Result<T, E = Error> = std::result::Result<T, E>;

/// Go `errors.New(msg)`。
fn err(message: impl Into<String>) -> Error {
    Error::new(message)
}

/// Go `fmt.Errorf("<prefix>%w", e)`：格式層或本套件的錯誤接在前綴之後。
fn wrap<E: Into<Error>>(prefix: impl std::fmt::Display) -> impl FnOnce(E) -> Error {
    move |e| e.into().wrap(prefix)
}

/// 本套件的錯誤交給 `fdother` 的 callback 時轉成 `FdotherError`；訊息原樣保留，`fdother` 原樣傳回時
/// 文字與 Go 相同（Go 兩邊都是同一個 `error` 值）。
fn to_fdother(e: Error) -> fd2_formats::fdother::FdotherError {
    fd2_formats::fdother::FdotherError::Contract(e.message().to_string())
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
