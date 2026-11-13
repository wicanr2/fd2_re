//! 移植自 `remake/internal/battle/native_system_group_march_test.go`。
//!
//! 預演會呼叫 D 組的 `nativeAICostRowForRecord` 與 `nativeAIPlanTowardRawDestination`（後者再走
//! E 組 `combat.go` 的移動範圍）。

use super::*;

/// Go `native_ai_runtime_test.go` 的 `nativeAIRuntimeUnit`（D 組測試檔的輔助函式，在這裡照抄一份）。
fn native_ai_runtime_unit(x: i64, y: i64, selector: u8, mode: u8) -> Unit {
    Unit {
        camp: ENEMY,
        x,
        y,
        on_field: true,
        hp: 20,
        max_hp: 20,
        mp: 4,
        max_mp: 4,
        ap: 20,
        dp: 1,
        mv: 2,
        battle_fig: 1,
        has_battle_fig: true,
        native_record_byte5: 0,
        has_native_record_byte5: true,
        native_record_byte6: selector,
        has_native_record_byte6: true,
        native_record_byte34: mode,
        has_native_record_byte34: true,
        native_record_byte35: 0,
        has_native_record_byte35: true,
        native_record_byte36: 0,
        has_native_record_byte36: true,
        native_record_byte8: 1,
        has_native_record_byte8: true,
        native_record_race: 0,
        has_native_record_race: true,
        native_record_class: 0,
        has_native_record_class: true,
        native_record_word42: 20,
        has_native_record_word42: true,
        native_record_word46: 4,
        has_native_record_word46: true,
        native_map_presentation: NativeMapPresentationState { x: x as u8, y: y as u8, ..Default::default() },
        has_native_map_presentation: true,
        inventory_slots: vec![0xff; 8],
        native_inventory_flags: vec![0; 8],
        ..Default::default()
    }
}

/// Go `nativeAIRuntimeCostRows`
fn native_ai_runtime_cost_rows() -> Vec<Vec<u8>> {
    vec![vec![1u8; NATIVE_MOVEMENT_COST_ROW_SIZE as usize]; NATIVE_MOVEMENT_COST_ROW_COUNT as usize]
}

/// Go：`TestNativeSystemGroupMarchPlansInRecordOrderAndCommits`
#[test]
fn native_system_group_march_plans_in_record_order_and_commits() {
    let mut state = State {
        w: 4,
        h: 2,
        units: vec![native_ai_runtime_unit(0, 0, 2, 0), native_ai_runtime_unit(0, 1, 2, 0)],
        native_composition_event_bytes: vec![0; 8],
        native_terrain_move_codes: vec![0; 8],
        native_movement_cost_rows: native_ai_runtime_cost_rows(),
        native_field_event_slots: vec![-1; 8],
        native_field_events: vec![NativeFieldEvent::default(); 16],
        ..Default::default()
    };
    let plan = state.plan_native_system_group_march(Cell { x: 3, y: 0 }).unwrap();
    assert!(
        plan.steps.len() == 2 && plan.steps[0].unit_index == 0 && plan.steps[1].unit_index == 1,
        "steps={:?}",
        plan.steps
    );
    let (first, second) = (&state.units[0], &state.units[1]);
    assert!(
        first.x == 0 && first.y == 0 && !first.acted && second.x == 0 && second.y == 1 && !second.acted,
        "planner mutated live units"
    );
    for step in &plan.steps {
        state.commit_native_system_group_march_step(step).unwrap();
    }
    let (first, second) = (&state.units[0], &state.units[1]);
    assert!(
        first.acted && first.native_record_byte5 & 0x80 != 0 && second.acted && second.native_record_byte5 & 0x80 != 0,
        "commit flags first={:#x} second={:#x}",
        first.native_record_byte5,
        second.native_record_byte5
    );
}

/// Go：`TestNativeSystemGroupMarchFailsAtomicallyOnEventOrMissingRaw`
#[test]
fn native_system_group_march_fails_atomically_on_event_or_missing_raw() {
    let mut state = State {
        w: 2,
        h: 1,
        units: vec![native_ai_runtime_unit(1, 0, 2, 0)],
        native_composition_event_bytes: vec![0; 2],
        native_terrain_move_codes: vec![0; 2],
        native_movement_cost_rows: native_ai_runtime_cost_rows(),
        native_field_event_slots: vec![0, -1],
        native_field_events: vec![NativeFieldEvent::default(); 16],
        ..Default::default()
    };
    state.native_field_events[0] = NativeFieldEvent { event_id: 82, selector: 1 };
    assert!(state.plan_native_system_group_march(Cell { x: 0, y: 0 }).is_err(), "unowned global event was accepted");
    let unit = &state.units[0];
    assert!(
        unit.x == 1 && unit.y == 0 && !unit.acted && unit.native_record_byte5 == 0,
        "event failure mutated unit: {unit:?}"
    );
    state.units[0].has_native_record_byte6 = false;
    state.native_field_event_slots[0] = -1;
    assert!(state.plan_native_system_group_march(Cell { x: 0, y: 0 }).is_err(), "missing raw gate was accepted");
}

