package battle

import "testing"

// 驗證序章 units.json 正確載入(M1-8 headless 回歸雛形)。
func TestLoadSerial0(t *testing.T) {
	st, err := Load("../../assets/map0_units.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if st.W != 24 || st.H != 24 {
		t.Errorf("size = %dx%d, want 24x24", st.W, st.H)
	}
	if len(st.Units) != 30 {
		t.Errorf("units = %d, want 30", len(st.Units))
	}
	own, ally, enemy := st.AliveCount(Own), st.AliveCount(Ally), st.AliveCount(Enemy)
	t.Logf("own=%d ally=%d enemy=%d deploy=%d turn=%d", own, ally, enemy, len(st.OwnDeploy), st.Turn)
	if own < 1 || enemy < 1 {
		t.Errorf("缺陣營:own=%d enemy=%d", own, enemy)
	}
	if st.Turn != 1 {
		t.Errorf("初始回合 = %d, want 1", st.Turn)
	}
	for _, u := range st.Units {
		if u.HP <= 0 || u.MaxHP <= 0 {
			t.Errorf("%s 單位 HP 異常:%d/%d", u.Camp, u.HP, u.MaxHP)
		}
		if u.MV <= 0 {
			t.Errorf("%s 單位移動力 = %d", u.Camp, u.MV)
		}
		// 註:own 不再自動塞部署格(部署格保留給 scenario spawn_party 主角隊,
		// 見 Load 內註解);units.json 的 own 沿用檔案座標,不驗部署格。
	}
	// UnitAt + Alive
	u0 := st.Units[0]
	if got := st.UnitAt(u0.X, u0.Y); got == nil {
		t.Errorf("UnitAt(%d,%d) = nil", u0.X, u0.Y)
	}
	u0.HP = 0
	if st.UnitAt(u0.X, u0.Y) == u0 {
		t.Error("陣亡單位不應被 UnitAt 回傳")
	}
}
