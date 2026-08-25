package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func loadFullCampaignAt(t *testing.T, node string) *Game {
	t.Helper()
	full, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	if full.Nodes[node] == nil {
		t.Fatalf("campaign_full missing node %q", node)
	}
	full.Start = node
	g := &Game{camp: campaign.NewRunner(full)}
	g.enterNode()
	if g.loadErr != "" {
		t.Fatalf("enter %s: %s", node, g.loadErr)
	}
	return g
}

func TestCampaignFullEarlyTownChurchReturn(t *testing.T) {
	g := loadFullCampaignAt(t, "town_ch02")
	if g.camp.Node().NativeTownVariant == nil {
		t.Fatal("town_ch02 lost its native town owner")
	}
	for g.campSel != 4 {
		if !g.moveNativeTownSelection(1) {
			t.Fatalf("town_ch02 could not move to church: selection=%d", g.campSel)
		}
	}
	g.camp.Advance("opt4")
	g.enterNode()
	if g.loadErr != "" || g.camp.NodeID() != "church_ch02" || g.churchMode != "menu" {
		t.Fatalf("town_ch02→church_ch02 node=%q mode=%q err=%q", g.camp.NodeID(), g.churchMode, g.loadErr)
	}
	g.leaveChurch()
	if g.loadErr != "" || g.camp.NodeID() != "town_ch02" {
		t.Fatalf("church_ch02→town_ch02 node=%q err=%q", g.camp.NodeID(), g.loadErr)
	}
}

func TestCampaignFullMiddleTownPreparationCancelAndConfirm(t *testing.T) {
	g := loadFullCampaignAt(t, "town_ch17")
	shared, err := loadNativeClassUIAssets()
	if err != nil {
		t.Skipf("player-provided original UI assets are absent: %v", err)
	}
	town, err := loadNativeTownUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeClassUI, g.nativeTownUI = shared, town

	enterPreparation := func() {
		t.Helper()
		g.campSel = 0
		if !g.moveNativeTownSelection(1) || !g.moveNativeTownSelection(1) || g.campSel != 2 {
			t.Fatalf("town_ch17 preparation selection=%d", g.campSel)
		}
		source, ok := g.composeNativeTownFrame()
		if !ok || len(source) != 320*200 {
			t.Fatal("town_ch17 native source frame is unavailable")
		}
		g.prepPromptSource = append([]byte(nil), source...)
		g.camp.Advance("opt2")
		g.enterNode()
		if g.loadErr != "" || g.camp.NodeID() != "preparation_ch17" || len(g.prepPromptSource) != 320*200 {
			t.Fatalf("town_ch17→preparation_ch17 node=%q source=%d err=%q", g.camp.NodeID(), len(g.prepPromptSource), g.loadErr)
		}
	}

	enterPreparation()
	g.camp.Advance("cancel")
	g.enterNode()
	if g.loadErr != "" || g.camp.NodeID() != "town_ch17" {
		t.Fatalf("preparation_ch17 cancel node=%q err=%q", g.camp.NodeID(), g.loadErr)
	}

	enterPreparation()
	if !g.acceptTownDeparturePrompt() {
		t.Fatal("empty representative roster unexpectedly entered selection mode")
	}
	g.camp.Advance("confirm")
	if g.camp.NodeID() != "story_ch17" {
		t.Fatalf("preparation_ch17 confirm node=%q", g.camp.NodeID())
	}
}
