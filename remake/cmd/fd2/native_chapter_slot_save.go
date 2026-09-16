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
