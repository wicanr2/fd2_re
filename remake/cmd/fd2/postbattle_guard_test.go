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
