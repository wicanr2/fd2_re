//! D 組測試共用的小工具（只在測試編譯）。Go 版的這些 helper 散在同套件的各個 `_test.go`，
//! Go 同一套件的測試檔互相看得到；Rust 集中放在這裡給 D 組的測試模組引用。

use super::*;
use crate::test_support::remake_asset;

/// `NATIVE_RECORD_SIZE` 的 `usize` 版。
pub(crate) const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go `completeNativeAIScoringUnit`（`native_ai_record_test.go`）。
pub(crate) fn complete_native_ai_scoring_unit() -> Unit {
    Unit {
        x: 99,
        y: 98,
        native_map_presentation: NativeMapPresentationState { x: 7, y: 8, pose: 3, motion: 4 },
        has_native_map_presentation: true,
        battle_fig: 9,
        has_battle_fig: true,
        native_identity: 10,
        has_native_identity: true,
        native_record_byte8: 0x60,
        has_native_record_byte8: true,
        native_record_race: 11,
        has_native_record_race: true,
        native_record_class: 12,
        has_native_record_class: true,
        native_record_byte5: 0x20,
        has_native_record_byte5: true,
        native_record_byte6: 1,
        has_native_record_byte6: true,
        native_record_byte34: 0x81,
        has_native_record_byte34: true,
        native_record_byte35: 13,
        has_native_record_byte35: true,
        native_record_byte36: 14,
        has_native_record_byte36: true,
        native_record_word42: 234,
        has_native_record_word42: true,
        native_record_word46: 18,
        has_native_record_word46: true,
        inventory_slots: vec![1, 2, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0x40, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
        native_command_mask: [1, 2, 3, 4, 5],
        native_transient: [6, 7, 8, 9, 10, 0],
        lv: 15,
        mv: 6,
        hp: 123,
        max_hp: 234,
        mp: 17,
        max_mp: 18,
        ap: 19,
        dp: 20,
        hit: 21,
        ev: 22,
        ..Default::default()
    }
}

/// Go `nativeCompositionBaseFlagsForTest`（`native_ai_phase_diagnostic_test.go`）。
pub(crate) fn native_composition_base_flags_for_test(st: &State) -> Vec<u8> {
    native_composition_base_flags(st.w, st.h, &st.native_composition_event_bytes).unwrap()
}

/// 36 筆、只填 ID 的指令表（Go 測試常見的 `book[id] = NativeCommandRecord{ID: id}` 迴圈）。
pub(crate) fn native_command_book_ids() -> Vec<NativeCommandRecord> {
    (0..NATIVE_COMMAND_RECORD_COUNT).map(|id| NativeCommandRecord { id, ..Default::default() }).collect()
}

/// Go 測試的 `Load("../../assets/<rel>")`；缺檔時依 `FD2_REQUIRE_ORIGINAL` 略過或失敗。
pub(crate) fn load_asset_state(rel: &str) -> Option<State> {
    remake_asset(rel).map(|path| load(&path).unwrap())
}

/// Go 測試的 `../../assets/<rel>` 路徑。
pub(crate) fn asset_path(rel: &str) -> Option<String> {
    remake_asset(rel)
}

/// Go `binary.LittleEndian.PutUint16(records[index*nativeRecordSize+offset:], v)`。
pub(crate) fn put_record16(records: &mut [u8], index: usize, offset: usize, value: u16) {
    records[index * RS + offset..index * RS + offset + 2].copy_from_slice(&value.to_le_bytes());
}

/// Go `binary.LittleEndian.Uint16(records[index*nativeRecordSize+offset:])`。
pub(crate) fn record16(records: &[u8], index: usize, offset: usize) -> u16 {
    u16::from_le_bytes([records[index * RS + offset], records[index * RS + offset + 1]])
}
