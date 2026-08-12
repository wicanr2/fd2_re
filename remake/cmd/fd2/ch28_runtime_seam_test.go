package main

import (
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func seedLateCampaignParty(t *testing.T, scenarioPath string, selected int) *Game {
	t.Helper()
	scenario, err := battle.LoadScenario(assetPath(scenarioPath))
	if err != nil {
		t.Fatal(err)
	}
	if selected <= 0 || len(scenario.Party) < selected {
		t.Fatalf("%s party=%d, want at least %d", scenarioPath, len(scenario.Party), selected)
	}
	order := make([]int, 0, len(scenario.Party))
	for _, unit := range scenario.Party {
		order = append(order, unit.Fig)
	}
	g := &Game{
		partyMembers:   make(map[int]bool, len(order)),
		partyJoinOrder: append([]int(nil), order...),
		partyDeploy:    make(map[int]bool, selected-1),
	}
	for _, id := range order {
		g.partyMembers[id] = true
	}
	for _, id := range order[1:selected] {
		g.partyDeploy[id] = true
	}
	return g
}

// newChapter28RuntimeBattle loads the authored map/unit/scenario files.  It
// deliberately does not fabricate a party or call Runner.Advance: the test
// below must enter the postbattle handler through the production result
// confirmation seam.
func newChapter28RuntimeBattle(t *testing.T) *Game {
	t.Helper()
	scenario, err := battle.LoadScenario(assetPath("assets/scenarios/ch28.json"))
	if err != nil {
		t.Fatal(err)
	}
	// ch28.json is the authored roster; the map's twenty deployment cells are
	// the only direct-start party slice.  Seed the persistent records through
	// the same LOADCH boundary helper used by the production campaign, rather
	// than constructing a roster from guessed stats.
	order := make([]int, 0, len(scenario.Party))
	for _, unit := range scenario.Party {
		order = append(order, unit.Fig)
	}
	if len(order) < 20 {
		t.Fatalf("ch28 authored party=%d, want at least 20 deployment records", len(order))
	}
	g := seedLateCampaignParty(t, "assets/scenarios/ch28.json", 20)
	if err := g.loadMap(assetPath("assets/maps/map27")); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map27/map27_units.json", "assets/scenarios/ch28.json")
	if g.loadErr != "" || g.st == nil || g.sc == nil {
		t.Fatalf("ch28 formal runtime setup err=%q state=%v scenario=%v", g.loadErr, g.st != nil, g.sc != nil)
	}
	deployed := append([]int{order[0]}, order[1:20]...)
	loaded := make([]*battle.Unit, 0, len(deployed))
	for _, id := range deployed {
		for _, unit := range g.st.Units {
			if unit != nil && unit.Fig == id && unit.Camp == battle.Own {
				loaded = append(loaded, unit)
				break
			}
		}
	}
	if len(loaded) != len(deployed) {
		t.Fatalf("ch28 runtime authored party records=%d, want %d", len(loaded), len(deployed))
	}
	if err := g.seedPersistentPartyFromLoadCH(deployed, loaded); err != nil {
		t.Fatalf("ch28 production party seed: %v", err)
	}
	full, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	full.Start = "battle_ch28"
	g.camp = campaign.NewRunner(full)
	return g
}

func TestLatePreHandlersReachTheirPlayerNumberedBattles(t *testing.T) {
	if os.Getenv("FD2_ORIGINAL_FDOTHER") == "" {
		t.Skip("late pre-handler indexed transitions require the read-only original FDOTHER/FDSHAP/FDICON bundle")
	}
	for _, tc := range []struct {
		name, start, wantBattle, scenario string
		wantRawRoster                     int
	}{
		{
			name: "player battle 28 uses raw index 27", start: "story_ch28", wantBattle: "battle_ch28",
			scenario: "assets/scenarios/ch28.json", wantRawRoster: 60,
		},
		{
			name: "player battle 29 uses raw index 28", start: "story_ch29", wantBattle: "battle_ch29",
			scenario: "assets/scenarios/ch29.json", wantRawRoster: 76,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := seedLateCampaignParty(t, tc.scenario, 20)
			full, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
			if err != nil {
				t.Fatal(err)
			}
			full.Start = tc.start
			g.camp = campaign.NewRunner(full)
			g.enterNode()
			if g.loadErr != "" {
				t.Fatalf("%s handler entry: %s", tc.start, g.loadErr)
			}
			if len(g.storyRoster) != tc.wantRawRoster {
				t.Fatalf("%s raw LOADCH roster=%d want %d", tc.start, len(g.storyRoster), tc.wantRawRoster)
			}
			if err := g.fastForwardShotCampaign(); err != nil {
				t.Fatal(err)
			}
			if g.loadErr != "" || g.camp.NodeID() != tc.wantBattle || g.st == nil {
				t.Fatalf("%s boundary node=%q state=%v err=%q", tc.start, g.camp.NodeID(), g.st != nil, g.loadErr)
			}
		})
	}
}

