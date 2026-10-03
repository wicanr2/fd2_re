package main

import (
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestNativePreparationRequiredFrontPreservesMembersAndPublishesAtomically(t *testing.T) {
	graph := &campaign.Campaign{
		Start: "prep",
		Nodes: map[string]*campaign.Node{
			"prep": {Type: "preparation", RequiredPartyIdentities: []int{21}, PartyFrontIdentities: []int{21}},
		},
	}
	g := &Game{
		camp:           campaign.NewRunner(graph),
		partyJoinOrder: []int{100, 109, 104, 121},
		partyDeploy:    map[int]bool{109: true, 121: true},
		partyRoster: map[int]battle.Unit{
			100: {NativeIdentity: 0, HasNativeIdentity: true, HP: 31},
			109: {NativeIdentity: 9, HasNativeIdentity: true, HP: 32},
			104: {NativeIdentity: 4, HasNativeIdentity: true, HP: 0},
			121: {NativeIdentity: 21, HasNativeIdentity: true, HP: 33},
		},
	}
	before := make(map[int]battle.Unit, len(g.partyRoster))
	for id, unit := range g.partyRoster {
		before[id] = unit
	}
	if err := g.frontNativePreparationRequiredRoster(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.partyJoinOrder, []int{100, 121, 109, 104}) ||
		!reflect.DeepEqual(g.partyRoster, before) || g.partyDeploy[104] {
		t.Fatal("required front changed fixed leader, member data, unselected order or deployment")
	}
	// A later missing identity must not publish the earlier successful move.
	graph.Nodes["prep"].PartyFrontIdentities = []int{9, 30}
	order := append([]int(nil), g.partyJoinOrder...)
	if err := g.frontNativePreparationRequiredRoster(); err == nil ||
		!reflect.DeepEqual(g.partyJoinOrder, order) || !reflect.DeepEqual(g.partyRoster, before) {
		t.Fatal("invalid required front partially published")
	}
	graph.Nodes["prep"].PartyFrontIdentities = []int{4}
	if err := g.frontNativePreparationRequiredRoster(); err == nil || !reflect.DeepEqual(g.partyJoinOrder, order) {
		t.Fatal("unselected member accepted as required front")
	}
}
