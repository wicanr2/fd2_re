package battle

import "testing"

// highClassTarget 建一個帶原版高階職業表的守方：列表[9] = row9。
func highClassTarget(fig, lv, hp, maxHP int, row9 byte) *Unit {
	record := make([]byte, 10)
	record[9] = row9
	return &Unit{Camp: Enemy, Lv: lv, HP: hp, MaxHP: maxHP, BattleFig: fig, HasBattleFig: true,
		NativeConstructor: &NativeConstructorTable{Branch: "high_class", Index: fig - 0x44, Record: record}}
}

func TestNativeDamageExperienceFollows1C81F(t *testing.T) {
	target := highClassTarget(0x44, 5, 70, 100, 12)
	// 存活：12 × 5 × 擲出 30 / 最大 HP 100 = 18。
	if got, err := NativeDamageExperience(target, 30, 70); err != nil || got != 18 {
		t.Fatalf("存活經驗 %d／%v，應為 18", got, err)
	}
	// 擊倒：整份 12 × 5，不乘傷害比例。
	if got, err := NativeDamageExperience(target, 300, 0); err != nil || got != 60 {
		t.Fatalf("擊倒經驗 %d／%v，應為 60", got, err)
	}
	// 擲出值超過剩餘 HP 時仍用擲出值（0x1C8CF imul edx,ebp），不截斷。
	if got, _ := NativeDamageExperience(target, 150, 1); got != 90 {
		t.Fatalf("擲出值經驗 %d，應為 90", got)
	}
	// 0x1C8AA jl：圖號 0x43 不累加，也不需要表。
	low := &Unit{Lv: 5, HP: 10, MaxHP: 100, BattleFig: 0x43, HasBattleFig: true}
	if got, err := NativeDamageExperience(low, 30, 10); err != nil || got != 0 {
		t.Fatalf("圖號 0x43 經驗 %d／%v，應為 0", got, err)
	}
	// 沒有原版圖號的正規化單位回 0。
	if got, err := NativeDamageExperience(&Unit{Lv: 5, MaxHP: 100}, 30, 10); err != nil || got != 0 {
		t.Fatalf("正規化單位經驗 %d／%v，應為 0", got, err)
	}
	// 高階圖號缺表：失敗即關閉。
	missing := &Unit{Lv: 5, HP: 10, MaxHP: 100, BattleFig: 0x44, HasBattleFig: true}
	if _, err := NativeDamageExperience(missing, 30, 10); err == nil {
		t.Fatal("缺高階職業表卻沒有拒收")
	}
	wrongIndex := highClassTarget(0x45, 5, 10, 100, 12)
	wrongIndex.NativeConstructor.Index = 0
	if _, err := NativeDamageExperience(wrongIndex, 30, 10); err == nil {
		t.Fatal("高階職業表索引和圖號不符卻沒有拒收")
	}
}

func TestNativeRestoreExperienceFollows1C916(t *testing.T) {
	target := func(fig int, class byte) *Unit {
		return &Unit{Camp: Own, Lv: 5, HP: 60, MaxHP: 80, BattleFig: fig, HasBattleFig: true,
			NativeRecordClass: class, HasNativeRecordClass: true}
	}
	cases := []struct {
		name  string
		fig   int
		class byte
		want  int
	}{
		{"職業 8 不加 30", 4, 8, 40 * 5 * 20 / 80},
		{"職業 9 加 30", 4, 9, 40 * 35 * 20 / 80},
		{"職業 24 加 30", 4, 24, 40 * 35 * 20 / 80},
		{"職業 25 不加 30", 4, 25, 40 * 5 * 20 / 80},
		{"圖號 0x4A 仍累加", 0x4a, 8, 40 * 5 * 20 / 80},
		{"圖號 0x4B 不累加", 0x4b, 8, 0},
	}
	for _, c := range cases {
		if got := NativeRestoreExperience(target(c.fig, c.class), 20); got != c.want {
			t.Fatalf("%s：%d，應為 %d", c.name, got, c.want)
		}
	}
	if got := NativeRestoreExperience(&Unit{Lv: 5, MaxHP: 80}, 20); got != 0 {
		t.Fatalf("正規化單位回復經驗 %d，應為 0", got)
	}
	heal := NativeCommandHealExperience([]NativeCommandHealResult{
		{Target: target(4, 9), Restore: NativeCommandRestore{Rolled: 30, Actual: 20}},
		{Target: target(4, 8), Restore: NativeCommandRestore{Rolled: 30, Actual: 0}},
	})
	if heal != 40*35*20/80 {
		t.Fatalf("回復指令經驗 %d，應為 %d", heal, 40*35*20/80)
	}
}

