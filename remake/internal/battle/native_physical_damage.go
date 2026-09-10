package battle

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// native_physical_damage.go — 原版 `sub_29F72`（0x29F72..0x2A289）的物理傷害。
//
// 這裡刻意只做「一次擲骰」，不碰單位狀態、不碰演出：原版把結果填進呼叫者的六格
// 陣列，`sub_2939D` 才依那六格決定演出與逐格扣血。證據見
// docs/knowledge-base/106-physical-attack-counterattack-20260910.md。
//
// **RNG 的消耗次數與順序也是契約。** 原版的 `sub_4E893` 是全域狀態，攻擊、演出
// 與 AI 共用同一條序列；少擲或多擲一次，之後每一次結算都會偏掉。所以未命中就
// 不再擲暴擊與隨機加成，武器分支該擲兩次的就擲兩次。

// NativePhysicalWeapon 是攻方武器 record（`sub_4E56C` 的 0x17 位元組列）在傷害
// 公式裡用到的三個欄位。`Reach` 不參與傷害，是反擊資格用的（見
// NativeCounterattackEligible）。
type NativePhysicalWeapon struct {
	Effect int // record +9：4＝暴擊加成、2＝狀態異常、3＝連擊
	Param  int // record +10：Effect 4 是暴擊加成，Effect 2 是發動機率
	Reach  int // record +11
}

// NativePhysicalRoll 是一次物理擲骰的輸入。攻守雙方的數值都是**已生效值**，
// 也就是原版 record 上的 `+0x48`／`+0x4A`／`+0x4C`／`+0x4E`；地形百分比由呼叫者
// 依 `sub_1F183` 的飛行閘門決定，飛行時傳 0。
type NativePhysicalRoll struct {
	AttackerAP  int
	DefenderDP  int
	AttackerHit int
	DefenderEV  int

	// AttackerCritPct 是 0x5239B[職業−1]，見 native_combat_tables.json。
	AttackerCritPct int

	AttackerTerrainAPPct int
	DefenderTerrainDPPct int

	Weapon   NativePhysicalWeapon
	RNGState uint16
}

// NativePhysicalRollResult 對應原版填進呼叫者的六格陣列。
type NativePhysicalRollResult struct {
	Missed bool // out[0]：原版 1 是未命中、0 是命中
	Crit   bool // out[1]
	Status bool // out[2]：武器 Effect 2 發動
	Extra  bool // out[4]：武器 Effect 3，同一次指令再打一次
	Damage int  // out[5]，也是原版的回傳值

	// StatusValue 是 Status 為真時原版寫進守方 `+37` 的值（`rand()%4 + 2`）。
	StatusValue int

	RNGState uint16
}

// RollNativePhysicalDamage 依 `sub_29F72` 的順序擲一次物理傷害。
func RollNativePhysicalDamage(in NativePhysicalRoll) (NativePhysicalRollResult, error) {
	if in.AttackerAP < 0 || in.DefenderDP < 0 {
		return NativePhysicalRollResult{}, fmt.Errorf(
			"native physical roll has negative stats: ap=%d dp=%d", in.AttackerAP, in.DefenderDP)
	}
	rng := in.RNGState
	next := func() int {
		rng = fdother.NativeRNGStep(rng)
		return int(rng)
	}

	// 1. 地形修正。原版是 `v30 += v30 * 表值 / 100`，所以負百分比也走同一式。
	ap := in.AttackerAP + in.AttackerAP*in.AttackerTerrainAPPct/100
	dp := in.DefenderDP + in.DefenderDP*in.DefenderTerrainDPPct/100

	// 2. 武器分支（0x29FE6 的 switch），在命中判定之前。
	result := NativePhysicalRollResult{Missed: true}
	critPct := in.AttackerCritPct
	switch in.Weapon.Effect {
	case 4:
		critPct += in.Weapon.Param
	case 2:
		if next()%100 < in.Weapon.Param {
			result.Status = true
			result.StatusValue = next()%4 + 2
		}
	case 3:
		result.Extra = true
	}

	// 3. 命中：`rand()%100 < 命中 − 迴避`。未命中就到此為止，不再擲。
	if next()%100 >= in.AttackerHit-in.DefenderEV {
		result.RNGState = rng
		return result, nil
	}
	result.Missed = false

	// 4. 暴擊讓守方防禦減半——**在地形修正之後**。
	if next()%100 < critPct {
		result.Crit = true
		dp /= 2
	}

	// 5. 傷害：九成的差額，再加最多約一成的隨機。
	damage := 9 * (ap - dp) / 10
	if damage < 0 {
		damage = 0
	}
	if span := damage / 9; span != 0 {
		damage += next() % span
	}
	result.Damage = damage
	result.RNGState = rng
	return result, nil
}

// NativeCounterattackDefender 是反擊資格判定需要的守方資訊。`WeaponReach` 是守方
// 已裝備第 0 類裝備的 record `+11`；沒有已裝備的第 0 類時 HasWeapon 為 false。
type NativeCounterattackDefender struct {
	X, Y        int
	Byte38      int // record +38 ＝ +0x26，六個暫時狀態之一；非零就不反擊
	HasWeapon   bool
	WeaponReach int
	AliveHP     int
}

// NativeCounterattackEligible 重現 `sub_1F0DC(攻方, 守方) == 1` 加上呼叫端的
// 「守方還活著」判斷。原版的完整條件是：
//
//	sub_2939D(攻方, 守方, …) != 0   守方打完 HP 還大於 0
//	sub_1F0DC(攻方, 守方) == 1      守方 +38 為 0、兩格相鄰、已裝備第 0 類且 +11 為 1
//	dword_540FF == 0                不在抑制模式（該旗標的語意尚未解，重製端一律視為 0）
//
// 相鄰是曼哈頓距離剛好 1，所以隔空攻擊不會被反擊。
func NativeCounterattackEligible(attackerX, attackerY int, defender NativeCounterattackDefender) bool {
	if defender.AliveHP <= 0 {
		return false
	}
	if defender.Byte38 != 0 {
		return false
	}
	if abs(attackerX-defender.X)+abs(attackerY-defender.Y) != 1 {
		return false
	}
	if !defender.HasWeapon {
		return false
	}
	return defender.WeaponReach == 1
}
