//! 移植自 `remake/internal/battle/native_ai_mode11_test.go`。

use super::*;

/// Go：`TestSelectNativeAIMode11TransactionPreservesIndependentGates`
#[test]
fn select_native_ai_mode11_transaction_preserves_independent_gates() {
    let tests = [
        ("both below", 5, 5, NATIVE_AI_MODE11_NO_FIRST, NATIVE_AI_MODE11_CALL14121),
        ("command only", 6, 5, NATIVE_AI_MODE11_CALL15311, NATIVE_AI_MODE11_CALL14121),
        ("physical only", 5, 6, NATIVE_AI_MODE11_NO_FIRST, NATIVE_AI_MODE11_CALL1548_E),
        ("both", 6, 6, NATIVE_AI_MODE11_CALL15311, NATIVE_AI_MODE11_CALL1548_E),
    ];
    for (name, score_c23, score_c4f, first, second) in tests {
        let got = select_native_ai_mode11_transaction(score_c23, score_c4f, true, true).unwrap();
        assert!(
            got.first_route == first && got.second_route == second,
            "{name}: transaction={got:?} want first={first:?} second={second:?}"
        );
    }
}

/// Go：`TestSelectNativeAIMode11TransactionFailsClosedWithoutRawScore`
#[test]
fn select_native_ai_mode11_transaction_fails_closed_without_raw_score() {
    assert!(
        select_native_ai_mode11_transaction(6, 6, false, true).is_err(),
        "missing raw command score unexpectedly accepted"
    );
    assert!(
        select_native_ai_mode11_transaction(6, 6, true, false).is_err(),
        "missing raw physical score unexpectedly accepted"
    );
}

/// Go：`TestNativeAIMode11StagesPreserveNativeOrder`
#[test]
fn native_ai_mode11_stages_preserve_native_order() {
    let tx = select_native_ai_mode11_transaction(6, 5, true, true).unwrap();
    let mut got = Vec::new();
    let mut execute = |stage: NativeAIMode11Stage| {
        got.push(stage);
        Ok(())
    };
    execute_native_ai_mode11_transaction(tx, Some(&mut execute)).unwrap();
    let want = vec![
        NativeAIMode11Stage { ordinal: 1, route: NATIVE_AI_MODE11_CALL15311 },
        NativeAIMode11Stage { ordinal: 2, route: NATIVE_AI_MODE11_CALL14121 },
    ];
    assert!(got == want, "stages={got:?}, want {want:?}");
}

/// Go：`TestNativeAIMode11StageFailureStopsLaterRoute`
#[test]
fn native_ai_mode11_stage_failure_stops_later_route() {
    let tx = select_native_ai_mode11_transaction(6, 6, true, true).unwrap();
    let mut called = 0;
    let mut execute = |stage: NativeAIMode11Stage| {
        called += 1;
        if stage.ordinal == 1 {
            return Err(battle_err!("presentation owner unavailable"));
        }
        Ok(())
    };
    assert!(
        execute_native_ai_mode11_transaction(tx, Some(&mut execute)).is_err(),
        "failed first stage unexpectedly accepted"
    );
    assert!(called == 1, "later stage executed after failure: called={called}");
}

/// Go：`TestNativeAIMode11StagesRejectInvalidRoute`
#[test]
fn native_ai_mode11_stages_reject_invalid_route() {
    let tx =
        NativeAIMode11Transaction { first_route: NATIVE_AI_MODE11_CALL14121, second_route: NATIVE_AI_MODE11_CALL14121 };
    assert!(tx.stages().is_err(), "invalid first route unexpectedly accepted");
    let mut execute = |_: NativeAIMode11Stage| Ok(());
    let tx = NativeAIMode11Transaction { second_route: NATIVE_AI_MODE11_NO_FIRST, ..Default::default() };
    assert!(
        execute_native_ai_mode11_transaction(tx, Some(&mut execute)).is_err(),
        "invalid second route unexpectedly accepted"
    );
}
