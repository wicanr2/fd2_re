package main

import (
	"bytes"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestOfficialShopItemNamesFitNativeSafeRectangle(t *testing.T) {
	displayFont := loadFont()
	if displayFont == nil {
		t.Fatal("official CJK font unavailable")
	}
	for _, localeID := range []string{"zh-Hans", "ja", "en"} {
		catalog, err := loadOfficialLocaleEntities(localeID)
		if err != nil {
			t.Fatal(err)
		}
		for itemID := 0; itemID <= 255; itemID++ {
			name, err := catalog.ItemName(itemID)
			if err != nil {
				continue
			}
			scale, ok := localizedShopItemScale(displayFont, name)
			if !ok {
				t.Errorf("%s item %d name %q exceeds safe rectangle", localeID, itemID, name)
				continue
			}
			if err := drawIndexedLocalizedText(
				make([]byte, 320*200), displayFont, name, 38, 122,
				localizedShopItemNameWidth, localizedShopItemNameHeight,
				scale, 0xcd, 0x4c,
			); err != nil {
				t.Errorf("%s item %d name %q: %v", localeID, itemID, name, err)
			}
		}
	}
}

func TestOfficialShopMessagesFitNativeDialogueRectangle(t *testing.T) {
	displayFont := loadFont()
	graph, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, localeID := range []string{"zh-Hans", "ja", "en"} {
		catalog, err := loadOfficialLocale(localeID)
		if err != nil {
			t.Fatal(err)
		}
		entities, err := loadOfficialLocaleEntities(localeID)
		if err != nil {
			t.Fatal(err)
		}
		g := &Game{localeID: localeID, localeCatalog: catalog, localeEntities: entities, font: displayFont}
		for _, variant := range []int{1, 3} {
			g.nativeShopVariant = variant
			for _, pair := range [][2]string{
				{"shop.greeting.weapon", "shop.greeting.item"},
				{"shop.purchase.insufficient.weapon", "shop.purchase.insufficient.item"},
				{"shop.purchase.no_recipient.weapon", "shop.purchase.no_recipient.item"},
			} {
				message, ok := g.localeMessage(g.localizedShopKey(pair[0], pair[1]))
				if !ok {
					t.Fatal(g.loadErr)
				}
				before := make([]byte, 320*200)
				after, err := g.drawLocalizedShopMessage(before, message, 119, 3)
				if err != nil || bytes.Equal(before, after) {
					t.Fatalf("%s variant %d message %q: %v", localeID, variant, message, err)
				}
			}
		}
		for _, key := range []string{
			"shop.purchase.equip_question",
			"shop.transfer.destination_prompt",
			"shop.transfer.source_prompt",
		} {
			message, ok := g.localeMessage(key)
			if !ok {
				t.Fatal(g.loadErr)
			}
			if _, err := g.drawLocalizedShopMessage(make([]byte, 320*200), message, 119, 3); err != nil {
				t.Errorf("%s %s %q: %v", localeID, key, message, err)
			}
		}
		for _, node := range graph.Nodes {
			if node.Type != "shop" {
				continue
			}
			g.nativeShopVariant = node.NativeHubVariant
			for _, good := range append(append([]campaign.Good(nil), node.Goods...), node.Secret...) {
				name, err := entities.ItemName(good.ID)
				if err != nil {
					t.Fatal(err)
				}
				key := g.localizedShopKey("shop.purchase.question.weapon", "shop.purchase.question.item")
				message, ok := g.localeMessage(key, name, good.Price)
				if !ok {
					t.Fatal(g.loadErr)
				}
				if _, err := g.drawLocalizedShopMessage(make([]byte, 320*200), message, 119, 3); err != nil {
					t.Errorf("%s item %d question %q: %v", localeID, good.ID, message, err)
				}
			}
		}
	}
}

func TestOfficialShopItemNameLookupFailsClosed(t *testing.T) {
	catalog, err := loadOfficialLocaleEntities("en")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.ItemName(255); err == nil {
		t.Fatal("missing item name was accepted")
	}
}
