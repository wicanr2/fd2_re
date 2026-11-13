//! Go 來源：`remake/internal/battle/native_command_palette_flash.go`（76 行）。

use super::*;

/// Go 原式：`NativeCommandRecordCount`
pub const NATIVE_COMMAND_PALETTE_FLASH_ENTRIES: i64 = 36;

/// NativeCommandPaletteFlashTable preserves the three command-indexed six-bit
/// DAC tables read by FD2.EXE 0x1D6C8. The high-level command/status names are
/// deliberately outside this raw presentation contract.
///
/// Go：`native_command_palette_flash.go:14`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeCommandPaletteFlashTable {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"cycles"`
    pub cycles: i64,
    /// Go 標籤：`json:"ticks_per_phase"`
    pub ticks_per_phase: i64,
    /// Go 標籤：`json:"sfx_resource"`
    pub sfx_resource: i64,
    /// Go 標籤：`json:"sfx_index"`
    pub sfx_index: i64,
    /// Go 標籤：`json:"entries"`
    #[serde(deserialize_with = "null_as_default")]
    pub entries: Vec<[u8; 3]>,
    /// Go 標籤：`json:"source"`
    pub source: NativeCommandFlashSource,
}

/// Go：`native_command_palette_flash.go:24`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeCommandFlashSource {
    /// Go 標籤：`json:"file"`
    pub file: String,
    /// Go 標籤：`json:"size"`
    pub size: i64,
    /// Go 標籤：`json:"md5"`
    pub md5: String,
    /// Go 標籤：`json:"sha256"`
    pub sha256: String,
    /// Go 標籤：`json:"ida_version"`
    pub ida_version: String,
    /// Go 標籤：`json:"address_space"`
    pub address_space: String,
    /// Go 標籤：`json:"red_table"`
    pub red_table: String,
    /// Go 標籤：`json:"green_table"`
    pub green_table: String,
    /// Go 標籤：`json:"blue_table"`
    pub blue_table: String,
}

/// Go：`native_command_palette_flash.go:36`，`func LoadNativeCommandPaletteFlashTable(path string) (*NativeCommandPaletteFlashTable, error)`
pub fn load_native_command_palette_flash_table(path: &str) -> Result<NativeCommandPaletteFlashTable> {
    let raw = read_file(path)?;
    let table: NativeCommandPaletteFlashTable =
        decode_json::<Option<NativeCommandPaletteFlashTable>>(&raw)?.unwrap_or_default();
    if table.schema_version != 1
        || table.cycles != 4
        || table.ticks_per_phase != 1
        || table.sfx_resource != 88
        || table.sfx_index != 0
        || table.entries.len() as i64 != NATIVE_COMMAND_PALETTE_FLASH_ENTRIES
    {
        return Err(battle_err!("native command palette flash metadata is invalid"));
    }
    if table.source.file != "FD2.EXE"
        || table.source.size != 357074
        || table.source.md5 != "b97caf2239a27a896069d03549d96e1e"
        || table.source.sha256 != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
        || table.source.address_space != "IDA linear, image base 0"
    {
        return Err(battle_err!("native command palette flash source identity is invalid"));
    }
    for (command_id, rgb) in table.entries.iter().enumerate() {
        if rgb.iter().any(|&component| component > 0x3f) {
            return Err(battle_err!("native command palette flash command {command_id} has non-DAC component"));
        }
    }
    Ok(table)
}

impl NativeCommandPaletteFlashTable {
    /// NativeCommandPaletteFlashPhases returns the exact color/black phase order
    /// for one command. Each entry is one 0x17AA9(1) boundary in the original.
    ///
    /// Go：`native_command_palette_flash.go:67`，`func (t *NativeCommandPaletteFlashTable) NativeCommandPaletteFlashPhases(commandID int) ([][3]byte, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_command_palette_flash_phases(&self, command_id: i64) -> Result<Vec<[u8; 3]>> {
        if command_id < 0 || command_id >= self.entries.len() as i64 || self.cycles != 4 || self.ticks_per_phase != 1 {
            return Err(battle_err!("native command palette flash unavailable id={command_id}"));
        }
        let mut phases = Vec::with_capacity((self.cycles * 2) as usize);
        for _ in 0..self.cycles {
            phases.push(self.entries[command_id as usize]);
            phases.push([0; 3]);
        }
        Ok(phases)
    }
}

#[cfg(test)]
mod native_command_palette_flash_test;
