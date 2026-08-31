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
		"zh-Hant": {"語言：日本語", "已存檔（槽位 2：town_ch02）"},
		"zh-Hans": {"语言：日本語", "已存档（槽位 2：town_ch02）"},
		"ja":      {"言語：日本語", "スロット 2 にセーブしました：town_ch02"},
		"en":      {"Language: 日本語", "Saved to slot 2: town_ch02"},
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
		for _, key := range []string{
			"system.audio.changed", "save.unsupported", "save.postbattle_blocked", "save.none", "save.node_missing", "save.loaded",
			"battle.mp.insufficient", "battle.command.unavailable", "battle.attack.choose_target",
			"battle.command.native_unavailable", "battle.spell.sealed", "battle.spell.none", "battle.item.choose_slot",
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
