//! 移植自 `remake/internal/campaign/native_current_snapshot_test.go`。
//!
//! `battle::Unit` 與 `battle::State` 有 fd2-rules 私有欄位，crate 外不能用結構更新語法，
//! 測試一律從 `Default` 逐欄設定。

use super::*;
use crate::test_support::remake_asset;

/// Go `completeNativeCurrentSaveUnit`。
pub(crate) fn complete_native_current_save_unit() -> Unit {
    let mut u = Unit::default();
    (u.x, u.y, u.on_field, u.acted) = (7, 8, true, true);
    u.native_map_presentation = battle::NativeMapPresentationState { x: 7, y: 8, pose: 2, motion: 3 };
    u.has_native_map_presentation = true;
    (u.battle_fig, u.has_battle_fig) = (9, true);
    (u.map_selector_key, u.has_map_selector_key) = (9, true);
    (u.map_selector_slot, u.has_map_selector_slot) = (0, true);
    (u.native_record_byte5, u.has_native_record_byte5) = (0x80, true);
    (u.native_record_byte6, u.has_native_record_byte6) = (2, true);
    (u.native_record_byte8, u.has_native_record_byte8) = (4, true);
    (u.native_record_race, u.has_native_record_race) = (1, true);
    (u.native_record_class, u.has_native_record_class) = (2, true);
    (u.native_record_byte34, u.has_native_record_byte34) = (0x81, true);
    (u.native_record_byte35, u.has_native_record_byte35) = (5, true);
    (u.native_record_byte36, u.has_native_record_byte36) = (6, true);
    (u.native_record_byte3_d, u.has_native_record_byte3_d) = (7, true);
    (u.native_record_death_effect, u.has_native_record_death_effect) = ([8, 9, 10], true);
    u.inventory_slots = vec![1, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff];
    u.native_inventory_flags = vec![0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80];
    (u.native_command_mask, u.native_transient) = ([1, 2, 3, 4, 5], [6, 7, 8, 9, 10, 11]);
    (u.lv, u.mv, u.exp, u.dx) = (12, 5, 13.0, 14);
    (u.hp, u.max_hp, u.mp, u.max_mp, u.ap, u.dp, u.hit, u.ev) = (15, 16, 17, 18, 19, 20, 21, 22);
    u
}

#[test]
fn test_build_native_current_persistent_records_adds_only_proven_join() {
    let Some(table_path) = remake_asset("data/native_join_constructor.json") else { return };
    let Some(rows_path) = remake_asset("data/native_item_effect_rows.json") else { return };
    let table = load_native_join_constructor_table(&table_path).unwrap();
    let item_rows = battle::load_native_item_effect_row_prefix(&rows_path).unwrap();
    let joined = table.materialize_persistent_unit(12, &Unit::default(), &item_rows).unwrap();
    let mut baseline = save::CurrentSnapshot::default();
    baseline.header.runtime_count = 1;
    baseline.header.persistent_count = 1;
    baseline.persistent_records[0].raw[8] = 4;
    let mut enemy = Unit::default();
    enemy.camp = battle::ENEMY;
    let mut state = battle::State::default();
    state.units = vec![enemy, joined];
    state.native_runtime_records = vec![battle::NativeRuntimeRecordState::default(); 2];
    let (records, count) =
        build_native_current_persistent_records(Some(&state), &baseline, &table, &item_rows).unwrap();
    assert!(
        count == 2
            && records[0].raw[8] == 4
            && records[1].raw[8] == 12
            && records[1].raw[7] == 12
            && records[1].raw[6] == 2,
        "persistent JOIN count={count} first={} joined={:?}",
        records[0].raw[8],
        records[1].view()
    );
    // Go 改 `joined` 指標指向的單位；Rust 直接改 `state.units[1]`。
    state.units[1].native_identity = 4;
    state.units[1].native_record_byte8 = 4;
    assert!(
        build_native_current_persistent_records(Some(&state), &baseline, &table, &item_rows).is_err(),
        "duplicate appended JOIN identity was accepted"
    );
}

#[test]
fn test_build_native_current_runtime_records_overlays_live_fields_and_preserves_opaque() {
    let mut state = battle::State::default();
    state.units = vec![complete_native_current_save_unit()];
    state.has_native_runtime_unit_projection = true;
    state.native_runtime_records =
        vec![battle::NativeRuntimeRecordState { selector_key: 9, selector_slot: 0, ..Default::default() }];
    state.native_runtime_records[0].raw = [0xcc; 80];
    state.native_runtime_records[0].raw[7] = 9;
    let records = build_native_current_runtime_records(Some(&state)).unwrap();
    let raw = records[0].raw;
    assert!(
        raw[0] == 7
            && raw[1] == 8
            && raw[3] == 2
            && raw[4] == 3
            && raw[5] == 0x80
            && raw[6] == 2
            && raw[7] == 9
            && raw[8] == 4
            && raw[0x31] == 8
            && raw[0x36] == 6
            && raw[0x3d] == 7,
        "live raw fields={:x?}",
        &raw[..0x3e]
    );
    assert!(
        raw[2] == 0xcc && raw[9] == 0xcc && raw[0x28] == 0xcc && raw[0x37] == 0xcc,
        "opaque baseline bytes were not preserved"
    );
    assert_eq!(u16::from_le_bytes([raw[0x40], raw[0x41]]), 15, "HP");
    state.units[0].hp = 99;
    assert_eq!(u16::from_le_bytes([raw[0x40], raw[0x41]]), 15, "returned record aliases Unit");
}

#[test]
fn test_build_native_current_runtime_records_fails_closed_on_topology_or_live_mismatch() {
    let mut base = battle::NativeRuntimeRecordState { selector_key: 9, selector_slot: 0, ..Default::default() };
    base.raw[7] = 9;
    let mut state = battle::State::default();
    state.units = vec![complete_native_current_save_unit()];
    state.native_runtime_records = vec![base];
    assert!(build_native_current_runtime_records(Some(&state)).is_err(), "non-CONTINUE roster unexpectedly accepted");
    state.has_native_runtime_unit_projection = true;
    state.units[0].x += 1;
    assert!(
        build_native_current_runtime_records(Some(&state)).is_err(),
        "stale map presentation unexpectedly accepted"
    );
    state.units[0].x -= 1;
    state.units[0].acted = false;
    assert!(build_native_current_runtime_records(Some(&state)).is_err(), "raw acted mismatch unexpectedly accepted");
}
