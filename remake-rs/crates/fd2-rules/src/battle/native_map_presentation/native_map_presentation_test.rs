//! 移植自 `remake/internal/battle/native_map_presentation_test.go`。

use super::*;

/// Go：`TestNativeMapHUDPersistentStateSeparatesSaveAndProcessLifetime`
#[test]
fn native_map_hud_persistent_state_separates_save_and_process_lifetime() {
    let mut persistent = initial_native_map_hud_persistent_state();
    assert!(
        persistent.display_gate_a == 1
            && persistent.anchor_x == 1
            && persistent.has_display_gate_a
            && persistent.has_anchor_x,
        "initial HUD persistent state={persistent:?}"
    );
    assert!(
        persistent.capture_native_map_hud(NativeMapHUDRuntimeState {
            display_gate_a: 0,
            display_gate_b: 7,
            anchor_x: 0xf2
        }),
        "complete runtime HUD state was not captured"
    );
    assert!(persistent.display_gate_a == 0 && persistent.anchor_x == 0xf2, "captured persistent state={persistent:?}");
    assert!(
        persistent.restore_saved_gate_a(3) && persistent.anchor_x == 0xf2,
        "saved gate A overwrote process-local anchor: {persistent:?}"
    );
    let (runtime, ok) = persistent.materialize_runtime(1);
    assert!(
        ok && runtime == NativeMapHUDRuntimeState { display_gate_a: 3, display_gate_b: 1, anchor_x: 0xf2 },
        "materialized runtime={runtime:?} ok={ok}"
    );
}

/// Go：`TestNativeMapHUDPersistentStateRequiresBothProvenanceBits`
#[test]
fn native_map_hud_persistent_state_requires_both_provenance_bits() {
    for persistent in [
        NativeMapHUDPersistentState { display_gate_a: 1, has_display_gate_a: true, ..Default::default() },
        NativeMapHUDPersistentState { anchor_x: 1, has_anchor_x: true, ..Default::default() },
    ] {
        assert!(!persistent.materialize_runtime(1).1, "incomplete persistent state accepted: {persistent:?}");
    }
}

/// Go：`TestNativeMapPresentationPreservesSevenTickGridLifecycle`
#[test]
fn native_map_presentation_preserves_seven_tick_grid_lifecycle() {
    let mut u = Unit { x: 4, y: 5, dir: 3, off_x: 7.0, off_y: 8.0, ..Default::default() };
    u.materialize_native_map_presentation().unwrap();
    assert!(
        u.native_map_presentation == NativeMapPresentationState { x: 4, y: 5, ..Default::default() }
            && u.dir == 0
            && u.off_x == 0.0
            && u.off_y == 0.0,
        "constructor state={:?} normalized=({},{},{})",
        u.native_map_presentation,
        u.dir,
        u.off_x,
        u.off_y
    );
    for motion in 1..=6 {
        assert!(u.set_native_map_grid_motion(3, motion), "motion {motion} rejected");
        let p = u.native_map_presentation;
        assert!(p.x == 4 && p.y == 5 && p.motion == motion as u8, "motion {motion} state={p:?}");
    }
    assert!(u.finish_native_map_grid_step(3, 5, 5), "seventh-tick commit rejected");
    assert!(
        u.native_map_presentation == NativeMapPresentationState { x: 5, y: 5, pose: 3, motion: 0 }
            && u.x == 5
            && u.y == 5
            && u.dir == 3,
        "completed state={:?} normalized=({},{},{})",
        u.native_map_presentation,
        u.x,
        u.y,
        u.dir
    );
}

/// Go：`TestNativeUnitLayerEntryRequiresCompleteRawProvenance`
#[test]
fn native_unit_layer_entry_requires_complete_raw_provenance() {
    let mut u = Unit {
        x: 2,
        y: 3,
        map_selector_slot: 4,
        has_map_selector_slot: true,
        native_record_byte5: 0x81,
        has_native_record_byte5: true,
        native_transient: [0, 0, 0, 0, 2, 0],
        ..Default::default()
    };
    assert!(!u.native_unit_layer_entry().1, "accepted unit without materialized raw presentation");
    u.materialize_native_map_presentation().unwrap();
    assert!(u.set_native_map_grid_motion(2, 6), "raw motion rejected");
    let (entry, ok) = u.native_unit_layer_entry();
    assert!(
        ok && entry
            == fdicon::NativeUnitLayerEntry {
                x: 2,
                y: 3,
                slot: 4,
                pose: 2,
                motion_offset: 6,
                flags: 0x81,
                force_base: true,
                inactive: true
            },
        "entry={entry:?} ok={ok}"
    );
}

/// Go：`TestMaterializeNativeMapSelectorSlotsIsAtomicOnInvalidCoordinate`
#[test]
fn materialize_native_map_selector_slots_is_atomic_on_invalid_coordinate() {
    let mut cache = fdicon::NativeSelectorCache::default();
    let mut units = vec![
        Unit { x: 1, y: 2, map_selector_key: 7, has_map_selector_key: true, ..Default::default() },
        Unit { x: 0x100, y: 2, map_selector_key: 8, has_map_selector_key: true, ..Default::default() },
    ];
    assert!(
        materialize_native_map_selector_slots(&mut units, Some(&mut cache)).is_err(),
        "accepted native coordinate outside byte range"
    );
    for (i, u) in units.iter().enumerate() {
        assert!(!u.has_map_selector_slot && !u.has_native_map_presentation, "unit {i} partially materialized: {u:?}");
    }
}

