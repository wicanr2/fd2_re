package battle

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func deathTestState(n int) *State {
	st := &State{W: 24, H: 24}
	for i := 0; i < n; i++ {
		st.Units = append(st.Units, &Unit{
			Camp: Enemy, HP: 20, MaxHP: 20, OnField: true,
			NativeRecordByte5: 0, HasNativeRecordByte5: true,
			NativeRecordByte6: 0, HasNativeRecordByte6: true,
			NativeRecordByte34: 0xa3, HasNativeRecordByte34: true,
		})
	}
	return st
}

// 各原語對照轉寫的語意：0x3419C 只動低四位、0x34A85 整個覆寫、0x34A0E AND、
// 0x32975 整個 +5 寫 1、[0x53AD5] 設值與遞增、[0x51A83]=1。
func TestNativeDeathOpsFollowTranscription(t *testing.T) {
	st := deathTestState(8)
	must := func(op NativeDeathOp) {
		t.Helper()
		if err := st.ApplyNativeDeathOp(op); err != nil {
			t.Fatalf("%s: %v", op.Op, err)
		}
	}
	must(NativeDeathOp{Op: "ai_mode_range", First: 1, Last: 2, Mode: 7})
	if st.Units[1].NativeRecordByte34 != 0xa7 || st.Units[2].NativeRecordByte34 != 0xa7 ||
		st.Units[0].NativeRecordByte34 != 0xa3 || st.Units[3].NativeRecordByte34 != 0xa3 {
		t.Fatalf("ai_mode_range 應只改 1..2 的低四位：%#x %#x %#x %#x", st.Units[0].NativeRecordByte34,
			st.Units[1].NativeRecordByte34, st.Units[2].NativeRecordByte34, st.Units[3].NativeRecordByte34)
	}
	must(NativeDeathOp{Op: "ai_byte_set_range", First: 3, Last: 20, Value: 0})
	if st.Units[3].NativeRecordByte34 != 0 || st.Units[7].NativeRecordByte34 != 0 {
		t.Fatal("ai_byte_set_range 應整個覆寫，超出現存單位的索引忽略")
	}
	must(NativeDeathOp{Op: "ai_byte_and_range", First: 0, Last: 0, Mask: 0x80})
	if st.Units[0].NativeRecordByte34 != 0x80 {
		t.Fatalf("ai_byte_and_range：%#x", st.Units[0].NativeRecordByte34)
	}
	must(NativeDeathOp{Op: "mark_inactive", Unit: 4})
	if st.Units[4].OnField || st.Units[4].NativeRecordByte5 != 1 {
		t.Fatal("mark_inactive 應讓單位離場並把 +5 寫成 1")
	}
	must(NativeDeathOp{Op: "state_set", Index: 0x10, Value: 1})
	must(NativeDeathOp{Op: "state_inc", Index: 0x10})
	if st.NativeEventState[0x10] != 2 {
		t.Fatalf("狀態表 0x10 = %d，應為 2", st.NativeEventState[0x10])
	}
	must(NativeDeathOp{Op: "range_one"})
	if st.NativeMapRangeMode != 1 || !st.HasNativeMapRangeModeState {
		t.Fatal("range_one 應把地圖範圍選擇值設回 1")
	}
	if err := st.ApplyNativeDeathOp(NativeDeathOp{Op: "staging"}); err == nil {
		t.Fatal("staging 由介面擁有者執行，狀態層不該默默接受")
	}
	// 還沒建構的記錄：原版照寫 raw 陣列，之後建構器整筆覆寫，所以是無作用。
	if err := st.ApplyNativeDeathOp(NativeDeathOp{Op: "record_bytes", Unit: 99,
		Writes: [][3]int{{6, 1, 1}}}); err != nil {
		t.Fatalf("寫還沒建構的記錄應為無作用：%v", err)
	}
	if err := st.ApplyNativeDeathOp(NativeDeathOp{Op: "record_bytes", Unit: -1}); err == nil {
		t.Fatal("負索引應該失敗")
	}
}

// 事件 30：萊汀倒下時被改寫成友軍身分 6、HP 1，死亡效果清掉，下一次也不會再觸發。
func TestNativeDeathRecordWritesReviveAsAlly(t *testing.T) {
	st := deathTestState(12)
	u := st.Units[11]
	u.HP, u.NativeRecordByte5 = 0, 1
	u.BattleFig, u.HasBattleFig = 118, true
	u.NativeRecordByte8, u.HasNativeRecordByte8 = 118, true
	u.DeathEffect = &DeathEffect{Type: 2, Value: 30}
	u.NativeRecordDeathEffect, u.HasNativeRecordDeathEffect = [3]byte{2, 30, 0}, true
	err := st.ApplyNativeDeathOp(NativeDeathOp{Op: "record_bytes", Unit: 11, Writes: [][3]int{
		{5, 0, 1}, {6, 1, 1}, {7, 6, 1}, {8, 6, 1}, {0x31, 0xff, 1}, {0x34, 0x80, 1}, {0x40, 1, 2}}})
	if err != nil {
		t.Fatal(err)
	}
	if u.HP != 1 || u.NativeRecordByte5 != 0 || !u.OnField || u.Camp != Ally || u.NativeRecordByte6 != 1 ||
		u.BattleFig != 6 || u.NativeRecordByte8 != 6 || u.NativeRecordByte34 != 0x80 {
		t.Fatalf("復活後 %+v", *u)
	}
	if _, _, ok := NativeDeathEffectOf(u); ok || u.DeathEffect != nil {
		t.Fatal("+0x31=0xFF 之後不應再有死亡效果")
	}
	if err := st.ApplyNativeDeathOp(NativeDeathOp{Op: "record_bytes", Unit: 11,
		Writes: [][3]int{{0x20, 1, 1}}}); err == nil {
		t.Fatal("沒有型別化欄位的 offset 應整筆拒絕")
	}
}

