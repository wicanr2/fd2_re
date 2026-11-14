//! Go 來源：`remake/internal/campaign/native_join_base.go`（92 行）。

use super::*;

pub(crate) const NATIVE_JOIN_BASE_EXESHA256: &str = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f";

/// NativeJoinBaseTable names decoded map records that supply the visual/base
/// fields for a native sub_112A5 JOIN when that character is not present in the
/// current runtime array.  The constructor statistics still come from
/// native_join_constructor.json; this table only preserves an explicit source
/// for fields that sub_112A5 receives from its character data helpers.
///
/// Go：`native_join_base.go:19`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeJoinBaseTable {
    pub(crate) rows: BTreeMap<i64, NativeJoinBaseRow>,
    pub(crate) source_dir: String,
}

/// Go：`native_join_base.go:24`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeJoinBaseRow {
    pub roster: String,
    pub unit_index: i64,
    pub raw_byte8: i64,
}

/// LoadNativeJoinBaseTable validates a small, evidence-labelled map-record
/// index.  It never searches arbitrary maps or infers Fig==character ID.
///
/// Go：`native_join_base.go:32`，`func LoadNativeJoinBaseTable(path string) (NativeJoinBaseTable, error)`
pub fn load_native_join_base_table(path: &str) -> Result<NativeJoinBaseTable> {
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct WireSource {
        exe_sha256: String,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct WireRow {
        id: i64,
        roster: String,
        unit_index: i64,
        raw_byte8: i64,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct Wire {
        schema_version: i64,
        source: WireSource,
        evidence_level: String,
        #[serde(deserialize_with = "battle::null_as_default")]
        rows: Vec<WireRow>,
    }
    let raw = battle::read_file(path)?;
    let wire: Wire = battle::decode_json(&raw)?;
    if wire.schema_version != 1
        || wire.source.exe_sha256 != NATIVE_JOIN_BASE_EXESHA256
        || wire.evidence_level != "強推論"
        || wire.rows.is_empty()
    {
        return Err(campaign_err!("native JOIN base manifest identity is invalid"));
    }
    let mut table = NativeJoinBaseTable { rows: BTreeMap::new(), source_dir: go_filepath_dir(path) };
    for row in &wire.rows {
        if !joinable_character_id(row.id)
            || row.roster.is_empty()
            || row.unit_index < 0
            || row.raw_byte8 < 0
            || row.raw_byte8 > 0xff
            || row.raw_byte8 != row.id
        {
            return Err(campaign_err!("native JOIN base row {} is invalid", row.id));
        }
        if table.rows.contains_key(&row.id) {
            return Err(campaign_err!("native JOIN base row {} repeats", row.id));
        }
        table.rows.insert(
            row.id,
            NativeJoinBaseRow { roster: row.roster.clone(), unit_index: row.unit_index, raw_byte8: row.raw_byte8 },
        );
    }
    Ok(table)
}

/// Go `filepath.Dir`：只用來保存資料表所在目錄，之後與 `filepath.Join` 搭配讀檔。Rust 以
/// `std::path` 取父目錄；沒有父目錄時與 Go 一樣回 `"."`。
pub(crate) fn go_filepath_dir(path: &str) -> String {
    match std::path::Path::new(path).parent() {
        Some(p) if !p.as_os_str().is_empty() => p.to_string_lossy().into_owned(),
        _ => ".".to_string(),
    }
}

impl NativeJoinBaseTable {
    /// LoadBaseUnit returns only the explicitly indexed decoded map record and
    /// verifies its independent raw +8 provenance before it can seed JOIN.
    ///
    /// Go：`native_join_base.go:72`，`func (table NativeJoinBaseTable) LoadBaseUnit(id int) (battle.Unit, error)`
    pub fn load_base_unit(&self, id: i64) -> Result<Unit> {
        let Some(row) = self.rows.get(&id) else {
            return Err(campaign_err!("native JOIN character {id} has no proven base record"));
        };
        let path = std::path::Path::new(&self.source_dir).join(&row.roster);
        let state = battle::load(&path.to_string_lossy())
            .map_err(|err| CampaignError::from(err).wrap(format!("native JOIN character {id} base roster: ")))?;
        // Go 另檢查 `state.Units[i] == nil`；Rust 的 `units` 沒有空位。
        if row.unit_index >= state.units.len() as i64 {
            return Err(campaign_err!("native JOIN character {id} base unit index {} is unavailable", row.unit_index));
        }
        let base = state.units[row.unit_index as usize].clone();
        if base.camp != battle::ALLY && base.camp != battle::OWN {
            return Err(campaign_err!("native JOIN character {id} base unit is not allied"));
        }
        if !base.has_native_record_byte8 || i64::from(base.native_record_byte8) != row.raw_byte8 {
            return Err(campaign_err!("native JOIN character {id} base record lacks raw +8={}", row.raw_byte8));
        }
        Ok(base)
    }
}

#[cfg(test)]
mod native_join_base_test;
