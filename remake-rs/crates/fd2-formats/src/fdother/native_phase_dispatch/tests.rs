//! 移植自 `native_phase_dispatch_test.go`。Go 測試以 `*[]string` 記錄呼叫順序，這裡用
//! `RefCell<Vec<String>>`；callback 改寫記錄則經 `&[Cell<u8>]`。

use crate::fdother::*;
use std::cell::{Cell, RefCell};

// Go: TestFindNativePhaseDispatchCandidatesUsesRawGates
#[test]
fn find_native_phase_dispatch_candidates_uses_raw_gates() {
    let mut raw = vec![0u8; 3 * 0x50];
    raw[6] = 1;
    raw[0x50 + 6] = 1;
    raw[0x50 + 5] = 0x80;
    raw[2 * 0x50 + 6] = 1;
    raw[2 * 0x50 + 0x26] = 1;
    let got = find_native_phase_dispatch_candidates(&raw, 3).unwrap();
    assert_eq!(got, [NativePhaseDispatchCandidate { unit_index: 0, selector_arg: 1 }]);
}

// Go: TestFindNativePhaseDispatchCandidatesRejectsTruncatedInput
#[test]
fn find_native_phase_dispatch_candidates_rejects_truncated_input() {
    assert!(find_native_phase_dispatch_candidates(&[0u8; 0x50 - 1], 1).is_err());
}

// Go: TestFindNativePhaseDispatchCandidatesAllowsZeroCountWithoutStorage
#[test]
fn find_native_phase_dispatch_candidates_allows_zero_count_without_storage() {
    assert_eq!(find_native_phase_dispatch_candidates(&[], 0), Ok(Vec::new()));
}

// Go: TestPlanNativePhaseUnitScansPreservesTwoZeroPassesAndSignedGate
#[test]
fn plan_native_phase_unit_scans_preserves_two_zero_passes_and_signed_gate() {
    let mut raw = vec![0u8; 5 * 0x50];
    raw[6] = 1;
    raw[0x50 + 6] = 0;
    raw[2 * 0x50 + 6] = 0;
    raw[3 * 0x50 + 6] = 0;
    raw[3 * 0x50 + 5] = 1;
    raw[4 * 0x50 + 6] = 0;
    raw[4 * 0x50 + 0x26] = 1;
    let mut scores = vec![NativePhasePreselectionScores::default(); 5];
    scores[1] = NativePhasePreselectionScores { spell_candidate_score_53c23: 5, item_command_score_53c33: 6 };
    scores[2] = NativePhasePreselectionScores { spell_candidate_score_53c23: -1, item_command_score_53c33: 5 };

    let plan = plan_native_phase_unit_scans(&raw, 5, &scores).unwrap();
    assert_eq!(plan.selector_one, [NativePhaseDispatchCandidate { unit_index: 0, selector_arg: 1 }]);
    assert_eq!(
        plan.zero_preselection,
        [
            NativePhaseZeroPreselection { unit_index: 1, selector_arg: 0, run_unit_action: true },
            NativePhaseZeroPreselection { unit_index: 2, selector_arg: 0, run_unit_action: false },
        ]
    );
    let second: Vec<i64> = plan.zero_second_pass.iter().map(|c| c.unit_index).collect();
    assert_eq!(second, [1, 2], "zero second pass");
}

// Go: TestPlanNativePhaseUnitScansRequiresScoreProvenance
#[test]
fn plan_native_phase_unit_scans_requires_score_provenance() {
    let raw = vec![0u8; 0x50];
    assert!(plan_native_phase_unit_scans(&raw, 1, &[]).is_err(), "expected missing preselection scores to fail closed");
}

// Go: TestPlanNativePhaseUnitScansDoesNotRequireUnusedZeroScores
#[test]
fn plan_native_phase_unit_scans_does_not_require_unused_zero_scores() {
    let mut raw = vec![0u8; 0x50];
    raw[6] = 1;
    let plan = plan_native_phase_unit_scans(&raw, 1, &[]).unwrap();
    assert!(
        plan.selector_one.len() == 1 && plan.zero_preselection.is_empty() && plan.zero_second_pass.is_empty(),
        "selector-one-only plan = {plan:?}"
    );
}

