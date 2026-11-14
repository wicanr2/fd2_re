//! 移植自 `remake/internal/campaign/native_continue_pending_groups_test.go`。
//!
//! Go 以 `state.Roster != nil`／`state.PendingGroups != nil` 判斷失敗的綁定沒有寫入；Rust 的 `Vec`／
//! `BTreeMap` 不分 nil 與空值，改判斷兩者為空且 `has_native_pending_group_binding` 為假。

use super::*;
use crate::campaign::native_continue_runtime_units::native_continue_runtime_units_test::state_from_asset_topology;
use crate::test_support::remake_asset;

type FieldControlMutate = fn(&mut [u8; save::CURRENT_FIELD_CONTROL_SIZE]);

/// Go `nativePendingGroupsInput`。
pub(crate) fn native_pending_groups_input(turn: u8, mutate: Option<FieldControlMutate>) -> save::ContinueRuntimeInput {
    let mut snapshot = save::CurrentSnapshot {
        header: save::CurrentRuntimeHeader {
            turn_counter: turn,
            chapter: 0,
            cursor_x: 7,
            cursor_y: 4,
            visible_cursor_x: 7,
            visible_cursor_y: 4,
            ..Default::default()
        },
        ..Default::default()
    };
    snapshot.native_field_control[2] = 30;
    for (index, row) in [[3u8, 0, 1], [4, 1, 0], [5, 2, 0], [6, 3, 1]].iter().enumerate() {
        snapshot.native_field_control[3 + index * 3..6 + index * 3].copy_from_slice(row);
    }
    if let Some(mutate) = mutate {
        mutate(&mut snapshot.native_field_control);
    }
    save::build_continue_runtime_input(
        &snapshot,
        save::ContinueRuntimeContext {
            chapter: 0,
            field_width: 24,
            field_height: 24,
            selector_group_count: 256,
            title_timer_tick: 0,
            has_title_timer_tick: true,
        },
    )
    .unwrap()
}

/// Go `nativePendingGroupsFixture`：`(state, assetState, scenario, itemRows)`。
pub(crate) fn native_pending_groups_fixture(
    input: &save::ContinueRuntimeInput,
) -> Option<(battle::State, battle::State, battle::Scenario, Vec<u8>)> {
    let map0 = remake_asset("maps/map0/map0_units.json")?;
    let ch01 = remake_asset("scenarios/ch01.json")?;
    let rows = remake_asset("data/native_item_effect_rows.json")?;
    let asset_state = battle::load(&map0).unwrap();
    let scenario = battle::load_scenario(&ch01).unwrap();
    let item_rows = battle::load_native_item_effect_row_prefix(&rows).unwrap();
    let mut state = state_from_asset_topology(&asset_state, true);
    materialize_native_continue_field_boundary(Some(&mut state), input, 0).unwrap();
    state.has_native_runtime_unit_projection = true;
    Some((state, asset_state, scenario, item_rows))
}

fn groups(list: &[i64]) -> BTreeMap<i64, bool> {
    list.iter().map(|g| (*g, true)).collect()
}

fn unbound(state: &battle::State) -> bool {
    !state.has_native_pending_group_binding && state.roster.is_empty() && state.pending_groups.is_empty()
}

#[test]
fn test_native_continue_spawn_schedule_accepts_all_native_append_scenarios() {
    for (chapter, want) in [(1, 4), (2, 1), (3, 1), (26, 9)] {
        let Some(path) = remake_asset(&format!("scenarios/ch{chapter:02}.json")) else { return };
        let scenario = battle::load_scenario(&path).unwrap();
        assert!(scenario.runtime_append_groups, "ch{chapter:02}: scenario lost native append ownership");
        let schedule = native_continue_spawn_schedule(&scenario).unwrap();
        assert_eq!(schedule.len(), want, "ch{chapter:02}: schedule keys");
    }
}

#[test]
fn test_materialize_native_continue_pending_groups_binds_current_and_future_schedule() {
    let input = native_pending_groups_input(1, None);
    let Some((mut state, asset_state, scenario, item_rows)) = native_pending_groups_fixture(&input) else { return };
    let before_units = state.units.clone();
    let before_controls = state.native_turn_event_controls;

    materialize_native_continue_pending_groups(
        Some(&mut state),
        &input,
        0,
        Some(&asset_state),
        Some(&scenario),
        &item_rows,
    )
    .unwrap();
    assert!(
        state.has_native_pending_group_binding
            && state.roster.len() == 15
            && state.units == before_units
            && state.native_turn_event_controls == before_controls,
        "pending-group state was not isolated: {state:?}"
    );
    let want_counts = BTreeMap::from([(3, 1), (4, 4), (5, 5), (6, 4), (7, 1)]);
    let mut got_counts: BTreeMap<i64, i64> = BTreeMap::new();
    for unit in &state.roster {
        *got_counts.entry(unit.group).or_default() += 1;
    }
    assert!(
        got_counts == want_counts && state.pending_groups == groups(&[3, 4, 5, 6, 7]),
        "roster groups={got_counts:?} pending={:?}",
        state.pending_groups
    );
    for forbidden in [1, 2, 10, 11] {
        assert!(
            !state.pending_groups.contains_key(&forbidden),
            "already-present/unscheduled group {forbidden} was bound"
        );
    }
    // 綁定必須深複製；後續 constructor 不可污染資產 loader 的來源列。
    state.roster[0].inventory_slots[0] ^= 0xff;
    state.roster[0].native_constructor.as_mut().unwrap().record[0] ^= 0xff;
    let source = asset_state.units.iter().find(|u| u.group == state.roster[0].group);
    assert!(
        source.is_some_and(|source| source.inventory_slots[0] != state.roster[0].inventory_slots[0]
            && source.native_constructor.as_ref().unwrap().record[0]
                != state.roster[0].native_constructor.as_ref().unwrap().record[0]),
        "pending roster shares mutable storage with chapter asset"
    );
}