func TestNativeCommandDamageExperienceSkipsMisses(t *testing.T) {
	hit := highClassTarget(0x44, 5, 70, 100, 12)
	miss := highClassTarget(0x44, 5, 100, 100, 12)
	got, err := NativeCommandDamageExperience([]NativeCommandDamageResult{
		{Target: hit, NativeCommandDamage: NativeCommandDamage{Hit: true, Damage: 30}, HPBefore: 100, HPAfter: 70},
		{Target: miss, NativeCommandDamage: NativeCommandDamage{Hit: false}, HPBefore: 100, HPAfter: 100},
	})
	if err != nil || got != 18 {
		t.Fatalf("指令傷害經驗 %d／%v，應為 18", got, err)
	}
	strike, err := NativeCommandDerivedStrikeExperience([]NativeCommand24Damage{
		{Target: highClassTarget(0x44, 5, 0, 100, 12), Rolled: 140, Damage: 100, HPBefore: 100, HPAfter: 0},
	})
	if err != nil || strike != 60 {
		t.Fatalf("衍生打擊擊倒經驗 %d／%v，應為 60", strike, err)
	}
}

func TestNativeCommand2022ExperienceFollowsTailWriters(t *testing.T) {
	ally := func(lv int, class byte) *Unit {
		return &Unit{Camp: Own, Lv: lv, HP: 40, MaxHP: 80, BattleFig: 4, HasBattleFig: true,
			NativeRecordClass: class, HasNativeRecordClass: true}
	}
	// 25 → 0x22C04：只有 +5 bit7 被清掉的目標，8 × L。
	acted, idle := ally(5, 9), ally(5, 9)
	plan := &NativeAICommand2022Plan{CommandID: 25, Results: []NativeAICommand2022Result{
		{Target: acted, Command25: true, ClearedByte5: true},
		{Target: idle, Command25: true},
	}}
	if got, err := NativeCommand2022Experience(plan); err != nil || got != 8*35 {
		t.Fatalf("指令 25 經驗 %d／%v，應為 %d", got, err, 8*35)
	}
	// 20／21 → 0x22AF6：4 × L 加上 0x1C916 的回復經驗。
	cleared := ally(5, 8)
	plan = &NativeAICommand2022Plan{CommandID: 20, Results: []NativeAICommand2022Result{
		{Target: cleared, Restore: &NativeRawFlagRestoreResult{Applied: true, Restore: NativeRawRestoreResult{Score: 7}}},
		{Target: ally(5, 8), Restore: &NativeRawFlagRestoreResult{}},
	}}
	if got, err := NativeCommand2022Experience(plan); err != nil || got != 7+4*5 {
		t.Fatalf("指令 20 經驗 %d／%v，應為 %d", got, err, 7+4*5)
	}
	// 22／26／27 → 0x22D1B：0x1C81F 傷害經驗加 8 × +0x21（不加 30）。
	enemy := highClassTarget(0x44, 6, 91, 100, 10)
	enemy.NativeRecordClass, enemy.HasNativeRecordClass = 9, true
	plan = &NativeAICommand2022Plan{CommandID: 26, Targets: []*Unit{enemy},
		Results: []NativeAICommand2022Result{{Target: enemy, Apply: &NativeRawApplicationResult{
			TargetIndex: 0, Applied: true, Damage: NativeRawDamageResult{Rolled: 9}}}},
		After: []NativeAICommand2022TargetState{{Target: enemy, HP: 91}}}
	want := 10*6*9/100 + 8*6
	if got, err := NativeCommand2022Experience(plan); err != nil || got != want {
		t.Fatalf("指令 26 經驗 %d／%v，應為 %d", got, err, want)
	}
}

