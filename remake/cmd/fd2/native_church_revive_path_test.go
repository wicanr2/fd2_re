package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func newNativeChurchRevivePathGame(t *testing.T, gold int) (*Game, *ebiten.Image) {
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
	dead := *nativeItemPanelTestUnit()
	dead.Name, dead.HP, dead.MaxHP, dead.OnField = "亞雷斯", 0, 24, false
	dead.Lv = 3
	dead.NativeRecordByte5, dead.HasNativeRecordByte5 = 1, true
	dead.NativeRecordClass, dead.HasNativeRecordClass = 1, true
	dead.NativeIdentity, dead.HasNativeIdentity = 0, true
	dead.MapSelectorKey, dead.HasMapSelectorKey = 0, true
	g := &Game{
		nativeClassUI: assets, nativeChurchTextIndex: 589,
		churchMode: "revive", churchIDs: []int{0}, churchReviveID: -1,
		gold: gold, reviveFeeRates: []int{0, 7},
		partyJoinOrder: []int{0}, partyRoster: map[int]battle.Unit{0: dead},
	}
	return g, ebiten.NewImage(640, 400)
}

func drainNativeChurchReviveJobs(t *testing.T, g *Game, screen *ebiten.Image) {
	t.Helper()
	now := time.Unix(100, 0)
	for steps := 0; g.nativeClassUIJob != nil; steps++ {
		if steps >= 256 {
			t.Fatalf("revive job did not settle: mode=%q", g.churchMode)
		}
		job := g.nativeClassUIJob
		if len(job.timeline) != 0 {
			g.stepNativeClassUILifecycle(now)
			if !g.drawNativeClassUIJob(screen) {
				t.Fatal("revive timeline did not draw")
			}
			now = now.Add(time.Hour)
			g.stepNativeClassUILifecycle(now)
			if !g.drawNativeClassUIJob(screen) {
				t.Fatal("revive final timeline frame did not draw")
			}
			g.stepNativeClassUILifecycle(now)
			continue
		}
		if !g.drawNativeClassUIJob(screen) {
			t.Fatal("revive indexed frame did not draw")
		}
		g.stepNativeClassUILifecycle(now)
	}
}

func enterNativeChurchReviveConfirmation(t *testing.T, g *Game, screen *ebiten.Image) {
	t.Helper()
	if !g.handleNativeChurchReviveInput(nativeChurchReviveInput{enter: true}) {
		t.Fatal("revive list input")
	}
	drainNativeChurchReviveJobs(t, g, screen)
	if g.churchMode != "revive_confirm" || g.churchReviveFee != 21 {
		t.Fatalf("confirm mode=%q fee=%d", g.churchMode, g.churchReviveFee)
	}
}

func TestNativeChurchReviveTypedInputCancelAndInsufficientAreAtomic(t *testing.T) {
	g, screen := newNativeChurchRevivePathGame(t, 20)
	want := g.partyRoster[0]
	enterNativeChurchReviveConfirmation(t, g, screen)
	g.handleNativeChurchReviveInput(nativeChurchReviveInput{escape: true})
	drainNativeChurchReviveJobs(t, g, screen)
	if g.churchMode != "revive" || g.gold != 20 || !reflect.DeepEqual(g.partyRoster[0], want) {
		t.Fatal("revive cancel changed gold or unit")
	}
	enterNativeChurchReviveConfirmation(t, g, screen)
	g.handleNativeChurchReviveInput(nativeChurchReviveInput{enter: true})
	drainNativeChurchReviveJobs(t, g, screen)
	if g.churchMode != "revive_insufficient" || g.gold != 20 || !reflect.DeepEqual(g.partyRoster[0], want) {
		t.Fatal("insufficient revive changed gold or unit")
	}
}

func TestNativeChurchReviveTypedInputSuccessReachesEmptyThenMenu(t *testing.T) {
	g, screen := newNativeChurchRevivePathGame(t, 100)
	enterNativeChurchReviveConfirmation(t, g, screen)
	g.handleNativeChurchReviveInput(nativeChurchReviveInput{enter: true})
	drainNativeChurchReviveJobs(t, g, screen)
	unit := g.partyRoster[0]
	if g.churchMode != "revive_empty" || g.gold != 79 || unit.HP != 24 || !unit.OnField {
		t.Fatalf("revive result mode=%q gold=%d unit=%#v", g.churchMode, g.gold, unit)
	}
	g.handleNativeChurchReviveInput(nativeChurchReviveInput{enter: true})
	drainNativeChurchReviveJobs(t, g, screen)
	if g.churchMode != "menu" {
		t.Fatalf("empty feedback return mode=%q", g.churchMode)
	}
}
