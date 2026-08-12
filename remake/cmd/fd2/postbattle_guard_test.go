package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestUnboundPostbattleCutsceneFailsClosed(t *testing.T) {
	c := &campaign.Campaign{
		Start: "postbattle_ch04_persist",
		Nodes: map[string]*campaign.Node{
			"postbattle_ch04_persist": {Type: "cutscene", Next: "town_ch05"},
			"town_ch05":               {Type: "town"},
		},
	}
	g := &Game{camp: campaign.NewRunner(c)}
	g.enterNode()
	if got := g.camp.NodeID(); got != "postbattle_ch04_persist" {
		t.Fatalf("unbound postbattle advanced to %q", got)
	}
	if !strings.Contains(g.loadErr, "no active handler binding") || g.msg == "" {
		t.Fatalf("missing fail-closed diagnostics: loadErr=%q msg=%q", g.loadErr, g.msg)
	}
}

func TestApproximatePostbattlePreservesAuthoredIntermissionBoundary(t *testing.T) {
	tests := []struct {
		name, next, nextType string
	}{
		{name: "town", next: "town_after", nextType: "town"},
		{name: "preparation", next: "preparation_after", nextType: "preparation"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &campaign.Campaign{
				Start: "postbattle_missing_handler",
				Nodes: map[string]*campaign.Node{
					"postbattle_missing_handler": {Type: "cutscene", Next: tc.next},
					tc.next:                      {Type: tc.nextType},
				},
			}
			unit := &battle.Unit{Fig: 0, Camp: battle.Own, OnField: true, HP: 4, MaxHP: 10, MP: 1, MaxMP: 3}
			g := &Game{
				camp:            campaign.NewRunner(c),
				approximateMode: true,
				st:              &battle.State{Units: []*battle.Unit{unit}},
				partyMembers:    map[int]bool{0: true},
				partyRoster:     map[int]battle.Unit{0: *unit},
			}
			g.enterNode()
			if g.loadErr != "" || !g.approximatePostbattle {
				t.Fatalf("approximate postbattle did not stop at prompt: err=%q pending=%v", g.loadErr, g.approximatePostbattle)
			}
			if got := g.camp.NodeID(); got != "postbattle_missing_handler" {
				t.Fatalf("approximate postbattle advanced before confirmation to %q", got)
			}
			if got := g.partyRoster[0].HP; got != 10 {
				t.Fatalf("battle snapshot was not synchronized before intermission: HP=%d", got)
			}
			if !g.continueApproximatePostbattle() {
				t.Fatal("approximate postbattle confirmation was rejected")
			}
			if got := g.camp.NodeID(); got != tc.next {
				t.Fatalf("confirmed approximate postbattle landed at %q, want %q", got, tc.next)
			}
			if g.st != nil {
				t.Fatalf("intermission boundary retained battle state for %s: %#v", tc.nextType, g.st)
			}
		})
	}
}

func TestApproximateCampaignFullUnboundPostbattleBoundaries(t *testing.T) {
	tests := []struct {
		postbattle, next string
	}{
		{postbattle: "postbattle_ch23_persist", next: "preparation_ch24"},
		{postbattle: "postbattle_ch24_persist", next: "preparation_ch25"},
		{postbattle: "postbattle_ch25_persist", next: "town_ch26"},
		{postbattle: "postbattle_ch29_persist", next: "preparation_ch30"},
	}
	for _, tc := range tests {
		t.Run(tc.postbattle, func(t *testing.T) {
			campaignData, err := campaign.Load("../../assets/scenarios/campaign_full.json")
			if err != nil {
				t.Fatal(err)
			}
			runner := campaign.NewRunner(campaignData)
			runner.Cur = tc.postbattle
			unit := &battle.Unit{
				Fig: 0, Camp: battle.Own, OnField: true,
				HP: 4, MaxHP: 10, MP: 1, MaxMP: 3,
			}
			g := &Game{
				camp:            runner,
				approximateMode: true,
				st:              &battle.State{Units: []*battle.Unit{unit}},
				partyMembers:    map[int]bool{0: true},
				partyRoster:     map[int]battle.Unit{0: *unit},
			}
			g.enterNode()
			if g.loadErr != "" || !g.approximatePostbattle {
				t.Fatalf("actual campaign node did not enter approximate prompt: node=%q err=%q pending=%v", g.camp.NodeID(), g.loadErr, g.approximatePostbattle)
			}
			if g.camp.NodeID() != tc.postbattle || g.msg == "" {
				t.Fatalf("actual campaign node advanced before confirmation: node=%q msg=%q", g.camp.NodeID(), g.msg)
			}
			if got := g.partyRoster[0].HP; got != 10 {
				t.Fatalf("actual campaign postbattle did not synchronize party HP: %d", got)
			}
			if !g.continueApproximatePostbattle() {
				t.Fatal("actual campaign approximate confirmation was rejected")
			}
			if g.camp.NodeID() != tc.next || g.st != nil {
				t.Fatalf("actual campaign intermission boundary node=%q state=%#v, want next=%q and cleared battle state", g.camp.NodeID(), g.st, tc.next)
			}
		})
	}
}

