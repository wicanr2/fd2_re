//! 移植自 `remake/internal/battle/native_ai_item_preselection_test.go`。

use super::*;
use crate::battle::native_ai_runtime::d_test_support::*;

const ROW: usize = NATIVE_ITEM_EFFECT_ROW_SIZE as usize;

/// Go `nativeAI1567EFixture`：回傳 records、itemRows、flags。
fn native_ai1567_e_fixture() -> (Vec<u8>, Vec<u8>, Vec<u8>) {
    let mut records = vec![0u8; 3 * RS];
    records[0] = 0;
    records[1] = 0;
    records[6] = 2;
    records[0x0a] = 0;
    records[0x0b] = 0;
    for slot in 1..8 {
        records[0x0a + slot * 2] = 0x80;
        records[0x0b + slot * 2] = 0xff;
    }
    records[RS] = 1;
    records[RS + 1] = 0;
    records[RS + 6] = 0;
    put_record16(&mut records, 1, 0x40, 30);
    put_record16(&mut records, 1, 0x42, 90);
    records[2 * RS] = 2;
    records[2 * RS + 1] = 0;
    records[2 * RS + 6] = 1;

    let mut item_rows = vec![0u8; ROW];
    item_rows[0x0d] = 5;
    item_rows[0x10] = 1;
    item_rows[0x11] = 2;
    item_rows[0x12] = 0;
    (records, item_rows, vec![0u8; 3])
}

/// Go：`TestScoreNativeAI1567ELowCommandKeepsFirstStrictWinner`
#[test]
fn score_native_ai1567_e_low_command_keeps_first_strict_winner() {
    let (mut records, item_rows, flags) = native_ai1567_e_fixture();
    records[0x0c] = 0;
    records[0x0d] = 0;
    let got = score_native_ai1567_e(3, 1, &records, 3, 0, 0, &item_rows, &[], &flags).unwrap();
    assert!(
        got.has_positive_winner
            && got.max_score == 8
            && got.x == 1
            && got.y == 0
            && got.inventory_slot == 0
            && got.item_id == 0,
        "low-command result={got:?}"
    );
}

/// Go：`TestScoreNativeAI1567EHighCommandUsesActorToDestinationLine`
#[test]
fn score_native_ai1567_e_high_command_uses_actor_to_destination_line() {
    let (mut records, _, flags) = native_ai1567_e_fixture();
    records[RS] = 2;
    records[RS + 6] = 2;
    put_record16(&mut records, 1, 0x40, 20);
    let mut item_rows = vec![0u8; ROW];
    item_rows[0x0d] = 0x18;
    item_rows[0x0e] = 20;
    item_rows[0x10] = 0x12;
    let got = score_native_ai1567_e(3, 1, &records, 3, 0, 0, &item_rows, &[], &flags).unwrap();
    assert!(
        got.has_positive_winner && got.max_score == 0x12 && got.x == 1 && got.y == 0,
        "high-command result={got:?}"
    );
}

/// Go：`TestScoreNativeAI1567EHighCommandRejectsRawZeroAndBlockedDirections`
#[test]
fn score_native_ai1567_e_high_command_rejects_raw_zero_and_blocked_directions() {
    let (mut records, _, mut flags) = native_ai1567_e_fixture();
    records[RS] = 2;
    records[RS + 6] = 0;
    records[2 * RS + 5] = 1;
    let mut item_rows = vec![0u8; ROW];
    (item_rows[0x0d], item_rows[0x0e], item_rows[0x10]) = (0x18, 20, 0x12);
    for target_camp in [0u8, 1, 2] {
        records[RS + 6] = target_camp;
        flags[1] = NATIVE_COMMAND_GRID_BLOCKED;
        let got = score_native_ai1567_e(3, 1, &records, 3, 0, 0, &item_rows, &[], &flags);
        assert!(matches!(&got, Ok(g) if !g.has_positive_winner), "blocked direction camp={target_camp} got={got:?}");
        flags[1] = 0;
        let got = score_native_ai1567_e(3, 1, &records, 3, 0, 0, &item_rows, &[], &flags);
        assert!(
            matches!(&got, Ok(g) if g.has_positive_winner == (target_camp != 0)),
            "selector zero camp={target_camp} got={got:?}"
        );
    }
}

/// Go：`TestScoreNativeAI1567ESelectorOneKeepsRowTargetCode`
#[test]
fn score_native_ai1567_e_selector_one_keeps_row_target_code() {
    let (mut records, mut item_rows, flags) = native_ai1567_e_fixture();
    item_rows[0x10] = 2;
    item_rows[0x11] = 2;
    put_record16(&mut records, 2, 0x40, 30);
    put_record16(&mut records, 2, 0x42, 90);
    let got = score_native_ai1567_e(3, 1, &records, 3, 0, 1, &item_rows, &[], &flags).unwrap();
    assert!(got.has_positive_winner && got.max_score == 8 && got.x == 2 && got.y == 0, "selector-one result={got:?}");
}

/// Go：`TestScoreNativeAI1567EFollowsCountSizedRawPrefix`
#[test]
fn score_native_ai1567_e_follows_count_sized_raw_prefix() {
    let (mut records, item_rows, flags) = native_ai1567_e_fixture();
    records[0x0a] = 0x80;
    records[0x0b] = 0xff;
    records[0x0c] = 0;
    records[0x0d] = 0;
    assert!(
        score_native_ai1567_e(3, 1, &records, 3, 0, 0, &item_rows, &[], &flags).is_err(),
        "native count-sized scan accepted unavailable stale item row"
    );
}

/// Go：`TestMap0AssetsAnchorNativeAI1567EWithTrackedItem79`
#[test]
fn map0_assets_anchor_native_ai1567_e_with_tracked_item79() {
    let (Some(mut st), Some(item_path), Some(spells)) = (
        load_asset_state("maps/map0/map0_units.json"),
        asset_path("data/native_item_effect_rows.json"),
        asset_path("spells.json"),
    ) else {
        return;
    };
    let actor = 23usize;
    st.units[actor].inventory_slots = vec![79, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff];
    st.units[actor].native_inventory_flags = vec![0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80];
    for unit in &mut st.units {
        unit.materialize_native_map_presentation().unwrap();
    }
    let records = native_ai_scoring_records(&st.units).unwrap();
    let item_rows = load_native_item_effect_row_prefix(&item_path).unwrap();
    let book = load_native_command_records(&spells).unwrap();
    let base = native_composition_base_flags_for_test(&st);
    let got =
        score_native_ai1567_e(st.w, st.h, &records, st.units.len() as i64, actor as i64, 0, &item_rows, &book, &base)
            .unwrap();
    // The former score8/(19,15) assertion selected a raw-zero ally.
    // 0x14AE4..0x14AF4 excludes that record for selector zero.
    assert!(
        !got.has_positive_winner && got.max_score == 0 && got.target_indices.is_empty(),
        "map0 item79 result={got:?}"
    );
}
