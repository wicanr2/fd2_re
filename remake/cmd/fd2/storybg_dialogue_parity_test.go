package main

import (
	"path/filepath"
	"testing"
)

// TestStoryBGFirstDialogueEnterAdvancesToKing 對應
// docs/data/ui-traces/storybg-dialogue-original-vs-remake-e1.json：原版 normal
// START 第一段 storyBG 對白按 Enter 後，焦點由索爾 (8,21) 移至國王 (7,5)，
// 鏡頭格座標由 (3,20) 移至 (3,4)。這裡從完整戰役的正式 runtime 起點重播，
// 並透過 production 輸入 owner 推進，不直接改 beat 或 camera。
func TestStoryBGFirstDialogueEnterAdvancesToKing(t *testing.T) {
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	t.Setenv("FD2_ASSET_PACK", filepath.Join("..", "..", "generated-assets", "fd2-original-b97caf22"))
	g := loadGame()
	if g.loadErr != "" {
		t.Fatalf("載入完整戰役：%v", g.loadErr)
	}

	for frame := 0; frame < 2000 && !storyEnterReady(g); frame++ {
		if err := g.Update(); err != nil {
			t.Fatalf("第一段對白 Update：%v", err)
		}
	}
	if !storyEnterReady(g) || g.camp.NodeID() != "story_ch00_handler" || g.beatIdx != 4 ||
		g.camX/24 != 3 || g.camY/24 != 20 || g.curX != 8 || g.curY != 21 {
		t.Fatalf("未抵達第一段同狀態：node=%q beat=%d camera=(%.0f,%.0f) cursor=(%d,%d) ready=%v err=%q",
			g.camp.NodeID(), g.beatIdx, g.camX/24, g.camY/24, g.curX, g.curY, storyEnterReady(g), g.loadErr)
	}
	if !g.handleNativeStoryInput(g.camp.Node(), nativeStoryInput{enter: true}) {
		t.Fatal("production story 輸入 owner 未接收 Enter")
	}

	for frame := 0; frame < 2000; frame++ {
		if err := g.Update(); err != nil {
			t.Fatalf("下一段對白 Update：%v", err)
		}
		if storyEnterReady(g) && g.curX == 7 && g.curY == 5 && g.camX/24 == 3 && g.camY/24 == 4 {
			return
		}
	}
	t.Fatalf("Enter 後未抵達國王對白：node=%q beat=%d camera=(%.0f,%.0f) cursor=(%d,%d) ready=%v err=%q",
		g.camp.NodeID(), g.beatIdx, g.camX/24, g.camY/24, g.curX, g.curY, storyEnterReady(g), g.loadErr)
}
