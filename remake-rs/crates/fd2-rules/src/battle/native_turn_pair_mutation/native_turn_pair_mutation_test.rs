//! 移植自 `remake/internal/battle/native_turn_pair_mutation_test.go`。

use super::*;
use crate::test_support::remake_asset;

fn event79_fixture() -> Option<(State, NativeTurnEvent)> {
    let path = remake_asset("scenarios/ch29.json")?;
    let sc = load_scenario(&path).unwrap();
    let event = sc.native_turn_events.iter().rfind(|c| c.event_id == 79).cloned().unwrap_or_default();
    let mut units = vec![Unit { group: 8, has_native_record_byte5: true, ..Default::default() }; 13];
    for unit in &mut units[10..13] {
        unit.group = 1;
    }
    let mut st =
        State { units, native_round_counter: 12, has_native_turn_event_control_state: true, ..Default::default() };
    for row in &mut st.native_turn_event_controls {
        row.turn = 0xff;
    }
    st.native_turn_event_controls[2] = NativeTurnEventControl { turn: 12, event_id: 79, raw_camp: 0 };
    st.native_event_state[21] = 10;
    Some((st, event))
}

/// Go：`TestEvent79ConsumesOneRNGStepAndMarksTwoCyclicGroup1Slots`
#[test]
fn event79_consumes_one_rng_step_and_marks_two_cyclic_group1_slots() {
    let Some((mut st, event)) = event79_fixture() else { return };
    let seed = 0xffffu16;
    let want_rng = fdother::native_rng_step(seed);
    let want = [10 + (want_rng % 3) as i64, 10 + (want_rng as i64 + 1) % 3];
    let (next, targets) = apply_native_turn_event79(Some(&mut st), &event, seed).unwrap();
    assert!(
        next == want_rng
            && targets == want
            && st.native_turn_event_controls[2] == NativeTurnEventControl { turn: 13, event_id: 79, raw_camp: 0 },
        "event79 rng={next:#x}/{want_rng:#x} targets={targets:?}/{want:?} row={:?}",
        st.native_turn_event_controls[2]
    );
    for index in 10..13 {
        let marked = st.units[index].native_record_byte5 & 0x80 != 0;
        assert_eq!(marked, index as i64 == want[0] || index as i64 == want[1], "event79 slot{index} targets={want:?}");
    }
}

/// Go：`TestEvent79MissingThirdGroupRecordFailsAtomically`
///
/// Go 失敗時另回傳原 RNG 值；Rust 的 `Result` 只回錯誤，RNG 由呼叫端保留，這一項由型別保證。
#[test]
fn event79_missing_third_group_record_fails_atomically() {
    let Some((mut st, event)) = event79_fixture() else { return };
    st.units[12].has_native_record_byte5 = false;
    let before_rows = st.native_turn_event_controls;
    let before_units = st.units.clone();
    let err = apply_native_turn_event79(Some(&mut st), &event, 7);
    assert!(
        err.is_err() && st.native_turn_event_controls == before_rows && st.units == before_units,
        "event79 atomic failure err={err:?} rows={:?}",
        st.native_turn_event_controls[2]
    );
}
