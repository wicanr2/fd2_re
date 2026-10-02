package main

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

// saveNativeChapterSlot 是酒店存檔（0x30012）的原版槽寫回：以標題 LOAD 讀進來的
// FD2.SAV plaintext 為底，把目前持續隊伍與金幣寫進 slot，重算校驗和後原子取代
// FD2_NATIVE_SAVE。寫成功後同一份 plaintext 成為下一次存檔的基底。
func (g *Game) saveNativeChapterSlot(slot int) error {
	if g == nil || g.nativeChapterSlotBaseline == nil || len(g.nativeChapterSlotPlain) != fdsave.FileSize {
		return errors.New("原版章節槽存檔：缺少四槽 LOAD 的 raw 基底")
	}
	path := nativeCurrentSavePath()
	if path == "" {
		return errors.New("原版章節槽存檔：找不到 FD2.SAV 路徑")
	}
	if g.handlerChapter < 0 || g.handlerChapter > 0xfe || g.gold < 0 || uint64(g.gold) > uint64(^uint32(0)) {
		return errors.New("原版章節槽存檔：章節或金幣超出 metadata 範圍")
	}
	joinTable, err := campaign.LoadNativeJoinConstructorTable(assetPath("assets/data/native_join_constructor.json"))
	if err != nil {
		return fmt.Errorf("原版章節槽存檔：JOIN constructor：%w", err)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(assetPath("assets/data/native_item_effect_rows.json"))
	if err != nil {
		return fmt.Errorf("原版章節槽存檔：item rows：%w", err)
	}
	built, err := campaign.BuildNativeChapterSlot(
		*g.nativeChapterSlotBaseline, g.partyRoster, g.partyJoinOrder,
		byte(g.handlerChapter), uint32(g.gold), joinTable, itemRows,
	)
	if err != nil {
		return fmt.Errorf("原版章節槽存檔：%w", err)
	}
	plain, err := fdsave.WriteSlot(g.nativeChapterSlotPlain, slot, built)
	if err != nil {
		return fmt.Errorf("原版章節槽存檔：%w", err)
	}
	stored, err := fdsave.Encode(plain)
	if err != nil {
		return fmt.Errorf("原版章節槽存檔：%w", err)
	}
	if err := replaceNativeCurrentSaveAtomic(path, stored); err != nil {
		return err
	}
	snapshot, err := fdsave.InspectChapterSlot(plain, slot)
	if err != nil {
		return fmt.Errorf("原版章節槽存檔：回讀：%w", err)
	}
	g.nativeChapterSlotPlain = plain
	g.nativeChapterSlotBaseline = &snapshot
	return nil
}

// packNativePreparationRoster models 0x31A74→0x320FC before either the
// required-character rejection or the final YES/NO owner. Record zero stays
// fixed; both selected and unselected partitions preserve their prior order.
// partyJoinOrder retains its historical identifier, but native selection can
// reorder this persistent sequence. No membership or record bytes are lost.
func (g *Game) packNativePreparationRoster() error {
	if len(g.partyJoinOrder) == 0 {
		return errors.New("preparation pack: persistent order unavailable")
	}
	order := []int{g.partyJoinOrder[0]}
	for _, selected := range []bool{true, false} {
		for _, id := range g.partyJoinOrder[1:] {
			if g.partyDeploy[id] == selected {
				order = append(order, id)
			}
		}
	}
	if g.nativeChapterSlotBaseline == nil {
		g.partyJoinOrder = order
		return nil
	}
	// Materialize any JOIN since LOAD through the existing constructor before
	// copying whole records. Publish only after the complete candidate validates.
	table, err := campaign.LoadNativeJoinConstructorTable(assetPath("assets/data/native_join_constructor.json"))
	if err != nil {
		return fmt.Errorf("preparation pack: JOIN constructor: %w", err)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(assetPath("assets/data/native_item_effect_rows.json"))
	if err != nil {
		return fmt.Errorf("preparation pack: item rows: %w", err)
	}
	built, err := campaign.BuildNativeChapterSlot(*g.nativeChapterSlotBaseline,
		g.partyRoster, g.partyJoinOrder, byte(g.handlerChapter), uint32(g.gold), table, itemRows)
	if err != nil {
		return fmt.Errorf("preparation pack: records: %w", err)
	}
	byIdentity := make(map[int][]byte, len(order))
	for index := 0; index < int(built.Metadata[1]); index++ {
		record := built.Roster[index*fdsave.UnitSize : (index+1)*fdsave.UnitSize]
		identity := int(record[8])
		if _, duplicate := byIdentity[identity]; duplicate {
			return fmt.Errorf("preparation pack: duplicate identity %d", identity)
		}
		byIdentity[identity] = append([]byte(nil), record...)
	}
	if len(byIdentity) != len(order) {
		return errors.New("preparation pack: persistent topology differs")
	}
	for index, id := range order {
		unit, exists := g.partyRoster[id]
		if !exists || !unit.HasNativeIdentity {
			return fmt.Errorf("preparation pack: member %d lacks identity", id)
		}
		record, exists := byIdentity[unit.NativeIdentity]
		if !exists {
			return fmt.Errorf("preparation pack: record identity %d unavailable", unit.NativeIdentity)
		}
		copy(built.Roster[index*fdsave.UnitSize:(index+1)*fdsave.UnitSize], record)
	}
	plain, err := fdsave.WriteSlot(g.nativeChapterSlotPlain, g.nativeChapterSlotBaseline.Slot, built)
	if err != nil {
		return fmt.Errorf("preparation pack: candidate slot: %w", err)
	}
	snapshot, err := fdsave.InspectChapterSlot(plain, g.nativeChapterSlotBaseline.Slot)
	if err != nil {
		return fmt.Errorf("preparation pack: candidate inspection: %w", err)
	}
	g.nativeChapterSlotBaseline = &snapshot
	g.partyJoinOrder = order
	return nil
}
