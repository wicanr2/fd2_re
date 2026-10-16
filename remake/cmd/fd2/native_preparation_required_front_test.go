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

func TestChapter26RequiresBothMembersAndFrontsCompleteRosterInCallerOrder(t *testing.T) {
	graph, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	graph.Start = "preparation_ch26"
	g := &Game{
		camp: campaign.NewRunner(graph), partyJoinOrder: []int{0, 4, 9, 29},
		prepIDs: []int{4, 9, 29}, partyDeploy: map[int]bool{},
		partyRoster: map[int]battle.Unit{},
	}
	for _, id := range g.partyJoinOrder {
		g.partyRoster[id] = battle.Unit{
			NativeIdentity: id, HasNativeIdentity: true,
			NativeRecordByte8: byte(id), HasNativeRecordByte8: true, HP: 100 + id,
		}
	}
	for _, missing := range []int{9, 29} {
		g.partyDeploy = map[int]bool{4: true, 9: true, 29: true}
		delete(g.partyDeploy, missing)
		got, err := g.missingPreparationRequiredIdentity()
		if err != nil || got == nil || *got != missing {
			t.Fatalf("缺少角色%d沒有拒收：missing=%v err=%v", missing, got, err)
		}
	}
	g.partyDeploy = map[int]bool{9: true, 29: true}
	if missing, err := g.missingPreparationRequiredIdentity(); err != nil || missing != nil {
		t.Fatalf("兩名必出角色齊全仍拒收：missing=%v err=%v", missing, err)
	}
	before := make(map[int]battle.Unit, len(g.partyRoster))
	for id, unit := range g.partyRoster {
		before[id] = unit
	}
	if err := g.frontNativePreparationRequiredRoster(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.partyJoinOrder, []int{0, 29, 9, 4}) ||
		!reflect.DeepEqual(g.partyRoster, before) || g.partyDeploy[4] {
		t.Fatal("第26章必出前置未保留索爾、完整角色資料或未出戰隊員順序")
	}
}

func TestLatePreparationRequiresYuniAndFrontsCompleteRoster(t *testing.T) {
	for _, chapter := range []string{"27", "28", "29"} {
		t.Run(chapter, func(t *testing.T) {
			graph, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
			if err != nil {
				t.Fatal(err)
			}
			graph.Start = "preparation_ch" + chapter
			g := &Game{
				camp: campaign.NewRunner(graph), partyJoinOrder: []int{0, 29, 9, 4},
				prepIDs: []int{29, 9, 4}, partyDeploy: map[int]bool{29: true, 4: true},
				partyRoster: map[int]battle.Unit{},
			}
			for _, id := range g.partyJoinOrder {
				g.partyRoster[id] = battle.Unit{
					NativeIdentity: id, HasNativeIdentity: true,
					NativeRecordByte8: byte(id), HasNativeRecordByte8: true, HP: 100 + id,
				}
			}
			if missing, err := g.missingPreparationRequiredIdentity(); err != nil || missing == nil || *missing != 9 {
				t.Fatalf("悠妮缺席未拒收：missing=%v err=%v", missing, err)
			}
			g.partyDeploy = map[int]bool{9: true, 4: true}
			if missing, err := g.missingPreparationRequiredIdentity(); err != nil || missing != nil {
				t.Fatalf("悠妮出戰仍要求其他人：missing=%v err=%v", missing, err)
			}
			before := make(map[int]battle.Unit, len(g.partyRoster))
			for id, unit := range g.partyRoster {
				before[id] = unit
			}
			if err := g.frontNativePreparationRequiredRoster(); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(g.partyJoinOrder, []int{0, 9, 29, 4}) || !reflect.DeepEqual(g.partyRoster, before) || g.partyDeploy[29] {
				t.Fatal("第27章完整名冊前置改了欄位、索爾位置或未出戰隊員")
			}
		})
	}
}
