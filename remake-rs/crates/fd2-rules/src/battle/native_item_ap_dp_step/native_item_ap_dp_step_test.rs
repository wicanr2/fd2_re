//! 移植自 `remake/internal/battle/native_item_ap_dp_step_test.go`。

use super::*;
use crate::test_support::{get16, put16};

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestNativeType15And16RetainedAPDPRoutes`
#[test]
fn native_type15_and16_retained_apdp_routes() {
    for (typ, stat, marker, word, current, want) in [
        (15u8, NATIVE_ITEM_DERIVED_DP, 0x23usize, 0x4ausize, 100u16, 116u16),
        (16, NATIVE_ITEM_DERIVED_AP, 0x22, 0x48, 200, 231),
    ] {
        let mut records = vec![0u8; RS];
        put16(&mut records, word, current);
        let (route, ok) = native_item_apdp_step_route_for_type(typ);
        assert!(ok && route.derived_stat == stat && !route.consumes_source, "type {typ:#x} route = {route:?}, {ok}");
        let (result, next_rng, _) = apply_native_item_apdp_step(&mut records, &[0], route, 0).unwrap();
        assert!(
            result.len() == 1 && result[0].processed && next_rng != 0,
            "type {typ:#x} result={result:?} rng={next_rng:#x}"
        );
        assert_eq!(get16(&records, word), want, "type {typ:#x} updated word");
        assert_ne!(records[marker], 0, "type {typ:#x} marker not written");
    }
}

/// Go：`TestNativeAPDPStepMarkedTargetSkipsRNG`
#[test]
fn native_apdp_step_marked_target_skips_rng() {
    let mut records = vec![0u8; RS];
    records[0x22] = 3;
    put16(&mut records, 0x48, 100);
    let (route, _) = native_item_apdp_step_route_for_type(16);
    let (result, next_rng, _) = apply_native_item_apdp_step(&mut records, &[0], route, 0x1234).unwrap();
    assert!(
        !result[0].processed && next_rng == 0x1234 && get16(&records, 0x48) == 100,
        "marked result={result:?} rng={next_rng:#x} word={}",
        get16(&records, 0x48)
    );
}