// 回合事件控制列：[0x53A55]+3+3×slot = [0x53BEF] + delta，缺回合來源就拒絕。
func TestNativeDeathControlTurnNeedsRoundProvenance(t *testing.T) {
	st := deathTestState(1)
	if err := st.ApplyNativeDeathOp(NativeDeathOp{Op: "control_turn", Slot: 1, Delta: 2}); err == nil {
		t.Fatal("沒有 [0x53BEF] 來源時不該猜回合")
	}
	st.HasNativeTurnEventControlState, st.NativeRoundCounter = true, 5
	st.NativeTurnEventControls[1] = NativeTurnEventControl{Turn: 0xff, EventID: 31, RawCamp: 0}
	if err := st.ApplyNativeDeathOp(NativeDeathOp{Op: "control_turn", Slot: 1, Delta: 2}); err != nil {
		t.Fatal(err)
	}
	if row := st.NativeTurnEventControls[1]; row.Turn != 7 || row.EventID != 31 {
		t.Fatalf("控制列 %+v，turn 應為 7 且事件不變", row)
	}
}

func TestNativeActionWhenReadsStateAtExecution(t *testing.T) {
	st := deathTestState(4)
	eq := &NativeActionWhen{StateEq: &[2]int{0x10, 1}}
	if ok, err := eq.Match(st); ok || err != nil {
		t.Fatalf("state_eq 還沒成立：%v %v", ok, err)
	}
	st.NativeEventState[0x10] = 1
	if ok, _ := eq.Match(st); !ok {
		t.Fatal("state_eq 應成立")
	}
	ne := &NativeActionWhen{StateNe: &[2]int{0x13, 0}}
	if ok, _ := ne.Match(st); ok {
		t.Fatal("state_ne：值是 0 時不成立")
	}
	round := 15
	lt := &NativeActionWhen{RoundLt: &round}
	if _, err := lt.Match(st); err == nil {
		t.Fatal("round_lt 沒有回合來源時應回錯")
	}
	st.NativeRoundCounter = 15
	if ok, _ := lt.Match(st); ok {
		t.Fatal("第 15 回合不小於 15（0x3488A jge）")
	}
	active := &NativeActionWhen{AnyActive: &[2]int{1, 2}}
	st.Units[1].NativeRecordByte5, st.Units[2].NativeRecordByte5 = 1, 1
	if ok, _ := active.Match(st); ok {
		t.Fatal("1..2 都不在場時 any_active 不成立")
	}
	st.Units[2].NativeRecordByte5 = 0
	if ok, _ := active.Match(st); !ok {
		t.Fatal("有一筆在場就成立")
	}
}

// 0x35BBA：從 first 起清 +0x40，帶 raw 投影時一起清。
func TestClearNativeHPFromSyncsProjection(t *testing.T) {
	st := deathTestState(4)
	st.HasNativeRuntimeUnitProjection = true
	st.NativeRuntimeRecords = make([]NativeRuntimeRecordState, 4)
	for i := range st.NativeRuntimeRecords {
		st.NativeRuntimeRecords[i].Raw[0x40] = 20
	}
	cleared := st.ClearNativeHPFrom(2)
	if len(cleared) != 2 || st.Units[1].HP != 20 || st.Units[2].HP != 0 || st.Units[3].HP != 0 {
		t.Fatalf("清除範圍錯誤：%d 筆", len(cleared))
	}
	if st.NativeRuntimeRecords[1].Raw[0x40] != 20 || st.NativeRuntimeRecords[3].Raw[0x40] != 0 {
		t.Fatal("raw 投影沒有跟著清")
	}
}

