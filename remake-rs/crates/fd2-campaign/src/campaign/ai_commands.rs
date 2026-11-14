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
    todo!("campaign/ai_commands.go:LoadAICommandSpellMap")
}
