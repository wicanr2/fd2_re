package main

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

type nativeShopUIAssets struct {
	shops      map[int]*campaign.NativeShopAssets
	portraits  map[int]dato.Frame
	itemAssets battle.NativeItemPanelDataAssets
	effectRows []byte
}

// advanceNativeShopEquipmentRecipient is the pure transition used by the
// production equipment-recipient input path. Up and down are deliberately
// bounded; horizontal input reaches this transition as up=false/down=false
// and is therefore a no-op. NativeThreeRowWindow remains the sole owner of
// the stateful three-row viewport.
func advanceNativeShopEquipmentRecipient(
	count, selected, start int, up, down bool,
) (nextSelected, nextStart int, ok bool) {
	if count <= 0 || selected < 0 || selected >= count || start < 0 {
		return selected, start, false
	}
	if up && selected > 0 {
		selected--
	}
	if down && selected+1 < count {
		selected++
	}
	start, visible := campaign.NativeThreeRowWindow(count, selected, start)
	if visible <= 0 {
		return selected, start, false
	}
	return selected, start, true
}

func parseNativeShopShotState(spec string) (service, pulse, gold int, ok bool) {
	parts := strings.Split(spec, ",")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	service, err := strconv.Atoi(parts[0])
	if err != nil || service < 0 || service > 3 {
		return 0, 0, 0, false
	}
	pulse, err = strconv.Atoi(parts[1])
	if err != nil || pulse < 0 || pulse > 3 {
		return 0, 0, 0, false
	}
	gold, err = strconv.Atoi(parts[2])
	if err != nil || gold < 0 || gold > 99999999 {
		return 0, 0, 0, false
	}
	return service, pulse, gold, true
}

func parseNativeShopPurchaseShotState(
	spec string,
) (selection, start, gold int, ok bool) {
	parts := strings.Split(spec, ",")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	selection, err := strconv.Atoi(parts[0])
	if err != nil || selection < 0 {
		return 0, 0, 0, false
	}
	start, err = strconv.Atoi(parts[1])
	if err != nil || start < 0 || start%2 != 0 {
		return 0, 0, 0, false
	}
	gold, err = strconv.Atoi(parts[2])
	if err != nil || gold < 0 || gold > 99999999 {
		return 0, 0, 0, false
	}
	return selection, start, gold, true
}

func parseNativeShopConfirmShotState(
	spec string,
) (good, choice, pulse, gold int, ok bool) {
	parts := strings.Split(spec, ",")
	if len(parts) != 4 {
		return 0, 0, 0, 0, false
	}
	good, err := strconv.Atoi(parts[0])
	if err != nil || good < 0 {
		return 0, 0, 0, 0, false
	}
	choice, err = strconv.Atoi(parts[1])
	if err != nil || choice < 0 || choice > 1 {
		return 0, 0, 0, 0, false
	}
	pulse, err = strconv.Atoi(parts[2])
	if err != nil || pulse < 0 || pulse > 3 {
		return 0, 0, 0, 0, false
	}
	gold, err = strconv.Atoi(parts[3])
	if err != nil || gold < 0 || gold > 99999999 {
		return 0, 0, 0, 0, false
	}
	return good, choice, pulse, gold, true
}

func parseNativeShopInsufficientShotState(
	spec string,
) (good, gold int, ok bool) {
	parts := strings.Split(spec, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	good, err := strconv.Atoi(parts[0])
	if err != nil || good < 0 {
		return 0, 0, false
	}
	gold, err = strconv.Atoi(parts[1])
	if err != nil || gold < 0 || gold > 99999999 {
		return 0, 0, false
	}
	return good, gold, true
}

func parseNativeShopEquipmentRecipientShotState(
	spec string,
) (good, selection, start, cycle, gold int, ok bool) {
	parts := strings.Split(spec, ",")
	if len(parts) != 5 {
		return 0, 0, 0, 0, 0, false
	}
	values := make([]int, len(parts))
	for i, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return 0, 0, 0, 0, 0, false
		}
		values[i] = value
	}
	if values[3] > 2 || values[4] > 99999999 {
		return 0, 0, 0, 0, 0, false
	}
	return values[0], values[1], values[2], values[3], values[4], true
}

func parseNativeShopSellShotState(
	spec string,
) (mode string, unit, selection, start, cycle, gold int, ok bool) {
	parts := strings.Split(spec, ",")
	if len(parts) != 6 || (parts[0] != "roster" && parts[0] != "items") {
		return "", 0, 0, 0, 0, 0, false
	}
	values := make([]int, 5)
	for i, part := range parts[1:] {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return "", 0, 0, 0, 0, 0, false
		}
		values[i] = value
	}
	if values[3] > 2 || values[4] > 99999999 {
		return "", 0, 0, 0, 0, 0, false
	}
	return parts[0], values[0], values[1], values[2], values[3], values[4], true
}

// setNativeShopShotState is a screenshot-only oracle hook. It may select a
// stable service-menu frame after setupNativeShop has claimed a proven native
// shop node. Gold is an explicit visible-state input for the one captured
// frame; the hook never synthesizes a shop, advances the campaign, or executes
// a transaction.
func (g *Game) setNativeShopShotState(service, pulse, gold int) bool {
	if service < 0 || service > 3 || pulse < 0 || pulse > 3 ||
		gold < 0 || gold > 99999999 ||
		g.camp == nil || g.nativeShopUI == nil ||
		g.nativeShopMode != "menu" {
		return false
	}
	n := g.camp.Node()
	if n == nil || n.Type != "shop" ||
		n.NativeHubVariant != g.nativeShopVariant {
		return false
	}
	if _, ok := g.nativeShopUI.shops[n.NativeHubVariant]; !ok {
		return false
	}
	if _, ok := g.nativeShopUI.portraits[n.NativeHubVariant]; !ok {
		return false
	}
	g.nativeShopUIJob = nil
	g.nativeShopServiceSel = service
	g.resetNativeShopUIPulse()
	g.nativeShopUIPulse = pulse
	g.gold = gold
	return true
}

// setNativeShopPurchaseShotState is the stable purchase-panel counterpart to
// setNativeShopShotState. It accepts the native raw selection/window state
// only after the production shop owner has claimed the editable node.
func (g *Game) setNativeShopPurchaseShotState(
	selection, start, gold int,
) bool {
	if selection < 0 || start < 0 || start%2 != 0 ||
		gold < 0 || gold > 99999999 ||
		g.camp == nil || g.nativeShopUI == nil ||
		g.nativeShopMode != "menu" {
		return false
	}
	n := g.camp.Node()
	if n == nil || n.Type != "shop" ||
		n.NativeHubVariant != g.nativeShopVariant {
		return false
	}
	if _, ok := g.nativeShopUI.shops[n.NativeHubVariant]; !ok {
		return false
	}
	if _, ok := g.nativeShopUI.portraits[n.NativeHubVariant]; !ok {
		return false
	}
	goods := g.camp.ShopGoods()
	nextStart, visible := campaign.NativeTwoColumnWindow(
		len(goods), selection, start,
	)
	if visible == 0 || nextStart != start {
		return false
	}
	g.nativeShopUIJob = nil
	g.nativeShopMode = "purchase"
	g.shopSel = selection
	g.nativeShopItemStart = start
	g.gold = gold
	return true
}

