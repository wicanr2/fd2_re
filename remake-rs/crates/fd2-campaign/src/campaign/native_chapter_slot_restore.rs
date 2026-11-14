//! Go 來源：`remake/internal/campaign/native_chapter_slot_restore.go`（235 行）。

use super::*;

pub(crate) const NATIVE_INTERMISSION_GATE_COUNT: i64 = 30;

pub(crate) const NATIVE_INTERMISSION_GATE_ADDRESS: &str = "0x526b9";

/// NativeIntermissionGateSource binds the editable table to the exact original
/// executable and byte address from which it was transcribed.
///
/// Go：`native_chapter_slot_restore.go:21`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeIntermissionGateSource {
    /// Go 標籤：`json:"reference_file"`
    pub reference_file: String,
    /// Go 標籤：`json:"exe_sha256"`
    pub exesha256: String,
    /// Go 標籤：`json:"address"`
    pub address: String,
}

/// NativeIntermissionGateTable is the original 30-byte table consumed by
/// 0x2cad7. Zero enters the selectable town hub; nonzero enters preparation
/// directly. Entries use the raw [0x53c03] chapter index, not display chapter.
///
/// Go：`native_chapter_slot_restore.go:30`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeIntermissionGateTable {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"source"`
    pub source: NativeIntermissionGateSource,
    /// Go 標籤：`json:"entries"`
    pub entries: Vec<u8>,
    /// Go 標籤：`json:"effective_route_overrides"`
    pub effective_route_overrides: Vec<NativeIntermissionRouteOverride>,
}

/// NativeIntermissionRouteOverride preserves a player-visible effective route
/// that cannot be derived from the 0x526b9 binary gate alone. It supplements,
/// rather than rewrites, the original table and must carry reviewable evidence.
///
/// Go：`native_chapter_slot_restore.go:40`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
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
    todo!("campaign/native_chapter_slot_restore.go:LoadNativeIntermissionGateTable")
}

impl NativeIntermissionGateTable {
    /// Go：`native_chapter_slot_restore.go:68`，`func (table *NativeIntermissionGateTable) validate() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn validate(&self) -> Result<()> {
        todo!("campaign/native_chapter_slot_restore.go:NativeIntermissionGateTable.validate")
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
    todo!("campaign/native_chapter_slot_restore.go:BuildNativeChapterSlotRestorePlan")
}
