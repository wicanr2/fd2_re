//! 對應凍結 Go 版 `remake/internal/localization`：讀取並驗證可編輯的 FD2 語系包。
//!
//! Go 套件註解：Package localization loads editable FD2 locale packs.
//!
//! 每個 Go 檔 `x.go` 對應子模組 `localization/x.rs`，測試放在 `localization/x/<t>_test.rs`。
//!
//! # 對應決定
//!
//! - JSON 用 serde：`DisallowUnknownFields` → `deny_unknown_fields`（Go 對巢狀結構同樣拒收）；Go 讀 `null`
//!   或缺欄位時留零值，Rust 每個欄位都用 `default` 加 `null_as_default`，結果相同。Go 解碼的錯誤文字
//!   屬標準函式庫，Rust 只保留本套件自己的前綴（例如 `decode locale pack "<路徑>": `）。
//! - Go 的 `regexp` 樣式改成逐字元比對的函式（`patterns.rs`），每個函式的 doc 寫出原本的樣式；
//!   樣式都是錨定的整串比對，沒有需要回溯的分支。
//! - `filepath.Join` 會做 `filepath.Clean`；錯誤訊息裡的路徑照 Go 的字面清理結果（[`go_join`]）。
//! - Go `map` → `BTreeMap`。`entities.json` 的驗證在 Go 以 map 的隨機順序走訪，有多筆不合法時回報哪一筆
//!   由順序決定；Rust 依字串鍵排序走訪，只是 Go 可能結果之一。
//! - `Catalog.Format(key, args ...any)` 的參數改成 [`FormatArg`] 切片；`fmt.Sprintf` 只實作語系文字會用到
//!   的動詞子集（見 [`go_sprintf`]），不移植 Go `fmt`。
//! - 錯誤用 `fd2_rules::battle::BattleError`（只保存訊息，`Display` 與 Go `err.Error()` 逐字相同）。

mod content;
mod entities;
mod locale;
mod patterns;

pub use content::*;
pub use entities::*;
pub use locale::*;

pub use fd2_rules::battle::BattleError as Error;

/// `localization` 的結果型別。
pub type Result<T, E = Error> = std::result::Result<T, E>;

fn err(message: impl Into<String>) -> Error {
    Error::new(message)
}

/// Go `strconv.Quote`（`%q`）：雙引號包住，跳脫 `"`、`\` 與控制字元，其餘 Unicode 原樣保留。
/// Go 對不可列印 Unicode 的判定表不移植；語系路徑與鍵名都是可列印字元。
pub(crate) fn go_quote(s: &str) -> String {
    let mut out = String::with_capacity(s.len() + 2);
    out.push('"');
    for c in s.chars() {
        match c {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\x07' => out.push_str("\\a"),
            '\x08' => out.push_str("\\b"),
            '\x0c' => out.push_str("\\f"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '\x0b' => out.push_str("\\v"),
            c if (c as u32) < 0x20 || c as u32 == 0x7f => out.push_str(&format!("\\x{:02x}", c as u32)),
            c => out.push(c),
        }
    }
    out.push('"');
    out
}

/// Go `filepath.Clean` 的字面清理（以 `/` 為分隔；Go 在 Windows 也得到相同的接受／拒收結果）：去掉
/// 空段與 `.`，`..` 吃掉前一個非 `..` 段，根目錄之上的 `..` 丟掉，空結果是 `.`。
pub(crate) fn go_clean(path: &str) -> String {
    let rooted = path.starts_with('/');
    let mut parts: Vec<&str> = Vec::new();
    for seg in path.split('/') {
        match seg {
            "" | "." => {}
            ".." => match parts.last() {
                Some(&last) if last != ".." => {
                    parts.pop();
                }
                _ if rooted => {}
                _ => parts.push(".."),
            },
            seg => parts.push(seg),
        }
    }
    let joined = parts.join("/");
    match (rooted, joined.is_empty()) {
        (true, _) => format!("/{joined}"),
        (false, true) => ".".to_string(),
        (false, false) => joined,
    }
}

/// Go `filepath.Join`：略過空字串，以 `/` 接起來再 [`go_clean`]；全部是空字串時回空字串。
pub(crate) fn go_join(parts: &[&str]) -> String {
    let parts: Vec<&str> = parts.iter().copied().filter(|p| !p.is_empty()).collect();
    if parts.is_empty() {
        return String::new();
    }
    go_clean(&parts.join("/"))
}

/// 嚴格 JSON 解碼的失敗種類（Go `json.Decoder` 配 `DisallowUnknownFields`，再讀一次確認沒有第二個值）。
pub(crate) enum StrictDecodeError {
    /// 第一個值解碼失敗；內容是 serde_json 的錯誤文字（Go 是 `encoding/json` 的文字，只比接受或拒收）。
    Decode(String),
    /// 後面還有一個完整的 JSON 值（Go `trailing JSON`）。
    Trailing,
    /// 後面有無法解析的位元組（Go `trailing JSON: <err>`）；內容是 serde_json 的錯誤文字。
    TrailingInvalid(String),
}

/// Go `dec.Decode(&v)` 之後再 `dec.Decode(&extra)` 要求 `io.EOF` 的組合：只有空白結尾才接受。
pub(crate) fn decode_strict<T: serde::de::DeserializeOwned>(raw: &[u8]) -> Result<T, StrictDecodeError> {
    let mut stream = serde_json::Deserializer::from_slice(raw).into_iter::<T>();
    let value = match stream.next() {
        Some(Ok(value)) => value,
        Some(Err(e)) => return Err(StrictDecodeError::Decode(e.to_string())),
        None => return Err(StrictDecodeError::Decode("EOF".to_string())),
    };
    let rest = &raw[stream.byte_offset()..];
    match serde_json::Deserializer::from_slice(rest).into_iter::<serde_json::Value>().next() {
        None => Ok(value),
        Some(Ok(_)) => Err(StrictDecodeError::Trailing),
        Some(Err(e)) => Err(StrictDecodeError::TrailingInvalid(e.to_string())),
    }
}

/// 依 Go 的三種文字回報嚴格解碼失敗：`<prefix>: <err>`、`<prefix>: trailing JSON`、
/// `<prefix>: trailing JSON: <err>`。`prefix` 例如 `decode locale pack "<路徑>"`。
pub(crate) fn strict_decode_error(prefix: &str, e: StrictDecodeError) -> Error {
    match e {
        StrictDecodeError::Decode(detail) => err(format!("{prefix}: {detail}")),
        StrictDecodeError::Trailing => err(format!("{prefix}: trailing JSON")),
        StrictDecodeError::TrailingInvalid(detail) => err(format!("{prefix}: trailing JSON: {detail}")),
    }
}
