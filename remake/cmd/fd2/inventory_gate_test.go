package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestPartyHasItemIDMatchesOriginalSixteenRuntimeSlotSearch(t *testing.T) {
	units := make([]*battle.Unit, 17)
	units[4] = &battle.Unit{Camp: battle.Enemy, OnField: false, Inventory: []int{0x64}}
	units[16] = &battle.Unit{Camp: battle.Own, Inventory: []int{0x65}}
	g := &Game{st: &battle.State{Units: units}}
	if !g.partyHasItemID(0x64) {
		t.Fatal("original lookup must include inactive/non-player runtime slots")
	}
	if g.partyHasItemID(0x65) {
		t.Fatal("original lookup must stop after runtime slot 15")
	}

	g.st = nil
	g.partyRoster = map[int]battle.Unit{9: {Inventory: []int{0x64}}}
	if !g.partyHasItemID(0x64) {
		t.Fatal("persistent roster fallback must preserve a node-boundary item lookup")
	}
}

func TestPersistentRosterCleanupRestoresNativeMaxFields(t *testing.T) {
	g := &Game{partyRoster: map[int]battle.Unit{3: {
		HP: 0, MaxHP: 41, MP: 2, MaxMP: 9, Acted: true, OnField: false,
		OffX: 3, OffY: 4, Poisoned: true, PoisonTurns: 2, BuffAPPct: 25,
	}}}
	g.resetPersistentRosterState()
	u := g.partyRoster[3]
	if u.HP != 41 || u.MP != 9 || u.Acted || u.OffX != 0 || u.OffY != 0 || u.Poisoned || u.BuffAPPct != 0 {
		t.Fatalf("persistent cleanup=%#v", u)
	}
}

func TestInventoryGateSkyKeyRoutesThroughSyncThenPreparation(t *testing.T) {
	itemID, chapter := 0x64, 27
	c := &campaign.Campaign{Start: "gate", Nodes: map[string]*campaign.Node{
		"gate": {
			Type: "inventory_gate", ItemID: &itemID,
			IfPresent: "success", IfMissing: "bad",
		},
		"success": {
			Type: "cutscene", Next: "prep",
			Beats: []campaign.Beat{{Op: "sync_party"}, {Op: "set_chapter", Chapter: &chapter}},
		},
		"prep": {Type: "preparation", Next: "next"},
		"next": {Type: "story"},
		"bad":  {Type: "ending", Text: "bad ending"},
	}}
	g := &Game{
		camp:         campaign.NewRunner(c),
		partyMembers: map[int]bool{0: true},
		st: &battle.State{Units: []*battle.Unit{
			{Camp: battle.Own, Fig: 0, HP: 9, MaxHP: 40, MP: 2, MaxMP: 8, OnField: true, Inventory: []int{0x64}},
		}},
	}
	g.enterNode()
	if g.camp.Cur != "success" || g.fade == nil || g.handlerChapter != 27 || g.loadErr != "" {
		t.Fatalf("present gate did not execute editable success tail: node=%q fade=%#v chapter=%d err=%q", g.camp.Cur, g.fade, g.handlerChapter, g.loadErr)
	}
	if got := g.partyRoster[0]; got.HP != 40 || len(got.Inventory) != 1 || got.Inventory[0] != 0x64 {
		t.Fatalf("sky-key success did not sync persistent party: %#v", got)
	}
	g.tick(storyFadeFrames)
	if g.camp.Cur != "prep" || g.camp.Node().Type != "preparation" || g.st != nil {
		t.Fatalf("sky-key path must stop at chapter28 preparation: node=%q state=%#v", g.camp.Cur, g.st)
	}
}

func TestInventoryGateMissingSkyKeyReachesBadEndingWithoutSync(t *testing.T) {
	itemID := 0x64
	c := &campaign.Campaign{Start: "gate", Nodes: map[string]*campaign.Node{
		"gate": {Type: "inventory_gate", ItemID: &itemID, IfPresent: "prep", IfMissing: "bad"},
		"prep": {Type: "preparation"},
		"bad":  {Type: "ending", Text: "bad ending"},
	}}
	g := &Game{
		camp: campaign.NewRunner(c),
		st: &battle.State{Units: []*battle.Unit{
			{Camp: battle.Own, Fig: 0, HP: 9, MaxHP: 40, OnField: true, Inventory: []int{1, 2, 3}},
		}},
	}
	g.enterNode()
	if g.camp.Cur != "bad" || g.camp.Node().Type != "ending" || g.st != nil {
		t.Fatalf("missing gate = node %q state %#v, want clean bad ending", g.camp.Cur, g.st)
	}
	if len(g.partyRoster) != 0 {
		t.Fatalf("bad ending must not invent a successful sync: %#v", g.partyRoster)
	}
}
