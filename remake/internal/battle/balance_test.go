package battle

import (
	"math/rand"
	"reflect"
	"testing"
)

func TestBalanceGrowthKeepsOriginalRollSequenceAndZeroGrowth(t *testing.T) {
	rows, err := LoadNativeGrowthSlots("../../../docs/data/exe_tables/native_growth_slots.json")
	if err != nil {
		t.Fatal(err)
	}
	base := Unit{Camp: Own, NativeIdentity: 4, HasNativeIdentity: true, BattleFig: 4, HasBattleFig: true, Lv: 2, HP: 10, MaxHP: 40, AP: 20, DP: 10, DX: 3}
	a, b, c := base, base, base
	st := &State{NativeGrowthRows: rows}
	_, old, seedA := st.AwardExpNative(&a, 100, 7)
	st.Balance = NewBalanceOptions()
	_, same, seedB := st.AwardExpNative(&b, 100, 7)
	if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(old, same) || seedA != seedB {
		t.Fatal("100% changed native growth or RNG")
	}
	st.Balance.Growth = GrowthRates{HP: 200, MP: 500, AP: 0, DP: 150, DX: 300}
	_, changed, seedC := st.AwardExpNative(&c, 100, 7)
	if seedC != seedA || len(changed) != 1 || changed[0].ApGain != 0 || changed[0].HpGain != old[0].HpGain*2 || changed[0].MpGain != 0 || c.HP != 10 {
		t.Fatalf("growth/RNG/heal changed: %+v %+v", old, changed)
	}
	d := base
	d.BattleFig = 36 // Promotion changes the growth selector, not the identity.
	st.Balance.Characters = map[int]GrowthRates{4: OriginalGrowthRates()}
	e := d
	st.GainExp(&d, 100, rand.New(rand.NewSource(7)))
	st.Balance = nil
	st.GainExp(&e, 100, rand.New(rand.NewSource(7)))
	if !reflect.DeepEqual(d, e) {
		t.Fatal("character override lost after promotion")
	}
}

func TestBalanceHitChangesActualPlanOnlyForSelectedCharacter(t *testing.T) {
	st, actor, target := nativeCommand0TestState()
	actor.NativeIdentity = 9
	actor.HasNativeIdentity = true
	st.NativeCommandBook[0].Hit = 0
	st.Balance = NewBalanceOptions()
	st.Balance.SkillHit = map[int]map[int]int{9: {0: 100}}
	plan, err := st.PlanNativeCommandDamage(actor, target, 0, map[int]int{5: 10}, 7)
	if err != nil || !plan.Results[0].Hit || plan.Results[0].Damage <= 0 || target.HP != 100 || actor.MP != 3 {
		t.Fatalf("custom hit plan %+v %v", plan, err)
	}
	actor.NativeIdentity = 10
	plan, err = st.PlanNativeCommandDamage(actor, target, 0, map[int]int{5: 10}, 7)
	if err != nil || plan.Results[0].Hit {
		t.Fatal("hit override leaked to another character")
	}
	if NativeCommandHasHitRoll(30) {
		t.Fatal("音速刃 acquired invented hit roll")
	}
	st.Balance.SkillHit[9][30] = 100
	if st.Balance.Validate() == nil {
		t.Fatal("accepted fabricated command30 hit setting")
	}
}

func TestBalanceEnemyScalingIsIdempotentAndPreservesWounds(t *testing.T) {
	u := &Unit{Camp: Enemy, HP: 30, MaxHP: 100, MP: 10, MaxMP: 20, AP: 65, DP: 40, DX: 10, HIT: 20, EV: 15, EquipmentBaseSet: true, BaseAP: 50, BaseDP: 30, BaseHIT: 20, BaseEV: 15}
	st := &State{Units: []*Unit{u}, Balance: NewBalanceOptions()}
	st.Balance.EnemyPercent = 200
	st.ApplyEnemyStrength()
	if u.MaxHP != 200 || u.HP != 60 || u.MaxMP != 40 || u.MP != 20 || u.AP != 115 || u.DP != 70 || u.DX != 20 || u.HIT != 30 {
		t.Fatalf("scaled unit %+v", u)
	}
	copy := *u
	st.ApplyEnemyStrength()
	if !reflect.DeepEqual(copy, *u) {
		t.Fatal("strength applied twice")
	}
	u.HP = 0
	st.Balance.EnemyPercent = 50
	st.ApplyEnemyStrength()
	if u.HP != 0 || u.MaxHP != 50 || u.AP != 40 || u.DX != 5 {
		t.Fatal("strength revived or compounded")
	}
	spawn := &Unit{Camp: Enemy, HP: 40, MaxHP: 40, AP: 10, DP: 8, DX: 6}
	st.Units = append(st.Units, spawn)
	st.ApplyEnemyStrength()
	if spawn.MaxHP != 20 || spawn.HP != 20 {
		t.Fatal("new spawn omitted")
	}
	st.Balance = nil
	st.ApplyEnemyStrength()
	if spawn.MaxHP != 40 || u.AP != 65 || u.HP != 0 {
		t.Fatal("reset did not restore baseline")
	}
}
