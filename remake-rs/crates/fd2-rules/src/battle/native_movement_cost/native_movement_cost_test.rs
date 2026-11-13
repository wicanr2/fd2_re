//! 移植自 `remake/internal/battle/native_movement_cost_test.go`。

use super::*;
use crate::test_support::remake_asset;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go `nativeMovementRowsForTest`
fn native_movement_rows_for_test() -> Vec<Vec<u8>> {
    vec![vec![0u8; NATIVE_MOVEMENT_COST_ROW_SIZE as usize]; NATIVE_MOVEMENT_COST_ROW_COUNT as usize]
}

/// Go：`TestNativeRelocationDestinationAllowedMatchesMode6Gates`——0x115B6 mode 6 的 Enter 判準：
/// 目的格沒有別的活記錄，而且該職業的地形成本**不是 20**（20 = 不可通行，`0x11706 cmp eax,0x14`
/// 相等就回輸入迴圈）。
#[test]
fn native_relocation_destination_allowed_matches_mode6_gates() {
    let mut records = vec![0u8; 3 * RS];
    (records[RS], records[RS + 1], records[RS + 0x20]) = (4, 5, 9);
    let mut rows = native_movement_rows_for_test();
    rows[9][3] = 1; // 可通行
    assert_eq!(native_relocation_destination_allowed(&records, 3, 1, 8, 9, 3, &rows), Ok(true), "allowed");
    (records[2 * RS], records[2 * RS + 1], records[2 * RS + 5]) = (8, 9, 0);
    assert_eq!(native_relocation_destination_allowed(&records, 3, 1, 8, 9, 3, &rows), Ok(false), "occupied");
    records[2 * RS + 5] = 1;
    assert_eq!(native_relocation_destination_allowed(&records, 3, 1, 8, 9, 3, &rows), Ok(true), "bit0 occupant");
    rows[9][3] = 20; // 不可通行 → 拒絕
    assert_eq!(native_relocation_destination_allowed(&records, 3, 1, 8, 9, 3, &rows), Ok(false), "不可通行地形");
}

/// Go：`TestNativeRelocationDestinationSelectorOverrides`（`t.Run` 子測試改成迴圈）
#[test]
fn native_relocation_destination_selector_overrides() {
    let mut records = vec![0u8; RS];
    let mut rows = native_movement_rows_for_test();
    // 只有被選中的那一列在這個地形碼上是 20（不可通行）；選錯列就會放行。
    (rows[1][2], rows[19][2]) = (20, 20);
    for (name, unit7, race, class_id, want_selector) in
        [("unit7", 0x1c, 4, 0x13, 1usize), ("class", 0, 0, 0x13, 19), ("race4", 0, 4, 3, 19), ("race5", 0, 5, 3, 19)]
    {
        records.fill(0);
        (records[0x07], records[0x1f], records[0x20]) = (unit7, race, class_id);
        let allowed = native_relocation_destination_allowed(&records, 1, 0, 1, 1, 2, &rows);
        assert!(allowed == Ok(false) && rows[want_selector][2] == 20, "{name}: allowed={allowed:?}");
    }
}

/// Go：`TestLoadNativeMovementCostRowsFixture`
#[test]
fn load_native_movement_cost_rows_fixture() {
    let Some(path) = remake_asset("data/native_movement_cost_rows.json") else { return };
    let rows = load_native_movement_cost_rows(&path).unwrap();
    assert!(rows.len() == 29 && rows[0].len() == 20, "shape={}/{}", rows.len(), rows[0].len());
    for (column, &value) in rows[0].iter().enumerate() {
        assert_eq!(value, 1, "row0[{column}]");
    }
    // 錨點是地形碼語意本身：碼 1／5 不可移動（20），碼 0 正常（1），碼 4 沼澤
    // 步行 -2（2）。這組對照同時來自 modify2.md 的靜態定義與 notes.md 的玩家實測
    // 扣減值；早先的匯出整份早一個位元組，碼 1 變成 1、碼 2 變成 20，海可以走、
    // 森林反而是牆。
    assert!(rows[1][1] == 20 && rows[1][5] == 20, "row1 不可移動錨點={}/{}", rows[1][1], rows[1][5]);
    assert!(
        rows[1][0] == 1 && rows[1][2] == 1 && rows[1][3] == 2 && rows[1][4] == 2,
        "row1 正常/森林/沼澤={}/{}/{}/{}",
        rows[1][0],
        rows[1][2],
        rows[1][3],
        rows[1][4]
    );
    // 騎兵列：森林（碼 2）多扣 1、沼澤（碼 4）多扣 1，對應 notes.md 的騎兵 -2／-3。
    assert!(
        rows[3][2] == 2 && rows[3][3] == 3 && rows[3][4] == 3,
        "騎兵列={}/{}/{}",
        rows[3][2],
        rows[3][3],
        rows[3][4]
    );
    // 飛行列：碼 1..4 一律 1，只有碼 5 仍然擋。
    for (code, &cost) in rows[19].iter().enumerate().take(5) {
        assert_eq!(cost, 1, "飛行列 碼{code}，應為 1");
    }
    assert_eq!(rows[19][5], 20, "飛行列 碼5，應為 20");
}
