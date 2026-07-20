package battle

import (
	"math/rand"
	"testing"
)

// mkFighter 建一個位置固定、攻防命中閃避暴擊皆可控的測試單位(供 AttackWithRNG 系列測試)。
func mkFighter(camp Camp, x, y, hp, ap, dp, hit, ev, crit int) *Unit {
	u := mkUnit(camp, x, y, hp, 0)
	u.AP, u.DP, u.HIT, u.EV, u.CritPct = ap, dp, hit, ev, crit
	return u
}

// TestAttackWithRNG_HitRateDistribution:命中率=(攻方HIT-守方EV)%(doc02 §4.1)。
// HIT90-EV40=50%,固定多個 seed 應同時出現命中與 Miss 兩種結果;Miss 必為 Amount=0
// 且不扣血,命中必為 Amount>0 且確實扣血。
func TestAttackWithRNG_HitRateDistribution(t *testing.T) {
	st := newTestState()
	sawHit, sawMiss := false, false
	for seed := int64(1); seed <= 50; seed++ {
		a := mkFighter(Own, 0, 0, 100, 50, 10, 90, 40, 0)
		d := mkFighter(Enemy, 1, 0, 100, 0, 10, 0, 0, 0)
		st.Units = []*Unit{a, d}
		r := st.AttackWithRNG(a, d, rand.New(rand.NewSource(seed)))
		if r.Missed {
			sawMiss = true
			if r.Amount != 0 {
				t.Errorf("seed %d: Miss 應 Amount=0,got %d", seed, r.Amount)
			}
			if d.HP != 100 {
				t.Errorf("seed %d: Miss 不應扣血,HP=%d", seed, d.HP)
			}
		} else {
			sawHit = true
			if r.Amount <= 0 {
				t.Errorf("seed %d: 命中應 Amount>0,got %d", seed, r.Amount)
			}
			if d.HP >= 100 {
				t.Errorf("seed %d: 命中應扣血,HP=%d", seed, d.HP)
			}
		}
		if !a.Acted {
			t.Errorf("seed %d: 攻方應標記已行動(miss 也算耗用行動)", seed)
		}
	}
	if !sawHit || !sawMiss {
		t.Errorf("50%% 命中率應同時出現命中與 Miss:sawHit=%v sawMiss=%v", sawHit, sawMiss)
	}
}

// TestAttackWithRNG_AlwaysMiss:HIT-EV<=0(命中率算出 <=0%)依公式視為必定 miss,
// 與 magic.go rollsHit 的「hit<=0 必中」特例不同語意(見 combat.go rollsHitPct 註解)。
func TestAttackWithRNG_AlwaysMiss(t *testing.T) {
	st := newTestState()
	for seed := int64(1); seed <= 10; seed++ {
		a := mkFighter(Own, 0, 0, 100, 50, 0, 10, 0, 0)   // 攻方 HIT=10
		d := mkFighter(Enemy, 1, 0, 100, 0, 10, 0, 50, 0) // 守方 EV=50 → 10-50=-40%
		st.Units = []*Unit{a, d}
		r := st.AttackWithRNG(a, d, rand.New(rand.NewSource(seed)))
		if !r.Missed || r.Amount != 0 {
			t.Errorf("seed %d: 命中率<=0 應必定 Miss,got Missed=%v Amount=%d", seed, r.Missed, r.Amount)
		}
	}
}

// TestAttackWithRNG_AlwaysHit:HIT-EV>=100% 視為必中,不擲骰也不受 seed 影響。
func TestAttackWithRNG_AlwaysHit(t *testing.T) {
	st := newTestState()
	for seed := int64(1); seed <= 10; seed++ {
		a := mkFighter(Own, 0, 0, 100, 50, 0, 100, 0, 0)
		d := mkFighter(Enemy, 1, 0, 100, 0, 10, 0, 0, 0)
		st.Units = []*Unit{a, d}
		r := st.AttackWithRNG(a, d, rand.New(rand.NewSource(seed)))
		if r.Missed {
			t.Errorf("seed %d: 命中率>=100%% 應必中,got Missed=true", seed)
		}
	}
}

