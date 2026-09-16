package battle

import (
	"encoding/binary"
	"fmt"
)

// NativeTransientOffset is the first byte of the original six-byte transient
// interval at runtime unit+0x22..+0x27.  Callers use raw offsets deliberately:
// only some effects have a recovered gameplay label, and the data layer must
// not invent one for the rest.
const (
	NativeTransientOffset = 0x22
	NativeTransientCount  = 6
)

// NativeTransientExpiry records one original raw interval byte which reached
// zero during the recovered 0x1A866 camp-phase sweep.  Recalculation of
// derived stats/render feedback is intentionally owned by higher layers;
// battle cannot import campaign's equipment table without creating a cycle.
type NativeTransientExpiry struct {
	Unit   *Unit
	Offset int
}

// NativeTransientDamage 保存 sub_1A866 第一段對 raw +0x25 記錄的逐筆 HP writer。
// Amount 即 MaxHP/10；即使 Amount 為 0，原版仍會走該筆回覆，因此照樣保留事件。
type NativeTransientDamage struct {
	Unit                  *Unit
	Slot                  int
	Amount, Before, After int
}

// NativeTransientPhaseResult 是單次 sub_1A866(selector) 的純狀態結果。它不帶
// killer，也不分派死亡效果；該 phase 的原版 ABI 根本沒有這兩項輸入。
type NativeTransientPhaseResult struct {
	Damage        []NativeTransientDamage
	InactiveSlots []int
	Expired       []NativeTransientExpiry
}

// NativeTransientDuration returns a raw duration byte by its original unit
// offset.  It rejects anything outside +0x22..+0x27 rather than silently
// mapping a guessed status field.
func (u *Unit) NativeTransientDuration(offset int) (byte, bool) {
	if u == nil || offset < NativeTransientOffset || offset >= NativeTransientOffset+NativeTransientCount {
		return 0, false
	}
	return u.NativeTransient[offset-NativeTransientOffset], true
}

// SetNativeTransientDuration writes one recovered raw duration byte.  It is a
// bounded storage primitive; command-specific gates and side effects remain
// in their separately recovered executors.
func (u *Unit) SetNativeTransientDuration(offset int, duration byte) bool {
	if u == nil || offset < NativeTransientOffset || offset >= NativeTransientOffset+NativeTransientCount {
		return false
	}
	u.NativeTransient[offset-NativeTransientOffset] = duration
	return true
}

// TickNativeTransientsRaw mirrors the recovered raw mutation portion of
// 0x1A866. The native routine gates on record+6 == selector and
// (record+5 & 1) == 0; it does not prove an OnField/Alive/Camp equivalence.
// Every nonzero byte +0x22..+0x27 is decremented independently.
func (s *State) TickNativeTransientsRaw(selector byte) []NativeTransientExpiry {
	if s == nil {
		return nil
	}
	var expired []NativeTransientExpiry
	for _, u := range s.Units {
		if u == nil || !u.HasNativeRecordByte6 || u.NativeRecordByte6 != selector ||
			!u.HasNativeRecordByte5 || u.NativeRecordByte5&1 != 0 {
			continue
		}
		for i, duration := range u.NativeTransient {
			if duration == 0 {
				continue
			}
			u.NativeTransient[i]--
			if u.NativeTransient[i] == 0 {
				expired = append(expired, NativeTransientExpiry{Unit: u, Offset: NativeTransientOffset + i})
			}
		}
	}
	return expired
}

