//! 移植自 `remake/internal/battle/native_ai_idle_recovery_test.go`。

use super::*;
use crate::battle::native_ai_runtime::d_test_support::*;

fn hp_record(current: u16, maximum: u16) -> Vec<u8> {
    let mut records = vec![0u8; RS];
    put_record16(&mut records, 0, 0x40, current);
    put_record16(&mut records, 0, 0x42, maximum);
    records
}

/// Go：`TestApplyNativeAIIdleRecoveryUsesOneFifthAndCaps`
#[test]
fn apply_native_ai_idle_recovery_uses_one_fifth_and_caps() {
    let mut records = hp_record(70, 100);
    let applied = apply_native_ai_idle_recovery(&mut records, 1, 0).unwrap();
    assert!(
        applied && record16(&records, 0, 0x40) == 90,
        "first recovery = ({applied},{}), want (true,90)",
        record16(&records, 0, 0x40)
    );
    let applied = apply_native_ai_idle_recovery(&mut records, 1, 0).unwrap();
    assert!(
        applied && record16(&records, 0, 0x40) == 100,
        "capped recovery = ({applied},{}), want (true,100)",
        record16(&records, 0, 0x40)
    );
}

/// Go：`TestPlanNativeAIIdleRecoveryIsPureAndPreservesRawDecisionFields`
#[test]
fn plan_native_ai_idle_recovery_is_pure_and_preserves_raw_decision_fields() {
    let mut records = hp_record(70, 100);
    records[0x25] = 0x12;
    records[0x26] = 0x34;
    let want = records.clone();
    let decision = plan_native_ai_idle_recovery(&records, 1, 0).unwrap();
    assert!(
        !decision.accepted
            && decision.current_hp == 70
            && decision.maximum_hp == 100
            && decision.next_hp == 0
            && decision.raw_gate25 == 0x12
            && decision.raw_gate26 == 0x34,
        "decision={decision:?}"
    );
    assert!(records == want, "pure recovery planner mutated raw record");

    (records[0x25], records[0x26]) = (0, 0);
    let decision = plan_native_ai_idle_recovery(&records, 1, 0).unwrap();
    assert!(decision.accepted && decision.next_hp == 90, "accepted decision={decision:?}");
}

/// Go：`TestApplyNativeAIIdleRecoveryHonorsRawGates`
#[test]
fn apply_native_ai_idle_recovery_honors_raw_gates() {
    for offset in [0x25, 0x26] {
        let mut records = hp_record(50, 100);
        records[offset] = 1;
        let applied = apply_native_ai_idle_recovery(&mut records, 1, 0).unwrap();
        assert!(!applied && record16(&records, 0, 0x40) == 50, "gate +{offset:#x} mutated HP");
    }
}

/// Go：`TestApplyNativeAIIdleRecoveryRejectsShortInput`
#[test]
fn apply_native_ai_idle_recovery_rejects_short_input() {
    assert!(apply_native_ai_idle_recovery(&mut [0u8; RS - 1], 1, 0).is_err(), "short record unexpectedly accepted");
}

/// Go：`TestBuildNativeAIIdleRecoveryPresentationPreservesRawSequence`
#[test]
fn build_native_ai_idle_recovery_presentation_preserves_raw_sequence() {
    let decision = NativeAIIdleRecoveryDecision {
        unit: 3,
        accepted: true,
        current_hp: 70,
        maximum_hp: 100,
        next_hp: 90,
        ..Default::default()
    };
    let p = build_native_ai_idle_recovery_presentation(decision).unwrap();
    assert!(
        p.unit == 3
            && p.before_hp == 70
            && p.after_hp == 90
            && p.sample_handle_expr == "[0x53EEC]"
            && p.sample_index == 4
            && p.sample_loop_count == 1
            && p.wait_ticks == [1, 1, 1],
        "presentation header={p:?}"
    );
    assert!(
        p.before_global_write == NativeAIIdleRecoveryGlobalWrite { address: 0x51A83, value: 0 }
            && p.coordinate_call == NativeAIIdleRecoveryCall { address: 0x12D7B, unit: 3 }
            && p.after_global_write == NativeAIIdleRecoveryGlobalWrite { address: 0x51A83, value: 1 },
        "raw calls/writes={:?}/{:?}/{:?}",
        p.before_global_write,
        p.coordinate_call,
        p.after_global_write
    );
    let decode = |mode, tail| NativeAIIdleRecoveryDecode { resource_base: "[0x53A49]+0x8088".to_string(), mode, tail };
    assert!(
        p.first_decode == decode(2, 0xFD) && p.second_decode == decode(0, 0),
        "decode passes={:?}/{:?}",
        p.first_decode,
        p.second_decode
    );
    let want_copy = NativeAIIdleRecoveryCopy {
        source_offset: 0xA0504,
        destination_offset: 0x140,
        source_stride: 0x1C8,
        destination_stride: 0x140,
        rows: 0xC0,
        length: 0x138,
    };
    assert!(
        p.first_copy == want_copy && p.second_copy == want_copy,
        "copy passes={:?}/{:?}",
        p.first_copy,
        p.second_copy
    );
}