// TestAttackWithRNG_Crit:暴擊率 100% 時必定觸發,DP 先減半再套地形%(notes.md 公式順序)。
// st.Cost 為 nil → TerrainAPDPPct 對任何座標回「正常地形」(AP+5%,DP-5%,與 MoveCost
// 對 nil 的預設一致,見 terrain.go 註解),下列數字已把這個預設地形加成算進去:
//
//	AP=100 → 有效 100(無 buff)→ 地形 ×105/100 = 105
//	DP=100 → 暴擊減半 50 → 地形 ×95/100 = 47(50*95/100=47.5,整數除法取 47)
//	max = 105-47 = 58;實際傷害 = 58*0.9(=52.2→52) ~ 58-1(=57)
func TestAttackWithRNG_Crit(t *testing.T) {
	st := newTestState()
	for seed := int64(1); seed <= 20; seed++ {
		a := mkFighter(Own, 0, 0, 100, 100, 0, 100, 0, 100) // HIT100 必中,Crit100% 必暴擊
		d := mkFighter(Enemy, 1, 0, 200, 0, 100, 0, 0, 0)
		st.Units = []*Unit{a, d}
		r := st.AttackWithRNG(a, d, rand.New(rand.NewSource(seed)))
		if !r.Crit {
			t.Fatalf("seed %d: CritPct=100 應必定暴擊", seed)
		}
		if r.Amount < 52 || r.Amount > 57 {
			t.Errorf("seed %d: 暴擊傷害 = %d, want [52,57](見函式註解手算)", seed, r.Amount)
		}
	}
}

// TestAttackWithRNG_TerrainLowersAttackerDamage:doc02 §3.2 沼澤 AP-5%(相對正常地形 AP+5%),
// 攻方站在沼澤應比站在正常地形造成更低傷害(同一組固定 seed、相同攻防雙方數值,只變攻方座標
// 對應的地形)。用 CritPct=0、HIT=100 排除暴擊/命中率的隨機干擾,只看地形這一個變數。
// 註:defender 站沼澤 vs 正常地形不會有差異——两類地形的 DP% 修正皆為 -5%(見 terrain.go),
// 真正因地形讓「防禦上升」的森林(DP+10%)目前無法從 map.json 的 cost[] 精確判定(terrain.go
// 檔頭已記錄此資料管線缺口),故用「攻方站沼澤 → AP 修正變差 → 傷害下降」示範地形已接上,
// 而非 doc42 範例提到的「盜賊站森林」那個尚未能精確重現的具體案例。
func TestAttackWithRNG_TerrainLowersAttackerDamage(t *testing.T) {
	for seed := int64(1); seed <= 10; seed++ {
		stNormal := &State{W: 5, H: 5, Cost: []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}}
		stSwamp := &State{W: 5, H: 5, Cost: []int{2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}}

		aN := mkFighter(Own, 0, 0, 100, 100, 0, 100, 0, 0)
		dN := mkFighter(Enemy, 1, 0, 300, 0, 20, 0, 0, 0)
		stNormal.Units = []*Unit{aN, dN}
		rN := stNormal.AttackWithRNG(aN, dN, rand.New(rand.NewSource(seed)))

		aS := mkFighter(Own, 0, 0, 100, 100, 0, 100, 0, 0) // 站 (0,0) = cost 2 = 沼澤
		dS := mkFighter(Enemy, 1, 0, 300, 0, 20, 0, 0, 0)
		stSwamp.Units = []*Unit{aS, dS}
		rS := stSwamp.AttackWithRNG(aS, dS, rand.New(rand.NewSource(seed)))

		if rS.Amount >= rN.Amount {
			t.Errorf("seed %d: 攻方站沼澤傷害(%d)應低於站正常地形(%d)", seed, rS.Amount, rN.Amount)
		}
	}
}

// TestAttackWithRNG_DamageRandomizationRange:實際傷害 = 最大傷害*0.9 ~ 最大傷害-1(doc02 §4.1),
// AP=100/DP=20,套預設(nil Cost=正常地形 AP+5%/DP-5%)後 max=86,期望區間 [77,85]
// (86*9/10=77.4→77,86-1=85);多 seed 跑,驗證都落在區間內、且低高兩端都會出現(非退化成固定值)。
func TestAttackWithRNG_DamageRandomizationRange(t *testing.T) {
	st := newTestState()
	seenLow, seenHigh := false, false
	for seed := int64(1); seed <= 100; seed++ {
		a := mkFighter(Own, 0, 0, 100, 100, 0, 100, 0, 0)
		d := mkFighter(Enemy, 1, 0, 300, 0, 20, 0, 0, 0)
		st.Units = []*Unit{a, d}
		r := st.AttackWithRNG(a, d, rand.New(rand.NewSource(seed)))
		if r.Amount < 77 || r.Amount > 85 {
			t.Errorf("seed %d: 傷害 = %d, want [77,85]", seed, r.Amount)
		}
		if r.Amount <= 79 {
			seenLow = true
		}
		if r.Amount >= 83 {
			seenHigh = true
		}
	}
	if !seenLow || !seenHigh {
		t.Errorf("100 個 seed 的傷害應涵蓋區間低端與高端,seenLow=%v seenHigh=%v", seenLow, seenHigh)
	}
}

