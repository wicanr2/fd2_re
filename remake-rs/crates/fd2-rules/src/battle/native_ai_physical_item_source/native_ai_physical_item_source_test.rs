//! 移植自 `remake/internal/battle/native_ai_physical_item_source_test.go`。

use super::*;
use crate::battle::native_ai_runtime::d_test_support::*;

const ROW: usize = NATIVE_ITEM_EFFECT_ROW_SIZE as usize;

/// Go：`TestResolveNativeAIPhysicalItemSourceMatches14237LookupChain`
#[test]
fn resolve_native_ai_physical_item_source_matches14237_lookup_chain() {
    let mut record = vec![0u8; RS];
    record[0x0a] = 0x40;
    record[0x0b] = 1;
    record[0x0a + 2] = 0x40;
    record[0x0b + 2] = 0x90; // high item is excluded by 0x1B83D(unit,0)
    let mut rows = vec![0u8; 2 * ROW];
    rows[ROW + 0x0b] = 0x12;
    rows[ROW + 0x0c] = 0x34;

    let (mut source, found) = resolve_native_ai_physical_item_source(&record, &rows).unwrap();
    assert!(found, "source={source:?} found={found}");
    assert!(
        source.slot == 0
            && source.item_id == 1
            && source.raw_geometry_byte0_b == 0x12
            && source.raw_geometry_byte0_c == 0x34
            && source.row.len() == ROW,
        "source={source:?}"
    );
    source.row[0x0b] = 0xff;
    assert!(rows[ROW + 0x0b] != 0xff, "item row snapshot aliases caller table");
}

/// Go：`TestResolveNativeAIPhysicalItemSourceKeepsNativeNotFoundAndBoundsClosed`
#[test]
fn resolve_native_ai_physical_item_source_keeps_native_not_found_and_bounds_closed() {
    let mut record = vec![0u8; RS];
    record[0x0a] = 0x40;
    record[0x0b] = 0x80;
    let got = resolve_native_ai_physical_item_source(&record, &[0u8; ROW]);
    assert!(matches!(&got, Ok((source, false)) if source.row.is_empty()), "high-item source={got:?}");

    record[0x0b] = 1;
    assert!(
        resolve_native_ai_physical_item_source(&record, &[0u8; ROW]).is_err(),
        "missing item row unexpectedly accepted"
    );
    assert!(
        resolve_native_ai_physical_item_source(&record[..RS - 1], &[0u8; ROW]).is_err(),
        "short runtime record unexpectedly accepted"
    );
}