func TestNativeCompoundCommandExperience(t *testing.T) {
	target := highClassTarget(0x44, 5, 70, 100, 12)
	got, err := NativeCompoundCommand32Experience(NativeCompoundCommand32Result{Targets: []NativeCompoundCommand32Target{
		{Target: target, Damage: NativeCommandDamage{Hit: true, Damage: 30}, HPBefore: 100, HPAfter: 70},
	}})
	if err != nil || got != 18 {
		t.Fatalf("指令 32 經驗 %d／%v，應為 18", got, err)
	}
	if got := NativeCompoundCommand33Experience(NativeCompoundCommand33Result{Restore: NativeRawRestoreBatch{
		Results: []NativeRawRestoreResult{{Score: 3}, {Score: 4}}}}); got != 7 {
		t.Fatalf("指令 33 經驗 %d，應為 7", got)
	}
	if got := NativeCompoundCommand34Experience(NativeCompoundCommand34Result{Stages: []NativeCommandModifierResult{
		{Accumulator: 10}, {Accumulator: 12}, {Accumulator: 0}}}); got != 22 {
		t.Fatalf("指令 34 經驗 %d，應為 22", got)
	}
	got, err = NativeCompoundCommand35Experience(NativeCompoundCommand35Result{
		Stages: []NativeCompoundCommand35Stage{{Accumulator: 40, Results: []NativeRawApplicationResult{
			{TargetIndex: 0, Applied: true, Damage: NativeRawDamageResult{Rolled: 10}}}}},
		StageStates: [][]NativeCommand35TargetState{{{Target: target, HP: 60}}},
	})
	if err != nil || got != 40+12*5*10/100 {
		t.Fatalf("指令 35 經驗 %d／%v，應為 %d", got, err, 40+12*5*10/100)
	}
}

func TestAwardNativePlayerActionExperienceFollows117E7(t *testing.T) {
	player := func(lv int) *Unit {
		return &Unit{Camp: Own, Lv: lv, HP: 50, MaxHP: 50, BattleFig: 4, HasBattleFig: true,
			NativeRecordByte5: 0, HasNativeRecordByte5: true, NativeRecordByte6: 2, HasNativeRecordByte6: true}
	}
	// 0x11959：超過 99 壓到 99；收下後 0x1E51A 清零。
	st := &State{}
	u := player(5)
	got, _, _ := st.AwardNativePlayerActionExperience(u, 150, 1)
	if got != 99 || u.Exp != 99 || st.NativeExperienceAccumulator != 0 {
		t.Fatalf("壓 99：收下 %d exp %v 累計 %d", got, u.Exp, st.NativeExperienceAccumulator)
	}
	// 0x1E2F2：滿級整段返回，不清累計。
	st = &State{}
	capped := player(40)
	got, _, _ = st.AwardNativePlayerActionExperience(capped, 30, 1)
	if got != 0 || capped.Exp != 0 || st.NativeExperienceAccumulator != 30 {
		t.Fatalf("滿級：收下 %d exp %v 累計 %d", got, capped.Exp, st.NativeExperienceAccumulator)
	}
	// 0x34B48：擊倒帶無條件 exp_cancel 的單位，整次行動經驗清零。
	victim := &Unit{Camp: Enemy, HP: 0}
	st = &State{NativeDeathExpCancel: func(target *Unit) bool { return target == victim }}
	u = player(5)
	got, _, _ = st.AwardNativePlayerActionExperience(u, 40, 1, victim)
	if got != 0 || u.Exp != 0 || st.NativeExperienceAccumulator != 0 {
		t.Fatalf("exp_cancel：收下 %d exp %v 累計 %d", got, u.Exp, st.NativeExperienceAccumulator)
	}
	// 存活的同一單位不觸發取消。
	victim.HP = 5
	got, _, _ = st.AwardNativePlayerActionExperience(u, 40, 1, victim)
	if got != 40 {
		t.Fatalf("目標存活時收下 %d，應為 40", got)
	}
}