// AdvanceNativeTransientPhaseRaw 重現 sub_1A866 的狀態寫入順序：先依 record 順序
// 對 +0x25 非零的同 selector 單位扣 MaxHP/10，再執行 sub_1DB65 的 HP==0 →
// raw +5=1，最後只替仍 active 的單位遞減 +0x22..+0x27。typed 與完整 raw
// projection 在同一個私有 State 中同步；任何矛盾都在 mutation 前拒絕。
func (s *State) AdvanceNativeTransientPhaseRaw(selector byte) (NativeTransientPhaseResult, error) {
	if s == nil || !s.HasNativeRuntimeUnitProjection || len(s.Units) == 0 ||
		len(s.Units) != len(s.NativeRuntimeRecords) {
		return NativeTransientPhaseResult{}, fmt.Errorf("native transient phase: runtime raw projection is incomplete")
	}
	for slot, unit := range s.Units {
		if unit == nil || !unit.HasNativeRecordByte5 || !unit.HasNativeRecordByte6 ||
			unit.HP < 0 || unit.HP > 0xffff || unit.MaxHP < 0 || unit.MaxHP > 0xffff {
			return NativeTransientPhaseResult{}, fmt.Errorf("native transient phase: unit %d lacks consumed raw fields", slot)
		}
		raw := s.NativeRuntimeRecords[slot].Raw
		if raw[5] != unit.NativeRecordByte5 || raw[6] != unit.NativeRecordByte6 ||
			binary.LittleEndian.Uint16(raw[0x40:0x42]) != uint16(unit.HP) ||
			binary.LittleEndian.Uint16(raw[0x42:0x44]) != uint16(unit.MaxHP) {
			return NativeTransientPhaseResult{}, fmt.Errorf("native transient phase: unit %d raw projection disagrees", slot)
		}
		for index, duration := range unit.NativeTransient {
			if raw[NativeTransientOffset+index] != duration {
				return NativeTransientPhaseResult{}, fmt.Errorf("native transient phase: unit %d transient projection disagrees", slot)
			}
		}
	}

	result := NativeTransientPhaseResult{}
	for slot, unit := range s.Units {
		if unit.NativeTransient[0x25-NativeTransientOffset] == 0 ||
			unit.NativeRecordByte6 != selector || unit.NativeRecordByte5&1 != 0 {
			continue
		}
		amount, before := unit.MaxHP/10, unit.HP
		after := before - amount
		if after < 0 {
			after = 0
		}
		unit.HP = after
		binary.LittleEndian.PutUint16(s.NativeRuntimeRecords[slot].Raw[0x40:0x42], uint16(after))
		result.Damage = append(result.Damage, NativeTransientDamage{
			Unit: unit, Slot: slot, Amount: amount, Before: before, After: after,
		})
	}

	// sub_1DB65 的兩個收尾 writer 都掃全部記錄並把整個 +5 byte 覆寫成 1；
	// 不只處理這次 selector，也不保留 bit7 等舊值。
	for slot, unit := range s.Units {
		if unit.HP != 0 {
			continue
		}
		if unit.NativeRecordByte5&1 == 0 {
			result.InactiveSlots = append(result.InactiveSlots, slot)
		}
		unit.NativeRecordByte5 = 1
		s.NativeRuntimeRecords[slot].Raw[5] = 1
	}

	for slot, unit := range s.Units {
		if unit.NativeRecordByte6 != selector || unit.NativeRecordByte5&1 != 0 {
			continue
		}
		for index, duration := range unit.NativeTransient {
			if duration == 0 {
				continue
			}
			unit.NativeTransient[index]--
			s.NativeRuntimeRecords[slot].Raw[NativeTransientOffset+index]--
			if unit.NativeTransient[index] == 0 {
				result.Expired = append(result.Expired, NativeTransientExpiry{
					Unit: unit, Offset: NativeTransientOffset + index,
				})
			}
		}
	}
	return result, nil
}

// TickNativeTransients is retained for source compatibility only. Camp is a
// normalized remake enum, not the raw selector passed as 0x1A866's argument;
// mapping one to the other would reintroduce the withdrawn assertion. Callers
// must provide the recovered raw selector through TickNativeTransientsRaw.
func (s *State) TickNativeTransients(_ Camp) []NativeTransientExpiry { return nil }

// AdvanceNativeTransientPhaseTyped 是 AdvanceNativeTransientPhaseRaw 在沒有 saved
// runtime raw 投影（從城鎮正常進戰場）時的版本：同一個 sub_1A866(selector) 的三段
// 順序（+0x25 扣 MaxHP/10 → sub_1DB65 標記 HP==0 → 遞減 +0x22..+0x27），只寫 typed
// 欄位。每個單位都要有 raw +5／+6 出處，缺了整段拒絕，不做部分掃描。
func (s *State) AdvanceNativeTransientPhaseTyped(selector byte) (NativeTransientPhaseResult, error) {
	if s == nil || len(s.Units) == 0 {
		return NativeTransientPhaseResult{}, fmt.Errorf("native transient phase: roster is empty")
	}
	for slot, unit := range s.Units {
		if unit == nil || !unit.HasNativeRecordByte5 || !unit.HasNativeRecordByte6 ||
			unit.HP < 0 || unit.HP > 0xffff || unit.MaxHP < 0 || unit.MaxHP > 0xffff {
			return NativeTransientPhaseResult{}, fmt.Errorf("native transient phase: unit %d lacks consumed raw fields", slot)
		}
	}
	result := NativeTransientPhaseResult{}
	for slot, unit := range s.Units {
		if unit.NativeTransient[0x25-NativeTransientOffset] == 0 ||
			unit.NativeRecordByte6 != selector || unit.NativeRecordByte5&1 != 0 {
			continue
		}
		amount, before := unit.MaxHP/10, unit.HP
		after := before - amount
		if after < 0 {
			after = 0
		}
		unit.HP = after
		result.Damage = append(result.Damage, NativeTransientDamage{
			Unit: unit, Slot: slot, Amount: amount, Before: before, After: after,
		})
	}
	for slot, unit := range s.Units {
		if unit.HP != 0 {
			continue
		}
		if unit.NativeRecordByte5&1 == 0 {
			result.InactiveSlots = append(result.InactiveSlots, slot)
		}
		unit.NativeRecordByte5 = 1
	}
	for _, unit := range s.Units {
		if unit.NativeRecordByte6 != selector || unit.NativeRecordByte5&1 != 0 {
			continue
		}
		for index, duration := range unit.NativeTransient {
			if duration == 0 {
				continue
			}
			unit.NativeTransient[index]--
			if unit.NativeTransient[index] == 0 {
				result.Expired = append(result.Expired, NativeTransientExpiry{
					Unit: unit, Offset: NativeTransientOffset + index,
				})
			}
		}
	}
	return result, nil
}
