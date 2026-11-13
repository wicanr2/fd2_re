//! Go 來源：`remake/internal/battle/native_command_scene.go`（76 行）。

use super::*;

pub const NATIVE_COMMAND_SCENE_CHAPTER_COUNT: i64 = 30;

/// NativeCommandSceneTable preserves the command-presentation background
/// initializer at 0x52363. It remains editable data but is accepted only when
/// bound to the fixed reference executable and complete chapter range.
///
/// Go：`native_command_scene.go:15`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommandSceneTable {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"chapter_initial_background"`
    pub chapter_initial_background: Vec<u8>,
    /// Go 標籤：`json:"source"`
    pub source: NativeCommandSceneSource,
}

/// Go：`native_command_scene.go:21`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
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
    todo!("battle/native_command_scene.go:LoadNativeCommandSceneTable")
}

impl NativeCommandSceneTable {
    /// Go：`native_command_scene.go:58`，`func (t *NativeCommandSceneTable) InitialBackground(chapter int) (byte, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn initial_background(&self, chapter: i64) -> Result<u8> {
        todo!("battle/native_command_scene.go:NativeCommandSceneTable.InitialBackground")
    }
}

/// NativeCommandBackgroundGate reproduces sub_1F183 from raw runtime fields.
/// It deliberately requires independent +7/+0x1f/+0x20 provenance and does
/// not substitute normalized class labels or map selectors.
///
/// Go：`native_command_scene.go:68`，`func NativeCommandBackgroundGate(unit *Unit) (bool, error)`
pub fn native_command_background_gate(unit: Option<&Unit>) -> Result<bool> {
    todo!("battle/native_command_scene.go:NativeCommandBackgroundGate")
}
