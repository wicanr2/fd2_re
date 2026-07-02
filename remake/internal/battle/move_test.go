package battle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestMoveCost_NoTerrainData_DefaultsFlat:State.Cost 為 nil(如舊測試直接手寫 State{}、
// 或 Load 找不到 map.json)時,MoveCost 全平地回 1,越界也回 1(不 panic)。
func TestMoveCost_NoTerrainData_DefaultsFlat(t *testing.T) {
	st := &State{W: 5, H: 5}
	if got := st.MoveCost(2, 2); got != 1 {
		t.Errorf("MoveCost = %d, want 1", got)
	}
	if got := st.MoveCost(-1, 0); got != 1 {
		t.Errorf("MoveCost(越界) = %d, want 1", got)
	}
	if got := st.MoveCost(99, 99); got != 1 {
		t.Errorf("MoveCost(越界) = %d, want 1", got)
	}
}

// TestMoveCost_FromTable:有 Cost 表時逐格查值(水=不可通行的 99、沼澤=2、平地=1)。
func TestMoveCost_FromTable(t *testing.T) {
	st := &State{W: 3, H: 1, Cost: []int{1, 99, 2}}
	cases := map[[2]int]int{{0, 0}: 1, {1, 0}: 99, {2, 0}: 2}
	for xy, want := range cases {
		if got := st.MoveCost(xy[0], xy[1]); got != want {
			t.Errorf("MoveCost(%d,%d) = %d, want %d", xy[0], xy[1], got, want)
		}
	}
}

// TestReachable_BlockedByTerrain:一整排不可通行地形(cost=99)擋在中間,單位過不去,
// 驗證「不可通行」不需要額外特判,靠 cost 夠大自然被 MV 篩掉(見 MoveCost 註解)。
func TestReachable_BlockedByTerrain(t *testing.T) {
	// 5x3 地圖,x=2 整欄是牆(cost=99),把地圖左右隔開。
	w, h := 5, 3
	cost := make([]int, w*h)
	for i := range cost {
		cost[i] = 1
	}
	for y := 0; y < h; y++ {
		cost[y*w+2] = 99
	}
	st := &State{W: w, H: h, Cost: cost}
	u := &Unit{Camp: Own, X: 0, Y: 1, MV: 10, HP: 1, MaxHP: 1, OnField: true}
	st.Units = []*Unit{u}
	reach := st.Reachable(u)
	for x := 3; x < w; x++ {
		for y := 0; y < h; y++ {
			if reach[Cell{x, y}] {
				t.Errorf("牆右側 (%d,%d) 不該可達(MV=10 仍應被 cost=99 擋停)", x, y)
			}
		}
	}
	if !reach[Cell{1, 1}] {
		t.Errorf("牆左側 (1,1) 應可達")
	}
}

// TestReachable_HighCostTerrainLimitsRange:沼澤(cost=2)比平地(cost=1)更耗 MV,
// 同樣 MV=3 的單位,走沼澤只能走 1 格,走平地能走 3 格——驗證 Reachable 真的按地形
// 差異扣血(進入格的成本查 s.Cost,不是每步固定 1),不是所有地形都當一步處理。
func TestReachable_HighCostTerrainLimitsRange(t *testing.T) {
	w, h := 5, 1
	swamp := make([]int, w*h)
	for x := 0; x < w; x++ {
		swamp[x] = 2 // 進入任何一格都是沼澤成本
	}
	flat := make([]int, w*h)
	for x := 0; x < w; x++ {
		flat[x] = 1
	}

	stSwamp := &State{W: w, H: h, Cost: swamp}
	uSwamp := &Unit{Camp: Own, X: 0, Y: 0, MV: 3, HP: 1, MaxHP: 1, OnField: true}
	stSwamp.Units = []*Unit{uSwamp}
	reachSwamp := stSwamp.Reachable(uSwamp)
	if !reachSwamp[Cell{1, 0}] {
		t.Errorf("沼澤:(1,0) 成本 2<=MV(3),應可達")
	}
	if reachSwamp[Cell{2, 0}] {
		t.Errorf("沼澤:(2,0) 累積成本 2+2=4>MV(3),不該可達")
	}

	stFlat := &State{W: w, H: h, Cost: flat}
	uFlat := &Unit{Camp: Own, X: 0, Y: 0, MV: 3, HP: 1, MaxHP: 1, OnField: true}
	stFlat.Units = []*Unit{uFlat}
	reachFlat := stFlat.Reachable(uFlat)
	if !reachFlat[Cell{3, 0}] {
		t.Errorf("平地:(3,0) 累積成本 1+1+1=3<=MV(3),應可達")
	}
}

// TestLoad_ReadsCostFromMapJSON:units.json 同目錄放一份含 "cost" 的 map.json,
// Load 應自動接上;w/h 對不上或缺檔則 Cost 保持 nil(不 fail)。
func TestLoad_ReadsCostFromMapJSON(t *testing.T) {
	dir := t.TempDir()

	unitsPath := filepath.Join(dir, "u.json")
	unitsRaw, _ := json.Marshal(map[string]any{
		"map": 0, "w": 2, "h": 2, "own_deploy": []any{},
		"units": []any{},
	})
	if err := os.WriteFile(unitsPath, unitsRaw, 0644); err != nil {
		t.Fatal(err)
	}
	mapRaw, _ := json.Marshal(map[string]any{
		"w": 2, "h": 2, "tileW": 24, "tileH": 24, "cols": 16,
		"tiles": []int{0, 0, 0, 0}, "cost": []int{1, 2, 99, 1},
	})
	if err := os.WriteFile(filepath.Join(dir, "map.json"), mapRaw, 0644); err != nil {
		t.Fatal(err)
	}

	st, err := Load(unitsPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []int{1, 2, 99, 1}
	if len(st.Cost) != len(want) {
		t.Fatalf("Cost len = %d, want %d", len(st.Cost), len(want))
	}
	for i, w := range want {
		if st.Cost[i] != w {
			t.Errorf("Cost[%d] = %d, want %d", i, st.Cost[i], w)
		}
	}
	if got := st.MoveCost(0, 1); got != 99 {
		t.Errorf("MoveCost(0,1) = %d, want 99", got)
	}
}

// TestLoad_NoMapJSON_CostNil:同目錄沒有 map.json(舊資產或還沒重新匯出)時,
// Load 不應失敗,Cost 保持 nil,MoveCost 退回全平地。
func TestLoad_NoMapJSON_CostNil(t *testing.T) {
	dir := t.TempDir()
	unitsPath := filepath.Join(dir, "u.json")
	unitsRaw, _ := json.Marshal(map[string]any{
		"map": 0, "w": 3, "h": 3, "own_deploy": []any{}, "units": []any{},
	})
	if err := os.WriteFile(unitsPath, unitsRaw, 0644); err != nil {
		t.Fatal(err)
	}
	st, err := Load(unitsPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if st.Cost != nil {
		t.Errorf("Cost = %v, want nil(無 map.json)", st.Cost)
	}
	if got := st.MoveCost(1, 1); got != 1 {
		t.Errorf("MoveCost = %d, want 1", got)
	}
}