func TestApproximateCampaignFullResultConfirmationKeepsUnboundIntermissions(t *testing.T) {
	tests := []struct {
		battle, postbattle, next string
	}{
		{battle: "battle_ch23", postbattle: "postbattle_ch23_persist", next: "preparation_ch24"},
		{battle: "battle_ch24", postbattle: "postbattle_ch24_persist", next: "preparation_ch25"},
		{battle: "battle_ch25", postbattle: "postbattle_ch25_persist", next: "town_ch26"},
		{battle: "battle_ch29", postbattle: "postbattle_ch29_persist", next: "preparation_ch30"},
	}
	for _, tc := range tests {
		t.Run(tc.battle, func(t *testing.T) {
			campaignData, err := campaign.Load("../../assets/scenarios/campaign_full.json")
			if err != nil {
				t.Fatal(err)
			}
			campaignData.Start = tc.battle
			unit := &battle.Unit{Fig: 0, Camp: battle.Own, OnField: true, HP: 10, MaxHP: 10, MP: 1, MaxMP: 3}
			g := &Game{
				camp:            campaign.NewRunner(campaignData),
				approximateMode: true,
				st: &battle.State{Units: []*battle.Unit{
					unit,
					{Fig: 1, Camp: battle.Enemy, OnField: false, HP: 0, MaxHP: 10},
				}},
				partyMembers: map[int]bool{0: true},
				partyRoster:  map[int]battle.Unit{0: *unit},
			}
			g.result = "win"
			if !g.confirmBattleResult() || g.result != "" {
				t.Fatalf("production result confirmation failed: node=%q result=%q err=%q", g.camp.NodeID(), g.result, g.loadErr)
			}
			if g.camp.NodeID() != tc.postbattle || g.loadErr != "" || !g.approximatePostbattle {
				t.Fatalf("result entered wrong intermission: node=%q err=%q pending=%v", g.camp.NodeID(), g.loadErr, g.approximatePostbattle)
			}
			if !g.continueApproximatePostbattle() {
				t.Fatal("approximate intermission confirmation was rejected")
			}
			if g.camp.NodeID() != tc.next || g.st != nil {
				t.Fatalf("approximate result boundary node=%q state=%#v, want %q and cleared battle", g.camp.NodeID(), g.st, tc.next)
			}
		})
	}
}

func TestCampaignFullUnboundPostbattleDefaultsFailClosed(t *testing.T) {
	for _, nodeID := range []string{
		"postbattle_ch23_persist", "postbattle_ch24_persist",
		"postbattle_ch25_persist", "postbattle_ch29_persist",
	} {
		t.Run(nodeID, func(t *testing.T) {
			campaignData, err := campaign.Load("../../assets/scenarios/campaign_full.json")
			if err != nil {
				t.Fatal(err)
			}
			runner := campaign.NewRunner(campaignData)
			runner.Cur = nodeID
			g := &Game{
				camp: runner,
				st: &battle.State{Units: []*battle.Unit{{
					Fig: 0, Camp: battle.Own, OnField: true, HP: 4, MaxHP: 10,
				}}},
			}
			g.enterNode()
			if g.camp.NodeID() != nodeID || g.loadErr == "" || g.approximatePostbattle {
				t.Fatalf("unbound actual campaign node was not fail-closed: node=%q err=%q pending=%v", g.camp.NodeID(), g.loadErr, g.approximatePostbattle)
			}
		})
	}
}