/// Go：`TestApplyNativeAIIdleRecoveryWithPresentationCommitsAfterCallback`
///
/// Go 的 callback 直接讀外層 `records` 確認 HP 尚未寫入；Rust 的 `records` 在呼叫期間被可變借用，
/// 改由 callback 收到的 `before_hp`／`after_hp` 與呼叫前後的值檢查同一件事。
#[test]
fn apply_native_ai_idle_recovery_with_presentation_commits_after_callback() {
    let mut records = hp_record(70, 100);
    let mut called = false;
    let mut present = |p: NativeAIIdleRecoveryPresentation| {
        called = true;
        assert!(p.before_hp == 70 && p.after_hp == 90, "callback presentation={p:?}");
        Ok(())
    };
    let applied = apply_native_ai_idle_recovery_with_presentation(&mut records, 1, 0, Some(&mut present));
    assert!(matches!(applied, Ok(true)) && called, "applied={applied:?} called={called}");
    let got = record16(&records, 0, 0x40);
    assert!(got == 90, "committed HP={got}, want 90");
}

/// Go：`TestApplyNativeAIIdleRecoveryWithPresentationFailsClosed`
#[test]
fn apply_native_ai_idle_recovery_with_presentation_fails_closed() {
    let mut records = hp_record(70, 100);
    assert!(
        apply_native_ai_idle_recovery_with_presentation(&mut records, 1, 0, None).is_err(),
        "missing presentation owner unexpectedly accepted"
    );
    let got = record16(&records, 0, 0x40);
    assert!(got == 70, "missing owner changed HP={got}");
    let mut present = |_: NativeAIIdleRecoveryPresentation| Err(battle_err!("renderer unavailable"));
    assert!(
        apply_native_ai_idle_recovery_with_presentation(&mut records, 1, 0, Some(&mut present)).is_err(),
        "callback failure unexpectedly accepted"
    );
    let got = record16(&records, 0, 0x40);
    assert!(got == 70, "callback failure changed HP={got}");
}

/// Go：`TestApplyNativeAIIdleRecoveryWithPresentationRejectsChangedRawRecord`
///
/// Go 的 callback 直接改外層 `records`，函式在 callback 之後重算 decision、發現不同就回滾報錯。
/// Rust 的 `records` 在呼叫期間被可變借用，safe code 的 callback 碰不到同一個切片，這條分支無法
/// 從外部觸發（列入 `KNOWN_DIFFERENCES`）。這裡改測該分支依賴的判定：記錄改變後重算的 decision
/// 與原本不同，且回滾後 HP 回到原值。
#[test]
fn apply_native_ai_idle_recovery_with_presentation_rejects_changed_raw_record() {
    let mut records = hp_record(70, 100);
    let decision = plan_native_ai_idle_recovery(&records, 1, 0).unwrap();
    let before = records.clone();
    put_record16(&mut records, 0, 0x40, 71);
    let current = plan_native_ai_idle_recovery(&records, 1, 0).unwrap();
    assert!(current != decision, "changed raw record accepted: applied decision unchanged");
    records.copy_from_slice(&before);
    let got = record16(&records, 0, 0x40);
    assert!(got == 70, "callback change was not rolled back: {got}");
}
