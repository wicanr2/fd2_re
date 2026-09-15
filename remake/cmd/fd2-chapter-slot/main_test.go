package main

import (
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

// 受版控的 30 份戰後 handler 必須各自恰有一個 set_chapter，且目標章 1..30 不重複；
// 建槽工具靠這個對映決定「通關第 k 章要套哪一份」。
func TestPostHandlersMapUniquelyToChapters(t *testing.T) {
	b := &builder{}
	if _, err := b.loadTables("../../assets"); err != nil {
		t.Fatal(err)
	}
	for chapter := 1; chapter <= 30; chapter++ {
		h, ok := b.handlers[chapter]
		if !ok {
			t.Fatalf("第 %d 章沒有戰後 handler", chapter)
		}
		if h.Phase != "post" || h.Chapter != chapter-1 {
			t.Fatalf("第 %d 章對到 %s（phase=%s chapter=%d），檔名慣例是 ch%02d_post", chapter, h.Handler, h.Phase, h.Chapter, chapter-1)
		}
	}
}

// JOIN 建構器輸出要能通過 0x1B750 重算的恆等檢查（建槽工具對基底做同一件事）。
func TestMaterializedJoinRecordIsRecalcStable(t *testing.T) {
	b := &builder{}
	if _, err := b.loadTables("../../assets"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{2, 5, 10, 13} {
		record, err := b.constructor.MaterializePersistentRecord(id, b.itemRows)
		if err != nil {
			t.Fatal(err)
		}
		raw := append([]byte(nil), record.Raw[:]...)
		if err := battle.ApplyNativeEquipmentRecalc(raw, b.itemRows); err != nil {
			t.Fatal(err)
		}
		if !equalBytes(raw[0x48:0x50], record.Raw[0x48:0x50]) {
			t.Fatalf("char %d 建構後重算不恆等：%x vs %x", id, raw[0x48:0x50], record.Raw[0x48:0x50])
		}
	}
}

// 升一級只動 +0x21、+0x37、+0x39、+0x3e、+0x42、+0x46 與重算後的 +0x48..+0x4e；
// 目前 HP／MP 與其他位元組不變。
func TestLevelUpTouchesOnlyGrowthFields(t *testing.T) {
	b := &builder{rng: rand.New(rand.NewSource(7))}
	if _, err := b.loadTables("../../assets"); err != nil {
		t.Fatal(err)
	}
	record, err := b.constructor.MaterializePersistentRecord(2, b.itemRows)
	if err != nil {
		t.Fatal(err)
	}
	b.roster = make([]byte, fdsave.RosterSize)
	copy(b.roster, record.Raw[:])
	b.count = 1
	before := append([]byte(nil), b.record(0)...)
	if !b.levelUp(5, 0) {
		t.Fatal("levelUp 回 false")
	}
	after := b.record(0)
	if after[recordLevel] != before[recordLevel]+1 {
		t.Fatalf("level %d → %d", before[recordLevel], after[recordLevel])
	}
	step := b.steps[0]
	for i, off := range []int{recordBaseAP, recordBaseDP, recordDX, recordMaxHP, recordMaxMP} {
		got := int(int16(binary.LittleEndian.Uint16(after[off:]))) - int(int16(binary.LittleEndian.Uint16(before[off:])))
		if got != step.Gains[i] {
			t.Fatalf("offset %#x 增量 %d，manifest 記 %d", off, got, step.Gains[i])
		}
	}
	for off := 0; off < fdsave.UnitSize; off++ {
		switch {
		case off == recordLevel, off >= recordBaseAP && off < recordBaseAP+4,
			off >= recordDX && off < recordDX+2, off >= recordMaxHP && off < recordMaxHP+2,
			off >= recordMaxMP && off < recordMaxMP+2, off >= 0x48 && off < 0x50,
			off >= recordCommandMask && off < recordCommandMask+5:
			continue
		}
		if after[off] != before[off] {
			t.Fatalf("offset %#x 不該被升級改動：%#x → %#x", off, before[off], after[off])
		}
	}
	if !equalBytes(after[recordHP:recordHP+2], before[recordHP:recordHP+2]) {
		t.Fatal("升級不該改目前 HP")
	}
}
