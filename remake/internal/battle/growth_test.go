package battle

import (
	"math/rand"
	"testing"
)

// ---- 升級成長(growthTable × GainExp)----

// TestGainExp_CrossesThreshold_MultiLevel:一次給的經驗值跨兩個 100 門檻,應連續升 2 級,
// 每級成長量落在 growthTable["索爾"]["劍士"] 的範圍內(doc02 §7.2:AP 6-7、DP 4-5、
// DX 2-2、HP 8-11、MP 0-0),且屬性確實被加進 Unit。
func TestGainExp_CrossesThreshold_MultiLevel(t *testing.T) {
	u := &Unit{Camp: Own, Name: "索爾", ClsName: "劍士", Lv: 1, AP: 6, DP: 4, DX: 2, HP: 42, MaxHP: 42, MP: 0}
	rng := rand.New(rand.NewSource(1))

	events := GainExp(u, 250, rng) // 250 = 2 級整 + 50 剩餘

	if len(events) != 2 {
		t.Fatalf("升級事件數 = %d, want 2", len(events))
	}
	if u.Lv != 3 {
		t.Errorf("Lv = %d, want 3", u.Lv)
	}
	if u.Exp != 50 {
		t.Errorf("剩餘經驗 = %v, want 50", u.Exp)
	}
	wantAPGain, wantDPGain, wantHPGain := 0, 0, 0
	for i, ev := range events {
		if ev.ApGain < 6 || ev.ApGain > 7 {
			t.Errorf("event[%d].ApGain = %d, 應落在 [6,7](doc02 §7.2 索爾/劍士)", i, ev.ApGain)
		}
		if ev.DpGain < 4 || ev.DpGain > 5 {
			t.Errorf("event[%d].DpGain = %d, 應落在 [4,5]", i, ev.DpGain)
		}
		if ev.DxGain != 2 {
			t.Errorf("event[%d].DxGain = %d, want 2(固定值)", i, ev.DxGain)
		}
		if ev.HpGain < 8 || ev.HpGain > 11 {
			t.Errorf("event[%d].HpGain = %d, 應落在 [8,11]", i, ev.HpGain)
		}
		if ev.MpGain != 0 {
			t.Errorf("event[%d].MpGain = %d, want 0(索爾劍士無 MP 成長)", i, ev.MpGain)
		}
		wantAPGain += ev.ApGain
		wantDPGain += ev.DpGain
		wantHPGain += ev.HpGain
	}
	if u.AP != 6+wantAPGain {
		t.Errorf("AP = %d, want %d", u.AP, 6+wantAPGain)
	}
	if u.DP != 4+wantDPGain {
		t.Errorf("DP = %d, want %d", u.DP, 4+wantDPGain)
	}
	if u.MaxHP != 42+wantHPGain || u.HP != 42+wantHPGain {
		t.Errorf("HP/MaxHP = %d/%d, want %d(升級當下回滿新增HP)", u.HP, u.MaxHP, 42+wantHPGain)
	}
}

// The EXE derives both HIT and EV from the same DX accumulator, then adds
// equipped item contributions.  A level-up must therefore move both derived
// values by the exact DX growth amount, without losing the equipment delta.
func TestGainExp_DXUpdatesDerivedHitAndEV(t *testing.T) {
	u := &Unit{Camp: Own, Name: "索爾", ClsName: "劍士", Lv: 1,
		AP: 16, DP: 12, DX: 2, HIT: 97, EV: 2,
		BaseHIT: 2, BaseEV: 2, EquipmentBaseSet: true}
	GainExp(u, 100, rand.New(rand.NewSource(1)))
	if u.DX != 4 {
		t.Fatalf("DX = %d, want 4", u.DX)
	}
	if u.HIT != 99 || u.EV != 4 || u.BaseHIT != 4 || u.BaseEV != 4 {
		t.Fatalf("DX synthesis lost equipment/base: HIT=%d EV=%d base=%d/%d", u.HIT, u.EV, u.BaseHIT, u.BaseEV)
	}
}

