//! 移植自 `remake/internal/battle/native_item_hit_ev_step_test.go`。

use super::*;
use crate::test_support::{get16, put16};

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestNativeType12AddsHITEVAndRetainsSourceContract`
#[test]
fn native_type12_adds_hitev_and_retains_source_contract() {
    let mut records = vec![0u8; 2 * RS];
    put16(&mut records, 0x4c, 90);
    put16(&mut records, 0x4e, 5);
    records[RS + 0x24] = 3;
    put16(&mut records, RS + 0x4c, 80);
    put16(&mut records, RS + 0x4e, 10);
    let (route, ok) = native_item_hitev_step_route_for_type(12);
    assert!(ok && !route.consumes_source, "type 12 route = {route:?}, {ok}");
    let (result, next_rng, _) = apply_native_item_hitev_step(&mut records, &[0, 1], route, 0).unwrap();
    assert!(
        result.len() == 2 && result[0].processed && !result[1].processed && next_rng != 0,
        "result={result:?} rng={next_rng:#x}"
    );
    assert_eq!(get16(&records, 0x4c), 105, "HIT");
    assert_eq!(get16(&records, 0x4e), 20, "EV");
    assert_eq!(get16(&records, RS + 0x4c), 80, "marked target HIT changed");
}
