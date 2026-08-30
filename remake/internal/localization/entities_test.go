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
		if len(catalog.items) != 120 {
			t.Fatalf("%s items=%d, want 120", locale, len(catalog.items))
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