func TestApproximateCampaignFullCh29Preparation30SaveLoadBoundary(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	oldCache := userDataDirCached
	userDataDirCached = ""
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Cleanup(func() { userDataDirCached = oldCache })

	campaignData, err := campaign.Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	runner := campaign.NewRunner(campaignData)
	runner.Cur = "battle_ch29"
	stale := battle.Unit{
		Fig: 0, Camp: battle.Own, OnField: true,
		Lv: 31, HP: 8, MaxHP: 40, MP: 2, MaxMP: 12,
		NativeIdentity: 0, HasNativeIdentity: true,
	}
	current := stale
	current.Lv, current.Exp, current.MaxHP, current.HP = 32, 18.5, 44, 9
	g := &Game{
		camp: runner, approximateMode: true, handlerChapter: 29,
		st:           &battle.State{Units: []*battle.Unit{&current}},
		partyMembers: map[int]bool{0: true}, partyJoinOrder: []int{0},
		partyRoster: map[int]battle.Unit{0: stale},
	}
	g.result = "win"
	if !g.confirmBattleResult() || g.camp.NodeID() != "postbattle_ch29_persist" || !g.approximatePostbattle {
		t.Fatalf("第29戰結果未停在近似戰後邊界: node=%q pending=%v err=%q", g.camp.NodeID(), g.approximatePostbattle, g.loadErr)
	}
	if !g.continueApproximatePostbattle() || g.camp.NodeID() != "preparation_ch30" || g.st != nil {
		t.Fatalf("第30戰整備邊界錯誤: node=%q battle=%v err=%q", g.camp.NodeID(), g.st != nil, g.loadErr)
	}
	wantRoster := clonePartyRoster(g.partyRoster)
	wantOrder := append([]int(nil), g.partyJoinOrder...)
	wantChapter := g.handlerChapter
	if got := wantRoster[0]; got.Lv != 32 || got.Exp != 18.5 || got.HP != 44 {
		t.Fatalf("第29戰結果未同步到持續隊伍: %#v", got)
	}

	g.saveGameToSlot(0)
	if !strings.Contains(g.msg, "preparation_ch30") {
		t.Fatalf("第30戰整備節點未建立存檔: %q", g.msg)
	}
	g.partyMembers, g.partyJoinOrder, g.partyDeploy, g.partyRoster = nil, nil, nil, nil
	g.handlerChapter = 0
	g.st = &battle.State{}
	g.loadGameFromSlot(0)
	if g.loadErr != "" || g.camp.NodeID() != "preparation_ch30" || g.st != nil ||
		g.handlerChapter != wantChapter || !reflect.DeepEqual(g.partyRoster, wantRoster) ||
		!reflect.DeepEqual(g.partyJoinOrder, wantOrder) {
		t.Fatalf("第30戰整備存讀檔不一致: node=%q chapter=%d roster=%#v order=%v battle=%v err=%q",
			g.camp.NodeID(), g.handlerChapter, g.partyRoster, g.partyJoinOrder, g.st != nil, g.loadErr)
	}
}

func clonePartyRoster(src map[int]battle.Unit) map[int]battle.Unit {
	dst := make(map[int]battle.Unit, len(src))
	for id, unit := range src {
		dst[id] = cloneNativeShopUnit(unit)
	}
	return dst
}

