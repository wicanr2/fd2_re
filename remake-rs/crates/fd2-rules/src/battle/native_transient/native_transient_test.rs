//! 移植自 `remake/internal/battle/native_transient_test.go`。

use super::*;
use crate::test_support::{get16, put16};

/// Go `nativeTransientPhaseTestState`：單位依序放進 `units`。
fn native_transient_phase_test_state(units: Vec<Unit>) -> State {
    let mut records = vec![NativeRuntimeRecordState::default(); units.len()];
    for (slot, unit) in units.iter().enumerate() {
        let raw = &mut records[slot].raw;
        raw[5] = unit.native_record_byte5;
        raw[6] = unit.native_record_byte6;
        raw[0x22..0x28].copy_from_slice(&unit.native_transient);
        put16(raw, 0x40, unit.hp as u16);
        put16(raw, 0x42, unit.max_hp as u16);
    }
    State { units, native_runtime_records: records, has_native_runtime_unit_projection: true, ..Default::default() }
}

/// Go `nativeTransientPhaseTestUnit`
fn native_transient_phase_test_unit(selector: u8, hp: i64, max_hp: i64, duration: u8) -> Unit {
    Unit {
        hp,
        max_hp,
        native_record_byte5: 0,
        has_native_record_byte5: true,
        native_record_byte6: selector,
        has_native_record_byte6: true,
        native_transient: [0, 0, 0, duration, 0, 0],
        ..Default::default()
    }
}

/// Go：`TestNativeTransientStorageIsBoundedByOriginalOffsets`
#[test]
fn native_transient_storage_is_bounded_by_original_offsets() {
    let mut u = Unit::default();
    assert!(
        !u.set_native_transient_duration(0x21, 3) && !u.set_native_transient_duration(0x28, 3),
        "out-of-range raw offsets must be rejected"
    );
    assert!(
        u.set_native_transient_duration(0x22, 3) && u.set_native_transient_duration(0x27, 1),
        "recovered transient range must be writable"
    );
    assert_eq!(u.native_transient_duration(0x22), (3, true), "+0x22");
    assert!(!u.native_transient_duration(0x28).1, "out-of-range read must fail closed");
}

/// Go：`TestTickNativeTransientsUsesRawGates`
#[test]
fn tick_native_transients_uses_raw_gates() {
    let active = Unit {
        camp: ENEMY,
        on_field: false,
        hp: 0,
        has_native_record_byte5: true,
        native_record_byte6: 7,
        has_native_record_byte6: true,
        native_transient: [1, 2, 0, 1, 0, 3],
        ..Default::default()
    };
    let other_selector = Unit {
        has_native_record_byte5: true,
        native_record_byte6: 8,
        has_native_record_byte6: true,
        native_transient: [1, 1, 0, 0, 0, 0],
        ..Default::default()
    };
    let blocked = Unit {
        native_record_byte5: 1,
        has_native_record_byte5: true,
        native_record_byte6: 7,
        has_native_record_byte6: true,
        native_transient: [1, 1, 0, 0, 0, 0],
        ..Default::default()
    };
    let missing_raw =
        Unit { camp: OWN, on_field: true, hp: 1, native_transient: [1, 1, 0, 0, 0, 0], ..Default::default() };
    let mut st = State { units: vec![active, other_selector, blocked, missing_raw], ..Default::default() };

    let expired = st.tick_native_transients_raw(7);
    assert_eq!(st.units[0].native_transient, [0, 1, 0, 0, 0, 2], "active sweep");
    assert!(
        expired.len() == 2 && expired[0].unit == Some(0) && expired[0].offset == 0x22 && expired[1].offset == 0x25,
        "expiry = {expired:?}, want +0x22/+0x25 for active unit"
    );
    for (i, u) in st.units.iter().enumerate().skip(1) {
        assert_eq!(
            u.native_transient,
            [1, 1, 0, 0, 0, 0],
            "unit {i} failing the native raw gate must not be decremented"
        );
    }
}

/// Go：`TestTickNativeTransientsCampWrapperFailsClosed`
#[test]
fn tick_native_transients_camp_wrapper_fails_closed() {
    let u = Unit {
        native_record_byte6: 1,
        has_native_record_byte6: true,
        has_native_record_byte5: true,
        native_transient: [1, 0, 0, 0, 0, 0],
        ..Default::default()
    };
    let st = State { units: vec![u], ..Default::default() };
    assert!(
        st.tick_native_transients(OWN).is_empty() && st.units[0].native_transient[0] == 1,
        "normalized Camp must not be guessed as the native selector"
    );
}

/// Go：`TestAdvanceNativeTransientPhaseAppliesDamageBeforeCountdown`
#[test]
fn advance_native_transient_phase_applies_damage_before_countdown() {
    let mut st = native_transient_phase_test_state(vec![native_transient_phase_test_unit(2, 100, 100, 2)]);
    let result = st.advance_native_transient_phase_raw(2).unwrap();
    let unit = &st.units[0];
    let raw = &st.native_runtime_records[0].raw;
    assert!(
        result.damage.len() == 1
            && result.damage[0].slot == 0
            && result.damage[0].amount == 10
            && result.damage[0].before == 100
            && result.damage[0].after == 90
            && unit.hp == 90
            && get16(raw, 0x40) == 90
            && unit.native_transient[3] == 1
            && raw[0x25] == 1
            && result.inactive_slots.is_empty()
            && result.expired.is_empty(),
        "result={result:?} HP={} duration={} rawHP={} rawDuration={}",
        unit.hp,
        unit.native_transient[3],
        get16(raw, 0x40),
        raw[0x25]
    );
}

