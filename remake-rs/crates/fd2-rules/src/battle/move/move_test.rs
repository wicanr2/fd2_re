//! 移植自 `remake/internal/battle/move_test.go`。

use super::*;
use crate::test_support::temp_dir;

/// Go：`TestMoveCost_NoTerrainData_DefaultsFlat`
///
/// State.Cost 為 nil(如舊測試直接手寫 State{}、或 Load 找不到 map.json)時,MoveCost 全平地回 1,
/// 越界也回 1(不 panic)。
#[test]
fn move_cost_no_terrain_data_defaults_flat() {
    let st = State { w: 5, h: 5, ..Default::default() };
    assert_eq!(st.move_cost(2, 2), 1, "MoveCost");
    assert_eq!(st.move_cost(-1, 0), 1, "MoveCost(越界)");
    assert_eq!(st.move_cost(99, 99), 1, "MoveCost(越界)");
}

/// Go：`TestMoveCost_FromTable`
///
/// 有 Cost 表時逐格查值(水=不可通行的 99、沼澤=2、平地=1)。
#[test]
fn move_cost_from_table() {
    let st = State { w: 3, h: 1, cost: Some(vec![1, 99, 2]), ..Default::default() };
    for ((x, y), want) in [((0, 0), 1), ((1, 0), 99), ((2, 0), 2)] {
        assert_eq!(st.move_cost(x, y), want, "MoveCost({x},{y})");
    }
}

fn walker(x: i64, y: i64, mv: i64) -> Unit {
    Unit { camp: OWN, x, y, mv, hp: 1, max_hp: 1, on_field: true, ..Default::default() }
}

/// Go map 查詢的零值：缺鍵讀成 false。
fn reached(reach: &BTreeMap<Cell, bool>, x: i64, y: i64) -> bool {
    reach.get(&Cell { x, y }).copied().unwrap_or(false)
}

/// Go：`TestReachable_BlockedByTerrain`
///
/// 一整排不可通行地形(cost=99)擋在中間,單位過不去,驗證「不可通行」不需要額外特判,
/// 靠 cost 夠大自然被 MV 篩掉(見 MoveCost 註解)。
#[test]
fn reachable_blocked_by_terrain() {
    // 5x3 地圖,x=2 整欄是牆(cost=99),把地圖左右隔開。
    let (w, h) = (5i64, 3i64);
    let mut cost = vec![1i64; (w * h) as usize];
    for y in 0..h {
        cost[(y * w + 2) as usize] = 99;
    }
    let st = State { w, h, cost: Some(cost), units: vec![walker(0, 1, 10)], ..Default::default() };
    let reach = st.reachable(0);
    for x in 3..w {
        for y in 0..h {
            assert!(!reached(&reach, x, y), "牆右側 ({x},{y}) 不該可達(MV=10 仍應被 cost=99 擋停)");
        }
    }
    assert!(reached(&reach, 1, 1), "牆左側 (1,1) 應可達");
}

/// Go：`TestReachable_HighCostTerrainLimitsRange`
///
/// 沼澤(cost=2)比平地(cost=1)更耗 MV,同樣 MV=3 的單位,走沼澤只能走 1 格,走平地能走 3 格——
/// 驗證 Reachable 真的按地形差異扣血(進入格的成本查 s.Cost,不是每步固定 1),不是所有地形都當一步處理。
#[test]
fn reachable_high_cost_terrain_limits_range() {
    let (w, h) = (5i64, 1i64);
    let st_swamp = State { w, h, cost: Some(vec![2; 5]), units: vec![walker(0, 0, 3)], ..Default::default() };
    let reach_swamp = st_swamp.reachable(0);
    assert!(reached(&reach_swamp, 1, 0), "沼澤:(1,0) 成本 2<=MV(3),應可達");
    assert!(!reached(&reach_swamp, 2, 0), "沼澤:(2,0) 累積成本 2+2=4>MV(3),不該可達");

    let st_flat = State { w, h, cost: Some(vec![1; 5]), units: vec![walker(0, 0, 3)], ..Default::default() };
    let reach_flat = st_flat.reachable(0);
    assert!(reached(&reach_flat, 3, 0), "平地:(3,0) 累積成本 1+1+1=3<=MV(3),應可達");
}