// TestAttackWithRNG_MinDamageFloor:即使公式算出 max<=0(守方 DP 過高),命中的攻擊仍至少造成 1
// (青衫「dmg<=2」是 AI 選標門檻,非玩家攻擊下限,見 combat.go 註解)。
func TestAttackWithRNG_MinDamageFloor(t *testing.T) {
	st := newTestState()
	a := mkFighter(Own, 0, 0, 100, 5, 0, 100, 0, 0)
	d := mkFighter(Enemy, 1, 0, 100, 0, 999, 0, 0, 0)
	st.Units = []*Unit{a, d}
	r := st.AttackWithRNG(a, d, rand.New(rand.NewSource(1)))
	if r.Missed {
		t.Fatal("HIT=100 應必中")
	}
	if r.Amount != 1 {
		t.Errorf("max<=0 時命中應保底 1,got %d", r.Amount)
	}
}

// TestAttack_BackwardCompat:main.go 呼叫的舊介面 Attack(a,d) 回傳與 AttackWithRNG 一致的
// Amount 語意(用必中/無暴擊/無地形干擾的簡單案例驗證扣血與回傳值一致)。
func TestAttack_BackwardCompat(t *testing.T) {
	st := newTestState()
	a := mkFighter(Own, 0, 0, 100, 50, 10, 100, 0, 0)
	d := mkFighter(Enemy, 1, 0, 200, 0, 10, 0, 0, 0)
	st.Units = []*Unit{a, d}
	before := d.HP
	amt := st.Attack(a, d)
	if amt <= 0 {
		t.Fatalf("Attack 應回傳正傷害,got %d", amt)
	}
	if before-d.HP != amt {
		t.Errorf("扣血量(%d)應等於回傳傷害(%d)", before-d.HP, amt)
	}
}

// TestEstDamage_UsesTerrain:AI 評分公式(doc11)套地形% 後,攻方站沼澤估算傷害應低於站正常地形
// (呼應 doc42 gap-audit 第 5 項「AI 評分公式未套地形%」)。
func TestEstDamage_UsesTerrain(t *testing.T) {
	stNormal := &State{W: 3, H: 1, Cost: []int{1, 1, 1}}
	stSwamp := &State{W: 3, H: 1, Cost: []int{2, 1, 1}}
	u := &Unit{X: 0, Y: 0, AP: 100}
	t2 := &Unit{X: 1, Y: 0, DP: 20}

	dmgNormal := stNormal.estDamage(u, t2)
	dmgSwamp := stSwamp.estDamage(u, t2)
	if dmgSwamp >= dmgNormal {
		t.Errorf("攻方站沼澤的 AI 估計傷害(%d)應低於站正常地形(%d)", dmgSwamp, dmgNormal)
	}
}

func TestNextAIPlan_SkipsLowDamageTarget(t *testing.T) {
	st := &State{W: 5, H: 1}
	ai := mkFighter(Enemy, 0, 0, 100, 5, 0, 100, 0, 0)
	target := mkFighter(Own, 1, 0, 100, 0, 20, 0, 0, 0)
	ai.MV = 3
	ai.OnField, target.OnField = true, true
	st.Units = []*Unit{ai, target}

	plan := st.NextAIPlan()
	if plan == nil || plan.U != ai {
		t.Fatalf("expected deterministic plan for AI unit, got %#v", plan)
	}
	if plan.Target != nil {
		t.Fatalf("estimated damage <=2 must skip attack target, got %q", plan.Target.Name)
	}
	if got := st.estDamage(ai, target); got > 2 {
		t.Fatalf("fixture damage must be at most two, got %d", got)
	}
}

func TestNextAIPlan_PrefersDamageAboveThreshold(t *testing.T) {
	st := &State{W: 6, H: 2}
	ai := mkFighter(Enemy, 0, 0, 100, 50, 0, 100, 0, 0)
	weak := mkFighter(Own, 0, 1, 100, 0, 100, 0, 0, 0) // low damage, nearer
	good := mkFighter(Own, 4, 0, 100, 0, 10, 0, 0, 0)  // viable damage
	ai.MV = 3
	ai.OnField, weak.OnField, good.OnField = true, true, true
	st.Units = []*Unit{ai, weak, good}

	plan := st.NextAIPlan()
	if plan == nil || plan.Target != good {
		t.Fatalf("AI should choose viable target over dmg<=2 target, got %#v", plan)
	}
}