// TestGainExp_EnemyNoop:Enemy 不會累積經驗值/升級(doc42 gap 範圍只涵蓋玩家停滯問題,
// 敵方無成長曲線資料,見 growth.go 檔頭說明)。
func TestGainExp_EnemyNoop(t *testing.T) {
	u := &Unit{Camp: Enemy, Name: "", ClsName: "戰士", Lv: 5, HP: 50, MaxHP: 50}
	rng := rand.New(rand.NewSource(1))
	events := GainExp(u, 999, rng)
	if events != nil {
		t.Errorf("Enemy 不應升級,got %d 筆事件", len(events))
	}
	if u.Lv != 5 || u.Exp != 0 {
		t.Errorf("Enemy 狀態不應被改動,Lv=%d Exp=%v", u.Lv, u.Exp)
	}
}

// TestGainExp_UnknownGrowth_StillLevelsNoStatGain:growthTable 查無資料(如友軍雜兵無名單位)
// 時等級仍照門檻演進(避免經驗卡死),但不套用屬性成長(誠實反映缺資料,不是靜默丟棄)。
func TestGainExp_UnknownGrowth_StillLevelsNoStatGain(t *testing.T) {
	u := &Unit{Camp: Ally, Name: "無名雜兵", ClsName: "士兵", Lv: 1, AP: 10, DP: 10, HP: 50, MaxHP: 50}
	rng := rand.New(rand.NewSource(1))
	events := GainExp(u, 100, rng)
	if len(events) != 0 {
		t.Errorf("查無成長資料不應有 LevelUpEvent,got %d", len(events))
	}
	if u.Lv != 2 {
		t.Errorf("Lv 仍應演進為 2,got %d", u.Lv)
	}
	if u.AP != 10 || u.DP != 10 || u.MaxHP != 50 {
		t.Errorf("查無成長資料不應改動屬性,got AP=%d DP=%d MaxHP=%d", u.AP, u.DP, u.MaxHP)
	}
}

// TestGainExp_MultiClassPromotion:growthTable 同一角色多職業列(轉職系統尚未實作,但資料
// 已備妥)——確認索爾「英雄」職業能查到獨立於「劍士」的成長列。
func TestGainExp_MultiClassPromotion(t *testing.T) {
	u := &Unit{Camp: Own, Name: "索爾", ClsName: "英雄", Lv: 20, AP: 100, HP: 200, MaxHP: 200}
	rng := rand.New(rand.NewSource(7))
	events := GainExp(u, 100, rng)
	if len(events) != 1 {
		t.Fatalf("升級事件數 = %d, want 1", len(events))
	}
	ev := events[0]
	if ev.ApGain < 10 || ev.ApGain > 14 { // doc02 §7.2 索爾/英雄 AP 10-14
		t.Errorf("英雄職業 ApGain = %d, 應落在 [10,14]", ev.ApGain)
	}
}

// ---- 經驗值公式(doc02 §4.5)手算案例 ----

// TestAttackExp_HandCalc:攻方Lv10 打 守方Lv20(每級經驗30),造成守方總HP的一半傷害。
// 手算:ratio=0.5;(20×30)=600;(20/10)=2 → 0.5×600×2=600。
func TestAttackExp_HandCalc(t *testing.T) {
	got := AttackExp(10, 20, 50, 100, 30)
	want := 600.0
	if got != want {
		t.Errorf("AttackExp = %v, want %v", got, want)
	}
}

// TestAttackExp_LethalCapsAtTotalHP:doc02「致死視同傷害HP=總HP」——即使呼叫端傳入的
// dmgHP 超過總HP(overkill),也應等效於總HP,不會算出超過應有值的經驗。
func TestAttackExp_LethalCapsAtTotalHP(t *testing.T) {
	got := AttackExp(10, 20, 150, 100, 30) // dmgHP(150) > defTotalHP(100)
	want := AttackExp(10, 20, 100, 100, 30)
	if got != want {
		t.Errorf("overkill 應等同 dmgHP=defTotalHP:got %v, want %v", got, want)
	}
}