// 事件 30 的 [0x53EC8]=0：擊倒帶無條件 exp_cancel 的單位，整次行動的經驗是 0。
func TestKillWithExpCancelAwardsNoExperience(t *testing.T) {
	rows, err := LoadNativeGrowthRows("../../../docs/data/exe_tables/growth.json")
	if err != nil {
		t.Fatal(err)
	}
	sc := &Scenario{NativeDeathPrograms: map[string][]Action{
		"2:30": {{Type: "native_death_op", NativeDeathOp: &NativeDeathOp{Op: "exp_cancel"}}},
	}}
	st := &State{W: 8, H: 8, NativeGrowthRows: rows, NativeDeathExpCancel: sc.NativeDeathCancelsExp}
	attacker := &Unit{Camp: Own, Lv: 5, HP: 40, MaxHP: 40, AP: 99, HIT: 200, OnField: true,
		BattleFig: 4, HasBattleFig: true}
	boss := &Unit{Camp: Enemy, Lv: 5, HP: 1, MaxHP: 30, DP: 0, EV: 0, OnField: true, X: 1,
		ExpPerLevel: 20, DeathEffect: &DeathEffect{Type: 2, Value: 30}}
	st.Units = []*Unit{attacker, boss}
	result := st.AttackWithRNG(attacker, boss, rand.New(rand.NewSource(1)))
	if boss.HP != 0 {
		t.Fatalf("測試前提：攻擊應擊倒頭目，剩 HP %d", boss.HP)
	}
	if result.ExpGained != 0 || attacker.Exp != 0 {
		t.Fatalf("擊倒萊汀不該有經驗：收下 %v、exp %v", result.ExpGained, attacker.Exp)
	}
	// 反對照：沒有取消的死亡效果照常給經驗。
	boss2 := &Unit{Camp: Enemy, Lv: 5, HP: 1, MaxHP: 30, OnField: true, X: 1, ExpPerLevel: 20,
		DeathEffect: &DeathEffect{Type: 3, Value: 8}}
	st.Units[1] = boss2
	if r := st.AttackWithRNG(attacker, boss2, rand.New(rand.NewSource(1))); r.ExpGained == 0 {
		t.Fatal("反對照：一般擊倒應有經驗")
	}
}

// 全部戰場的死亡效果都有程式可跑；死亡事件生成的群組不能在開局就在場。
func TestNativeDeathProgramsCoverEveryBattleMap(t *testing.T) {
	known := map[string]bool{"native_death_op": true, "dialogue": true, "pan": true,
		"native_acting": true, "reset_pose": true, "spawn_group": true, "delay": true}
	ops := map[string]bool{"ai_mode_range": true, "ai_byte_set_range": true, "ai_byte_and_range": true,
		"record_bytes": true, "control_turn": true, "state_set": true, "state_inc": true,
		"mark_inactive": true, "range_one": true, "exp_cancel": true, "staging": true,
		"clear_hp_from": true, "reward": true}
	gaps := map[string]bool{}
	paths, _ := filepath.Glob("../../assets/scenarios/ch[0-9][0-9].json")
	if len(paths) != 30 {
		t.Fatalf("劇本 %d 份，應為 30", len(paths))
	}
	for _, path := range paths {
		sc, err := LoadScenario(path)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(fmt.Sprintf("../../assets/maps/map%d/map%d_units.json", sc.Map, sc.Map))
		if err != nil {
			t.Fatal(err)
		}
		var units struct {
			Units []struct {
				Group       int          `json:"group"`
				DeathEffect *DeathEffect `json:"death_effect"`
			} `json:"units"`
		}
		if err := json.Unmarshal(raw, &units); err != nil {
			t.Fatal(err)
		}
		source := fmt.Sprintf("FDTXT_%03d", sc.Map+1)
		initial := map[int]bool{}
		for _, g := range sc.InitialGroups {
			initial[g] = true
		}
		for _, u := range units.Units {
			e := u.DeathEffect
			if e == nil || (e.Type != 2 && e.Type != 3) {
				continue
			}
			key := fmt.Sprintf("%d:%d", e.Type, e.Value)
			name := filepath.Base(path) + " " + key
			program, ok := sc.NativeDeathPrograms[key]
			if !ok {
				if !gaps[name] {
					t.Errorf("%s 沒有死亡程式", name)
				}
				continue
			}
			if gaps[name] {
				t.Errorf("%s 已經有程式了，把它從已知缺口移走", name)
			}
			for _, a := range program {
				if !known[a.Type] {
					t.Errorf("%s 有未知動作 %q", name, a.Type)
				}
				if a.NativeDeathOp != nil && !ops[a.NativeDeathOp.Op] {
					t.Errorf("%s 有未知原語 %q", name, a.NativeDeathOp.Op)
				}
				if a.NativeDialogueRef != nil && a.NativeDialogueRef.SourceDAT != source {
					t.Errorf("%s 的對白來自 %s，戰場文字庫應是 %s", name, a.NativeDialogueRef.SourceDAT, source)
				}
				spawned := a.Groups
				if a.NativeDeathOp != nil && a.NativeDeathOp.Op == "staging" {
					spawned = []int{a.NativeDeathOp.Group}
				}
				for _, g := range spawned {
					if initial[g] {
						t.Errorf("%s 會生成群組 %d，它卻在開局就在場", name, g)
					}
				}
			}
		}
	}
}
