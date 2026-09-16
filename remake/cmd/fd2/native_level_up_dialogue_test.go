package main

import (
	"fmt"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// 假的 FDTXT_000：只放 0x1E292 會用到的幾句，控制碼照原版（0x1E8 只有 FFFA；0x1E9 尾端
// FFFD；0x1EA..0x1EE FFFA＋FFFD；0x24B FFFC；0x1B9+id 是指令名）。
type fakeLevelUpStrings map[int][]uint16

func (f fakeLevelUpStrings) Words(index int) ([]uint16, error) {
	words, ok := f[index]
	if !ok {
		return nil, fmt.Errorf("no string %#x", index)
	}
	return append([]uint16(nil), words...), nil
}

func levelUpFixtureStrings() fakeLevelUpStrings {
	return fakeLevelUpStrings{
		0x1e8:     {100, 0xfffa, 101},
		0x1e9:     {102, 0xfffd},
		0x1ea:     {110, 0xfffa, 111, 0xfffd},
		0x1eb:     {112, 0xfffa, 111, 0xfffd},
		0x1ec:     {113, 0xfffa, 111, 0xfffd},
		0x1ed:     {114, 0xfffa, 111, 0xfffd},
		0x1ee:     {115, 0xfffa, 111, 0xfffd},
		0x24b:     {120, 0xfffc, 121},
		0x1b9 + 5: {130, 131},
	}
}

func TestNativeLevelUpDialogueLinesSplitPagesAtFFFDAndSkipZeroGrowth(t *testing.T) {
	pages, lastWait, err := nativeLevelUpDialogueLines(levelUpFixtureStrings(), pendingNativeLevelUp{
		exp: 16, ups: []battle.LevelUpEvent{{NewLv: 5, ApGain: 7, DpGain: 4, DxGain: 0, HpGain: 10, MpGain: 0, LearnedCommandIDs: []int{5}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// r4 seq 968..972：第一頁「得到經驗16點！／等級上升了！！」，之後每欄一頁；速度與 MMP 為零不寫；
	// 「學會了」在最後一欄的 FFFD 之後寫、本身沒有 FFFD，所以是最後一頁且不等鍵。
	want := [][][]uint16{
		{{100, 1, 6, 101}, {102}},
		{{110, 7, 111}},
		{{112, 4, 111}},
		{{114, 1, 0, 111}},
		{{120, 130, 131, 121}},
	}
	if fmt.Sprint(pages) != fmt.Sprint(want) || lastWait {
		t.Fatalf("pages=%v lastWait=%v\nwant %v", pages, lastWait, want)
	}
	pages, lastWait, err = nativeLevelUpDialogueLines(levelUpFixtureStrings(), pendingNativeLevelUp{
		exp: 16, ups: []battle.LevelUpEvent{{NewLv: 5, ApGain: 7}},
	})
	if err != nil || len(pages) != 2 || !lastWait {
		t.Fatalf("pages=%v lastWait=%v err=%v（最後一欄以 FFFD 收尾要等鍵）", pages, lastWait, err)
	}
}

func TestNativeLevelUpDialogueLinesWithoutLevelUpIsOneUnwaitedPage(t *testing.T) {
	pages, lastWait, err := nativeLevelUpDialogueLines(levelUpFixtureStrings(), pendingNativeLevelUp{exp: 9})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(pages) != fmt.Sprint([][][]uint16{{{100, 9, 101}}}) || lastWait {
		t.Fatalf("pages=%v lastWait=%v", pages, lastWait)
	}
}

func TestQueueNativeLevelUpDialogueOnlyForRawPlayerRecords(t *testing.T) {
	g := &Game{}
	enemy := &battle.Unit{HasNativeRecordByte6: true, NativeRecordByte6: 0}
	player := &battle.Unit{HasNativeRecordByte6: true, NativeRecordByte6: 2}
	g.queueNativeLevelUpDialogue(enemy, 12, nil)
	g.queueNativeLevelUpDialogue(player, 0, nil)
	g.queueNativeLevelUpDialogue(player, 12, nil)
	if len(g.pendingNativeLevelUps) != 1 || g.pendingNativeLevelUps[0].unit != player || g.pendingNativeLevelUps[0].exp != 12 {
		t.Fatalf("pending=%+v", g.pendingNativeLevelUps)
	}
}