// setNativeShopConfirmShotState is a screenshot-only stable-state adapter. It
// selects a real editable good and original Yes/No pulse only after the
// production native shop owner has admitted the node and its resources.
func (g *Game) setNativeShopConfirmShotState(
	good, choice, pulse, gold int,
) bool {
	if good < 0 || choice < 0 || choice > 1 ||
		pulse < 0 || pulse > 3 ||
		gold < 0 || gold > 99999999 ||
		g.camp == nil || g.nativeShopUI == nil ||
		g.nativeClassUI == nil ||
		len(g.nativeClassUI.choices) <= 52 ||
		len(g.nativeClassUI.dialogue) <= 17 ||
		g.nativeClassUI.strings == nil ||
		g.nativeClassUI.font == nil ||
		g.nativeShopMode != "menu" {
		return false
	}
	n := g.camp.Node()
	if n == nil || n.Type != "shop" ||
		n.NativeHubVariant != g.nativeShopVariant ||
		good >= len(g.camp.ShopGoods()) {
		return false
	}
	if _, ok := g.nativeShopUI.shops[n.NativeHubVariant]; !ok {
		return false
	}
	if _, ok := g.nativeShopUI.portraits[n.NativeHubVariant]; !ok {
		return false
	}
	g.nativeShopUIJob = nil
	g.nativeShopMode = "confirm"
	g.shopSel = good
	g.nativeShopConfirmSel = choice
	g.nativeShopUIPulse = pulse
	g.nativeShopUIHasTick = false
	g.gold = gold
	return true
}

// setNativeShopInsufficientShotState exposes only the stable feedback reached
// after a real editable good fails the original affordability comparison.
// It never debits gold or mutates a recipient.
func (g *Game) setNativeShopInsufficientShotState(good, gold int) bool {
	if good < 0 || gold < 0 || gold > 99999999 ||
		g.camp == nil || g.nativeShopUI == nil ||
		g.nativeClassUI == nil ||
		len(g.nativeClassUI.choices) <= 52 ||
		len(g.nativeClassUI.dialogue) <= 17 ||
		g.nativeClassUI.strings == nil ||
		g.nativeClassUI.font == nil ||
		g.nativeShopMode != "menu" {
		return false
	}
	n := g.camp.Node()
	goods := g.camp.ShopGoods()
	if n == nil || n.Type != "shop" ||
		n.NativeHubVariant != g.nativeShopVariant ||
		good >= len(goods) || gold >= goods[good].Price {
		return false
	}
	if _, ok := g.nativeShopUI.shops[n.NativeHubVariant]; !ok {
		return false
	}
	if _, ok := g.nativeShopUI.portraits[n.NativeHubVariant]; !ok {
		return false
	}
	oldMode, oldSel, oldGold, oldJob :=
		g.nativeShopMode, g.shopSel, g.gold, g.nativeShopUIJob
	g.nativeShopUIJob = nil
	g.nativeShopMode = "insufficient"
	g.shopSel = good
	g.gold = gold
	_, ok := g.composeNativeShopInsufficientGold()
	if !ok {
		g.nativeShopMode = oldMode
		g.shopSel = oldSel
		g.gold = oldGold
		g.nativeShopUIJob = oldJob
	}
	return ok
}

// setNativeShopEquipmentRecipientShotState exposes only a recipient state
// admitted from an already materialized persistent party. The companion
// screenshot bootstrap may derive that party from a compiled LOADCH binding,
// but this adapter never invents names, classes, order, or stats.
func (g *Game) setNativeShopEquipmentRecipientShotState(
	good, selection, start, cycle, gold int,
) bool {
	if good < 0 || selection < 0 || start < 0 || cycle < 0 || cycle > 2 ||
		gold < 0 || gold > 99999999 ||
		g.camp == nil || g.nativeShopUI == nil ||
		g.nativeClassUI == nil || g.nativeShopMode != "menu" {
		return false
	}
	goods := g.camp.ShopGoods()
	if good >= len(goods) || gold < goods[good].Price {
		return false
	}
	oldMode, oldShopSel, oldRecipientSel, oldRecipientStart, oldRecipientCycle,
		oldGold, oldJob :=
		g.nativeShopMode, g.shopSel, g.shopRecipientSel,
		g.nativeShopRecipientStart, g.nativeShopRecipientCycle,
		g.gold, g.nativeShopUIJob
	oldRecipients := append([]int(nil), g.shopRecipients...)
	rollback := func() {
		g.nativeShopMode = oldMode
		g.shopSel = oldShopSel
		g.shopRecipientSel = oldRecipientSel
		g.nativeShopRecipientStart = oldRecipientStart
		g.nativeShopRecipientCycle = oldRecipientCycle
		g.gold = oldGold
		g.nativeShopUIJob = oldJob
		g.shopRecipients = oldRecipients
	}

	g.nativeShopUIJob = nil
	g.shopSel = good
	g.gold = gold
	if !g.setupNativeShopRecipients() ||
		g.nativeShopMode != "recipient_equipment" ||
		selection >= len(g.shopRecipients) {
		rollback()
		return false
	}
	normalizedStart, visible := campaign.NativeThreeRowWindow(
		len(g.shopRecipients), selection, start,
	)
	if visible == 0 || normalizedStart != start {
		rollback()
		return false
	}
	g.shopRecipientSel = selection
	g.nativeShopRecipientStart = start
	g.nativeShopRecipientCycle = cycle
	if _, ok := g.composeNativeShopEquipmentRecipient(); !ok {
		rollback()
		return false
	}
	return true
}