/// Go：`TestNativeSystemGroupMarchPreflightsOwnedEvent75WithoutLiveMutation`
#[test]
fn native_system_group_march_preflights_owned_event75_without_live_mutation() {
    let mut unit = native_ai_runtime_unit(1, 0, 2, 0);
    unit.native_record_byte8 = 9;
    unit.has_native_record_byte8 = true;
    let mut state = State {
        w: 2,
        h: 1,
        units: vec![unit],
        native_round_counter: 8,
        native_composition_event_bytes: vec![0; 2],
        native_terrain_move_codes: vec![0; 2],
        native_movement_cost_rows: native_ai_runtime_cost_rows(),
        native_field_event_slots: vec![0, -1],
        native_field_events: vec![NativeFieldEvent::default(); 16],
        native_field_event_rules: vec![NativeFieldEventRule {
            event_id: 75,
            selector: 1,
            trigger_gate: "record_byte6_nonzero".into(),
            turn_chain: Some(NativeFieldTurnChain {
                handler: "0x35c79".into(),
                trigger_record_byte8: 9,
                mismatch_text_index: 0,
                success_text_index: 1,
                state_writes: vec![
                    NativeEventStateWrite { index: 17, value: 1 },
                    NativeEventStateWrite { index: 16, value: 4 },
                ],
                turn_activations: vec![
                    NativeTurnActivation { slot: 1, event_id: 76, raw_camp: 2, turn_delta: 1 },
                    NativeTurnActivation { slot: 0, event_id: 74, raw_camp: 0, turn_delta: 0 },
                ],
            }),
            ..Default::default()
        }],
        has_native_turn_event_control_state: true,
        ..Default::default()
    };
    state.native_field_events[0] = NativeFieldEvent { event_id: 75, selector: 1 };
    state.native_turn_event_controls[0] = NativeTurnEventControl { turn: 0xff, event_id: 74, raw_camp: 0 };
    state.native_turn_event_controls[1] = NativeTurnEventControl { turn: 0xff, event_id: 76, raw_camp: 2 };
    let plan = state.plan_native_system_group_march(Cell { x: 0, y: 0 }).unwrap();
    assert!(
        plan.steps.len() == 1
            && plan.steps[0].events
                == [NativeSystemGroupMarchEvent { path_index: 1, event_id: 75, text_index: 1, ..Default::default() }],
        "plan={plan:?}"
    );
    let unit = &state.units[0];
    assert!(
        unit.x == 1
            && !unit.acted
            && state.native_event_state[16] == 0
            && state.native_event_state[17] == 0
            && state.native_turn_event_controls[0].turn == 0xff,
        "preflight mutated live state: unit={unit:?} state={:?} rows={:?}",
        &state.native_event_state[16..18],
        &state.native_turn_event_controls[..2]
    );
}

/// Go：`TestNativeSystemGroupMarchEvent61ProjectionExtendsDynamicRecordBound`
#[test]
fn native_system_group_march_event61_projection_extends_dynamic_record_bound() {
    let mut actor = native_ai_runtime_unit(2, 0, 2, 0);
    actor.inventory = vec![0xD0];
    actor.equipped = vec![false];
    actor.inventory_slots = vec![0xD0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff];
    actor.native_inventory_flags = vec![0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80];
    let mut joined = native_ai_runtime_unit(1, 0, 2, 0);
    joined.group = 1;
    joined.fig = 31;
    joined.camp = OWN;
    joined.inventory_slots = vec![0xff; 8];
    joined.native_inventory_flags = vec![0x80; 8];
    let presentation = NativeFieldPresentation {
        archive: "FDOTHER.DAT".into(),
        resource: 45,
        frames: 59,
        helper: "0x2935b".into(),
        destination_offset: 48356,
        stride: 320,
        transparent: -1,
        delay_helper: "0x17aa9".into(),
        delay_ticks: 2,
    };
    let mut state = State {
        w: 3,
        h: 1,
        units: vec![actor],
        roster: vec![joined],
        native_composition_event_bytes: vec![0; 3],
        native_terrain_move_codes: vec![0; 3],
        native_movement_cost_rows: native_ai_runtime_cost_rows(),
        native_field_event_slots: vec![-1, 0, -1],
        native_field_events: vec![NativeFieldEvent::default(); 16],
        native_field_event_rules: vec![NativeFieldEventRule {
            event_id: 61,
            selector: 1,
            once_state: Some(12),
            required_item: Some(0xD0),
            consume_item: true,
            spawn_group: Some(1),
            join_character: Some(31),
            text_indices: Some(NativeFieldTextIndices { missing_item: 2, success: 3, r#final: 4 }),
            presentation: Some(presentation),
            ..Default::default()
        }],
        ..Default::default()
    };
    state.native_field_events[0] = NativeFieldEvent { event_id: 61, selector: 1 };
    let plan = state.plan_native_system_group_march(Cell { x: 0, y: 0 }).unwrap();
    assert!(
        plan.steps.len() == 2
            && plan.steps[0].unit_index == 0
            && plan.steps[0].events.len() == 1
            && plan.steps[0].events[0].event_id == 61
            && plan.steps[1].unit_index == 1,
        "dynamic event61 plan={plan:?}"
    );
    let actor = &state.units[0];
    assert!(
        state.units.len() == 1
            && state.roster.len() == 1
            && state.native_event_state[12] == 0
            && actor.inventory == [0xD0]
            && actor.x == 2
            && !actor.acted,
        "event61 preflight mutated live state: units={} roster={} state={} actor={actor:?}",
        state.units.len(),
        state.roster.len(),
        state.native_event_state[12]
    );
}
