package main

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

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