/// Go：`TestAdvanceNativeTransientPhaseFatalDamageMarksInactiveAndSkipsCountdown`（`t.Run` 子測試改成迴圈）
#[test]
fn advance_native_transient_phase_fatal_damage_marks_inactive_and_skips_countdown() {
    for (name, hp) in [("exact", 10), ("clamped", 4)] {
        let mut unit = native_transient_phase_test_unit(2, hp, 100, 2);
        unit.native_record_byte5 = 0x80;
        let mut st = native_transient_phase_test_state(vec![unit]);
        let result = st.advance_native_transient_phase_raw(2).unwrap();
        let unit = &st.units[0];
        let raw = &st.native_runtime_records[0].raw;
        assert!(
            unit.hp == 0
                && unit.native_record_byte5 == 1
                && raw[5] == 1
                && unit.native_transient[3] == 2
                && raw[0x25] == 2
                && result.inactive_slots == [0]
                && result.expired.is_empty(),
            "{name}: result={result:?} HP={} byte5={:02x} duration={}",
            unit.hp,
            unit.native_record_byte5,
            unit.native_transient[3]
        );
    }
}

/// Go：`TestAdvanceNativeTransientPhaseZeroDamageStillCountsDown`
#[test]
fn advance_native_transient_phase_zero_damage_still_counts_down() {
    let mut st = native_transient_phase_test_state(vec![native_transient_phase_test_unit(2, 5, 9, 1)]);
    let result = st.advance_native_transient_phase_raw(2).unwrap();
    let unit = &st.units[0];
    assert!(
        result.damage.len() == 1
            && result.damage[0].amount == 0
            && unit.hp == 5
            && unit.native_transient[3] == 0
            && result.expired.len() == 1
            && result.expired[0].offset == 0x25,
        "result={result:?} HP={} duration={}",
        unit.hp,
        unit.native_transient[3]
    );
}

/// Go：`TestAdvanceNativeTransientPhaseUsesSelectorAndInactiveGates`
#[test]
fn advance_native_transient_phase_uses_selector_and_inactive_gates() {
    let selected = native_transient_phase_test_unit(2, 50, 100, 2);
    let other = native_transient_phase_test_unit(1, 50, 100, 2);
    let mut inactive = native_transient_phase_test_unit(2, 50, 100, 2);
    inactive.native_record_byte5 = 1;
    let mut st = native_transient_phase_test_state(vec![selected, other, inactive]);
    st.advance_native_transient_phase_raw(2).unwrap();
    let u = &st.units;
    assert!(
        u[0].hp == 40
            && u[0].native_transient[3] == 1
            && u[1].hp == 50
            && u[1].native_transient[3] == 2
            && u[2].hp == 50
            && u[2].native_transient[3] == 2,
        "selected={}/{} other={}/{} inactive={}/{}",
        u[0].hp,
        u[0].native_transient[3],
        u[1].hp,
        u[1].native_transient[3],
        u[2].hp,
        u[2].native_transient[3]
    );
}

/// Go：`TestAdvanceNativeTransientPhaseMarksMultipleDeathsWithoutDispatchState`
#[test]
fn advance_native_transient_phase_marks_multiple_deaths_without_dispatch_state() {
    let mut st = native_transient_phase_test_state(vec![
        native_transient_phase_test_unit(2, 10, 100, 2),
        native_transient_phase_test_unit(2, 1, 20, 3),
        native_transient_phase_test_unit(2, 30, 100, 2),
    ]);
    let result = st.advance_native_transient_phase_raw(2).unwrap();
    let u = &st.units;
    assert!(
        result.inactive_slots == [0, 1]
            && result.damage.len() == 3
            && u[0].native_record_byte5 == 1
            && u[1].native_record_byte5 == 1
            && u[2].hp == 20
            && u[0].native_transient[3] == 2
            && u[1].native_transient[3] == 3
            && u[2].native_transient[3] == 1,
        "result={result:?} units={u:?}"
    );
}

/// Go：`TestAdvanceNativeTransientPhaseFailsClosedOnRawMismatch`
#[test]
fn advance_native_transient_phase_fails_closed_on_raw_mismatch() {
    let mut st = native_transient_phase_test_state(vec![native_transient_phase_test_unit(2, 50, 100, 2)]);
    st.native_runtime_records[0].raw[0x40] += 1;
    let before_unit = st.units[0].clone();
    let before_raw = st.native_runtime_records[0];
    assert!(st.advance_native_transient_phase_raw(2).is_err(), "raw HP mismatch was accepted");
    assert!(
        st.units[0] == before_unit && st.native_runtime_records[0] == before_raw,
        "rejected phase mutated typed or raw state"
    );
}