/// Go `completeNativePhaseCallbacks`。
fn complete_native_phase_callbacks(calls: &RefCell<Vec<String>>) -> NativePhaseScanCallbacks<'_> {
    NativePhaseScanCallbacks {
        preselect: Some(Box::new(move |unit| {
            calls.borrow_mut().push(format!("preselect:{unit}"));
            Ok(NativePhasePreselectionScores::default())
        })),
        run_unit_action: Some(Box::new(move |unit, selector| {
            calls.borrow_mut().push(format!("action:{unit}:{selector}"));
            Ok(NATIVE_NO_GLOBAL_EVENT_ID)
        })),
        run_global_event: Some(Box::new(move |event_id, unit| {
            calls.borrow_mut().push(format!("global:{event_id}:{unit}"));
            Ok(())
        })),
        chapter_index: Some(Box::new(move || {
            calls.borrow_mut().push("chapter-index".to_string());
            Ok(3)
        })),
        run_chapter_event: Some(Box::new(move |chapter, unit| {
            calls.borrow_mut().push(format!("chapter:{chapter}:{unit}"));
            Ok(())
        })),
        pending_code: Some(Box::new(move || {
            calls.borrow_mut().push("pending".to_string());
            Ok(0)
        })),
    }
}

fn actions(calls: &RefCell<Vec<String>>) -> Vec<String> {
    calls.borrow().iter().filter(|c| c.starts_with("action:")).cloned().collect()
}

// Go: TestExecuteNativePhaseUnitScansRunsTailForEveryRecord
#[test]
fn execute_native_phase_unit_scans_runs_tail_for_every_record() {
    let mut raw = vec![0u8; 2 * 0x50];
    raw[6] = 1;
    raw[0x50 + 6] = 2; // 三個 pass 都不合格
    let records = Cell::from_mut(raw.as_mut_slice()).as_slice_of_cells();
    let calls = RefCell::new(Vec::new());
    let mut callbacks = complete_native_phase_callbacks(&calls);
    callbacks.run_unit_action = Some(Box::new(|unit, selector| {
        calls.borrow_mut().push(format!("action:{unit}:{selector}"));
        Ok(7)
    }));

    let result = execute_native_phase_unit_scans(records, 2, &mut callbacks).unwrap();
    assert!(!result.stopped, "result = {result:?}");
    drop(callbacks);
    let want = [
        "action:0:1",
        "global:7:0",
        "chapter-index",
        "chapter:3:0",
        "pending",
        "chapter-index",
        "chapter:3:1",
        "pending",
        "chapter-index",
        "chapter:3:0",
        "pending",
        "chapter-index",
        "chapter:3:1",
        "pending",
        "chapter-index",
        "chapter:3:0",
        "pending",
        "chapter-index",
        "chapter:3:1",
        "pending",
    ];
    assert_eq!(*calls.borrow(), want);
}

// Go: TestExecuteNativePhaseUnitScansRereadsBitSevenBeforeSecondPass
#[test]
fn execute_native_phase_unit_scans_rereads_bit_seven_before_second_pass() {
    let mut raw = vec![0u8; 0x50];
    raw[6] = 0;
    let records = Cell::from_mut(raw.as_mut_slice()).as_slice_of_cells();
    let calls = RefCell::new(Vec::new());
    let mut callbacks = complete_native_phase_callbacks(&calls);
    callbacks.preselect = Some(Box::new(|unit| {
        calls.borrow_mut().push(format!("preselect:{unit}"));
        Ok(NativePhasePreselectionScores { spell_candidate_score_53c23: 6, ..Default::default() })
    }));
    callbacks.run_unit_action = Some(Box::new(|unit, selector| {
        calls.borrow_mut().push(format!("action:{unit}:{selector}"));
        let flag = &records[(unit * 0x50 + 5) as usize];
        flag.set(flag.get() | 0x80);
        Ok(NATIVE_NO_GLOBAL_EVENT_ID)
    }));

    execute_native_phase_unit_scans(records, 1, &mut callbacks).unwrap();
    drop(callbacks);
    assert_eq!(actions(&calls), ["action:0:0"]);
}

