//! Go 來源：`remake/internal/battle/command_learn.go`（106 行）。

use super::*;

/// CommandLearnEntry is one exact (required level, command ID) pair from the
/// 12-byte native learning row selected by growth-row learn_idx.
///
/// Go：`command_learn.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct CommandLearnEntry {
    /// Go 標籤：`json:"required_level"`
    pub required_level: i64,
    /// Go 標籤：`json:"command_id"`
    pub command_id: i64,
}

/// Go：`command_learn.go:16`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub(crate) struct CommandLearnFileRow {
    /// Go 標籤：`json:"idx"`
    pub idx: i64,
    /// Go 標籤：`json:"entries"`
    #[serde(deserialize_with = "null_as_default")]
    pub entries: Vec<CommandLearnEntry>,
}

/// Go：`command_learn.go:21`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
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
    let raw = read_file(path)?;
    let rows: Vec<CommandLearnFileRow> = decode_json(&raw)?;
    if rows.is_empty() {
        return Err(battle_err!("command learn: empty table"));
    }
    let mut out = BTreeMap::new();
    for (i, row) in rows.iter().enumerate() {
        if row.idx != i as i64 || row.entries.len() > 6 {
            return Err(battle_err!("command learn: invalid row {i}"));
        }
        let mut previous = 0;
        for entry in &row.entries {
            if entry.required_level <= previous || entry.command_id < 0 || entry.command_id >= 40 {
                return Err(battle_err!("command learn: invalid entry in row {i}"));
            }
            previous = entry.required_level;
        }
        out.insert(row.idx, row.entries.clone());
    }
    Ok(out)
}

/// LoadCommandLearnSelectors reads the 11-byte 0x4E4D1 growth records and
/// preserves their final learn_idx byte. Native level-up does not index the
/// command table with portrait: it first resolves unit+7 through this table,
/// then uses byte10 as the 0x4E4A2 command-learn row. 0xFF means no row.
///
/// Go：`command_learn.go:62`，`func LoadCommandLearnSelectors(path string) (map[int]int, error)`
pub fn load_command_learn_selectors(path: &str) -> Result<BTreeMap<i64, i64>> {
    let raw = read_file(path)?;
    let rows: Vec<CommandLearnSelectorRow> = decode_json(&raw)?;
    if rows.is_empty() {
        return Err(battle_err!("command learn selectors: empty growth table"));
    }
    let mut out = BTreeMap::new();
    for (i, row) in rows.iter().enumerate() {
        if row.idx != i as i64 || row.learn_idx < 0 || row.learn_idx > 0xff {
            return Err(battle_err!("command learn selectors: invalid row {i}"));
        }
        if row.learn_idx != 0xff {
            out.insert(row.idx, row.learn_idx);
        }
    }
    Ok(out)
}

impl State {
    /// Go：`command_learn.go:86`，`func (s *State) learnNativeCommandsAtLevel(u *Unit) []int`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn learn_native_commands_at_level(&mut self, u: Option<UnitIdx>) -> Vec<i64> {
        let Some(u) = u else {
            return Vec::new();
        };
        let (Some(learn), Some(selectors)) = (&self.command_learn, &self.command_learn_selectors) else {
            return Vec::new();
        };
        if !self.units[u].has_battle_fig {
            return Vec::new();
        }
        let Some(learn_idx) = selectors.get(&self.units[u].battle_fig) else {
            return Vec::new();
        };
        let Some(entries) = learn.get(learn_idx).cloned() else {
            return Vec::new();
        };
        let mut learned = Vec::new();
        let unit = &mut self.units[u];
        for entry in entries {
            if entry.required_level == unit.lv && unit.enable_native_command(entry.command_id) {
                learned.push(entry.command_id);
            }
        }
        learned
    }
}
