//! 移植自 `remake/internal/battle/native_item_table_test.go`。

use super::*;
use crate::test_support::{get16, remake_asset, temp_dir};

const RS: usize = NATIVE_ITEM_EFFECT_ROW_SIZE as usize;

fn tracked_table() -> Option<Vec<u8>> {
    let path = remake_asset("data/native_item_effect_rows.json")?;
    Some(load_native_item_effect_row_prefix(&path).unwrap())
}

fn row(table: &[u8], item_id: usize) -> &[u8] {
    &table[item_id * RS..]
}

/// Go：`TestLoadTrackedNativeItemEffectRowPrefix`
#[test]
fn load_tracked_native_item_effect_row_prefix() {
    let Some(table) = tracked_table() else { return };
    assert_eq!(table.len() / RS, 0xd7, "row count");
    assert_eq!(
        table[..RS],
        [
            0x01, 0x0a, 0x00, 0x5f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x05,
            0x00, 0x32, 0x00, 0x05, 0x00,
        ],
        "row 0"
    );
}

#[derive(serde::Deserialize)]
struct ItemJson {
    id: i64,
    ap: i64,
    hit: i64,
    dp: i64,
    ev: i64,
}

/// Go：`TestTrackedNativeEquipmentWordsMatchNormalizedItems`
#[test]
fn tracked_native_equipment_words_match_normalized_items() {
    let Some(table) = tracked_table() else { return };
    let Some(path) = remake_asset("data/item.json") else { return };
    let items: Vec<ItemJson> = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
    assert_eq!(items.len() * RS, table.len(), "normalized bytes vs raw table bytes");
    for (i, item) in items.iter().enumerate() {
        assert_eq!(item.id, i as i64, "item row {i} id");
        let r = row(&table, i);
        let got = [get16(r, 1), get16(r, 3), get16(r, 5), get16(r, 7)].map(|v| v as i64);
        let want = [item.ap, item.hit, item.dp, item.ev];
        assert_eq!(got, want, "item {i} raw equipment words vs normalized");
    }
}

/// Go：`TestTrackedPermanentBaseStatItemRows`
#[test]
fn tracked_permanent_base_stat_item_rows() {
    let Some(table) = tracked_table() else { return };
    for (item_id, typ, amount, stat) in [
        (198usize, 8u8, 9u16, NATIVE_ITEM_BASE_AP),
        (199, 9, 9, NATIVE_ITEM_BASE_DP),
        (200, 0xa, 7, NATIVE_ITEM_BASE_DX),
    ] {
        let r = row(&table, item_id);
        assert_eq!(r[0x0d], typ, "item {item_id} effect type");
        assert_eq!(get16(r, 0x0e), amount, "item {item_id} amount");
        let (route, ok) = native_item_word_delta_route_for_type(typ as i64);
        assert!(ok && route.base_stat == stat, "item {item_id} route = {route:?}, {ok}");
    }
}

/// Go：`TestTrackedHPRestoreItemRowsPreserveConsumptionBranch`
#[test]
fn tracked_hp_restore_item_rows_preserve_consumption_branch() {
    let Some(table) = tracked_table() else { return };
    for (item_id, want_typ, want_amount, consumes) in
        [(192usize, 5u8, 40u16, true), (195, 5, 999, true), (211, 13, 200, false)]
    {
        let r = row(&table, item_id);
        let (typ, amount) = (r[0x0d], get16(r, 0x0e));
        assert!(
            typ == want_typ && amount == want_amount,
            "item {item_id} type/amount = {typ:#x}/{amount}, want {want_typ:#x}/{want_amount}"
        );
        let (route, ok) = native_item_hp_restore_route_for_type(typ, amount);
        assert!(ok && route.consumes_source == consumes, "item {item_id} route = {route:?}, {ok}");
    }
}

/// Go：`TestTrackedMPRestoreItemRows`
#[test]
fn tracked_mp_restore_item_rows() {
    let Some(table) = tracked_table() else { return };
    for (item_id, want_amount) in [(206usize, 80u16), (207, 200)] {
        let r = row(&table, item_id);
        let (typ, amount) = (r[0x0d], get16(r, 0x0e));
        assert!(
            typ == 11 && amount == want_amount,
            "item {item_id} type/amount = {typ:#x}/{amount}, want 0xb/{want_amount}"
        );
        let (route, ok) = native_item_mp_restore_route_for_type(typ, amount);
        assert!(ok && route.amount == want_amount, "item {item_id} MP route = {route:?}, {ok}");
    }
}

/// Go：`TestTrackedHITEVStepItemRow`
#[test]
fn tracked_hitev_step_item_row() {
    let Some(table) = tracked_table() else { return };
    let r = row(&table, 210);
    assert_eq!(r[0x0d], 12, "item 210 effect type");
    let (route, ok) = native_item_hitev_step_route_for_type(r[0x0d]);
    assert!(ok && route.increment == 15 && !route.consumes_source, "item 210 HIT/EV route = {route:?}, {ok}");
}

