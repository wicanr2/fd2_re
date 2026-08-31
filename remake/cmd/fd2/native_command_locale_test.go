package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestLocalizedSpellBookUsesCommandCatalogAndKeepsEmptySlot(t *testing.T) {
	entities, err := loadOfficialLocaleEntities("en")
	if err != nil {
		t.Fatal(err)
	}
	source := make([]battle.Spell, 36)
	for id := range source {
		source[id] = battle.Spell{ID: id, Name: "舊名稱"}
	}
	got, err := localizedSpellBook(entities, source)
	if err != nil {
		t.Fatal(err)
	}
	if got[21].Name != "Cure Paralysis" || got[27].Name != "Paralysis" || got[31].Name != "" {
		t.Fatalf("localized spell names=%q/%q/empty=%q", got[21].Name, got[27].Name, got[31].Name)
	}
	if source[21].Name != "舊名稱" {
		t.Fatal("localized spell book mutated its source")
	}
}

func TestLocalizedCommandCorrectionsPreserveSourceAndDisplayLayers(t *testing.T) {
	entities, err := loadOfficialLocaleEntities("zh-Hant")
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[int]string{21: "祛麻術", 27: "麻痺術"} {
		got, err := entities.CommandName(id)
		if err != nil || got != want {
			t.Fatalf("command %d=%q, err=%v, want %q", id, got, err, want)
		}
	}
}
