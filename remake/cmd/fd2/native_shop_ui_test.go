package main

import (
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func TestNativeShopProductionOwnerDrawsOriginalMenuAndPurchaseList(t *testing.T) {
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
				Goods: []campaign.Good{
					{ID: 0, Name: "item0", Price: 100},
					{ID: 1, Name: "item1", Price: 200},
				},
				Next: "town",
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
		gold: 1234,
	}
	if !g.setupNativeShop() || g.nativeShopMode != "menu" ||
		g.nativeShopUIJob == nil || len(g.nativeShopUIJob.frames) != 4 {
		t.Fatal("native shop did not claim the original node with four-frame service opening")
	}
	screen := ebiten.NewImage(640, 400)
	g.nativeShopUIJob = nil
	g.nativeShopUIPulse = 0
	normalMenu, ok := g.composeNativeShopServiceMenu()
	if !ok {
		t.Fatal("native shop normal service pulse did not compose")
	}
	g.nativeShopUIPulse = 2
	selectedMenu, ok := g.composeNativeShopServiceMenu()
	if !ok || string(normalMenu) == string(selectedMenu) {
		t.Fatal("production shop owner collapsed the four-phase service pulse")
	}
	if !g.drawNativeShop(screen) {
		t.Fatal("native shop service menu unexpectedly fell back")
	}
	if !g.setNativeShopInsufficientShotState(0, 0) ||
		g.nativeShopMode != "insufficient" ||
		!g.drawNativeShop(screen) {
		t.Fatal("strict insufficient screenshot state did not compose from real assets")
	}
	g.nativeShopMode = "menu"
	g.gold = 1234
	g.nativeShopMode = "purchase"
	if !g.drawNativeShop(screen) {
		t.Fatal("native shop purchase list unexpectedly fell back")
	}
	if !g.beginNativeShopPurchaseOpening() ||
		len(g.nativeShopUIJob.frames) != 6 {
		t.Fatal("native shop purchase list did not use the six-frame original lifecycle")
	}
	g.nativeShopUIJob = nil
	if !g.beginNativeShopPurchaseClosing(nil) ||
		len(g.nativeShopUIJob.frames) != 5 ||
		len(g.nativeShopUIJob.restore) != 320*200 {
		t.Fatal("native shop purchase list did not use five close frames and stable restore")
	}
	g.nativeShopUIJob = nil
	g.nativeShopMode = "confirm"
	g.nativeShopConfirmSel = 0
	if !g.drawNativeShop(screen) {
		t.Fatal("native shop purchase confirmation unexpectedly fell back")
	}
	if !g.beginNativeShopConfirmationOpening() ||
		len(g.nativeShopUIJob.frames) != 4 {
		t.Fatal("native shop confirmation did not use four original opening frames")
	}
	g.nativeShopUIJob = nil
	if !g.beginNativeShopConfirmationChoiceClosing(nil) ||
		len(g.nativeShopUIJob.frames) != 4 {
		t.Fatal("native shop confirmation did not use four original closing frames")
	}
	postChoiceClose := append(
		[]byte(nil),
		g.nativeShopUIJob.frames[len(g.nativeShopUIJob.frames)-1]...,
	)
	g.nativeShopUIJob = nil
	g.nativeShopMode = "insufficient"
	insufficient, ok := g.composeNativeShopInsufficientGold()
	if !ok || string(insufficient) == string(postChoiceClose) {
		t.Fatal("native shop insufficient-gold feedback did not append after choice close")
	}
	if !g.beginNativeShopDialogueClosing(insufficient, nil) ||
		len(g.nativeShopUIJob.frames) != 5 ||
		len(g.nativeShopUIJob.restore) != 320*200 {
		t.Fatal("native shop insufficient-gold feedback did not use five dialogue close frames")
	}
	unit := battle.Unit{
		BattleFig: 0, NativeIdentity: 0, HasNativeIdentity: true,
		MapSelectorKey: 0, HasMapSelectorKey: true,
		NativeRecordByte6: 1, HasNativeRecordByte6: true,
		NativeRecordRace: 1, HasNativeRecordRace: true,
		NativeRecordClass: 1, HasNativeRecordClass: true,
		Lv: 8, MV: 5, Exp: 10, DX: 17,
		HP: 30, MaxHP: 35, MP: 5, MaxMP: 9,
		AP: 41, DP: 32, HIT: 70, EV: 22,
		InventorySlots:       []int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		NativeInventoryFlags: []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80},
		BaseAP:               29, BaseDP: 25, EquipmentBaseSet: true,
	}
	g.nativeShopUIJob = nil
	g.partyJoinOrder = []int{0}
	g.partyRoster = map[int]battle.Unit{0: unit}
	g.shopItemTypes = map[int]int{0: 0x20}
	g.shopEquipTypes = map[int][]int{1: []int{0, 1, 2, 3, 4, 5}}
	if !g.setupNativeShopRecipients() ||
		g.nativeShopMode != "recipient_consumable" ||
		!g.drawNativeShop(screen) {
		t.Fatal("native consumable recipient production owner unexpectedly fell back")
	}
	if !g.beginNativeShopRecipientOpening() ||
		len(g.nativeShopUIJob.frames) != 6 {
		t.Fatal("native consumable recipient did not use six opening frames")
	}
	g.nativeShopUIJob = nil
	if !g.beginNativeShopRecipientClosing(nil) ||
		len(g.nativeShopUIJob.frames) != 5 ||
		len(g.nativeShopUIJob.restore) != 320*200 {
		t.Fatal("native consumable recipient did not use five close frames")
	}
	g.nativeShopUIJob = nil
	g.shopItemTypes[0] = 0
	if !g.setupNativeShopRecipients() ||
		g.nativeShopMode != "recipient_equipment" ||
		!g.drawNativeShop(screen) {
		t.Fatal("native equipment recipient production owner unexpectedly fell back")
	}
	for i := range unit.NativeInventoryFlags {
		unit.NativeInventoryFlags[i] = 0
		unit.InventorySlots[i] = i
	}
	g.partyRoster[0] = unit
	g.nativeShopMode = "recipient_full"
	if !nativeShopInventoryFull(unit) || !g.drawNativeShop(screen) {
		t.Fatal("native recipient-full production feedback unexpectedly fell back")
	}
	if !g.beginNativeShopRecipientFullOpening() ||
		len(g.nativeShopUIJob.frames) != 6 {
		t.Fatal("native recipient-full feedback did not use six opening frames")
	}
	g.nativeShopUIJob = nil
	g.nativeShopPendingUnit = cloneNativeShopUnit(unit)
	g.nativeShopHasPendingUnit = true
	g.shopPending = campaign.Good{ID: 0, Name: "item0", Price: 100}
	g.shopEquipUnit = 0
	for _, tc := range []struct {
		variant int
		steps   int
		ticks   int
	}{
		{variant: 1, steps: 6, ticks: 10},
		{variant: 3, steps: 3, ticks: 9},
		{variant: 5, steps: 8, ticks: 14},
	} {
		g.nativeShopVariant = tc.variant
		if !g.beginNativeShopPurchaseSuccess() {
			t.Fatalf("native shop variant %d success timeline fell back", tc.variant)
		}
		if len(g.nativeShopUIJob.timeline) != tc.steps {
			t.Fatalf(
				"native shop variant %d timeline steps=%d, want %d",
				tc.variant, len(g.nativeShopUIJob.timeline), tc.steps,
			)
		}
		total := time.Duration(0)
		for _, step := range g.nativeShopUIJob.timeline {
			total += step.duration
		}
		if total != time.Duration(tc.ticks)*nativeBIOSTickPeriod {
			t.Fatalf(
				"native shop variant %d duration=%v, want %d BIOS ticks",
				tc.variant, total, tc.ticks,
			)
		}
		bare, bareOK := g.composeNativeShopBare()
		assets, portrait, portraitID, stateOK := g.nativeShopState()
		animation, _, composeErr := campaign.ComposeNativeShopPurchaseSuccessFrames(
			bare, assets, portrait, portraitID, tc.variant,
		)
		expectedFirst := animation[0]
		if tc.variant == 3 {
			expectedFirst = bare
		}
		if !bareOK || !stateOK || composeErr != nil ||
			!reflect.DeepEqual(g.nativeShopUIJob.timeline[0].frame, expectedFirst) {
			t.Fatalf(
				"native shop variant %d success did not start from bare scene: bare=%v state=%v err=%v",
				tc.variant, bareOK, stateOK, composeErr,
			)
		}
		g.nativeShopUIJob = nil
	}
	g.nativeShopVariant = 1
	g.nativeShopMode = "equip_confirm"
	g.nativeShopEquipSel = 0
	if !g.drawNativeShop(screen) ||
		!g.beginNativeShopEquipConfirmationOpening() ||
		len(g.nativeShopUIJob.frames) != 4 {
		t.Fatal("native optional-equip confirmation unexpectedly fell back")
	}
	empty := cloneNativeShopUnit(unit)
	empty.Inventory = nil
	empty.Equipped = nil
	empty.InventorySlots = []int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	empty.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	g.nativeShopUIJob = nil
	g.nativeShopVariant = 1
	g.nativeShopMode = "recipient_consumable"
	g.nativeShopHasPendingUnit = false
	g.partyRoster[0] = empty
	g.shopRecipients = []int{0}
	g.shopRecipientSel = 0
	g.shopItemTypes[0] = 0x20
	g.gold = 1234
	if !g.stageNativeShopPurchase() {
		t.Fatal("native consumable transaction did not enter success timeline")
	}
	if got := g.partyRoster[0]; len(got.Inventory) != 1 ||
		got.Inventory[0] != 0 || g.gold != 1234 {
		t.Fatalf(
			"native insert/debit ordering changed: inventory=%#v gold=%d",
			got.Inventory, g.gold,
		)
	}
	finish := g.nativeShopUIJob.after
	g.nativeShopUIJob = nil
	finish()
	if g.gold != 1134 || g.nativeShopMode != "success" ||
		g.nativeShopUIJob == nil ||
		len(g.nativeShopUIJob.timeline) != 9 {
		t.Fatalf(
			"native debit roll start = gold %d mode %q job=%v frames=%d",
			g.gold, g.nativeShopMode, g.nativeShopUIJob != nil,
			len(g.nativeShopUIJob.timeline),
		)
	}
	debitTotal := time.Duration(0)
	for _, step := range g.nativeShopUIJob.timeline {
		if step.duration !=
			campaign.NativeGoldRollDelayMilliseconds*time.Millisecond {
			t.Fatalf("native debit phase duration=%v", step.duration)
		}
		debitTotal += step.duration
	}
	if debitTotal != 90*time.Millisecond {
		t.Fatalf("native debit total=%v, want 90ms", debitTotal)
	}
	finish = g.nativeShopUIJob.after
	g.nativeShopUIJob = nil
	finish()
	if g.nativeShopMode != "purchase" || g.nativeShopUIJob == nil {
		t.Fatalf(
			"native product-loop completion mode=%q job=%v",
			g.nativeShopMode, g.nativeShopUIJob != nil,
		)
	}
	assertNativeShopTransactionTownRoundTrip(
		t, g, 1, 1134, cloneNativeShopUnit(g.partyRoster[0]),
	)
	g.nativeShopUIJob = nil
	g.shopItemTypes[0] = 0
	g.shopEquipTypes[1] = []int{1, 2, 3, 4, 5, 6}
	if !g.setupNativeShopRecipients() ||
		g.nativeShopMode != "no_recipient" ||
		!g.drawNativeShop(screen) {
		t.Fatal("native no-eligible-recipient feedback unexpectedly fell back")
	}
	if !g.beginNativeShopNoEligibleRecipientOpening() ||
		len(g.nativeShopUIJob.frames) != 6 {
		t.Fatal("native no-eligible-recipient feedback did not use six opening frames")
	}
	g.nativeShopUIJob = nil
	if !g.beginNativeShopNoEligibleRecipientClosing(nil) ||
		len(g.nativeShopUIJob.frames) != 5 ||
		len(g.nativeShopUIJob.restore) != 320*200 {
		t.Fatal("native no-eligible-recipient feedback did not use five close frames")
	}
	seller := cloneNativeShopUnit(empty)
	seller.Inventory = []int{0}
	seller.Equipped = []bool{false}
	seller.InventorySlots[0] = 0
	seller.NativeInventoryFlags[0] = 0
	g.nativeShopUIJob = nil
	g.partyRoster[0] = seller
	g.shopItemStats = map[int]campaign.ItemStats{
		0: {Type: 0, AP: 1},
	}
	g.gold = 100
	if !g.setupNativeShopSellRoster() ||
		!g.drawNativeShop(screen) ||
		!g.beginNativeShopSellRosterOpening() ||
		len(g.nativeShopUIJob.frames) != 6 {
		t.Fatal("native sell roster production owner unexpectedly fell back")
	}
	g.nativeShopUIJob = nil
	if !g.setupNativeShopSellItems() ||
		!g.drawNativeShop(screen) ||
		!g.beginNativeShopSellItemsOpening() ||
		len(g.nativeShopUIJob.frames) != 6 {
		t.Fatal("native sell item selector unexpectedly fell back")
	}
	g.nativeShopUIJob = nil
	g.nativeShopMode = "sell_confirm"
	g.nativeShopSellConfirmSel = 0
	if !g.drawNativeShop(screen) ||
		!g.beginNativeShopSellConfirmationOpening() ||
		len(g.nativeShopUIJob.frames) != 4 {
		t.Fatal("native sell confirmation unexpectedly fell back")
	}
	g.nativeShopUIJob = nil
	if !g.beginNativeShopSellSuccess() {
		t.Fatal("native sell success timeline unexpectedly fell back")
	}
	if g.gold != 100 || len(g.partyRoster[0].Inventory) != 1 {
		t.Fatalf(
			"native sell mutated before success: gold=%d inventory=%#v",
			g.gold, g.partyRoster[0].Inventory,
		)
	}
	finishSell := g.nativeShopUIJob.after
	g.nativeShopUIJob = nil
	finishSell()
	creditFrames := 0
	if g.nativeShopUIJob != nil {
		creditFrames = len(g.nativeShopUIJob.timeline)
	}
	if g.gold != 137 || len(g.partyRoster[0].Inventory) != 1 ||
		g.nativeShopMode != "sell_success" || g.nativeShopUIJob == nil ||
		creditFrames != 63 {
		t.Fatalf(
			"native sell credit start = gold %d inventory=%#v mode=%q job=%v frames=%d",
			g.gold, g.partyRoster[0].Inventory, g.nativeShopMode,
			g.nativeShopUIJob != nil, creditFrames,
		)
	}
	creditTotal := time.Duration(0)
	for _, step := range g.nativeShopUIJob.timeline {
		if step.duration !=
			campaign.NativeGoldRollDelayMilliseconds*time.Millisecond {
			t.Fatalf("native credit phase duration=%v", step.duration)
		}
		creditTotal += step.duration
	}
	if creditTotal != 630*time.Millisecond {
		t.Fatalf("native credit total=%v, want 630ms", creditTotal)
	}
	finishSell = g.nativeShopUIJob.after
	g.nativeShopUIJob = nil
	finishSell()
	if g.gold != 137 || len(g.partyRoster[0].Inventory) != 0 ||
		g.partyRoster[0].InventorySlots[0] != 0xff ||
		g.nativeShopMode != "sell_roster" {
		t.Fatalf(
			"native sell completion = gold %d inventory=%#v slots=%#v mode=%q",
			g.gold, g.partyRoster[0].Inventory,
			g.partyRoster[0].InventorySlots, g.nativeShopMode,
		)
	}
	assertNativeShopTransactionTownRoundTrip(
		t, g, 2, 137, cloneNativeShopUnit(g.partyRoster[0]),
	)
	g.nativeShopUIJob = nil
	g.nativeShopMode = "sell_empty"
	if !g.drawNativeShop(screen) ||
		!g.beginNativeShopSellEmptyOpening() ||
		len(g.nativeShopUIJob.frames) != 6 {
		t.Fatal("native sell-empty feedback unexpectedly fell back")
	}

	equipper := cloneNativeShopUnit(unit)
	equipper.ClassID = 1
	equipper.Inventory = []int{0, 1}
	equipper.Equipped = []bool{true, false}
	equipper.InventorySlots = []int{0, 1, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	equipper.NativeInventoryFlags = []int{0x40, 0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	equipper.BaseAP, equipper.BaseDP = 29, 25
	equipper.EquipmentBaseSet = true
	g.nativeShopUIJob = nil
	g.partyRoster[0] = equipper
	g.shopItemTypes[0], g.shopItemTypes[1] = 0, 0
	g.shopEquipTypes[1] = []int{0, 2, 3, 4, 5, 6}
	g.shopItemStats = map[int]campaign.ItemStats{
		0: {Type: 0, AP: 1},
		1: {Type: 0, AP: 3},
	}
	if !g.setupNativeShopEquipRoster() ||
		!g.drawNativeShop(screen) ||
		!g.beginNativeShopEquipRosterOpening() ||
		len(g.nativeShopUIJob.frames) != 6 {
		t.Fatal("native standalone-equip roster unexpectedly fell back")
	}
	g.nativeShopUIJob = nil
	if !g.openNativeShopEquipPanel() {
		t.Fatal("native standalone-equip item/status panel unexpectedly fell back")
	}
	g.itemAnimStep = 11
	if !g.drawNativeShop(screen) {
		t.Fatal("native standalone-equip panel did not render in production")
	}
	beforeFailedEquip := cloneNativeShopUnit(g.partyRoster[0])
	beforeFailedPanel := g.nativeItemPanel
	beforeFailedBase := append([]byte(nil), g.nativeItemPanelBase...)
	beforeFailedRecord := append([]byte(nil), g.nativeItemPanelRecord...)
	savedPalette := g.nativeUIPalette
	g.nativeUIPalette = nil
	g.itemSel = 1
	if g.applyNativeShopEquipSelection() {
		t.Fatal("native standalone-equip published without renderer palette")
	}
	if !reflect.DeepEqual(g.partyRoster[0], beforeFailedEquip) ||
		g.nativeItemPanel != beforeFailedPanel ||
		!reflect.DeepEqual(g.nativeItemPanelBase, beforeFailedBase) ||
		!reflect.DeepEqual(g.nativeItemPanelRecord, beforeFailedRecord) ||
		g.itemSel != 1 {
		t.Fatalf(
			"failed native equip leaked state: unit=%#v panel=%p selection=%d",
			g.partyRoster[0], g.nativeItemPanel, g.itemSel,
		)
	}
	g.nativeUIPalette = savedPalette
	g.itemSel = 1
	if !g.applyNativeShopEquipSelection() {
		t.Fatal("native standalone-equip transaction failed")
	}
	equipped := g.partyRoster[0]
	if !reflect.DeepEqual(equipped.Equipped, []bool{false, true}) ||
		!reflect.DeepEqual(
			equipped.NativeInventoryFlags,
			[]int{0, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80},
		) ||
		equipped.AP != 32 {
		t.Fatalf(
			"native standalone equip = equipped %#v flags %#v AP=%d",
			equipped.Equipped, equipped.NativeInventoryFlags, equipped.AP,
		)
	}
	g.beginNativeShopEquipPanelClose()
	for i := 0; i < 12; i++ {
		g.stepNativeShopEquipPanelLifecycle()
	}
	if g.nativeShopMode != "equip_roster" || g.nativeShopUIJob == nil ||
		len(g.nativeShopUIJob.frames) != 6 {
		t.Fatal("native standalone-equip panel did not restore roster lifecycle")
	}

	g.nativeShopUIJob = nil
	if !g.setupNativeShopTransfer() ||
		g.nativeShopMode != "transfer_intro" ||
		g.nativeShopUIJob == nil ||
		len(g.nativeShopUIJob.frames) != 6 {
		t.Fatal("native shop transfer source prompt unexpectedly fell back")
	}
	g.nativeShopUIJob = nil
	g.openNativeShopTransferSourceRoster()
	if g.nativeShopMode != "transfer_source" ||
		len(g.nativeShopTransferIDs) != 1 ||
		g.nativeShopTransferIDs[0] != 0 ||
		!g.drawNativeShop(screen) {
		t.Fatal("native shop transfer source roster unexpectedly fell back")
	}
	g.nativeShopUIJob = nil
	g.nativeShopTransferSource = 0
	if !g.openNativeShopTransferItems() ||
		g.nativeShopMode != "transfer_items" ||
		len(g.nativeShopTransferItems) != 2 ||
		!g.drawNativeShop(screen) {
		t.Fatal("native shop transfer item list unexpectedly fell back")
	}
	g.nativeShopUIJob = nil
	g.nativeShopTransferItem = g.nativeShopTransferItems[0]
	g.openNativeShopTransferDestinationRoster()
	if len(g.nativeShopTransferIDs) != 1 ||
		g.nativeShopTransferIDs[0] != 0 {
		t.Fatal("native destination roster incorrectly removed the source actor")
	}
	beforeCancel := cloneNativeShopUnit(g.partyRoster[0])
	beforeCancelGold := g.gold
	g.nativeShopUIJob = nil
	if !g.beginNativeShopTransferDestinationCancel() ||
		g.nativeShopMode != "transfer_dest" ||
		g.nativeShopUIJob == nil ||
		len(g.nativeShopUIJob.frames) != 5 ||
		len(g.nativeShopUIJob.restore) != 320*200 {
		t.Fatal("native transfer destination cancel did not publish five closing frames")
	}
	cancelAfter := g.nativeShopUIJob.after
	g.nativeShopUIJob = nil
	cancelAfter()
	if g.nativeShopMode != "transfer_intro" ||
		g.nativeShopUIJob == nil || len(g.nativeShopUIJob.frames) != 6 ||
		g.gold != beforeCancelGold ||
		!reflect.DeepEqual(g.partyRoster[0], beforeCancel) {
		t.Fatalf(
			"native transfer destination cancel leaked state: mode=%q gold=%d unit=%#v",
			g.nativeShopMode, g.gold, g.partyRoster[0],
		)
	}
	g.nativeShopUIJob = nil
	if g.beginNativeShopTransferDestinationCancel() ||
		g.nativeShopMode != "transfer_intro" || g.nativeShopUIJob != nil {
		t.Fatal("native transfer destination cancel accepted a non-destination state")
	}

	// 取消回到來源提示後，重新沿正式 owner 選相同來源與物品，才測試 self-transfer。
	g.openNativeShopTransferSourceRoster()
	g.nativeShopUIJob = nil
	g.nativeShopTransferSource = 0
	if !g.openNativeShopTransferItems() {
		t.Fatal("native shop transfer item list did not reopen after destination cancel")
	}
	g.nativeShopUIJob = nil
	g.nativeShopTransferItem = g.nativeShopTransferItems[0]
	g.openNativeShopTransferDestinationRoster()
	if !g.applyNativeShopTransfer(0) {
		t.Fatal("native self-transfer raw remove/append failed")
	}
	reordered := g.partyRoster[0]
	if !reflect.DeepEqual(reordered.Inventory, []int{1, 0}) ||
		!reflect.DeepEqual(reordered.Equipped, []bool{true, false}) ||
		!reflect.DeepEqual(
			reordered.NativeInventoryFlags,
			[]int{0x40, 0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80},
		) ||
		reordered.AP != 32 {
		t.Fatalf(
			"native self-transfer = inventory %#v equipped %#v flags %#v AP=%d",
			reordered.Inventory, reordered.Equipped,
			reordered.NativeInventoryFlags, reordered.AP,
		)
	}

	emptySource := cloneNativeShopUnit(reordered)
	emptySource.Inventory = nil
	emptySource.Equipped = nil
	emptySource.InventorySlots = []int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	emptySource.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	g.partyRoster[0] = emptySource
	g.nativeShopMode = "transfer_empty"
	g.nativeShopTransferSource = 0
	if !g.drawNativeShop(screen) ||
		!g.beginNativeShopTransferMessageOpening() ||
		len(g.nativeShopUIJob.frames) != 6 {
		t.Fatal("native shop transfer empty-source feedback fell back")
	}

	fullDestination := cloneNativeShopUnit(reordered)
	fullDestination.NativeIdentity = 1
	fullDestination.Inventory = []int{0, 1, 2, 3, 4, 5, 6, 7}
	fullDestination.Equipped = []bool{false, false, false, false, false, false, false, false}
	fullDestination.InventorySlots = []int{0, 1, 2, 3, 4, 5, 6, 7}
	fullDestination.NativeInventoryFlags = []int{0, 0, 0, 0, 0, 0, 0, 0}
	g.nativeShopUIJob = nil
	g.partyRoster[1] = fullDestination
	g.nativeShopMode = "transfer_full"
	g.nativeShopTransferDest = 1
	if !g.drawNativeShop(screen) ||
		!g.beginNativeShopTransferMessageOpening() ||
		len(g.nativeShopUIJob.frames) != 6 {
		t.Fatal("native shop transfer full-destination feedback fell back")
	}

	// 正式獨立裝備與轉移擁有者已在上方依原始拓撲（raw topology）
	// 產生 reordered；現在必須經離店→town 節點，才可進入重製 JSON 存檔邊界。
	g.partyRoster[0] = reordered
	delete(g.partyRoster, 1)
	g.partyMembers = map[int]bool{0: true}
	g.partyJoinOrder = []int{0}
	g.partyDeploy = map[int]bool{0: true}
	g.handlerChapter = 1
	g.nativeShopVariant = 1
	g.leaveShop()
	if g.camp.NodeID() != "town" || g.campSel != 1 {
		t.Fatalf("native shop leave boundary=(%q,%d), want town/1", g.camp.NodeID(), g.campSel)
	}
	g.saveGameToSlot(2)
	if g.msg != "已存檔(槽位3：town)" {
		t.Fatalf("native shop town save message=%q", g.msg)
	}

	// 模擬讀檔前仍有另一個商店子畫面；這些欄位皆未序列化，不得穿越 town save。
	g.partyRoster = nil
	g.partyMembers = nil
	g.partyJoinOrder = nil
	g.partyDeploy = nil
	g.nativeShopUIJob = &nativeClassUIJob{frames: [][]byte{{1}}}
	g.nativeShopMode = "transfer_full"
	g.nativeShopVariant = 5
	g.nativeShopServiceSel = 3
	g.nativeShopItemStart = 4
	g.nativeShopConfirmSel = 1
	g.nativeShopRecipientStart = 2
	g.nativeShopRecipientCycle = 1
	g.nativeShopEquipSel = 1
	g.nativeShopHasPendingUnit = true
	g.nativeShopPendingGold = 999
	g.nativeShopSellRosterTop = 2
	g.nativeShopSellItemTop = 2
	g.nativeShopSellConfirmSel = 1
	g.nativeShopSellItemIDs = []int{0}
	g.nativeShopEquipRosterTop = 2
	g.nativeShopEquipUnitSel = 1
	g.nativeShopTransferSource = 7
	g.nativeShopTransferItem = 6
	g.nativeShopTransferItems = []int{6}
	g.nativeShopTransferDest = 8
	g.nativeShopTransferIDs = []int{7, 8}
	g.nativeShopTransferSel = 1
	g.nativeShopTransferTop = 2
	g.shopPicking = true
	g.shopEquipPrompt = true
	g.shopEquipUnit = 7
	g.shopEquipSlot = 6
	g.shopRecipients = []int{7}
	g.shopRecipientSel = 1
	g.shopSellPicking = true
	g.shopSellUnitSel = 2
	g.shopSellSlotSel = 3
	g.shopMode = "sell"
	g.nativeItemPanel = ebiten.NewImage(1, 1)
	g.nativeItemPanelBase = []byte{1}
	g.nativeItemPanelRecord = []byte{1}
	g.nativeItemEffectRows = []byte{1}
	g.itemAnimStep = 11
	g.itemClosing = true

	g.loadGameFromSlot(2)
	restored, ok := g.partyRoster[0]
	if !ok || !reflect.DeepEqual(restored.Inventory, []int{1, 0}) ||
		!reflect.DeepEqual(restored.Equipped, []bool{true, false}) ||
		!reflect.DeepEqual(restored.InventorySlots, []int{1, 0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}) ||
		!reflect.DeepEqual(restored.NativeInventoryFlags, []int{0x40, 0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}) ||
		restored.AP != 32 || !restored.EquipmentBaseSet {
		t.Fatalf("native shop mutation did not survive JSON round-trip: %#v", restored)
	}
	if g.camp.NodeID() != "town" || g.handlerChapter != 1 ||
		len(g.partyJoinOrder) != 1 || !g.partyMembers[0] || !g.partyDeploy[0] {
		t.Fatalf("native shop party boundary did not restore: node=%q chapter=%d members=%#v order=%#v deploy=%#v", g.camp.NodeID(), g.handlerChapter, g.partyMembers, g.partyJoinOrder, g.partyDeploy)
	}
	if g.nativeShopUIJob != nil || g.nativeShopMode != "" || g.nativeShopVariant != 0 ||
		g.nativeShopHasPendingUnit || len(g.nativeShopSellItemIDs) != 0 ||
		g.nativeShopTransferSource != -1 || len(g.nativeShopTransferItems) != 0 ||
		len(g.nativeShopTransferIDs) != 0 || g.shopPicking || g.shopEquipPrompt ||
		len(g.shopRecipients) != 0 || g.shopSellPicking || g.shopMode != "" ||
		g.nativeItemPanel != nil || len(g.nativeItemPanelBase) != 0 ||
		len(g.nativeItemPanelRecord) != 0 || len(g.nativeItemEffectRows) != 0 ||
		g.itemAnimStep != 0 || g.itemClosing {
		t.Fatalf("town load retained shop transient state: mode=%q variant=%d transfer=%d itemPanel=%v", g.nativeShopMode, g.nativeShopVariant, g.nativeShopTransferSource, g.nativeItemPanel != nil)
	}
}

// assertNativeShopTransactionTownRoundTrip 從正式交易完成點穿越離店、城鎮與
// 重製 JSON 冷讀檔。重新進店只為讓同一原始資產 fixture 繼續驗證下一種交易；
// 它不算在持久化證據內。
func assertNativeShopTransactionTownRoundTrip(
	t *testing.T, g *Game, slot, wantGold int, want battle.Unit,
) {
	t.Helper()
	g.partyMembers = map[int]bool{0: true}
	g.partyJoinOrder = []int{0}
	g.partyDeploy = map[int]bool{0: true}
	g.handlerChapter = 1
	g.nativeShopVariant = 1
	g.nativeShopUIJob = nil
	g.leaveShop()
	if g.camp.NodeID() != "town" || g.campSel != 1 {
		t.Fatalf("shop transaction leave boundary=(%q,%d), want town/1", g.camp.NodeID(), g.campSel)
	}
	g.saveGameToSlot(slot)
	if g.msg != "已存檔(槽位"+strconv.Itoa(slot+1)+"：town)" {
		t.Fatalf("shop transaction save message=%q", g.msg)
	}

	g.gold = -1
	g.partyRoster = nil
	g.partyMembers = nil
	g.partyJoinOrder = nil
	g.partyDeploy = nil
	g.nativeShopMode = "sell_success"
	g.nativeShopHasPendingUnit = true
	g.loadGameFromSlot(slot)
	restored, ok := g.partyRoster[0]
	if !ok || g.gold != wantGold ||
		!reflect.DeepEqual(restored.Inventory, want.Inventory) ||
		!reflect.DeepEqual(restored.Equipped, want.Equipped) ||
		!reflect.DeepEqual(restored.InventorySlots, want.InventorySlots) ||
		!reflect.DeepEqual(restored.NativeInventoryFlags, want.NativeInventoryFlags) ||
		restored.AP != want.AP || restored.DP != want.DP ||
		restored.HIT != want.HIT || restored.EV != want.EV ||
		restored.EquipmentBaseSet != want.EquipmentBaseSet {
		t.Fatalf("shop transaction JSON round-trip=(gold %d unit %#v), want gold %d unit %#v", g.gold, restored, wantGold, want)
	}
	if g.camp.NodeID() != "town" || len(g.partyJoinOrder) != 1 ||
		!g.partyMembers[0] || !g.partyDeploy[0] ||
		g.nativeShopMode != "" || g.nativeShopHasPendingUnit {
		t.Fatalf("shop transaction town restore leaked state: node=%q order=%#v mode=%q pending=%v", g.camp.NodeID(), g.partyJoinOrder, g.nativeShopMode, g.nativeShopHasPendingUnit)
	}

	// 後續仍需共用這個高成本原始資產 fixture；回到 shop 後不把此步驟當成
	// 一般玩家或存檔可達性證據。
	g.camp.Cur = "shop"
	g.enterNode()
	g.nativeShopUIJob = nil
}

func TestNativeShopShotStateIsStrictAndStableMenuOnly(t *testing.T) {
	for _, tc := range []struct {
		spec    string
		service int
		pulse   int
		gold    int
		ok      bool
	}{
		{spec: "0,0,0", service: 0, pulse: 0, gold: 0, ok: true},
		{spec: "3,3,99999999", service: 3, pulse: 3, gold: 99999999, ok: true},
		{spec: "-1,0,0"},
		{spec: "4,0,0"},
		{spec: "0,-1,0"},
		{spec: "0,4,0"},
		{spec: "0,0,-1"},
		{spec: "0,0,100000000"},
		{spec: "1"},
		{spec: "1,2"},
		{spec: "1,2,3,4"},
		{spec: "x,0,0"},
	} {
		service, pulse, gold, ok := parseNativeShopShotState(tc.spec)
		if ok != tc.ok || service != tc.service ||
			pulse != tc.pulse || gold != tc.gold {
			t.Fatalf(
				"parseNativeShopShotState(%q)=(%d,%d,%d,%v), want (%d,%d,%d,%v)",
				tc.spec, service, pulse, gold, ok,
				tc.service, tc.pulse, tc.gold, tc.ok,
			)
		}
	}

	c := &campaign.Campaign{
		Start: "shop",
		Nodes: map[string]*campaign.Node{
			"shop": {Type: "shop", NativeHubVariant: 5},
		},
	}
	g := &Game{
		camp: campaign.NewRunner(c),
		nativeShopUI: &nativeShopUIAssets{
			shops:     map[int]*campaign.NativeShopAssets{5: {}},
			portraits: map[int]dato.Frame{5: {}},
		},
		nativeShopVariant:    5,
		nativeShopMode:       "menu",
		nativeShopUIJob:      &nativeClassUIJob{},
		nativeShopUIPulse:    1,
		nativeShopUILastTick: 123,
		nativeShopUIHasTick:  true,
	}
	if !g.setNativeShopShotState(2, 3, 456) {
		t.Fatal("native shop screenshot state rejected")
	}
	if g.nativeShopServiceSel != 2 || g.nativeShopUIPulse != 3 ||
		g.gold != 456 ||
		g.nativeShopUIJob != nil || g.nativeShopUIHasTick ||
		g.nativeShopUILastTick != 0 {
		t.Fatalf(
			"shop shot state=(service %d pulse %d gold %d job %v last %d has %v)",
			g.nativeShopServiceSel, g.nativeShopUIPulse, g.gold,
			g.nativeShopUIJob != nil, g.nativeShopUILastTick,
			g.nativeShopUIHasTick,
		)
	}

	g.nativeShopMode = "purchase"
	if g.setNativeShopShotState(0, 0, 0) {
		t.Fatal("non-menu shop screenshot state accepted")
	}
	g.nativeShopMode = "menu"
	g.nativeShopVariant = 3
	if g.setNativeShopShotState(0, 0, 0) {
		t.Fatal("mismatched native shop variant accepted")
	}
}

func TestNativeShopPurchaseShotStateIsStrictAndWindowBound(t *testing.T) {
	for _, tc := range []struct {
		spec      string
		selection int
		start     int
		gold      int
		ok        bool
	}{
		{spec: "0,0,0", selection: 0, start: 0, gold: 0, ok: true},
		{spec: "7,2,99999999", selection: 7, start: 2, gold: 99999999, ok: true},
		{spec: "-1,0,0"},
		{spec: "0,-1,0"},
		{spec: "0,1,0"},
		{spec: "0,0,-1"},
		{spec: "0,0,100000000"},
		{spec: "0,0"},
		{spec: "0,0,0,0"},
		{spec: "x,0,0"},
	} {
		selection, start, gold, ok :=
			parseNativeShopPurchaseShotState(tc.spec)
		if ok != tc.ok || selection != tc.selection ||
			start != tc.start || gold != tc.gold {
			t.Fatalf(
				"parseNativeShopPurchaseShotState(%q)=(%d,%d,%d,%v), want (%d,%d,%d,%v)",
				tc.spec, selection, start, gold, ok,
				tc.selection, tc.start, tc.gold, tc.ok,
			)
		}
	}

	c := &campaign.Campaign{
		Start: "shop",
		Nodes: map[string]*campaign.Node{
			"shop": {
				Type: "shop", NativeHubVariant: 1,
				Goods: []campaign.Good{
					{ID: 0x80}, {ID: 0x81}, {ID: 0x84}, {ID: 0xa5},
				},
			},
		},
	}
	g := &Game{
		camp: campaign.NewRunner(c),
		nativeShopUI: &nativeShopUIAssets{
			shops:     map[int]*campaign.NativeShopAssets{1: {}},
			portraits: map[int]dato.Frame{1: {}},
		},
		nativeShopVariant: 1,
		nativeShopMode:    "menu",
		nativeShopUIJob:   &nativeClassUIJob{},
	}
	if !g.setNativeShopPurchaseShotState(2, 0, 456) {
		t.Fatal("native purchase screenshot state rejected")
	}
	if g.nativeShopMode != "purchase" || g.shopSel != 2 ||
		g.nativeShopItemStart != 0 || g.gold != 456 ||
		g.nativeShopUIJob != nil {
		t.Fatalf(
			"purchase shot state=(mode %q selection %d start %d gold %d job %v)",
			g.nativeShopMode, g.shopSel, g.nativeShopItemStart, g.gold,
			g.nativeShopUIJob != nil,
		)
	}

	g.nativeShopMode = "menu"
	if g.setNativeShopPurchaseShotState(4, 0, 0) {
		t.Fatal("out-of-range purchase selection accepted")
	}
	if g.setNativeShopPurchaseShotState(0, 2, 0) {
		t.Fatal("non-normalized purchase window accepted")
	}
	g.nativeShopVariant = 3
	if g.setNativeShopPurchaseShotState(0, 0, 0) {
		t.Fatal("mismatched purchase shop variant accepted")
	}
}

func TestNativeShopConfirmShotStateIsStrictAndGoodBound(t *testing.T) {
	for _, tc := range []struct {
		spec                string
		good, choice, pulse int
		gold                int
		ok                  bool
	}{
		{spec: "0,0,0,0", ok: true},
		{spec: "7,1,3,99999999", good: 7, choice: 1, pulse: 3, gold: 99999999, ok: true},
		{spec: "-1,0,0,0"},
		{spec: "0,-1,0,0"},
		{spec: "0,2,0,0"},
		{spec: "0,0,-1,0"},
		{spec: "0,0,4,0"},
		{spec: "0,0,0,-1"},
		{spec: "0,0,0,100000000"},
		{spec: "0,0,0"},
		{spec: "x,0,0,0"},
	} {
		good, choice, pulse, gold, ok :=
			parseNativeShopConfirmShotState(tc.spec)
		if ok != tc.ok || good != tc.good || choice != tc.choice ||
			pulse != tc.pulse || gold != tc.gold {
			t.Fatalf(
				"parseNativeShopConfirmShotState(%q)=(%d,%d,%d,%d,%v), want (%d,%d,%d,%d,%v)",
				tc.spec, good, choice, pulse, gold, ok,
				tc.good, tc.choice, tc.pulse, tc.gold, tc.ok,
			)
		}
	}

	c := &campaign.Campaign{
		Start: "shop",
		Nodes: map[string]*campaign.Node{
			"shop": {
				Type: "shop", NativeHubVariant: 1,
				Goods: []campaign.Good{{ID: 0x80}, {ID: 0x81}},
			},
		},
	}
	g := &Game{
		camp: campaign.NewRunner(c),
		nativeClassUI: &nativeClassUIAssets{
			choices:  make([]fdother.RawCell, 53),
			dialogue: make([]fdother.RawCell, 18),
			strings:  &fdtxt.Strings{},
			font:     &fdtxt.Font{},
		},
		nativeShopUI: &nativeShopUIAssets{
			shops:     map[int]*campaign.NativeShopAssets{1: {}},
			portraits: map[int]dato.Frame{1: {}},
		},
		nativeShopVariant:   1,
		nativeShopMode:      "menu",
		nativeShopUIJob:     &nativeClassUIJob{},
		nativeShopUIHasTick: true,
	}
	if !g.setNativeShopConfirmShotState(1, 1, 2, 456) {
		t.Fatal("native confirmation screenshot state rejected")
	}
	if g.nativeShopMode != "confirm" || g.shopSel != 1 ||
		g.nativeShopConfirmSel != 1 || g.nativeShopUIPulse != 2 ||
		g.gold != 456 || g.nativeShopUIJob != nil ||
		g.nativeShopUIHasTick {
		t.Fatalf(
			"confirm shot state=(mode %q good %d choice %d pulse %d gold %d job %v tick %v)",
			g.nativeShopMode, g.shopSel, g.nativeShopConfirmSel,
			g.nativeShopUIPulse, g.gold, g.nativeShopUIJob != nil,
			g.nativeShopUIHasTick,
		)
	}

	g.nativeShopMode = "menu"
	if g.setNativeShopConfirmShotState(2, 0, 0, 0) {
		t.Fatal("out-of-range confirmation good accepted")
	}
	g.nativeShopVariant = 3
	if g.setNativeShopConfirmShotState(0, 0, 0, 0) {
		t.Fatal("mismatched confirmation shop variant accepted")
	}
	g.nativeShopVariant = 1
	g.nativeClassUI.choices = nil
	if g.setNativeShopConfirmShotState(0, 0, 0, 0) {
		t.Fatal("confirmation accepted incomplete shared choice assets")
	}
}

func TestNativeShopInsufficientShotStateRequiresRealFailedPrice(t *testing.T) {
	for _, tc := range []struct {
		spec       string
		good, gold int
		ok         bool
	}{
		{spec: "0,0", ok: true},
		{spec: "7,99999999", good: 7, gold: 99999999, ok: true},
		{spec: "-1,0"},
		{spec: "0,-1"},
		{spec: "0,100000000"},
		{spec: "0"},
		{spec: "0,0,0"},
		{spec: "x,0"},
	} {
		good, gold, ok := parseNativeShopInsufficientShotState(tc.spec)
		if ok != tc.ok || good != tc.good || gold != tc.gold {
			t.Fatalf(
				"parseNativeShopInsufficientShotState(%q)=(%d,%d,%v), want (%d,%d,%v)",
				tc.spec, good, gold, ok, tc.good, tc.gold, tc.ok,
			)
		}
	}

	c := &campaign.Campaign{
		Start: "shop",
		Nodes: map[string]*campaign.Node{
			"shop": {
				Type: "shop", NativeHubVariant: 1,
				Goods: []campaign.Good{{ID: 0x80, Price: 50}},
			},
		},
	}
	g := &Game{
		camp: campaign.NewRunner(c),
		nativeClassUI: &nativeClassUIAssets{
			choices:  make([]fdother.RawCell, 53),
			dialogue: make([]fdother.RawCell, 18),
			strings:  &fdtxt.Strings{},
			font:     &fdtxt.Font{},
		},
		nativeShopUI: &nativeShopUIAssets{
			shops:     map[int]*campaign.NativeShopAssets{1: {}},
			portraits: map[int]dato.Frame{1: {}},
		},
		nativeShopVariant: 1,
		nativeShopMode:    "menu",
	}
	// The synthetic assets cannot compose a visible feedback frame, so the
	// strict final compositor admission must reject them.
	if g.setNativeShopInsufficientShotState(0, 0) {
		t.Fatal("insufficient screenshot state accepted incomplete assets")
	}
	if g.nativeShopMode != "menu" {
		t.Fatalf("final compositor rejection mutated mode=%q", g.nativeShopMode)
	}

	g.nativeShopUI = nil
	if g.setNativeShopInsufficientShotState(0, 0) {
		t.Fatal("insufficient screenshot state accepted missing shop assets")
	}
	g.nativeShopUI = &nativeShopUIAssets{
		shops:     map[int]*campaign.NativeShopAssets{1: {}},
		portraits: map[int]dato.Frame{1: {}},
	}
	if g.setNativeShopInsufficientShotState(0, 50) {
		t.Fatal("insufficient screenshot state accepted affordable price")
	}
	if g.setNativeShopInsufficientShotState(1, 0) {
		t.Fatal("insufficient screenshot state accepted out-of-range good")
	}
}

func TestNativeShopProductionOwnerFailsClosedForCustomVariant(t *testing.T) {
	g := &Game{
		camp: campaign.NewRunner(&campaign.Campaign{
			Start: "shop",
			Nodes: map[string]*campaign.Node{
				"shop": {Type: "shop", NativeHubVariant: 0},
			},
		}),
	}
	if g.setupNativeShop() || g.nativeShopMode != "" {
		t.Fatal("custom shop node was incorrectly claimed by original shop owner")
	}
}
