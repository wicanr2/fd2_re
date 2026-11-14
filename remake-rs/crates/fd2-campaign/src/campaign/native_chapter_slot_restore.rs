//! Go 來源：`remake/internal/campaign/native_chapter_slot_restore.go`（235 行）。

use super::*;

pub(crate) const NATIVE_INTERMISSION_GATE_COUNT: i64 = 30;

pub(crate) const NATIVE_INTERMISSION_GATE_ADDRESS: &str = "0x526b9";

/// NativeIntermissionGateSource binds the editable table to the exact original
/// executable and byte address from which it was transcribed.
///
/// Go：`native_chapter_slot_restore.go:21`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeIntermissionGateSource {
    /// Go 標籤：`json:"reference_file"`
    pub reference_file: String,
    /// Go 標籤：`json:"exe_sha256"`
    #[serde(rename = "exe_sha256")]
    pub exesha256: String,
    /// Go 標籤：`json:"address"`
    pub address: String,
}

/// NativeIntermissionGateTable is the original 30-byte table consumed by
/// 0x2cad7. Zero enters the selectable town hub; nonzero enters preparation
/// directly. Entries use the raw [0x53c03] chapter index, not display chapter.
///
/// Go：`native_chapter_slot_restore.go:30`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeIntermissionGateTable {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"source"`
    pub source: NativeIntermissionGateSource,
    /// Go 標籤：`json:"entries"`（Go `[]byte`：base64 字串，讀取時也接受數字陣列）
    #[serde(with = "battle::go_bytes")]
    pub entries: Vec<u8>,
    /// Go 標籤：`json:"effective_route_overrides"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub effective_route_overrides: Vec<NativeIntermissionRouteOverride>,
}

/// NativeIntermissionRouteOverride preserves a player-visible effective route
/// that cannot be derived from the 0x526b9 binary gate alone. It supplements,
/// rather than rewrites, the original table and must carry reviewable evidence.
///
/// Go：`native_chapter_slot_restore.go:40`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeIntermissionRouteOverride {
    /// Go 標籤：`json:"raw_chapter"`
    pub raw_chapter: i64,
    /// Go 標籤：`json:"entry_node"`
    pub entry_node: String,
    /// Go 標籤：`json:"evidence"`
    pub evidence: String,
    /// Go 標籤：`json:"evidence_level"`
    pub evidence_level: String,
    /// Go 標籤：`json:"save_sha256"`
    pub save_sha256: String,
}

/// LoadNativeIntermissionGateTable loads and validates a complete version-bound
/// transcription. Missing, reordered, non-binary, or wrong-version data fails
/// closed before campaign state can be changed.
///
/// Go：`native_chapter_slot_restore.go:51`，`func LoadNativeIntermissionGateTable( path string, ) (*NativeIntermissionGateTable, error)`
pub fn load_native_intermission_gate_table(path: &str) -> Result<NativeIntermissionGateTable> {
    let raw = battle::read_file(path).map_err(|err| CampaignError::from(err).wrap("native intermission gate: "))?;
    let table: NativeIntermissionGateTable =
        battle::decode_json(&raw).map_err(|err| CampaignError::from(err).wrap("native intermission gate: "))?;
    table.validate()?;
    Ok(table)
}

impl NativeIntermissionGateTable {
    /// Go：`native_chapter_slot_restore.go:68`，`func (table *NativeIntermissionGateTable) validate() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn validate(&self) -> Result<()> {
        if self.schema_version != 2 {
            return Err(campaign_err!("native intermission gate: schema version={}, want 2", self.schema_version));
        }
        if self.source.reference_file != "docs/data/fd2-reference-files.json"
            || self.source.exesha256 != NATIVE_CHARACTER_EXESHA256
            || self.source.address != NATIVE_INTERMISSION_GATE_ADDRESS
        {
            return Err(campaign_err!("native intermission gate: source does not match reference FD2.EXE"));
        }
        if self.entries.len() as i64 != NATIVE_INTERMISSION_GATE_COUNT {
            return Err(campaign_err!(
                "native intermission gate: entries={}, want {}",
                self.entries.len(),
                NATIVE_INTERMISSION_GATE_COUNT
            ));
        }
        for (index, value) in self.entries.iter().enumerate() {
            if *value > 1 {
                return Err(campaign_err!("native intermission gate: entry {index}={value}, want 0 or 1"));
            }
        }
        let mut seen = BTreeSet::new();
        for over in &self.effective_route_overrides {
            let digest = battle::decode_hex(&over.save_sha256);
            if over.raw_chapter < 0
                || over.raw_chapter >= self.entries.len() as i64
                || over.entry_node.is_empty()
                || over.evidence.is_empty()
                || over.evidence_level != "confirmed-fixed-input"
                || seen.contains(&over.raw_chapter)
                || digest.is_none_or(|d| d.len() != 32)
            {
                return Err(campaign_err!(
                    "native intermission gate: invalid effective route override for raw chapter {}",
                    over.raw_chapter
                ));
            }
            seen.insert(over.raw_chapter);
        }
        Ok(())
    }
}

