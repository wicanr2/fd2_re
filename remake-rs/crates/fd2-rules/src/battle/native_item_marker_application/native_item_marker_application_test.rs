//! 移植自 `remake/internal/battle/native_item_marker_application_test.go`。

use super::*;
use crate::test_support::{get16, put16};
use fd2_formats::native_rng::native_rng_step;

const RS: usize = NATIVE_RECORD_SIZE as usize;

fn find_marker_application_seed(succeeds: bool) -> u16 {
    (0..=0xffffu16)
        .find(|&seed| (native_rng_step(seed) % 100 < 50) == succeeds)
        .unwrap_or_else(|| panic!("no RNG seed found for succeeds={succeeds}"))
}

/// Go：`TestNativeType14And22MarkerRoutesRetainSourceContract`
#[test]
fn native_type14_and22_marker_routes_retain_source_contract() {
    for (typ, code, marker) in [(14u8, 27i64, 0x26i64), (22, 22, 0x27)] {
        let mut records = vec![0u8; RS];
        put16(&mut records, 0x40, 100);
        let (route, ok) = native_item_marker_application_route_for_type(typ);
        assert!(
            ok && route.presentation_code == code
                && route.marker_offset == marker
                && route.hp_damage_base == 10
                && !route.consumes_source,
            "type {typ:#x} route = {route:?}, {ok}"
        );
        let (result, next_rng, _) =
            apply_native_item_marker_application(&mut records, &[0], route, find_marker_application_seed(true))
                .unwrap();
        assert!(
            result.len() == 1
                && result[0].applied
                && result[0].damage.rolled >= 9
                && result[0].damage.rolled <= 10
                && next_rng != 0
                && get16(&records, 0x40) == (100 - result[0].damage.actual) as u16
                && records[marker as usize] != 0,
            "type {typ:#x} result={result:?} rng={next_rng:#x} hp={} marker={}",
            get16(&records, 0x40),
            records[marker as usize]
        );
    }
}

/// Go：`TestNativeMarkerApplicationFailedGateUsesOneRNGAndNoMutation`
#[test]
fn native_marker_application_failed_gate_uses_one_rng_and_no_mutation() {
    let mut records = vec![0u8; RS];
    put16(&mut records, 0x40, 100);
    let (route, _) = native_item_marker_application_route_for_type(14);
    let seed = find_marker_application_seed(false);
    let (result, next_rng, _) = apply_native_item_marker_application(&mut records, &[0], route, seed).unwrap();
    assert!(
        !result[0].applied && next_rng == native_rng_step(seed) && get16(&records, 0x40) == 100 && records[0x26] == 0,
        "failed gate result={result:?} rng={next_rng:#x} hp={} marker={}",
        get16(&records, 0x40),
        records[0x26]
    );
}
