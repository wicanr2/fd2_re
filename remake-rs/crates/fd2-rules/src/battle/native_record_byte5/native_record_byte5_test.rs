//! 移植自 `remake/internal/battle/native_record_byte5_test.go`。

use super::*;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestSetNativeRecordByte5OneOverwritesWholeByte`
#[test]
fn set_native_record_byte5_one_overwrites_whole_byte() {
    let mut records = vec![0u8; 2 * RS];
    records[5] = 0x80;
    records[RS + 5] = 0xff;
    set_native_record_byte5_one(&mut records, 1).unwrap();
    assert!(records[5] == 0x80 && records[RS + 5] == 1, "bytes={:#x}/{:#x}", records[5], records[RS + 5]);
}

/// Go：`TestSetNativeRecordByte5OneRejectsBounds`
#[test]
fn set_native_record_byte5_one_rejects_bounds() {
    assert!(set_native_record_byte5_one(&mut [0u8; RS], 1).is_err(), "out-of-range record unexpectedly accepted");
    assert!(set_native_record_byte5_one(&mut [0u8; RS], -1).is_err(), "negative record unexpectedly accepted");
}

/// Go：`TestHPWritersMirrorRawByte5OnlyWhenProvenanceExists`
#[test]
fn hp_writers_mirror_raw_byte5_only_when_provenance_exists() {
    let mut dead =
        Unit { hp: 5, max_hp: 5, has_native_record_byte5: true, native_record_byte5: 0x80, ..Default::default() };
    dead.apply_hp_damage(9);
    assert!(
        dead.hp == 0 && dead.native_record_byte5 == 1,
        "native death writer hp={} byte5={:#x}",
        dead.hp,
        dead.native_record_byte5
    );
    dead.restore_native_hp();
    assert!(
        dead.hp == 5 && dead.native_record_byte5 == 0,
        "native revive writer hp={} byte5={:#x}",
        dead.hp,
        dead.native_record_byte5
    );
    let mut legacy = Unit { hp: 5, max_hp: 5, native_record_byte5: 0x80, ..Default::default() };
    legacy.apply_hp_damage(9);
    assert!(
        legacy.hp == 0 && legacy.native_record_byte5 == 0x80,
        "legacy path fabricated raw provenance hp={} byte5={:#x}",
        legacy.hp,
        legacy.native_record_byte5
    );
}

/// Go：`TestMarkNativeUnitInactiveLetsDepartedEnemyStopBlockingResult`
///
/// 第四章事件 12 讓第 2 群組改成模式 7 撤退；抵達目的座標時 0x32975 把 +5 寫成 1。原版預設勝負
/// 規則 0x205BE 只計 +5 bit0 清零的 raw +6==0 列；引擎的 result 看 on_field，離場必須一起清掉。
#[test]
fn mark_native_unit_inactive_lets_departed_enemy_stop_blocking_result() {
    let hero = Unit {
        name: "索爾".into(),
        camp: OWN,
        on_field: true,
        hp: 10,
        max_hp: 10,
        has_native_record_byte5: true,
        ..Default::default()
    };
    let fleeing = Unit {
        camp: ENEMY,
        on_field: true,
        hp: 36,
        max_hp: 36,
        native_record_byte5: 0x80,
        has_native_record_byte5: true,
        ..Default::default()
    };
    let mut st = State {
        units: vec![hero, fleeing],
        native_runtime_records: vec![NativeRuntimeRecordState { raw: [0; 80], selector_key: 0, selector_slot: 0 }; 2],
        has_native_runtime_unit_projection: true,
        ..Default::default()
    };
    st.native_runtime_records[1].raw[5] = 0x80;
    assert_eq!(st.result("索爾"), "", "before departure");
    st.mark_native_unit_inactive(1).unwrap();
    let u = &st.units[1];
    assert!(!u.on_field && u.native_record_byte5 == 1 && u.has_native_record_byte5, "departed unit {u:?}");
    assert_eq!(st.native_runtime_records[1].raw[5], 1, "raw record +5 must be the whole byte 1");
    assert!(u.alive(), "departure must not change HP");
    assert_eq!(st.result("索爾"), "win", "after departure");
}

/// Go：`TestMarkNativeUnitInactiveRejectsForeignUnit`
#[test]
fn mark_native_unit_inactive_rejects_out_of_range_index() {
    let mut st =
        State { units: vec![Unit { camp: ENEMY, on_field: true, hp: 1, ..Default::default() }], ..Default::default() };
    assert!(st.mark_native_unit_inactive(1).is_err(), "out-of-range index unexpectedly accepted");
    assert!(st.units[0].on_field, "rejected departure must not write any unit");
}
