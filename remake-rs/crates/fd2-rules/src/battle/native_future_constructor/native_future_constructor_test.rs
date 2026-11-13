//! 移植自 `remake/internal/battle/native_future_constructor_test.go`。

use super::*;
use crate::test_support::{put16, remake_asset};

fn table(branch: &str, record: Vec<u8>, aux_record: Vec<u8>) -> NativeConstructorTable {
    NativeConstructorTable { branch: branch.into(), index: 0, record, aux_record }
}

/// Go：`TestDecodeNativeFutureConstructorBaseHighClass`
#[test]
fn decode_native_future_constructor_base_high_class() {
    let t = table("high_class", vec![4, 5, 0x34, 0x12, 6, 7, 8, 9, 10, 0], vec![]);
    let got = decode_native_future_constructor_base(Some(&t), 3).unwrap();
    let want = NativeFutureConstructorBase {
        record_race: 4,
        record_class: 5,
        base_ap: 21,
        base_dp: 24,
        base_dx: 27,
        mobility: 10,
        max_hp: 0x369c,
        max_mp: 18,
    };
    assert_eq!(got, want, "high constructor");
}

/// Go：`TestDecodeNativeFutureConstructorBaseLowerClass`
#[test]
fn decode_native_future_constructor_base_lower_class() {
    let mut record = vec![0u8; 24];
    (record[0], record[1], record[7]) = (1, 2, 9);
    (record[3], record[4]) = (100, 0);
    (record[5], record[6]) = (20, 0);
    (record[0x12], record[0x14], record[0x16]) = (10, 11, 12);
    let t = table("lower_class", record, vec![2, 0, 3, 0, 4, 0, 5, 0, 6, 0, 0]);
    let got = decode_native_future_constructor_base(Some(&t), 4).unwrap();
    let want = NativeFutureConstructorBase {
        record_race: 1,
        record_class: 2,
        base_ap: 18,
        base_dp: 23,
        base_dx: 28,
        mobility: 9,
        max_hp: 115,
        max_mp: 38,
    };
    assert_eq!(got, want, "lower constructor");
}

/// Go：`TestDecodeNativeFutureConstructorBasePreservesLevelZeroWrap`
#[test]
fn decode_native_future_constructor_base_preserves_level_zero_wrap() {
    let mut record = vec![0u8; 24];
    (record[3], record[5]) = (4, 3);
    let t = table("lower_class", record, vec![0, 0, 0, 0, 0, 0, 5, 0, 6, 0, 0]);
    let got = decode_native_future_constructor_base(Some(&t), 0).unwrap();
    assert!(got.max_hp == 0xffff && got.max_mp == 0xfffd, "level-zero wrap HP/MP={:#x}/{:#x}", got.max_hp, got.max_mp);
}

/// Go：`TestAllMapUnitsMatchNativeFutureConstructorProjection`
#[test]
fn all_map_units_match_native_future_constructor_projection() {
    let mut matches = 0;
    for map_index in 0..33 {
        let Some(path) = remake_asset(&format!("maps/map{map_index}/map{map_index}_units.json")) else { return };
        let st = load(&path).unwrap_or_else(|e| panic!("map{map_index} load: {e}"));
        for (unit_index, unit) in st.units.iter().enumerate() {
            let Some(constructor) = &unit.native_constructor else { continue };
            let got = decode_native_future_constructor_base(Some(constructor), unit.lv as u8)
                .unwrap_or_else(|e| panic!("map{map_index} unit{unit_index} constructor: {e}"));
            assert!(
                unit.has_native_record_race
                    && unit.native_record_race == got.record_race
                    && unit.has_native_record_class
                    && unit.native_record_class == got.record_class
                    && unit.has_native_record_word42
                    && unit.native_record_word42 == got.max_hp
                    && unit.has_native_record_word46
                    && unit.native_record_word46 == got.max_mp,
                "map{map_index} unit{unit_index} projection race/class/HP/MP={}/{}/{}/{} want {}/{}/{}/{}",
                unit.native_record_race,
                unit.native_record_class,
                unit.native_record_word42,
                unit.native_record_word46,
                got.record_race,
                got.record_class,
                got.max_hp,
                got.max_mp
            );
            matches += 1;
        }
    }
    assert!(matches > 0, "no map unit carried native constructor provenance");
}

/// Go：`TestMaterializeNativeFutureConstructorClearsTransientsAndRecomputes`
#[test]
fn materialize_native_future_constructor_clears_transients_and_recomputes() {
    let mut unit = Unit {
        lv: 2,
        native_record_byte6: 1,
        has_native_record_byte6: true,
        native_constructor: Some(table("high_class", vec![4, 5, 10, 0, 3, 6, 7, 8, 9, 0], vec![])),
        inventory: vec![0],
        equipped: vec![true],
        inventory_slots: vec![0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
        native_transient: [1, 2, 3, 4, 5, 6],
        ap: 999,
        dp: 999,
        hit: 999,
        ev: 999,
        ..Default::default()
    };
    let mut rows = vec![0u8; NATIVE_ITEM_EFFECT_ROW_SIZE as usize];
    put16(&mut rows, 1, 3);
    put16(&mut rows, 5, 4);
    put16(&mut rows, 3, 5);
    put16(&mut rows, 7, 6);
    materialize_native_future_constructor(Some(&mut unit), &rows).unwrap();
    assert!(
        unit.ap == 15
            && unit.dp == 18
            && unit.hit == 21
            && unit.ev == 22
            && unit.hp == 20
            && unit.max_hp == 20
            && unit.mp == 6
            && unit.max_mp == 6
            && unit.mv == 9
            && unit.dx == 16
            && unit.exp == 255.0
            && unit.native_record_race == 4
            && unit.native_record_class == 5
            && unit.native_transient == [0; 6]
            && unit.native_record_byte5 == 0
            && unit.equipment_base_set,
        "materialized constructor={unit:?}"
    );
}

/// Go：`TestMaterializeNativeFutureConstructorFailsAtomically`
#[test]
fn materialize_native_future_constructor_fails_atomically() {
    let mut unit = Unit {
        lv: 1,
        native_record_byte6: 2,
        has_native_record_byte6: true,
        native_constructor: Some(table("high_class", vec![1, 2, 1, 0, 1, 1, 1, 1, 1, 0], vec![])),
        inventory: vec![2],
        equipped: vec![true],
        inventory_slots: vec![2, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
        ap: 77,
        ..Default::default()
    };
    let before = unit.clone();
    assert!(
        materialize_native_future_constructor(Some(&mut unit), &vec![0u8; NATIVE_ITEM_EFFECT_ROW_SIZE as usize])
            .is_err(),
        "missing selected item row accepted"
    );
    assert_eq!(unit, before, "failed constructor mutated unit");
}