func TestChapter27PreReactivatesOnlyNonzeroHPPartyRecords(t *testing.T) {
	actors := make([]battle.Unit, 20)
	for i := range actors {
		actors[i] = battle.Unit{
			Camp: battle.Own, HP: 10, OnField: false,
			NativeRecordByte5: 1, HasNativeRecordByte5: true,
		}
	}
	actors[7].HP = 0
	g := newBeatTestGame(t, []campaign.Beat{{Op: "reactivate_nonzero_hp", Source: "0x33cea", Count: 20}})
	g.storyActors = actors
	g.beatAdvance()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	for i := range g.storyActors {
		wantActive := i != 7
		if g.storyActors[i].OnField != wantActive || (wantActive && g.storyActors[i].NativeRecordByte5 != 0) || (!wantActive && g.storyActors[i].NativeRecordByte5 != 1) {
			t.Fatalf("slot%d active/raw5=%v/%d want %v", i, g.storyActors[i].OnField, g.storyActors[i].NativeRecordByte5, wantActive)
		}
	}
}

func TestChapter27PreReactivationPreflightIsAtomic(t *testing.T) {
	actors := make([]battle.Unit, 20)
	for i := range actors {
		actors[i] = battle.Unit{Camp: battle.Own, HP: 10, NativeRecordByte5: 1, HasNativeRecordByte5: true}
	}
	actors[19].HasNativeRecordByte5 = false
	g := newBeatTestGame(t, []campaign.Beat{{Op: "reactivate_nonzero_hp", Source: "0x33cea", Count: 20}})
	g.storyActors = actors
	g.beatAdvance()
	if g.loadErr == "" {
		t.Fatal("missing raw byte+5 provenance was accepted")
	}
	for i := 0; i < 19; i++ {
		if g.storyActors[i].OnField || g.storyActors[i].NativeRecordByte5 != 1 {
			t.Fatalf("preflight failure partially mutated slot%d", i)
		}
	}
}

// TestChapter28BattleResultRunsCh27PostToPreparation29 verifies the complete
// authored ch28 result boundary.  It uses faithful mode and the production
// confirmation method, then consumes the real dialogue pages before letting
// the handler's sync_party/set_chapter beats finish.  No native renderer or
// guessed runtime frontier is substituted because ch27_post has no such
// operation.
func TestChapter28BattleResultRunsCh27PostToPreparation29(t *testing.T) {
	g := newChapter28RuntimeBattle(t)
	if g.camp.NodeID() != "battle_ch28" {
		t.Fatalf("formal battle cursor=%q, want battle_ch28", g.camp.NodeID())
	}

	// The authored scenario is the runtime input.  Marking the already loaded
	// battle result is the same seam used after the real enemy phase; it does
	// not alter units, party membership, or handler data.
	g.result = "win"
	if !g.confirmBattleResult() || g.result != "" {
		t.Fatalf("battle result confirmation failed: node=%q result=%q err=%q", g.camp.NodeID(), g.result, g.loadErr)
	}
	if g.camp.NodeID() != "postbattle_ch28_persist" {
		t.Fatalf("ch28 result entered %q, want postbattle_ch28_persist", g.camp.NodeID())
	}
	if g.loadErr != "" || len(g.beats) != 3 {
		t.Fatalf("ch27_post admission beats=%d err=%q", len(g.beats), g.loadErr)
	}

	// Consume the actual ch28 dialogue through dlgAdvance, including any page
	// scroll delay, rather than clearing the dialog queue as a test shortcut.
	for steps := 0; len(g.dialog) > 0 && steps < 100; steps++ {
		// Update() decrements this timer once per frame.  Keep the same
		// production timer semantics here; tick() intentionally covers only
		// beat jobs and does not own the dialogue lifecycle.
		for g.dlgScrollT > 0 {
			g.dlgScrollT--
		}
		if g.loadErr != "" {
			t.Fatalf("ch27_post dialogue setup failed: %s", g.loadErr)
		}
		g.dlgAdvance()
	}
	if len(g.dialog) != 0 {
		t.Fatalf("ch27_post dialogue did not finish: remaining=%d page=%d scroll=%d", len(g.dialog), g.dlgPage, g.dlgScrollT)
	}
	// The dialog beat remains the active beat until the same callback used by
	// the normal input path advances it.  This consumes sync_party and then
	// set_chapter without bypassing either authored operation.
	g.beatAdvance()
	if g.loadErr != "" {
		t.Fatalf("ch27_post sync/set-chapter failed: beat=%d err=%q", g.beatIdx, g.loadErr)
	}

	// The handler's final beat starts the normal story fade.  Advance it by
	// ticking the real fade state; the callback enters the editable preparation
	// boundary and clears transient battle state.
	for ticks := 0; g.camp.NodeID() == "postbattle_ch28_persist" && ticks < 120; ticks++ {
		g.tick(1)
	}
	if g.loadErr != "" {
		t.Fatalf("ch27_post transition failed: %s", g.loadErr)
	}
	if g.camp.NodeID() != "preparation_ch29" || g.handlerChapter != 28 {
		t.Fatalf("ch28 postbattle boundary node=%q chapter=%d beat=%d/%d fade=%v", g.camp.NodeID(), g.handlerChapter, g.beatIdx, len(g.beats), g.fade != nil)
	}
	if g.st != nil {
		t.Fatal("preparation_ch29 retained transient battle state")
	}
	if len(g.partyRoster) == 0 {
		t.Fatal("ch27_post sync_party did not publish any authored party record")
	}
	for id, unit := range g.partyRoster {
		if unit.Camp != battle.Own {
			t.Fatalf("persistent roster id=%d contains non-player camp=%v", id, unit.Camp)
		}
	}
}
