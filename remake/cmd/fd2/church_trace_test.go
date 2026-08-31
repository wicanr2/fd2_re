package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestCampaignTownChurchReviveReturnTrace(t *testing.T) {
	catalog, err := loadOfficialLocale("zh-Hant")
	if err != nil {
		t.Fatal(err)
	}
	c := &campaign.Campaign{
		Start: "town_ch02",
		Flags: map[string]bool{},
		Nodes: map[string]*campaign.Node{
			"town_ch02": {Type: "town", Options: []campaign.Option{
				{Label: "武器店", To: "shop"},
				{Label: "教會", To: "church_ch02"},
			}},
			"shop":        {Type: "shop", Next: "town_ch02"},
			"church_ch02": {Type: "church", Next: "town_ch02"},
		},
	}
	dead := battle.Unit{
		Name:                 "亞雷斯",
		ClassID:              1,
		NativeRecordClass:    1,
		HasNativeRecordClass: true,
		NativeRecordByte5:    1,
		HasNativeRecordByte5: true,
		Lv:                   3,
		HP:                   0,
		MaxHP:                24,
		OnField:              false,
	}
	g := &Game{
		camp:           campaign.NewRunner(c),
		gold:           100,
		reviveFeeRates: []int{0, 7},
		partyRoster:    map[int]battle.Unit{0: dead},
		partyMembers:   map[int]bool{0: true},
		partyJoinOrder: []int{0},
		localeCatalog:  catalog,
	}
	// town options: down→church, enter→opt1.
	g.stepCampaignMenu(campaign.MenuDown)
	selected, confirm := g.stepCampaignMenu(campaign.MenuConfirm)
	if selected != 1 || !confirm || g.camp.Advance("opt1") != "church_ch02" {
		t.Fatalf("town→church trace=(%d,%v,%q)", selected, confirm, g.camp.NodeID())
	}
	g.churchMode, g.churchIDs = "revive", []int{0}
	if !g.reviveChurchUnit(0) {
		t.Fatalf("revive failed: msg=%q", g.msg)
	}
	if g.gold != 79 || g.partyRoster[0].HP != 24 || !g.partyRoster[0].OnField {
		t.Fatalf("revive result gold=%d unit=%#v", g.gold, g.partyRoster[0])
	}
	g.leaveChurch()
	if got := g.camp.NodeID(); got != "town_ch02" {
		t.Fatalf("church return node=%q, want town_ch02", got)
	}
}

func TestReviveFailsClosedBeforeMutationWithoutLocaleCatalog(t *testing.T) {
	dead := battle.Unit{
		Name: "亞雷斯", Lv: 3, HP: 0, MaxHP: 24, OnField: false,
		NativeRecordClass: 1, HasNativeRecordClass: true,
		NativeRecordByte5: 1, HasNativeRecordByte5: true,
	}
	g := &Game{
		gold: 100, reviveFeeRates: []int{0, 7},
		partyRoster: map[int]battle.Unit{0: dead},
	}
	if g.reviveChurchUnit(0) {
		t.Fatal("revive succeeded without official locale catalog")
	}
	got := g.partyRoster[0]
	if g.gold != 100 || got.HP != 0 || got.OnField {
		t.Fatalf("failed locale gate leaked revive mutation: gold=%d unit=%#v", g.gold, got)
	}
}
