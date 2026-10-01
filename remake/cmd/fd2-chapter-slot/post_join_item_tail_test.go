package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

func joinTailBuilder(t *testing.T) *builder {
	t.Helper()
	b := &builder{roster: make([]byte, fdsave.RosterSize), count: 1}
	if _, err := b.loadTables("../../assets"); err != nil {
		t.Fatal(err)
	}
	for i := range b.roster {
		b.roster[i] = 0xf8
	}
	initial, err := b.constructor.MaterializePersistentRecord(0, b.itemRows)
	if err != nil {
		t.Fatal(err)
	}
	copy(b.record(0), initial.Raw[:])
	return b
}

// 使用同一正式 battle.Load 邊界，但讓尾格來源可變；不能只測恰好 ff 的原版列。
func joinTailSource(t *testing.T, b *builder, identity any, tail []int) {
	t.Helper()
	raw, err := os.ReadFile("../../assets/maps/map1/map1_units.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	unit := doc["units"].([]any)[22].(map[string]any)
	delete(unit, "native_record_byte8")
	if identity != nil {
		unit["native_record_byte8"] = identity
	}
	unit["inventory_slots"] = append([]int{44, 129, 255, 255, 255, 255}, tail...)
	doc["units"] = []any{unit}
	fixture, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	b.assetsDir = t.TempDir()
	dir := filepath.Join(b.assetsDir, "maps", "map1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "map1_units.json"), fixture, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPostJoinedItemTailRequiresRawIdentityAndSync(t *testing.T) {
	for _, tc := range []struct {
		name     string
		identity int
		tail     []int
		want     []byte
	}{
		{"original_ff", 8, []int{255, 255}, []byte{0x80, 255, 0x80, 255}},
		{"present_items", 8, []int{1, 2}, []byte{0, 1, 0, 2}},
		{"mixed", 8, []int{255, 3}, []byte{0x80, 255, 0, 3}},
		{"same_portrait_different_raw_identity", 9, []int{255, 255}, []byte{0x80, 0xf8, 0x80, 0xf8}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := joinTailBuilder(t)
			joinTailSource(t, b, tc.identity, tc.tail)
			if err := b.capturePostSpawnItemTail(2, 4); err != nil {
				t.Fatal(err)
			}
			if err := b.syncPostJoinedItemTail(2, nil); err != nil {
				t.Fatal(err)
			}
			if err := b.join(2, 8, nil); err != nil {
				t.Fatal(err)
			}
			before := append([]byte(nil), b.roster...)
			if got := b.record(1)[0x17]; got != 0xf8 {
				t.Fatalf("同步在 JOIN 之前，不能提前投影：%#x", got)
			}
			if err := b.syncPostJoinedItemTail(2, nil); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(b.record(1)[0x16:0x1a], tc.want) {
				t.Fatalf("尾格=% x，來源期望=% x", b.record(1)[0x16:0x1a], tc.want)
			}
			for i := range b.roster {
				if i >= fdsave.UnitSize+0x16 && i < fdsave.UnitSize+0x1a {
					continue
				}
				if b.roster[i] != before[i] {
					t.Fatalf("投影改寫範圍外 byte %#x", i)
				}
			}
			if b.sourceRefs["maps/map1/map1_units.json"] == "" {
				t.Fatal("manifest 漏掉場上物品來源雜湊")
			}
		})
	}
}

func TestPostJoinedItemTailLeavesExistingMemberAndClearsBetweenChapters(t *testing.T) {
	b := joinTailBuilder(t)
	if err := b.join(2, 8, nil); err != nil {
		t.Fatal(err)
	}
	b.postJoined = nil // identity8 是既有成員，不是這一章的新 JOIN。
	if err := b.capturePostSpawnItemTail(2, 4); err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), b.roster...)
	if err := b.syncPostJoinedItemTail(2, nil); err != nil || !bytes.Equal(b.roster, before) {
		t.Fatalf("既有成員不應套新登場投影：%v", err)
	}
	b.postSpawn = []postSpawnItemTail{{rawByte8: 12, cells: [4]byte{0, 1, 0, 2}}}
	b.eventStates = map[[2]int]int{{7, 17}: 1}
	if err := b.applyChapter(7); err != nil {
		t.Fatal(err)
	}
	if b.count != 3 || b.record(2)[recordIdentity] != 12 || b.record(2)[0x17] != 0xf8 || b.record(2)[0x19] != 0xf8 {
		t.Fatalf("上章來源被錯套到第七章 JOIN12：% x", b.record(2))
	}
}

func TestPostJoinedItemTailForJoinBeforeSpawn(t *testing.T) {
	// 第六章的 JOIN13 在 spawn3 之前；到共同 sync 尾端才抄回新登場列。
	b := joinTailBuilder(t)
	if err := b.applyChapter(6); err != nil {
		t.Fatal(err)
	}
	if b.count != 2 || b.record(1)[recordIdentity] != 13 ||
		!bytes.Equal(b.record(1)[0x16:0x1a], []byte{0x80, 255, 0x80, 255}) {
		t.Fatalf("JOIN→spawn→sync 沒有接回尾格來源：% x", b.record(1))
	}
}

func TestPostJoinedItemTailRejectsMissingOrMalformedSourceAndLaterGrant(t *testing.T) {
	for _, tc := range []struct {
		identity any
		tail     []int
	}{{nil, []int{255, 255}}, {8, []int{256, 255}}, {8, []int{255}}} {
		b := joinTailBuilder(t)
		joinTailSource(t, b, tc.identity, tc.tail)
		if err := b.capturePostSpawnItemTail(2, 4); err == nil || len(b.postSpawn) != 0 {
			t.Fatal("缺證據的來源被接受或部分發布")
		}
	}
	b := joinTailBuilder(t)
	b.assetsDir = t.TempDir()
	if err := b.capturePostSpawnItemTail(2, 4); err == nil {
		t.Fatal("缺少 map 來源仍通過")
	}
	b = joinTailBuilder(t)
	if err := b.capturePostSpawnItemTail(2, -1); err == nil {
		t.Fatal("越界 group 被接受")
	}
	if err := b.capturePostSpawnItemTail(2, 4); err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), b.roster...)
	if err := b.applyBeats(2, []map[string]any{{"op": "grant_item", "item_id": float64(1)}}); err == nil || !bytes.Equal(b.roster, before) {
		t.Fatal("新登場後未驗證的戰場物品異動沒有拒絕")
	}
}
