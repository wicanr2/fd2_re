//! 移植自 `remake/internal/battle/native_command_target_test.go`。
//!
//! Go 以指標比較目標（`got[i] != want[i]`）；Rust 比較 `units` 索引。

use super::*;
use crate::test_support::remake_asset;

/// Go：`TestNativeCompositionBaseFlagsMasksArchiveHighBits`
#[test]
fn native_composition_base_flags_masks_archive_high_bits() {
    let got = native_composition_base_flags(3, 1, &[0xff, 0x40, 0x85]).unwrap();
    assert!(got[0] == 0x1f && got[1] == 0 && got[2] == 5, "composition base flags={got:?}");
}

/// Go：`TestStateNativeCommandBaseFlagsAreRebuiltPerCall`
#[test]
fn state_native_command_base_flags_are_rebuilt_per_call() {
    let st = State { w: 2, h: 1, native_composition_event_bytes: vec![0xE5, 0x7F], ..Default::default() };
    let mut first = st.native_command_base_flags().unwrap();
    first[0] = 0xFF;
    let second = st.native_command_base_flags().unwrap();
    let want = [0x05, 0x1F];
    assert!(second == want, "rebuilt flags={second:?} want {want:?}");
}

/// Go：`TestStateNativeCommandBaseFlagsRejectMissingSource`
///
/// Go 另測 nil `*State` 接收者；Rust 的 `self` 不會是 null，該段不移植。
#[test]
fn state_native_command_base_flags_reject_missing_source() {
    let st = State { w: 1, h: 1, ..Default::default() };
    assert!(st.native_command_base_flags().is_err(), "expected missing composition event bytes to fail closed");
}

/// Go 兩個 runtime-flag 測試的 `unit` 輔助。
fn raw_unit(x: u8, y: u8, selector: u8, byte5: u8) -> Unit {
    Unit {
        native_map_presentation: NativeMapPresentationState { x, y, ..Default::default() },
        has_native_map_presentation: true,
        native_record_byte5: byte5,
        has_native_record_byte5: true,
        native_record_byte6: selector,
        has_native_record_byte6: true,
        ..Default::default()
    }
}

/// Go：`TestNativeCommandRuntimeFlagsMatches145CDWriters`
#[test]
fn native_command_runtime_flags_matches_145cd_writers() {
    let units = vec![raw_unit(1, 1, 1, 0), raw_unit(0, 0, 0, 0), raw_unit(2, 2, 1, 1)];
    let flags = native_command_runtime_flags(3, 3, &[0xff, 0, 0, 0, 0, 0, 0, 0, 0], &units, 0).unwrap();
    let want = [0x1f, 0x80, 0, 0x80, 0x40, 0x80, 0, 0x80, 0];
    for index in 0..want.len() {
        assert!(
            flags[index] == want[index],
            "selector0 flags[{index}]={:#x} want {:#x}; all={flags:?}",
            flags[index],
            want[index]
        );
    }

    let flags = native_command_runtime_flags(3, 3, &[0; 9], &units, 1).unwrap();
    assert!(!(flags[0] != 0x40 || flags[1] != 0x80 || flags[3] != 0x80 || flags[4] != 0), "selector1 flags={flags:?}");
}

/// Go：`TestNativeCommandRuntimeFlagsFailsClosedWithoutRawProvenance`
#[test]
fn native_command_runtime_flags_fails_closed_without_raw_provenance() {
    assert!(
        native_command_runtime_flags(1, 1, &[0], &[Unit::default()], 0).is_err(),
        "runtime flag writer accepted incomplete roster"
    );
}

/// Go：`TestNativeCommandRuntimeFlagsAllowsEmptyRuntimeArray`
#[test]
fn native_command_runtime_flags_allows_empty_runtime_array() {
    let got = native_command_runtime_flags(2, 1, &[0xe5, 0x40], &[], 0).unwrap();
    let want = [5, 0];
    assert!(got == want, "empty-runtime flags={got:?} want {want:?}");
}

