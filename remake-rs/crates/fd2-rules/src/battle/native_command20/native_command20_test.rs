//! 移植自 `remake/internal/battle/native_command20_test.go`。

use super::*;

fn native_command_clear_restore_book(id: i64) -> Vec<NativeCommandRecord> {
    let mut book: Vec<NativeCommandRecord> =
        (0..36).map(|i| NativeCommandRecord { id: i, ..Default::default() }).collect();
    book[10] = NativeCommandRecord { id: 10, damage: 100, ..Default::default() };
    book[id as usize] =
        NativeCommandRecord { id, selection_mode: 1, effect_mode: 0, mp_cost: 2, target_code: 1, ..Default::default() };
    book
}

fn actor_unit() -> Unit {
    Unit { camp: OWN, on_field: true, hp: 10, mp: 5, x: 0, y: 0, ..Default::default() }
}

fn clear_restore_state(target: Unit, id: i64) -> State {
    State {
        w: 2,
        h: 1,
        units: vec![actor_unit(), target],
        native_composition_event_bytes: vec![0; 2],
        native_command_book: native_command_clear_restore_book(id),
        ..Default::default()
    }
}

/// Go：`TestExecuteNativeCommandClearRestoreUsesRecordTenAndRawFlag`
#[test]
fn execute_native_command_clear_restore_uses_record_ten_and_raw_flag() {
    let target = Unit {
        camp: OWN,
        on_field: true,
        hp: 1,
        max_hp: 100,
        x: 1,
        y: 0,
        native_transient: [0, 0, 0, 3, 0, 0],
        ..Default::default()
    };
    let mut st = clear_restore_state(target, 20);
    let mut rng = GoMathRand::new(2);
    let got = st.execute_native_command_clear_restore(Some(0), Some(1), 20, Some(&mut rng));
    let ok = matches!(&got, Ok(g) if g.len() == 1 && g[0].cleared && g[0].offset == 0x25
        && (90..=99).contains(&g[0].restore.rolled));
    assert!(ok, "clear/restore = {got:?}");
    let got = got.unwrap();
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        target.native_transient[3] == 0 && target.hp == 1 + got[0].restore.actual && actor.mp == 3 && actor.acted,
        "post clear/restore actor={actor:?} target={target:?}"
    );
}

/// Go：`TestExecuteNativeCommandClearRestoreConsumesCommandWhenFlagEmpty`
#[test]
fn execute_native_command_clear_restore_consumes_command_when_flag_empty() {
    let target = Unit { camp: OWN, on_field: true, hp: 10, max_hp: 20, x: 1, y: 0, ..Default::default() };
    let mut st = clear_restore_state(target, 21);
    let mut rng = GoMathRand::new(1);
    let got = st.execute_native_command_clear_restore(Some(0), Some(1), 21, Some(&mut rng));
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        matches!(&got, Ok(g) if g.len() == 1 && !g[0].cleared) && target.hp == 10 && actor.mp == 3 && actor.acted,
        "empty-flag route = {got:?} actor={actor:?} target={target:?}"
    );
}

/// Go：`TestApplyNativeCommandRestoreClampsStateButReportsRolledAmount`
#[test]
fn apply_native_command_restore_clamps_state_but_reports_rolled_amount() {
    let mut target = Unit { hp: 95, max_hp: 100, ..Default::default() };
    let mut rng = GoMathRand::new(3);
    let got = apply_native_command_restore(Some(&mut target), 100, Some(&mut rng));
    assert!(
        matches!(got, Ok(r) if (90..=99).contains(&r.rolled) && r.actual == 5) && target.hp == 100,
        "restore cap = {got:?} target={target:?}"
    );
}

/// Go：`TestExecuteNativeCommandClearRestorePreflightsBeforeMPDebit`
#[test]
fn execute_native_command_clear_restore_preflights_before_mp_debit() {
    let target = Unit {
        camp: OWN,
        on_field: true,
        hp: 11,
        max_hp: 10,
        x: 1,
        y: 0,
        native_transient: [0, 0, 0, 3, 0, 0],
        ..Default::default()
    };
    let mut st = clear_restore_state(target, 20);
    let mut rng = GoMathRand::new(2);
    assert!(
        st.execute_native_command_clear_restore(Some(0), Some(1), 20, Some(&mut rng)).is_err(),
        "invalid target HP state accepted"
    );
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        actor.mp == 5 && !actor.acted && target.native_transient[3] == 3 && target.hp == 11,
        "failed preflight mutated actor={actor:?} target={target:?}"
    );
}
