//! 移植自 `remake/internal/battle/native_command25_test.go`。

use super::*;

fn native_command25_book() -> Vec<NativeCommandRecord> {
    let mut book: Vec<NativeCommandRecord> =
        (0..36).map(|id| NativeCommandRecord { id, ..Default::default() }).collect();
    book[25] = NativeCommandRecord {
        id: 25,
        selection_mode: 1,
        effect_mode: 0,
        mp_cost: 2,
        target_code: 1,
        ..Default::default()
    };
    book
}

fn actor_unit() -> Unit {
    Unit { camp: OWN, on_field: true, hp: 10, mp: 5, x: 0, y: 0, ..Default::default() }
}

fn confirmed_unit() -> Unit {
    Unit {
        camp: OWN,
        on_field: true,
        hp: 10,
        acted: true,
        x: 1,
        y: 0,
        native_record_byte5: 0x80,
        has_native_record_byte5: true,
        ..Default::default()
    }
}

/// Go：`TestExecuteNativeCommand25ClearsOnlyFinalActedTargets`
#[test]
fn execute_native_command25_clears_only_final_acted_targets() {
    let other = Unit {
        camp: OWN,
        on_field: true,
        hp: 10,
        acted: false,
        x: 2,
        y: 0,
        has_native_record_byte5: true,
        ..Default::default()
    };
    let mut st = State {
        w: 3,
        h: 1,
        units: vec![actor_unit(), confirmed_unit(), other],
        native_composition_event_bytes: vec![0; 3],
        native_command_book: native_command25_book(),
        ..Default::default()
    };
    let (actor, confirmed, other) = (0, 1, 2);
    let got = st.execute_native_command25(Some(actor), Some(confirmed));
    assert!(
        matches!(&got, Ok(g) if g.len() == 1 && g[0].target == Some(confirmed) && g[0].cleared),
        "result = {got:?}; want confirmed acted clear"
    );
    let (a, c, o) = (&st.units[actor], &st.units[confirmed], &st.units[other]);
    assert!(
        c.acted && c.native_record_byte5 == 0 && a.acted && !o.acted && a.mp == 3,
        "post state actor={a:?} confirmed={c:?} other={o:?}"
    );
}

/// Go：`TestExecuteNativeCommand25FailsBeforeMPOnInvalidConfirmation`
#[test]
fn execute_native_command25_fails_before_mp_on_invalid_confirmation() {
    let mut st = State {
        w: 2,
        h: 1,
        units: vec![actor_unit(), confirmed_unit()],
        native_command_book: native_command25_book(),
        ..Default::default()
    };
    let err = st.execute_native_command25(Some(0), Some(1));
    let (a, c) = (&st.units[0], &st.units[1]);
    assert!(
        err.is_err() && a.mp == 5 && !a.acted && c.acted && c.native_record_byte5 == 0x80,
        "invalid native target data must fail before mutation: actor={a:?} confirmed={c:?} err={err:?}"
    );
}
