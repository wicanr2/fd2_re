//! Go 來源：`remake/internal/battle/native_item_table.go`（49 行）。

use super::*;

/// Go：`native_item_table.go:10`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Deserialize)]
#[serde(default)]
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
    let data = read_file(path).map_err(|e| e.wrap("read native item effect rows: "))?;
    let encoded: Vec<NativeItemEffectRowJSON> = decode_json::<Option<Vec<NativeItemEffectRowJSON>>>(&data)
        .map_err(|e| e.wrap("decode native item effect rows: "))?
        .unwrap_or_default();
    if encoded.is_empty() || encoded.len() > 0x100 {
        return Err(battle_err!("native item effect row count {} is invalid", encoded.len()));
    }
    let size = NATIVE_ITEM_EFFECT_ROW_SIZE as usize;
    let mut table = vec![0u8; encoded.len() * size];
    for (i, row) in encoded.iter().enumerate() {
        if row.id != i as i64 {
            return Err(battle_err!("native item effect row {i} has id {}", row.id));
        }
        // Go `hex.DecodeString` 的錯誤文字屬標準函式庫，Rust 只保留 Go 自己的前綴。
        let Some(raw) = decode_hex(&row.raw) else {
            return Err(battle_err!("decode native item effect row {i}: invalid hex"));
        };
        if raw.len() != size {
            return Err(battle_err!("native item effect row {i} has {} bytes, want {size}", raw.len()));
        }
        table[i * size..(i + 1) * size].copy_from_slice(&raw);
    }
    Ok(table)
}

#[cfg(test)]
mod native_item_table_test;
