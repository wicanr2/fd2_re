//! Go 來源：`remake/internal/campaign/native_join_constructor.go`（283 行）。

use super::*;

pub(crate) const NATIVE_JOIN_EXE_SIZE: i64 = 357074;

pub(crate) const NATIVE_JOIN_EXEMD5: &str = "b97caf2239a27a896069d03549d96e1e";

pub(crate) const NATIVE_JOIN_EXESHA256: &str = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f";

/// NativeJoinConstructorTable is the address-preserving projection of the
/// 32-row tables consumed by original sub_112A5. It deliberately does not reuse
/// the unrelated FDFIELD constructor table.
///
/// Go：`native_join_constructor.go:25`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeJoinConstructorTable {
    pub(crate) rows: BTreeMap<i64, NativeJoinConstructorRow>,
}

/// Go：`native_join_constructor.go:29`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativeJoinConstructorRow {
    pub(crate) defaults: [u8; 24],
    pub(crate) growth: [u8; 11],
}

/// LoadNativeJoinConstructorTable validates the executable identity, row order,
/// source offsets and raw strides before exposing any constructor row.
///
/// Go：`native_join_constructor.go:36`，`func LoadNativeJoinConstructorTable(path string) (NativeJoinConstructorTable, error)`
pub fn load_native_join_constructor_table(path: &str) -> Result<NativeJoinConstructorTable> {
    todo!("campaign/native_join_constructor.go:LoadNativeJoinConstructorTable")
}

/// Go：`native_join_constructor.go:95`，`func parseNativeJoinOffset(value string) (int, error)`
pub(crate) fn parse_native_join_offset(value: &str) -> Result<i64> {
    todo!("campaign/native_join_constructor.go:parseNativeJoinOffset")
}

/// Go：`native_join_constructor.go:100`，`func nativeJoinWord(raw []byte, offset int) int`
pub(crate) fn native_join_word(raw: &[u8], offset: i64) -> i64 {
    todo!("campaign/native_join_constructor.go:nativeJoinWord")
}

impl NativeJoinConstructorTable {
    /// MaterializePersistentUnit transcribes the proven sub_112A5 writes into a
    /// local persistent record and runs its sub_1145A equipment tail. The local
    /// record is transaction staging, not a claim that untouched bytes of every
    /// native 0x50-byte record are known. No Unit is returned unless the raw item
    /// rows needed by every equipped slot are available.
    ///
    /// Go：`native_join_constructor.go:109`，`func (table NativeJoinConstructorTable) MaterializePersistentUnit( id int, base battle.Unit, itemTable []byte, ) (battle.Unit, error)`
    pub fn materialize_persistent_unit(&self, id: i64, base: &Unit, item_table: &[u8]) -> Result<Unit> {
        self.materialize_persistent_unit_on(save::PersistentRecord::default(), id, base.clone(), item_table)
    }

    /// MaterializePersistentUnitOn 同 MaterializePersistentUnit，但建構器沒寫到的 byte 照
    /// residual（sub_112A5 寫進去的那一格在寫入前的內容）；空物品格的 item byte 也跟著
    /// residual 走，之後寫回槽時才會和原版逐 byte 相同。
    ///
    /// Go：`native_join_constructor.go:120`，`func (table NativeJoinConstructorTable) MaterializePersistentUnitOn( residual fdsave.PersistentRecord, id int, base battle.Unit, itemTable []byte, ) (battle.Unit, error)`
    pub fn materialize_persistent_unit_on(
        &self,
        residual: save::PersistentRecord,
        id: i64,
        base: Unit,
        item_table: &[u8],
    ) -> Result<Unit> {
        todo!("campaign/native_join_constructor.go:NativeJoinConstructorTable.MaterializePersistentUnitOn")
    }

    /// MaterializePersistentRecord exposes the exact local 0x50-byte record built
    /// by the proven JOIN constructor. It is used when current-battle SAVE must add
    /// a newly joined character to the native persistent roster without inventing
    /// unknown bytes from a normalized Unit.
    ///
    /// Go：`native_join_constructor.go:208`，`func (table NativeJoinConstructorTable) MaterializePersistentRecord( id int, itemTable []byte, ) (fdsave.PersistentRecord, error)`
    pub fn materialize_persistent_record(&self, id: i64, item_table: &[u8]) -> Result<save::PersistentRecord> {
        self.materialize_persistent_record_on(save::PersistentRecord::default(), id, item_table)
    }

    /// MaterializePersistentRecordOn 把 sub_112A5 的寫入疊在 residual 上：持續名冊記憶體是
    /// 0x10010 從槽整批還原的，count 之後那一格在 JOIN 之前就是 LOAD 時槽裡的 bytes。
    /// 沒有 LOAD 時本引擎使用零基底，不宣稱原版新開局全零；首章 START 收據已有非零反證。
    /// 建構器沒寫到的 byte 原封不動留在存檔裡（第七章 r6 凱麗：+0..+4、+0x17、
    /// +0x19、+0x28..+0x30、+0x32..+0x36、+0x3d 都是 LOAD 進來的殘值）。
    ///
    /// Go：`native_join_constructor.go:220`，`func (table NativeJoinConstructorTable) MaterializePersistentRecordOn( residual fdsave.PersistentRecord, id int, itemTable []byte, ) (fdsave.PersistentRecord, error)`
    pub fn materialize_persistent_record_on(
        &self,
        residual: save::PersistentRecord,
        id: i64,
        item_table: &[u8],
    ) -> Result<save::PersistentRecord> {
        todo!("campaign/native_join_constructor.go:NativeJoinConstructorTable.MaterializePersistentRecordOn")
    }
}
