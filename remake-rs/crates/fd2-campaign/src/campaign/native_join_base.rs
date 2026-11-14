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
    todo!("campaign/native_join_base.go:LoadNativeJoinBaseTable")
}

impl NativeJoinBaseTable {
    /// LoadBaseUnit returns only the explicitly indexed decoded map record and
    /// verifies its independent raw +8 provenance before it can seed JOIN.
    ///
    /// Go：`native_join_base.go:72`，`func (table NativeJoinBaseTable) LoadBaseUnit(id int) (battle.Unit, error)`
    pub fn load_base_unit(&self, id: i64) -> Result<Unit> {
        todo!("campaign/native_join_base.go:NativeJoinBaseTable.LoadBaseUnit")
    }
}
