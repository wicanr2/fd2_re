package battle

import (
	"math/rand"
	"testing"
)

func newTestState() *State { return &State{W: 20, H: 20} }

func mkUnit(camp Camp, x, y, hp, mp int) *Unit {
	return &Unit{Camp: camp, HP: hp, MaxHP: hp, MP: mp, OnField: true, X: x, Y: y}
}

// TestCastArea_AoE_HitsAllInRange:波及範圍(曼哈頓距離 ≤ Range)內的合法目標(同陣營篩選)
// 全部各中一次,範圍外、陣營不合的單位不受影響。
func TestCastArea_AoE_HitsAllInRange(t *testing.T) {
	st := newTestState()
	caster := mkUnit(Own, 5, 5, 100, 50)
	e1 := mkUnit(Enemy, 5, 5, 100, 0) // 中心,距離0
	e2 := mkUnit(Enemy, 5, 6, 100, 0) // 距離1(範圍內)
	e3 := mkUnit(Enemy, 4, 5, 100, 0) // 距離1(範圍內)
	e4 := mkUnit(Enemy, 5, 8, 100, 0) // 距離3(範圍外)
	friend := mkUnit(Own, 6, 5, 100, 0) // 距離1,但同陣營 → 攻擊法術不應命中
	st.Units = []*Unit{caster, e1, e2, e3, e4, friend}

	sp := Spell{ID: 999, Dmg: 100, Hit: 100, Dist: 5, Range: 1, MP: 20, Target: 0}
	results := st.CastArea(caster, 5, 5, sp, rand.New(rand.NewSource(1)))

	if len(results) != 3 {
		t.Fatalf("AoE 命中數 = %d, want 3(中心 + 2 個範圍內敵人)", len(results))
	}
	for _, r := range results {
		if r.Missed || r.Amount <= 0 {
			t.Errorf("目標 %p 應命中造成傷害,got Missed=%v Amount=%d", r.Target, r.Missed, r.Amount)
		}
	}
	if e4.HP != 100 {
		t.Errorf("範圍外的 e4 不應受影響,HP=%d", e4.HP)
	}
	if friend.HP != 100 {
		t.Errorf("同陣營的 friend 不應被攻擊法術命中,HP=%d", friend.HP)
	}
}

// TestCastArea_MPDeductedOnce:AoE 命中多個目標時 MP 只扣一次(非每個目標各扣一次)。
func TestCastArea_MPDeductedOnce(t *testing.T) {
	st := newTestState()
	caster := mkUnit(Own, 0, 0, 100, 20)
	e1 := mkUnit(Enemy, 0, 0, 50, 0)
	e2 := mkUnit(Enemy, 0, 1, 50, 0)
	st.Units = []*Unit{caster, e1, e2}

	sp := Spell{ID: 998, Dmg: 10, Hit: 100, Dist: 5, Range: 1, MP: 20, Target: 0}
	results := st.CastArea(caster, 0, 0, sp, rand.New(rand.NewSource(2)))

	if len(results) != 2 {
		t.Fatalf("命中數 = %d, want 2", len(results))
	}
	if caster.MP != 0 {
		t.Errorf("MP 應只扣一次(20),剩 %d", caster.MP)
	}
}

// TestCastArea_InsufficientMP:MP 不足時不扣血、不套效果,回 nil。
func TestCastArea_InsufficientMP(t *testing.T) {
	st := newTestState()
	caster := mkUnit(Own, 0, 0, 100, 5)
	st.Units = []*Unit{caster}

	sp := Spell{ID: 997, Dmg: 10, Hit: 100, Dist: 5, Range: 0, MP: 20, Target: 0}
	if got := st.CastArea(caster, 0, 0, sp, rand.New(rand.NewSource(3))); got != nil {
		t.Errorf("MP 不足應回 nil,got %+v", got)
	}
	if caster.MP != 5 {
		t.Errorf("MP 不足不應扣血,got %d", caster.MP)
	}
}

