//! 移植自 `remake/internal/battle/native_ai_phase_diagnostic_test.go`。
//! `nativeCompositionBaseFlagsForTest` 在 `native_ai_runtime/d_test_support.rs`。

use super::*;
use crate::battle::native_ai_runtime::d_test_support::*;

/// Go `nativeAIPhaseDiagnosticFixture`。
fn native_ai_phase_diagnostic_fixture() -> State {
    let mut units = vec![complete_native_ai_scoring_unit(), complete_native_ai_scoring_unit()];
    for (index, unit) in units.iter_mut().enumerate() {
        unit.native_map_presentation.x = index as u8;
        unit.native_map_presentation.y = 0;
        unit.native_record_byte6 = 1 - index as u8;
        unit.native_transient[4] = 0;
        unit.native_command_mask = [0; 5];
        unit.native_inventory_flags = vec![0x80; 8];
        unit.inventory_slots = vec![0xff; 8];
    }
    State {
        w: 2,
        h: 1,
        units,
        native_composition_event_bytes: vec![0; 2],
        native_terrain_move_codes: vec![0; 2],
        ..Default::default()
    }
}

fn input(unit_index: i64, cost_row: &[u8]) -> NativeAIPhaseZeroInput {
    NativeAIPhaseZeroInput { unit_index, cost_row: cost_row.to_vec(), skip: None }
}

/// Go：`TestBuildNativeAIPhaseDiagnosticPlanRequiresExactEligibleInputs`
#[test]
fn build_native_ai_phase_diagnostic_plan_requires_exact_eligible_inputs() {
    let st = native_ai_phase_diagnostic_fixture();
    let book = native_command_book_ids();
    let item_rows = vec![0u8; NATIVE_ITEM_EFFECT_ROW_SIZE as usize];
    let cost_row = vec![0u8; NATIVE_MOVEMENT_COST_ROW_SIZE as usize];

    assert!(
        build_native_ai_phase_diagnostic_plan(Some(&st), &book, &item_rows, &[]).is_err(),
        "missing eligible selector-zero input was accepted"
    );
    assert!(
        build_native_ai_phase_diagnostic_plan(Some(&st), &book, &item_rows, &[input(0, &cost_row)]).is_err(),
        "selector-one input was accepted as selector-zero provenance"
    );
    assert!(
        build_native_ai_phase_diagnostic_plan(
            Some(&st),
            &book,
            &item_rows,
            &[input(1, &cost_row), input(1, &cost_row)]
        )
        .is_err(),
        "duplicate selector-zero input was accepted"
    );
}

/// Go：`TestBuildNativeAIPhaseDiagnosticPlanIsReadOnlyAndKeepsThreePasses`
#[test]
fn build_native_ai_phase_diagnostic_plan_is_read_only_and_keeps_three_passes() {
    let st = native_ai_phase_diagnostic_fixture();
    let before = native_ai_scoring_records(&st.units).unwrap();
    let (plan, diagnostics) = build_native_ai_phase_diagnostic_plan(
        Some(&st),
        &native_command_book_ids(),
        &[0u8; NATIVE_ITEM_EFFECT_ROW_SIZE as usize],
        &[input(1, &[0u8; NATIVE_MOVEMENT_COST_ROW_SIZE as usize])],
    )
    .unwrap();
    let after = native_ai_scoring_records(&st.units).unwrap();
    assert!(before == after, "native AI diagnostic plan mutated battle state");
    assert!(
        plan.selector_one.len() == 1
            && plan.selector_one[0].unit_index == 0
            && plan.zero_preselection.len() == 1
            && plan.zero_preselection[0].unit_index == 1
            && !plan.zero_preselection[0].run_unit_action
            && plan.zero_second_pass.len() == 1
            && plan.zero_second_pass[0].unit_index == 1,
        "phase plan={plan:?}"
    );
    assert!(
        diagnostics.len() == 1
            && diagnostics[0].unit_index == 1
            && diagnostics[0].command_mask_result.max_score == 0
            && diagnostics[0].item_command_result.max_score == 0,
        "diagnostics={diagnostics:?}"
    );
}

