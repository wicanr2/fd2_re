package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestPreparationSelectionUsesOriginalCapsWithoutChangingJoinRoster(t *testing.T) {
	g := &Game{
		partyJoinOrder: []int{0, 9, 4, 30},
		partyRoster: map[int]battle.Unit{
			0: {Fig: 0, Name: "索爾"}, 9: {Fig: 9, Name: "悠妮"},
			4: {Fig: 4, Name: "亞雷斯"}, 30: {Fig: 30, Name: "蓋亞"},
		},
		partyMembers: map[int]bool{0: true, 9: true, 4: true, 30: true},
	}
	g.setupPreparation(&campaign.Node{Type: "preparation", PartyLimit: 19})
	if g.preparationSelected() != 4 || len(g.partyDeploy) != 4 {
		t.Fatalf("small roster should keep all selected: selected=%d deploy=%v", g.preparationSelected(), g.partyDeploy)
	}
	if len(g.partyMembers) != 4 || !g.partyMembers[30] {
		t.Fatalf("preparation changed permanent JOIN roster: %#v", g.partyMembers)
	}

	sc := &battle.Scenario{Party: []battle.PartyMember{{Fig: 0, Name: "索爾"}, {Fig: 9, Name: "悠妮"}, {Fig: 4, Name: "亞雷斯"}}}
	filterScenarioParty(sc, map[int]bool{0: true, 4: true})
	if len(sc.Party) != 2 || sc.Party[0].Fig != 0 || sc.Party[1].Fig != 4 {
		t.Fatalf("battle deployment filter=%#v", sc.Party)
	}
}
