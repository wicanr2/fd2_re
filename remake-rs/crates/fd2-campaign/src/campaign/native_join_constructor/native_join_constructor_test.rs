//! 移植自 `remake/internal/campaign/native_join_constructor_test.go`。

use super::*;
use crate::test_support::{remake_asset, temp_dir};

fn join_table() -> Option<NativeJoinConstructorTable> {
    let path = remake_asset("data/native_join_constructor.json")?;
    Some(load_native_join_constructor_table(&path).unwrap())
}

fn item_rows() -> Option<Vec<u8>> {
    let path = remake_asset("data/native_item_effect_rows.json")?;
    Some(battle::load_native_item_effect_row_prefix(&path).unwrap())
}

#[test]
fn test_native_join_constructor_materializes_keli_from_raw_tables() {
    let (Some(table), Some(item_rows)) = (join_table(), item_rows()) else { return };
    // `Unit` 有 fd2-rules 私有欄位，crate 外不能用結構更新語法，逐欄設定。
    let mut base = Unit::default();
    base.name = "凱麗".into();
    (base.hp, base.max_hp, base.mp, base.max_mp, base.ap, base.dp) = (90, 90, 7, 7, 44, 33);
    base.exp = 42.0;
    base.native_command_mask = [9, 9, 9, 9, 9];
    base.native_transient = [1, 2, 3, 4, 5, 6];
    let unit = table.materialize_persistent_unit(12, &base, &item_rows).unwrap();
    assert!(
        unit.name == "凱麗"
            && unit.lv == 10
            && unit.class_id == 8
            && unit.mv == 5
            && unit.hp == 151
            && unit.max_hp == 151
            && unit.mp == 0
            && unit.max_mp == 0
            && unit.native_record_word42 == 151
            && unit.has_native_record_word42
            && unit.native_record_word46 == 0
            && unit.has_native_record_word46
            && unit.native_identity == 12
            && unit.has_native_identity
            && unit.native_record_byte5 == 0
            && unit.has_native_record_byte5
            && unit.native_record_byte6 == 2
            && unit.has_native_record_byte6
            && unit.map_selector_key == 12
            && unit.has_map_selector_key
            && unit.battle_fig == 12
            && unit.has_battle_fig
            && unit.exp == 0.0
            && unit.native_command_mask == [0; 5]
            && unit.native_transient == [0; 6],
        "unexpected Keli JOIN projection: {unit:?}"
    );
    assert!(
        unit.ap == 100
            && unit.dp == 79
            && unit.hit == 110
            && unit.ev == 15
            && unit.dx == 10
            && unit.base_ap == 80
            && unit.base_dp == 69
            && unit.base_hit == 10
            && unit.base_ev == 10
            && unit.equipment_base_set,
        "unexpected Keli 0x1145A projection: {unit:?}"
    );
    assert!(
        unit.native_inventory_flags == vec![0x40, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80]
            && unit.inventory_slots == vec![0x3e, 0xac, 0xff, 0xff, 0xff, 0xff, 0, 0]
            && unit.inventory == vec![0x3e, 0xac]
            && unit.equipped == vec![true, true],
        "unexpected Keli JOIN inventory: slots={:?} flags={:?} inventory={:?} equipped={:?}",
        unit.inventory_slots,
        unit.native_inventory_flags,
        unit.inventory,
        unit.equipped
    );
}

#[test]
fn test_native_join_constructor_rejects_wrong_executable_identity() {
    let path = temp_dir("join-identity").join("join.json");
    std::fs::write(&path, r#"{"schema_version":1,"source":{"exe_size":1},"evidence_level":"已證實","rows":[]}"#)
        .unwrap();
    assert!(
        load_native_join_constructor_table(&path.to_string_lossy()).is_err(),
        "wrong executable identity was accepted"
    );
}

#[test]
fn test_native_join_constructor_rejects_unknown_identity() {
    let Some(table) = join_table() else { return };
    assert!(
        table.materialize_persistent_unit(32, &Unit::default(), &[]).is_err(),
        "unknown JOIN identity was accepted"
    );
}

#[test]
fn test_native_join_constructor_rejects_missing_equipped_item_row_atomically() {
    let Some(table) = join_table() else { return };
    let mut base = Unit::default();
    (base.name, base.ap, base.inventory) = ("不可變".into(), 777, vec![9]);
    let short = vec![0u8; battle::NATIVE_ITEM_EFFECT_ROW_SIZE as usize];
    if let Ok(unit) = table.materialize_persistent_unit(12, &base, &short) {
        panic!("short item table was accepted: {unit:?}");
    }
    assert!(base.ap == 777 && base.inventory == vec![9], "failed transaction mutated caller base: {base:?}");
}

#[test]
fn test_native_join_constructor_materializes_all_known_rows() {
    let (Some(table), Some(item_rows)) = (join_table(), item_rows()) else { return };
    for id in 0..32 {
        let unit = table
            .materialize_persistent_unit(id, &Unit::default(), &item_rows)
            .unwrap_or_else(|err| panic!("row {id}: {err}"));
        assert!(
            unit.native_identity == id
                && unit.inventory_slots.len() == 8
                && unit.native_inventory_flags.len() == 8
                && unit.equipment_base_set,
            "row {id} incomplete projection: {unit:?}"
        );
    }
}

#[test]
fn test_native_join_constructor_rejects_offset_drift() {
    let Some(source) = remake_asset("data/native_join_constructor.json") else { return };
    let raw = std::fs::read_to_string(&source).unwrap();
    let tampered = raw.replacen(r#""default_file_offset": "0x55ba1""#, r#""default_file_offset": "0x55ba2""#, 1);
    assert_ne!(raw, tampered, "fixture did not contain row-zero source offset");
    let path = temp_dir("join-drift").join("join.json");
    std::fs::write(&path, tampered).unwrap();
    assert!(load_native_join_constructor_table(&path.to_string_lossy()).is_err(), "shifted source offset was accepted");
}

#[test]
fn test_native_join_constructor_clears_item_flags_and_transient_over_residual() {
    // 0x1138B：有物品的格旗標寫 0；0x113C9：+0x22..+0x27 清零。第十章 r4 酒店存檔 id 11 的
    // +0x0E 殘值 0xF8 被寫成 0。
    let (Some(table), Some(item_rows)) = (join_table(), item_rows()) else { return };
    let residual = save::PersistentRecord { raw: [0xf8; save::UNIT_SIZE] };
    let record = table.materialize_persistent_record_on(residual, 11, &item_rows).unwrap();
    let raw = record.raw;
    for slot in 0..4 {
        let cell = 0x0e + slot * 2;
        let want = if raw[cell + 1] == 0xff { 0x80 } else { 0 };
        assert_eq!(raw[cell], want, "slot {slot} flag={:#x} item={:#x} want {want:#x}", raw[cell], raw[cell + 1]);
    }
    assert_eq!(raw[0x22..0x28], [0u8; 6], "transient bytes kept residual: {:x?}", &raw[0x22..0x28]);
}
