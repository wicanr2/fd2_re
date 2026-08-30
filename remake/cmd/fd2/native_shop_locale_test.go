package main

import "testing"

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

func TestOfficialShopItemNameLookupFailsClosed(t *testing.T) {
	catalog, err := loadOfficialLocaleEntities("en")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.ItemName(255); err == nil {
		t.Fatal("missing item name was accepted")
	}
}