// TestCastArea_SealedCannotCast:被封咒術禁止施法的單位無法再施法(doc02 §6.4)。
func TestCastArea_SealedCannotCast(t *testing.T) {
	st := newTestState()
	caster := mkUnit(Own, 0, 0, 100, 50)
	caster.Sealed = true
	st.Units = []*Unit{caster}

	sp := Spell{ID: 996, Dmg: 10, Hit: 100, Dist: 5, Range: 0, MP: 5, Target: 0}
	if got := st.CastArea(caster, 0, 0, sp, rand.New(rand.NewSource(4))); got != nil {
		t.Errorf("被封咒應無法施法,got %+v", got)
	}
	if caster.MP != 50 {
		t.Errorf("施法失敗不應扣 MP,got %d", caster.MP)
	}
}

// TestApplySpell_HitRollBothOutcomes:毒擊術(id26,hit=50)固定 seed 多次擲骰應同時出現命中與 Miss;
// 命中時才施加中毒狀態,傷害套用 doc02 §4.3 隨機公式(dmg=10 恆得 9,對應攻略「附加 9 點傷害」)。
func TestApplySpell_HitRollBothOutcomes(t *testing.T) {
	sp := Spell{ID: 26, Dmg: 10, Hit: 50, Dist: 4, Range: 0, MP: 8, Target: 0}
	rng := rand.New(rand.NewSource(42))
	sawHit, sawMiss := false, false
	for i := 0; i < 40; i++ {
		st := newTestState()
		caster := mkUnit(Own, 0, 0, 100, 100)
		tgt := mkUnit(Enemy, 0, 0, 100, 0)
		st.Units = []*Unit{caster, tgt}

		results := st.CastArea(caster, 0, 0, sp, rng)
		if len(results) != 1 {
			t.Fatalf("第 %d 次:預期 1 筆結果,got %d", i, len(results))
		}
		r := results[0]
		if r.Missed {
			sawMiss = true
			if tgt.Poisoned {
				t.Error("Miss 不應施加中毒")
			}
		} else {
			sawHit = true
			if !tgt.Poisoned || tgt.PoisonTurns < 2 || tgt.PoisonTurns > 4 {
				t.Errorf("命中應施加中毒且回合數在 2–4,got Poisoned=%v Turns=%d", tgt.Poisoned, tgt.PoisonTurns)
			}
			if r.Amount != 9 {
				t.Errorf("毒擊 dmg=10 經隨機公式應恆得 9,got %d", r.Amount)
			}
		}
	}
	if !sawHit || !sawMiss {
		t.Errorf("40 次擲骰(hit=50%%)應同時出現命中與 Miss,sawHit=%v sawMiss=%v", sawHit, sawMiss)
	}
}

// TestApplySpell_BuffAP:魔刃術(id17)AP +15%,持續 2–4 回合。
func TestApplySpell_BuffAP(t *testing.T) {
	st := newTestState()
	caster := mkUnit(Own, 5, 5, 100, 100) // 離中心(0,0)有距離,避免施法者自己也落在 Range=0 內
	tgt := mkUnit(Own, 0, 0, 100, 0)
	tgt.AP = 100
	st.Units = []*Unit{caster, tgt}

	sp := Spell{ID: 17, Dmg: 0, Hit: 0, Dist: 4, Range: 0, MP: 5, Target: 1}
	results := st.CastArea(caster, 0, 0, sp, rand.New(rand.NewSource(9)))
	if len(results) != 1 || results[0].Missed {
		t.Fatalf("魔刃術應命中我方目標,got %+v", results)
	}
	if tgt.BuffAPPct != 15 {
		t.Errorf("BuffAPPct = %d, want 15", tgt.BuffAPPct)
	}
	if tgt.BuffTurns < 2 || tgt.BuffTurns > 4 {
		t.Errorf("BuffTurns = %d, want 2-4", tgt.BuffTurns)
	}
	if got := tgt.EffectiveAP(); got != 115 {
		t.Errorf("EffectiveAP = %d, want 115", got)
	}
}

