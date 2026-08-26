package main

import (
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func newNativeChurchClassPathGame(t *testing.T) (*Game, *ebiten.Image) {
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
	unit := battle.Unit{
		Name: "悠妮", Portrait: 9, BattleFig: 9, HasBattleFig: true,
		ClassID: 5, NativeRecordClass: 5, HasNativeRecordClass: true,
		NativeIdentity: 9, HasNativeIdentity: true,
		MapSelectorKey: 9, HasMapSelectorKey: true,
		Lv: 20, Exp: 31, HP: 22, MaxHP: 30, MP: 7, MaxMP: 10,
		AP: 20, DP: 18, DX: 12, MV: 5,
		Inventory: []int{0x5a, 0x64}, Equipped: []bool{false, true},
		InventorySlots:       []int{0x5a, 0x64, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		NativeInventoryFlags: []int{0, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80},
	}
	special := 0x34
	g := &Game{
		nativeClassUI: assets, nativeChurchTextIndex: 585,
		churchMode: "class", churchIDs: []int{9}, churchClassID: -1,
		partyJoinOrder: []int{9}, partyRoster: map[int]battle.Unit{9: unit},
		classChangeTable: campaign.ClassChangeTable{
			Current: map[int]campaign.ClassChangeCurrent{9: {
				Portrait: 9, DefaultTarget: 0x29, SpecialItem: 0x5a, SpecialTarget: &special,
			}},
			Targets: map[int]campaign.ClassChangeTarget{
				0x29: {Portrait: 0x29, ClassID: 13},
				0x34: {Portrait: 0x34, ClassID: 21, MobilityIncrement: 2},
			},
		},
		classChangeGrowth: map[int]campaign.ClassChangeGrowth{0x34: {
			AP: [2]int{10, 11}, DP: [2]int{20, 21}, DX: [2]int{30, 31},
			HP: [2]int{40, 41}, MP: [2]int{50, 51},
		}},
		shopItemStats: map[int]campaign.ItemStats{0x64: {AP: 3, DP: 2, HIT: 1, EV: 2, MV: 1}},
		rng:           rand.New(rand.NewSource(1)),
	}
	return g, ebiten.NewImage(640, 400)
}

func drainNativeChurchClassJobs(t *testing.T, g *Game, screen *ebiten.Image) {
	t.Helper()
	for steps := 0; g.nativeClassUIJob != nil; steps++ {
		if steps >= 64 {
			t.Fatalf("class job did not settle: mode=%q", g.churchMode)
		}
		if !g.drawNativeClassUIJob(screen) {
			t.Fatal("class indexed frame did not draw")
		}
		g.stepNativeClassUILifecycle(time.Time{})
	}
}

func enterNativeChurchClassConfirmation(t *testing.T, g *Game, screen *ebiten.Image) {
	t.Helper()
	if !g.handleNativeChurchClassInput(nativeChurchClassInput{enter: true}) {
		t.Fatal("class list input")
	}
	drainNativeChurchClassJobs(t, g, screen)
	if g.churchMode != "class_confirm" || g.churchClassID != 9 || len(g.churchBranches) != 1 {
		t.Fatalf("class confirmation mode=%q id=%d branches=%#v", g.churchMode, g.churchClassID, g.churchBranches)
	}
}

func TestNativeChurchClassTypedInputCancelIsAtomic(t *testing.T) {
	g, screen := newNativeChurchClassPathGame(t)
	want := g.partyRoster[9]
	enterNativeChurchClassConfirmation(t, g, screen)
	g.handleNativeChurchClassInput(nativeChurchClassInput{escape: true})
	drainNativeChurchClassJobs(t, g, screen)
	if g.churchMode != "class" || !reflect.DeepEqual(g.partyRoster[9], want) {
		t.Fatal("class cancel changed persistent unit")
	}
}

func TestNativeChurchClassTypedInputMissingTargetFailsClosed(t *testing.T) {
	g, _ := newNativeChurchClassPathGame(t)
	want := g.partyRoster[9]
	g.classChangeTable.Targets = nil
	g.handleNativeChurchClassInput(nativeChurchClassInput{enter: true})
	if g.churchMode != "class" || g.nativeClassUIJob != nil ||
		!reflect.DeepEqual(g.partyRoster[9], want) || g.msg != "缺少原版轉職目標資料" {
		t.Fatalf("missing target leaked state: mode=%q msg=%q", g.churchMode, g.msg)
	}
}

func TestNativeChurchClassTypedInputSuccessPublishesCompleteUnit(t *testing.T) {
	g, screen := newNativeChurchClassPathGame(t)
	enterNativeChurchClassConfirmation(t, g, screen)
	g.handleNativeChurchClassInput(nativeChurchClassInput{enter: true})
	drainNativeChurchClassJobs(t, g, screen)
	changed := g.partyRoster[9]
	if g.churchMode != "class" || changed.Portrait != 0x34 || changed.ClassID != 21 ||
		changed.NativeRecordClass != 21 || changed.MapSelectorKey != 0x34 || changed.MV != 8 ||
		changed.Exp != 0 || len(changed.Inventory) != 1 || changed.Inventory[0] != 0x64 ||
		len(changed.Equipped) != 1 || !changed.Equipped[0] {
		t.Fatalf("class success mode=%q unit=%#v", g.churchMode, changed)
	}
}
