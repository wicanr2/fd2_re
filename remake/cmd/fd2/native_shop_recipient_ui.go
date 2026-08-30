package main

import (
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func (g *Game) setupNativeShopRecipients() bool {
	good, ok := g.nativeShopSelectedGood()
	if !ok || len(g.partyJoinOrder) == 0 {
		return false
	}
	itemType, ok := g.shopItemTypes[good.ID]
	if !ok {
		return false
	}
	ids := make([]int, 0, len(g.partyJoinOrder))
	for _, id := range g.partyJoinOrder {
		unit, ok := g.partyRoster[id]
		if !ok || !unit.HasNativeIdentity || !unit.HasMapSelectorKey ||
			len(unit.InventorySlots) != 8 ||
			len(unit.NativeInventoryFlags) != 8 {
			return false
		}
		if itemType < 0x20 {
			if !unit.HasNativeRecordClass ||
				!campaign.CanEquip(
					int(unit.NativeRecordClass), itemType, g.shopEquipTypes,
				) {
				continue
			}
		}
		ids = append(ids, id)
	}
	g.shopRecipients = ids
	g.shopRecipientSel = 0
	g.nativeShopRecipientStart = 0
	if len(ids) == 0 {
		g.nativeShopMode = "no_recipient"
		return true
	}
	if itemType < 0x20 {
		g.nativeShopMode = "recipient_equipment"
	} else {
		g.nativeShopMode = "recipient_consumable"
	}
	return true
}

func (g *Game) nativeShopRecipientState() (
	campaign.Good,
	int,
	[]int,
	bool,
) {
	good, goodOK := g.nativeShopSelectedGood()
	itemType, typeOK := g.shopItemTypes[good.ID]
	if !goodOK || !typeOK || len(g.shopRecipients) == 0 ||
		g.shopRecipientSel < 0 ||
		g.shopRecipientSel >= len(g.shopRecipients) {
		return campaign.Good{}, 0, nil, false
	}
	return good, itemType, g.shopRecipients, true
}

func (g *Game) composeNativeShopConsumableRecipient() ([]byte, bool) {
	_, itemType, ids, ok := g.nativeShopRecipientState()
	if !ok || g.nativeShopMode != "recipient_consumable" ||
		itemType < 0x20 {
		return nil, false
	}
	start, visible := campaign.NativeTwoColumnWindow(
		len(ids), g.shopRecipientSel, g.nativeShopRecipientStart,
	)
	if visible == 0 {
		return nil, false
	}
	g.nativeShopRecipientStart = start
	rows := make([]campaign.NativeRosterRow, 0, visible)
	identities := make([]int, 0, visible)
	for i := 0; i < visible; i++ {
		unit, ok := g.partyRoster[ids[start+i]]
		if !ok || !unit.HasNativeIdentity || !unit.HasMapSelectorKey {
			return nil, false
		}
		sprite, err := g.nativeClassUI.units.SpriteFor(
			unit.MapSelectorKey, 0, g.nativeShopRecipientCycle,
		)
		if err != nil {
			return nil, false
		}
		rows = append(rows, campaign.NativeRosterRow{
			Sprite: sprite, NameTextIndex: unit.NativeIdentity + 1,
		})
		identities = append(identities, unit.NativeIdentity)
	}
	stable, stableOK := g.composeNativeShopStable()
	assets, _, _, stateOK := g.nativeShopState()
	if !stableOK || !stateOK {
		return nil, false
	}
	var frame []byte
	var err error
	if g.localeID != "" && g.localeID != "zh-Hant" {
		frame, err = g.composeLocalizedNativeRoster(
			stable, assets.Panel, rows, identities, g.shopRecipientSel-start,
		)
	} else {
		frame, err = campaign.ComposeNativeShopConsumableRecipientFrame(
			stable, assets, rows, g.shopRecipientSel-start, itemType,
			g.nativeClassUI.strings, g.nativeClassUI.font,
		)
	}
	return frame, err == nil
}

func (g *Game) composeNativeShopEquipmentRecipient() ([]byte, bool) {
	good, itemType, ids, ok := g.nativeShopRecipientState()
	if !ok || g.nativeShopMode != "recipient_equipment" ||
		itemType >= 0x20 {
		return nil, false
	}
	start, visible := campaign.NativeThreeRowWindow(
		len(ids), g.shopRecipientSel, g.nativeShopRecipientStart,
	)
	if visible == 0 {
		return nil, false
	}
	g.nativeShopRecipientStart = start
	rows := make([]campaign.NativeShopEquipmentRecipientRow, 0, visible)
	for i := 0; i < visible; i++ {
		unit, ok := g.partyRoster[ids[start+i]]
		if !ok || !unit.HasNativeIdentity || !unit.HasMapSelectorKey {
			return nil, false
		}
		sprite, err := g.nativeClassUI.units.SpriteFor(
			unit.MapSelectorKey, 0, g.nativeShopRecipientCycle,
		)
		if err != nil {
			return nil, false
		}
		record, err := campaign.NativeShopEquipmentRecordForUnit(&unit)
		if err != nil {
			return nil, false
		}
		current, err := campaign.NativeShopEquipmentCurrentStats(record)
		if err != nil {
			return nil, false
		}
		candidate, err := campaign.NativeShopEquipmentCandidateStats(
			record, good.ID, g.nativeShopUI.effectRows,
		)
		if err != nil {
			return nil, false
		}
		rows = append(rows, campaign.NativeShopEquipmentRecipientRow{
			Sprite:        sprite,
			NameTextIndex: unit.NativeIdentity + 1,
			Current:       current,
			Candidate:     candidate,
		})
	}
	stable, stableOK := g.composeNativeShopStable()
	assets, _, _, stateOK := g.nativeShopState()
	if !stableOK || !stateOK {
		return nil, false
	}
	frame, err := campaign.ComposeNativeShopEquipmentRecipientFrame(
		stable, assets, g.nativeShopUI.itemAssets,
		rows, g.shopRecipientSel-start,
	)
	return frame, err == nil
}

func (g *Game) composeNativeShopRecipient() ([]byte, bool) {
	switch g.nativeShopMode {
	case "recipient_consumable":
		return g.composeNativeShopConsumableRecipient()
	case "recipient_equipment":
		return g.composeNativeShopEquipmentRecipient()
	default:
		return nil, false
	}
}

func (g *Game) beginNativeShopRecipientOpening() bool {
	final, ok := g.composeNativeShopRecipient()
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

func (g *Game) beginNativeShopRecipientClosing(after func()) bool {
	final, ok := g.composeNativeShopRecipient()
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

func nativeShopInventoryFull(unit battle.Unit) bool {
	if len(unit.InventorySlots) != 8 ||
		len(unit.NativeInventoryFlags) != 8 {
		return false
	}
	for _, flag := range unit.NativeInventoryFlags {
		if flag&0x80 != 0 {
			return false
		}
	}
	return true
}

func (g *Game) composeNativeShopRecipientFull() ([]byte, bool) {
	if g.nativeShopMode != "recipient_full" ||
		g.shopRecipientSel < 0 ||
		g.shopRecipientSel >= len(g.shopRecipients) {
		return nil, false
	}
	unit, ok := g.partyRoster[g.shopRecipients[g.shopRecipientSel]]
	if !ok || unit.BattleFig < 0 || unit.BattleFig > 0xff {
		return nil, false
	}
	stable, stableOK := g.composeNativeShopStable()
	_, portrait, portraitID, stateOK := g.nativeShopState()
	if !stableOK || !stateOK {
		return nil, false
	}
	if g.localeID != "" && g.localeID != "zh-Hant" {
		if !unit.HasNativeIdentity {
			return nil, false
		}
		name, err := g.localeEntities.CharacterName(unit.NativeIdentity)
		if err != nil {
			return nil, false
		}
		frame, err := g.composeLocalizedNativeShopPlainMessage(
			stable, portrait, portraitID, "shop.recipient.full", name,
		)
		return frame, err == nil
	}
	frame, err := campaign.ComposeNativeShopPurchaseRecipientFull(
		stable, g.nativeClassUI.dialogue, portrait, portraitID,
		g.nativeClassUI.strings, g.nativeClassUI.font,
		g.nativeShopVariant, unit.BattleFig+1,
	)
	return frame, err == nil
}

func (g *Game) beginNativeShopRecipientFullOpening() bool {
	final, ok := g.composeNativeShopRecipientFull()
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

func (g *Game) beginNativeShopRecipientFullClosing(after func()) bool {
	final, ok := g.composeNativeShopRecipientFull()
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

func (g *Game) composeNativeShopNoEligibleRecipient() ([]byte, bool) {
	if g.nativeShopMode != "no_recipient" {
		return nil, false
	}
	good, goodOK := g.nativeShopSelectedGood()
	stable, stableOK := g.composeNativeShopStable()
	_, portrait, portraitID, stateOK := g.nativeShopState()
	if !goodOK || !stableOK || !stateOK {
		return nil, false
	}
	if g.localeID != "" && g.localeID != "zh-Hant" {
		key := g.localizedShopKey(
			"shop.purchase.no_recipient.weapon", "shop.purchase.no_recipient.item",
		)
		frame, err := g.composeLocalizedNativeShopPlainMessage(
			stable, portrait, portraitID, key,
		)
		return frame, err == nil
	}
	frame, err := campaign.ComposeNativeShopPurchaseMessage(
		stable, g.nativeClassUI.dialogue, portrait, portraitID,
		g.nativeClassUI.strings, g.nativeClassUI.font,
		campaign.NativeShopPurchaseNoEligibleRecipient,
		g.nativeShopVariant, good.ID, good.Price,
	)
	return frame, err == nil
}

func (g *Game) beginNativeShopNoEligibleRecipientOpening() bool {
	final, ok := g.composeNativeShopNoEligibleRecipient()
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

func (g *Game) beginNativeShopNoEligibleRecipientClosing(after func()) bool {
	final, ok := g.composeNativeShopNoEligibleRecipient()
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