// TestApplySpell_BuffWindWalk:風行術(id19)HIT+15、EV+15(doc02 §6.4;非先前規格草案猜測的 MV+2)。
func TestApplySpell_BuffWindWalk(t *testing.T) {
	st := newTestState()
	caster := mkUnit(Own, 0, 0, 100, 100)
	tgt := mkUnit(Own, 0, 0, 100, 0)
	st.Units = []*Unit{caster, tgt}

	sp := Spell{ID: 19, Dmg: 0, Hit: 0, Dist: 4, Range: 0, MP: 8, Target: 1}
	st.CastArea(caster, 0, 0, sp, rand.New(rand.NewSource(10)))
	if tgt.BuffHit != 15 || tgt.BuffEV != 15 {
		t.Errorf("風行術應 HIT+15/EV+15,got BuffHit=%d BuffEV=%d", tgt.BuffHit, tgt.BuffEV)
	}
	if tgt.BuffDPPct != 0 || tgt.BuffAPPct != 0 {
		t.Errorf("風行術不應動到 AP/DP,got AP%%=%d DP%%=%d", tgt.BuffAPPct, tgt.BuffDPPct)
	}
}

// TestApplySpell_ComboSpells:破壞神(魔刃+魔鎧+風行)、暗邪鬼(麻痺+封咒+毒擊)組合技(doc02 §6.4)。
func TestApplySpell_ComboSpells(t *testing.T) {
	st := newTestState()
	caster := mkUnit(Own, 0, 0, 100, 200)
	ally := mkUnit(Own, 0, 0, 100, 0)
	st.Units = []*Unit{caster, ally}

	spBreak := Spell{ID: 34, Dmg: 0, Hit: 0, Dist: 5, Range: 0, MP: 28, Target: 1}
	st.CastArea(caster, 0, 0, spBreak, rand.New(rand.NewSource(11)))
	if ally.BuffAPPct != 15 || ally.BuffDPPct != 15 || ally.BuffHit != 15 || ally.BuffEV != 15 {
		t.Errorf("破壞神應同時套用魔刃+魔鎧+風行,got AP%%=%d DP%%=%d Hit=%d EV=%d",
			ally.BuffAPPct, ally.BuffDPPct, ally.BuffHit, ally.BuffEV)
	}

	st2 := newTestState()
	caster2 := mkUnit(Own, 0, 0, 100, 200)
	enemy := mkUnit(Enemy, 0, 0, 100, 0)
	st2.Units = []*Unit{caster2, enemy}

	spDark := Spell{ID: 35, Dmg: 0, Hit: 0, Dist: 4, Range: 0, MP: 36, Target: 0}
	st2.CastArea(caster2, 0, 0, spDark, rand.New(rand.NewSource(12)))
	if !enemy.Paralyzed || !enemy.Sealed || !enemy.Poisoned {
		t.Errorf("暗邪鬼應同時施麻痺+封咒+毒擊,got Paralyzed=%v Sealed=%v Poisoned=%v",
			enemy.Paralyzed, enemy.Sealed, enemy.Poisoned)
	}
}

// TestApplySpell_ActionSpellResetsActed:行動術(id25)讓已行動單位本回合可再次行動。
func TestApplySpell_ActionSpellResetsActed(t *testing.T) {
	st := newTestState()
	caster := mkUnit(Own, 0, 0, 100, 100)
	tgt := mkUnit(Own, 0, 0, 100, 0)
	tgt.Acted = true
	st.Units = []*Unit{caster, tgt}

	sp := Spell{ID: 25, Dmg: 0, Hit: 0, Dist: 3, Range: 1, MP: 24, Target: 1}
	st.CastArea(caster, 0, 0, sp, rand.New(rand.NewSource(14)))
	if tgt.Acted {
		t.Error("行動術應重置 Acted=false")
	}
}