/// Go：`TestNativeFutureGroupPlacementMatches10C50OccupancyAndTieOrder`
#[test]
fn native_future_group_placement_matches_10c50_occupancy_and_tie_order() {
    // Both selector classes are marked because 0x10c50 calls 0x145cd with
    // zero and one. The inactive record at (2,0) must not block its cell.
    let units = vec![raw_unit(1, 1, 0, 0), raw_unit(2, 1, 2, 0), raw_unit(2, 0, 1, 1)];
    let got = native_future_group_placement(
        3,
        3,
        &[0; 9],
        &units,
        NativePositionRecord { x_word: 1, y_word: 1, ..Default::default() },
        0,
    )
    .unwrap();
    // Four distance-one candidates surround the occupied origin. Row-major
    // scanning replaces equal distances, so the last one, (1,2), wins.
    let want = Cell { x: 1, y: 2 };
    assert!(got == want, "future placement={got:?} want {want:?}");
}

/// Go：`TestNativeFutureGroupPlacementUsesPositionLowBytesAndRawGate`
#[test]
fn native_future_group_placement_uses_position_low_bytes_and_raw_gate() {
    let got = native_future_group_placement(
        4,
        4,
        &[0; 16],
        &[],
        NativePositionRecord { x_word: 0x1203, y_word: 0x3402, ..Default::default() },
        7,
    )
    .unwrap();
    let want = Cell { x: 3, y: 2 };
    assert!(got == want, "direct future placement={got:?} want {want:?}");
}