func TestApproximateCampaignFullFinalWinSyncsCurrentPartyBeforeEnding(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	campaignData, err := campaign.Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	// 本測試只驗證玩家結果確認的資料邊界；結局資產播放器有獨立 admission
	// 回歸，因此移除測試副本的呈現入口，避免把玩家外部資產變成此測試前提。
	campaignData.Nodes["ending"].NativeEndingPrefix = nil
	runner := campaign.NewRunner(campaignData)
	runner.Cur = "battle_ch30"
	stale := battle.Unit{
		Fig: 0, Camp: battle.Own, OnField: true,
		Lv: 39, HP: 30, MaxHP: 50,
		NativeIdentity: 0, HasNativeIdentity: true,
	}
	current := stale
	current.Lv, current.Exp, current.HP, current.MaxHP = 40, 77.25, 7, 56
	g := &Game{
		camp: runner, approximateMode: true,
		st:           &battle.State{Units: []*battle.Unit{&current}},
		partyMembers: map[int]bool{0: true}, partyJoinOrder: []int{0},
		partyRoster: map[int]battle.Unit{0: stale},
		result:      "win",
	}
	if !g.confirmBattleResult() || g.camp.NodeID() != "ending" || g.result != "" || g.loadErr != "" {
		t.Fatalf("最終勝利未進入結局: node=%q result=%q err=%q", g.camp.NodeID(), g.result, g.loadErr)
	}
	got := g.partyRoster[0]
	if got.Lv != 40 || got.Exp != 77.25 || got.MaxHP != 56 || got.HP != 56 {
		t.Fatalf("結局仍讀到最終戰之前的隊伍資料: %#v", got)
	}
}

func TestApproximateTerminalPartySyncFailsClosedBeforeCampaignAdvance(t *testing.T) {
	c := &campaign.Campaign{Start: "battle", Nodes: map[string]*campaign.Node{
		"battle": {Type: "battle", OnWin: "ending", ApproximateWinSync: true},
		"ending": {Type: "ending"},
	}}
	g := &Game{camp: campaign.NewRunner(c), approximateMode: true, result: "win"}
	if g.confirmBattleResult() || g.camp.NodeID() != "battle" || g.result != "win" ||
		!strings.Contains(g.loadErr, "缺少已完成的戰場狀態") {
		t.Fatalf("終局同步失敗仍跨越戰役邊界: node=%q result=%q err=%q", g.camp.NodeID(), g.result, g.loadErr)
	}
}

func TestApproximateTerminalPartySyncRejectsZeroMatchedRecords(t *testing.T) {
	c := &campaign.Campaign{Start: "battle", Nodes: map[string]*campaign.Node{
		"battle": {Type: "battle", OnWin: "ending", ApproximateWinSync: true},
		"ending": {Type: "ending"},
	}}
	current := battle.Unit{
		Fig: 0, Camp: battle.Own, OnField: true,
		NativeIdentity: 7, HasNativeIdentity: true,
	}
	stale := current
	stale.NativeIdentity = 8
	g := &Game{
		camp: campaign.NewRunner(c), approximateMode: true, result: "win",
		st:           &battle.State{Units: []*battle.Unit{&current}},
		partyMembers: map[int]bool{0: true}, partyRoster: map[int]battle.Unit{0: stale},
	}
	if g.confirmBattleResult() || g.camp.NodeID() != "battle" || g.result != "win" ||
		!strings.Contains(g.loadErr, "沒有任何持續隊伍身分符合") {
		t.Fatalf("零筆終局同步仍跨越戰役邊界: node=%q result=%q err=%q", g.camp.NodeID(), g.result, g.loadErr)
	}
}

func TestFaithfulModeDoesNotConsumeApproximateTerminalPartySync(t *testing.T) {
	c := &campaign.Campaign{Start: "battle", Nodes: map[string]*campaign.Node{
		"battle": {Type: "battle", OnWin: "ending", ApproximateWinSync: true},
		"ending": {Type: "ending"},
	}}
	stale := battle.Unit{Fig: 0, Camp: battle.Own, OnField: true, Lv: 20, HP: 30, MaxHP: 30}
	current := stale
	current.Lv = 21
	g := &Game{
		camp: campaign.NewRunner(c), result: "win",
		st:           &battle.State{Units: []*battle.Unit{&current}},
		partyMembers: map[int]bool{0: true}, partyRoster: map[int]battle.Unit{0: stale},
	}
	if !g.confirmBattleResult() || g.camp.NodeID() != "ending" {
		t.Fatalf("忠實模式終局邊錯誤: node=%q err=%q", g.camp.NodeID(), g.loadErr)
	}
	if got := g.partyRoster[0].Lv; got != 20 {
		t.Fatalf("忠實模式誤用了近似終局同步: Lv=%d", got)
	}
}
