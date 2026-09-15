package battle

import (
	"math/rand"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func nativeExperiencePair() (*Unit, *Unit) {
	a, d := completeNativeAIScoringUnit(), completeNativeAIScoringUnit()
	a.Camp, a.NativeRecordByte6, a.NativeRecordClass = Own, 2, 2
	a.NativeRecordByte5, a.NativeRecordByte8 = 0, 0
	a.Lv, a.Exp, a.HIT, a.AP, a.CritPct = 4, 0, 100, 29, 0
	a.HP, a.MaxHP, a.X, a.Y = 48, 48, 0, 0
	d.Camp, d.NativeRecordByte6, d.NativeRecordByte5 = Enemy, 0, 0
	d.Lv, d.HP, d.MaxHP, d.DP, d.EV = 2, 28, 28, 4, 0
	d.X, d.Y, d.BattleFig = 1, 0, 96
	d.NativeConstructor = &NativeConstructorTable{Branch: "high_class", Index: 28, Record: []byte{1, 7, 14, 0, 0, 7, 1, 1, 4, 21}}
	// 刻意保留舊錯誤 ex 值，確保正式結算只消費原始建構列。
	d.ExpPerLevel = 110
	return a, d
}

func TestNativePhysicalExperienceTwoIntegerDivisions(t *testing.T) {
	a, d := nativeExperiencePair()
	p, err := planNativePhysicalExperience(a, d)
	if err != nil {
		t.Fatal(err)
	}
	if p.award(22, false) != 7 || p.award(6, true) != 10 {
		t.Fatal("未依原版分段取整或擊倒全額結算")
	}
	a.NativeRecordClass = 9
	p, err = planNativePhysicalExperience(a, d)
	if err != nil || p.base != 1 {
		t.Fatalf("轉職分母=%+v err=%v", p, err)
	}
	a.NativeRecordClass, a.NativeRecordByte8 = 2, 28
	p, err = planNativePhysicalExperience(a, d)
	if err != nil || p.base != 1 {
		t.Fatal("raw +8=28 的分母未加30")
	}
	a.NativeRecordByte8, a.Lv, d.Lv = 0, 1, 20
	p, err = planNativePhysicalExperience(a, d)
	if err != nil || p.award(28, true) != 420 {
		// 99 的上限在玩家路徑 0x11959，不在 sub_29F72 的一擊計算裡。
		t.Fatalf("一擊經驗未依 base 全額回傳：%+v err=%v", p, err)
	}
	if p.award(0, false) != 0 {
		t.Fatal("打空的一擊應該把 [0x53EC8] 寫成 0")
	}
}

// nativeCounterExperiencePair 讓敵方（`+6`＝0）打我方（`+6`＝2）：敵人打不死我方，
// 我方反擊一擊必殺。兩邊都帶原版建構列，所以反擊的經驗計畫和主攻一樣可算。
func nativeCounterExperiencePair(t *testing.T) (*State, *Unit, *Unit) {
	t.Helper()
	a, d := nativeExperiencePair()
	// 攻方換成敵人：+6＝0、命中極低所以永遠打空；守方是我方 lv4，反擊 AP 遠高於敵 DP。
	a.Camp, a.NativeRecordByte6, a.NativeRecordByte5 = Enemy, 0, 0
	a.Lv, a.HP, a.MaxHP, a.HIT, a.AP, a.DP, a.EV = 2, 28, 28, 0, 1, 0, 0
	a.BattleFig = 96
	a.NativeConstructor = &NativeConstructorTable{Branch: "high_class", Index: 28, Record: []byte{1, 7, 14, 0, 0, 7, 1, 1, 4, 21}}
	d.Camp, d.NativeRecordByte6, d.NativeRecordByte5, d.NativeRecordClass = Own, 2, 0, 2
	d.NativeRecordByte8 = 0
	d.Lv, d.Exp, d.HP, d.MaxHP, d.HIT, d.EV, d.AP, d.DP = 4, 95, 48, 48, 100, 100, 200, 4
	d.BattleFig = 9
	d.NativeTransient = [6]byte{} // +0x26 非零就不反擊
	d.Inventory, d.Equipped = []int{1}, []bool{true}
	d.InventorySlots, d.NativeInventoryFlags = []int{1}, []int{0x40}
	s := &State{W: 2, H: 1, Units: []*Unit{a, d}}
	if err := s.BindNativeFutureItemRows(make([]byte, 2*NativeItemEffectRowSize)); err != nil {
		t.Fatal(err)
	}
	s.nativeFutureItemRows[NativeItemEffectRowSize+11] = 1 // 武器射程 1 才能反擊
	// 對應 growth.json 的 [6,8)、[4,5)、[2,2)、[8,11)、[0,0)：DP 跨距 1 仍擲一次（恆 0），
	// DX 與 MP 跨距 0 不擲。
	s.NativeGrowthRows = map[int]GrowthRow{9: {
		AP: StatRange{Min: 6, Max: 7, Native: true}, DP: StatRange{Min: 4, Max: 4, Native: true},
		DX: StatRange{Min: 2, Max: 2, Native: true, Fixed: true}, HP: StatRange{Min: 8, Max: 10, Native: true},
		MP: StatRange{Min: 0, Max: 0, Native: true, Fixed: true},
	}}
	return s, a, d
}

// TestNativeCounterKillAwardsDefenderExperienceAndLevelsUpNatively 釘住 0x1566A：敵方
// 行動結束後對被打的我方單位呼叫 0x1E292，[0x53EC8] 是我方反擊最後一擊寫的值。
// 反擊擊倒 lv2 敵人：base = 21×2/4 = 10，全額；我方 95+10 → 升級，成長擲原版 RNG。
func TestNativeCounterKillAwardsDefenderExperienceAndLevelsUpNatively(t *testing.T) {
	s, a, d := nativeCounterExperiencePair(t)
	ap0, dp0, hp0, maxHP0 := d.AP, d.DP, d.HP, d.MaxHP
	r, err := s.AttackNativePhysicalWithExperience(a, d, 0x1234, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	if r.Counter == nil || a.HP != 0 {
		t.Fatalf("反擊未擊倒敵人：%+v a.HP=%d", r, a.HP)
	}
	if r.Attack.ExpGained != 0 || r.Attack.LevelUps != nil {
		t.Fatalf("敵方攻擊不該拿經驗：%+v", r.Attack)
	}
	if r.CounterExpGained != 10 || d.Exp != 5 || d.Lv != 5 || len(r.CounterLevelUps) != 1 {
		t.Fatalf("反擊經驗／升級：exp=%d d.Exp=%v d.Lv=%d ups=%+v", r.CounterExpGained, d.Exp, d.Lv, r.CounterLevelUps)
	}
	// 成長要接在反擊之後的 RNG 上：AP 6..7、DP 恆 4 但擲一次、DX 固定 2（不擲）、
	// HP 8..10、MP 固定 0（不擲）——第四章 r9 收據 0x1E54A 對 id 4 正好四筆。
	rngAfterStrikes := r.CounterStrikes[len(r.CounterStrikes)-1].Roll.RNGState
	roll := nativeStatRoll(&rngAfterStrikes)
	wantAP, wantDP, wantHP := 6+roll(2), 4+roll(1), 8+roll(3)
	if d.AP != ap0+wantAP || d.DP != dp0+wantDP || d.MaxHP != maxHP0+wantHP || d.HP != hp0 {
		t.Fatalf("成長未依原版 RNG：AP %d→%d DP %d→%d MaxHP %d→%d HP=%d want +%d/+%d/+%d",
			ap0, d.AP, dp0, d.DP, maxHP0, d.MaxHP, d.HP, wantAP, wantDP, wantHP)
	}
	if r.RNGState != rngAfterStrikes {
		t.Fatalf("結算後 RNG=%#x，應為成長擲骰之後的 %#x", r.RNGState, rngAfterStrikes)
	}
}

// TestNativeExchangeStopsWhenDefenderFalls 釘住 0x29B4C：守方 HP 歸零就不再揮第二擊，
// 也不再消耗 RNG。挑一個開頭 3% 會給兩擊的狀態，第一擊就打死。
func TestNativeExchangeStopsWhenDefenderFalls(t *testing.T) {
	s, a, d := nativeCounterExperiencePair(t)
	a.HP, a.MaxHP = 1, 1
	found := false
	for state := 0; state <= 0xFFFF && !found; state++ {
		if int(fdother.NativeRNGStep(uint16(state)))%100 >= 3 {
			continue
		}
		a.HP = 1
		strikes, _, err := s.nativePhysicalExchange(d, a, uint16(state))
		if err != nil {
			t.Fatal(err)
		}
		if len(strikes) != 1 || strikes[0].DefenderHP != 0 {
			t.Fatalf("state=%#x strikes=%+v：守方倒下後仍揮了第二擊", state, strikes)
		}
		found = true
	}
	if !found {
		t.Fatal("找不到二連擊的起始狀態")
	}
}

func TestNativePhysicalAttackThenMovementRecordsRemainSerializable(t *testing.T) {
	a, d := nativeExperiencePair()
	s := &State{W: 2, H: 1, Units: []*Unit{a, d}}
	r, err := s.AttackWithNativeExperience(a, d, rand.New(rand.NewSource(3)))
	if err != nil {
		t.Fatal(err)
	}
	want := r.Amount * 10 / 28
	if r.Missed || r.ExpGained != float64(want) || a.Exp != float64(want) || !a.Acted || a.NativeRecordByte5&0x80 == 0 {
		t.Fatalf("攻擊／EXP／行動=%+v actor=%+v", r, a)
	}
	records, err := NativeAIScoringRecords(s.Units)
	if err != nil {
		t.Fatalf("攻擊後下一個移動的記錄建構失敗：%v", err)
	}
	if records[0x3c] != byte(want) || records[5]&0x80 == 0 {
		t.Fatal("原生記錄未同步 EXP／行動")
	}
	if _, err := NativeItemPanelRecordForUnit(a); err != nil {
		t.Fatal(err)
	}
}

func TestNativePhysicalExperienceMissingSourceRejectsBeforeMutation(t *testing.T) {
	a, d := nativeExperiencePair()
	d.NativeConstructor = nil
	s := &State{W: 2, H: 1, Units: []*Unit{a, d}}
	rng, control := rand.New(rand.NewSource(1)), rand.New(rand.NewSource(1))
	if _, err := s.AttackWithNativeExperience(a, d, rng); err == nil {
		t.Fatal("缺建構列未拒絕")
	}
	if a.Acted || a.Exp != 0 || a.NativeRecordByte5 != 0 || d.HP != 28 || rng.Int63() != control.Int63() {
		t.Fatal("預檢失敗仍變更交易或RNG")
	}
}

func TestMap0PhysicalExperienceUsesRawConstructorInsteadOfLegacyEX(t *testing.T) {
	s, err := Load("../../assets/maps/map0/map0_units.json")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := nativeExperiencePair()
	p, err := planNativePhysicalExperience(a, s.Units[0])
	if err != nil || p.base != 10 {
		t.Fatalf("map0原始經驗基數=%+v err=%v", p, err)
	}
}
