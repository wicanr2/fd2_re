package localization

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOfficialEntityCatalogsShareItemIdentity(t *testing.T) {
	root := filepath.Join("..", "..", "assets", "locales")
	var reference map[int]struct{}
	for _, locale := range []string{"zh-Hant", "zh-Hans", "ja", "en"} {
		catalog, err := LoadOfficialEntities(root, locale)
		if err != nil {
			t.Fatalf("%s: %v", locale, err)
		}
		if len(catalog.items) != 200 {
			t.Fatalf("%s items=%d, want 200", locale, len(catalog.items))
		}
		if len(catalog.characters) != 32 {
			t.Fatalf("%s characters=%d, want 32", locale, len(catalog.characters))
		}
		if name, err := catalog.CharacterName(9); err != nil || (locale == "zh-Hant" && name != "悠妮") {
			t.Fatalf("%s character 9=%q, err=%v", locale, name, err)
		}
		if reference == nil {
			reference = make(map[int]struct{}, len(catalog.items))
			for id := range catalog.items {
				reference[id] = struct{}{}
			}
			continue
		}
		for id := range reference {
			if _, ok := catalog.items[id]; !ok {
				t.Fatalf("%s misses item %d", locale, id)
			}
		}
	}
}

func TestEntityCatalogCharacterLookupFailsClosed(t *testing.T) {
	catalog, err := LoadOfficialEntities(filepath.Join("..", "..", "assets", "locales"), "en")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.CharacterName(32); err == nil {
		t.Fatal("missing character name was accepted")
	}
}

func TestEntityCatalogRejectsConfirmedEmptyItemIDs(t *testing.T) {
	catalog, err := LoadOfficialEntities(filepath.Join("..", "..", "assets", "locales"), "zh-Hant")
	if err != nil {
		t.Fatal(err)
	}
	for id := 108; id <= 122; id++ {
		if _, err := catalog.ItemName(id); err == nil {
			t.Fatalf("confirmed-empty item %d was accepted", id)
		}
	}
}

func TestEntityCatalogRejectsFutureFields(t *testing.T) {
	root := filepath.Join("..", "..", "assets", "locales")
	raw, err := os.ReadFile(filepath.Join(root, "en", "entities.json"))
	if err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(string(raw), `"schema_version": 1,`, `"schema_version": 1, "future": true,`, 1)
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "en"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "en", "entities.json"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOfficialEntities(directory, "en"); err == nil {
		t.Fatal("future entity field was accepted")
	}
}
