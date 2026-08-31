package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOfficialLocalesCoverSystemAndSaveMessages(t *testing.T) {
	wants := map[string][]string{
		"zh-Hant": {"語言：日本語", "已存檔（槽位 2：town_ch02）", "索爾 施放 火球：命中 2（造成 18）、未命中 1"},
		"zh-Hans": {"语言：日本語", "已存档（槽位 2：town_ch02）", "索尔 施放 火球：命中 2（造成 18）、未命中 1"},
		"ja":      {"言語：日本語", "スロット 2 にセーブしました：town_ch02", "ソルはファイアを唱えた：2体に命中（18ダメージ）、1体にミス"},
		"en":      {"Language: 日本語", "Saved to slot 2: town_ch02", "Sol casts Fire: 2 hit (18 damage), 1 missed"},
	}
	for localeID, want := range wants {
		catalog, err := loadOfficialLocale(localeID)
		if err != nil {
			t.Fatalf("%s: %v", localeID, err)
		}
		changed, err := catalog.Format("system.locale.changed", "日本語")
		if err != nil || changed != want[0] {
			t.Fatalf("%s locale message=%q err=%v", localeID, changed, err)
		}
		saved, err := catalog.Format("save.saved", 2, "town_ch02")
		if err != nil || saved != want[1] {
			t.Fatalf("%s save message=%q err=%v", localeID, saved, err)
		}
		names := []string{"索爾", "火球"}
		if localeID == "zh-Hans" {
			names = []string{"索尔", "火球"}
		} else if localeID == "ja" {
			names = []string{"ソル", "ファイア"}
		} else if localeID == "en" {
			names = []string{"Sol", "Fire"}
		}
		game := &Game{localeCatalog: catalog}
		result, ok := game.localizedSpellResult(names[0], names[1], 2, 18, false, 1)
		if !ok || result != want[2] {
			t.Fatalf("%s spell result=%q ok=%v", localeID, result, ok)
		}
		for _, key := range []string{
			"system.audio.changed", "save.unsupported", "save.postbattle_blocked", "save.none", "save.node_missing", "save.loaded",
			"battle.mp.insufficient", "battle.command.unavailable", "battle.attack.choose_target",
			"battle.command.native_unavailable", "battle.spell.sealed", "battle.spell.none", "battle.item.choose_slot",
			"battle.spell.target_prompt", "battle.spell.blocked", "battle.unit.paralyzed",
			"battle.spell.result_damage", "battle.spell.result_heal", "battle.spell.miss_suffix",
			"battle.treasure.gold", "battle.treasure.item", "battle.treasure.inventory_full",
			"battle.reward.item", "battle.reward.item_full", "battle.reward.gold",
			"battle.spell.menu_title",
			"church.transfer.empty", "church.transfer.success", "church.revive.success",
			"church.class_change.success",
			"church.class_change.confirm_title", "church.class_change.empty", "church.class_change.target",
			"shop.purchase.success", "shop.sell.success", "shop.purchase.equip_prompt", "shop.recipient.none",
			"shop.purchase.equip_prompt.title", "shop.purchase.equip_prompt.controls",
			"shop.sell.item_title", "shop.item.equipped_label", "shop.sell.roster_title",
			"shop.sell.inventory_count", "shop.purchase.recipient_title",
			"shop.purchase.recipient_inventory", "shop.panel.title_controls",
			"hotel.title.fallback", "hotel.controls", "preparation.title",
			"preparation.save_prompt", "preparation.deploy.controls",
			"preparation.unknown_character", "preparation.enter_confirm",
			"preparation.save_hint",
			"battle.result.win", "battle.result.lose", "battle.result.continue",
			"postbattle.preparation.title",
			"title.load.controls", "title.load.slot", "save.slot.empty", "save.slot.present",
			"church.service.status", "church.service.transfer", "church.service.revive",
			"church.service.class_change", "church.controls", "church.transfer.source",
			"church.transfer.unequipped_item", "church.transfer.destination",
			"church.empty_selection", "church.transfer.item_label",
			"church.selection_controls", "church.execute_controls", "ending.title",
			"common.yes", "common.no",
		} {
			entry, ok := catalog.Entries[key]
			if !ok || strings.TrimSpace(entry.Text) == "" {
				t.Fatalf("%s missing %s", localeID, key)
			}
		}
	}
}

func TestSettingsRoundTripKeepsAudioAndLocaleOutsideSave(t *testing.T) {
	oldXDG, hadXDG := os.LookupEnv("XDG_DATA_HOME")
	oldCache := userDataDirCached
	t.Cleanup(func() {
		userDataDirCached = oldCache
		if hadXDG {
			_ = os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			_ = os.Unsetenv("XDG_DATA_HOME")
		}
	})
	root := t.TempDir()
	if err := os.Setenv("XDG_DATA_HOME", root); err != nil {
		t.Fatal(err)
	}
	userDataDirCached = ""

	saveSettings(settings{BGMSource: "mt32", LocaleID: "ja"})
	got := loadSettings()
	if got.BGMSource != "mt32" || got.LocaleID != "ja" {
		t.Fatalf("settings=%+v", got)
	}
	raw, err := os.ReadFile(filepath.Join(root, "fd2_re", "fd2_settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	if _, ok := document["locale_id"]; !ok {
		t.Fatal("settings lost locale_id")
	}
	var save saveData
	saveRaw, err := json.Marshal(save)
	if err != nil {
		t.Fatal(err)
	}
	var saveDocument map[string]any
	if err := json.Unmarshal(saveRaw, &saveDocument); err != nil {
		t.Fatal(err)
	}
	if _, ok := saveDocument["locale_id"]; ok {
		t.Fatal("campaign save contains locale_id")
	}
}

func TestSettingsRejectUnsupportedLocale(t *testing.T) {
	oldXDG, hadXDG := os.LookupEnv("XDG_DATA_HOME")
	oldCache := userDataDirCached
	t.Cleanup(func() {
		userDataDirCached = oldCache
		if hadXDG {
			_ = os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			_ = os.Unsetenv("XDG_DATA_HOME")
		}
	})
	root := t.TempDir()
	if err := os.Setenv("XDG_DATA_HOME", root); err != nil {
		t.Fatal(err)
	}
	userDataDirCached = ""
	dir := filepath.Join(root, "fd2_re")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fd2_settings.json"), []byte(`{"bgm_source":"fm","locale_id":"xx"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := loadSettings(); got.LocaleID != "zh-Hant" {
		t.Fatalf("unsupported locale accepted: %+v", got)
	}
}