/// Go：`TestLoad_ReadsCostFromMapJSON`
///
/// units.json 同目錄放一份含 "cost" 的 map.json,Load 應自動接上;w/h 對不上或缺檔則 Cost 保持 nil(不 fail)。
///
/// Go 以 `json.Marshal` 寫出測試檔：`[]byte` 欄位是 base64 字串，`[]NativeFieldEvent` 用 JSON 標籤。
/// 這裡直接寫出同樣的 JSON。
#[test]
#[ignore = "等 F 合併"]
fn load_reads_cost_from_map_json() {
    let dir = temp_dir("move-load-cost");
    let units_path = dir.join("u.json");
    std::fs::write(&units_path, br#"{"h":2,"map":0,"own_deploy":[],"units":[],"w":2}"#).unwrap();
    let mut events: Vec<serde_json::Value> = vec![serde_json::json!({"event_id": 0, "selector": 0}); 16];
    events[3] = serde_json::json!({"event_id": 82, "selector": 1});
    let map = serde_json::json!({
        "w": 2, "h": 2, "tileW": 24, "tileH": 24, "cols": 16,
        "tiles": [0, 1, 0, 1], "cost": [1, 2, 99, 1],
        "native_composition_event_bytes": "AECAAA==",
        "native_field_event_slots": [-1, 3, -1, -1],
        "native_field_events": events,
        "native_tile_blit_modes": "AAAAAA==",
        "native_terrain_control": "AAAAAAAEAAA=",
    });
    std::fs::write(dir.join("map.json"), serde_json::to_vec(&map).unwrap()).unwrap();

    let st = load(&units_path.to_string_lossy()).unwrap_or_else(|err| panic!("Load: {err}"));
    let want = [1, 2, 99, 1];
    let cost = st.cost.clone().unwrap_or_default();
    assert_eq!(cost.len(), want.len(), "Cost len");
    for (i, w) in want.iter().enumerate() {
        assert_eq!(cost[i], *w, "Cost[{i}]");
    }
    assert_eq!(st.move_cost(0, 1), 99, "MoveCost(0,1)");
    assert_eq!(st.native_composition_event_bytes, [0, 0x40, 0x80, 0], "NativeCompositionEventBytes");
    let got = native_field_event_id_at(Some(&st), 1, 0, 1);
    assert!(got == (82, true), "NativeFieldEventIDAt={got:?}, want (82,true)");
    assert!(
        st.native_tile_blit_modes == [0xff, 0xff, 0xff, 0xff]
            && st.native_terrain_control == [0, 0, 0, 0, 0, 4, 0, 0]
            && st.native_terrain_move_codes == [0, 4, 0, 4],
        "Native terrain renderer inputs modes={:?} control={:?} move codes={:?}",
        st.native_tile_blit_modes,
        st.native_terrain_control,
        st.native_terrain_move_codes
    );
}

/// Go：`TestLoad_NoMapJSON_CostNil`
///
/// 同目錄沒有 map.json(舊資產或還沒重新匯出)時,Load 不應失敗,Cost 保持 nil,MoveCost 退回全平地。
/// Go 另查 `NativeCompositionEventBytes`／`NativeTerrainMoveCodes` 為 nil；Rust 的 `Vec` 不分 nil，改查空。
#[test]
fn load_no_map_json_cost_nil() {
    let dir = temp_dir("move-load-nomap");
    let units_path = dir.join("u.json");
    std::fs::write(&units_path, br#"{"h":3,"map":0,"own_deploy":[],"units":[],"w":3}"#).unwrap();
    let st = load(&units_path.to_string_lossy()).unwrap_or_else(|err| panic!("Load: {err}"));
    assert!(st.cost.is_none(), "Cost = {:?}, want nil(無 map.json)", st.cost);
    assert!(
        st.native_composition_event_bytes.is_empty(),
        "NativeCompositionEventBytes={:?} want nil",
        st.native_composition_event_bytes
    );
    assert!(
        st.native_terrain_move_codes.is_empty(),
        "NativeTerrainMoveCodes={:?} want nil",
        st.native_terrain_move_codes
    );
    assert_eq!(st.move_cost(1, 1), 1, "MoveCost");
}