/// Go：`TestTrackedAPDPModifierItemRows`
#[test]
fn tracked_apdp_modifier_item_rows() {
    let Some(table) = tracked_table() else { return };
    for (item_id, typ, stat) in [(213usize, 15u8, NATIVE_ITEM_DERIVED_DP), (214, 16, NATIVE_ITEM_DERIVED_AP)] {
        let r = row(&table, item_id);
        assert_eq!(r[0x0d], typ, "item {item_id} effect type");
        let (route, ok) = native_item_apdp_step_route_for_type(r[0x0d]);
        assert!(
            ok && route.derived_stat == stat && !route.consumes_source,
            "item {item_id} AP/DP route = {route:?}, {ok}"
        );
    }
}

/// Go：`TestTrackedMarkerApplicationItemRows`
#[test]
fn tracked_marker_application_item_rows() {
    let Some(table) = tracked_table() else { return };
    for (item_id, typ, marker) in [(212usize, 14u8, 0x26i64), (57, 22, 0x27)] {
        let r = row(&table, item_id);
        assert_eq!(r[0x0d], typ, "item {item_id} effect type");
        let (route, ok) = native_item_marker_application_route_for_type(r[0x0d]);
        assert!(
            ok && route.marker_offset == marker && !route.consumes_source,
            "item {item_id} marker route = {route:?}, {ok}"
        );
    }
}

/// Go：`TestTrackedMarkerClearRestoreItemRows`
#[test]
fn tracked_marker_clear_restore_item_rows() {
    let Some(table) = tracked_table() else { return };
    for (item_id, typ, marker) in [(196usize, 6u8, 0x25i64), (197, 7, 0x26)] {
        let r = row(&table, item_id);
        assert_eq!(r[0x0d], typ, "item {item_id} effect type");
        let (route, ok) = native_item_marker_clear_restore_route_for_type(r[0x0d]);
        assert!(
            ok && route.marker_offset == marker && route.consumes_source,
            "item {item_id} marker-clear route = {route:?}, {ok}"
        );
    }
}

/// Go：`TestTrackedCapacityStepItemRows`
#[test]
fn tracked_capacity_step_item_rows() {
    let Some(table) = tracked_table() else { return };
    for (item_id, want_typ, want_amount, stat) in
        [(94usize, 17u8, 20u16, NATIVE_ITEM_MAX_HP), (95, 18, 20, NATIVE_ITEM_MAX_MP), (96, 19, 1, NATIVE_ITEM_MV)]
    {
        let r = row(&table, item_id);
        let (typ, amount) = (r[0x0d], get16(r, 0x0e));
        assert!(
            typ == want_typ && amount == want_amount,
            "item {item_id} type/amount = {typ:#x}/{amount}, want {want_typ:#x}/{want_amount}"
        );
        let (route, ok) = native_item_capacity_step_route_for_type(typ, amount);
        assert!(ok && route.stat == stat && route.consumes_source, "item {item_id} capacity route = {route:?}, {ok}");
    }
}

/// Go：`TestTrackedCommandDamageItemRows`
#[test]
fn tracked_command_damage_item_rows() {
    let Some(table) = tracked_table() else { return };
    for (item_id, item_type, command_id) in [
        (11usize, 20i64, 2i64),
        (29, 21, 6),
        (38, 21, 1),
        (51, 21, 7),
        (56, 20, 0),
        (60, 20, 2),
        (79, 24, 3),
        (99, 21, 6),
    ] {
        let r = row(&table, item_id);
        let (route, ok) = native_item_command_damage_route_for_type(r[0x0d], get16(r, 0x0e));
        assert!(
            ok && route.item_type as i64 == item_type && route.command_id == command_id && !route.consumes_source,
            "item {item_id} command-damage route = {route:?}, {ok}"
        );
    }
}

/// Go：`TestTrackedRelocationItemRow`
#[test]
fn tracked_relocation_item_row() {
    let Some(table) = tracked_table() else { return };
    let r = row(&table, 101);
    let (route, ok) = native_item_relocation_route_for_type(r[0x0d], get16(r, 0x0e));
    assert!(
        ok && route.row_word == 1
            && route.command_id == 23
            && route.required_identity == 24
            && route.required_max_mp == 20
            && !route.consumes_source,
        "item 101 relocation route = {route:?}, {ok}"
    );
}

/// Go：`TestLoadNativeItemEffectRowPrefixRejectsNonConsecutiveID`
#[test]
fn load_native_item_effect_row_prefix_rejects_non_consecutive_id() {
    let path = temp_dir("item-rows").join("rows.json");
    std::fs::write(&path, r#"[{"id":1,"raw":"0000000000000000000000000000000000000000000000"}]"#).unwrap();
    assert!(load_native_item_effect_row_prefix(&path.to_string_lossy()).is_err(), "expected non-consecutive id error");
}
