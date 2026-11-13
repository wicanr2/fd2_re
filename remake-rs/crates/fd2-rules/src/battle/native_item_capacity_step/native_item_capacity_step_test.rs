//! 移植自 `remake/internal/battle/native_item_capacity_step_test.go`。

use super::*;
use crate::test_support::{get16, put16};

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestNativeType17And18IncreaseMaxCapacityAndConsumeSource`
#[test]
fn native_type17_and18_increase_max_capacity_and_consume_source() {
    for (typ, stat, offset, start) in
        [(17u8, NATIVE_ITEM_MAX_HP, 0x42usize, 100u16), (18, NATIVE_ITEM_MAX_MP, 0x46, 50)]
    {
        let mut records = vec![0u8; 2 * RS];
        put16(&mut records, offset, start);
        for slot in 0..8 {
            (records[RS + 0x0a + slot * 2], records[RS + 0x0b + slot * 2]) = (0, 0xd0 + slot as u8);
        }
        let (route, ok) = native_item_capacity_step_route_for_type(typ, 20);
        assert!(ok && route.stat == stat && route.consumes_source, "type {typ:#x} route = {route:?}, {ok}");
        let got = apply_native_item_capacity_step(&mut records, 0, route, 1, 0).unwrap();
        assert!(
            got == start + 20 && get16(&records, offset) == got,
            "type {typ:#x} result={got} raw={}",
            get16(&records, offset)
        );
        let source = &records[RS..];
        assert!(
            source[0x0b] == 0xd1 && source[0x18] == 0x80,
            "type {typ:#x} source not consumed: {:02x?}",
            &source[0x0a..0x1a]
        );
    }
}

/// Go：`TestNativeType19IncreasesMVAndPreservesEXP`
#[test]
fn native_type19_increases_mv_and_preserves_exp() {
    let mut records = vec![0u8; 2 * RS];
    (records[0x3b], records[0x3c]) = (7, 99);
    (records[RS + 0x0a], records[RS + 0x0b]) = (0, 96);
    let (route, ok) = native_item_capacity_step_route_for_type(19, 1);
    assert!(ok && route.stat == NATIVE_ITEM_MV, "type 19 route = {route:?}, {ok}");
    let got = apply_native_item_capacity_step(&mut records, 0, route, 1, 0).unwrap();
    assert!(
        got == 8 && records[0x3b] == 8 && records[0x3c] == 99 && records[RS + 0x18] == 0x80,
        "result={got} mv={} exp={} inventory={:02x?}",
        records[0x3b],
        records[0x3c],
        &records[RS + 0x0a..RS + 0x1a]
    );
}

/// Go：`TestNativeCapacityStepPreflightsSourceBeforeMutation`
#[test]
fn native_capacity_step_preflights_source_before_mutation() {
    let mut records = vec![0u8; RS];
    put16(&mut records, 0x42, 100);
    let (route, _) = native_item_capacity_step_route_for_type(17, 20);
    assert!(
        apply_native_item_capacity_step(&mut records, 0, route, 0, 8).is_err(),
        "invalid source slot unexpectedly accepted"
    );
    assert_eq!(get16(&records, 0x42), 100, "invalid source removal partially mutated max HP");
}

/// Rust 補充：路由驗證失敗的訊息與 Go `%#v` 逐字相同（期望字串由 Go 1.22 實際輸出抄錄）。
#[test]
fn native_capacity_step_invalid_route_message_matches_go() {
    let mut records = vec![0u8; RS];
    let (mut route, _) = native_item_capacity_step_route_for_type(19, 1);
    route.field_offset = 0x3c;
    let err = apply_native_item_capacity_step(&mut records, 0, route, 0, 0).unwrap_err();
    assert_eq!(
        err.to_string(),
        "native item capacity route is invalid: battle.NativeItemCapacityStepRoute{ItemType:0x13, Stat:\"mv\", \
         FieldOffset:60, Amount:0x1, PresentationCode:19, ConsumesSource:true}"
    );
}
