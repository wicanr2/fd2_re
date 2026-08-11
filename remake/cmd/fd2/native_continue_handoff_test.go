package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

func TestNativeContinueBattlePublicationFromRealCurrentSnapshot(t *testing.T) {
	const (
		unitsPath    = "assets/maps/map0/map0_units.json"
		scenarioPath = "assets/scenarios/ch01.json"
	)
	savePath := filepath.Join(
		"../../../org_game/炎龍騎士團/FLAME2", "FD2.SAV",
	)
	stored, err := os.ReadFile(savePath)
	if err != nil {
		t.Skipf("player-provided FD2.SAV is absent: %v", err)
	}
	plain, err := fdsave.Decode(stored)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := fdsave.InspectCurrentSnapshot(plain)
	if err != nil {
		t.Fatal(err)
	}
	assetState, err := battle.Load(assetPath(unitsPath))
	if err != nil {
		t.Fatal(err)
	}
	scenario, err := battle.LoadScenario(assetPath(scenarioPath))
	if err != nil {
		t.Fatal(err)
	}
	input, err := fdsave.BuildContinueRuntimeInput(
		snapshot,
		fdsave.ContinueRuntimeContext{
			Chapter:            int(snapshot.Header.Chapter),
			FieldWidth:         assetState.W,
			FieldHeight:        assetState.H,
			SelectorGroupCount: 0x100,
			// The signed BIOS tick is deliberately supplied by the caller;
			// this deterministic contract test uses a valid zero seed and does
			// not claim title-clock or pixel-timing E2.
			TitleTimerTick: 0, HasTitleTimerTick: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	state := &battle.State{
		W: assetState.W, H: assetState.H,
		OwnDeploy: append([]battle.Cell(nil), assetState.OwnDeploy...),
		Cost:      append([]int(nil), assetState.Cost...),
		NativeCompositionEventBytes: append(
			[]byte(nil), assetState.NativeCompositionEventBytes...,
		),
		NativeMapEventGrid:    append([]byte(nil), assetState.NativeMapEventGrid...),
		HasNativeMapEventGrid: assetState.HasNativeMapEventGrid,
		NativeFieldEventSlots: append(
			[]int(nil), assetState.NativeFieldEventSlots...,
		),
		NativeFieldEvents: append(
			[]battle.NativeFieldEvent(nil), assetState.NativeFieldEvents...,
		),
		NativeFieldEventRules: append(
			[]battle.NativeFieldEventRule(nil), assetState.NativeFieldEventRules...,
		),
		NativeTileBlitModes:  append([]byte(nil), assetState.NativeTileBlitModes...),
		NativeTerrainControl: append([]byte(nil), assetState.NativeTerrainControl...),
		NativeTerrainMoveCodes: append(
			[]byte(nil), assetState.NativeTerrainMoveCodes...,
		),
	}
	if err := campaign.MaterializeNativeContinueFieldBoundary(
		state, input, int(snapshot.Header.Chapter),
	); err != nil {
		t.Fatal(err)
	}
	catalog, err := campaign.LoadNativeCharacterCatalog(
		assetPath("assets/data/native_character_catalog.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := campaign.MaterializeNativeContinueRuntimeUnits(
		state, input, catalog,
	); err != nil {
		t.Fatal(err)
	}
	if err := campaign.MaterializeNativeContinueMapTiming(state, input); err != nil {
		t.Fatal(err)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(
		assetPath("assets/data/native_item_effect_rows.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := campaign.MaterializeNativeContinuePendingGroups(
		state, input, int(snapshot.Header.Chapter), assetState, scenario, itemRows,
	); err != nil {
		t.Fatal(err)
	}
	if err := campaign.MaterializeNativeContinueInteractiveBoundary(state, input); err != nil {
		t.Fatal(err)
	}

	graph := &campaign.Campaign{
		Start: "battle_ch01",
		Nodes: map[string]*campaign.Node{
			"battle_ch01": {
				Type: "battle", Units: unitsPath, Scenario: scenarioPath,
			},
		},
	}
	g := &Game{
		camp: campaign.NewRunner(graph),
		m: &MapData{
			W: assetState.W, H: assetState.H, TileW: 24, TileH: 24,
		},
		dialog: []battle.DialogLine{{Text: "stale"}},
	}
	if err := g.publishNativeContinueBattle(
		input, state, scenario, "battle_ch01", unitsPath, scenarioPath,
	); err != nil {
		t.Fatal(err)
	}
	if g.camp.NodeID() != "battle_ch01" || g.st != state || g.sc != scenario ||
		g.titlePhase != "" || len(g.dialog) != 0 || g.sel != nil ||
		g.curX != int(snapshot.Header.CursorX) ||
		g.curY != int(snapshot.Header.CursorY) {
		t.Fatalf("published CONTINUE state node=%q st=%p sc=%p title=%q cursor=(%d,%d)",
			g.camp.NodeID(), g.st, g.sc, g.titlePhase, g.curX, g.curY)
	}
}
