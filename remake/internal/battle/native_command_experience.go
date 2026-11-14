package battle

import "fmt"

// 玩家指令的經驗累計：原版每個效果寫入端直接累加 `[0x53EC8]`，玩家控制器 `0x117E7`
// 在行動結束後統一發放：
//
//	0x118EE mov [0x53EC8],0        選到單位時清零（Game.confirm 已重現）
//	0x11943 call 0x18890            玩家行動（指令效果與死亡分派都在裡面）
//	0x11959 cmp [0x53EC8],63h       壓到 99
//	0x1196D call 0x1E292(行動單位)  發經驗與升級
//
// 寫入端清單由 IDA 9.4 對 FD2.EXE（SHA-256 222b7d06…）的資料 xref 取得
// （tools/ida_export_data_writers.py、tools/ida_export_function_listing.py）。
// 本檔的函式只算「這一次累加多少」，不寫狀態；發放由
// State.AwardNativePlayerActionExperience 負責。物理攻擊的寫入端 0x2A25B／0x2A275
// 另由 native_physical_experience.go 承接，不經過這裡。
//
// 下文的 L 是「經驗用等級」：記錄 +0x21（Unit.Lv），職業 +0x20 在 9..24 時加 30
// （各寫入端的 `cmp ecx,8; jle; cmp ecx,19h; jge; add edx,1Eh`）。

// nativeExperienceLevel 是各寫入端共用的 L。
func nativeExperienceLevel(u *Unit) int {
	level := u.Lv
	if u.HasNativeRecordClass && u.NativeRecordClass > 8 && u.NativeRecordClass < 0x19 {
		level += 30
	}
	return level
}

// nativeHighClassExperienceRow9 是 0x1C8B3..0x1C8BC 的 `0x4E4FF(+7 − 0x44)` 第 9 byte，
// 和物理經驗（0x2A209 起）讀同一張高階職業表。來源不完整時失敗即關閉。
func nativeHighClassExperienceRow9(target *Unit) (int, error) {
	table := target.NativeConstructor
	if table == nil || table.validate() != nil || table.Branch != "high_class" ||
		table.Index != target.BattleFig-0x44 || target.Lv < 1 || target.Lv > 255 ||
		target.MaxHP < 1 || target.MaxHP > 65535 {
		return 0, fmt.Errorf("native command EXP: high-class table unavailable for fig %d", target.BattleFig)
	}
	return int(table.Record[9]), nil
}

// NativeDamageExperience 是 0x1C81F 扣血之後的經驗累加（0x1C8A3..0x1C8E2）：
//
//	守方 +7 < 0x44 → 0（0x1C8AA jl 0x14230）
//	e = 列表[9] × 守方 +0x21               （0x1C8BC..0x1C8C4）
//	守方扣血後 HP 非 0 → e = e × 擲出傷害 / 守方 +0x42（0x1C8CB..0x1C8D7，有號整數除法）
//
// rolled 是 0x1C81F 內 `amount*9/10 + (rng%100)*amount/1000` 的擲出值（未依剩餘 HP 截斷），
// hpAfter 是同一次寫回後的 +0x40。和物理經驗不同，這裡不除以施術者等級。沒有原版
// BattleFig 的正規化單位回 0。
func NativeDamageExperience(target *Unit, rolled, hpAfter int) (int, error) {
	if target == nil {
		return 0, fmt.Errorf("native command EXP: nil target")
	}
	if !target.HasBattleFig || target.BattleFig < 0x44 {
		return 0, nil
	}
	row9, err := nativeHighClassExperienceRow9(target)
	if err != nil {
		return 0, err
	}
	exp := row9 * target.Lv
	if hpAfter != 0 {
		exp = exp * rolled / target.MaxHP
	}
	return exp, nil
}

// NativeRestoreExperience 是 0x1C916 回復 HP 之後的經驗累加（0x1C99C..0x1C9CF）：
// 受術者 +7 < 0x4B 時累加 `40 × L × 實際回復量 / +0x42`。實際回復量是寫回後減寫回前
// （0x1C990..0x1C996）。最大 HP 為 0 時原版會除以零，這裡回 0（同 ApplyNativeRawHPRestore）。
func NativeRestoreExperience(target *Unit, actual int) int {
	if target == nil || !target.HasBattleFig || target.BattleFig >= 0x4B || target.MaxHP == 0 {
		return 0
	}
	return 40 * nativeExperienceLevel(target) * actual / target.MaxHP
}

// NativeCommandDamageExperience 加總 0x2A6BD → 0x1C75E → 0x1C81F（指令 0～12）每個命中
// 目標的經驗；0x1C75E 打空時不呼叫 0x1C81F，所以沒有累加。
func NativeCommandDamageExperience(results []NativeCommandDamageResult) (int, error) {
	total := 0
	for _, result := range results {
		if !result.Hit {
			continue
		}
		exp, err := NativeDamageExperience(result.Target, result.Damage, result.HPAfter)
		if err != nil {
			return 0, err
		}
		total += exp
	}
	return total, nil
}

// NativeCommandHealExperience 加總指令 13～16 每個目標的 0x1C916 經驗。
func NativeCommandHealExperience(results []NativeCommandHealResult) int {
	total := 0
	for _, result := range results {
		total += NativeRestoreExperience(result.Target, result.Restore.Actual)
	}
	return total
}

// NativeCommandDerivedStrikeExperience 加總 0x276EC → 0x1C81F（指令 24、28～31）每個目標
// 的經驗；擲出值用 Rolled（截斷前），不是發布的 Damage。
func NativeCommandDerivedStrikeExperience(results []NativeCommand24Damage) (int, error) {
	total := 0
	for _, result := range results {
		exp, err := NativeDamageExperience(result.Target, result.Rolled, result.HPAfter)
		if err != nil {
			return 0, err
		}
		total += exp
	}
	return total, nil
}

