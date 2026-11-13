//! Go 來源：`remake/internal/battle/native_item_table.go`（49 行）。

use super::*;

/// Go：`native_item_table.go:10`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeItemEffectRowJSON {
    /// Go 標籤：`json:"id"`
    pub id: i64,
    /// Go 標籤：`json:"raw"`
    pub raw: String,
}

/// LoadNativeItemEffectRowPrefix loads an explicitly bounded prefix of the raw
/// 0x4e56c table. Rows must be consecutive and exactly 0x17 bytes. The function
/// deliberately does not infer that the fixture's final ID is the native table
/// boundary.
///
/// Go：`native_item_table.go:19`，`func LoadNativeItemEffectRowPrefix(path string) ([]byte, error)`
pub fn load_native_item_effect_row_prefix(path: &str) -> Result<Vec<u8>> {
    todo!("battle/native_item_table.go:LoadNativeItemEffectRowPrefix")
}
