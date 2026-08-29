package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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