#[test]
fn test_materialize_native_continue_pending_groups_accepts_proven_static_scenario() {
    let input = native_pending_groups_input(1, None);
    let Some((mut state, mut asset_state, mut scenario, item_rows)) = native_pending_groups_fixture(&input) else {
        return;
    };
    scenario.runtime_append_groups = false;
    scenario.events = Vec::new();
    asset_state.native_field_event_rules = Vec::new();
    state.native_field_event_rules = Vec::new();
    materialize_native_continue_pending_groups(
        Some(&mut state),
        &input,
        0,
        Some(&asset_state),
        Some(&scenario),
        &item_rows,
    )
    .unwrap();
    assert!(
        state.has_native_pending_group_binding && state.roster.is_empty() && state.pending_groups.is_empty(),
        "靜態 pending binding 不完整：roster={:?} pending={:?} bound={}",
        state.roster,
        state.pending_groups,
        state.has_native_pending_group_binding
    );
}

#[test]
fn test_materialize_native_continue_pending_groups_rejects_static_future_schedule() {
    let input = native_pending_groups_input(1, None);
    let Some((mut state, asset_state, mut scenario, item_rows)) = native_pending_groups_fixture(&input) else { return };
    scenario.runtime_append_groups = false;
    assert!(
        materialize_native_continue_pending_groups(
            Some(&mut state),
            &input,
            0,
            Some(&asset_state),
            Some(&scenario),
            &item_rows
        )
        .is_err(),
        "缺少 runtime append owner 的 future schedule 被接受"
    );
    assert!(unbound(&state), "失敗的靜態 future schedule 污染 state：{state:?}");
}

#[test]
fn test_materialize_native_continue_pending_groups_keeps_saved_turn_pending() {
    let input = native_pending_groups_input(4, None);
    let Some((mut state, asset_state, scenario, item_rows)) = native_pending_groups_fixture(&input) else { return };
    materialize_native_continue_pending_groups(
        Some(&mut state),
        &input,
        0,
        Some(&asset_state),
        Some(&scenario),
        &item_rows,
    )
    .unwrap();
    assert_eq!(state.pending_groups, groups(&[4, 5, 6]), "saved-turn pending groups");
}

#[test]
fn test_materialize_native_continue_pending_groups_excludes_saved_turn_selector2() {
    // turn3/event0 已在上一輪增加到3後的 selector2 checkpoint 掃描。
    let input = native_pending_groups_input(3, Some(|raw| raw[5] = 2));
    let Some((mut state, asset_state, scenario, item_rows)) = native_pending_groups_fixture(&input) else { return };
    materialize_native_continue_pending_groups(
        Some(&mut state),
        &input,
        0,
        Some(&asset_state),
        Some(&scenario),
        &item_rows,
    )
    .unwrap();
    assert_eq!(state.pending_groups, groups(&[4, 5, 6]), "selector2 pending groups");
}

#[test]
fn test_materialize_native_continue_pending_groups_rejects_mutated_schedule_atomically() {
    // event 0 的 live turn 已被改寫，不能用 scenario turn 3 猜回。
    let input = native_pending_groups_input(1, Some(|raw| raw[3] = 2));
    let Some((mut state, asset_state, scenario, item_rows)) = native_pending_groups_fixture(&input) else { return };
    let before_units = state.units.clone();
    assert!(
        materialize_native_continue_pending_groups(
            Some(&mut state),
            &input,
            0,
            Some(&asset_state),
            Some(&scenario),
            &item_rows
        )
        .is_err(),
        "mutated live turn schedule was accepted"
    );
    assert!(unbound(&state) && state.units == before_units, "failed pending binding mutated state={state:?}");
}

