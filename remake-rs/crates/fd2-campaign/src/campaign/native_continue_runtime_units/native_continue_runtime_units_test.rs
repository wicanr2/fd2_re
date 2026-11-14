//! 移植自 `remake/internal/campaign/native_continue_runtime_units_test.go`。
//!
//! `TestMaterializeNativeContinueRuntimeUnitsOriginalSnapshot` 在 Go 讀 `FD2_NATIVE_SAVE_FIXTURE`；
//! Rust 改讀 `FD2_FLAME2_DIR/FD2.SAV`（理由見 `native_persistent_party_test.rs`），期望值不變。

use super::*;
use crate::campaign::native_continue_field_boundary::native_continue_field_boundary_test::continue_field_boundary_state;
use crate::test_support::{input, remake_asset};

/// Go `nativeRuntimeCatalog`；缺素材時回 `None`。
pub(crate) fn native_runtime_catalog() -> Option<NativeCharacterCatalog> {
    let path = remake_asset("data/native_character_catalog.json")?;
    Some(load_native_character_catalog(&path).unwrap())
}

/// Go 以 `append([]T(nil), asset...)` 複製 FDFIELD 拓撲到新 `battle.State`。
pub(crate) fn state_from_asset_topology(asset: &battle::State, with_rules: bool) -> battle::State {
    let mut state = battle::State::default();
    (state.w, state.h) = (asset.w, asset.h);
    state.native_composition_event_bytes = asset.native_composition_event_bytes.clone();
    state.native_field_event_slots = asset.native_field_event_slots.clone();
    state.native_field_events = asset.native_field_events.clone();
    if with_rules {
        state.native_field_event_rules = asset.native_field_event_rules.clone();
    }
    state
}

#[test]
fn test_materialize_native_continue_runtime_units_original_snapshot() {
    let Some(path) = input("FD2_FLAME2_DIR", "FD2.SAV") else { return };
    let Some(mut catalog) = native_runtime_catalog() else { return };
    let Some(map0) = remake_asset("maps/map0/map0_units.json") else { return };
    let Some(ch01) = remake_asset("scenarios/ch01.json") else { return };
    let Some(rows) = remake_asset("data/native_item_effect_rows.json") else { return };
    let stored = std::fs::read(path).unwrap();
    let plain = save::decode(&stored).unwrap();
    let snapshot = save::inspect_current_snapshot(&plain).unwrap();
    let input = save::build_continue_runtime_input(
        &snapshot,
        save::ContinueRuntimeContext {
            chapter: i64::from(snapshot.header.chapter),
            // The supplied reference snapshot is chapter 0. The selected
            // map0_units.json asset is the authoritative 24×24 field source.
            field_width: 24,
            field_height: 24,
            selector_group_count: 256,
            title_timer_tick: 0,
            has_title_timer_tick: true,
        },
    )
    .unwrap();
    let asset_state = battle::load(&map0).unwrap();
    let mut state = state_from_asset_topology(&asset_state, false);
    let chapter = i64::from(snapshot.header.chapter);
    materialize_native_continue_field_boundary(Some(&mut state), &input, chapter).unwrap();
    materialize_native_continue_runtime_units(Some(&mut state), &input, &mut catalog).unwrap();
    assert_eq!(state.units.len(), 12, "runtime units");
    let got: Vec<&str> = state.units[..4].iter().map(|u| u.name.as_str()).collect();
    assert_eq!(got, ["索爾", "悠妮", "亞雷斯", "蓋亞"], "own runtime names");
    assert!(
        state.units[4].camp == battle::ENEMY
            && !state.units[4].has_native_identity
            && state.units[4].native_record_byte8 == 96,
        "enemy runtime +8 was misclassified: {:?}",
        state.units[4]
    );
    let scenario = battle::load_scenario(&ch01).unwrap();
    let item_rows = battle::load_native_item_effect_row_prefix(&rows).unwrap();
    materialize_native_continue_pending_groups(
        Some(&mut state),
        &input,
        chapter,
        Some(&asset_state),
        Some(&scenario),
        &item_rows,
    )
    .unwrap();
    let want: BTreeMap<i64, bool> = [3, 4, 5, 6, 7].into_iter().map(|g| (g, true)).collect();
    assert!(
        state.roster.len() == 15 && state.pending_groups == want,
        "original pending roster={} groups={:?}",
        state.roster.len(),
        state.pending_groups
    );
}

