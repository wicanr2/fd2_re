//! Go 來源：`remake/internal/battle/command_learn.go`（106 行）。

use super::*;

/// CommandLearnEntry is one exact (required level, command ID) pair from the
/// 12-byte native learning row selected by growth-row learn_idx.
///
/// Go：`command_learn.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct CommandLearnEntry {
    /// Go 標籤：`json:"required_level"`
    pub required_level: i64,
    /// Go 標籤：`json:"command_id"`
    pub command_id: i64,
}

/// Go：`command_learn.go:16`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct CommandLearnFileRow {
    /// Go 標籤：`json:"idx"`
    pub idx: i64,
    /// Go 標籤：`json:"entries"`
    pub entries: Vec<CommandLearnEntry>,
}

/// Go：`command_learn.go:21`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct CommandLearnSelectorRow {
    /// Go 標籤：`json:"idx"`
    pub idx: i64,
    /// Go 標籤：`json:"learn_idx"`
    pub learn_idx: i64,
}

/// LoadCommandLearn reads the editable export of 0x626b3 + learn_idx*12.
/// Rows must be dense and pairs strictly increasing by level so malformed
/// authored data cannot silently grant a different native command.
///
/// Go：`command_learn.go:29`，`func LoadCommandLearn(path string) (map[int][]CommandLearnEntry, error)`
pub fn load_command_learn(path: &str) -> Result<BTreeMap<i64, Vec<CommandLearnEntry>>> {
    todo!("battle/command_learn.go:LoadCommandLearn")
}

/// LoadCommandLearnSelectors reads the 11-byte 0x4E4D1 growth records and
/// preserves their final learn_idx byte. Native level-up does not index the
/// command table with portrait: it first resolves unit+7 through this table,
/// then uses byte10 as the 0x4E4A2 command-learn row. 0xFF means no row.
///
/// Go：`command_learn.go:62`，`func LoadCommandLearnSelectors(path string) (map[int]int, error)`
pub fn load_command_learn_selectors(path: &str) -> Result<BTreeMap<i64, i64>> {
    todo!("battle/command_learn.go:LoadCommandLearnSelectors")
}

impl State {
    /// Go：`command_learn.go:86`，`func (s *State) learnNativeCommandsAtLevel(u *Unit) []int`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn learn_native_commands_at_level(&mut self, u: Option<UnitIdx>) -> Vec<i64> {
        todo!("battle/command_learn.go:State.learnNativeCommandsAtLevel")
    }
}
