package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// TestNativeStoryAdvanceAcceptsEscapeLikeEnter 釘住原版故事等待的按鍵契約。
//
// 收據 docs/data/ui-traces/fd2-opening-esc-20260909.json：同一份控制序列只把
// Enter 換成 ESC，19 個控制邊界的畫面 PNG 逐格 bit-identical；完全不送鍵的
// 第三份收據則從第 8 個邊界起畫面停住不動。ESC 在原版是與 Enter 同義的推進
// 鍵，不是跳過整段的捷徑。
func TestNativeStoryAdvanceAcceptsEscapeLikeEnter(t *testing.T) {
	build := func() *Game {
		c := &campaign.Campaign{Start: "prologue", Nodes: map[string]*campaign.Node{
			"prologue": {Type: "story", Next: "after", Lines: []campaign.Line{
				{Speaker: 0, Text: "第一句"},
				{Speaker: 0, Text: "第二句"},
			}},
			"after": {Type: "ending", Text: "done"},
		}}
		g := &Game{localeID: "zh-Hant", camp: campaign.NewRunner(c)}
		g.enterNode()
		if g.loadErr != "" || len(g.dialog) != 2 {
			t.Fatalf("序幕節點未備妥兩句對白：err=%q dialog=%d", g.loadErr, len(g.dialog))
		}
		return g
	}

	type observation struct {
		node    string
		dialogs int
	}
	run := func(input nativeStoryInput) []observation {
		g := build()
		seen := []observation{}
		for i := 0; i < 2; i++ {
			if !g.handleNativeStoryInput(g.camp.Node(), input) {
				t.Fatalf("story 節點必須擁有輸入（第 %d 次）", i)
			}
			// 最後一句消費完之後 advanceStoryNode 會先起淡出，節點在淡出跑完
			// 才換；不推進這幾幀會把「已推進」誤讀成「沒反應」。
			g.tick(storyFadeFrames)
			seen = append(seen, observation{node: g.camp.NodeID(), dialogs: len(g.dialog)})
		}
		return seen
	}

	enter := run(nativeStoryInput{enter: true})
	escape := run(nativeStoryInput{escape: true})
	silent := run(nativeStoryInput{})

	want := []observation{{node: "prologue", dialogs: 1}, {node: "after", dialogs: 0}}
	for name, got := range map[string][]observation{"enter": enter, "escape": escape} {
		for i, w := range want {
			if got[i] != w {
				t.Fatalf("%s 第 %d 次推進 = %+v，應為 %+v", name, i, got[i], w)
			}
		}
	}
	for i := range silent {
		if (silent[i] != observation{node: "prologue", dialogs: 2}) {
			t.Fatalf("不送鍵時第 %d 次 = %+v，原版對應收據是畫面停住", i, silent[i])
		}
	}
}