/// Go：`TestStateOwnsNativeMapCycleGlobalsAfterConstruction`
#[test]
fn state_owns_native_map_cycle_globals_after_construction() {
    let mut st = State::default();
    assert!(!st.advance_native_map_presentation_cycles(5), "legacy state accepted native cycle update");
    st.append_native_map_selector_batch(vec![Unit {
        x: 1,
        y: 2,
        map_selector_key: 7,
        has_map_selector_key: true,
        ..Default::default()
    }])
    .unwrap();
    assert!(
        st.has_native_map_cycle_state && st.advance_native_map_presentation_cycles(5),
        "native cycle globals were not materialized"
    );
    assert!(
        st.has_native_terrain_phase_state
            && st.advance_native_terrain_phase(3, -1)
            && st.native_terrain_phase_state.phase == 1,
        "terrain phase state={:?} valid={}",
        st.native_terrain_phase_state,
        st.has_native_terrain_phase_state
    );
    assert_eq!(
        st.native_map_cycle_state,
        fdicon::NativeMapSpriteCycleState { idle: 1, moving: 1, last_timer_tick: 5 },
        "first cycle call"
    );
    assert!(
        st.advance_native_map_presentation_cycles(6)
            && st.native_map_cycle_state
                == fdicon::NativeMapSpriteCycleState { idle: 1, moving: 2, last_timer_tick: 5 },
        "gated cycle call={:?}",
        st.native_map_cycle_state
    );
}

/// Go：`TestNativeMapFrameRosterRequiresCompleteRawRoster`
#[test]
fn native_map_frame_roster_requires_complete_raw_roster() {
    let mut st = State::default();
    let u = Unit {
        x: 1,
        y: 2,
        map_selector_key: 7,
        has_map_selector_key: true,
        native_record_byte5: 0x80,
        has_native_record_byte5: true,
        battle_fig: 9,
        has_battle_fig: true,
        native_record_race: 3,
        has_native_record_race: true,
        native_record_class: 4,
        has_native_record_class: true,
        ..Default::default()
    };
    st.append_native_map_selector_batch(vec![u]).unwrap();
    assert!(st.units[0].set_native_map_grid_motion(1, 5), "raw motion rejected");
    let roster = st.native_map_frame_roster().unwrap();
    assert!(
        roster.units.len() == 1
            && roster.foreground.len() == 1
            && roster.units[0].motion_offset == 5
            && roster.foreground[0]
                == fdicon::NativeForegroundLayerEntry {
                    x: 1,
                    y: 2,
                    pose: 1,
                    motion_offset: 5,
                    inactive: false,
                    unit7: 9,
                    race: 3,
                    class: 4
                },
        "roster={roster:?}"
    );
    st.units[0].has_battle_fig = false;
    assert!(st.native_map_frame_roster().is_err(), "accepted legacy BattleFig fallback as raw unit+7");
}

/// Go：`TestNativeMapFrameRosterDoesNotReturnPartialInput`
#[test]
fn native_map_frame_roster_does_not_return_partial_input() {
    let mut st = State::default();
    let units = vec![
        Unit {
            x: 1,
            y: 2,
            map_selector_key: 7,
            has_map_selector_key: true,
            has_native_record_byte5: true,
            battle_fig: 9,
            has_battle_fig: true,
            native_record_race: 3,
            has_native_record_race: true,
            native_record_class: 4,
            has_native_record_class: true,
            ..Default::default()
        },
        Unit {
            x: 3,
            y: 4,
            map_selector_key: 8,
            has_map_selector_key: true,
            has_native_record_byte5: true,
            battle_fig: 10,
            has_battle_fig: true,
            native_record_race: 5,
            has_native_record_race: true,
            ..Default::default()
        },
    ];
    st.append_native_map_selector_batch(units).unwrap();
    // Go 另外檢查失敗時回傳的 Units／Foreground 是 nil；Rust 失敗只有 Err，沒有部分結果可回傳。
    assert!(st.native_map_frame_roster().is_err(), "partial roster accepted");
}

/// Go：`TestNativeMapHUDStatePreservesRawGatesAndPersistentAnchor`
#[test]
fn native_map_hud_state_preserves_raw_gates_and_persistent_anchor() {
    let mut st = State::default();
    assert!(!st.advance_native_map_hud_anchor(2, 6), "legacy state accepted HUD anchor update");
    assert!(st.materialize_native_map_hud_state(0, 7, 1), "HUD state materialization rejected");
    assert_eq!(
        st.native_map_hud_state,
        NativeMapHUDRuntimeState { display_gate_a: 0, display_gate_b: 7, anchor_x: 1 },
        "HUD state"
    );
    // 0x1ACF3 閘門 A 為 0 時提早返回，0x1AD2A 不跑：anchor 不變。
    assert!(!st.advance_native_map_hud_anchor(2, 6) && st.native_map_hud_state.anchor_x == 1, "gated anchor");
    st.native_map_hud_state.display_gate_a = 1;
    assert!(st.advance_native_map_hud_anchor(2, 6) && st.native_map_hud_state.anchor_x == 0xf2, "left anchor");
    assert!(st.advance_native_map_hud_anchor(5, 6) && st.native_map_hud_state.anchor_x == 0xf2, "retained anchor");
    assert!(st.advance_native_map_hud_anchor(10, 6) && st.native_map_hud_state.anchor_x == 1, "right anchor");
}