#[test]
fn test_materialize_native_continue_pending_groups_rejects_past_row_moved_to_future() {
    // 原 turn3/event0 已過，但 handler 將 live row 延到未來。
    let input = native_pending_groups_input(4, Some(|raw| raw[3] = 5));
    let Some((mut state, asset_state, scenario, item_rows)) = native_pending_groups_fixture(&input) else { return };
    let before_controls = state.native_turn_event_controls;
    assert!(
        materialize_native_continue_pending_groups(
            Some(&mut state),
            &input,
            0,
            Some(&asset_state),
            Some(&scenario),
            &item_rows
        )
        .is_err(),
        "past schedule row moved to the future was accepted"
    );
    assert!(
        unbound(&state) && state.native_turn_event_controls == before_controls,
        "failed shifted-row binding mutated state={state:?}"
    );
}

#[test]
fn test_materialize_native_continue_pending_groups_rejects_wrong_same_size_map_atomically() {
    let input = native_pending_groups_input(1, None);
    let Some((mut state, asset_state, scenario, item_rows)) = native_pending_groups_fixture(&input) else { return };
    state.native_composition_event_bytes[0] ^= 0xff;
    let before_composition = state.native_composition_event_bytes.clone();
    assert!(
        materialize_native_continue_pending_groups(
            Some(&mut state),
            &input,
            0,
            Some(&asset_state),
            Some(&scenario),
            &item_rows
        )
        .is_err(),
        "same-size state with different map composition was accepted"
    );
    assert!(
        unbound(&state) && state.native_composition_event_bytes == before_composition,
        "failed map-identity check mutated state={state:?}"
    );
}

#[test]
fn test_materialize_native_continue_pending_groups_uses_map25_event61_once_state() {
    let Some(map25) = remake_asset("maps/map25/map25_units.json") else { return };
    let Some(ch26) = remake_asset("scenarios/ch26.json") else { return };
    let Some(rows) = remake_asset("data/native_item_effect_rows.json") else { return };
    let asset_state = battle::load(&map25).unwrap();
    let scenario = battle::load_scenario(&ch26).unwrap();
    let item_rows = battle::load_native_item_effect_row_prefix(&rows).unwrap();
    let build = |once: u8, mutate_live_selector: bool| -> (save::ContinueRuntimeInput, battle::State) {
        let mut snapshot = save::CurrentSnapshot {
            header: save::CurrentRuntimeHeader {
                turn_counter: 1,
                chapter: 25,
                cursor_x: 7,
                cursor_y: 4,
                visible_cursor_x: 7,
                visible_cursor_y: 4,
                ..Default::default()
            },
            ..Default::default()
        };
        snapshot.native_field_control[2] = asset_state.units.len() as u8;
        for (index, turn) in [2u8, 4, 6, 8, 10, 12, 15, 16, 17].into_iter().enumerate() {
            snapshot.native_field_control[3 + index * 3..6 + index * 3].copy_from_slice(&[turn, 57, 0]);
        }
        const FIELD_EVENT_OFFSET: usize = 3 + 16 * 3;
        for (index, event) in asset_state.native_field_events.iter().enumerate() {
            snapshot.native_field_control[FIELD_EVENT_OFFSET + index * 2] = event.event_id;
            snapshot.native_field_control[FIELD_EVENT_OFFSET + index * 2 + 1] = event.selector;
        }
        if mutate_live_selector {
            snapshot.native_field_control[FIELD_EVENT_OFFSET + 2 * 2 + 1] = 0;
        }
        snapshot.native_event_state[12] = once;
        let input = save::build_continue_runtime_input(
            &snapshot,
            save::ContinueRuntimeContext {
                chapter: 25,
                field_width: asset_state.w,
                field_height: asset_state.h,
                selector_group_count: 256,
                title_timer_tick: 0,
                has_title_timer_tick: true,
            },
        )
        .unwrap();
        let mut state = state_from_asset_topology(&asset_state, true);
        materialize_native_continue_field_boundary(Some(&mut state), &input, 25).unwrap();
        state.has_native_runtime_unit_projection = true;
        (input, state)
    };

    let (input, mut state) = build(0, false);
    materialize_native_continue_pending_groups(
        Some(&mut state),
        &input,
        25,
        Some(&asset_state),
        Some(&scenario),
        &item_rows,
    )
    .unwrap();
    assert!(
        state.pending_groups.get(&1).copied().unwrap_or(false),
        "unconsumed map25/event61 group1 was not bound: {:?}",
        state.pending_groups
    );

    let (consumed_input, mut consumed) = build(1, false);
    materialize_native_continue_pending_groups(
        Some(&mut consumed),
        &consumed_input,
        25,
        Some(&asset_state),
        Some(&scenario),
        &item_rows,
    )
    .unwrap();
    assert!(
        !consumed.pending_groups.get(&1).copied().unwrap_or(false),
        "consumed map25/event61 group1 was rebound: {:?}",
        consumed.pending_groups
    );

    let (wrong_input, mut wrong) = build(0, true);
    assert!(
        materialize_native_continue_pending_groups(
            Some(&mut wrong),
            &wrong_input,
            25,
            Some(&asset_state),
            Some(&scenario),
            &item_rows
        )
        .is_err(),
        "map25/event61 with the wrong live selector was accepted"
    );
    assert!(unbound(&wrong), "rejected live field-event binding mutated state={wrong:?}");
}
