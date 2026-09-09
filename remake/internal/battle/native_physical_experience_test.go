package battle

import (
	"math/rand"
	"testing"
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
	if err != nil || p.award(28, true) != 99 {
		t.Fatal("未限制單次玩家經驗99")
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
