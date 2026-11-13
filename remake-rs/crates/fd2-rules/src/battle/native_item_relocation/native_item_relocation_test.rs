//! 移植自 `remake/internal/battle/native_item_relocation_test.go`。

use super::*;
use crate::test_support::{get16, put16, remake_asset};

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestNativeCommand23RelocationUsesSpellGatesAndIsAtomic`
#[test]
fn native_command23_relocation_uses_spell_gates_and_is_atomic() {
    let Some(path) = remake_asset("spells.json") else { return };
    let book = load_native_command_records(&path).unwrap();
    for mp in [19u16, 30] {
        let mut records = vec![0u8; 3 * RS];
        records[0x08] = 7; // The spell handler does not require item caster24.
        put16(&mut records, 0x44, mp);
        (records[RS], records[RS + 1], records[RS + 0x20], records[RS + 0x21]) = (1, 0, 9, 7);
        let before = records.clone();
        let got = apply_native_command23_relocation(&mut records, 0, &[1, 2], 8, 7, &book);
        if mp < 20 {
            assert!(got.is_err() && before == records, "insufficient spell MP changed records");
            continue;
        }
        assert!(
            matches!(&got, Ok(g) if g.accumulator_delta == 370 && g.remaining_mp == 10)
                && records[RS] == 8
                && records[RS + 1] == 7
                && records[2 * RS..] == before[2 * RS..],
            "spell relocation: {got:?}"
        );
    }
}

fn native_relocation_command_book() -> Vec<NativeCommandRecord> {
    let mut book: Vec<NativeCommandRecord> =
        (0..NATIVE_COMMAND_RECORD_COUNT).map(|id| NativeCommandRecord { id, ..Default::default() }).collect();
    book[23].mp_cost = 20;
    book
}

/// Go：`TestNativeType23RelocatesFirstTargetAndWrapsMP`
#[test]
fn native_type23_relocates_first_target_and_wraps_mp() {
    let mut records = vec![0u8; 3 * RS];
    records[0x08] = 24;
    put16(&mut records, 0x44, 10);
    put16(&mut records, 0x46, 20);
    (records[RS], records[RS + 1], records[RS + 0x20], records[RS + 0x21]) = (4, 5, 9, 7);
    (records[2 * RS], records[2 * RS + 1]) = (8, 9);
    let (route, ok) = native_item_relocation_route_for_type(23, 1);
    assert!(
        ok && route.command_id == 23 && route.presentation == 0x22253 && !route.consumes_source,
        "route = {route:?}, {ok}"
    );
    let got = apply_native_item_relocation(&mut records, 0, &[1, 2], 12, 13, route, &native_relocation_command_book())
        .unwrap();
    assert!(
        got.target_unit == 1
            && got.from_x == 4
            && got.from_y == 5
            && got.remaining_mp == 0xfff6
            && got.accumulator_delta == 370,
        "result = {got:?}"
    );
    assert!(
        get16(&records, 0x44) == 0xfff6
            && records[RS] == 12
            && records[RS + 1] == 13
            && records[2 * RS] == 8
            && records[2 * RS + 1] == 9,
        "actorMP={:#x} target={},{} other={},{}",
        get16(&records, 0x44),
        records[RS],
        records[RS + 1],
        records[2 * RS],
        records[2 * RS + 1]
    );
}

/// Go：`TestNativeType23PreflightsIdentityAndMaxMP`
#[test]
fn native_type23_preflights_identity_and_max_mp() {
    for (name, identity, max_mp, want_target) in
        [("identity", 23u8, 20u16, 1u8), ("max_mp", 24, 19, 1), ("target", 24, 20, 2)]
    {
        let mut records = vec![0u8; 2 * RS];
        records[0x08] = identity;
        put16(&mut records, 0x44, 30);
        put16(&mut records, 0x46, max_mp);
        let (route, _) = native_item_relocation_route_for_type(23, 1);
        assert!(
            apply_native_item_relocation(
                &mut records,
                0,
                &[want_target],
                9,
                10,
                route,
                &native_relocation_command_book()
            )
            .is_err(),
            "{name}: invalid relocation unexpectedly accepted"
        );
        assert_eq!(get16(&records, 0x44), 30, "{name}: invalid relocation mutated MP");
    }
}
