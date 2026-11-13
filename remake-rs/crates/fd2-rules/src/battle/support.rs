//! 單位索引、callback 包裝與共用的 serde 輔助。沒有對應的 Go 檔；設計說明見 `battle` 模組文件。

use std::sync::Arc;

use super::*;

/// 讓各子模組經由 `use super::*;` 取得 `battle_err!`（`#[macro_export]` 的巨集只在 crate 根）。
pub(crate) use crate::battle_err;

/// 單位參照：`State::units` 裡的位置（自由函式則是該函式 `units` 參數裡的位置）。
///
/// Go 版以 `*Unit` 指標表示；`units` 在戰鬥中只 append 不刪除，所以索引在整場戰鬥中穩定。
pub type UnitIdx = usize;

/// Go func 欄位與可改寫全域 func 變數的包裝。
///
/// Go 的 func 值可以複製，也可以在多個結構間共用，所以用 `Arc`；`Clone` 只複製參照，與 Go 相同。
/// Go 的 func 不能比較相等，含 callback 的結構因此不 derive `PartialEq`。
pub struct Callback<F: ?Sized>(pub Arc<F>);

impl<F: ?Sized> Callback<F> {
    /// 由 `Arc` 包好的閉包建立。
    pub fn new(f: Arc<F>) -> Self {
        Self(f)
    }
}

impl<F: ?Sized> Clone for Callback<F> {
    fn clone(&self) -> Self {
        Self(Arc::clone(&self.0))
    }
}

impl<F: ?Sized> fmt::Debug for Callback<F> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str("Callback(..)")
    }
}

impl<F: ?Sized> std::ops::Deref for Callback<F> {
    type Target = F;

    fn deref(&self) -> &F {
        &self.0
    }
}

/// serde 輔助：Go `[]byte` 欄位（PORTING.md「開工前的決定」第 2 點）。
///
/// Go `encoding/json` 把 `[]byte` 寫成標準 base64 字串（nil 寫成 `null`）；讀取時接受 base64 字串、
/// 數字陣列（每個元素 0..=255）與 `null`。重製端資產多半由 Python 工具寫成數字陣列，Go 寫出的快照
/// 則是 base64，兩種都要讀。用法：`#[serde(default, with = "go_bytes")]` 用在 `Vec<u8>`，
/// `#[serde(default, with = "go_bytes::option")]` 用在 `Option<Vec<u8>>`。
///
/// Rust 的 `Vec<u8>` 不分 nil 與空切片：`go_bytes` 把空 `Vec` 寫成 `null`（Go 零值 nil 的輸出）；
/// 需要區分 `null` 與 `""` 的欄位改用 `go_bytes::option`（`None` → `null`，`Some` → base64）。
/// 損壞輸入只比接受或拒收，錯誤文字不比（`rust-port-no-go-stdlib`）。
pub mod go_bytes {
    use base64::Engine as _;
    use serde::de::{self, SeqAccess, Visitor};
    use serde::{Deserializer, Serializer};
    use std::fmt;

    struct GoBytesVisitor;

    impl<'de> Visitor<'de> for GoBytesVisitor {
        type Value = Option<Vec<u8>>;

        fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
            f.write_str("base64 字串、0..=255 的數字陣列或 null")
        }

        fn visit_none<E: de::Error>(self) -> Result<Self::Value, E> {
            Ok(None)
        }

        fn visit_unit<E: de::Error>(self) -> Result<Self::Value, E> {
            Ok(None)
        }

