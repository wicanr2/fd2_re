package battle

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// rngSequence 回傳從 state 起連續 n 次 NativeRNGStep 的值，用來預測原版擲骰。
func rngSequence(state uint16, n int) []uint16 {
	out := make([]uint16, 0, n)
	for i := 0; i < n; i++ {
		state = fdother.NativeRNGStep(state)
		out = append(out, state)
	}
	return out
}

// TestRollNativePhysicalDamageFollowsTheOriginalOrder 釘住 sub_29F72 的順序：
// 地形修正 →命中 →暴擊（防禦減半）→九成差額加隨機。順序若和重製端舊公式一樣
// 反過來（先暴擊減半再套地形），整數除法會給出不同的傷害。
func TestRollNativePhysicalDamageFollowsTheOriginalOrder(t *testing.T) {
	const state = 0x1234
	seq := rngSequence(state, 3)
	in := NativePhysicalRoll{
		AttackerAP: 40, DefenderDP: 11,
		AttackerHit: 100, DefenderEV: 0,
		AttackerCritPct:      100, // 讓暴擊必定發生，好驗證減半的時機
		AttackerTerrainAPPct: 10,
		DefenderTerrainDPPct: 10,
		RNGState:             state,
	}
	got, err := RollNativePhysicalDamage(in)
	if err != nil {
		t.Fatal(err)
	}
	if got.Missed || !got.Crit {
		t.Fatalf("命中 100%%、暴擊 100%% 卻得到 missed=%v crit=%v", got.Missed, got.Crit)
	}
	// 原版順序：ap=40+4=44、dp=11+1=12、暴擊後 dp=6 → 9*(44-6)/10 = 34。
	ap := 40 + 40*10/100
	dp := 11 + 11*10/100
	dp /= 2
	want := 9 * (ap - dp) / 10
	if span := want / 9; span != 0 {
		want += int(seq[2]) % span
	}
	if got.Damage != want {
		t.Fatalf("傷害 %d，原版順序算出來是 %d", got.Damage, want)
	}
	// 反過來（先減半再套地形）會是另一個值；不同才證明這支測試在看順序。
	reversed := 9 * ((40 + 40*10/100) - (11/2 + (11/2)*10/100)) / 10
	if reversed == 9*(ap-dp)/10 {
		t.Fatal("兩種順序算出同一個基礎傷害，這組數字驗不到順序")
	}
	if got.RNGState != seq[2] {
		t.Fatalf("RNG 停在 %#x，應該是第三次擲骰後的 %#x", got.RNGState, seq[2])
	}
}

