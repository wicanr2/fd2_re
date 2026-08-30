package main

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

const (
	localizedShopMessageX        = 12
	localizedShopMessageY        = 119
	localizedShopMessageWidth    = 292
	localizedShopMessageLineStep = 19
	localizedShopMessageRows     = 3
	localizedShopMessageScale    = 0.78
)

func (g *Game) localizedShopKey(weaponKey, otherKey string) string {
	if g.nativeShopVariant == 1 {
		return weaponKey
	}
	return otherKey
}

func (g *Game) composeLocalizedNativeShopStable(
	assets *campaign.NativeShopAssets,
	portrait dato.Frame,
	portraitID int,
) ([]byte, error) {
	key := g.localizedShopKey("shop.greeting.weapon", "shop.greeting.item")
	message, ok := g.localeMessage(key)
	if !ok {
		return nil, errors.New("localized shop greeting is unavailable")
	}
	frame, err := campaign.ComposeNativeShopSceneWithoutText(
		assets, g.nativeClassUI.dialogue, g.nativeClassUI.digits,
		portrait, portraitID, g.gold,
	)
	if err != nil {
		return nil, err
	}
	return g.drawLocalizedShopMessage(frame, message, localizedShopMessageY, localizedShopMessageRows)
}

func (g *Game) composeLocalizedNativeShopPurchaseQuestion(
	source []byte,
	portrait dato.Frame,
	portraitID int,
	good campaign.Good,
) ([]byte, error) {
	name, err := g.localeEntities.ItemName(good.ID)
	if err != nil {
		return nil, err
	}
	key := g.localizedShopKey("shop.purchase.question.weapon", "shop.purchase.question.item")
	message, ok := g.localeMessage(key, name, good.Price)
	if !ok {
		return nil, errors.New("localized shop purchase question is unavailable")
	}
	frame, err := campaign.ComposeNativeChurchDialogueOverlayAt(
		source, g.nativeClassUI.dialogue, portrait,
		campaign.NativeFacilityPortraitOffset(portraitID),
	)
	if err != nil {
		return nil, err
	}
	return g.drawLocalizedShopMessage(frame, message, localizedShopMessageY, localizedShopMessageRows)
}

func (g *Game) composeLocalizedNativeShopPlainMessage(
	source []byte,
	portrait dato.Frame,
	portraitID int,
	key string,
	args ...any,
) ([]byte, error) {
	message, ok := g.localeMessage(key, args...)
	if !ok {
		return nil, errors.New("localized shop message is unavailable")
	}
	frame, err := campaign.ComposeNativeChurchDialogueOverlayAt(
		source, g.nativeClassUI.dialogue, portrait,
		campaign.NativeFacilityPortraitOffset(portraitID),
	)
	if err != nil {
		return nil, err
	}
	return g.drawLocalizedShopMessage(frame, message, localizedShopMessageY, localizedShopMessageRows)
}

func (g *Game) drawLocalizedShopMessage(frame []byte, message string, y, maxRows int) ([]byte, error) {
	if g == nil || g.font == nil || len(frame) != 320*200 || message == "" || maxRows <= 0 {
		return nil, errors.New("localized shop message state is invalid")
	}
	lines := g.font.Wrap(message, localizedShopMessageScale, localizedShopMessageWidth)
	if len(lines) == 0 || len(lines) > maxRows {
		return nil, fmt.Errorf("localized shop message requires %d rows, max %d", len(lines), maxRows)
	}
	result := append([]byte(nil), frame...)
	for index, line := range lines {
		if line == "" {
			return nil, errors.New("localized shop message contains an empty row")
		}
		if err := drawIndexedLocalizedText(
			result, g.font, line, localizedShopMessageX,
			y+index*localizedShopMessageLineStep,
			localizedShopMessageWidth, localizedShopMessageLineStep,
			localizedShopMessageScale, 0xcd, 0x4c,
		); err != nil {
			return nil, err
		}
	}
	return result, nil
}

const (
	localizedShopItemNameWidth  = 67
	localizedShopItemNameHeight = 16
	localizedShopItemMaxScale   = 0.65
	localizedShopItemMinScale   = 0.40
)

func (g *Game) composeLocalizedNativeShopPurchaseList(
	stable []byte,
	assets *campaign.NativeShopAssets,
	goods []campaign.Good,
	itemIDs []int,
	start int,
) ([]byte, error) {
	if g == nil || g.localeEntities == nil || g.font == nil {
		return nil, errors.New("localized shop item catalog or font is unavailable")
	}
	frame, err := campaign.ComposeNativeShopItemListFrameWithoutNames(
		stable, assets, g.nativeShopUI.itemAssets, itemIDs, start, g.shopSel,
		g.nativeShopUI.effectRows, battle.NativeFacilityFullPrice,
	)
	if err != nil {
		return nil, err
	}
	visible := len(goods) - start
	if visible > 6 {
		visible = 6
	}
	for index := 0; index < visible; index++ {
		good := goods[start+index]
		name, err := g.localeEntities.ItemName(good.ID)
		if err != nil {
			return nil, err
		}
		column, line := index%2, index/2
		x := 10 + 148*column + 28
		y := 119 + 26*line + 3
		foreground := byte(0xcd)
		if start+index == g.shopSel {
			foreground = 0xc9
		}
		scale, ok := localizedShopItemScale(g.font, name)
		if !ok {
			return nil, fmt.Errorf("localized shop item %d name %q exceeds safe rectangle", good.ID, name)
		}
		if err := drawIndexedLocalizedText(
			frame, g.font, name, x, y,
			localizedShopItemNameWidth, localizedShopItemNameHeight,
			scale, foreground, 0x4c,
		); err != nil {
			return nil, err
		}
	}
	return frame, nil
}

func localizedShopItemScale(displayFont *Font, text string) (float64, bool) {
	if displayFont == nil || text == "" {
		return 0, false
	}
	for scale := localizedShopItemMaxScale; scale >= localizedShopItemMinScale-0.001; scale -= 0.05 {
		if displayFont.Width(text, scale) <= localizedShopItemNameWidth {
			return scale, true
		}
	}
	return 0, false
}