// setNativeShopSellShotState exposes only stable states already owned by the
// production sell compositor. The caller must have materialized a complete
// typed/raw party; this adapter never invents actors, items, flags, or prices.
func (g *Game) setNativeShopSellShotState(
	mode string, unit, selection, start, cycle, gold int,
) bool {
	if (mode != "roster" && mode != "items") || unit < 0 ||
		selection < 0 || start < 0 || cycle < 0 || cycle > 2 ||
		gold < 0 || gold > 99999999 ||
		g.camp == nil || g.nativeShopUI == nil ||
		g.nativeClassUI == nil || g.nativeShopMode != "menu" {
		return false
	}
	n := g.camp.Node()
	if n == nil || n.Type != "shop" ||
		n.NativeHubVariant != g.nativeShopVariant {
		return false
	}
	if _, ok := g.nativeShopUI.shops[n.NativeHubVariant]; !ok {
		return false
	}
	if _, ok := g.nativeShopUI.portraits[n.NativeHubVariant]; !ok {
		return false
	}

	oldMode, oldUnit, oldSlot, oldRosterTop, oldRosterCycle,
		oldItemTop, oldGold, oldJob :=
		g.nativeShopMode, g.shopSellUnitSel, g.shopSellSlotSel,
		g.nativeShopSellRosterTop, g.nativeShopSellRosterCycle,
		g.nativeShopSellItemTop,
		g.gold, g.nativeShopUIJob
	oldItems := append([]int(nil), g.nativeShopSellItemIDs...)
	rollback := func() {
		g.nativeShopMode = oldMode
		g.shopSellUnitSel = oldUnit
		g.shopSellSlotSel = oldSlot
		g.nativeShopSellRosterTop = oldRosterTop
		g.nativeShopSellRosterCycle = oldRosterCycle
		g.nativeShopSellItemTop = oldItemTop
		g.gold = oldGold
		g.nativeShopUIJob = oldJob
		g.nativeShopSellItemIDs = oldItems
	}

	g.nativeShopUIJob = nil
	g.gold = gold
	g.nativeShopSellRosterCycle = cycle
	if !g.setupNativeShopSellRoster() || unit >= len(g.partyJoinOrder) {
		rollback()
		return false
	}
	g.shopSellUnitSel = unit
	if mode == "roster" {
		normalized, visible := campaign.NativeTwoColumnWindow(
			len(g.partyJoinOrder), selection, start,
		)
		if selection != unit || visible == 0 || normalized != start {
			rollback()
			return false
		}
		g.nativeShopSellRosterTop = start
		if _, ok := g.composeNativeShopSellRoster(); !ok {
			rollback()
			return false
		}
		return true
	}
	if !g.setupNativeShopSellItems() {
		rollback()
		return false
	}
	normalized, visible := campaign.NativeTwoColumnWindow(
		len(g.nativeShopSellItemIDs), selection, start,
	)
	if visible == 0 || normalized != start {
		rollback()
		return false
	}
	g.shopSellSlotSel = selection
	g.nativeShopSellItemTop = start
	if _, ok := g.composeNativeShopSellItems(); !ok {
		rollback()
		return false
	}
	return true
}

