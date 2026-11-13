//! 移植自 `native_unit_mode_branch_test.go`。

use crate::fdother::NativeUnitModeAction::*;
use crate::fdother::*;

// Go: TestPlanNativeUnitMode0PreservesNestedFallbacks
#[test]
fn plan_native_unit_mode0_preserves_nested_fallbacks() {
    assert_eq!(plan_native_unit_mode0(true, false, false), [Call14EF0], "successful mode 0 actions");
    assert_eq!(plan_native_unit_mode0(false, true, false), [Call14EF0, Call14121], "blocked-search mode 0 actions");
    assert_eq!(
        plan_native_unit_mode0(false, false, false),
        [Call14EF0, Call14121, Call13E9C, Call13FD4],
        "nested mode 0 fallback actions"
    );
    assert_eq!(
        plan_native_unit_mode0(false, false, true),
        [Call14EF0, Call14121, Call13E9C],
        "nearest-coordinate mode 0 actions"
    );
}

// Go: TestPlanNativeUnitMode1UsesSharedRecoveryWithoutNearestFallback
#[test]
fn plan_native_unit_mode1_uses_shared_recovery_without_nearest_fallback() {
    assert_eq!(plan_native_unit_mode1(true, false), [Call14EF0], "successful mode 1 actions");
    assert_eq!(plan_native_unit_mode1(false, true), [Call14EF0, Call14121], "blocked-search mode 1 actions");
    assert_eq!(plan_native_unit_mode1(false, false), [Call14EF0, Call14121, Call13FD4], "mode 1 shared recovery");
}

// Go: TestPlanNativeUnitMode3KeepsLookupIndexAndFallback
#[test]
fn plan_native_unit_mode3_keeps_lookup_index_and_fallback() {
    assert_eq!(
        plan_native_unit_mode3(false, -1, false, false, false),
        [Call14EF0, Call12C60, Call14121, Call13E9C, Call13FD4],
        "mode 3 lookup fallback actions"
    );
    assert_eq!(
        plan_native_unit_mode3(false, 4, false, false, false),
        [Call14EF0, Call12C60, Call12D7B, Call14B78, Call13FD4, Write51A83Zero],
        "mode 3 found-index actions"
    );
}

// Go: TestPlanNativeUnitMode4And10ReuseMovementRecovery
#[test]
fn plan_native_unit_mode4_and10_reuse_movement_recovery() {
    let want = vec![Write51A83Zero, Call12D7B, Call14B78, Call13FD4];
    assert_eq!(plan_native_unit_mode4(false), want, "mode 4 actions");
    let mut want10 = vec![Call14EF0];
    want10.extend(want);
    assert_eq!(plan_native_unit_mode10(false, false), want10, "mode 10 actions");
}

// Go: TestPlanNativeUnitMode5PreservesArrivalWrites
#[test]
fn plan_native_unit_mode5_preserves_arrival_writes() {
    let mut input = NativeUnitMode5Inputs {
        event_lookup_succeeded: true,
        arrived_at_event_coordinate: true,
        event_record_type: 0,
        ..Default::default()
    };
    assert_eq!(
        plan_native_unit_mode5(input),
        [
            Call14EF0,
            Write51A83Zero,
            Call12D7B,
            Call15DF3,
            Call14B78,
            Call13FD4,
            WriteUnit31And32,
            Call1BB8C,
            Write53AD5One,
            Call25B45,
            Call12263,
            WriteUnit34Mode7,
        ],
        "mode 5 arrival actions"
    );
    input.event_record_type = 2;
    let got = plan_native_unit_mode5(input);
    assert!(
        !got.iter().any(|a| *a == WriteUnit31And32 || *a == Call1BB8C),
        "mode 5 type>=2 unexpectedly writes/calls: {got:?}"
    );
}

// Go: TestPlanNativeUnitMode5StateAndLookupFallback
#[test]
fn plan_native_unit_mode5_state_and_lookup_fallback() {
    for input in [
        NativeUnitMode5Inputs { event_state_byte: 1, ..Default::default() },
        NativeUnitMode5Inputs { event_lookup_succeeded: false, ..Default::default() },
    ] {
        assert_eq!(
            plan_native_unit_mode5(input),
            [Call14EF0, Write51A83Zero, Call12D7B, Call14121, Call13E9C, Call13FD4],
            "mode 5 fallback actions"
        );
    }
}

// Go: TestPlanNativeUnitMode7And9PreserveCoordinateGates
#[test]
fn plan_native_unit_mode7_and9_preserve_coordinate_gates() {
    assert_eq!(
        plan_native_unit_mode7(false, true),
        [Write51A83Zero, Call12D7B, Call14B78, Call13FD4, Call32975],
        "mode 7 actions"
    );
    assert_eq!(
        plan_native_unit_mode9(-1, false, false, false, false),
        [Call12C60, Call14EF0, Call14121, Call13E9C, Call13FD4],
        "mode 9 general fallback actions"
    );
    assert_eq!(
        plan_native_unit_mode9(6, true, true, true, false),
        [Call12C60, Call12D7B, Call14B78, Call13FD4],
        "mode 9 found-index actions"
    );
}

// Go: TestPlanNativeUnitMode2PreservesSharedRecovery
#[test]
fn plan_native_unit_mode2_preserves_shared_recovery() {
    assert_eq!(plan_native_unit_mode2(false), [Call14EF0, Call14237, Call13FD4], "mode 2 actions");
    assert_eq!(plan_native_unit_mode2(true), [Call14EF0], "successful mode 2 actions");
}

// Go: TestPlanNativeUnitMode11UsesIndependentSignedGates
#[test]
fn plan_native_unit_mode11_uses_independent_signed_gates() {
    assert_eq!(
        plan_native_unit_mode11(6, 5, false),
        [Call1598A, Call15311, Call14237, Call14121, Call13FD4],
        "mode 11 command branch"
    );
    assert_eq!(plan_native_unit_mode11(-1, 6, false), [Call1598A, Call14237, Call1548E], "mode 11 physical branch");
}

// Go: TestPlanNativeUnitMode11StopsAfterBlockedSearchSuccess
#[test]
fn plan_native_unit_mode11_stops_after_blocked_search_success() {
    assert_eq!(plan_native_unit_mode11(5, 5, true), [Call1598A, Call14237, Call14121], "mode 11 blocked-search branch");
}
