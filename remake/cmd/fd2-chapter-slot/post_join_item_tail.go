package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// 只保存戰後新登場列的尾兩格；不模擬該角色的戰鬥或完整 0x11506。
type postSpawnItemTail struct {
	rawByte8 byte
	cells    [4]byte
}

func (b *builder) capturePostSpawnItemTail(chapter, group int) error {
	handler, ok := b.handlers[chapter]
	if !ok || b.assetsDir == "" || group < 0 || group > 0xff {
		return fmt.Errorf("第 %d 章 spawn group=%d 缺少已驗證的地圖來源", chapter, group)
	}
	relative := filepath.Join("maps", fmt.Sprintf("map%d", handler.Chapter), fmt.Sprintf("map%d_units.json", handler.Chapter))
	path := filepath.Join(b.assetsDir, relative)
	state, err := battle.Load(path)
	if err != nil {
		return fmt.Errorf("第 %d 章 spawn 物品來源：%w", chapter, err)
	}
	staged := []postSpawnItemTail{}
	for _, unit := range state.Units {
		if unit.Group != group {
			continue
		}
		if !unit.HasNativeRecordByte8 {
			return fmt.Errorf("第 %d 章 group=%d 缺少 raw +8，不能以肖像代替", chapter, group)
		}
		records, err := battle.NativeInventoryRecords([]*battle.Unit{unit}, 1)
		if err != nil {
			return fmt.Errorf("第 %d 章 group=%d 物品來源：%w", chapter, group, err)
		}
		row := postSpawnItemTail{rawByte8: unit.NativeRecordByte8}
		copy(row.cells[:], records[0x16:0x1a])
		staged = append(staged, row)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if b.sourceRefs != nil {
		b.sourceRefs[filepath.ToSlash(relative)] = sha(raw)
	}
	b.postSpawn = append(b.postSpawn, staged...)
	return nil
}

func (b *builder) syncPostJoinedItemTail(chapter int, source any) error {
	for _, row := range b.postSpawn {
		for slot := 0; slot < b.count; slot++ {
			if !b.postJoined[slot] || b.record(slot)[recordIdentity] != row.rawByte8 {
				continue
			}
			if row.rawByte8 == 0 {
				return fmt.Errorf("第 %d 章 identity0 同步需要原始 bit0 gate，不能推論", chapter)
			}
		}
	}
	for _, row := range b.postSpawn {
		for slot := 0; slot < b.count; slot++ {
			if !b.postJoined[slot] || b.record(slot)[recordIdentity] != row.rawByte8 {
				continue
			}
			// 0x11576 複製的尾格投影；其他欄位維持建槽政策，不宣稱整筆 copy。
			copy(b.record(slot)[0x16:0x1a], row.cells[:])
			id, index := int(row.rawByte8), slot
			b.applied = append(b.applied, appliedOp{Chapter: chapter, Op: "sync_join_item_tail",
				CharID: &id, RosterSlot: &index, Source: source, Result: "projected_four_bytes"})
		}
	}
	return nil
}
