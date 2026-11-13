//! 移植自 `remake/internal/battle/native_ai_scored_candidates_test.go`。

use super::*;
use crate::test_support::remake_asset;

const RS: usize = NATIVE_RECORD_SIZE as usize;

fn records_from(rows: &[(u8, u8, u8)]) -> Vec<u8> {
    let mut records = vec![0u8; rows.len() * RS];
    for (i, &(x, y, side)) in rows.iter().enumerate() {
        (records[i * RS], records[i * RS + 1], records[i * RS + 6]) = (x, y, side);
    }
    records
}

/// Go `nativeAIScoredCandidateFixture`：回傳 records、flags、terrain。
fn native_ai_scored_candidate_fixture() -> (Vec<u8>, Vec<u8>, Vec<u8>) {
    (records_from(&[(1, 1, 0), (0, 1, 2), (2, 1, 1), (1, 0, 2)]), vec![0; 9], vec![0; 9])
}

fn ones_cost_row() -> Vec<u8> {
    vec![1u8; NATIVE_MOVEMENT_COST_ROW_SIZE as usize]
}

fn group(x: i64, y: i64, targets: &[u8]) -> NativeAIScoredCommandCandidateGroup {
    NativeAIScoredCommandCandidateGroup { destination: Cell { x, y }, target_indices: targets.to_vec() }
}

/// Go：`TestNativeAIScoredCommandCandidateGroupsPreservesGeometryAndRawOrder`
#[test]
fn native_ai_scored_command_candidate_groups_preserves_geometry_and_raw_order() {
    let (records, flags, terrain) = native_ai_scored_candidate_fixture();
    let command = NativeCommandRecord { selection_mode: 1, effect_mode: 1, target_code: 0, ..Default::default() };
    let groups =
        native_ai_scored_command_candidate_groups(3, 3, &records, 4, 0, 0, command, &flags, &terrain, &ones_cost_row())
            .unwrap();
    let want = vec![group(1, 0, &[3]), group(0, 1, &[1]), group(1, 1, &[1, 2, 3]), group(2, 1, &[2])];
    assert!(groups == want, "groups={groups:?} want={want:?}");
}

/// Go：`TestNativeAIScoredCommandCandidateGroupsSelectorOneUsesCommandTargetCode`
#[test]
fn native_ai_scored_command_candidate_groups_selector_one_uses_command_target_code() {
    let (records, flags, terrain) = native_ai_scored_candidate_fixture();
    let command = NativeCommandRecord { selection_mode: 0, effect_mode: 1, target_code: 0, ..Default::default() };
    let groups =
        native_ai_scored_command_candidate_groups(3, 3, &records, 4, 0, 1, command, &flags, &terrain, &ones_cost_row())
            .unwrap();
    assert!(groups.len() == 1 && groups[0].target_indices == [0], "selector-one target groups={groups:?}");
}

/// Go：`TestNativeAIScoredCommandCandidateGroupsDoesNotInventPhysicalOccupancyFlags`
#[test]
fn native_ai_scored_command_candidate_groups_does_not_invent_physical_occupancy_flags() {
    let records = records_from(&[(0, 0, 0), (1, 0, 2)]);
    let command = NativeCommandRecord { selection_mode: 1, effect_mode: 0, target_code: 0, ..Default::default() };
    let groups =
        native_ai_scored_command_candidate_groups(2, 1, &records, 2, 0, 0, command, &[0; 2], &[0; 2], &ones_cost_row())
            .unwrap();
    assert!(
        groups.len() == 1 && groups[0].destination == Cell { x: 1, y: 0 },
        "roster-derived blocker was invented: {groups:?}"
    );
}

/// Go：`TestNativeAIScoredCommandCandidateGroupsFailsClosedOnMalformedInputs`
#[test]
fn native_ai_scored_command_candidate_groups_fails_closed_on_malformed_inputs() {
    let (records, flags, terrain) = native_ai_scored_candidate_fixture();
    let cost = vec![0u8; NATIVE_MOVEMENT_COST_ROW_SIZE as usize];
    let command = NativeCommandRecord { selection_mode: 1, effect_mode: 1, ..Default::default() };
    assert!(
        native_ai_scored_command_candidate_groups(
            3,
            3,
            &records[..records.len() - 1],
            4,
            0,
            0,
            command,
            &flags,
            &terrain,
            &cost
        )
        .is_err(),
        "short record buffer accepted"
    );
    assert!(
        native_ai_scored_command_candidate_groups(3, 3, &records, 4, 0, 2, command, &flags, &terrain, &cost).is_err(),
        "unknown selector accepted"
    );
}

/// Go：`TestMap0AssetsBuildNativeAIScoredCommandCandidateGroups`
#[test]
fn map0_assets_build_native_ai_scored_command_candidate_groups() {
    let (Some(units_path), Some(spells_path), Some(rows_path)) = (
        remake_asset("maps/map0/map0_units.json"),
        remake_asset("spells.json"),
        remake_asset("data/native_movement_cost_rows.json"),
    ) else {
        return;
    };
    let mut st = load(&units_path).unwrap();
    for unit in &mut st.units {
        unit.materialize_native_map_presentation().unwrap();
    }
    let records = native_ai_scoring_records(&st.units).unwrap();
    let book = load_native_command_records(&spells_path).unwrap();
    let rows = load_native_movement_cost_rows(&rows_path).unwrap();
    let (mut actor, mut target) = (-1i64, -1i64);
    for (index, unit) in st.units.iter().enumerate() {
        if actor < 0 && unit.native_record_byte8 == 103 && unit.has_native_record_byte8 && !unit.has_native_identity {
            actor = index as i64;
        }
        if target < 0 && unit.native_record_byte6 == 1 && unit.has_native_record_byte6 {
            target = index as i64;
        }
    }
    assert!(actor >= 0 && target >= 0, "map0 native actor/target anchors missing: actor={actor} target={target}");
    // Go `nativeCompositionBaseFlagsForTest`（`native_ai_phase_diagnostic_test.go`）。
    let base_flags = native_composition_base_flags(st.w, st.h, &st.native_composition_event_bytes).unwrap();
    let groups = native_ai_scored_command_candidate_groups(
        st.w,
        st.h,
        &records,
        st.units.len() as i64,
        actor,
        0,
        book[0],
        &base_flags,
        &st.native_terrain_move_codes,
        &rows[0],
    )
    .unwrap();
    let t = target as usize;
    let want_cell = Cell { x: records[t * RS] as i64, y: records[t * RS + 1] as i64 };
    let found = groups.iter().any(|group| {
        group.destination == want_cell && group.target_indices.iter().any(|&index| index as i64 == target)
    });
    assert!(found, "map0 candidate at {want_cell:?} for target {target} missing from {groups:?}");
}
