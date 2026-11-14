//! Go 來源：`remake/internal/campaign/ai_commands.go`（50 行）。

use super::*;

/// LoadAICommandSpellMap extracts the spell commands available from the
/// editable 23-byte EXE item rows.  The dump format is
/// TY,AP,HT,DP,EV,S1,S2,R1,R2,K1..K6,MM (23 bytes); command-bearing K4 is
/// therefore raw byte 0x11 (the old RE note calling it +0x10 used a
/// one-based/field-relative offset).  We intentionally expose only command
/// bytes >= 0x10: the native AI converts those to spell_id=command-0x10, while
/// lower bytes are physical/status commands and must not be treated as spells.
///
/// Go：`ai_commands.go:17`，`func LoadAICommandSpellMap(path string) (map[int]int, error)`
pub fn load_ai_command_spell_map(path: &str) -> Result<BTreeMap<i64, i64>> {
    /// Go 函式內的匿名結構 `struct { ID int; Raw string }`。
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct Row {
        id: i64,
        #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
        raw: String,
    }
    let raw = battle::read_file(path)?;
    let rows: Vec<Row> = battle::decode_json::<Option<Vec<Row>>>(&raw)?.unwrap_or_default();
    let mut out = BTreeMap::new();
    for row in &rows {
        if !(0..=0xff).contains(&row.id) {
            return Err(campaign_err!("invalid item id {}", row.id));
        }
        if row.raw.is_empty() {
            return Err(campaign_err!("item {} has no raw record", row.id));
        }
        // Go `hex.DecodeString` 的錯誤文字屬標準函式庫；Rust 只保留本套件的前綴，細節不同
        // （KNOWN_DIFFERENCES）。
        let Some(b) = battle::decode_hex(&row.raw) else {
            return Err(campaign_err!("item {} raw: encoding/hex: invalid byte", row.id));
        };
        if b.len() != 23 {
            return Err(campaign_err!("item {} raw length={}, want 23", row.id, b.len()));
        }
        let command = b[0x11] as i64;
        if command >= 0x10 {
            out.insert(row.id, command - 0x10);
        }
    }
    Ok(out)
}

// 測試：Go `remake/internal/campaign/ai_commands_test.go` 逐條移植。
#[cfg(test)]
mod ai_commands_test;
