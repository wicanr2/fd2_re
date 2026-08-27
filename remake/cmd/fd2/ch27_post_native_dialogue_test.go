package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestChapter28BattleResultNativePostbattleReachesPreparationAndColdSave(t *testing.T) {
	fdotherPath, datoPath := nativeFDOTHERPath(), nativeDATOPath()
	if fdotherPath == "" || datoPath == "" {
		t.Skip("第28戰戰後原生對話需要玩家提供 FDOTHER.DAT 與 DATO.DAT")
	}
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skipf("FDOTHER.DAT unavailable: %v", err)
	}
	if _, err := os.Stat(datoPath); err != nil {
		t.Skipf("DATO.DAT unavailable: %v", err)
	}

	g := seedLateCampaignParty(t, "assets/scenarios/ch28.json", 20)
	// 正常晚期路徑來自 persistent record；聚焦fixture沒有真的先寫入原生槽，
	// 因此把該格式已證實的+0x42/+0x46投影補回，而不是從戰後畫面猜值。
	for id, unit := range g.partyRoster {
		if unit.MaxHP < 0 || unit.MaxHP > 0xffff || unit.MaxMP < 0 || unit.MaxMP > 0xffff {
			t.Fatalf("late persistent fixture id=%d HP/MP超出raw word: %d/%d", id, unit.MaxHP, unit.MaxMP)
		}
		unit.NativeRecordWord42, unit.HasNativeRecordWord42 = uint16(unit.MaxHP), true
		unit.NativeRecordWord46, unit.HasNativeRecordWord46 = uint16(unit.MaxMP), true
		g.partyRoster[id] = unit
	}
	full, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	full.Start = "story_ch28"
	g.camp = campaign.NewRunner(full)
	g.enterNode()
	if g.loadErr != "" {
		t.Fatalf("第28戰前置handler入口: %s", g.loadErr)
	}
	if err := g.fastForwardShotCampaign(); err != nil {
		t.Fatal(err)
	}
	if g.loadErr != "" || g.camp.NodeID() != "battle_ch28" || g.st == nil || g.sc == nil ||
		len(g.st.Units) != 64 {
		t.Fatalf("第28戰入口 node=%q err=%q state=%v scenario=%v slots=%d，want 64",
			g.camp.NodeID(), g.loadErr, g.st != nil, g.sc != nil, g.handlerUnitCount())
	}
	// 真實長程程序會從存檔／controller保留HUD gate；這個聚焦fixture從
	// story_ch28建立戰況而沒有先冷讀原版槽，因此只補上已閉合的controller
	// entry gate，不改鏡頭、游標、selector cache或任何單位資料。
	if !g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatal("第28戰聚焦fixture無法物化已證實的HUD controller gate")
	}
	for index, unit := range g.st.Units {
		if _, ok := unit.NativeUnitLayerEntry(); !ok {
			t.Fatalf("第28戰聚焦fixture slot%d缺原生unit layer: %#v", index, unit)
		}
		if _, ok := unit.NativeForegroundLayerEntry(); !ok {
			t.Fatalf("第28戰聚焦fixture slot%d缺原生foreground: %#v", index, unit)
		}
	}

	g.result = "win"
	if !g.confirmBattleResult() || g.result != "" || g.camp.NodeID() != "postbattle_ch28_persist" {
		t.Fatalf("第28戰勝利邊界 node=%q result=%q err=%q", g.camp.NodeID(), g.result, g.loadErr)
	}

	seen := make(map[[2]int]bool, 5)
	for frame := 0; frame < 30000 && g.camp.NodeID() != "preparation_ch29"; frame++ {
		if len(g.dialog) != 0 {
			current := g.dialog[len(g.dialog)-1]
			if current.NativeDialogue == nil || current.Upper == nil || *current.Upper ||
				current.NativeDialogue.SourceDAT != "FDTXT_028" ||
				current.NativeDialogue.StringIndex != 7 ||
				len(g.nativeDialogueOpening) != 5 || len(g.nativeDialogueClosing) != 5 {
				t.Fatalf("第28戰戰後對話生命週期漂移: %#v", current)
			}
			seen[[2]int{7, current.NativeDialogue.Utterance}] = true
			if g.nativeStoryDialogueAtInputWait() &&
				!g.handleNativeStoryInput(g.camp.Node(), nativeStoryInput{enter: true}) {
				t.Fatal("第28戰戰後正式故事輸入遭拒")
			}
		}
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		if g.loadErr != "" {
			t.Fatalf("第28戰戰後 node=%q beat=%d/%d: %s (state=%v view=%v cycle=%v cache=%v hud=%v range=%v vga=%d cursor=%d,%d assets=%v)",
				g.camp.NodeID(), g.beatIdx, len(g.beats), g.loadErr, g.st != nil,
				g.st != nil && g.st.HasNativeMapViewState,
				g.st != nil && g.st.HasNativeMapCycleState,
				g.st != nil && g.st.NativeMapSelectorCache != nil,
				g.st != nil && g.st.HasNativeMapHUDState,
				g.st != nil && g.st.HasNativeMapRangeModeState, len(g.nativeMapVGA),
				g.curX, g.curY, nativeMapAssetsAvailable(g.nativeMapAssets))
		}
	}
	if g.camp.NodeID() != "preparation_ch29" || g.st != nil || g.handlerChapter != 28 || len(seen) != 5 {
		t.Fatalf("第28戰戰後邊界 node=%q battle=%v chapter=%d dialogues=%d",
			g.camp.NodeID(), g.st != nil, g.handlerChapter, len(seen))
	}

	oldCache := userDataDirCached
	userDataDirCached = ""
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Cleanup(func() { userDataDirCached = oldCache })
	g.saveGameToSlot(0)
	wantRoster := clonePartyRoster(g.partyRoster)
	wantOrder := append([]int(nil), g.partyJoinOrder...)
	coldCampaign, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	cold := &Game{camp: campaign.NewRunner(coldCampaign)}
	cold.loadGameFromSlot(0)
	if cold.loadErr != "" || cold.camp.NodeID() != "preparation_ch29" || cold.st != nil ||
		cold.handlerChapter != 28 || !reflect.DeepEqual(cold.partyRoster, wantRoster) ||
		!reflect.DeepEqual(cold.partyJoinOrder, wantOrder) {
		t.Fatalf("preparation_ch29冷讀 node=%q chapter=%d roster=%d order=%v battle=%v err=%q",
			cold.camp.NodeID(), cold.handlerChapter, len(cold.partyRoster), cold.partyJoinOrder,
			cold.st != nil, cold.loadErr)
	}
}
