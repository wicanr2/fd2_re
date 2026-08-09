package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestEndTurnEnemyPhaseResultEntersPostbattleCutsceneThenTown(t *testing.T) {
	chapter := 2
	c := &campaign.Campaign{
		Start: "battle_fixture",
		Flags: map[string]bool{},
		Nodes: map[string]*campaign.Node{
			"battle_fixture": {
				Type: "battle", OnWin: "postbattle_fixture",
			},
			"postbattle_fixture": {
				Type: "cutscene", Next: "town_fixture",
				// The editable postbattle beats persist the party before the
				// chapter marker and visual fade; no native handler meaning is
				// invented by this fixture.
				Beats: []campaign.Beat{
					{Op: "sync_party"},
					{Op: "set_chapter", Chapter: &chapter},
				},
			},
			"town_fixture": {Type: "town", Town: "fixture"},
		},
	}
	st := &battle.State{Units: []*battle.Unit{
		{Name: "索爾", Camp: battle.Own, OnField: true, HP: 10, MaxHP: 10},
		// A dead, non-field enemy gives NextAIPlan an empty plan while still
		// exercising the real endTurn→enemy phase→finishTurn seam.
		{Camp: battle.Enemy, OnField: false, HP: 0, MaxHP: 10},
	}}
	g := &Game{camp: campaign.NewRunner(c), st: st, sc: &battle.Scenario{}}

	g.endTurn()
	if !g.aiBusy {
		t.Fatal("endTurn did not enter enemy phase")
	}
	g.aiStep()
	if g.aiBusy || g.result != "win" || st.Turn != 1 {
		t.Fatalf("enemy phase completion: aiBusy=%v result=%q turn=%d", g.aiBusy, g.result, st.Turn)
	}
	if got := g.camp.NodeID(); got != "battle_fixture" {
		t.Fatalf("result advanced before confirmation: node=%q", got)
	}

	// This calls the production battle-result confirmation seam; it must not
	// bypass the authored postbattle cutscene node.
	if !g.confirmBattleResult() || g.result != "" {
		t.Fatalf("battle result confirmation failed: result=%q", g.result)
	}
	if got := g.camp.NodeID(); got != "postbattle_fixture" {
		t.Fatalf("battle result→postbattle=%q", got)
	}
	if got, ok := g.partyRoster[0]; !ok || got.HP != got.MaxHP || got.MP != got.MaxMP {
		t.Fatalf("postbattle sync party=%+v, want restored persistent snapshot", g.partyRoster)
	}
	if g.fade == nil || !g.fade.out {
		t.Fatal("postbattle cutscene did not reach its fade transition")
	}

	// Complete only the deterministic fade-out callback; the callback enters
	// the declared town node and leaves the fade-in lifecycle for the runtime.
	g.fade.then()
	if got := g.camp.NodeID(); got != "town_fixture" {
		t.Fatalf("postbattle cutscene→town=%q", got)
	}
}

func TestUIShellVerticalTraceKeepsPostbattleTownAndShopBoundary(t *testing.T) {
	// This is deliberately a state/input contract, not a claim about native
	// renderer pixels.  The important invariant is that a won battle enters
	// its editable postbattle node and town hub before the next battle.
	var title TitleMenuState
	title.Step(TitleMenuConfirm)
	for i := 0; i < 24; i++ {
		title.Step(TitleMenuTick)
	}
	if title.Selection != 0 {
		t.Fatalf("title start selection=%d, want 0", title.Selection)
	}

	c := &campaign.Campaign{
		Start: "story_ch01_pre",
		Flags: map[string]bool{},
		Nodes: map[string]*campaign.Node{
			"story_ch01_pre": {Type: "story", Next: "battle_ch01"},
			"battle_ch01":    {Type: "battle", OnWin: "postbattle_ch01_persist"},
			"postbattle_ch01_persist": {
				Type: "cutscene", Next: "town_ch02",
				Beats: []campaign.Beat{{Op: "sync_party"}},
			},
			"town_ch02": {
				Type: "town", Town: "羅德鎮",
				Options: []campaign.Option{
					{Label: "武器店", To: "shop_ch02_weapon"},
					{Label: "出戰整備", To: "preparation_ch02"},
				},
			},
			"shop_ch02_weapon": {Type: "shop", Next: "town_ch02", Goods: []campaign.Good{{ID: 1, Name: "短劍", Price: 10}}},
			"preparation_ch02": {Type: "preparation", Next: "battle_ch02"},
			"battle_ch02":      {Type: "battle"},
		},
	}
	g := &Game{camp: campaign.NewRunner(c)}
	trace := []string{g.camp.NodeID()}
	if got := g.camp.Advance(""); got != "battle_ch01" {
		t.Fatalf("story→battle=%q", got)
	}
	trace = append(trace, g.camp.NodeID())
	if got := g.camp.Advance("win"); got != "postbattle_ch01_persist" {
		t.Fatalf("battle win→postbattle=%q", got)
	}
	trace = append(trace, g.camp.NodeID())
	if got := g.camp.Advance(""); got != "town_ch02" {
		t.Fatalf("postbattle→town=%q", got)
	}
	trace = append(trace, g.camp.NodeID())
	if got, confirm := g.stepCampaignMenu(campaign.MenuConfirm); got != 0 || !confirm {
		t.Fatalf("town initial confirm=(%d,%v)", got, confirm)
	}
	if got := g.camp.Advance("opt0"); got != "shop_ch02_weapon" {
		t.Fatalf("town→shop=%q", got)
	}
	trace = append(trace, g.camp.NodeID())
	if got := g.camp.Advance(""); got != "town_ch02" {
		t.Fatalf("shop→town=%q", got)
	}
	trace = append(trace, g.camp.NodeID())
	want := []string{"story_ch01_pre", "battle_ch01", "postbattle_ch01_persist", "town_ch02", "shop_ch02_weapon", "town_ch02"}
	if len(trace) != len(want) {
		t.Fatalf("trace=%v, want=%v", trace, want)
	}
	for i := range want {
		if trace[i] != want[i] {
			t.Fatalf("trace[%d]=%q, want %q (full=%v)", i, trace[i], want[i], trace)
		}
	}
	if g.camp.NodeID() == "battle_ch02" {
		t.Fatal("vertical trace bypassed town/shop and entered next battle")
	}
}

