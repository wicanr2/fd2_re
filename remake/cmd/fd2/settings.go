// settings.go — 全域系統設定(跨存檔,獨立於 fd2_save.json)。
// 目前提供重製端 F2 設定與兩套預錄 OGG，在操作意圖上近似原版
// SETSOUND.EXE 的音源選擇；它不執行原版程式、驅動或即時合成。
// 兩套 OGG 以 Sound Blaster（FM）與 Roland MT-32 身分並存；重製端預設選 FM。
// 這個預設與音色描述不是原版 SETSOUND 的動態證據，也不代表即時合成 parity。
package main

import (
	"encoding/json"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// settingsPath 設定檔位置:$XDG_DATA_HOME/fd2_re/fd2_settings.json(理由同 savePath,見 assets.go)。
func settingsPath() string { return userDataPath("fd2_settings.json") }

// bgmSources 可切換音源(對應 assets/music_<id>/ 資料夾)。
var bgmSources = []string{"fm", "mt32"}

var bgmSourceName = map[string]string{
	"fm":   "Sound Blaster (FM)",
	"mt32": "Roland MT-32",
}

var localeIDs = []string{"zh-Hant", "zh-Hans", "ja", "en"}

var localeDisplayName = map[string]string{
	"zh-Hant": "繁體中文",
	"zh-Hans": "简体中文",
	"ja":      "日本語",
	"en":      "English",
}

type settings struct {
	BGMSource string `json:"bgm_source"` // "fm"(預設)或 "mt32"
	LocaleID  string `json:"locale_id"`  // BCP 47；不屬於戰役存檔
}

// loadSettings 讀 fd2_settings.json；無檔或不合法時回重製端預設 FM。
func loadSettings() settings {
	s := settings{BGMSource: "fm", LocaleID: "zh-Hant"}
	if raw, err := os.ReadFile(settingsPath()); err == nil {
		json.Unmarshal(raw, &s)
	}
	if bgmSourceName[s.BGMSource] == "" {
		s.BGMSource = "fm"
	}
	if localeDisplayName[s.LocaleID] == "" {
		s.LocaleID = "zh-Hant"
	}
	return s
}

func saveSettings(s settings) {
	if raw, err := json.MarshalIndent(s, "", " "); err == nil {
		os.WriteFile(settingsPath(), raw, 0o644)
	}
}

// cycleBGMSource 切到下一個音源、持久化、並強制重播目前曲(以新音源)。
func (g *Game) cycleBGMSource() {
	i := 0
	for k, s := range bgmSources {
		if s == g.bgmSource {
			i = k
			break
		}
	}
	g.bgmSource = bgmSources[(i+1)%len(bgmSources)]
	saveSettings(settings{BGMSource: g.bgmSource, LocaleID: g.localeID})
	if message, ok := g.localeMessage("system.audio.changed", bgmSourceName[g.bgmSource]); ok {
		g.msg = message
	}
	// 強制重播目前曲(繞過同曲不重播:清 bgmCur)
	if cur := g.bgmCur; cur != "" {
		g.bgmCur = ""
		g.playBGM(cur)
	}
}

// cycleLocale 先完整驗證下一個官方包，再原子切換並保存；載入失敗時維持舊語系。
func (g *Game) cycleLocale() {
	if g == nil {
		return
	}
	i := 0
	for k, id := range localeIDs {
		if id == g.localeID {
			i = k
			break
		}
	}
	next := localeIDs[(i+1)%len(localeIDs)]
	catalog, err := loadOfficialLocale(next)
	if err != nil {
		g.loadErr = "locale setting: " + err.Error()
		return
	}
	content, err := loadOfficialLocaleContent(next)
	if err != nil {
		g.loadErr = "locale content: " + err.Error()
		return
	}
	entities, err := loadOfficialLocaleEntities(next)
	if err != nil {
		g.loadErr = "locale entities: " + err.Error()
		return
	}
	var spells []battle.Spell
	if len(g.spells) > 0 {
		spells, err = localizedSpellBook(entities, g.spells)
		if err != nil {
			g.loadErr = "locale spell book: " + err.Error()
			return
		}
	}
	g.localeID, g.localeCatalog, g.localeContent, g.localeEntities = next, catalog, content, entities
	if spells != nil {
		g.spells = spells
	}
	if g.st != nil && spells != nil {
		g.st.SpellBook = append([]battle.Spell(nil), spells...)
	}
	saveSettings(settings{BGMSource: g.bgmSource, LocaleID: g.localeID})
	if message, ok := g.localeMessage("system.locale.changed", localeDisplayName[next]); ok {
		g.msg = message
	}
}
