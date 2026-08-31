package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func newNativeChurchTransferPathGame(t *testing.T) *Game {
	t.Helper()
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	if _, err := os.Stat(filepath.Join(base, "FDOTHER.DAT")); err != nil {
		t.Skip("player-provided original resources are absent")
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(base, "DATO.DAT"))
	assets, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := loadOfficialLocale("zh-Hant")
	if err != nil {
		t.Fatal(err)
	}
	source := setNativeTransferInventory(*nativeItemPanelTestUnit(), 0, 1)
	source.NativeIdentity, source.HasNativeIdentity = 0, true
	source.MapSelectorKey, source.HasMapSelectorKey = 0, true
	destination := setNativeTransferInventory(*nativeItemPanelTestUnit())
	destination.NativeIdentity, destination.HasNativeIdentity = 9, true
	destination.MapSelectorKey, destination.HasMapSelectorKey = 9, true
	g := &Game{
		nativeClassUI: assets, nativeChurchTextIndex: 512,
		churchMode: "transfer_source", churchIDs: []int{0, 9},
		churchTransferSource: -1, churchTransferItem: -1, churchTransferDest: -1,
		partyJoinOrder: []int{0, 9},
		partyRoster:    map[int]battle.Unit{0: source, 9: destination},
		localeCatalog:  catalog,
	}
	return g
}

func drainNativeChurchTransferJobs(t *testing.T, g *Game) {
	t.Helper()
	for steps := 0; g.nativeClassUIJob != nil; steps++ {
		if steps >= 64 {
			t.Fatalf("church transfer job did not settle: mode=%q", g.churchMode)
		}
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
}

func enterNativeChurchTransferDestination(t *testing.T, g *Game) {
	t.Helper()
	if !g.handleNativeChurchTransferInput(nativeChurchTransferInput{enter: true}) {
		t.Fatal("source input")
	}
	drainNativeChurchTransferJobs(t, g)
	if g.churchMode != "transfer_item" {
		t.Fatalf("source mode=%q", g.churchMode)
	}
	if !g.handleNativeChurchTransferInput(nativeChurchTransferInput{enter: true}) {
		t.Fatal("item input")
	}
	drainNativeChurchTransferJobs(t, g)
	if g.churchMode != "transfer_dest" {
		t.Fatalf("item mode=%q", g.churchMode)
	}
}

func TestNativeChurchTransferTypedInputSuccessAndCancel(t *testing.T) {
	g := newNativeChurchTransferPathGame(t)
	want := g.partyRoster[0]
	enterNativeChurchTransferDestination(t, g)
	g.handleNativeChurchTransferInput(nativeChurchTransferInput{escape: true})
	drainNativeChurchTransferJobs(t, g)
	if g.churchMode != "transfer_source" || !reflect.DeepEqual(g.partyRoster[0], want) {
		t.Fatal("destination cancel changed source or failed to return")
	}

	enterNativeChurchTransferDestination(t, g)
	g.handleNativeChurchTransferInput(nativeChurchTransferInput{delta: 1})
	g.handleNativeChurchTransferInput(nativeChurchTransferInput{enter: true})
	drainNativeChurchTransferJobs(t, g)
	if g.churchMode != "transfer_source" ||
		!reflect.DeepEqual(g.partyRoster[0].Inventory, []int{1}) ||
		!reflect.DeepEqual(g.partyRoster[9].Inventory, []int{0}) {
		t.Fatalf("cross-character transfer result source=%#v destination=%#v",
			g.partyRoster[0], g.partyRoster[9])
	}
}

func TestNativeChurchTransferTypedInputFullIsAtomic(t *testing.T) {
	g := newNativeChurchTransferPathGame(t)
	g.partyRoster[9] = setNativeTransferInventory(g.partyRoster[9], 0, 0, 0, 0, 0, 0, 0, 0)
	wantSource, wantDestination := g.partyRoster[0], g.partyRoster[9]
	enterNativeChurchTransferDestination(t, g)
	g.handleNativeChurchTransferInput(nativeChurchTransferInput{delta: 1})
	g.handleNativeChurchTransferInput(nativeChurchTransferInput{enter: true})
	drainNativeChurchTransferJobs(t, g)
	if g.churchMode != "transfer_full" || !reflect.DeepEqual(g.partyRoster[0], wantSource) ||
		!reflect.DeepEqual(g.partyRoster[9], wantDestination) {
		t.Fatal("full destination leaked a transaction")
	}
	g.handleNativeChurchTransferInput(nativeChurchTransferInput{escape: true})
	drainNativeChurchTransferJobs(t, g)
	if g.churchMode != "transfer_source" {
		t.Fatalf("full return mode=%q", g.churchMode)
	}
}

func TestNativeChurchTransferTypedInputSelfReordersItem(t *testing.T) {
	g := newNativeChurchTransferPathGame(t)
	enterNativeChurchTransferDestination(t, g)
	g.handleNativeChurchTransferInput(nativeChurchTransferInput{enter: true})
	drainNativeChurchTransferJobs(t, g)
	got := g.partyRoster[0]
	if g.churchMode != "transfer_source" ||
		!reflect.DeepEqual(got.Inventory, []int{1, 0}) ||
		!reflect.DeepEqual(got.Equipped, []bool{false, false}) {
		t.Fatalf("self transfer mode=%q unit=%#v", g.churchMode, got)
	}
}