// NativeCommand2022Experience 加總指令 20～22、25～27 尾段的經驗：
//
//   - 20／21 → 0x22AF6：清掉非零標記的目標累加 4 × L（0x22BAC..0x22BAF），同一目標的
//     0x1C916(目標,10) 另累加回復經驗（NativeRawRestoreResult.Score，條件與公式同
//     NativeRestoreExperience）。
//   - 22／26／27 → 0x22D1B：命中的目標先經 0x1C81F(目標,10) 累加傷害經驗，再累加
//     8 × +0x21（0x22E04..0x22E0B，不加 30）。
//   - 25 → 0x22C04：+5 bit7 被清掉的目標累加 8 × L（0x22C83..0x22C9B）。
func NativeCommand2022Experience(plan *NativeAICommand2022Plan) (int, error) {
	if plan == nil {
		return 0, fmt.Errorf("native command EXP: nil command tail plan")
	}
	total := 0
	for _, result := range plan.Results {
		switch {
		case result.Command25:
			if result.ClearedByte5 && result.Target != nil {
				total += 8 * nativeExperienceLevel(result.Target)
			}
		case result.Restore != nil:
			if result.Restore.Applied && result.Target != nil {
				total += result.Restore.Restore.Score + 4*nativeExperienceLevel(result.Target)
			}
		case result.Apply != nil:
			if !result.Apply.Applied {
				continue
			}
			index := result.Apply.TargetIndex
			if index < 0 || index >= len(plan.After) || plan.After[index].Target != result.Target {
				return 0, fmt.Errorf("native command EXP: command tail target state unavailable")
			}
			exp, err := NativeDamageExperience(result.Target, result.Apply.Damage.Rolled, plan.After[index].HP)
			if err != nil {
				return 0, err
			}
			total += exp + 8*result.Target.Lv
		}
	}
	return total, nil
}

// NativeCompoundCommand32Experience：0x2111A → 0x1C75E → 0x1C81F，同指令 0～12。
func NativeCompoundCommand32Experience(result NativeCompoundCommand32Result) (int, error) {
	total := 0
	for _, target := range result.Targets {
		if !target.Damage.Hit {
			continue
		}
		exp, err := NativeDamageExperience(target.Target, target.Damage.Damage, target.HPAfter)
		if err != nil {
			return 0, err
		}
		total += exp
	}
	return total, nil
}

// NativeCompoundCommand33Experience：0x211A4 對每個目標呼叫 0x1C916，累加量是各次的
// NativeRawRestoreResult.Score。
func NativeCompoundCommand33Experience(result NativeCompoundCommand33Result) int {
	total := 0
	for _, restore := range result.Restore.Results {
		total += restore.Score
	}
	return total
}

// NativeCompoundCommand34Experience：0x27FC9 依序呼叫 0x22721／0x22866／0x22997，
// 每段的累加量已在 NativeCommandModifierResult.Accumulator。
func NativeCompoundCommand34Experience(result NativeCompoundCommand34Result) int {
	total := 0
	for _, stage := range result.Stages {
		total += stage.Accumulator
	}
	return total
}

// NativeCompoundCommand35Experience：0x27FC9 呼叫三次 0x22D1B；每段的 8 × +0x21 已在
// Accumulator，命中目標的 0x1C81F(目標,10) 傷害經驗依該段寫回後的 HP 另算。
func NativeCompoundCommand35Experience(result NativeCompoundCommand35Result) (int, error) {
	if len(result.StageStates) != len(result.Stages) {
		return 0, fmt.Errorf("native command EXP: command35 stage states unavailable")
	}
	total := 0
	for stageIndex, stage := range result.Stages {
		total += stage.Accumulator
		states := result.StageStates[stageIndex]
		for _, application := range stage.Results {
			if !application.Applied {
				continue
			}
			index := application.TargetIndex
			if index < 0 || index >= len(states) || states[index].Target == nil {
				return 0, fmt.Errorf("native command EXP: command35 target state unavailable")
			}
			exp, err := NativeDamageExperience(states[index].Target, application.Damage.Rolled, states[index].HP)
			if err != nil {
				return 0, err
			}
			total += exp
		}
	}
	return total, nil
}

// AwardNativePlayerActionExperience 是玩家路徑的行動收尾：把這次行動累加的 delta 加進
// [0x53EC8]，擊倒的單位帶無條件 exp_cancel 時整筆清零（0x34B48，原版在 0x18890 裡
// 分派），超過 99 壓到 99（0x11959），再 0x1E292 發給行動者；有收下才清零（0x1E51A，
// 滿級／死亡早退不抵達）。和 AttackNativePhysicalWithExperience 的玩家分支相同。
// targets 是這次行動影響到的單位，只用來查 exp_cancel。
func (s *State) AwardNativePlayerActionExperience(actor *Unit, delta int, rngState uint16, targets ...*Unit) (int, []LevelUpEvent, uint16) {
	if s == nil || actor == nil {
		return 0, nil, rngState
	}
	s.NativeExperienceAccumulator += delta
	if s.killCancelsExp(targets...) {
		s.NativeExperienceAccumulator = 0
	}
	if s.NativeExperienceAccumulator > 99 {
		s.NativeExperienceAccumulator = 99
	}
	got, ups, next := s.AwardExpNative(actor, s.NativeExperienceAccumulator, rngState)
	if got > 0 {
		s.NativeExperienceAccumulator = 0
	}
	return got, ups, next
}
