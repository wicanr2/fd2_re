package main

import (
	"encoding/json"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestSaveDataRoundTripsPersistentParty(t *testing.T) {
	want := saveData{
		Node: "story_ch02", Flags: map[string]bool{"won_ch01": true}, Gold: 321,
		PartyMembers: map[int]bool{0: true, 9: true}, PartyJoinOrder: []int{0, 9},
		PartyRoster: map[int]battle.Unit{
			9: {Fig: 9, Name: "悠妮", Lv: 4, HP: 23, MaxHP: 37, MP: 18, MaxMP: 24, Exp: 67.5, Spells: []int{0, 4, 13}, Inventory: []int{0xc6}},
		},
		Chapter: 1,
	}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got saveData
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	yuni, ok := got.PartyRoster[9]
	if !ok || got.Node != want.Node || got.Chapter != 1 || !got.PartyMembers[0] || len(got.PartyJoinOrder) != 2 {
		t.Fatalf("campaign progress did not round-trip: %#v", got)
	}
	if yuni.Fig != 9 || yuni.Lv != 4 || yuni.HP != 23 || yuni.MaxHP != 37 || yuni.MP != 18 || yuni.MaxMP != 24 || yuni.Exp != 67.5 || len(yuni.Spells) != 3 || len(yuni.Inventory) != 1 || yuni.Inventory[0] != 0xc6 {
		t.Fatalf("party roster did not round-trip: %#v", yuni)
	}
}

func TestSaveRejectsPostBattleHandlerWithoutSerializableRuntimeContext(t *testing.T) {
	c := &campaign.Campaign{Start: "post", Nodes: map[string]*campaign.Node{
		"post":   {Type: "cutscene", HandlerBinding: "assets/cutscenes/bindings/ch01_post.json", Next: "choice"},
		"choice": {Type: "choice"},
	}}
	g := &Game{camp: campaign.NewRunner(c), st: &battle.State{Units: []*battle.Unit{{Fig: 0}}}}
	g.saveGame()
	if g.msg != "戰後演出進行中，請在下一個節點存檔" {
		t.Fatalf("unsafe postbattle save was not rejected: %q", g.msg)
	}
}
