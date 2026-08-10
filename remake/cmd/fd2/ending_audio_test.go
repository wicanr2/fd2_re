package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestEnterEndingStopsUnverifiedPreviousBGM(t *testing.T) {
	c := &campaign.Campaign{
		Start: "ending",
		Nodes: map[string]*campaign.Node{
			"ending": {Type: "ending", Text: "完"},
		},
	}
	g := &Game{
		camp:   campaign.NewRunner(c),
		bgmCur: "FDMUS_019",
	}
	g.enterNode()
	if g.bgmCur != "" || g.bgm != nil {
		t.Fatalf("ending kept previous BGM: track=%q player=%v", g.bgmCur, g.bgm)
	}
}
