package battle

import (
	"fmt"
	"math"
	"math/rand"
)

// 原版普通攻擊的兩階段整數除法與玩家交易上限。
// 證據：docs/knowledge-base/94-ch01-town-parity-20260908.md。
// 只閉合 EXP 邊界，傷害與成長仍沿用既有執行期的證據限制。
type nativePhysicalExperiencePlan struct {
	base, targetMaxHP int
}

func planNativePhysicalExperience(actor, target *Unit) (nativePhysicalExperiencePlan, error) {
	fail := func() (nativePhysicalExperiencePlan, error) {
		return nativePhysicalExperiencePlan{}, fmt.Errorf("native physical EXP: raw source or integer state unavailable")
	}
	if actor == nil || target == nil || !actor.HasNativeRecordByte6 || !actor.HasNativeRecordByte5 {
		return fail()
	}
	if actor.NativeRecordByte6 != 2 {
		return nativePhysicalExperiencePlan{}, nil
	}
	if actor.Camp != Own || !actor.HasNativeRecordClass ||
		actor.Lv < 1 || actor.Lv > 255 || actor.Exp < 0 || actor.Exp >= 100 ||
		math.IsNaN(actor.Exp) || actor.Exp != math.Trunc(actor.Exp) || !target.HasBattleFig {
		return fail()
	}
	identity, ok := nativeRecordByte8ForUnit(actor)
	if !ok || target.BattleFig < 0 || target.BattleFig > 255 {
		return fail()
	}
	if target.BattleFig < 68 {
		return nativePhysicalExperiencePlan{}, nil
	}
	table := target.NativeConstructor
	if table == nil || table.validate() != nil || table.Branch != "high_class" ||
		table.Index != target.BattleFig-68 || target.Lv < 1 || target.Lv > 255 ||
		target.MaxHP < 1 || target.MaxHP > 65535 {
		return fail()
	}
	level := byte(actor.Lv)
	if actor.NativeRecordClass > 8 && actor.NativeRecordClass < 25 || identity == 28 {
		level += 30
	}
	if level == 0 {
		return fail()
	}
	return nativePhysicalExperiencePlan{
		base: int(table.Record[9]) * target.Lv / int(level), targetMaxHP: target.MaxHP,
	}, nil
}

// award 是 `0x2A209..0x2A27A` 的一擊經驗：守方倒下（`[esp+0x28]` 剩餘 HP 為 0）
// 就是整份 base，否則乘上「這一擊的傷害／守方最大 HP」——打空時傷害 0 就是 0。
// 99 的上限不在這裡：那是玩家路徑 `0x11959` 才有的，敵方路徑 `0x1566A` 沒有。
func (p nativePhysicalExperiencePlan) award(damage int, killed bool) int {
	if p.base == 0 {
		return 0
	}
	if killed {
		return p.base
	}
	if damage <= 0 {
		return 0
	}
	return damage * p.base / p.targetMaxHP
}

// AttackWithNativeExperience 先預檢所有 EXP 來源，再消耗 RNG 與提交攻擊。
// 玩家與敵方正式物理入口共用；不在 raw 資料缺少時猜用相容 ex 欄。
func (s *State) AttackWithNativeExperience(actor, target *Unit, rng *rand.Rand) (AttackResult, error) {
	if s == nil || rng == nil {
		return AttackResult{}, fmt.Errorf("native physical EXP: state or RNG unavailable")
	}
	plan, err := planNativePhysicalExperience(actor, target)
	if err != nil {
		return AttackResult{}, err
	}
	return s.attackWithExperience(actor, target, rng, &plan), nil
}