/// Go：`TestAppendGroupWithNativePlacementConsumesPerCallGate`
#[test]
fn append_group_with_native_placement_consumes_per_call_gate() {
    let active = Unit {
        x: 1,
        y: 1,
        map_selector_key: 2,
        has_map_selector_key: true,
        native_map_presentation: NativeMapPresentationState { x: 1, y: 1, ..Default::default() },
        has_native_map_presentation: true,
        native_record_byte5: 0,
        has_native_record_byte5: true,
        native_record_byte6: 2,
        has_native_record_byte6: true,
        ..Default::default()
    };
    let pending = || Unit {
        group: 6,
        dir: 0,
        lv: 2,
        map_selector_key: 1,
        has_map_selector_key: true,
        native_record_byte5: 0,
        has_native_record_byte5: true,
        native_record_byte6: 1,
        has_native_record_byte6: true,
        native_position_record: NativePositionRecord { x_word: 1, y_word: 1, ..Default::default() },
        has_native_position_record: true,
        native_constructor: Some(NativeConstructorTable {
            branch: "high_class".into(),
            index: 0,
            record: vec![4, 5, 10, 0, 3, 6, 7, 8, 9, 0],
            ..Default::default()
        }),
        inventory: vec![0],
        equipped: vec![true],
        inventory_slots: vec![0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
        ..Default::default()
    };
    let new_state = |unit: Unit| {
        let mut st =
            State { w: 3, h: 3, roster: vec![unit], native_composition_event_bytes: vec![0; 9], ..Default::default() };
        st.bind_native_future_item_rows(&vec![0; NATIVE_ITEM_EFFECT_ROW_SIZE as usize]).unwrap();
        st.append_native_map_selector_batch(vec![active.clone()]).unwrap();
        st
    };

    let mut search = new_state(pending());
    let n = search.append_group_with_native_placement(6, 0);
    assert!(matches!(n, Ok(1)), "search append n={n:?}");
    let (got, want) = (Cell { x: search.units[1].x, y: search.units[1].y }, Cell { x: 1, y: 2 });
    assert!(got == want, "gate=0 placement={got:?} want {want:?}");

    let direct_pending = pending();
    let mut direct = new_state(direct_pending.clone());
    let n = direct.append_group_with_native_placement(6, 1);
    assert!(matches!(n, Ok(1)), "direct append n={n:?}");
    let (got, want) = (Cell { x: direct.units[1].x, y: direct.units[1].y }, Cell { x: 1, y: 1 });
    assert!(got == want, "gate=1 placement={got:?} want {want:?}");
    let spawned = &direct.units[1];
    assert!(
        !(spawned.ap != 12
            || spawned.dp != 14
            || spawned.hit != 16
            || spawned.ev != 16
            || spawned.hp != 20
            || spawned.mp != 6
            || spawned.mv != 9
            || spawned.native_record_race != 4
            || spawned.native_record_class != 5),
        "constructor projection={spawned:?}"
    );
    // Go 檢查原 roster 指標沒被改；Rust 的 roster 列在登場時被移走，原本傳進去的是複本，
    // 這裡改查保留的那份複本，語意同為「preflight 不改來源記錄」。
    assert!(
        !(direct_pending.ap != 0 || direct_pending.has_native_record_race),
        "preflight mutated the source roster record"
    );
}

/// Go：`TestAppendGroupWithNativePlacementFailsBeforeRosterMutation`
#[test]
fn append_group_with_native_placement_fails_before_roster_mutation() {
    let pending = Unit { group: 2, map_selector_key: 1, has_map_selector_key: true, ..Default::default() };
    let mut st = State {
        w: 1,
        h: 1,
        roster: vec![pending.clone()],
        native_composition_event_bytes: vec![0],
        ..Default::default()
    };
    assert!(
        st.append_group_with_native_placement(2, 0).is_err(),
        "native append accepted a row without its six-byte position record"
    );
    // Go 比較 `st.Roster[0] != pending` 指標；Rust 比較值。
    assert!(
        st.units.is_empty() && st.roster.len() == 1 && st.roster[0] == pending,
        "failed append mutated state: units={} roster={:?}",
        st.units.len(),
        st.roster
    );
}

/// Go：`TestNativeFutureGroupPlacementFailsClosed`
#[test]
fn native_future_group_placement_fails_closed() {
    let occupied = Unit {
        native_map_presentation: NativeMapPresentationState::default(),
        has_native_map_presentation: true,
        native_record_byte5: 0,
        has_native_record_byte5: true,
        native_record_byte6: 2,
        has_native_record_byte6: true,
        ..Default::default()
    };
    assert!(
        native_future_group_placement(1, 1, &[0], &[occupied], NativePositionRecord::default(), 0).is_err(),
        "future placement accepted a fully occupied map"
    );
    assert!(
        native_future_group_placement(1, 1, &[0], &[], NativePositionRecord { x_word: 1, ..Default::default() }, 1)
            .is_err(),
        "future placement accepted an out-of-bounds direct coordinate"
    );
    assert!(
        native_future_group_placement(1, 1, &[0], &[Unit::default()], NativePositionRecord::default(), 0).is_err(),
        "future placement accepted a roster without raw provenance"
    );
}

/// Go map 查詢的零值：缺鍵讀成 false。
fn cell_set(cells: &BTreeMap<Cell, bool>, x: i64, y: i64) -> bool {
    cells.get(&Cell { x, y }).copied().unwrap_or(false)
}

/// Go：`TestNativeCommandTargetCellsFloodFillHonorsRawFlags`
#[test]
fn native_command_target_cells_flood_fill_honors_raw_flags() {
    let mut flags = vec![0u8; 5];
    flags[1] = NATIVE_COMMAND_GRID_BLOCKED;
    flags[3] = NATIVE_COMMAND_GRID_ZERO_BUDGET;
    let cells = native_command_target_cells(5, 1, Cell { x: 0, y: 0 }, 2, &flags).unwrap();
    assert!(!(cell_set(&cells, 1, 0) || cell_set(&cells, 2, 0)), "blocked cell leaked through: {cells:?}");
    // Bit 0x80 forces the entered cell's remaining byte to zero, so it is
    // reachable but cannot form the previously asserted zero-cost chain.
    let mut flags = vec![0u8; 5];
    (flags[1], flags[2], flags[3]) =
        (NATIVE_COMMAND_GRID_ZERO_BUDGET, NATIVE_COMMAND_GRID_ZERO_BUDGET, NATIVE_COMMAND_GRID_ZERO_BUDGET);
    let cells = native_command_target_cells(5, 1, Cell::default(), 1, &flags);
    assert!(matches!(&cells, Ok(c) if cell_set(c, 1, 0) && !cell_set(c, 2, 0)), "zero-budget terminal cells={cells:?}");
}

/// Go：`TestNativeCommandTargetFieldBytesMatches14818Lifecycle`
#[test]
fn native_command_target_field_bytes_matches_14818_lifecycle() {
    let got = native_command_target_field_bytes(5, 1, Cell { x: 2, y: 0 }, 2, 1, &[0; 5]).unwrap();
    let want = [0, 1, 0xff, 1, 0];
    for i in 0..want.len() {
        assert!(got[i] == want[i], "field[{i}]={:#x} want {:#x}; all={got:?}", got[i], want[i]);
    }
    let blocked = NATIVE_COMMAND_GRID_BLOCKED;
    let cross =
        native_command_target_field_bytes(5, 1, Cell { x: 2, y: 0 }, 0x11, 0, &[blocked, blocked, 0, blocked, blocked])
            .unwrap();
    let want = [0xff, 0, 0, 0, 0xff];
    assert!(cross.len() == want.len(), "cross len={}", cross.len());
    for i in 0..want.len() {
        assert!(cross[i] == want[i], "cross[{i}]={:#x} want {:#x}", cross[i], want[i]);
    }
}

/// Go：`TestNativeCommandTargetCellsCrossIgnoresRawFlags`
#[test]
fn native_command_target_cells_cross_ignores_raw_flags() {
    let flags = [NATIVE_COMMAND_GRID_BLOCKED; 5];
    let cells = native_command_target_cells(5, 1, Cell { x: 2, y: 0 }, 0x12, &flags);
    assert!(matches!(&cells, Ok(c) if c.len() == 5), "cross cells={cells:?}");
}

/// Go：`TestNativeCommandTargetsMatchesRecoveredCampCodes`
#[test]
fn native_command_targets_matches_recovered_camp_codes() {
    let units = vec![
        Unit { camp: ENEMY, x: 0, y: 0, hp: 1, on_field: true, ..Default::default() },
        Unit { camp: ALLY, x: 1, y: 0, hp: 1, on_field: true, ..Default::default() },
        Unit { camp: OWN, x: 2, y: 0, hp: 1, on_field: true, ..Default::default() },
    ];
    let (enemy, ally, own) = (0, 1, 2);
    let cases: [(i64, Vec<UnitIdx>); 4] = [(0, vec![enemy]), (1, vec![ally, own]), (2, vec![ally]), (3, vec![own])];
    for (code, want) in &cases {
        let got = native_command_targets(3, 1, Cell { x: 1, y: 0 }, 2, *code, &[0; 3], &units);
        assert!(matches!(&got, Ok(g) if g.len() == want.len()), "code={code} got={got:?} want={want:?}");
        let got = got.unwrap();
        for i in 0..want.len() {
            assert!(got[i] == want[i], "code={code} got[{i}]={} want={}", got[i], want[i]);
        }
    }
}

/// Go：`TestNativeCursorConfirmationMatches115B6And14742`
#[test]
fn native_cursor_confirmation_matches_115b6_and_14742() {
    let units = vec![
        Unit { camp: ENEMY, x: 1, y: 0, has_native_record_byte5: true, ..Default::default() },
        Unit { camp: ALLY, x: 0, y: 0, has_native_record_byte5: true, ..Default::default() },
        Unit { camp: ENEMY, x: 0, y: 0, has_native_record_byte5: true, native_record_byte5: 1, ..Default::default() },
    ];
    let c0 = Cell::default();

    // targetCode 5 and 0xff cells reject before the 0x14742 count. A
    // non-0xff targetCode 4 accepts before any roster scan.
    let got = native_cursor_confirmation_allowed(c0, 0, 3, 5, &[]);
    assert!(matches!(got, Ok(false)), "target code 5 reject {got:?}");
    let got = native_cursor_confirmation_allowed(c0, 0xff, 3, 4, &[]);
    assert!(matches!(got, Ok(false)), "0xff reject {got:?}");
    let got = native_cursor_confirmation_allowed(c0, 0, 3, 4, &[]);
    assert!(matches!(got, Ok(true)), "target code 4 accept {got:?}");

    // selector 3 becomes strict radius 2, so the enemy at distance one
    // accepts. Selector 2 becomes radius one, where distance one is excluded.
    let got = native_cursor_confirmation_allowed(c0, 0, 3, 0, &units);
    assert!(matches!(got, Ok(true)), "enemy inside strict radius rejected: {got:?}");
    let got = native_cursor_confirmation_allowed(c0, 0, 2, 0, &units);
    assert!(matches!(got, Ok(false)), "strict radius boundary accepted: {got:?}");
    let got = native_cursor_confirmation_allowed(c0, 0, 2, 2, &units);
    assert!(matches!(got, Ok(true)), "code 2 failed to match ally exactly: {got:?}");
}

/// Go：`TestNativeCursorConfirmationFailsClosedWithoutRawRoster`
#[test]
fn native_cursor_confirmation_fails_closed_without_raw_roster() {
    let units = [Unit { camp: ENEMY, ..Default::default() }];
    assert!(
        native_cursor_confirmation_allowed(Cell::default(), 0, 3, 0, &units).is_err(),
        "incomplete raw roster accepted"
    );
    assert!(
        native_cursor_confirmation_allowed(Cell::default(), 0, 3, 6, &[]).is_err(),
        "relocation target code accepted by generic gate"
    );
}

/// Go：`TestNativeCommandTargetsPrefersRawByte5Predicate`
#[test]
fn native_command_targets_prefers_raw_byte5_predicate() {
    // A fully materialized roster follows the recovered 0x14818 byte+5 gate;
    // normalized HP/OnField values must not reintroduce a second predicate.
    let units = vec![
        Unit {
            camp: ENEMY,
            x: 1,
            y: 0,
            hp: 0,
            on_field: false,
            has_native_record_byte5: true,
            native_record_byte5: 0,
            ..Default::default()
        },
        Unit {
            camp: ENEMY,
            x: 2,
            y: 0,
            hp: 10,
            on_field: true,
            has_native_record_byte5: true,
            native_record_byte5: 1,
            ..Default::default()
        },
    ];
    let usable_raw = 0;
    let got = native_command_targets(3, 1, Cell { x: 0, y: 0 }, 2, 0, &[0; 3], &units);
    assert!(matches!(&got, Ok(g) if g.len() == 1 && g[0] == usable_raw), "raw target filter got={got:?}");
}

/// Go：`TestNativeCommandTargetCellsFailsClosedWithoutRawFlags`
#[test]
fn native_command_target_cells_fails_closed_without_raw_flags() {
    assert!(native_command_target_cells(2, 2, Cell::default(), 1, &[]).is_err(), "missing raw flags must fail closed");
}

/// Go：`TestNativeCommandEffectTargetsRequiresOriginalTwoStages`
#[test]
fn native_command_effect_targets_requires_original_two_stages() {
    let units = vec![
        Unit { camp: OWN, x: 0, y: 0, hp: 10, on_field: true, ..Default::default() },
        Unit { camp: ENEMY, x: 1, y: 0, hp: 10, on_field: true, ..Default::default() },
        Unit { camp: ENEMY, x: 2, y: 0, hp: 10, on_field: true, ..Default::default() },
    ];
    let (actor, confirmed, other_enemy) = (0, 1, 2);

    // Generic command: +3=1 chooses the adjacent enemy as center; +4=0
    // resolves the final effect only on that confirmed cell.
    let effects = native_command_effect_targets(3, 1, Some(actor), Some(confirmed), 1, 0, 0, &[0; 3], &units);
    assert!(matches!(&effects, Ok(e) if e.len() == 1 && e[0] == confirmed), "effects={effects:?}");
    assert!(
        native_command_effect_targets(3, 1, Some(actor), Some(other_enemy), 1, 0, 0, &[0; 3], &units).is_err(),
        "a non-selection candidate must not become the effect origin"
    );
}

fn command6_units(actor: Cell, inactive: bool, missing_raw: bool) -> Vec<Unit> {
    vec![
        Unit {
            camp: OWN,
            x: actor.x,
            y: actor.y,
            hp: 331,
            mp: 1101,
            on_field: true,
            has_native_record_byte5: true,
            ..Default::default()
        },
        Unit { camp: OWN, x: 22, y: 20, hp: 877, on_field: true, has_native_record_byte5: true, ..Default::default() },
        Unit {
            camp: ENEMY,
            x: 22,
            y: 18,
            hp: 83,
            class_id: 4,
            on_field: true,
            has_native_record_byte5: !missing_raw,
            native_record_byte5: u8::from(inactive),
            ..Default::default()
        },
    ]
}

/// Go：`TestNativeCommand6PlayerCursorCenterOutsideDirectCandidates`
///
/// #160：正常原版 actor(20,22)、我方中心(22,20)、敵目標(22,18)。
#[test]
fn native_command6_player_cursor_center_outside_direct_candidates() {
    let Some(spells) = remake_asset("spells.json") else { return };
    let book = load_native_command_records(&spells).unwrap();
    let st = State {
        w: 30,
        h: 30,
        units: command6_units(Cell { x: 20, y: 22 }, false, false),
        native_command_book: book,
        native_composition_event_bytes: vec![0; 900],
        ..Default::default()
    };
    let (actor, center, target) = (0, 1, 2);
    let cursor = Cell { x: st.units[center].x, y: st.units[center].y };
    let plan =
        st.plan_native_command6_damage_at_cursor(Some(actor), cursor, &BTreeMap::from([(4, 4)]), 3473, None).unwrap();
    assert!(
        plan.results.len() == 1 && plan.results[0].target == Some(target) && plan.mp_after == 1071,
        "plan={plan:?}"
    );
    let (a, c, t) = (&st.units[actor], &st.units[center], &st.units[target]);
    assert!(!(a.mp != 1101 || a.acted || t.hp != 83 || c.hp != 877), "planning mutated state");
}

/// Go：`TestNativeCommand1012FormalSelfCenterPlansEnemyEffects`
#[test]
fn native_command1012_formal_self_center_plans_enemy_effects() {
    let Some(spells) = remake_asset("spells.json") else { return };
    let book = load_native_command_records(&spells).unwrap();
    for id in [10i64, 11, 12] {
        let st = State {
            w: 5,
            h: 5,
            units: vec![
                Unit {
                    camp: OWN,
                    x: 2,
                    y: 2,
                    hp: 180,
                    mp: 100,
                    on_field: true,
                    has_native_record_byte5: true,
                    ..Default::default()
                },
                Unit {
                    camp: ENEMY,
                    x: 3,
                    y: 2,
                    hp: 500,
                    class_id: 4,
                    on_field: true,
                    has_native_record_byte5: true,
                    ..Default::default()
                },
            ],
            native_command_book: book.clone(),
            native_composition_event_bytes: vec![0; 25],
            ..Default::default()
        };
        let (actor, target) = (0, 1);
        let resist = BTreeMap::from([(4, 4)]);
        let plan = st.plan_native_command_damage_at_cursor(Some(actor), Cell { x: 2, y: 2 }, id, &resist, 7, None);
        assert!(
            matches!(&plan, Ok(p) if p.results.len() == 1 && p.results[0].target == Some(target)
                && p.mp_after == 100 - book[id as usize].mp_cost),
            "command{id} plan={plan:?}"
        );
        let (a, t) = (&st.units[actor], &st.units[target]);
        assert!(!(a.mp != 100 || a.acted || t.hp != 500), "command{id}: planning mutated live units");
        let moved = st.plan_native_command_damage_at_cursor(Some(actor), Cell { x: 3, y: 2 }, id, &resist, 7, None);
        assert!(
            matches!(&moved, Ok(m) if m.results.len() == 1 && m.results[0].target == Some(target)),
            "command{id}: selection-mode0 confirmation changed its fixed enemy list"
        );
        assert!(
            st.plan_native_command_damage_at_cursor(Some(actor), Cell { x: 5, y: 2 }, id, &resist, 7, None).is_err(),
            "command{id}: out-of-grid confirmation accepted"
        );
    }
}

/// Go：`TestNativeCommand6CursorAdmissionAndAtomicPreflight`
#[test]
fn native_command6_cursor_admission_and_atomic_preflight() {
    let Some(spells) = remake_asset("spells.json") else { return };
    let book = load_native_command_records(&spells).unwrap();
    struct Case {
        name: &'static str,
        actor: Cell,
        cursor: Cell,
        inactive: bool,
        missing_raw: bool,
        want: bool,
    }
    let c = |x, y| Cell { x, y };
    let case =
        |name, actor, cursor, inactive, missing_raw, want| Case { name, actor, cursor, inactive, missing_raw, want };
    let cases = [
        case("own-center", c(20, 22), c(22, 20), false, false, true),
        case("empty-center", c(20, 22), c(21, 19), false, false, true),
        case("direct-enemy", c(20, 20), c(22, 18), false, false, true),
        case("outside-selection", c(20, 22), c(22, 17), false, false, false),
        case("outside-grid", c(20, 22), c(30, 20), false, false, false),
        case("strict-radius-boundary", c(20, 22), c(22, 21), false, false, false),
        case("no-enemy-nearby", c(20, 22), c(20, 22), false, false, false),
        case("inactive-enemy", c(20, 22), c(22, 20), true, false, false),
        case("missing-raw", c(20, 22), c(22, 20), false, true, false),
    ];
    for tc in &cases {
        let name = tc.name;
        let st = State {
            w: 30,
            h: 30,
            units: command6_units(tc.actor, tc.inactive, tc.missing_raw),
            native_command_book: book.clone(),
            native_composition_event_bytes: vec![0; 900],
            ..Default::default()
        };
        let (actor, target) = (0, 2);
        let before = st.units.clone();
        let plan =
            st.plan_native_command6_damage_at_cursor(Some(actor), tc.cursor, &BTreeMap::from([(4, 4)]), 3473, None);
        assert!(plan.is_ok() == tc.want, "{name}: plan={plan:?} want={}", tc.want);
        assert!(st.units == before, "{name}: preflight changed units");
        if tc.want {
            let plan = plan.unwrap();
            assert!(
                plan.results.len() == 1 && plan.results[0].target == Some(target),
                "{name}: targets={:?}",
                plan.results
            );
            let mut called = false;
            let err = {
                let mut walk = |_: i64, _: &mut dyn FnMut(i64, u16) -> Result<(u16, bool)>| -> Result<u16> {
                    called = true;
                    Ok(0)
                };
                st.plan_native_command6_damage_at_cursor(
                    Some(actor),
                    tc.cursor,
                    &BTreeMap::new(),
                    3473,
                    Some(&mut walk),
                )
            };
            assert!(err.is_err() && !called, "{name}: missing resistance reached RNG walk");
        }
    }
}