func loadNativeShopUIAssets(
	shared *nativeClassUIAssets,
) (*nativeShopUIAssets, error) {
	if shared == nil || shared.units == nil || shared.strings == nil ||
		shared.font == nil || len(shared.dialogue) <= 17 ||
		len(shared.digits) != 10 {
		return nil, errors.New("native shop UI: shared facility assets unavailable")
	}
	fdotherPath := nativeFDOTHERPath()
	if fdotherPath == "" {
		return nil, errors.New("native shop UI: FDOTHER.DAT unavailable")
	}
	base := filepath.Dir(fdotherPath)
	out := &nativeShopUIAssets{
		shops:     make(map[int]*campaign.NativeShopAssets, 3),
		portraits: make(map[int]dato.Frame, 3),
	}
	for variant, resourceID := range map[int]int{1: 12, 3: 29, 5: 63} {
		assets, err := campaign.DecodeNativeShopAssets(fdotherPath, resourceID)
		if err != nil {
			return nil, err
		}
		out.shops[variant] = assets
	}
	for variant, portraitID := range map[int]int{1: 0x80, 3: 0x82, 5: 0x84} {
		frames, err := dato.DecodeResource(
			filepath.Join(base, "DATO.DAT"), portraitID,
		)
		if err != nil || len(frames) == 0 {
			if err != nil {
				return nil, err
			}
			return nil, errors.New("native shop UI: DATO portrait has no frames")
		}
		out.portraits[variant] = frames[0]
	}
	var err error
	out.itemAssets, err = battle.LoadNativeItemPanelDataAssets(
		fdotherPath, filepath.Join(base, "FDTXT.DAT"),
	)
	if err != nil {
		return nil, err
	}
	out.effectRows, err = battle.LoadNativeItemEffectRowPrefix(
		assetPath("assets/data/native_item_effect_rows.json"),
	)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (g *Game) setupNativeShop() bool {
	n := g.camp.Node()
	if n == nil || n.Type != "shop" || n.NativeHubVariant == 0 ||
		g.nativeShopUI == nil {
		return false
	}
	if _, ok := g.nativeShopUI.shops[n.NativeHubVariant]; !ok {
		return false
	}
	if _, ok := g.nativeShopUI.portraits[n.NativeHubVariant]; !ok {
		return false
	}
	g.nativeShopVariant = n.NativeHubVariant
	g.nativeShopMode = "menu"
	g.nativeShopServiceSel = 0
	g.nativeShopItemStart = 0
	g.nativeShopConfirmSel = 0
	g.shopSel = 0
	g.resetNativeShopUIPulse()
	return g.beginNativeShopServiceOpening()
}

func (g *Game) nativeShopState() (
	*campaign.NativeShopAssets, dato.Frame, int, bool,
) {
	if g.nativeShopUI == nil || g.nativeClassUI == nil ||
		g.nativeShopVariant != 1 && g.nativeShopVariant != 3 &&
			g.nativeShopVariant != 5 {
		return nil, dato.Frame{}, 0, false
	}
	assets, assetsOK := g.nativeShopUI.shops[g.nativeShopVariant]
	portrait, portraitOK := g.nativeShopUI.portraits[g.nativeShopVariant]
	portraitID := map[int]int{1: 0x80, 3: 0x82, 5: 0x84}[g.nativeShopVariant]
	return assets, portrait, portraitID, assetsOK && portraitOK
}

func (g *Game) composeNativeShopStable() ([]byte, bool) {
	assets, portrait, portraitID, ok := g.nativeShopState()
	if !ok {
		return nil, false
	}
	textIndex := 440
	if g.nativeShopVariant == 1 {
		textIndex = 501
	}
	shared := g.nativeClassUI
	frame, err := campaign.ComposeNativeShopScene(
		assets, shared.dialogue, shared.digits, portrait, portraitID,
		shared.strings, shared.font, g.gold, textIndex,
	)
	return frame, err == nil
}

func (g *Game) composeNativeShopBare() ([]byte, bool) {
	assets, _, _, ok := g.nativeShopState()
	if !ok || g.nativeClassUI == nil {
		return nil, false
	}
	frame, err := campaign.ComposeNativeShopBareScene(
		assets, g.nativeClassUI.digits, g.gold,
	)
	return frame, err == nil
}

func (g *Game) composeNativeShopServiceMenu() ([]byte, bool) {
	assets, _, _, ok := g.nativeShopState()
	stable, stableOK := g.composeNativeShopStable()
	if !ok || !stableOK || g.nativeShopMode != "menu" ||
		g.nativeShopServiceSel < 0 || g.nativeShopServiceSel > 3 {
		return nil, false
	}
	frame, err := campaign.ComposeNativeShopServiceSteadyFrame(
		stable, assets, g.nativeShopServiceSel, g.nativeShopUIPulse,
	)
	return frame, err == nil
}

func (g *Game) composeNativeShopPurchaseList() ([]byte, bool) {
	if g.nativeShopMode != "purchase" || g.camp == nil {
		return nil, false
	}
	assets, _, _, ok := g.nativeShopState()
	stable, stableOK := g.composeNativeShopStable()
	goods := g.camp.ShopGoods()
	if !ok || !stableOK || len(goods) == 0 ||
		g.shopSel < 0 || g.shopSel >= len(goods) {
		return nil, false
	}
	start, visible := campaign.NativeTwoColumnWindow(
		len(goods), g.shopSel, g.nativeShopItemStart,
	)
	if visible == 0 {
		return nil, false
	}
	g.nativeShopItemStart = start
	itemIDs := make([]int, len(goods))
	for i, good := range goods {
		itemIDs[i] = good.ID
	}
	frame, err := campaign.ComposeNativeShopItemListFrame(
		stable, assets, g.nativeShopUI.itemAssets,
		itemIDs, start, g.shopSel, g.nativeShopUI.effectRows,
		battle.NativeFacilityFullPrice,
	)
	return frame, err == nil
}

func (g *Game) nativeShopSelectedGood() (campaign.Good, bool) {
	if g.camp == nil {
		return campaign.Good{}, false
	}
	goods := g.camp.ShopGoods()
	if g.shopSel < 0 || g.shopSel >= len(goods) {
		return campaign.Good{}, false
	}
	return goods[g.shopSel], true
}

func (g *Game) composeNativeShopPurchaseQuestion() ([]byte, bool) {
	good, goodOK := g.nativeShopSelectedGood()
	_, portrait, portraitID, stateOK := g.nativeShopState()
	stable, stableOK := g.composeNativeShopStable()
	if !goodOK || !stateOK || !stableOK {
		return nil, false
	}
	shared := g.nativeClassUI
	frame, err := campaign.ComposeNativeShopPurchaseMessage(
		stable, shared.dialogue, portrait, portraitID,
		shared.strings, shared.font, campaign.NativeShopPurchaseQuestion,
		g.nativeShopVariant, good.ID, good.Price,
	)
	return frame, err == nil
}

func (g *Game) composeNativeShopPurchaseConfirmation() ([]byte, bool) {
	if g.nativeShopMode != "confirm" ||
		g.nativeShopConfirmSel < 0 || g.nativeShopConfirmSel > 1 {
		return nil, false
	}
	question, ok := g.composeNativeShopPurchaseQuestion()
	if !ok {
		return nil, false
	}
	frame, err := campaign.ComposeNativeConfirmationChoices(
		question, g.nativeClassUI.choices,
		g.nativeShopConfirmSel, g.nativeShopUIPulse/2,
	)
	return frame, err == nil
}

func (g *Game) nativeShopPostChoiceCloseFrame() ([]byte, bool) {
	question, ok := g.composeNativeShopPurchaseQuestion()
	if !ok {
		return nil, false
	}
	frames, err := campaign.NativeClassConfirmationClosingFrames(
		question, g.nativeClassUI.choices,
	)
	if err != nil || len(frames) != 4 {
		return nil, false
	}
	// 0x197e5 presents the four inward choice frames, then 0x19913 restores
	// the saved 310x86 question region before returning to its caller.
	return question, true
}

func (g *Game) composeNativeShopInsufficientGold() ([]byte, bool) {
	if g.nativeShopMode != "insufficient" {
		return nil, false
	}
	postChoiceClose, ok := g.nativeShopPostChoiceCloseFrame()
	if !ok {
		return nil, false
	}
	frame, err := campaign.ComposeNativeShopPurchaseInsufficientGold(
		postChoiceClose, g.nativeClassUI.strings, g.nativeClassUI.font,
		g.nativeShopVariant,
	)
	if err != nil || len(g.nativeClassUI.dialogue) <= 18 {
		return nil, false
	}
	// 0x16c57(mode=1) initially blits FDOTHER#5 cell18 as the wait marker.
	// The ch02 variant1 DOSBox oracle fixes this caller-owned VGA anchor.
	if err := g.nativeClassUI.dialogue[18].BlitOpaqueAtOffset(
		frame, 320, 181*320+143,
	); err != nil {
		return nil, false
	}
	return frame, true
}

func (g *Game) beginNativeShopServiceOpening() bool {
	assets, _, _, ok := g.nativeShopState()
	stable, stableOK := g.composeNativeShopStable()
	if !ok || !stableOK {
		return false
	}
	frames := make([][]byte, 4)
	for step := range frames {
		var err error
		frames[step], err = campaign.ComposeNativeShopServiceOpeningFrame(
			stable, assets, step,
		)
		if err != nil {
			return false
		}
	}
	g.resetNativeShopUIPulse()
	g.nativeShopUIJob = &nativeClassUIJob{frames: frames}
	return true
}

func (g *Game) beginNativeShopServiceClosing(after func()) bool {
	assets, _, _, ok := g.nativeShopState()
	stable, stableOK := g.composeNativeShopStable()
	if !ok || !stableOK {
		return false
	}
	frames := make([][]byte, 4)
	for step := range frames {
		var err error
		frames[step], err = campaign.ComposeNativeShopServiceClosingFrame(
			stable, assets, step,
		)
		if err != nil {
			return false
		}
	}
	g.nativeShopUIJob = &nativeClassUIJob{
		frames: frames, restore: stable, after: after,
	}
	return true
}

func (g *Game) beginNativeShopPurchaseOpening() bool {
	final, ok := g.composeNativeShopPurchaseList()
	if !ok {
		return false
	}
	stable, ok := g.composeNativeShopStable()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassListOpeningFrames(stable, final)
	if err != nil || len(frames) != 6 {
		return false
	}
	g.nativeShopUIJob = &nativeClassUIJob{frames: frames}
	return true
}

func (g *Game) beginNativeShopPurchaseClosing(after func()) bool {
	final, ok := g.composeNativeShopPurchaseList()
	if !ok {
		return false
	}
	stable, ok := g.composeNativeShopStable()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassListClosingFrames(stable, final)
	if err != nil || len(frames) != 5 {
		return false
	}
	g.nativeShopUIJob = &nativeClassUIJob{
		frames: frames, restore: stable, after: after,
	}
	return true
}

func (g *Game) beginNativeShopConfirmationOpening() bool {
	question, ok := g.composeNativeShopPurchaseQuestion()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassConfirmationOpeningFrames(
		question, g.nativeClassUI.choices,
	)
	if err != nil || len(frames) != 4 {
		return false
	}
	g.resetNativeShopUIPulse()
	g.nativeShopUIJob = &nativeClassUIJob{frames: frames}
	return true
}

func (g *Game) beginNativeShopConfirmationChoiceClosing(after func()) bool {
	question, ok := g.composeNativeShopPurchaseQuestion()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassConfirmationClosingFrames(
		question, g.nativeClassUI.choices,
	)
	if err != nil || len(frames) != 4 {
		return false
	}
	g.nativeShopUIJob = &nativeClassUIJob{frames: frames, after: after}
	return true
}

func (g *Game) beginNativeShopDialogueClosing(
	composed []byte,
	after func(),
) bool {
	stable, ok := g.composeNativeShopStable()
	if !ok || len(composed) != 320*200 {
		return false
	}
	frames, err := campaign.NativeClassListClosingFrames(stable, composed)
	if err != nil || len(frames) != 5 {
		return false
	}
	g.nativeShopUIJob = &nativeClassUIJob{
		frames: frames, restore: stable, after: after,
	}
	return true
}

func (g *Game) returnToNativeShopPurchaseList() {
	g.nativeShopMode = "purchase"
	if !g.beginNativeShopPurchaseOpening() {
		g.nativeShopMode = ""
		g.msg = "原版商店商品清單無法還原"
	}
}

func (g *Game) stepNativeShopUILifecycle(now time.Time) {
	if g.nativeShopMode == "equip_panel" {
		g.stepNativeShopEquipPanelLifecycle()
	}
	job := g.nativeShopUIJob
	if job != nil && len(job.timeline) != 0 {
		if job.started.IsZero() {
			job.started = now
		}
		job.elapsed = now.Sub(job.started)
		if job.frame == 1 && job.drawn {
			after := job.after
			g.nativeShopUIJob = nil
			if after != nil {
				after()
			}
		}
		return
	}
	if job != nil && job.drawn {
		job.drawn = false
		if job.frame < len(job.frames) {
			job.frame++
			if job.frame < len(job.frames) || len(job.restore) != 0 {
				return
			}
		}
		if job.frame >= len(job.frames) {
			after := job.after
			g.nativeShopUIJob = nil
			if after != nil {
				after()
			}
		}
	}
	if g.nativeShopUIJob == nil &&
		(g.nativeShopMode == "menu" || g.nativeShopMode == "confirm" ||
			g.nativeShopMode == "equip_confirm" ||
			g.nativeShopMode == "sell_confirm") {
		g.stepNativeShopUIPulseTick(g.nativeShopUIClock.Sample(now))
	}
}

func (g *Game) drawNativeShopUIJob(screen *ebiten.Image) bool {
	job := g.nativeShopUIJob
	if job == nil {
		return false
	}
	if len(job.timeline) != 0 {
		elapsed := job.elapsed
		total := time.Duration(0)
		for _, candidate := range job.timeline {
			total += candidate.duration
		}
		step := job.timeline[len(job.timeline)-1]
		for _, candidate := range job.timeline {
			if elapsed < candidate.duration {
				step = candidate
				break
			}
			elapsed -= candidate.duration
		}
		if job.elapsed >= total {
			job.frame = 1
		}
		g.presentNativeClassFrameWithPalette(screen, step.frame, step.palette)
		job.drawn = true
		return true
	}
	if job.frame < len(job.frames) {
		g.presentNativeClassFrame(screen, job.frames[job.frame])
		job.drawn = true
		return true
	}
	if len(job.restore) == 320*200 {
		g.presentNativeClassFrame(screen, job.restore)
		job.drawn = true
		return true
	}
	return false
}

func (g *Game) drawNativeShop(screen *ebiten.Image) bool {
	if g.nativeShopMode == "" {
		return false
	}
	if g.drawNativeShopUIJob(screen) {
		return true
	}
	var frame []byte
	var ok bool
	switch g.nativeShopMode {
	case "menu":
		frame, ok = g.composeNativeShopServiceMenu()
	case "purchase":
		frame, ok = g.composeNativeShopPurchaseList()
	case "confirm":
		frame, ok = g.composeNativeShopPurchaseConfirmation()
	case "insufficient":
		frame, ok = g.composeNativeShopInsufficientGold()
	case "recipient_consumable", "recipient_equipment":
		frame, ok = g.composeNativeShopRecipient()
	case "recipient_full":
		frame, ok = g.composeNativeShopRecipientFull()
	case "no_recipient":
		frame, ok = g.composeNativeShopNoEligibleRecipient()
	case "equip_confirm":
		frame, ok = g.composeNativeShopEquipConfirmation()
	case "sell_roster":
		frame, ok = g.composeNativeShopSellRoster()
	case "sell_items":
		frame, ok = g.composeNativeShopSellItems()
	case "sell_empty":
		frame, ok = g.composeNativeShopSellEmpty()
	case "sell_confirm":
		frame, ok = g.composeNativeShopSellConfirmation()
	case "equip_roster":
		frame, ok = g.composeNativeShopEquipRoster()
	case "equip_panel":
		frame, ok = g.composeNativeShopStable()
	case "transfer_intro", "transfer_empty", "transfer_dest_prompt", "transfer_full":
		frame, ok = g.composeNativeShopTransferMessage()
	case "transfer_source", "transfer_dest":
		frame, ok = g.composeNativeShopTransferRoster()
	case "transfer_items":
		frame, ok = g.composeNativeShopTransferItems()
	}
	if !ok {
		return false
	}
	g.presentNativeClassFrame(screen, frame)
	if g.nativeShopMode == "equip_panel" {
		return g.drawNativeItemPanel(screen)
	}
	return true
}

func (g *Game) nativeShopUIBlocksInput() bool {
	return g.nativeShopUIJob != nil
}

func (g *Game) resetNativeShopUIPulse() {
	g.nativeShopUIClock.Reset()
	g.nativeShopUIPulse = 2
	g.nativeShopUILastTick = 0
	g.nativeShopUIHasTick = false
}

func (g *Game) stepNativeShopUIPulseTick(rawTick int) {
	if !g.nativeShopUIHasTick {
		g.nativeShopUILastTick = rawTick
		g.nativeShopUIHasTick = true
		return
	}
	delta := int16(uint16(rawTick) - uint16(g.nativeShopUILastTick))
	if delta < 2 {
		return
	}
	g.nativeShopUILastTick = rawTick
	g.nativeShopUIPulse = (g.nativeShopUIPulse + 1) & 3
}

func (g *Game) handleNativeShopInput(enter bool) bool {
	if g.nativeShopMode == "" {
		return false
	}
	if g.nativeShopUIBlocksInput() {
		return true
	}
	switch g.nativeShopMode {
	case "menu":
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
			g.nativeShopServiceSel = (g.nativeShopServiceSel + 3) % 4
			g.resetNativeShopUIPulse()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			g.nativeShopServiceSel = (g.nativeShopServiceSel + 1) % 4
			g.resetNativeShopUIPulse()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if !g.beginNativeShopServiceClosing(g.leaveShop) {
				g.leaveShop()
			}
			return true
		}
		if enter {
			if g.nativeShopServiceSel > 3 {
				g.msg = "此原版商店服務的 production owner 尚未接線"
				return true
			}
			open := func() {
				if g.nativeShopServiceSel == 0 {
					g.nativeShopMode = "purchase"
					g.shopSel = 0
					g.nativeShopItemStart = 0
					g.beginNativeShopPurchaseOpening()
					return
				}
				if g.nativeShopServiceSel == 1 {
					if !g.setupNativeShopSellRoster() ||
						!g.beginNativeShopSellRosterOpening() {
						g.nativeShopMode = ""
						g.msg = "原版商店 sell roster 無法還原"
					}
					return
				}
				if g.nativeShopServiceSel == 2 {
					if !g.setupNativeShopEquipRoster() ||
						!g.beginNativeShopEquipRosterOpening() {
						g.nativeShopMode = ""
						g.msg = "原版商店 equip roster 無法還原"
					}
					return
				}
				if !g.setupNativeShopTransfer() {
					g.nativeShopMode = ""
					g.msg = "原版商店 transfer owner 無法還原"
				}
			}
			if !g.beginNativeShopServiceClosing(open) {
				open()
			}
			return true
		}
	case "purchase":
		goods := g.camp.ShopGoods()
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) && g.shopSel > 0 {
			g.shopSel--
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) &&
			g.shopSel+1 < len(goods) {
			g.shopSel++
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && g.shopSel >= 2 {
			g.shopSel -= 2
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) &&
			g.shopSel+2 < len(goods) {
			g.shopSel += 2
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			openMenu := func() {
				g.nativeShopMode = "menu"
				g.nativeShopServiceSel = 0
				g.beginNativeShopServiceOpening()
			}
			if !g.beginNativeShopPurchaseClosing(openMenu) {
				openMenu()
			}
			return true
		}
		if enter && len(goods) != 0 {
			openConfirm := func() {
				g.nativeShopMode = "confirm"
				g.nativeShopConfirmSel = 0
				if !g.beginNativeShopConfirmationOpening() {
					g.nativeShopMode = ""
					g.msg = "原版購買確認視窗無法還原"
				}
			}
			if !g.beginNativeShopPurchaseClosing(openConfirm) {
				openConfirm()
			}
			return true
		}
	case "confirm":
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
			g.nativeShopConfirmSel = campaign.AdvanceNativeClassConfirmation(
				g.nativeShopConfirmSel, -1,
			)
			g.resetNativeShopUIPulse()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			g.nativeShopConfirmSel = campaign.AdvanceNativeClassConfirmation(
				g.nativeShopConfirmSel, 1,
			)
			g.resetNativeShopUIPulse()
		}
		cancel := inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
			enter && g.nativeShopConfirmSel == 1
		if cancel {
			closeDialogue := func() {
				postChoiceClose, ok := g.nativeShopPostChoiceCloseFrame()
				if !ok || !g.beginNativeShopDialogueClosing(
					postChoiceClose, g.returnToNativeShopPurchaseList,
				) {
					g.returnToNativeShopPurchaseList()
				}
			}
			if !g.beginNativeShopConfirmationChoiceClosing(closeDialogue) {
				closeDialogue()
			}
			return true
		}
		if enter {
			good, ok := g.nativeShopSelectedGood()
			if !ok {
				return true
			}
			afterChoiceClose := func() {
				if g.gold < good.Price {
					g.nativeShopMode = "insufficient"
					return
				}
				postChoiceClose, frameOK := g.nativeShopPostChoiceCloseFrame()
				openRecipient := func() {
					if !g.setupNativeShopRecipients() {
						g.msg = "原版購買 recipient 缺少 raw 候選資料"
						g.returnToNativeShopPurchaseList()
						return
					}
					if g.nativeShopMode == "no_recipient" {
						if !g.beginNativeShopNoEligibleRecipientOpening() {
							g.msg = "原版購買無合適角色訊息無法還原"
							g.returnToNativeShopPurchaseList()
						}
						return
					}
					if !g.beginNativeShopRecipientOpening() {
						g.msg = "原版購買 recipient 面板無法還原"
						g.returnToNativeShopPurchaseList()
					}
				}
				if !frameOK || !g.beginNativeShopDialogueClosing(
					postChoiceClose, openRecipient,
				) {
					openRecipient()
				}
			}
			if !g.beginNativeShopConfirmationChoiceClosing(afterChoiceClose) {
				afterChoiceClose()
			}
			return true
		}
	case "insufficient":
		if enter || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			frame, ok := g.composeNativeShopInsufficientGold()
			if !ok || !g.beginNativeShopDialogueClosing(
				frame, g.returnToNativeShopPurchaseList,
			) {
				g.returnToNativeShopPurchaseList()
			}
			return true
		}
	case "recipient_consumable", "recipient_equipment":
		count := len(g.shopRecipients)
		if g.nativeShopMode == "recipient_consumable" {
			delta := 0
			switch {
			case inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft):
				delta = -1
			case inpututil.IsKeyJustPressed(ebiten.KeyArrowRight):
				delta = 1
			case inpututil.IsKeyJustPressed(ebiten.KeyArrowUp):
				delta = -2
			case inpututil.IsKeyJustPressed(ebiten.KeyArrowDown):
				delta = 2
			}
			if delta != 0 {
				g.shopRecipientSel = campaign.AdvanceNativeTwoColumnSelection(
					g.shopRecipientSel, count, delta,
				)
				g.nativeShopRecipientStart, _ = campaign.NativeTwoColumnWindow(
					count, g.shopRecipientSel, g.nativeShopRecipientStart,
				)
			}
		} else {
			nextSelection, nextStart, ok := advanceNativeShopEquipmentRecipient(
				count,
				g.shopRecipientSel,
				g.nativeShopRecipientStart,
				inpututil.IsKeyJustPressed(ebiten.KeyArrowUp),
				inpututil.IsKeyJustPressed(ebiten.KeyArrowDown),
			)
			if !ok {
				g.msg = "原版購買 recipient 游標狀態無效"
				g.returnToNativeShopPurchaseList()
				return true
			}
			g.shopRecipientSel = nextSelection
			g.nativeShopRecipientStart = nextStart
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if !g.beginNativeShopRecipientClosing(
				g.returnToNativeShopPurchaseList,
			) {
				g.returnToNativeShopPurchaseList()
			}
			return true
		}
		if enter && count != 0 {
			unit := g.partyRoster[g.shopRecipients[g.shopRecipientSel]]
			if nativeShopInventoryFull(unit) {
				openFull := func() {
					g.nativeShopMode = "recipient_full"
					if !g.beginNativeShopRecipientFullOpening() {
						g.msg = "原版購買滿欄訊息無法還原"
						g.returnToNativeShopPurchaseList()
					}
				}
				if !g.beginNativeShopRecipientClosing(openFull) {
					openFull()
				}
				return true
			}
			beginTransaction := func() {
				if !g.stageNativeShopPurchase() {
					g.nativeShopHasPendingUnit = false
					g.nativeShopPendingUnit = battle.Unit{}
					g.msg = "原版購買交易缺少 raw 資料"
					g.returnToNativeShopPurchaseList()
				}
			}
			if !g.beginNativeShopRecipientClosing(beginTransaction) {
				beginTransaction()
			}
			return true
		}
	case "recipient_full":
		if enter || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if !g.beginNativeShopRecipientFullClosing(
				g.returnToNativeShopPurchaseList,
			) {
				g.returnToNativeShopPurchaseList()
			}
			return true
		}
	case "no_recipient":
		if enter || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if !g.beginNativeShopNoEligibleRecipientClosing(
				g.returnToNativeShopPurchaseList,
			) {
				g.returnToNativeShopPurchaseList()
			}
			return true
		}
	case "equip_confirm":
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
			g.nativeShopEquipSel = campaign.AdvanceNativeClassConfirmation(
				g.nativeShopEquipSel, -1,
			)
			g.resetNativeShopUIPulse()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			g.nativeShopEquipSel = campaign.AdvanceNativeClassConfirmation(
				g.nativeShopEquipSel, 1,
			)
			g.resetNativeShopUIPulse()
		}
		cancel := inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
			enter && g.nativeShopEquipSel == 1
		if cancel || enter {
			if enter && !cancel {
				staged := cloneNativeShopUnit(g.nativeShopPendingUnit)
				if err := campaign.EquipItem(
					&staged, g.shopEquipSlot, g.shopItemStats,
				); err != nil {
					g.nativeShopHasPendingUnit = false
					g.nativeShopPendingUnit = battle.Unit{}
					g.msg = err.Error()
					if !g.beginNativeShopEquipConfirmationClosing(
						g.returnToNativeShopPurchaseList,
					) {
						g.returnToNativeShopPurchaseList()
					}
					return true
				}
				g.nativeShopPendingUnit = staged
			}
			afterPrompt := func() {
				if !g.beginNativeShopPurchaseSuccess() {
					g.nativeShopHasPendingUnit = false
					g.nativeShopPendingUnit = battle.Unit{}
					g.msg = "原版購買成功演出無法還原"
					g.returnToNativeShopPurchaseList()
				}
			}
			if !g.beginNativeShopEquipConfirmationClosing(afterPrompt) {
				afterPrompt()
			}
			return true
		}
	case "sell_roster":
		count := len(g.partyJoinOrder)
		delta := 0
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft):
			delta = -1
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowRight):
			delta = 1
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowUp):
			delta = -2
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowDown):
			delta = 2
		}
		if delta != 0 {
			g.shopSellUnitSel = campaign.AdvanceNativeTwoColumnSelection(
				g.shopSellUnitSel, count, delta,
			)
			g.nativeShopSellRosterTop, _ = campaign.NativeTwoColumnWindow(
				count, g.shopSellUnitSel, g.nativeShopSellRosterTop,
			)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			openMenu := func() {
				g.nativeShopMode = "menu"
				g.nativeShopServiceSel = 1
				g.beginNativeShopServiceOpening()
			}
			if !g.beginNativeShopSellRosterClosing(openMenu) {
				openMenu()
			}
			return true
		}
		if enter && count != 0 {
			_, unit, ok := g.nativeShopSellUnit()
			items, rawOK := nativeShopActiveItemIDs(unit)
			if !ok || !rawOK {
				g.msg = "原版 sell inventory 缺少 raw 資料"
				return true
			}
			openChild := func() {
				if len(items) == 0 {
					g.nativeShopMode = "sell_empty"
					if !g.beginNativeShopSellEmptyOpening() {
						g.msg = "原版 sell empty 訊息無法還原"
						g.setupNativeShopSellRoster()
					}
					return
				}
				if !g.setupNativeShopSellItems() ||
					!g.beginNativeShopSellItemsOpening() {
					g.msg = "原版 sell item list 無法還原"
					g.setupNativeShopSellRoster()
				}
			}
			if !g.beginNativeShopSellRosterClosing(openChild) {
				openChild()
			}
			return true
		}
	case "sell_empty":
		if enter || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			returnRoster := func() {
				g.returnToNativeShopSellRoster()
			}
			if !g.beginNativeShopSellEmptyClosing(returnRoster) {
				returnRoster()
			}
			return true
		}
	case "sell_items":
		count := len(g.nativeShopSellItemIDs)
		delta := 0
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft):
			delta = -1
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowRight):
			delta = 1
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowUp):
			delta = -2
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowDown):
			delta = 2
		}
		if delta != 0 {
			g.shopSellSlotSel = campaign.AdvanceNativeTwoColumnSelection(
				g.shopSellSlotSel, count, delta,
			)
			g.nativeShopSellItemTop, _ = campaign.NativeTwoColumnWindow(
				count, g.shopSellSlotSel, g.nativeShopSellItemTop,
			)
		}
		returnRoster := func() {
			g.returnToNativeShopSellRoster()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if !g.beginNativeShopSellItemsClosing(returnRoster) {
				returnRoster()
			}
			return true
		}
		if enter && count != 0 {
			openConfirm := func() {
				g.nativeShopMode = "sell_confirm"
				g.nativeShopSellConfirmSel = 0
				if !g.beginNativeShopSellConfirmationOpening() {
					g.msg = "原版 sell confirmation 無法還原"
					returnRoster()
				}
			}
			if !g.beginNativeShopSellItemsClosing(openConfirm) {
				openConfirm()
			}
			return true
		}
	case "sell_confirm":
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
			g.nativeShopSellConfirmSel =
				campaign.AdvanceNativeClassConfirmation(
					g.nativeShopSellConfirmSel, -1,
				)
			g.resetNativeShopUIPulse()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			g.nativeShopSellConfirmSel =
				campaign.AdvanceNativeClassConfirmation(
					g.nativeShopSellConfirmSel, 1,
				)
			g.resetNativeShopUIPulse()
		}
		cancel := inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
			enter && g.nativeShopSellConfirmSel == 1
		if cancel {
			returnRoster := func() {
				g.returnToNativeShopSellRoster()
			}
			if !g.beginNativeShopSellConfirmationClosing(returnRoster) {
				returnRoster()
			}
			return true
		}
		if enter {
			afterConfirm := func() {
				if !g.beginNativeShopSellSuccess() {
					g.msg = "原版 sell transaction 無法還原"
					g.returnToNativeShopSellRoster()
				}
			}
			if !g.beginNativeShopSellConfirmationClosing(afterConfirm) {
				afterConfirm()
			}
			return true
		}
	case "equip_roster":
		count := len(g.partyJoinOrder)
		delta := 0
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft):
			delta = -1
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowRight):
			delta = 1
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowUp):
			delta = -2
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowDown):
			delta = 2
		}
		if delta != 0 {
			g.nativeShopEquipUnitSel = campaign.AdvanceNativeTwoColumnSelection(
				g.nativeShopEquipUnitSel, count, delta,
			)
			g.nativeShopEquipRosterTop, _ = campaign.NativeTwoColumnWindow(
				count, g.nativeShopEquipUnitSel,
				g.nativeShopEquipRosterTop,
			)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			openMenu := func() {
				g.nativeShopMode = "menu"
				g.nativeShopServiceSel = 2
				g.beginNativeShopServiceOpening()
			}
			if !g.beginNativeShopEquipRosterClosing(openMenu) {
				openMenu()
			}
			return true
		}
		if enter && count != 0 {
			openPanel := func() {
				if !g.openNativeShopEquipPanel() {
					g.nativeShopMode = ""
					g.msg = "原版商店 equip item panel 無法還原"
				}
			}
			if !g.beginNativeShopEquipRosterClosing(openPanel) {
				openPanel()
			}
			return true
		}
	case "equip_panel":
		if g.nativeShopEquipPanelBlocksInput() {
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.beginNativeShopEquipPanelClose()
			return true
		}
		_, unit, ok := g.nativeShopEquipUnit()
		if !ok {
			return true
		}
		rawSlots := nativeItemRawSlots(&unit)
		if len(rawSlots) != 0 {
			key := 0
			switch {
			case inpututil.IsKeyJustPressed(ebiten.KeyArrowUp):
				key = 72
			case inpututil.IsKeyJustPressed(ebiten.KeyArrowDown):
				key = 80
			case inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft):
				key = 75
			case inpututil.IsKeyJustPressed(ebiten.KeyArrowRight):
				key = 77
			}
			if key != 0 {
				selected, _, err := battle.AdvanceNativeItemSelector(
					g.itemSel, len(rawSlots), key, false, 0,
				)
				if err == nil && selected != g.itemSel {
					g.itemSel = selected
					g.refreshNativeItemPanelMode(&unit, true)
				}
			}
		}
		if enter && !g.applyNativeShopEquipSelection() {
			g.msg = "原版商店 equip transaction 缺少 raw 對映"
		}
		return true
	case "transfer_intro":
		if enter || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if !g.beginNativeShopTransferMessageClosing(
				g.openNativeShopTransferSourceRoster,
			) {
				g.openNativeShopTransferSourceRoster()
			}
		}
		return true
	case "transfer_empty", "transfer_full":
		if enter || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			after := func() {
				if !g.returnToNativeShopTransferLoop() {
					g.nativeShopMode = ""
					g.msg = "原版商店 transfer source prompt 無法還原"
				}
			}
			if !g.beginNativeShopTransferMessageClosing(after) {
				after()
			}
		}
		return true
	case "transfer_dest_prompt":
		if enter || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if !g.beginNativeShopTransferMessageClosing(
				g.openNativeShopTransferDestinationRoster,
			) {
				g.openNativeShopTransferDestinationRoster()
			}
		}
		return true
	case "transfer_source", "transfer_dest":
		count := len(g.nativeShopTransferIDs)
		delta := 0
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft):
			delta = -1
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowRight):
			delta = 1
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowUp):
			delta = -2
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowDown):
			delta = 2
		}
		if delta != 0 {
			g.nativeShopTransferSel = campaign.AdvanceNativeTwoColumnSelection(
				g.nativeShopTransferSel, count, delta,
			)
			g.nativeShopTransferTop, _ = campaign.NativeTwoColumnWindow(
				count, g.nativeShopTransferSel, g.nativeShopTransferTop,
			)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if g.nativeShopMode == "transfer_source" {
				openMenu := func() {
					g.nativeShopMode = "menu"
					g.nativeShopServiceSel = 3
					g.beginNativeShopServiceOpening()
				}
				if !g.beginNativeShopTransferRosterClosing(openMenu) {
					openMenu()
				}
			} else {
				after := func() {
					if !g.returnToNativeShopTransferLoop() {
						g.nativeShopMode = ""
					}
				}
				if !g.beginNativeShopTransferRosterClosing(after) {
					after()
				}
			}
			return true
		}
		if enter && count != 0 {
			selectedID := g.nativeShopTransferIDs[g.nativeShopTransferSel]
			if g.nativeShopMode == "transfer_source" {
				afterSource := func() {
					g.nativeShopTransferSource = selectedID
					unit := g.partyRoster[selectedID]
					items, ok := nativeShopTransferItemSlots(unit)
					if !ok {
						g.nativeShopMode = ""
						g.msg = "原版 transfer source raw 對映不一致"
						return
					}
					if len(items) == 0 {
						g.nativeShopMode = "transfer_empty"
						if !g.beginNativeShopTransferMessageOpening() {
							g.nativeShopMode = ""
						}
						return
					}
					if !g.openNativeShopTransferItems() {
						g.nativeShopMode = ""
					}
				}
				if !g.beginNativeShopTransferRosterClosing(afterSource) {
					afterSource()
				}
				return true
			}
			afterDestination := func() {
				destination := g.partyRoster[selectedID]
				count, err := battle.NativeInventoryAvailableCount(
					destination.NativeInventoryFlags,
				)
				if err != nil {
					g.nativeShopMode = ""
					g.msg = "原版 transfer destination flags 無法驗證"
					return
				}
				if count == 8 {
					g.nativeShopTransferDest = selectedID
					g.nativeShopMode = "transfer_full"
					if !g.beginNativeShopTransferMessageOpening() {
						g.nativeShopMode = ""
					}
					return
				}
				if !g.applyNativeShopTransfer(selectedID) ||
					!g.returnToNativeShopTransferLoop() {
					g.nativeShopMode = ""
					g.msg = "原版 transfer transaction 無法還原"
				}
			}
			if !g.beginNativeShopTransferRosterClosing(afterDestination) {
				afterDestination()
			}
			return true
		}
	case "transfer_items":
		count := len(g.nativeShopTransferItems)
		delta := 0
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft):
			delta = -1
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowRight):
			delta = 1
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowUp):
			delta = -2
		case inpututil.IsKeyJustPressed(ebiten.KeyArrowDown):
			delta = 2
		}
		if delta != 0 {
			g.nativeShopTransferSel = campaign.AdvanceNativeTwoColumnSelection(
				g.nativeShopTransferSel, count, delta,
			)
			g.nativeShopTransferTop, _ = campaign.NativeTwoColumnWindow(
				count, g.nativeShopTransferSel, g.nativeShopTransferTop,
			)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			after := func() {
				if !g.returnToNativeShopTransferLoop() {
					g.nativeShopMode = ""
				}
			}
			if !g.beginNativeShopTransferItemsClosing(after) {
				after()
			}
			return true
		}
		if enter && count != 0 {
			compact := g.nativeShopTransferItems[g.nativeShopTransferSel]
			openPrompt := func() {
				g.nativeShopTransferItem = compact
				g.nativeShopMode = "transfer_dest_prompt"
				if !g.beginNativeShopTransferMessageOpening() {
					g.nativeShopMode = ""
				}
			}
			if !g.beginNativeShopTransferItemsClosing(openPrompt) {
				openPrompt()
			}
			return true
		}
	}
	return true
}
