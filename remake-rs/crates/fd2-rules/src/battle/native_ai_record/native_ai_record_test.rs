//! 移植自 `remake/internal/battle/native_ai_record_test.go`。

use super::*;
use crate::test_support::{get16, remake_asset};

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go `completeNativeAIScoringUnit`
fn complete_native_ai_scoring_unit() -> Unit {
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

/// Go：`TestNativeAIScoringRecordsUsesRawPresentationAndAIBytes`
#[test]
fn native_ai_scoring_records_uses_raw_presentation_and_ai_bytes() {
    let unit = complete_native_ai_scoring_unit();
    let mut records = native_ai_scoring_records(std::slice::from_ref(&unit)).unwrap();
    assert_eq!(records.len(), RS, "record len");
    assert!(
        records[0] == 7
            && records[1] == 8
            && records[3] == 3
            && records[4] == 4
            && records[5] == 0x20
            && records[6] == 1
            && records[8] == 0x60
            && records[0x34] == 0x81
            && records[0x35] == 13
            && records[0x36] == 14,
        "raw AI fields were not preserved: {:x?}",
        &records[..0x37]
    );
    assert_eq!(get16(&records, 0x40), 123, "raw HP");
    assert_eq!(get16(&records, 0x42), 234, "raw max HP");
    assert_eq!(get16(&records, 0x46), 18, "raw max MP");
    records[0] = 0;
    assert_eq!(unit.native_map_presentation.x, 7, "returned record aliases runtime unit");
}

/// Go：`TestNativeAIScoringRecordsFailsClosedWithoutEveryProvenanceField`（`t.Run` 子測試改成迴圈）
#[test]
fn native_ai_scoring_records_fails_closed_without_every_provenance_field() {
    let cases: [(&str, fn(&mut Unit)); 9] = [
        ("presentation", |u| u.has_native_map_presentation = false),
        ("battle_fig", |u| u.has_battle_fig = false),
        ("byte5", |u| u.has_native_record_byte5 = false),
        ("byte34", |u| u.has_native_record_byte34 = false),
        ("byte35", |u| u.has_native_record_byte35 = false),
        ("byte36", |u| u.has_native_record_byte36 = false),
        ("word42", |u| u.has_native_record_word42 = false),
        ("word46", |u| u.has_native_record_word46 = false),
        ("raw_byte8", |u| {
            u.has_native_record_byte8 = false;
            u.has_native_identity = false;
        }),
    ];
    for (name, drop) in cases {
        let mut unit = complete_native_ai_scoring_unit();
        drop(&mut unit);
        assert!(native_ai_scoring_records(&[unit]).is_err(), "{name}: incomplete record accepted");
    }
}

/// Go：`TestMap0AssetsMaterializeNativeAIScoringRecords`
#[test]
fn map0_assets_materialize_native_ai_scoring_records() {
    let Some(path) = remake_asset("maps/map0/map0_units.json") else { return };
    let mut st = load(&path).unwrap();
    for (index, unit) in st.units.iter_mut().enumerate() {
        unit.materialize_native_map_presentation().unwrap_or_else(|e| panic!("unit {index} presentation: {e}"));
    }
    let records = native_ai_scoring_records(&st.units).unwrap();
    assert_eq!(records.len(), st.units.len() * RS, "map0 records len");
    let first = &records[..RS];
    assert!(
        first[0] == 1
            && first[1] == 3
            && first[5] == 0
            && first[6] == 0
            && first[0x34] == 0
            && get16(first, 0x40) == 28,
        "map0 first native AI record has wrong anchors: {:x?}",
        &first[..0x44]
    );
}

/// Go：`TestMap19AssetsCarryNativeAICommandMaskAndConstructorMP`
#[test]
fn map19_assets_carry_native_ai_command_mask_and_constructor_mp() {
    let Some(path) = remake_asset("maps/map19/map19_units.json") else { return };
    let Some(spells) = remake_asset("spells.json") else { return };
    let mut st = load(&path).unwrap();
    assert!(st.units.len() > 55, "map19 roster len={}", st.units.len());
    let unit = &mut st.units[55];
    assert!(
        unit.native_record_byte8 == 92
            && unit.has_native_record_byte8
            && !unit.has_native_identity
            && unit.native_command_mask == [4, 0, 0, 8, 0]
            && unit.has_native_record_word46
            && unit.native_record_word46 == 288
            && unit.mp == 288
            && unit.max_mp == 288,
        "map19 unit55 raw AI inputs={unit:?}"
    );
    let book = load_native_command_records(&spells).unwrap();
    assert_eq!(native_available_ai_scored_command_ids(Some(unit), &book), [2, 27], "map19 unit55 available commands");
    unit.materialize_native_map_presentation().unwrap();
    let records = native_ai_scoring_records(std::slice::from_ref(unit)).unwrap();
    assert_eq!(get16(&records, 0x44), 288, "map19 unit55 current MP");
    assert_eq!(get16(&records, 0x46), 288, "map19 unit55 max MP");
}
