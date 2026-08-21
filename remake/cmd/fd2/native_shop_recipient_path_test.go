package main

import (
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func newNativeShopRecipientPathGame(t *testing.T) (*Game, *ebiten.Image) {
	t.Helper()
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided original resources are absent")
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(base, "DATO.DAT"))
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""

	shared, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	shop, err := loadNativeShopUIAssets(shared)
	if err != nil {
		t.Fatal(err)
	}
	c := &campaign.Campaign{
		Start: "shop",
		Nodes: map[string]*campaign.Node{
			"shop": {
				Type: "shop", NativeHubVariant: 1,
				Goods: []campaign.Good{{ID: 0, Name: "item0", Price: 100}},
				Next:  "town",
			},
			"town": {Type: "town"},
		},
	}
	g := &Game{
		camp:          campaign.NewRunner(c),
		nativeClassUI: shared,
		nativeShopUI:  shop,
		nativeUIPalette: append(
			color.Palette(nil), shared.palette...,
		),
		gold:           1234,
		shopItemTypes:  map[int]int{0: 0},
		shopEquipTypes: map[int][]int{1: {0}},
		shopItemStats: map[int]campaign.ItemStats{
			0: {Type: 0, AP: 1},
		},
		partyRoster: make(map[int]battle.Unit, 6),
	}
	for id := 0; id < 6; id++ {
		g.partyJoinOrder = append(g.partyJoinOrder, id)
		g.partyRoster[id] = battle.Unit{
			Name: "recipient", ClassID: 1, BattleFig: 0,
			NativeIdentity: id, HasNativeIdentity: true,
			MapSelectorKey: 0, HasMapSelectorKey: true,
			NativeRecordByte6: 1, HasNativeRecordByte6: true,
			NativeRecordRace: 1, HasNativeRecordRace: true,
			NativeRecordClass: 1, HasNativeRecordClass: true,
			Lv: 8, MV: 5, Exp: 10, DX: 17,
			HP: 30, MaxHP: 35, MP: 5, MaxMP: 9,
			AP: 41, DP: 32, HIT: 70, EV: 22,
			InventorySlots: []int{
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
			NativeInventoryFlags: []int{
				0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
			},
			BaseAP: 29, BaseDP: 25, BaseHIT: 17, BaseEV: 17,
			BaseMV: 5, EquipmentBaseSet: true,
		}
	}
	if !g.setupNativeShop() {
		t.Fatal("production shop setup rejected six-recipient fixture")
	}
	return g, ebiten.NewImage(640, 400)
}

func drainNativeShopProductionJobs(
	t *testing.T, g *Game, screen *ebiten.Image,
) {
	t.Helper()
	now := time.Unix(100, 0)
	for guard := 0; guard < 256 && g.nativeShopUIJob != nil; guard++ {
		job := g.nativeShopUIJob
		if len(job.timeline) != 0 {
			g.stepNativeShopUILifecycle(now)
			total := time.Duration(0)
			for _, step := range job.timeline {
				total += step.duration
			}
			now = now.Add(total)
			g.stepNativeShopUILifecycle(now)
			if !g.drawNativeShopUIJob(screen) {
				t.Fatal("production timeline final frame did not draw")
			}
			g.stepNativeShopUILifecycle(now)
			continue
		}
		if !g.drawNativeShopUIJob(screen) {
			t.Fatalf(
				"production frame job did not draw: mode=%q frame=%d/%d restore=%d",
				g.nativeShopMode, job.frame, len(job.frames), len(job.restore),
			)
		}
		g.stepNativeShopUILifecycle(now)
	}
	if g.nativeShopUIJob != nil {
		t.Fatalf("production shop lifecycle did not settle: mode=%q", g.nativeShopMode)
	}
}

func enterNativeShopEquipmentRecipients(
	t *testing.T, g *Game, screen *ebiten.Image,
) {
	t.Helper()
	drainNativeShopProductionJobs(t, g, screen)
	for _, wantMode := range []string{"purchase", "confirm", "recipient_equipment"} {
		if !g.handleNativeShopInput(true) {
			t.Fatalf("production input did not consume Enter before %q", wantMode)
		}
		drainNativeShopProductionJobs(t, g, screen)
		if g.nativeShopMode != wantMode {
			t.Fatalf("production input mode=%q, want %q", g.nativeShopMode, wantMode)
		}
	}
}

func TestNativeShopEquipmentRecipientProductionInputScrollAndPurchase(
	t *testing.T,
) {
	g, screen := newNativeShopRecipientPathGame(t)
	enterNativeShopEquipmentRecipients(t, g, screen)
	if len(g.shopRecipients) != 6 {
		t.Fatalf("eligible recipients=%v, want six", g.shopRecipients)
	}
	for step := 1; step <= 3; step++ {
		if !g.handleNativeShopRecipientInput(nativeShopRecipientInput{down: true}) {
			t.Fatalf("recipient Down%d was not consumed", step)
		}
	}
	if g.shopRecipientSel != 3 || g.nativeShopRecipientStart != 1 {
		t.Fatalf(
			"three-row scroll=(selection%d,start%d), want (3,1)",
			g.shopRecipientSel, g.nativeShopRecipientStart,
		)
	}
	if !g.handleNativeShopRecipientInput(nativeShopRecipientInput{
		left: true, right: true,
	}) || g.shopRecipientSel != 3 || g.nativeShopRecipientStart != 1 {
		t.Fatal("equipment recipient horizontal input changed the three-row window")
	}
	if !g.handleNativeShopRecipientInput(nativeShopRecipientInput{enter: true}) {
		t.Fatal("recipient Enter was not consumed")
	}
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "equip_confirm" || !g.nativeShopHasPendingUnit ||
		g.shopEquipUnit != 3 {
		t.Fatalf(
			"staged purchase mode=%q pending=%v unit=%d",
			g.nativeShopMode, g.nativeShopHasPendingUnit, g.shopEquipUnit,
		)
	}
	if !g.handleNativeShopInput(true) {
		t.Fatal("equip-confirm Enter was not consumed")
	}
	drainNativeShopProductionJobs(t, g, screen)
	got := g.partyRoster[3]
	if g.nativeShopMode != "purchase" || g.gold != 1134 ||
		len(got.Inventory) != 1 || got.Inventory[0] != 0 ||
		len(got.Equipped) != 1 || !got.Equipped[0] ||
		g.nativeShopHasPendingUnit {
		t.Fatalf(
			"completed purchase mode=%q gold=%d inventory=%v equipped=%v pending=%v",
			g.nativeShopMode, g.gold, got.Inventory, got.Equipped,
			g.nativeShopHasPendingUnit,
		)
	}
}

func TestNativeShopEquipmentRecipientProductionInputFullIsAtomic(
	t *testing.T,
) {
	g, screen := newNativeShopRecipientPathGame(t)
	full := cloneNativeShopUnit(g.partyRoster[3])
	full.Inventory = []int{1, 2, 3, 4, 5, 6, 7, 8}
	full.Equipped = make([]bool, 8)
	for i := range full.InventorySlots {
		full.InventorySlots[i] = full.Inventory[i]
		full.NativeInventoryFlags[i] = 0
	}
	g.partyRoster[3] = full
	want := cloneNativeShopUnit(full)
	enterNativeShopEquipmentRecipients(t, g, screen)
	for step := 0; step < 3; step++ {
		g.handleNativeShopRecipientInput(nativeShopRecipientInput{down: true})
	}
	g.handleNativeShopRecipientInput(nativeShopRecipientInput{enter: true})
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "recipient_full" || g.gold != 1234 ||
		g.nativeShopHasPendingUnit || !reflect.DeepEqual(g.partyRoster[3], want) {
		t.Fatalf(
			"full feedback mode=%q gold=%d pending=%v unit=%#v",
			g.nativeShopMode, g.gold, g.nativeShopHasPendingUnit, g.partyRoster[3],
		)
	}
	g.handleNativeShopRecipientInput(nativeShopRecipientInput{enter: true})
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "purchase" || g.gold != 1234 ||
		!reflect.DeepEqual(g.partyRoster[3], want) {
		t.Fatal("full feedback return mutated gold or recipient")
	}
}

func TestNativeShopNoEligibleRecipientProductionInputIsAtomic(t *testing.T) {
	g, screen := newNativeShopRecipientPathGame(t)
	g.shopEquipTypes = map[int][]int{1: {1, 2, 3, 4, 5, 6}}
	want := make(map[int]battle.Unit, len(g.partyRoster))
	for id, unit := range g.partyRoster {
		want[id] = cloneNativeShopUnit(unit)
	}
	drainNativeShopProductionJobs(t, g, screen)
	for _, wantMode := range []string{"purchase", "confirm", "no_recipient"} {
		g.handleNativeShopInput(true)
		drainNativeShopProductionJobs(t, g, screen)
		if g.nativeShopMode != wantMode {
			t.Fatalf("no-recipient path mode=%q, want %q", g.nativeShopMode, wantMode)
		}
	}
	if g.gold != 1234 || g.nativeShopHasPendingUnit {
		t.Fatalf("no-recipient feedback changed transaction state: gold=%d pending=%v", g.gold, g.nativeShopHasPendingUnit)
	}
	for id, unit := range want {
		if !reflect.DeepEqual(g.partyRoster[id], unit) {
			t.Fatalf("no-recipient feedback mutated unit %d", id)
		}
	}
	g.handleNativeShopRecipientInput(nativeShopRecipientInput{escape: true})
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "purchase" || g.gold != 1234 {
		t.Fatalf("no-recipient return mode=%q gold=%d", g.nativeShopMode, g.gold)
	}
}

func TestNativeShopEquipmentRecipientProductionInputCancelIsAtomic(
	t *testing.T,
) {
	g, screen := newNativeShopRecipientPathGame(t)
	want := make(map[int]battle.Unit, len(g.partyRoster))
	for id, unit := range g.partyRoster {
		want[id] = cloneNativeShopUnit(unit)
	}
	enterNativeShopEquipmentRecipients(t, g, screen)
	if !g.handleNativeShopRecipientInput(nativeShopRecipientInput{escape: true}) {
		t.Fatal("recipient Escape was not consumed")
	}
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "purchase" || g.gold != 1234 ||
		g.nativeShopHasPendingUnit {
		t.Fatalf(
			"recipient cancel mode=%q gold=%d pending=%v",
			g.nativeShopMode, g.gold, g.nativeShopHasPendingUnit,
		)
	}
	for id, unit := range want {
		if !reflect.DeepEqual(g.partyRoster[id], unit) {
			t.Fatalf("recipient cancel mutated unit %d", id)
		}
	}
}
