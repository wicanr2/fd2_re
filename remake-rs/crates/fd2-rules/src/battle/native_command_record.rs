//! Go 來源：`remake/internal/battle/native_command_record.go`（73 行）。

use super::*;

pub const NATIVE_COMMAND_RECORD_COUNT: i64 = 36;

/// NativeCommandRecord is the verified 7-byte 0x4e516 record for IDs 0..35.
/// The field names intentionally describe only their proven call-site roles:
/// generic 0x1cff0 uses SelectionMode (+3) from the actor, then EffectMode
/// (+4) from the confirmed cursor; +5 is MP cost and +6 is target code.
/// It is not the normalized Spell gameplay model.
///
/// Go：`native_command_record.go:17`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommandRecord {
    pub id: i64,
    pub damage: i64,
    pub hit: i64,
    pub selection_mode: i64,
    pub effect_mode: i64,
    pub mp_cost: i64,
    pub target_code: i64,
}

/// Go：`native_command_record.go:27`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub(crate) struct NativeCommandRecordJSON {
    /// Go 標籤：`json:"id"`
    pub id: i64,
    /// Go 標籤：`json:"dmg"`
    pub dmg: i64,
    /// Go 標籤：`json:"hit"`
    pub hit: i64,
    /// Go 標籤：`json:"dist"`
    pub dist: i64,
    /// Go 標籤：`json:"range"`
    pub range: i64,
    /// Go 標籤：`json:"mp"`
    pub mp: i64,
    /// Go 標籤：`json:"target"`
    pub target: i64,
    /// Go 標籤：`json:"raw"`
    #[serde(deserialize_with = "null_as_default")]
    pub raw: String,
}

/// LoadNativeCommandRecords loads only the byte-identical EXE command slice.
/// Existing spells.json is accepted as its physical export, but each editable
/// presentation field must agree with its raw 7-byte source.  This prevents a
/// later normalized spell edit from silently changing native command behavior.
///
/// Go：`native_command_record.go:42`，`func LoadNativeCommandRecords(path string) ([]NativeCommandRecord, error)`
pub fn load_native_command_records(path: &str) -> Result<Vec<NativeCommandRecord>> {
    let raw = read_file(path)?;
    let rows: Vec<NativeCommandRecordJSON> = decode_json(&raw)?;
    if rows.len() as i64 != NATIVE_COMMAND_RECORD_COUNT {
        return Err(battle_err!("native command records len={} want {}", rows.len(), NATIVE_COMMAND_RECORD_COUNT));
    }
    let mut result = Vec::with_capacity(rows.len());
    for (i, row) in rows.iter().enumerate() {
        if row.id != i as i64 {
            return Err(battle_err!("native command id={} at index={}", row.id, i));
        }
        let Some(bytes) = decode_hex(&row.raw).filter(|b| b.len() == 7) else {
            return Err(battle_err!("native command id={} invalid raw", row.id));
        };
        let damage = bytes[0] as i64 | (bytes[1] as i64) << 8;
        if row.dmg != damage
            || row.hit != bytes[2] as i64
            || row.dist != bytes[3] as i64
            || row.range != bytes[4] as i64
            || row.mp != bytes[5] as i64
            || row.target != bytes[6] as i64
        {
            return Err(battle_err!("native command id={} fields disagree with raw", row.id));
        }
        result.push(NativeCommandRecord {
            id: row.id,
            damage,
            hit: bytes[2] as i64,
            selection_mode: bytes[3] as i64,
            effect_mode: bytes[4] as i64,
            mp_cost: bytes[5] as i64,
            target_code: bytes[6] as i64,
        });
    }
    Ok(result)
}

#[cfg(test)]
mod native_command_record_test;
