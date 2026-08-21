package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func event63GameFixture(t *testing.T) *Game {
	t.Helper()
	fdotherPath, err := filepath.Abs("../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("使用者持有的原版 FDOTHER.DAT 不在測試掛載中")
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
	// Direct chapter starts have no preceding JOIN/save history. Build Keli's
	// persistent record through the same hash-bound sub_112A5 table used by a
	// normal JOIN; ch27.json's approximate HP must not become raw provenance.
	scenario, err := battle.LoadScenario(assetPath("assets/scenarios/ch27.json"))
	if err != nil {
		t.Fatal(err)
	}
	var persistentKeli battle.Unit
	joinTable, err := campaign.LoadNativeJoinConstructorTable(assetPath("assets/data/native_join_constructor.json"))
	if err != nil {
		t.Fatal(err)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(assetPath("assets/data/native_item_effect_rows.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, unit := range scenario.PartyUnits(nil) {
		if unit != nil && unit.Fig == 12 {
			persistentKeli, err = joinTable.MaterializePersistentUnit(12, *unit, itemRows)
			if err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	if !persistentKeli.HasNativeRecordWord42 || persistentKeli.NativeRecordWord42 != 151 {
		t.Fatal("chapter27 fixture has no Keli party member")
	}
	g := &Game{
		partyRoster:              map[int]battle.Unit{12: persistentKeli},
		nativeJoinConstructor:    joinTable,
		hasNativeJoinConstructor: true,
	}
	if err := g.loadMap(assetPath("assets/maps/map26")); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map26/map26_units.json", "assets/scenarios/ch27.json")
	if g.loadErr != "" || g.st == nil || g.sc == nil {
		t.Fatalf("chapter27 fixture err=%q state=%v scenario=%v", g.loadErr, g.st != nil, g.sc != nil)
	}
	c, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !g.materializeNativeMapRuntime(c.Nodes["battle_ch27"]) {
		t.Fatalf("chapter27 native runtime: %s", g.loadErr)
	}
	g.st.NativeTurnEventControls[0] = battle.NativeTurnEventControl{
		Turn: byte(g.st.NativeRoundCounter), EventID: 63, RawCamp: 0,
	}
	return g
}

func countActiveGroup(st *battle.State, group int) int {
	count := 0
	for _, unit := range st.Units {
		if unit != nil && unit.Group == group {
			count++
		}
	}
	return count
}

func finishNativeTurnStagingPan(t *testing.T, g *Game, screen *ebiten.Image) {
	t.Helper()
	for steps := 0; g.camPan != nil && steps < 100; steps++ {
		g.stepCamPan()
		if g.nativeTurnStaging != nil {
			if g.nativeTurnStaging.indexed {
				if !g.drawNativeTurnStaging(screen) {
					t.Fatalf("staging pan draw failed: %s", g.loadErr)
				}
			} else {
				g.Draw(screen)
			}
		}
	}
	if g.camPan != nil || g.loadErr != "" {
		t.Fatalf("staging pan did not finish: pan=%v err=%q", g.camPan != nil, g.loadErr)
	}
}

func advanceNativeTurnStagingFlash(t *testing.T, g *Game, screen *ebiten.Image) {
	t.Helper()
	for i := 0; i < nativeDelayTicks(300)+1; i++ {
		g.stepNativeTurnStaging()
	}
	job := g.nativeTurnStaging
	if job == nil || job.phase != nativeTurnStagingFlash {
		t.Fatalf("staging flash phase=%v err=%q", job, g.loadErr)
	}
	if job.indexed {
		for index, c := range job.palette {
			r, green, b, _ := c.RGBA()
			if r != 0xffff || green != 0xffff || b != 0xffff {
				t.Fatalf("white flash palette[%d]=%#x,%#x,%#x", index, r, green, b)
			}
		}
		if !g.drawNativeTurnStaging(screen) {
			t.Fatalf("staging flash draw failed: %s", g.loadErr)
		}
	} else {
		g.Draw(screen)
		if !g.nativeFullDACWhite || !job.drawn {
			t.Fatalf("legacy full-DAC white cover flag=%v drawn=%v", g.nativeFullDACWhite, job.drawn)
		}
	}
	for i := 0; i < nativeDelayTicks(200)+1; i++ {
		g.stepNativeTurnStaging()
	}
}

func TestEvent63RunsBeforeEnemyAIWithTwoAtomicStagingCalls(t *testing.T) {
	g := event63GameFixture(t)
	if _, ok := g.nativeMapHUDInput(); !ok {
		u := g.st.UnitAt(g.curX, g.curY)
		t.Fatalf("chapter27 HUD input unavailable at (%d,%d): unit=%+v", g.curX, g.curY, u)
	}
	beforeUnits := len(g.st.Units)
	beforeRoster := len(g.st.Roster)
	if countActiveGroup(g.st, 1) != 0 || countActiveGroup(g.st, 2) != 0 {
		t.Fatal("event63 groups were active before their live row")
	}
	g.endTurn()
	if g.nativeTurnStaging == nil || g.loadErr != "" {
		t.Fatalf("end-turn event63 job=%v err=%q", g.nativeTurnStaging, g.loadErr)
	}
	if !g.nativeTurnStaging.indexed {
		t.Fatal("production ch27 HUD provenance did not admit event63 indexed DAC")
	}
	if g.aiBusy || len(g.st.Units) != beforeUnits || len(g.st.Roster) != beforeRoster {
		t.Fatal("event63 changed roster or started AI before the first pan completed")
	}

	screen := ebiten.NewImage(640, 400)
	finishNativeTurnStagingPan(t, g, screen)
	if countActiveGroup(g.st, 1) == 0 || countActiveGroup(g.st, 2) != 0 || g.aiBusy {
		t.Fatalf("first staging publish group1=%d group2=%d ai=%v", countActiveGroup(g.st, 1), countActiveGroup(g.st, 2), g.aiBusy)
	}
	advanceNativeTurnStagingFlash(t, g, screen)
	if g.nativeTurnStaging == nil || g.nativeTurnStaging.call != 1 || g.camPan == nil {
		t.Fatalf("second staging call did not start: job=%#v pan=%v", g.nativeTurnStaging, g.camPan != nil)
	}

	finishNativeTurnStagingPan(t, g, screen)
	if countActiveGroup(g.st, 2) == 0 || g.aiBusy {
		t.Fatalf("second staging publish group2=%d ai=%v", countActiveGroup(g.st, 2), g.aiBusy)
	}
	advanceNativeTurnStagingFlash(t, g, screen)
	if g.nativeTurnStaging != nil || !g.aiBusy || g.loadErr != "" {
		t.Fatalf("event63 completion job=%v ai=%v err=%q", g.nativeTurnStaging, g.aiBusy, g.loadErr)
	}
}

func TestEvent63PreflightRejectsBadSecondCallWithoutMutation(t *testing.T) {
	g := event63GameFixture(t)
	beforeUnits := cloneBattleUnitPointers(g.st.Units)
	corrupted := false
	for _, unit := range g.st.Roster {
		if unit != nil && unit.Group == 2 {
			unit.HasNativePositionRecord = false
			corrupted = true
			break
		}
	}
	if !corrupted {
		t.Fatal("chapter27 fixture has no pending group2 row")
	}
	beforeRoster := cloneBattleUnitPointers(g.st.Roster)
	started, err := g.startNativeRawCamp0TurnEvents()
	if err == nil || started {
		t.Fatalf("bad event63 call started=%v err=%v", started, err)
	}
	if !reflect.DeepEqual(g.st.Units, beforeUnits) || !reflect.DeepEqual(g.st.Roster, beforeRoster) || g.aiBusy || g.nativeTurnStaging != nil {
		t.Fatal("rejected event63 call mutated battle state")
	}
}

func TestEvent63UnknownMatchingRowFailsClosedBeforeAI(t *testing.T) {
	g := event63GameFixture(t)
	g.st.NativeTurnEventControls[1] = battle.NativeTurnEventControl{
		Turn: byte(g.st.NativeRoundCounter), EventID: 65, RawCamp: 0,
	}
	before := cloneBattleUnitPointers(g.st.Units)
	g.endTurn()
	if g.loadErr == "" || g.aiBusy || g.nativeTurnStaging != nil || !reflect.DeepEqual(g.st.Units, before) {
		t.Fatalf("unknown row err=%q ai=%v job=%v", g.loadErr, g.aiBusy, g.nativeTurnStaging)
	}
}

func TestEvent74PreflightResolvesOneDynamicGroupAtomically(t *testing.T) {
	g := &Game{}
	if err := g.loadMap(assetPath("assets/maps/map28")); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map28/map28_units.json", "assets/scenarios/ch29.json")
	if g.loadErr != "" || g.st == nil || g.sc == nil {
		t.Fatalf("chapter29 fixture err=%q", g.loadErr)
	}
	c, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !g.materializeNativeMapRuntime(c.Nodes["battle_ch29"]) {
		t.Fatalf("chapter29 native runtime: %s", g.loadErr)
	}
	g.st.NativeRoundCounter = 8
	g.st.NativeEventState[16] = 4
	g.st.NativeTurnEventControls[0] = battle.NativeTurnEventControl{Turn: 8, EventID: 74, RawCamp: 0}
	beforeUnits := cloneBattleUnitPointers(g.st.Units)
	resolved, states, _, _, err := g.preflightNativeTurnStaging(g.sc.NativeTurnEvents[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Staging.Calls) != 1 || resolved.Staging.Calls[0].Group != 4 || len(states) != 1 {
		t.Fatalf("event74 resolved=%#v states=%d", resolved.Staging.Calls, len(states))
	}
	got := states[0]
	if countActiveGroup(got, 4) == 0 || got.NativeEventState[16] != 5 ||
		got.NativeTurnEventControls[0] != (battle.NativeTurnEventControl{Turn: 9, EventID: 74, RawCamp: 0}) {
		t.Fatalf("event74 snapshot group4=%d state16=%d row=%#v", countActiveGroup(got, 4), got.NativeEventState[16], got.NativeTurnEventControls[0])
	}
	if !reflect.DeepEqual(g.st.Units, beforeUnits) || g.st.NativeEventState[16] != 4 {
		t.Fatal("event74 preflight mutated the published state")
	}
}