/// Go `continueRuntimeUnitsInput`。
pub(crate) fn continue_runtime_units_input(
    mutate: Option<fn(&mut [save::PersistentRecord])>,
) -> save::ContinueRuntimeInput {
    let mut records = vec![save::PersistentRecord::default(); 3];
    for (index, rec) in records.iter_mut().enumerate() {
        let record = &mut rec.raw;
        let index = index as u8;
        (record[0], record[1]) = (4 + index, 5 + index);
        (record[3], record[4]) = (index, index + 1);
        (record[5], record[6]) = (0, 0);
        (record[7], record[8]) = (0x60, 0x60);
        for slot in 0..8 {
            record[0x0a + slot * 2] = 0x80;
            record[0x0b + slot * 2] = 0xff;
        }
        (record[0x1f], record[0x20], record[0x21]) = (1, 7, 2);
        (record[0x3b], record[0x3c]) = (5, 9);
        for (offset, value) in [
            (0x37usize, 10i16),
            (0x39, 11),
            (0x3e, 12),
            (0x40, 13),
            (0x42, 14),
            (0x44, 15),
            (0x46, 16),
            (0x48, 17),
            (0x4a, 18),
            (0x4c, 19),
            (0x4e, 20),
        ] {
            record[offset..offset + 2].copy_from_slice(&(value as u16).to_le_bytes());
        }
    }
    (records[0].raw[6], records[0].raw[7], records[0].raw[8]) = (2, 9, 9);
    records[0].raw[2] = 0xee;
    (records[0].raw[0x0a], records[0].raw[0x0b]) = (0x40, 0x12);
    records[0].raw[0x22..0x28].copy_from_slice(&[1, 2, 3, 4, 5, 6]);
    records[0].raw[0x31..0x34].copy_from_slice(&[2, 0x34, 0x12]);
    records[0].raw[0x3d] = 7;
    records[1].raw[5] = 1;
    (records[1].raw[0], records[1].raw[1]) = (0xff, 0xfe);
    (records[1].raw[3], records[1].raw[4]) = (0xfd, 0xfc);
    records[1].raw[2] = 0xed;
    records[2].raw[2] = 0xec;
    if let Some(mutate) = mutate {
        mutate(&mut records);
    }

    let mut snapshot = save::CurrentSnapshot {
        header: save::CurrentRuntimeHeader {
            turn_counter: 7,
            runtime_count: records.len() as u8,
            chapter: 1,
            camera_x: 1,
            camera_y: 2,
            cursor_x: 8,
            cursor_y: 7,
            visible_cursor_x: 7,
            visible_cursor_y: 5,
            ..Default::default()
        },
        ..Default::default()
    };
    snapshot.runtime_records = records;
    save::build_continue_runtime_input(
        &snapshot,
        save::ContinueRuntimeContext {
            chapter: 1,
            field_width: 30,
            field_height: 20,
            selector_group_count: 128,
            title_timer_tick: -123,
            has_title_timer_tick: true,
        },
    )
    .unwrap()
}

/// Go `continueRuntimeUnitsState`。
pub(crate) fn continue_runtime_units_state(input: &save::ContinueRuntimeInput) -> battle::State {
    let mut state = continue_field_boundary_state();
    materialize_native_continue_field_boundary(Some(&mut state), input, 1).unwrap();
    state
}

#[test]
fn test_materialize_native_continue_runtime_units_preserves_raw_roster() {
    let Some(mut catalog) = native_runtime_catalog() else { return };
    let input = continue_runtime_units_input(None);
    let mut state = continue_runtime_units_state(&input);
    materialize_native_continue_runtime_units(Some(&mut state), &input, &mut catalog).unwrap();
    assert!(
        state.has_native_runtime_unit_projection && state.units.len() == 3 && state.native_map_selector_cache.is_some(),
        "runtime projection state={state:?}"
    );
    let got_slots: Vec<i64> = state.units.iter().map(|u| u.map_selector_slot).collect();
    assert_eq!(got_slots, [0, 1, 1], "selector slots");
    let own = &state.units[0];
    assert!(
        own.name == "悠妮"
            && own.camp == battle::OWN
            && own.has_native_identity
            && own.native_identity == 9
            && own.has_native_record_byte8
            && own.native_record_byte8 == 9
            && own.map_selector_key == 9
            && own.battle_fig == 9
            && own.map_selector_slot != i64::from(input.runtime_records[0].raw.raw[2])
            && own.native_map_presentation.motion == 1
            && own.native_transient == [1, 2, 3, 4, 5, 6]
            && own.has_native_record_death_effect
            && own.native_record_death_effect == [2, 0x34, 0x12]
            && own.has_native_record_byte3_d
            && own.native_record_byte3_d == 7
            && own.native_record_word42 == 14
            && own.native_record_word46 == 16
            && own.inventory == vec![0x12]
            && own.equipped[0],
        "own runtime unit={own:?}"
    );
    let inactive = &state.units[1];
    assert!(
        !inactive.on_field
            && !inactive.has_native_identity
            && inactive.native_record_byte8 == 0x60
            && inactive.native_map_presentation.x == 0xff
            && inactive.native_map_presentation.pose == 0xfd,
        "inactive runtime unit={inactive:?}"
    );
    assert!(
        !state.has_native_map_cycle_state
            && !state.has_native_terrain_phase_state
            && !state.has_native_map_binary_timing_state,
        "runtime unit projection guessed unresolved map timing"
    );

    let raw_records = battle::native_ai_scoring_records(&state.units).unwrap();
    assert!(
        raw_records[8] == 9 && raw_records[0x50 + 8] == 0x60,
        "raw +8 projection was lost: {:x} {:x}",
        raw_records[8],
        raw_records[0x58]
    );
}

#[test]
fn test_materialize_native_continue_runtime_units_rejects_atomically() {
    type Mutate = fn(&mut [save::PersistentRecord]);
    let tests: [(&str, Mutate); 2] =
        [("unknown camp", |records| records[2].raw[6] = 3), ("unknown class", |records| records[2].raw[0x20] = 0xff)];
    for (name, mutate) in tests {
        let Some(mut catalog) = native_runtime_catalog() else { return };
        let input = continue_runtime_units_input(Some(mutate));
        let mut state = continue_runtime_units_state(&input);
        let before_units = state.units.clone();
        assert!(
            materialize_native_continue_runtime_units(Some(&mut state), &input, &mut catalog).is_err(),
            "{name}: invalid runtime roster was accepted"
        );
        assert!(
            state.units == before_units
                && state.native_map_selector_cache.is_none()
                && !state.has_native_runtime_unit_projection,
            "{name}: failed runtime projection mutated state={state:?}"
        );
    }
}
