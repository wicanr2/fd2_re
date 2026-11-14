//! Go 來源：`remake/internal/campaign/native_chapter_slot_save.go`（171 行）。

use super::*;

// native_chapter_slot_save.go — 酒店存檔（0x30012）寫回原版四槽的 roster 與 metadata。
//
// 原版把記憶體裡的 32 筆持續紀錄整段抄進 `0x312B + slot×0xA28`，再寫 metadata
// `+0` 章節、`+1` 筆數、`+2..+5` 金幣（`+6..+9` 由系統設定寫，這裡沿用 baseline）。
// 重製端的持續隊伍是 battle.Unit 投影，沒有保留每一個 byte，所以每筆紀錄以
// 讀檔時的 raw baseline 為底，只覆寫已證實的欄位；沒有 raw 來源的 byte 一律不動。

/// EncodeNativePersistentRecord 把一個持續隊伍單位覆寫到它讀檔時的 raw 紀錄上。
/// 覆寫的欄位與 BuildNativeCurrentRuntimeRecords 同一組（座標、+5、+6..+8、物品欄、
/// 指令位元、種族／職業／等級、暫時狀態、+0x34..+0x36、+0x3B..+0x4F），再加上升級
/// 會動的 +0x37／+0x39 基礎攻防（`0x1E529` 寫的兩個 word）。
///
/// Go：`native_chapter_slot_save.go:23`，`func EncodeNativePersistentRecord(baseline fdsave.PersistentRecord, unit *battle.Unit) (fdsave.PersistentRecord, error)`
pub fn encode_native_persistent_record(
    baseline: save::PersistentRecord,
    unit: Option<&Unit>,
) -> Result<save::PersistentRecord> {
    todo!("campaign/native_chapter_slot_save.go:EncodeNativePersistentRecord")
}

/// BuildNativeChapterSlot 由讀檔時的槽快照、目前的持續隊伍與金幣組出要寫回的槽。
/// baseline 裡的每一筆有效紀錄都要在隊伍裡找得到同一個 +8 身分；隊伍多出來的身分
/// 依加入順序以 JOIN 建構列補在後面。不在場的 byte 全部照 baseline。
///
/// Go：`native_chapter_slot_save.go:90`，`func BuildNativeChapterSlot( baseline fdsave.ChapterSlotSnapshot, roster map[int]battle.Unit, joinOrder []int, chapter byte, gold uint32, table NativeJoinConstructorTable, itemRows []byte, ) (fdsave.Slot, error)`
pub fn build_native_chapter_slot(
    baseline: &save::ChapterSlotSnapshot,
    roster: &BTreeMap<i64, Unit>,
    join_order: &[i64],
    chapter: u8,
    gold: u32,
    table: &NativeJoinConstructorTable,
    item_rows: &[u8],
) -> Result<save::Slot> {
    todo!("campaign/native_chapter_slot_save.go:BuildNativeChapterSlot")
}