// TestAttackExp_MissingExpPerLevel_ZerosOut:defExpPerLv<=0(資料尚未接線,見 Unit.ExpPerLevel
// 註解)時誠實回 0,不假造經驗值。
func TestAttackExp_MissingExpPerLevel_ZerosOut(t *testing.T) {
	if got := AttackExp(10, 20, 50, 100, 0); got != 0 {
		t.Errorf("AttackExp with defExpPerLv=0 = %v, want 0", got)
	}
}

// TestHealExpTerm_HandCalc:恢復 20/40 HP、受法者 Lv15 → term = 0.5×15 = 7.5。
// 疊加另一位恢復 10/50、Lv5 → term = 0.2×5 = 1.0。Σ=8.5,套用「40/施法者Lv(10)」→ 34.0。
func TestHealExpTerm_HandCalc(t *testing.T) {
	sum := healExpTerm(20, 40, 15) + healExpTerm(10, 50, 5)
	if sum != 8.5 {
		t.Fatalf("Σterm = %v, want 8.5", sum)
	}
	total := 40 / float64(10) * sum
	if total != 34.0 {
		t.Errorf("恢復法術經驗值 = %v, want 34.0", total)
	}
}

// TestTeleportExp_HandCalc:傳送術 10×(受法者Lv/施法者Lv)。Lv20 施法者對 Lv10 受法者。
func TestTeleportExp_HandCalc(t *testing.T) {
	got := TeleportExp(20, 10)
	if got != 5.0 {
		t.Errorf("TeleportExp(20,10) = %v, want 5.0", got)
	}
}

// TestActionExp_HandCalc:行動術 8×(受法者Lv/施法者Lv)。同Lv(10,10)→ 8。
func TestActionExp_HandCalc(t *testing.T) {
	got := ActionExp(10, 10)
	if got != 8.0 {
		t.Errorf("ActionExp(10,10) = %v, want 8.0", got)
	}
}

// TestBuffExpTerm_HandCalc:魔刃/魔鎧/風行 2×Σ(受法者Lv/施法者Lv)。施法者Lv10,兩位受法者
// Lv20、Lv5 → Σ=2.0+0.5=2.5,×2=5.0。
func TestBuffExpTerm_HandCalc(t *testing.T) {
	sum := buffExpTerm(10, 20) + buffExpTerm(10, 5)
	if sum != 2.5 {
		t.Fatalf("Σterm = %v, want 2.5", sum)
	}
	if total := 2 * sum; total != 5.0 {
		t.Errorf("buff 經驗值 = %v, want 5.0", total)
	}
}

// TestStatusExpTerm_HandCalc:麻痺/毒擊(及解毒/祛麻共用同式)Σ(40×9/受法者總HP)×(受法者Lv/施法者Lv)。
// 受法者總HP=90 → 360/90=4;Lv20/施法者Lv10=2 → 4×2=8。
func TestStatusExpTerm_HandCalc(t *testing.T) {
	got := statusExpTerm(10, 20, 90)
	if got != 8.0 {
		t.Errorf("statusExpTerm(10,20,90) = %v, want 8.0", got)
	}
}

// ---- 端對端接線(combat.go / magic.go 呼叫路徑)----

// TestAttackWithRNG_GrantsExpAndLevelsUp:近戰攻擊命中後,Own 攻方應取得經驗值;經驗值
// 夠大時連續升級,AttackResult 帶回 LevelUps。
func TestAttackWithRNG_GrantsExpAndLevelsUp(t *testing.T) {
	st := newTestState()
	a := &Unit{Camp: Own, Name: "索爾", ClsName: "劍士", Lv: 1, AP: 400, DP: 4, HP: 42, MaxHP: 42,
		HIT: 100, EV: 5, OnField: true, X: 0, Y: 0}
	d := &Unit{Camp: Enemy, Lv: 30, DP: 0, HP: 500, MaxHP: 500, EV: 0, ExpPerLevel: 100,
		OnField: true, X: 0, Y: 1}
	st.Units = []*Unit{a, d}

	res := st.AttackWithRNG(a, d, rand.New(rand.NewSource(3)))
	if res.Missed {
		t.Fatalf("命中率 100%% 不應 Miss")
	}
	if res.ExpGained <= 0 {
		t.Fatalf("ExpGained = %v, 應 > 0(高等級敵方、每級經驗100)", res.ExpGained)
	}
	if len(res.LevelUps) == 0 {
		t.Errorf("經驗值夠大應觸發至少一次升級,LevelUps 為空;ExpGained=%v", res.ExpGained)
	}
	if a.Lv <= 1 {
		t.Errorf("攻方 Lv 應已提升,got %d", a.Lv)
	}
}