func TestCampaignTownPreparationInputTrace(t *testing.T) {
	c := &campaign.Campaign{
		Start: "town_ch02",
		Flags: map[string]bool{},
		Nodes: map[string]*campaign.Node{
			"town_ch02": {
				Type: "town",
				Options: []campaign.Option{
					{Label: "酒店", To: "rumor"},
					{Label: "武器店", To: "shop"},
					{Label: "出戰整備", To: "preparation_ch02"},
				},
			},
			"rumor":            {Type: "story"},
			"shop":             {Type: "shop"},
			"preparation_ch02": {Type: "preparation", Next: "story_ch02_pre"},
			"story_ch02_pre":   {Type: "cutscene", Next: "battle_ch02"},
			"battle_ch02":      {Type: "battle"},
		},
	}
	g := &Game{camp: campaign.NewRunner(c)}
	if got := g.camp.NodeID(); got != "town_ch02" {
		t.Fatalf("start node=%q", got)
	}
	if got, confirm := g.stepCampaignMenu(campaign.MenuDown); got != 1 || confirm {
		t.Fatalf("town down #1=(%d,%v)", got, confirm)
	}
	if got, confirm := g.stepCampaignMenu(campaign.MenuDown); got != 2 || confirm {
		t.Fatalf("town down #2=(%d,%v)", got, confirm)
	}
	selected, confirm := g.stepCampaignMenu(campaign.MenuConfirm)
	if selected != 2 || !confirm {
		t.Fatalf("town confirm=(%d,%v)", selected, confirm)
	}
	if got := g.camp.Advance("opt2"); got != "preparation_ch02" {
		t.Fatalf("town→preparation=%q", got)
	}
	if got := g.camp.NodeID(); got != "preparation_ch02" {
		t.Fatalf("runner node after town=%q", got)
	}
	if got := g.camp.Advance(""); got != "story_ch02_pre" {
		t.Fatalf("preparation→pre handler=%q", got)
	}
	if got := g.camp.Advance(""); got != "battle_ch02" {
		t.Fatalf("pre handler→battle=%q", got)
	}
}

func TestPreparationCancellationReturnsToEditableTownBoundary(t *testing.T) {
	c := &campaign.Campaign{
		Start: "preparation_ch02",
		Flags: map[string]bool{},
		Nodes: map[string]*campaign.Node{
			"town_ch02":        {Type: "town"},
			"preparation_ch02": {Type: "preparation", Next: "story_ch02_pre", Cancel: "town_ch02"},
			"story_ch02_pre":   {Type: "cutscene"},
		},
	}
	r := campaign.NewRunner(c)
	if got := r.Advance("cancel"); got != "town_ch02" {
		t.Fatalf("preparation cancel=%q, want town_ch02", got)
	}
	r.Cur = "preparation_ch02"
	if got := r.Advance("confirm"); got != "story_ch02_pre" {
		t.Fatalf("preparation confirm=%q, want story_ch02_pre", got)
	}
}
