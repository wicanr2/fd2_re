//! 移植自 `remake/internal/battle/native_item_command_damage_test.go`。

use super::*;

fn book_with(id: usize, damage: i64, hit: i64) -> Vec<NativeCommandRecord> {
    let mut book: Vec<NativeCommandRecord> =
        (0..NATIVE_COMMAND_RECORD_COUNT).map(|id| NativeCommandRecord { id, ..Default::default() }).collect();
    book[id] = NativeCommandRecord { id: id as i64, damage, hit, ..Default::default() };
    book
}

fn target(class_id: i64) -> Unit {
    Unit { class_id, hp: 200, max_hp: 200, ..Default::default() }
}

/// Go：`TestNativeType21ReusesCommandDamageWithoutConsumingSource`
#[test]
fn native_type21_reuses_command_damage_without_consuming_source() {
    let (route, ok) = native_item_command_damage_route_for_type(21, 6);
    assert!(
        ok && route.command_id == 6 && route.presentation == 0x1cac7 && !route.consumes_source,
        "route = {route:?}, {ok}"
    );
    let book = book_with(6, 100, 100);
    let mut targets = vec![target(3), target(3)];
    let (results, _) =
        apply_native_item_command_damage(&mut targets, route, &book, &BTreeMap::from([(3, 10)]), 1).unwrap();
    assert!(
        results.len() == 2 && results[0].hit && results[1].hit && targets[0].hp < 200 && targets[1].hp < 200,
        "results={results:?} hp={}/{}",
        targets[0].hp,
        targets[1].hp
    );
}

/// Go：`TestNativeTypes20And24UseTenFramePresentationAndCommandDamage`
#[test]
fn native_types20_and24_use_ten_frame_presentation_and_command_damage() {
    for typ in [20u8, 24] {
        let (route, ok) = native_item_command_damage_route_for_type(typ, 2);
        assert!(
            ok && route.command_id == 2 && route.presentation == 0x1cd17 && !route.consumes_source,
            "type {typ} route = {route:?}, {ok}"
        );
    }
}

/// Go：`TestNativeType21PreflightsAllTargets`
#[test]
fn native_type21_preflights_all_targets() {
    let (route, _) = native_item_command_damage_route_for_type(21, 1);
    let book = book_with(1, 100, 100);
    let mut targets = vec![target(3), target(4)];
    assert!(
        apply_native_item_command_damage(&mut targets, route, &book, &BTreeMap::from([(3, 10)]), 1).is_err(),
        "missing resistance unexpectedly accepted"
    );
    assert_eq!(targets[0].hp, 200, "first target mutated before preflight");
}

/// Go：`TestNativeType21RejectsNonCommandWord`
#[test]
fn native_type21_rejects_non_command_word() {
    assert!(
        !native_item_command_damage_route_for_type(21, NATIVE_COMMAND_RECORD_COUNT as u16).1,
        "out-of-range command word unexpectedly accepted"
    );
    assert!(!native_item_command_damage_route_for_type(19, 1).1, "wrong item type unexpectedly accepted");
}
