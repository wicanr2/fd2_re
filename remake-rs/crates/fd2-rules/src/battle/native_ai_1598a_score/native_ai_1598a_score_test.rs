//! 移植自 `remake/internal/battle/native_ai_1598a_score_test.go`。

use super::*;
use crate::battle::native_ai_runtime::d_test_support::*;

/// Go：`TestScoreNativeAI1598ARejectsAllZeroWinner`
#[test]
fn score_native_ai1598_a_rejects_all_zero_winner() {
    let mut unit = complete_native_ai_scoring_unit();
    unit.native_command_mask = [0, 0, 0, 1, 0];
    unit.native_map_presentation.x = 0;
    unit.native_map_presentation.y = 0;
    unit.mp = 255;
    let mut book = native_command_book_ids();
    book[24] = NativeCommandRecord { id: 24, selection_mode: 0, effect_mode: 0, target_code: 1, ..Default::default() };
    let records = native_ai_scoring_records(std::slice::from_ref(&unit)).unwrap();
    let cost = vec![0u8; NATIVE_MOVEMENT_COST_ROW_SIZE as usize];
    let got = score_native_ai1598_a(1, 1, &records, 1, 0, 1, Some(&unit), &book, &[0], &[0], &cost, None).unwrap();
    assert!(got.max_score == 0 && !got.has_positive_winner, "all-zero result invented winner: {got:?}");
}

/// Go：`TestScoreNativeAI1598AFailsClosedWhenActorRecordDisagrees`
#[test]
fn score_native_ai1598_a_fails_closed_when_actor_record_disagrees() {
    let mut unit = complete_native_ai_scoring_unit();
    unit.native_map_presentation.x = 0;
    unit.native_map_presentation.y = 0;
    let records = native_ai_scoring_records(std::slice::from_ref(&unit)).unwrap();
    let book = native_command_book_ids();
    unit.mp += 1;
    let cost = vec![0u8; NATIVE_MOVEMENT_COST_ROW_SIZE as usize];
    assert!(
        score_native_ai1598_a(1, 1, &records, 1, 0, 0, Some(&unit), &book, &[0], &[0], &cost, None).is_err(),
        "actor MP/record mismatch was accepted"
    );
}

/// Go：`TestMap0AssetsProducePositiveNativeAI1598AScoreForCommand0`
#[test]
fn map0_assets_produce_positive_native_ai1598_a_score_for_command0() {
    let (Some(mut st), Some(spells), Some(rows_path)) = (
        load_asset_state("maps/map0/map0_units.json"),
        asset_path("spells.json"),
        asset_path("data/native_movement_cost_rows.json"),
    ) else {
        return;
    };
    let actor = 23usize;
    {
        let unit = &st.units[actor];
        assert!(
            unit.native_record_byte8 == 103 && unit.has_native_record_byte8 && !unit.has_native_identity,
            "map0 actor raw +8={} want 103",
            unit.native_record_byte8
        );
    }
    st.units[actor].native_command_mask = [1, 0, 0, 0, 0];
    st.units[actor].mp = 255;
    for unit in &mut st.units {
        unit.materialize_native_map_presentation().unwrap();
    }
    let records = native_ai_scoring_records(&st.units).unwrap();
    let book = load_native_command_records(&spells).unwrap();
    let rows = load_native_movement_cost_rows(&rows_path).unwrap();
    let base = native_composition_base_flags_for_test(&st);
    let count = st.units.len() as i64;
    let got = score_native_ai1598_a(
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
        &rows[0],
        None,
    )
    .unwrap();
    assert!(got.has_positive_winner && got.max_score == 96, "map0 command0 native score={got:?}");
    assert!(
        got.positive_winner.command_id == 0 && got.positive_winner.x == 23 && got.positive_winner.y == 14,
        "map0 command0 native winner={got:?}"
    );
    // 0x159A5 push 0 → 0x4E555：施法落點固定用成本列 0，行動者自己的移動成本列
    // （這裡用全部是 10 的列代表崎嶇地形）不能讓落點變少。
    let heavy = vec![10u8; NATIVE_MOVEMENT_COST_ROW_SIZE as usize];
    let again = score_native_ai1598_a(
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
        &heavy,
        None,
    );
    assert!(matches!(&again, Ok(a) if *a == got), "行動者成本列影響了 0x1598a 落點：{again:?}，要 {got:?}");
}
