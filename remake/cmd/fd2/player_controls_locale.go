package main

// player_controls_locale.go — 按鍵設定面板的語系檔 assets/locales/<語系>/controls.json。
//
// 面板是播放器外框，不是原版畫面，所以字串不放進 pack.json：pack.json 的鍵集合是 Go 與
// Rust 兩個引擎共用的精確契約，多一個鍵另一邊就拒收整份語系。這裡另立一份同樣精確的契約：
// 鍵集合與變數必須完全相同，缺鍵、多鍵或變數不符就整份拒收，不混用其他語系。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

const controlsLocaleKind = "fd2_controls_locale"

// controlsLocaleContract 是 controls.json 的鍵與變數；動作名稱依 controlActions 產生。
func controlsLocaleContract() map[string][]string {
	contract := map[string][]string{
		"controls.title": {}, "controls.subtitle": {},
		"controls.pad.disconnected": {}, "controls.pad.connected": {"%d"},
		"controls.column.action": {}, "controls.column.keyboard": {}, "controls.column.gamepad": {},
		"controls.group.movement": {}, "controls.group.confirm": {}, "controls.group.system": {},
		"controls.unbound": {}, "controls.dpad": {}, "controls.stick": {}, "controls.deadzone": {},
		"controls.msg.ready": {}, "controls.msg.capture": {}, "controls.msg.conflict": {"%s", "%s"},
		"controls.msg.assigned": {}, "controls.msg.cleared": {}, "controls.msg.reset_row": {},
		"controls.msg.reset_all": {}, "controls.msg.reserved": {}, "controls.msg.corrupt": {},
		"controls.msg.partial": {}, "controls.msg.save_failed": {"%s"},
		"controls.help.keyboard": {}, "controls.help.gamepad": {}, "controls.reset_all": {},
	}
	for _, action := range controlActions {
		contract["controls.action."+action.ID] = []string{}
	}
	return contract
}

var controlsGroupKeys = [controlGroupCount]string{"controls.group.movement", "controls.group.confirm", "controls.group.system"}

type controlsLocaleEntry struct {
	Text      string   `json:"text"`
	Variables []string `json:"variables"`
}

type controlsLocaleFile struct {
	SchemaVersion int                            `json:"schema_version"`
	Kind          string                         `json:"kind"`
	Locale        string                         `json:"locale"`
	Entries       map[string]controlsLocaleEntry `json:"entries"`
}

type controlsLocale struct {
	locale  string
	entries map[string]controlsLocaleEntry
}

func loadControlsLocale(localeID string) (*controlsLocale, error) {
	return loadControlsLocaleFrom(filepath.Join(assetPath("assets/locales"), localeID, "controls.json"), localeID)
}

func loadControlsLocaleFrom(path, localeID string) (*controlsLocale, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file controlsLocaleFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("controls locale %s: %w", localeID, err)
	}
	if file.SchemaVersion != 1 || file.Kind != controlsLocaleKind || file.Locale != localeID {
		return nil, fmt.Errorf("controls locale %s: header mismatch", localeID)
	}
	contract := controlsLocaleContract()
	if len(file.Entries) != len(contract) {
		return nil, fmt.Errorf("controls locale %s has %d keys, want exact %d", localeID, len(file.Entries), len(contract))
	}
	for key, variables := range contract {
		entry, ok := file.Entries[key]
		if !ok {
			return nil, fmt.Errorf("controls locale %s is missing %s", localeID, key)
		}
		if entry.Text == "" || !reflect.DeepEqual(append([]string{}, entry.Variables...), variables) {
			return nil, fmt.Errorf("controls locale %s has invalid text or variables for %s", localeID, key)
		}
	}
	return &controlsLocale{locale: localeID, entries: file.Entries}, nil
}

// text 依契約格式化；語系檔不可用時回傳鍵名，讓缺件看得見，不改用其他語系。
func (c *controlsLocale) text(key string, args ...any) string {
	if c == nil {
		return key
	}
	entry, ok := c.entries[key]
	if !ok || len(entry.Variables) != len(args) {
		return key
	}
	if len(args) == 0 {
		return entry.Text
	}
	return fmt.Sprintf(entry.Text, args...)
}
