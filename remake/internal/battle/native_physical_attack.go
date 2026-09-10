package battle

import (
	"fmt"
	"math/rand"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// native_physical_attack.go — 把 `sub_28A6C` 的兩次 `sub_2939D` 接成一次完整的
// 物理攻擊：攻方先打，守方符合條件就還手。證據見
// docs/knowledge-base/106-physical-attack-counterattack-20260910.md。

// NativePhysicalStrike 是一次揮擊：一次 `sub_29F72` 加上把傷害寫回守方。
// 原版同一次 `sub_2939D` 可能揮兩次——開頭 3% 的機率，或武器 record `+9`＝3。
type NativePhysicalStrike struct {
	Roll       NativePhysicalRollResult
	Damage     int // 實際扣掉的 HP（守方剩餘不足時小於 Roll.Damage）
	DefenderHP int // 這一擊之後守方的 HP
}

// NativePhysicalAttackResult 是一次物理攻擊指令的完整結果。
type NativePhysicalAttackResult struct {
	Attack        AttackResult // 主攻：經驗與升級只算在攻方身上
	AttackStrikes []NativePhysicalStrike

	// Counter 為 nil 表示沒有反擊。反擊不給守方經驗——原版的經驗處理
	// （`sub_1B6B7`→`sub_1AA1D`）在 `sub_28A6C` 之外，而且只對攻方跑一次。
	Counter        *AttackResult
	CounterStrikes []NativePhysicalStrike

	RNGState uint16
}

// nativeIgnoresTerrainModifier 重現 `sub_1F183`：回真就不套地形攻防修正，也就是
// 飛行。缺欄位時各欄位當 0，那正好落在「非飛行」那一邊（0 ≠ 28、0 ≠ 19、
// 0 ∉ {4,5}），與重製端既有行為一致。
func nativeIgnoresTerrainModifier(u *Unit) bool {
	if u == nil {
		return false
	}
	fig := 0
	if u.HasBattleFig {
		fig = u.BattleFig
	}
	if fig == 28 {
		return false
	}
	if u.HasNativeRecordClass && u.NativeRecordClass == 19 {
		return true
	}
	if u.HasNativeRecordRace {
		race := int(u.NativeRecordRace)
		return race == 4 || race == 5
	}
	return false
}

// nativeEquippedWeaponSlot 重現 `sub_1B83D(unit, 0)`：找第一個「已裝備且 ID 小於
// 0x80」的槽。原版看的是 `unit+10+2i` 的 bit 0x40；有原始旗標就用原始旗標，沒有
// 才退回正規化的 Equipped。
func nativeEquippedWeaponSlot(u *Unit) (int, bool) {
	if u == nil {
		return 0, false
	}
	for slot := 0; slot < 8 && slot < len(u.Inventory); slot++ {
		if u.Inventory[slot] >= 0x80 {
			continue
		}
		if slot < len(u.NativeInventoryFlags) {
			if u.NativeInventoryFlags[slot]&0x40 != 0 {
				return slot, true
			}
			continue
		}
		if slot < len(u.Equipped) && u.Equipped[slot] {
			return slot, true
		}
	}
	return 0, false
}

// nativeEquippedWeapon 取出攻方武器 record 的 `+9`／`+10`／`+11`。拿不到原版物品表
// 時回零值，也就是沒有武器特效——那與重製端接線前的行為相同，不是新的猜測。
func (s *State) nativeEquippedWeapon(u *Unit) (NativePhysicalWeapon, bool) {
	slot, ok := nativeEquippedWeaponSlot(u)
	if !ok {
		return NativePhysicalWeapon{}, false
	}
	offset, err := NativeItemEffectRowOffset(u.Inventory[slot])
	if err != nil || offset+NativeItemEffectRowSize > len(s.nativeFutureItemRows) {
		return NativePhysicalWeapon{}, false
	}
	row := s.nativeFutureItemRows[offset : offset+NativeItemEffectRowSize]
	return NativePhysicalWeapon{
		Effect: int(row[9]), Param: int(row[10]), Reach: int(row[11]),
	}, true
}

// buildNativePhysicalRoll 組出一次擲骰的輸入。
func (s *State) buildNativePhysicalRoll(a, d *Unit, rngState uint16) NativePhysicalRoll {
	apPct, dpPct := 0, 0
	if !nativeIgnoresTerrainModifier(a) {
		apPct, _ = s.TerrainAPDPPct(a.X, a.Y)
	}
	if !nativeIgnoresTerrainModifier(d) {
		_, dpPct = s.TerrainAPDPPct(d.X, d.Y)
	}
	weapon, _ := s.nativeEquippedWeapon(a)
	return NativePhysicalRoll{
		AttackerAP: a.EffectiveAP(), DefenderDP: d.EffectiveDP(),
		AttackerHit: a.EffectiveHIT(), DefenderEV: d.EffectiveEV(),
		AttackerCritPct:      a.CritPct,
		AttackerTerrainAPPct: apPct, DefenderTerrainDPPct: dpPct,
		Weapon:   weapon,
		RNGState: rngState,
	}
}

// nativePhysicalExchange 重現一次 `sub_2939D`：開頭 3% 決定要不要揮兩次，之後每
// 一輪擲一次 `sub_29F72` 並把傷害寫回；武器 record `+9`＝3 會再多一輪，但只多一次
// （原版的 `v40` 旗標）。每一輪都重讀守方當下的 HP，所以第二擊打的是已扣血的目標。
func (s *State) nativePhysicalExchange(a, d *Unit, rngState uint16) ([]NativePhysicalStrike, uint16, error) {
	rng := fdother.NativeRNGStep(rngState)
	budget := 1
	if int(rng)%100 < 3 {
		budget = 2
	}
	strikes := make([]NativePhysicalStrike, 0, 2)
	extraUsed := false
	for budget > 0 {
		budget--
		roll, err := RollNativePhysicalDamage(s.buildNativePhysicalRoll(a, d, rng))
		if err != nil {
			return nil, rngState, err
		}
		rng = roll.RNGState
		strike := NativePhysicalStrike{Roll: roll}
		if !roll.Missed {
			before := d.HP
			d.ApplyHPDamage(roll.Damage)
			strike.Damage = before - d.HP
		}
		strike.DefenderHP = d.HP
		strikes = append(strikes, strike)
		if !extraUsed && roll.Extra {
			extraUsed = true
			budget++
		}
	}
	return strikes, rng, nil
}

// counterattackDefender 把守方的狀態整理成反擊資格判定的輸入。`Byte38` 是原版
// `unit+38`，重製端還沒有那個欄位，語意也未解，所以一律當 0（＝不阻擋反擊）。
func (s *State) counterattackDefender(d *Unit) NativeCounterattackDefender {
	weapon, ok := s.nativeEquippedWeapon(d)
	return NativeCounterattackDefender{
		X: d.X, Y: d.Y, AliveHP: d.HP,
		HasWeapon: ok, WeaponReach: weapon.Reach,
	}
}

// AttackNativePhysical 依原版結算一次物理攻擊，含反擊。rngState 是原版的全域
// `0x627B8`；呼叫者要把回傳的新狀態存回去，否則之後每一次結算都會偏掉。
//
// rng 只用在經驗值與升級（那條仍是重製端既有的規則），傷害本身完全走原版 RNG。
func (s *State) AttackNativePhysical(a, d *Unit, rngState uint16, rng *rand.Rand) (NativePhysicalAttackResult, error) {
	return s.attackNativePhysical(a, d, rngState, rng, nil)
}

func (s *State) attackNativePhysical(
	a, d *Unit, rngState uint16, rng *rand.Rand, nativeEXP *nativePhysicalExperiencePlan,
) (NativePhysicalAttackResult, error) {
	a.Acted = true
	if nativeEXP != nil && a.HasNativeRecordByte5 {
		a.NativeRecordByte5 |= 0x80
	}

	strikes, next, err := s.nativePhysicalExchange(a, d, rngState)
	if err != nil {
		return NativePhysicalAttackResult{}, err
	}
	result := NativePhysicalAttackResult{AttackStrikes: strikes, RNGState: next}
	result.Attack = summarizeNativeStrikes(strikes)
	result.Attack.ExpGained, result.Attack.LevelUps = s.awardNativePhysicalExperience(
		a, d, result.Attack.Amount, rng, nativeEXP)

	if !NativeCounterattackEligible(a.X, a.Y, s.counterattackDefender(d)) {
		return result, nil
	}
	counterStrikes, next, err := s.nativePhysicalExchange(d, a, result.RNGState)
	if err != nil {
		return NativePhysicalAttackResult{}, err
	}
	counter := summarizeNativeStrikes(counterStrikes)
	result.Counter = &counter
	result.CounterStrikes = counterStrikes
	result.Attack.Counter = &counter
	result.RNGState = next
	return result, nil
}

// summarizeNativeStrikes 把一次交鋒的每一擊併成既有的 AttackResult 形狀：傷害相加，
// 全部落空才算 Missed，任一擊暴擊就算 Crit。
func summarizeNativeStrikes(strikes []NativePhysicalStrike) AttackResult {
	out := AttackResult{Missed: true}
	for _, strike := range strikes {
		out.Amount += strike.Damage
		if !strike.Roll.Missed {
			out.Missed = false
		}
		if strike.Roll.Crit {
			out.Crit = true
		}
	}
	return out
}

// awardNativePhysicalExperience 沿用重製端既有的經驗值規則；原版的經驗鏈
// （`sub_1B6B7`→`sub_1AA1D`）在 `sub_28A6C` 之外，本輪未解，所以不動它。
func (s *State) awardNativePhysicalExperience(
	a, d *Unit, damage int, rng *rand.Rand, nativeEXP *nativePhysicalExperiencePlan,
) (float64, []LevelUpEvent) {
	if a.Camp != Own && a.Camp != Ally {
		return 0, nil
	}
	forExp := damage
	if d.HP == 0 {
		forExp = d.MaxHP
	}
	exp := AttackExp(a.Lv, d.Lv, forExp, d.MaxHP, d.ExpPerLevel)
	if nativeEXP != nil {
		exp = float64(nativeEXP.award(damage, d.HP == 0))
	}
	return exp, s.GainExp(a, exp, rng)
}

// AttackNativePhysicalWithExperience 是正式物理入口：傷害與反擊照原版，經驗值在
// 攻方帶有原版 `+6` 來源時走 planNativePhysicalExperience，否則走重製端既有規則。
//
// 預檢在改動任何 HP 之前完成：raw 資料缺項時整筆拒絕，不猜用相容欄位。
func (s *State) AttackNativePhysicalWithExperience(
	actor, target *Unit, rngState uint16, rng *rand.Rand,
) (NativePhysicalAttackResult, error) {
	if s == nil || rng == nil {
		return NativePhysicalAttackResult{}, fmt.Errorf(
			"native physical attack: state or RNG unavailable")
	}
	if actor == nil || target == nil {
		return NativePhysicalAttackResult{}, fmt.Errorf(
			"native physical attack: actor or target unavailable")
	}
	if !actor.HasNativeRecordByte6 {
		return s.attackNativePhysical(actor, target, rngState, rng, nil)
	}
	plan, err := planNativePhysicalExperience(actor, target)
	if err != nil {
		return NativePhysicalAttackResult{}, err
	}
	return s.attackNativePhysical(actor, target, rngState, rng, &plan)
}
