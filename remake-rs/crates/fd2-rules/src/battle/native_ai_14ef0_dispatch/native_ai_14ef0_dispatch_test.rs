//! 移植自 `remake/internal/battle/native_ai_14ef0_dispatch_test.go`。

use super::*;

/// Go `nativeAI14EF0TestInput`。
fn native_ai14_ef0_test_input() -> NativeAI14EF0Input {
    NativeAI14EF0Input {
        has_raw_score_c4_f: true,
        has_raw_score_c23: true,
        has_raw_score_c33: true,
        has_raw_record34: true,
        has_raw_actor_word48: true,
        has_raw_target_word4_a: true,
        has_raw_command_id: true,
        has_raw_command_word: true,
        command_id: 2,
        command_word: 10,
        actor_word48: 20,
        target_word4_a: 5,
        ..Default::default()
    }
}

fn scores(r#in: &mut NativeAI14EF0Input, c4f: i32, c23: i32, c33: i32) {
    (r#in.score_c4_f, r#in.score_c23, r#in.score_c33) = (c4f, c23, c33);
}

/// Go：`TestSelectNativeAI14EF0TailPreservesRawRoutes`
#[test]
fn select_native_ai14_ef0_tail_preserves_raw_routes() {
    type Mutate = fn(&mut NativeAI14EF0Input);
    let tests: Vec<(&str, Mutate, NativeAI14EF0Tail)> = vec![
        ("below threshold", |r| scores(r, 5, 5, 5), NATIVE_AI14_EF0_NO_TAIL),
        ("strict c4f", |r| scores(r, 8, 7, 7), NATIVE_AI14_EF0_CALL1548_E),
        (
            "c4f c23 tie command word",
            |r| {
                scores(r, 8, 8, 7);
                (r.command_id, r.command_word) = (2, 15);
            },
            NATIVE_AI14_EF0_CALL15311,
        ),
        (
            "c4f c23 tie command word too small",
            |r| {
                scores(r, 8, 8, 7);
                (r.command_id, r.command_word) = (2, 14);
            },
            NATIVE_AI14_EF0_CALL1548_E,
        ),
        (
            "c4f c23 tie id high bit clear",
            |r| {
                scores(r, 8, 8, 7);
                (r.command_id, r.record34) = (11, 0);
            },
            NATIVE_AI14_EF0_CALL15311,
        ),
        (
            "c4f c23 tie id high bit set",
            |r| {
                scores(r, 8, 8, 7);
                (r.command_id, r.record34) = (11, 0x40);
            },
            NATIVE_AI14_EF0_CALL1548_E,
        ),
        (
            "c4f c33 tie bit clear",
            |r| {
                scores(r, 8, 7, 8);
                r.record34 = 0;
            },
            NATIVE_AI14_EF0_CALL15055,
        ),
        (
            "c4f c33 tie bit set",
            |r| {
                scores(r, 8, 7, 8);
                r.record34 = 0x40;
            },
            NATIVE_AI14_EF0_CALL1548_E,
        ),
        ("strict c23", |r| scores(r, 7, 9, 9), NATIVE_AI14_EF0_CALL15311),
        ("strict c33", |r| scores(r, 7, 8, 9), NATIVE_AI14_EF0_CALL15055),
        ("remaining three-way tie falls through", |r| scores(r, 8, 8, 8), NATIVE_AI14_EF0_NO_TAIL),
    ];
    for (name, mutate, want) in tests {
        let mut r#in = native_ai14_ef0_test_input();
        mutate(&mut r#in);
        let got = select_native_ai14_ef0_tail(r#in).unwrap_or_else(|e| panic!("{name}: error = {e}"));
        assert!(got == want, "{name}: SelectNativeAI14EF0Tail() = {}, want {}", got.0, want.0);
        let want_accepted = name != "below threshold";
        let decision = plan_native_ai14_ef0_dispatch(r#in);
        assert!(
            matches!(decision, Ok(d) if d.tail == want && d.accepted == want_accepted),
            "{name}: dispatch={decision:?} want tail={} accepted={want_accepted}",
            want.0
        );
    }
}

/// Go：`TestNativeAI14EF0NoTailReturnBoundary`
#[test]
fn native_ai14_ef0_no_tail_return_boundary() {
    for score in [-1, 0, 5, 6, 8] {
        let mut r#in = native_ai14_ef0_test_input();
        scores(&mut r#in, score, score, score);
        let decision = plan_native_ai14_ef0_dispatch(r#in);
        assert!(
            matches!(decision, Ok(d) if d.tail == NATIVE_AI14_EF0_NO_TAIL && d.accepted == (score >= 6)),
            "score={score} decision={decision:?}"
        );
    }
    let mut r#in = native_ai14_ef0_test_input();
    scores(&mut r#in, 8, 8, 8);
    r#in.has_raw_score_c23 = false;
    // Go 在錯誤時另檢查回傳的 decision 不被接受；Rust 的 `Err` 不帶 decision。
    let decision = plan_native_ai14_ef0_dispatch(r#in);
    assert!(decision.is_err(), "missing score admitted: decision={decision:?}");
}

/// Go：`TestSelectNativeAI14EF0TailFailsClosedWithoutRawInputs`
#[test]
fn select_native_ai14_ef0_tail_fails_closed_without_raw_inputs() {
    let mut r#in = native_ai14_ef0_test_input();
    r#in.has_raw_target_word4_a = false;
    assert!(select_native_ai14_ef0_tail(r#in).is_err(), "expected missing raw target word to fail closed");
    let mut r#in = native_ai14_ef0_test_input();
    scores(&mut r#in, 8, 8, 7);
    r#in.has_raw_command_word = false;
    assert!(select_native_ai14_ef0_tail(r#in).is_err(), "expected command tie without 0x4e516 word to fail closed");
}