/// NativeChapterSlotRestorePlan is a fully validated, side-effect-free restore
/// transaction. Raw option bytes are preserved but not connected to gameplay
/// consumers whose semantics remain open.
///
/// Go：`native_chapter_slot_restore.go:119`
#[derive(Debug, Clone, Default, PartialEq)]
pub struct NativeChapterSlotRestorePlan {
    pub slot: i64,
    pub native_chapter_index: i64,
    pub display_chapter: i64,
    pub entry_node: String,
    pub currency: u32,
    pub hud_gate_a: u8,
    pub raw53_af9: u8,
    pub raw51_e61: u8,
    pub raw51_e62: u8,
    pub party_members: BTreeMap<i64, bool>,
    pub party_join_order: Vec<i64>,
    pub party_roster: BTreeMap<i64, Unit>,
}

/// BuildNativeChapterSlotRestorePlan converts one native four-slot snapshot
/// into an existing editable intermission node. The original save callers sit
/// inside 0x2cad7/inn flows, after post-battle inventory gates; therefore this
/// must not replay inventory_recipe_ch21 or inventory_gate_ch27.
///
/// Go：`native_chapter_slot_restore.go:138`，`func BuildNativeChapterSlotRestorePlan( snapshot fdsave.ChapterSlotSnapshot, catalog *NativeCharacterCatalog, table *NativeIntermissionGateTable, graph *Campaign, ) (NativeChapterSlotRestorePlan, error)`
pub fn build_native_chapter_slot_restore_plan(
    snapshot: &save::ChapterSlotSnapshot,
    catalog: &mut NativeCharacterCatalog,
    table: &NativeIntermissionGateTable,
    graph: Option<&Campaign>,
) -> Result<NativeChapterSlotRestorePlan> {
    catalog.validate()?;
    table.validate()?;
    let Some(graph) = graph else {
        return Err(campaign_err!("native chapter restore: missing campaign graph"));
    };
    if snapshot.slot >= save::SLOT_COUNT {
        return Err(campaign_err!(
            "native chapter restore: slot {} outside 0..{}",
            snapshot.slot,
            save::SLOT_COUNT - 1
        ));
    }
    if snapshot.verified.roster_count as usize > save::ROSTER_UNITS {
        return Err(campaign_err!(
            "native chapter restore: roster count {} exceeds native capacity {}",
            snapshot.verified.roster_count,
            save::ROSTER_UNITS
        ));
    }
    let raw_chapter = i64::from(snapshot.verified.chapter);
    if raw_chapter >= table.entries.len() as i64 {
        return Err(campaign_err!("native chapter restore: raw chapter {raw_chapter} outside gate table"));
    }
    let display_chapter = raw_chapter + 1;
    let mut node_type = "town";
    let mut node_id = format!("town_ch{display_chapter:02}");
    if table.entries[raw_chapter as usize] != 0 {
        node_type = "preparation";
        node_id = format!("preparation_ch{display_chapter:02}");
    }
    for over in &table.effective_route_overrides {
        if over.raw_chapter == raw_chapter && over.save_sha256 == snapshot.source_save_sha256 {
            node_id = over.entry_node.clone();
            if node_id.starts_with("preparation_") {
                node_type = "preparation";
            } else if node_id.starts_with("town_") {
                node_type = "town";
            } else {
                return Err(campaign_err!(
                    "native chapter restore: unsupported effective route {}",
                    battle::go_quote(&node_id)
                ));
            }
            break;
        }
    }
    // Go 另檢查 `node == nil`；Rust 的 `nodes` 沒有空值。
    if graph.nodes.get(&node_id).is_none_or(|node| node.r#type != node_type) {
        return Err(campaign_err!(
            "native chapter restore: {node_type} node {} is unavailable",
            battle::go_quote(&node_id)
        ));
    }

    let mut plan = NativeChapterSlotRestorePlan {
        slot: snapshot.slot as i64,
        native_chapter_index: raw_chapter,
        display_chapter,
        entry_node: node_id,
        currency: snapshot.verified.currency,
        hud_gate_a: snapshot.verified.hud_gate_a,
        raw53_af9: snapshot.verified.raw_53af9,
        raw51_e61: snapshot.verified.raw_51e61,
        raw51_e62: snapshot.verified.raw_51e62,
        party_members: BTreeMap::new(),
        party_join_order: Vec::new(),
        party_roster: BTreeMap::new(),
    };
    for (index, record) in snapshot.active_records().into_iter().enumerate() {
        let unit = materialize_native_persistent_party_record(record, catalog)
            .map_err(|err| err.wrap(format!("native chapter restore: roster record {index}: ")))?;
        let identity = unit.native_identity;
        if plan.party_members.get(&identity).copied().unwrap_or(false) {
            return Err(campaign_err!("native chapter restore: duplicate identity {identity}"));
        }
        plan.party_members.insert(identity, true);
        plan.party_join_order.push(identity);
        plan.party_roster.insert(identity, unit);
    }
    Ok(plan)
}

#[cfg(test)]
mod native_chapter_slot_restore_test;