        fn visit_some<D: Deserializer<'de>>(self, d: D) -> Result<Self::Value, D::Error> {
            d.deserialize_any(self)
        }

        fn visit_str<E: de::Error>(self, v: &str) -> Result<Self::Value, E> {
            base64::engine::general_purpose::STANDARD.decode(v).map(Some).map_err(E::custom)
        }

        fn visit_seq<A: SeqAccess<'de>>(self, mut seq: A) -> Result<Self::Value, A::Error> {
            let mut out = Vec::new();
            while let Some(b) = seq.next_element::<u8>()? {
                out.push(b);
            }
            Ok(Some(out))
        }
    }

    /// 讀取；`null` 讀成空 `Vec`。
    pub fn deserialize<'de, D: Deserializer<'de>>(d: D) -> Result<Vec<u8>, D::Error> {
        Ok(d.deserialize_option(GoBytesVisitor)?.unwrap_or_default())
    }

    /// 寫出；空 `Vec` 寫成 `null`，其餘寫成標準 base64。
    pub fn serialize<S: Serializer>(v: &[u8], s: S) -> Result<S::Ok, S::Error> {
        if v.is_empty() {
            s.serialize_none()
        } else {
            s.serialize_str(&base64::engine::general_purpose::STANDARD.encode(v))
        }
    }

    /// `Option<Vec<u8>>` 版本：保留 `null`（`None`）與 `""`（`Some(空)`）的差別。
    pub mod option {
        use super::*;

        /// 讀取；`null` 讀成 `None`。
        pub fn deserialize<'de, D: Deserializer<'de>>(d: D) -> Result<Option<Vec<u8>>, D::Error> {
            d.deserialize_option(GoBytesVisitor)
        }

        /// 寫出；`None` 寫成 `null`，`Some` 一律寫成 base64（空切片是 `""`）。
        pub fn serialize<S: Serializer>(v: &Option<Vec<u8>>, s: S) -> Result<S::Ok, S::Error> {
            match v {
                None => s.serialize_none(),
                Some(b) => s.serialize_str(&base64::engine::general_purpose::STANDARD.encode(b)),
            }
        }
    }
}

/// Go `fmt` 的 `%#x`：負數寫成 `-0x..`（Rust 的 `{:#x}` 對負數輸出二補數）。
pub(crate) fn go_sharp_x(v: i64) -> String {
    if v < 0 { format!("-{:#x}", v.unsigned_abs()) } else { format!("{v:#x}") }
}

/// Go `fmt` 的 `%q`（`strconv.Quote`）：雙引號包住，跳脫 `"`、`\` 與控制字元，可列印的 Unicode
/// 原樣保留。只涵蓋 `battle` 錯誤訊息用到的字串；Go 對不可列印 Unicode 的判定表不移植。
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

/// Go `os.ReadFile`。錯誤文字不與 Go 逐字相同（`os` 錯誤屬 Go 標準函式庫，見 `KNOWN_DIFFERENCES`），
/// 呼叫端只看成功或失敗。
pub(crate) fn read_file(path: &str) -> Result<Vec<u8>> {
    std::fs::read(path).map_err(|e| BattleError::new(format!("open {path}: {e}")))
}

/// Go `json.Unmarshal`（以 serde_json 實作）。錯誤文字不與 Go 逐字相同，只比接受或拒收。
pub(crate) fn decode_json<T: serde::de::DeserializeOwned>(raw: &[u8]) -> Result<T> {
    serde_json::from_slice(raw).map_err(|e| BattleError::new(e.to_string()))
}

/// Go `hex.DecodeString` 的成功值：大小寫十六進位都接受，奇數長度或非十六進位字元回 `None`。
pub(crate) fn decode_hex(s: &str) -> Option<Vec<u8>> {
    let b = s.as_bytes();
    if !b.len().is_multiple_of(2) {
        return None;
    }
    let nibble = |c: u8| (c as char).to_digit(16).map(|d| d as u8);
    b.chunks(2).map(|p| Some(nibble(p[0])? << 4 | nibble(p[1])?)).collect()
}

/// serde 輔助：Go `omitempty` 對數值欄位的判定（零值省略）。
/// 用法：`#[serde(skip_serializing_if = "go_is_zero")]`。
pub fn go_is_zero<T: Default + PartialEq>(v: &T) -> bool {
    *v == T::default()
}

/// serde 輔助：Go 讀 JSON `null` 到切片、map 或字串時留下零值；Rust 讀成 `Default`。
///
/// 用法：`#[serde(default, deserialize_with = "null_as_default")]`。
pub fn null_as_default<'de, D, T>(d: D) -> std::result::Result<T, D::Error>
where
    D: serde::Deserializer<'de>,
    T: serde::Deserialize<'de> + Default,
{
    use serde::Deserialize as _;
    Ok(Option::<T>::deserialize(d)?.unwrap_or_default())
}
