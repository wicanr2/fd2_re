//! Go 來源：`remake/internal/battle/native_command_scene.go`（76 行）。

use super::*;

pub const NATIVE_COMMAND_SCENE_CHAPTER_COUNT: i64 = 30;

/// NativeCommandSceneTable preserves the command-presentation background
/// initializer at 0x52363. It remains editable data but is accepted only when
/// bound to the fixed reference executable and complete chapter range.
///
/// Go：`native_command_scene.go:15`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeCommandSceneTable {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"chapter_initial_background"`
    #[serde(with = "go_bytes")]
    pub chapter_initial_background: Vec<u8>,
    /// Go 標籤：`json:"source"`
    pub source: NativeCommandSceneSource,
}

/// Go：`native_command_scene.go:21`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeCommandSceneSource {
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
    /// Go 標籤：`json:"initial_selector_table"`
    pub initial_selector_table: String,
    /// Go 標籤：`json:"raw_gate"`
    pub raw_gate: String,
}

/// Go：`native_command_scene.go:32`，`func LoadNativeCommandSceneTable(path string) (*NativeCommandSceneTable, error)`
pub fn load_native_command_scene_table(path: &str) -> Result<NativeCommandSceneTable> {
    let raw = read_file(path)?;
    // Go 讀 JSON `null` 得到零值表，之後由 metadata 檢查拒收；Rust 同樣先讀成預設值。
    let table: NativeCommandSceneTable = decode_json::<Option<NativeCommandSceneTable>>(&raw)?.unwrap_or_default();
    if table.schema_version != 1
        || table.chapter_initial_background.len() as i64 != NATIVE_COMMAND_SCENE_CHAPTER_COUNT
        || table.source.file != "FD2.EXE"
        || table.source.size != 357074
        || table.source.md5 != "b97caf2239a27a896069d03549d96e1e"
        || table.source.sha256 != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
        || table.source.address_space != "IDA linear, image base 0"
        || table.source.initial_selector_table != "0x52363..0x52380"
        || table.source.raw_gate != "sub_1F183 0x1F183..0x1F1BE"
    {
        return Err(battle_err!("native command scene metadata is invalid"));
    }
    for (chapter, &selector) in table.chapter_initial_background.iter().enumerate() {
        if selector > 3 {
            return Err(battle_err!("native command scene chapter {chapter} selector is invalid"));
        }
    }
    Ok(table)
}

impl NativeCommandSceneTable {
    /// Go：`native_command_scene.go:58`，`func (t *NativeCommandSceneTable) InitialBackground(chapter int) (byte, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn initial_background(&self, chapter: i64) -> Result<u8> {
        if chapter < 0 || chapter >= self.chapter_initial_background.len() as i64 {
            return Err(battle_err!("native command scene chapter unavailable: {chapter}"));
        }
        Ok(self.chapter_initial_background[chapter as usize])
    }
}

/// NativeCommandBackgroundGate reproduces sub_1F183 from raw runtime fields.
/// It deliberately requires independent +7/+0x1f/+0x20 provenance and does
/// not substitute normalized class labels or map selectors.
///
/// Go：`native_command_scene.go:68`，`func NativeCommandBackgroundGate(unit *Unit) (bool, error)`
pub fn native_command_background_gate(unit: Option<&Unit>) -> Result<bool> {
    let Some(unit) = unit.filter(|u| u.has_battle_fig && u.has_native_record_race && u.has_native_record_class) else {
        return Err(battle_err!("native command background gate raw record unavailable"));
    };
    if unit.battle_fig == 28 {
        return Ok(false);
    }
    Ok(unit.native_record_class == 19 || unit.native_record_race == 4 || unit.native_record_race == 5)
}

#[cfg(test)]
mod native_command_scene_test;
