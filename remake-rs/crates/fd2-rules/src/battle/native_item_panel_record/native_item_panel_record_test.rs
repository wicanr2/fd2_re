//! 移植自 `remake/internal/battle/native_item_panel_record_test.go`。

use super::*;
use crate::test_support::get16;

/// Go：`TestNativeItemPanelRecordForUnitUsesProvenRawSelectors`
#[test]
fn native_item_panel_record_for_unit_uses_proven_raw_selectors() {
    let unit = Unit {
        battle_fig: 9,
        native_identity: 4,
        has_native_identity: true,
        native_record_byte8: 0x60,
        has_native_record_byte8: true,
        native_record_byte6: 1,
        has_native_record_byte6: true,
        native_record_race: 7,
        has_native_record_race: true,
        native_record_class: 8,
        has_native_record_class: true,
        lv: 12,
        mv: 5,
        exp: 34.0,
        dx: 56,
        hp: 80,
        max_hp: 100,
        mp: 20,
        max_mp: 40,
        ap: 123,
        dp: 98,
        hit: 76,
        ev: 54,
        inventory_slots: vec![0, 79, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0x40, 0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
        native_transient: [1, 2, 3, 4, 5, 6],
        ..Default::default()
    };
    let record = native_item_panel_record_for_unit(Some(&unit)).unwrap();
    assert!(
        record.len() == 80
            && record[6] == 1
            && record[7] == 9
            && record[8] == 0x60
            && record[0x0a] == 0x40
            && record[0x0b] == 0
            && record[0x0d] == 79
            && record[0x1f] == 7
            && record[0x20] == 8
            && record[0x21] == 12
            && record[0x22] == 1
            && record[0x27] == 6
            && record[0x3b] == 5
            && record[0x3c] == 34,
        "record bytes={record:?}"
    );
    for (offset, want) in
        [(0x3e, 56), (0x40, 80), (0x42, 100), (0x44, 20), (0x46, 40), (0x48, 123), (0x4a, 98), (0x4c, 76), (0x4e, 54)]
    {
        assert_eq!(get16(&record, offset), want, "record[{offset:#x}]");
    }
}

/// Go：`TestNativeItemPanelRecordForUnitRejectsMissingRawProvenance`
#[test]
fn native_item_panel_record_for_unit_rejects_missing_raw_provenance() {
    let missing = Unit {
        battle_fig: 1,
        native_identity: 2,
        has_native_identity: true,
        lv: 1,
        inventory_slots: vec![0; 8],
        native_inventory_flags: vec![0; 8],
        ..Default::default()
    };
    assert!(native_item_panel_record_for_unit(Some(&missing)).is_err(), "missing raw provenance accepted");
    let fractional = Unit {
        battle_fig: 1,
        native_identity: 2,
        has_native_identity: true,
        native_record_byte6: 1,
        has_native_record_byte6: true,
        native_record_race: 1,
        has_native_record_race: true,
        native_record_class: 2,
        has_native_record_class: true,
        lv: 1,
        exp: 1.5,
        inventory_slots: vec![0; 8],
        native_inventory_flags: vec![0; 8],
        ..Default::default()
    };
    assert!(native_item_panel_record_for_unit(Some(&fractional)).is_err(), "fractional EXP accepted");
}
