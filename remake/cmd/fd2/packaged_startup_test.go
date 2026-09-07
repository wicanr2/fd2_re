package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyPackagedPlayerDefaults(t *testing.T) {
	t.Run("未設定時載入完整戰役", func(t *testing.T) {
		old, existed := os.LookupEnv("FD2_CAMPAIGN")
		_ = os.Unsetenv("FD2_CAMPAIGN")
		t.Cleanup(func() {
			if existed {
				_ = os.Setenv("FD2_CAMPAIGN", old)
			} else {
				_ = os.Unsetenv("FD2_CAMPAIGN")
			}
		})
		applyPackagedPlayerDefaults()
		if got := os.Getenv("FD2_CAMPAIGN"); got != defaultPlayerCampaign {
			t.Fatalf("FD2_CAMPAIGN=%q，應為 %q", got, defaultPlayerCampaign)
		}
	})

	for _, value := range []string{"", "assets/scenarios/custom.json"} {
		value := value
		t.Run("保留明確設定_"+value, func(t *testing.T) {
			t.Setenv("FD2_CAMPAIGN", value)
			applyPackagedPlayerDefaults()
			if got := os.Getenv("FD2_CAMPAIGN"); got != value {
				t.Fatalf("FD2_CAMPAIGN=%q，應保留 %q", got, value)
			}
		})
	}
}

func TestNormalPackagedStartupOwnsTitleAndChapterZero(t *testing.T) {
	old, existed := os.LookupEnv("FD2_CAMPAIGN")
	_ = os.Unsetenv("FD2_CAMPAIGN")
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv("FD2_CAMPAIGN", old)
		} else {
			_ = os.Unsetenv("FD2_CAMPAIGN")
		}
	})
	t.Setenv("FD2_TITLE", "1")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_ASSET_PACK", filepath.Join("..", "..", "generated-assets", "fd2-original-b97caf22"))

	applyPackagedPlayerDefaults()
	g := loadGame()
	if g.loadErr != "" || g.startupBlocked || g.titlePhase != "cutscene" || g.titleAssets == nil {
		t.Fatalf("正常封包未進開頭：phase=%q blocked=%v assets=%v err=%q", g.titlePhase, g.startupBlocked, g.titleAssets != nil, g.loadErr)
	}
	if g.camp == nil {
		t.Fatal("標題後方未載入完整戰役")
	}
	if g.camp.NodeID() != "story_ch00_handler" || len(g.beats) == 0 {
		t.Fatalf("標題後方未準備第 0 章：node=%q beats=%d", g.camp.NodeID(), len(g.beats))
	}

	g.enterTitleMenu()
	if !g.applyTitleMenuEvent(TitleMenuConfirm) {
		t.Fatal("START 未由正式標題輸入 owner 接收")
	}
	for i := 0; i < 24; i++ {
		if !g.applyTitleMenuEvent(TitleMenuTick) {
			t.Fatalf("START 確認閃爍 tick %d 未被接收", i)
		}
	}
	if g.titlePhase != "" || g.camp.NodeID() != "story_ch00_handler" || len(g.beats) == 0 {
		t.Fatalf("START 未揭示第 0 章：phase=%q node=%q beats=%d", g.titlePhase, g.camp.NodeID(), len(g.beats))
	}
}
