package main

import (
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

func clonePartyRoster(src map[int]battle.Unit) map[int]battle.Unit {
	dst := make(map[int]battle.Unit, len(src))
	for id, unit := range src {
		dst[id] = cloneNativeShopUnit(unit)
	}
	return dst
}

func TestCampaignFullFinalWinSyncsCurrentPartyBeforeEnding(t *testing.T) {
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
		camp:         runner,
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

func TestEndingPartySnapshotFailsClosedBeforeCampaignAdvance(t *testing.T) {
	c := &campaign.Campaign{Start: "battle", Nodes: map[string]*campaign.Node{
		"battle": {Type: "battle", OnWin: "ending", EndingPartySnapshotOnWin: true},
		"ending": {Type: "ending"},
	}}
	g := &Game{camp: campaign.NewRunner(c), result: "win"}
	if g.confirmBattleResult() || g.camp.NodeID() != "battle" || g.result != "win" ||
		!strings.Contains(g.loadErr, "缺少已完成的戰場狀態") {
		t.Fatalf("終局同步失敗仍跨越戰役邊界: node=%q result=%q err=%q", g.camp.NodeID(), g.result, g.loadErr)
	}
}

func TestEndingPartySnapshotRejectsZeroMatchedRecords(t *testing.T) {
	c := &campaign.Campaign{Start: "battle", Nodes: map[string]*campaign.Node{
		"battle": {Type: "battle", OnWin: "ending", EndingPartySnapshotOnWin: true},
		"ending": {Type: "ending"},
	}}
	current := battle.Unit{
		Fig: 0, Camp: battle.Own, OnField: true,
		NativeIdentity: 7, HasNativeIdentity: true,
	}
	stale := current
	stale.NativeIdentity = 8
	g := &Game{
		camp: campaign.NewRunner(c), result: "win",
		st:           &battle.State{Units: []*battle.Unit{&current}},
		partyMembers: map[int]bool{0: true}, partyRoster: map[int]battle.Unit{0: stale},
	}
	if g.confirmBattleResult() || g.camp.NodeID() != "battle" || g.result != "win" ||
		!strings.Contains(g.loadErr, "沒有任何持續隊伍身分符合") {
		t.Fatalf("零筆終局同步仍跨越戰役邊界: node=%q result=%q err=%q", g.camp.NodeID(), g.result, g.loadErr)
	}
}

func TestSourceBoundEndingConsumesFinalPartySnapshot(t *testing.T) {
	c := &campaign.Campaign{Start: "battle", Nodes: map[string]*campaign.Node{
		"battle": {Type: "battle", OnWin: "ending", EndingPartySnapshotOnWin: true},
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
	if got := g.partyRoster[0].Lv; got != 21 {
		t.Fatalf("來源約束終局未消費最後隊伍快照: Lv=%d", got)
	}
}