// TestAttackWithRNG_EnemyAttacker_NoExp:Enemy 當攻方時不應取得經驗值(doc42 gap 範圍僅玩家)。
func TestAttackWithRNG_EnemyAttacker_NoExp(t *testing.T) {
	st := newTestState()
	a := &Unit{Camp: Enemy, Lv: 1, AP: 400, HP: 50, MaxHP: 50, HIT: 100, OnField: true, X: 0, Y: 0}
	d := &Unit{Camp: Own, Lv: 30, DP: 0, HP: 500, MaxHP: 500, ExpPerLevel: 100, OnField: true, X: 0, Y: 1}
	st.Units = []*Unit{a, d}

	res := st.AttackWithRNG(a, d, rand.New(rand.NewSource(3)))
	if res.ExpGained != 0 || res.LevelUps != nil {
		t.Errorf("Enemy 攻方不應取得經驗值,got ExpGained=%v LevelUps=%v", res.ExpGained, res.LevelUps)
	}
}

// TestCastArea_HealGrantsExp:治療法術命中後,Own 施法者應依恢復法術公式取得經驗值,
// CastResult 帶回 ExpGained/LevelUps(彙總值,每筆結果相同)。
func TestCastArea_HealGrantsExp(t *testing.T) {
	st := newTestState()
	caster := &Unit{Camp: Own, Name: "瑪琳", ClsName: "僧侶", Lv: 1, MP: 50, HP: 30, MaxHP: 30, OnField: true, X: 0, Y: 0}
	tgt := &Unit{Camp: Own, Lv: 40, HP: 10, MaxHP: 200, OnField: true, X: 0, Y: 1}
	st.Units = []*Unit{caster, tgt}

	sp := Spell{ID: 13, Dmg: 70, Hit: 0, Dist: 3, Range: 0, MP: 5, Target: 1, Name: "治療"}
	results := st.CastArea(caster, 0, 1, sp, rand.New(rand.NewSource(9)))

	if len(results) != 1 {
		t.Fatalf("結果數 = %d, want 1", len(results))
	}
	if results[0].ExpGained <= 0 {
		t.Errorf("ExpGained = %v, 應 > 0(高Lv受法者、大量恢復)", results[0].ExpGained)
	}
	if caster.Exp <= 0 && len(results[0].LevelUps) == 0 {
		t.Errorf("施法者應累積到經驗值,caster.Exp=%v LevelUps=%v", caster.Exp, results[0].LevelUps)
	}
}

// TestCastArea_SealSpell_NoExp:封咒術(22)doc02 §4.5 未列經驗公式,施放不應給經驗
// (誠實的「文件未涵蓋」,不是遺漏)。
func TestCastArea_SealSpell_NoExp(t *testing.T) {
	st := newTestState()
	caster := &Unit{Camp: Own, Lv: 10, MP: 50, HP: 30, MaxHP: 30, OnField: true, X: 0, Y: 0}
	tgt := &Unit{Camp: Enemy, Lv: 10, HP: 100, MaxHP: 100, OnField: true, X: 0, Y: 1}
	st.Units = []*Unit{caster, tgt}

	sp := Spell{ID: 22, Dmg: 0, Hit: 0, Dist: 3, Range: 0, MP: 5, Target: 0, Name: "封咒"}
	results := st.CastArea(caster, 0, 1, sp, rand.New(rand.NewSource(1)))
	if len(results) != 1 {
		t.Fatalf("結果數 = %d, want 1", len(results))
	}
	if results[0].ExpGained != 0 {
		t.Errorf("封咒術不應給經驗值,got %v", results[0].ExpGained)
	}
}
