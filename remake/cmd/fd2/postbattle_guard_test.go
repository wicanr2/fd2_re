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
