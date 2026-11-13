//! `battle` 共用的錯誤型別。沒有對應的 Go 檔：Go 版各檔直接用 `fmt.Errorf`／`errors.New`。
//!
//! Go 版 `battle` 沒有哨兵錯誤，也沒有任何地方用 `errors.Is`／`errors.As` 判斷錯誤種類
//! （2026-11-14 以 `grep` 確認 `remake/internal/battle` 與 `remake/cmd`），呼叫端與測試只看
//! 錯誤是否存在與 `Error()` 文字。所以 Rust 版只保存最終訊息：`Display` 必須與 Go 的錯誤字串
//! 逐字相同，`%w` 包裝在 Rust 以 [`BattleError::wrap`] 串接前綴。

use super::*;

/// Go `error` 的 Rust 對應。訊息即 Go `err.Error()` 的完整文字。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct BattleError {
    message: String,
}

impl BattleError {
    /// 以完整訊息建立錯誤，對應 Go `errors.New(msg)` 或已格式化的 `fmt.Errorf`。
    pub fn new(message: impl Into<String>) -> Self {
        Self { message: message.into() }
    }

    /// 對應 Go `fmt.Errorf("<prefix>%w", err)`：回傳 `prefix` 接上原訊息的新錯誤。
    pub fn wrap(self, prefix: impl fmt::Display) -> Self {
        Self { message: format!("{prefix}{}", self.message) }
    }

    /// Go `err.Error()`。
    pub fn message(&self) -> &str {
        &self.message
    }
}

impl fmt::Display for BattleError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.message)
    }
}

impl std::error::Error for BattleError {}

/// `battle` 的結果型別；`(T, error)` 對應 `Result<T>`。
pub type Result<T, E = BattleError> = std::result::Result<T, E>;

/// 以 `format!` 語法建立 [`BattleError`]，對應不含 `%w` 的 `fmt.Errorf`。
///
/// Go 的 `%d`、`%v`、`%s` 對整數與字串和 Rust `{}` 相同；`%q`、`%x`、`%#x`、`%02x` 與結構的
/// `%v` 要逐處對照 Go 的輸出格式，不能直接換成 `{:?}`。
#[macro_export]
macro_rules! battle_err {
    ($($arg:tt)*) => {
        $crate::battle::BattleError::new(format!($($arg)*))
    };
}

macro_rules! from_format_error {
    ($($t:ty),* $(,)?) => {
        $(
            impl From<$t> for BattleError {
                /// 格式層錯誤原樣轉成訊息；對應 Go 直接回傳下層 `error`。
                fn from(err: $t) -> Self {
                    Self::new(err.to_string())
                }
            }
        )*
    };
}

from_format_error!(
    fd2_formats::archive::ArchiveError,
    fd2_formats::dato::DatoError,
    fd2_formats::fdicon::FdiconError,
    fd2_formats::fdother::FdotherError,
    fd2_formats::fdother::NativeFlowError,
    fd2_formats::fdtxt::FdtxtError,
);