/// Go：`TestMap0CrossFixtureBuildsNativeAIPhaseDiagnosticGate`
#[test]
fn map0_cross_fixture_builds_native_ai_phase_diagnostic_gate() {
    let (Some(mut st), Some(spells), Some(item_path), Some(rows_path)) = (
        load_asset_state("maps/map0/map0_units.json"),
        asset_path("spells.json"),
        asset_path("data/native_item_effect_rows.json"),
        asset_path("data/native_movement_cost_rows.json"),
    ) else {
        return;
    };
    let actor = 23usize;
    for (index, unit) in st.units.iter_mut().enumerate() {
        unit.materialize_native_map_presentation().unwrap();
        if unit.native_record_byte6 == 0 && index != actor {
            unit.native_record_byte5 |= 0x80;
        }
    }
    {
        let unit = &mut st.units[actor];
        unit.native_command_mask = [1, 0, 0, 0, 0];
        unit.mp = 255;
        unit.inventory_slots = vec![79, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff];
        unit.native_inventory_flags = vec![0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80];
    }
    let book = load_native_command_records(&spells).unwrap();
    let item_rows = load_native_item_effect_row_prefix(&item_path).unwrap();
    let cost_rows = load_native_movement_cost_rows(&rows_path).unwrap();
    let before = native_ai_scoring_records(&st.units).unwrap();
    let (plan, diagnostics) =
        build_native_ai_phase_diagnostic_plan(Some(&st), &book, &item_rows, &[input(actor as i64, &cost_rows[0])])
            .unwrap();
    let after = native_ai_scoring_records(&st.units).unwrap();
    assert!(before == after, "map0 diagnostic plan mutated battle state");
    assert!(
        plan.zero_preselection.len() == 1
            && plan.zero_preselection[0].unit_index == actor as i64
            && plan.zero_preselection[0].run_unit_action
            && plan.zero_second_pass.len() == 1
            && plan.zero_second_pass[0].unit_index == actor as i64,
        "map0 phase plan={plan:?}"
    );
    assert!(diagnostics.len() == 1, "map0 diagnostics={diagnostics:?}");
    let got = &diagnostics[0];
    let c = &got.command_mask_result;
    assert!(
        c.max_score == 96
            && c.has_positive_winner
            && c.positive_winner.command_id == 0
            && c.positive_winner.x == 23
            && c.positive_winner.y == 14,
        "map0 command-mask diagnostic={c:?}"
    );
    assert!(
        got.item_command_result.max_score == 0 && !got.item_command_result.has_positive_winner,
        "map0 item-command diagnostic={:?}",
        got.item_command_result
    );
    assert!(
        got.phase_gate_score_pair.spell_candidate_score_53c23 == 96
            && got.phase_gate_score_pair.item_command_score_53c33 == 0,
        "map0 score pair={:?}",
        got.phase_gate_score_pair
    );
}

/// Go：`TestMap19RealAssetInputsProduceZeroAIScores`
#[test]
fn map19_real_asset_inputs_produce_zero_ai_scores() {
    let (Some(mut st), Some(spells), Some(item_path), Some(rows_path)) = (
        load_asset_state("maps/map19/map19_units.json"),
        asset_path("spells.json"),
        asset_path("data/native_item_effect_rows.json"),
        asset_path("data/native_movement_cost_rows.json"),
    ) else {
        return;
    };
    let events = &st.native_composition_event_bytes;
    let nonzero_flags = events.iter().filter(|&&f| f != 0).count();
    assert!(
        events.len() == 40 * 40 && nonzero_flags == 7 && events[122] == 5 && events[310] == 7 && events[711] == 6,
        "map19 composition event bytes lack FDFIELD provenance"
    );
    for unit in &mut st.units {
        unit.materialize_native_map_presentation().unwrap();
    }
    let records = native_ai_scoring_records(&st.units).unwrap();
    let book = load_native_command_records(&spells).unwrap();
    let item_rows = load_native_item_effect_row_prefix(&item_path).unwrap();
    let cost_rows = load_native_movement_cost_rows(&rows_path).unwrap();
    let actor = 55usize;
    {
        let unit = &st.units[actor];
        assert!(
            unit.native_record_byte8 == 92
                && unit.has_native_record_byte8
                && !unit.has_native_identity
                && unit.native_command_mask == [4, 0, 0, 8, 0]
                && unit.mp == 288,
            "map19 actor55={unit:?}"
        );
    }
    let base = native_composition_base_flags_for_test(&st);
    let count = st.units.len() as i64;
    let command_mask = score_native_ai1598_a(
        st.w,
        st.h,
        &records,
        count,
        actor as i64,
        0,
        Some(&st.units[actor]),
        &book,
        &base,
        &st.native_terrain_move_codes,
        &cost_rows[0],
        None,
    )
    .unwrap();
    let item_command =
        score_native_ai1567_e(st.w, st.h, &records, count, actor as i64, 0, &item_rows, &book, &base).unwrap();
    assert!(
        command_mask.max_score == 0
            && !command_mask.has_positive_winner
            && item_command.max_score == 0
            && !item_command.has_positive_winner,
        "map19 real scores command-mask={command_mask:?} item-command={item_command:?}"
    );
}
