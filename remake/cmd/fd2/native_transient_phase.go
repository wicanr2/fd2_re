package main

import (
	"encoding/binary"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// applyNativeTransientPhases owns one or more ordered sub_1A866(selector)
// sweeps as a single publication. Each sweep applies the proven +0x25 HP
// writer, sub_1DB65 inactive mark and raw +0x22..+0x27 countdown. When any
// byte expires, it runs the proven 0x1B750 equipment/derived-stat
// recalculation before publishing the state. This path deliberately has no
// killer and never enters the action-only 0x1B6B7/0x1AA1D dispatch.
func (g *Game) buildNativeTransientPhases(selectors ...byte) (*battle.State, []battle.NativeTransientExpiry, error) {
	if g == nil || g.st == nil || len(selectors) == 0 {
		return nil, nil, fmt.Errorf("native transient phase: state unavailable")
	}
	if !g.st.HasNativeRuntimeUnitProjection {
		return g.buildNativeTransientPhasesTyped(selectors...)
	}
	if len(g.st.Units) != len(g.st.NativeRuntimeRecords) {
		return nil, nil, fmt.Errorf("native transient phase: runtime raw projection is incomplete")
	}
	candidate := *g.st
	candidate.Units = make([]*battle.Unit, len(g.st.Units))
	for index, source := range g.st.Units {
		if source == nil {
			return nil, nil, fmt.Errorf("native transient phase: unit %d is nil", index)
		}
		clone := *source
		candidate.Units[index] = &clone
	}
	candidate.NativeRuntimeRecords = append(
		[]battle.NativeRuntimeRecordState(nil), g.st.NativeRuntimeRecords...,
	)
	selected := make(map[byte]struct{}, len(selectors))
	var expired []battle.NativeTransientExpiry
	for _, selector := range selectors {
		if _, duplicate := selected[selector]; duplicate {
			return nil, nil, fmt.Errorf("native transient phase: duplicate selector %d", selector)
		}
		selected[selector] = struct{}{}
		result, err := candidate.AdvanceNativeTransientPhaseRaw(selector)
		if err != nil {
			return nil, nil, err
		}
		expired = append(expired, result.Expired...)
	}
	expiredUnits := make(map[*battle.Unit]struct{}, len(expired))
	for _, event := range expired {
		expiredUnits[event.Unit] = struct{}{}
	}
	for index, unit := range candidate.Units {
		if !unit.HasNativeRecordByte6 ||
			!unit.HasNativeRecordByte5 || unit.NativeRecordByte5&1 != 0 {
			continue
		}
		if _, included := selected[unit.NativeRecordByte6]; !included {
			continue
		}
		panel, err := battle.NativeItemPanelRecordForUnit(unit)
		if err != nil {
			return nil, nil, fmt.Errorf("native transient phase: unit %d record: %w", index, err)
		}
		record := append([]byte(nil), candidate.NativeRuntimeRecords[index].Raw[:]...)
		copy(record[6:0x28], panel[6:0x28])
		copy(record[0x3b:0x48], panel[0x3b:0x48])
		if _, didExpire := expiredUnits[unit]; didExpire {
			itemRows, err := battle.LoadNativeItemEffectRowPrefix(
				assetPath("assets/data/native_item_effect_rows.json"),
			)
			if err != nil {
				return nil, nil, fmt.Errorf("native transient phase: item rows: %w", err)
			}
			if err := battle.ApplyNativeRuntimeEquipmentRecalc(record, itemRows); err != nil {
				return nil, nil, fmt.Errorf("native transient phase: unit %d recompute: %w", index, err)
			}
			unit.AP = int(int16(binary.LittleEndian.Uint16(record[0x48:])))
			unit.DP = int(int16(binary.LittleEndian.Uint16(record[0x4a:])))
			unit.HIT = int(int16(binary.LittleEndian.Uint16(record[0x4c:])))
			unit.EV = int(int16(binary.LittleEndian.Uint16(record[0x4e:])))
		}
		copy(candidate.NativeRuntimeRecords[index].Raw[:], record)
	}
	return &candidate, expired, nil
}

func (g *Game) applyNativeTransientPhases(selectors ...byte) ([]battle.NativeTransientExpiry, error) {
	candidate, expired, err := g.buildNativeTransientPhases(selectors...)
	if err != nil {
		return nil, err
	}
	*g.st = *candidate
	return expired, nil
}

func (g *Game) applyNativeTransientPhase(selector byte) ([]battle.NativeTransientExpiry, error) {
	return g.applyNativeTransientPhases(selector)
}

// buildNativeTransientPhasesTyped 是從城鎮正常進戰場（沒有 saved runtime raw 投影）
// 時的 sub_1A866 掃描：同樣的 +0x25 扣血、sub_1DB65 標記與 +0x22..+0x27 遞減，
// 到期的單位以 0x1B750 的規則從基礎攻防＋裝備重算 +0x48..+0x4E（強化到期時
// AP 從 1.15 倍回到基礎值；第六章 r4 收據 seq 177：記錄 15 的 +0x22 歸零、AP 93→80）。
// 每個單位都要有 raw +5／+6 與基礎攻防出處，缺了整段拒絕。
func (g *Game) buildNativeTransientPhasesTyped(selectors ...byte) (*battle.State, []battle.NativeTransientExpiry, error) {
	candidate := *g.st
	candidate.Units = make([]*battle.Unit, len(g.st.Units))
	for index, source := range g.st.Units {
		if source == nil {
			return nil, nil, fmt.Errorf("native transient phase: unit %d is nil", index)
		}
		clone := *source
		candidate.Units[index] = &clone
	}
	selected := make(map[byte]struct{}, len(selectors))
	var expired []battle.NativeTransientExpiry
	for _, selector := range selectors {
		if _, duplicate := selected[selector]; duplicate {
			return nil, nil, fmt.Errorf("native transient phase: duplicate selector %d", selector)
		}
		selected[selector] = struct{}{}
		result, err := candidate.AdvanceNativeTransientPhaseTyped(selector)
		if err != nil {
			return nil, nil, err
		}
		expired = append(expired, result.Expired...)
	}
	if len(expired) == 0 {
		return &candidate, nil, nil
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(assetPath("assets/data/native_item_effect_rows.json"))
	if err != nil {
		return nil, nil, fmt.Errorf("native transient phase: item rows: %w", err)
	}
	expiredUnits := make(map[*battle.Unit]struct{}, len(expired))
	for _, event := range expired {
		expiredUnits[event.Unit] = struct{}{}
	}
	for index, unit := range candidate.Units {
		if _, didExpire := expiredUnits[unit]; !didExpire {
			continue
		}
		record, err := campaign.NativeShopEquipmentRecordForUnit(unit)
		if err != nil {
			return nil, nil, fmt.Errorf("native transient phase: unit %d record: %w", index, err)
		}
		if err := battle.ApplyNativeRuntimeEquipmentRecalc(record, itemRows); err != nil {
			return nil, nil, fmt.Errorf("native transient phase: unit %d recompute: %w", index, err)
		}
		unit.AP = int(int16(binary.LittleEndian.Uint16(record[0x48:])))
		unit.DP = int(int16(binary.LittleEndian.Uint16(record[0x4a:])))
		unit.HIT = int(int16(binary.LittleEndian.Uint16(record[0x4c:])))
		unit.EV = int(int16(binary.LittleEndian.Uint16(record[0x4e:])))
	}
	return &candidate, expired, nil
}
