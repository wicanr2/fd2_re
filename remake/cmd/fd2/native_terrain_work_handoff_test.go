package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// #52 的正常 LOAD → 城鎮 → 整備 → 故事 → 戰場診斷。
// 兩種繪圖方式都必須延續原版 tile21 literal 寫入的索引 138；不注入像素。
func TestNativeTerrainWorkHandoffDiagnostic(t *testing.T) {
	slot, out := os.Getenv("FD2_PARITY_SLOT"), os.Getenv("FD2_TERRAIN_HANDOFF_OUT")
	if slot == "" || out == "" {
		t.Skip("需要固定第十二章槽與診斷輸出")
	}
	for _, draw := range []bool{false, true} {
		t.Setenv("FD2_TITLE", "1")
		t.Setenv("FD2_SEED", "4")
		t.Setenv("FD2_MUTE", "1")
		t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
		t.Setenv("FD2_NATIVE_SAVE", slot)
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		userDataDirCached = ""
		g := loadGame()
		if g.loadErr != "" {
			t.Fatal(g.loadErr)
		}
		g.titlePhase, g.titleSel = "menu", 0
		if !g.applyTitleMenuEvent(TitleMenuDown) || !g.applyTitleMenuEvent(TitleMenuConfirm) {
			t.Fatal("LOAD")
		}
		for i := 0; i < 24; i++ {
			g.applyTitleMenuEvent(TitleMenuTick)
		}
		if !g.applyTitleSlotEvent(TitleSlotConfirm) || g.camp.NodeID() != "town_ch12" {
			t.Fatal("槽")
		}
		r := &parityReplay{t: t, g: g, battle: "battle_ch12", town: "town_ch12"}
		r.settleTown()
		for g.campSel != 2 {
			if !g.moveNativeTownSelection(1) {
				t.Fatal("城鎮出口")
			}
		}
		r.enterTownOption(2)
		if !pump(t, g, 600, func() bool { return !g.nativeClassUIBlocksInput() }) {
			t.Fatal("整備")
		}
		if !g.handleNativePreparationInput(nativePreparationInput{enter: true}) ||
			!pump(t, g, 600, func() bool { return g.camp.Node().Type != "preparation" }) {
			t.Fatal("YES")
		}
		screen := ebiten.NewImage(640, 400)
		var points []map[string]interface{}
		var prior string
		observe := func() bool {
			node := g.camp.NodeID()
			if draw && node != "battle_ch12" {
				g.Draw(screen)
			}
			value := -1
			const offset = 0x8088 + 105*456 + 7
			if len(g.nativeMapWork) > offset {
				value = int(g.nativeMapWork[offset])
			}
			key := node + " " + string(rune(value+1))
			if key != prior {
				points = append(points, map[string]interface{}{"node": node, "draw": draw, "work_size": len(g.nativeMapWork), "work_offset": offset, "index": value, "story_view": g.storyNativeMapView, "has_story_view": g.hasStoryNativeMapView, "beat": g.beatIdx})
				prior = key
			}
			return node == "battle_ch12"
		}
		driveStory(t, g, newJourneyTrace(), observe)
		const retainedOffset = 0x8088 + 105*456 + 7
		if len(g.nativeMapWork) <= retainedOffset || g.nativeMapWork[retainedOffset] != 138 {
			t.Fatalf("故事交接未保留原版 literal 138：draw=%v work_size=%d", draw, len(g.nativeMapWork))
		}
		if err := g.composeNativeMapFrame(); err != nil {
			t.Fatal(err)
		}
		if got := g.nativeMapWork[retainedOffset]; got != 138 {
			t.Fatalf("首次戰場合成改寫 mode3 保留值：draw=%v got=%d", draw, got)
		}
		points = append(points, map[string]interface{}{"node": "battle_ch12_composed", "draw": draw, "index": g.nativeMapWork[0x8088+105*456+7]})
		if err := os.MkdirAll(out, 0755); err != nil {
			t.Fatal(err)
		}
		name := "without-draw.json"
		if draw {
			name = "with-draw.json"
		}
		data, err := json.MarshalIndent(points, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(out, name), append(data, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
		t.Logf("draw=%v observed=%d", draw, len(points))
		screen.Dispose()
	}
	userDataDirCached = ""
}

// 完整緩衝只可交給同一 LOADCH；其他章或不完整來源不能沿用殘留像素。
func TestNativeCh12StoryWorkHandoffRejectsForeignOrIncompleteSource(t *testing.T) {
	makeGame := func() *Game {
		return &Game{
			camp:                  campaign.NewRunner(&campaign.Campaign{Start: "battle_ch12"}),
			nativeMapAssets:       &nativeMapAssets{MapIndex: 11},
			hasStoryNativeMapView: true, storyNativeMapState: &battle.State{},
			storyActors: []battle.Unit{{}}, storyRosterPath: "roster11", storyPartyScenario: "scenario11",
			nativeMapWork: make([]byte, indexedmap.NativeUnitPresentWorkSize),
			nativeMapVGA:  make([]byte, indexedmap.NativeMapVGASize),
		}
	}
	g := makeGame()
	if !g.nativeCh12StoryWorkHandoff("roster11", "scenario11") {
		t.Fatal("同一LOADCH完整來源遭拒")
	}
	cases := []struct {
		name   string
		mutate func(*Game)
	}{
		{"別章", func(g *Game) { g.camp.Cur = "battle_ch13" }},
		{"別地圖", func(g *Game) { g.nativeMapAssets.MapIndex = 12 }},
		{"別隊伍", func(g *Game) { g.storyRosterPath = "other" }},
		{"別場景", func(g *Game) { g.storyPartyScenario = "other" }},
		{"無故事視圖", func(g *Game) { g.hasStoryNativeMapView = false }},
		{"無故事狀態", func(g *Game) { g.storyNativeMapState = nil }},
		{"無角色", func(g *Game) { g.storyActors = nil }},
		{"工作緩衝短缺", func(g *Game) { g.nativeMapWork = g.nativeMapWork[:len(g.nativeMapWork)-1] }},
		{"顯示緩衝短缺", func(g *Game) { g.nativeMapVGA = g.nativeMapVGA[:len(g.nativeMapVGA)-1] }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := makeGame()
			c.mutate(g)
			if g.nativeCh12StoryWorkHandoff("roster11", "scenario11") {
				t.Fatal("不相容來源沿用緩衝")
			}
		})
	}
	if g.nativeCh12StoryWorkHandoff("", "scenario11") || g.nativeCh12StoryWorkHandoff("roster11", "") {
		t.Fatal("空來源沿用緩衝")
	}
}