// TestRollNativePhysicalDamageStopsRollingOnAMiss 釘住未命中就不再擲暴擊與隨機
// 加成。RNG 是全域共用的，多擲一次會讓之後每一次結算都偏掉。
func TestRollNativePhysicalDamageStopsRollingOnAMiss(t *testing.T) {
	const state = 0x4321
	seq := rngSequence(state, 1)
	got, err := RollNativePhysicalDamage(NativePhysicalRoll{
		AttackerAP: 40, DefenderDP: 5,
		AttackerHit: 0, DefenderEV: 0, // 命中率 0：`rand%100 >= 0` 恆成立
		AttackerCritPct: 100,
		RNGState:        state,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Missed {
		t.Fatal("命中率 0 卻命中了")
	}
	if got.Damage != 0 || got.Crit {
		t.Fatalf("未命中卻有傷害 %d／暴擊 %v", got.Damage, got.Crit)
	}
	if got.RNGState != seq[0] {
		t.Fatalf("未命中共擲了不只一次：停在 %#x，應為 %#x", got.RNGState, seq[0])
	}
}

// TestRollNativePhysicalDamageWeaponBranches 釘住武器 record +9 的三個分支，
// 包含它們各自消耗幾次 RNG。
func TestRollNativePhysicalDamageWeaponBranches(t *testing.T) {
	const state = 0x2468
	t.Run("effect4 把 +10 加進暴擊率", func(t *testing.T) {
		// 職業暴擊率 0、武器加成 100 → 必定暴擊。
		got, err := RollNativePhysicalDamage(NativePhysicalRoll{
			AttackerAP: 30, DefenderDP: 10, AttackerHit: 100,
			AttackerCritPct: 0, Weapon: NativePhysicalWeapon{Effect: 4, Param: 100},
			RNGState: state,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !got.Crit {
			t.Fatal("武器加成 100 卻沒有暴擊")
		}
	})
	t.Run("effect3 是連擊且不擲骰", func(t *testing.T) {
		seq := rngSequence(state, 3)
		got, err := RollNativePhysicalDamage(NativePhysicalRoll{
			AttackerAP: 30, DefenderDP: 10, AttackerHit: 100,
			Weapon: NativePhysicalWeapon{Effect: 3}, RNGState: state,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !got.Extra {
			t.Fatal("effect 3 沒有標記連擊")
		}
		if got.RNGState != seq[2] {
			t.Fatalf("effect 3 多擲或少擲了骰：停在 %#x，應為 %#x", got.RNGState, seq[2])
		}
	})
	t.Run("effect2 發動時多擲一次算狀態值", func(t *testing.T) {
		seq := rngSequence(state, 2)
		got, err := RollNativePhysicalDamage(NativePhysicalRoll{
			AttackerAP: 30, DefenderDP: 10, AttackerHit: 100,
			Weapon: NativePhysicalWeapon{Effect: 2, Param: 100}, RNGState: state,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !got.Status {
			t.Fatal("發動機率 100 卻沒有狀態異常")
		}
		if want := int(seq[1])%4 + 2; got.StatusValue != want {
			t.Fatalf("狀態值 %d，原版是第二次擲骰的 %%4+2 ＝ %d", got.StatusValue, want)
		}
		if got.StatusValue < 2 || got.StatusValue > 5 {
			t.Fatalf("狀態值 %d 落在 2..5 之外", got.StatusValue)
		}
	})
	t.Run("effect2 沒發動就只擲一次", func(t *testing.T) {
		got, err := RollNativePhysicalDamage(NativePhysicalRoll{
			AttackerAP: 30, DefenderDP: 10, AttackerHit: 100,
			Weapon: NativePhysicalWeapon{Effect: 2, Param: 0}, RNGState: state,
		})
		if err != nil {
			t.Fatal(err)
		}
		if got.Status {
			t.Fatal("發動機率 0 卻發動了")
		}
	})
}

// TestRollNativePhysicalDamageHasNoFloor 釘住原版沒有「至少 1 點」的下限：
// 防禦高於攻擊時傷害就是 0。重製端舊公式對玩家攻擊硬給 1 點，那不是原版行為。
func TestRollNativePhysicalDamageHasNoFloor(t *testing.T) {
	got, err := RollNativePhysicalDamage(NativePhysicalRoll{
		AttackerAP: 5, DefenderDP: 50, AttackerHit: 100, RNGState: 0x1111,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Missed {
		t.Fatal("命中率 100 卻沒命中")
	}
	if got.Damage != 0 {
		t.Fatalf("防禦遠高於攻擊時傷害是 %d，原版是 0", got.Damage)
	}
}

// TestNativeCounterattackEligibleChecksEveryCondition 逐條驗反擊資格，每一條都
// 要單獨拿掉一次——否則「全都成立時回真」證明不了任何一條真的被檢查。
func TestNativeCounterattackEligibleChecksEveryCondition(t *testing.T) {
	base := NativeCounterattackDefender{X: 5, Y: 19, AliveHP: 8, HasWeapon: true, WeaponReach: 1}
	const ax, ay = 6, 19
	if !NativeCounterattackEligible(ax, ay, base) {
		t.Fatal("四項條件全中卻判成不能反擊")
	}
	for _, tc := range []struct {
		name   string
		mutate func(*NativeCounterattackDefender)
	}{
		{"守方被打死", func(d *NativeCounterattackDefender) { d.AliveHP = 0 }},
		{"守方 +38 非零", func(d *NativeCounterattackDefender) { d.Byte38 = 1 }},
		{"沒有已裝備的第 0 類", func(d *NativeCounterattackDefender) { d.HasWeapon = false }},
		{"武器 +11 不是 1", func(d *NativeCounterattackDefender) { d.WeaponReach = 2 }},
		{"隔一格", func(d *NativeCounterattackDefender) { d.X = 4 }},
		{"斜角", func(d *NativeCounterattackDefender) { d.X, d.Y = 5, 20 }},
		{"同一格", func(d *NativeCounterattackDefender) { d.X, d.Y = ax, ay }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defender := base
			tc.mutate(&defender)
			if NativeCounterattackEligible(ax, ay, defender) {
				t.Fatalf("%s 仍判成可以反擊", tc.name)
			}
		})
	}
}

// TestCounterattackReadsTheRealTransientByte 釘住反擊判定讀的是真正的
// `+0x26`（六個暫時狀態剩餘回合數之一），不是恆為 0 的佔位值。原版
// `sub_1F0DC` 讀 `unit+38`，同一個閘門也擋掉回合開始的自動回復。
func TestCounterattackReadsTheRealTransientByte(t *testing.T) {
	state := &State{}
	defender := &Unit{X: 5, Y: 19, HP: 8}
	if got := state.counterattackDefender(defender); got.Byte38 != 0 {
		t.Fatalf("沒有狀態時 Byte38 是 %d，應為 0", got.Byte38)
	}
	if !defender.SetNativeTransientDuration(0x26, 3) {
		t.Fatal("寫不進 +0x26")
	}
	got := state.counterattackDefender(defender)
	if got.Byte38 != 3 {
		t.Fatalf("+0x26 設成 3 之後 Byte38 是 %d", got.Byte38)
	}
	if NativeCounterattackEligible(6, 19, got) {
		t.Fatal("+0x26 生效中仍判成可以反擊")
	}
	// 其他五個 transient 不該影響反擊——原版只讀 +0x26。
	defender.SetNativeTransientDuration(0x26, 0)
	for _, offset := range []int{0x22, 0x23, 0x24, 0x25, 0x27} {
		defender.SetNativeTransientDuration(offset, 4)
	}
	other := state.counterattackDefender(defender)
	other.HasWeapon, other.WeaponReach = true, 1
	if !NativeCounterattackEligible(6, 19, other) {
		t.Fatal("只有其他 transient 生效時就不能反擊了，原版只看 +0x26")
	}
}