// TestCureAndTickStatus:解毒/祛麻術清除對應異常;TickStatus 每回合遞減、到期自動解除、
// 中毒每回合扣 MaxHP 的 10%。
func TestCureAndTickStatus(t *testing.T) {
	st := newTestState()
	caster := mkUnit(Own, 0, 0, 100, 100)
	tgt := mkUnit(Own, 0, 0, 100, 0)
	tgt.Poisoned, tgt.PoisonTurns = true, 3
	tgt.Paralyzed, tgt.ParalyzeTurns = true, 3
	st.Units = []*Unit{caster, tgt}

	spCure := Spell{ID: 20, Dmg: 0, Hit: 0, Dist: 4, Range: 0, MP: 5, Target: 1}
	st.CastArea(caster, 0, 0, spCure, rand.New(rand.NewSource(13)))
	if tgt.Poisoned || tgt.PoisonTurns != 0 {
		t.Errorf("解毒術後應清除中毒,got Poisoned=%v Turns=%d", tgt.Poisoned, tgt.PoisonTurns)
	}
	if !tgt.Paralyzed {
		t.Error("解毒術不應影響麻痺狀態")
	}

	poisoned := mkUnit(Enemy, 1, 1, 100, 0)
	poisoned.Poisoned, poisoned.PoisonTurns = true, 1
	poisoned.TickStatus()
	if poisoned.HP != 90 {
		t.Errorf("中毒應扣 MaxHP 的 10%%,HP=%d, want 90", poisoned.HP)
	}
	if poisoned.Poisoned {
		t.Error("PoisonTurns 歸零後應自動解除中毒狀態")
	}

	for i := 0; i < 3; i++ {
		tgt.TickStatus()
	}
	if tgt.Paralyzed {
		t.Error("ParalyzeTurns 遞減 3 次後應自動解除麻痺")
	}
}

// TestCast_BackwardCompat:舊版單體施法介面(引擎目前呼叫的簽名)維持可用。
func TestCast_BackwardCompat(t *testing.T) {
	st := newTestState()
	caster := mkUnit(Own, 0, 0, 100, 100)
	tgt := mkUnit(Enemy, 0, 0, 50, 0)
	st.Units = []*Unit{caster, tgt}

	sp := Spell{ID: 995, Dmg: 20, Hit: 100, Dist: 5, Range: 0, MP: 10, Target: 0}
	amt := st.Cast(caster, tgt, sp)
	if amt <= 0 {
		t.Errorf("Cast 舊介面應回傳正傷害,got %d", amt)
	}
	if caster.MP != 90 {
		t.Errorf("MP 應扣 10,剩 %d", caster.MP)
	}

	caster2 := mkUnit(Own, 0, 0, 100, 5)
	if got := st.Cast(caster2, tgt, sp); got != -1 {
		t.Errorf("MP 不足應回 -1,got %d", got)
	}
}

// TestLoadSpells_Integration:實際 spells.json(EXE dump)載入 + 施放,交叉驗證 doc02 §6.1 傷害區間。
func TestLoadSpells_Integration(t *testing.T) {
	spells, err := LoadSpells("../../assets/spells.json")
	if err != nil {
		t.Fatalf("LoadSpells: %v", err)
	}
	if len(spells) != 36 {
		t.Fatalf("法術數 = %d, want 36", len(spells))
	}

	var fireball Spell
	found := false
	for _, sp := range spells {
		if sp.ID == 0 {
			fireball, found = sp, true
		}
	}
	if !found || fireball.Name != "火炎" {
		t.Fatalf("id0 法術名 = %q, want 火炎", fireball.Name)
	}

	st := newTestState()
	caster := mkUnit(Own, 0, 0, 100, 100)
	tgt := mkUnit(Enemy, 0, 0, 100, 0)
	st.Units = []*Unit{caster, tgt}

	results := st.CastArea(caster, 0, 0, fireball, rand.New(rand.NewSource(20)))
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if !results[0].Missed && (results[0].Amount < 45 || results[0].Amount > 49) {
		t.Errorf("火炎術命中傷害 = %d, want 45–49(doc02 §6.1)", results[0].Amount)
	}
}
