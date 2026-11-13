//! 移植自 `remake/internal/battle/native_battle_result_test.go`。

use super::*;
use crate::test_support::remake_asset;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestNativeBattleResultCode205B4KeepsZeroForAnyActiveRawCampZero`
#[test]
fn native_battle_result_code205_b4_keeps_zero_for_any_active_raw_camp_zero() {
    let mut records = vec![0u8; 3 * RS];
    records[6] = 1;
    records[RS + 6] = 0;
    records[RS + 5] = 0;
    records[2 * RS + 6] = 0;
    records[2 * RS + 5] = 1;
    assert_eq!(native_battle_result_code205_b4(&records, 3), Ok(0));
}

/// Go：`TestNativeBattleResultCode205B4ReturnsTwoWithoutActiveRawCampZero`
#[test]
fn native_battle_result_code205_b4_returns_two_without_active_raw_camp_zero() {
    let mut records = vec![0u8; 2 * RS];
    records[6] = 1;
    records[RS + 6] = 0;
    records[RS + 5] = 1;
    assert_eq!(native_battle_result_code205_b4(&records, 2), Ok(2));
}

/// Go：`TestNativeBattleResultCode205B4RecordZeroBitOverridesWithOne`
#[test]
fn native_battle_result_code205_b4_record_zero_bit_overrides_with_one() {
    let mut records = vec![0u8; 2 * RS];
    records[5] = 1;
    records[6] = 1;
    records[RS + 6] = 0;
    records[RS + 5] = 0;
    assert_eq!(native_battle_result_code205_b4(&records, 2), Ok(1));
}

/// Go：`TestNativeBattleResultCode205B4IgnoresRawBitSeven`
#[test]
fn native_battle_result_code205_b4_ignores_raw_bit_seven() {
    let mut records = vec![0u8; 2 * RS];
    records[6] = 1;
    records[RS + 6] = 0;
    records[RS + 5] = 0x80;
    assert_eq!(native_battle_result_code205_b4(&records, 2), Ok(0));
}

/// Go：`TestNativeBattleResultCode205B4RejectsMalformedRecords`
#[test]
fn native_battle_result_code205_b4_rejects_malformed_records() {
    for (records, count) in [(Vec::new(), 0), (vec![0u8; RS - 1], 1), (vec![0u8; RS], 2)] {
        assert!(
            native_battle_result_code205_b4(&records, count).is_err(),
            "accepted malformed count={count} bytes={}",
            records.len()
        );
    }
}

/// Go：`TestMap0AssetsAnchorNativeBattleResultCode205B4`
#[test]
fn map0_assets_anchor_native_battle_result_code205_b4() {
    let Some(path) = remake_asset("maps/map0/map0_units.json") else { return };
    let mut st = load(&path).unwrap();
    for unit in &mut st.units {
        unit.materialize_native_map_presentation().unwrap();
    }
    let records = native_ai_scoring_records(&st.units).unwrap();
    assert_eq!(native_battle_result_code205_b4(&records, st.units.len() as i64), Ok(0), "map0 result");
}
