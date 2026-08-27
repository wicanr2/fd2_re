package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestChapter26BattleResultNativePostbattleBranchesReachTownAndColdSave(t *testing.T) {
	fdotherPath, datoPath := nativeFDOTHERPath(), nativeDATOPath()
	if fdotherPath == "" || datoPath == "" {
		t.Skip("第26戰戰後原生對話需要玩家提供 FDOTHER.DAT 與 DATO.DAT")
	}
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skipf("FDOTHER.DAT unavailable: %v", err)
	}
	if _, err := os.Stat(datoPath); err != nil {
		t.Skipf("DATO.DAT unavailable: %v", err)
	}
	for _, tc := range []struct {
		name            string
		eventState12    byte
		wantUtterances  int
		wantStringIndex map[int]bool
	}{
		{"event_state_12_zero", 0, 18, map[int]bool{5: true, 7: true, 8: true, 10: true, 11: true}},
		{"event_state_12_nonzero", 1, 33, map[int]bool{6: true, 7: true, 9: true, 10: true, 11: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scenario, err := battle.LoadScenario(assetPath("assets/scenarios/ch26.json"))
			if err != nil {
				t.Fatal(err)
			}
			order := make([]int, len(scenario.Party))
			members := make(map[int]bool, len(order))
			for i, unit := range scenario.Party {
				order[i] = unit.Fig
				members[unit.Fig] = true
			}
			g := &Game{
				partyMembers:   members,
				partyJoinOrder: append([]int(nil), order...),
				partyDeploy:    make(map[int]bool, 15),
			}
			for _, id := range order[1:16] {
				g.partyDeploy[id] = true
			}
			if err := g.seedPersistentPartyFromLoadCH(order, scenario.PartyUnits(nil)); err != nil {
				t.Fatal(err)
			}
			if err := g.loadMap("assets/maps/map25"); err != nil {
				t.Fatal(err)
			}
			g.resetBattle("assets/maps/map25/map25_units.json", "assets/scenarios/ch26.json")
			if g.loadErr != "" || g.st == nil || g.sc == nil || len(g.st.Units) != 57 {
				t.Fatalf("第26戰入口 err=%q state=%v scenario=%v slots=%d，want 57",
					g.loadErr, g.st != nil, g.sc != nil, g.handlerUnitCount())
			}
			g.st.NativeEventState[12] = tc.eventState12

			emptyX, emptyY, found := 0, 0, false
			for y := 0; y < g.st.H && !found; y++ {
				for x := 0; x < g.st.W; x++ {
					if g.st.UnitAt(x, y) == nil {
						emptyX, emptyY, found = x, y, true
						break
					}
				}
			}
			if !found {
				t.Fatal("第26戰地圖沒有可供原生視圖使用的空格")
			}
			g.curX, g.curY = emptyX, emptyY
			if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{
				CursorX: emptyX, CursorY: emptyY, VisibleCursorX: emptyX, VisibleCursorY: emptyY,
			}); err != nil || !g.st.MaterializeNativeMapHUDState(1, 1, 1) ||
				!g.st.MaterializeNativeMapRangeMode(1) {
				t.Fatalf("第26戰原生視圖初始化失敗: %v", err)
			}
			if err := g.composeNativeMapFrame(); err != nil {
				t.Fatal(err)
			}

			full, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
			if err != nil {
				t.Fatal(err)
			}
			full.Start = "battle_ch26"
			g.camp = campaign.NewRunner(full)
			g.result = "win"
			if !g.confirmBattleResult() || g.result != "" || g.camp.NodeID() != "postbattle_ch26_persist" {
				t.Fatalf("第26戰勝利邊界 node=%q result=%q err=%q", g.camp.NodeID(), g.result, g.loadErr)
			}

			seenDialogue := make(map[[2]int]bool, tc.wantUtterances)
			seenActs := make(map[string]bool, 4)
			for frame := 0; frame < 120000 && g.camp.NodeID() != "town_ch27"; frame++ {
				if g.actJob != nil && g.beatIdx >= 0 && g.beatIdx < len(g.beats) {
					seenActs[g.beats[g.beatIdx].Source] = true
				}
				if len(g.dialog) != 0 {
					current := g.dialog[len(g.dialog)-1]
					if current.NativeDialogue == nil || current.Upper == nil ||
						current.NativeDialogue.SourceDAT != "FDTXT_026" ||
						!tc.wantStringIndex[current.NativeDialogue.StringIndex] ||
						len(g.nativeDialogueOpening) != 5 || len(g.nativeDialogueClosing) != 5 {
						t.Fatalf("第26戰戰後對話生命週期漂移: %#v", current)
					}
					seenDialogue[[2]int{current.NativeDialogue.StringIndex, current.NativeDialogue.Utterance}] = true
					if g.nativeStoryDialogueAtInputWait() &&
						!g.handleNativeStoryInput(g.camp.Node(), nativeStoryInput{enter: true}) {
						t.Fatal("第26戰戰後正式故事輸入遭拒")
					}
				}
				if err := g.Update(); err != nil {
					t.Fatal(err)
				}
				if g.loadErr != "" {
					t.Fatalf("第26戰戰後 node=%q beat=%d/%d: %s", g.camp.NodeID(), g.beatIdx, len(g.beats), g.loadErr)
				}
			}
			if g.camp.NodeID() != "town_ch27" || g.st != nil || g.handlerChapter != 26 {
				t.Fatalf("第26戰戰後邊界 node=%q battle=%v chapter=%d", g.camp.NodeID(), g.st != nil, g.handlerChapter)
			}
			if len(seenDialogue) != tc.wantUtterances {
				t.Fatalf("第26戰戰後對話=%d，want %d", len(seenDialogue), tc.wantUtterances)
			}
			for _, source := range []string{"0x24f57", "0x24f92", "0x24fd8", "0x25013"} {
				if !seenActs[source] {
					t.Fatalf("第26戰戰後未執行原始 ACTING caller %s；實際=%v", source, seenActs)
				}
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
			if cold.loadErr != "" || cold.camp.NodeID() != "town_ch27" || cold.st != nil ||
				cold.handlerChapter != 26 || !reflect.DeepEqual(cold.partyRoster, wantRoster) ||
				!reflect.DeepEqual(cold.partyJoinOrder, wantOrder) {
				t.Fatalf("town_ch27 冷讀 node=%q chapter=%d roster=%d order=%v battle=%v err=%q",
					cold.camp.NodeID(), cold.handlerChapter, len(cold.partyRoster), cold.partyJoinOrder,
					cold.st != nil, cold.loadErr)
			}
		})
	}
}
