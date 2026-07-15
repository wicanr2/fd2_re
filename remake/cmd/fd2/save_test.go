package main

import (
	"encoding/json"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestSaveDataRoundTripsPersistentParty(t *testing.T) {
	want := saveData{
		Node: "story_ch02", Flags: map[string]bool{"won_ch01": true}, Gold: 321,
		PartyMembers: map[int]bool{0: true, 9: true}, PartyJoinOrder: []int{0, 9},
		PartyRoster: map[int]battle.Unit{
			9: {Fig: 9, Name: "悠妮", Lv: 4, HP: 23, MaxHP: 37, MP: 18, MaxMP: 24, Exp: 67.5, Spells: []int{0, 4, 13}},
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
	if yuni.Fig != 9 || yuni.Lv != 4 || yuni.HP != 23 || yuni.MaxHP != 37 || yuni.MP != 18 || yuni.MaxMP != 24 || yuni.Exp != 67.5 || len(yuni.Spells) != 3 {
		t.Fatalf("party roster did not round-trip: %#v", yuni)
	}
}