// Go: TestExecuteNativePhaseUnitScansLetsLowScoreReachSecondPass
#[test]
fn execute_native_phase_unit_scans_lets_low_score_reach_second_pass() {
    let mut raw = vec![0u8; 0x50];
    raw[6] = 0;
    let records = Cell::from_mut(raw.as_mut_slice()).as_slice_of_cells();
    let calls = RefCell::new(Vec::new());
    let mut callbacks = complete_native_phase_callbacks(&calls);
    callbacks.preselect = Some(Box::new(|unit| {
        calls.borrow_mut().push(format!("preselect:{unit}"));
        Ok(NativePhasePreselectionScores { spell_candidate_score_53c23: 5, item_command_score_53c33: 5 })
    }));

    execute_native_phase_unit_scans(records, 1, &mut callbacks).unwrap();
    drop(callbacks);
    assert_eq!(actions(&calls), ["action:0:0"]);
}

// Go: TestExecuteNativePhaseUnitScansChecksPendingAfterChapterEvent
#[test]
fn execute_native_phase_unit_scans_checks_pending_after_chapter_event() {
    let mut raw = vec![0u8; 2 * 0x50];
    raw[6] = 1;
    raw[0x50 + 6] = 1;
    let records = Cell::from_mut(raw.as_mut_slice()).as_slice_of_cells();
    let calls = RefCell::new(Vec::new());
    let pending = Cell::new(0);
    let mut callbacks = complete_native_phase_callbacks(&calls);
    callbacks.run_global_event = Some(Box::new(|event_id, unit| {
        calls.borrow_mut().push(format!("global:{event_id}:{unit}"));
        pending.set(1);
        Ok(())
    }));
    callbacks.run_unit_action = Some(Box::new(|unit, selector| {
        calls.borrow_mut().push(format!("action:{unit}:{selector}"));
        Ok(9)
    }));
    callbacks.pending_code = Some(Box::new(|| {
        calls.borrow_mut().push("pending".to_string());
        Ok(pending.get())
    }));

    let result = execute_native_phase_unit_scans(records, 2, &mut callbacks).unwrap();
    drop(callbacks);
    assert_eq!(
        result,
        NativePhaseScanResult {
            stopped: true,
            pass: NativePhaseScanPass::SELECTOR_ONE,
            unit_index: 0,
            pending_code: 1
        }
    );
    assert_eq!(*calls.borrow(), ["action:0:1", "global:9:0", "chapter-index", "chapter:3:0", "pending"]);
}

// Go: TestExecuteNativePhaseUnitScansFailsClosedOnTableBounds
#[test]
fn execute_native_phase_unit_scans_fails_closed_on_table_bounds() {
    let mut raw = vec![0u8; 0x50];
    raw[6] = 1;
    let records = Cell::from_mut(raw.as_mut_slice()).as_slice_of_cells();
    let calls = RefCell::new(Vec::new());
    let mut callbacks = complete_native_phase_callbacks(&calls);
    callbacks.run_unit_action = Some(Box::new(|_, _| Ok(90)));
    assert!(
        execute_native_phase_unit_scans(records, 1, &mut callbacks).is_err(),
        "expected out-of-range global event index to fail closed"
    );

    let mut callbacks = complete_native_phase_callbacks(&calls);
    callbacks.chapter_index = Some(Box::new(|| Ok(30)));
    assert!(
        execute_native_phase_unit_scans(records, 1, &mut callbacks).is_err(),
        "expected out-of-range chapter index to fail closed"
    );
}

// Go: TestExecuteNativePhaseUnitScansRequiresEveryNativeTailCallback
#[test]
fn execute_native_phase_unit_scans_requires_every_native_tail_callback() {
    let mut raw = vec![0u8; 0x50];
    let records = Cell::from_mut(raw.as_mut_slice()).as_slice_of_cells();
    let calls = RefCell::new(Vec::new());
    let mut callbacks = complete_native_phase_callbacks(&calls);
    callbacks.run_chapter_event = None;
    assert!(
        execute_native_phase_unit_scans(records, 1, &mut callbacks).is_err(),
        "expected missing chapter callback to fail closed"
    );
}
