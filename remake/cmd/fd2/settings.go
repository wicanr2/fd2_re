// settings.go — 全域系統設定(跨存檔,獨立於 fd2_save.json)。
// 目前提供重製端 F3 設定與兩套預錄 OGG，在操作意圖上近似原版
// SETSOUND.EXE 的音源選擇；它不執行原版程式、驅動或即時合成。
// 兩套 OGG 以 Sound Blaster（FM）與 Roland MT-32 身分並存；重製端預設選 FM。
// 這個預設與音色描述不是原版 SETSOUND 的動態證據，也不代表即時合成 parity。
package main

import (
	"encoding/json"
	"fmt"
	"math"
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

// fontScales 是可選的字級倍率。英日文譯文比中文長，同一個框裡常常斷不完；
// 調小一格能多塞一行。原版字模畫的索引畫面不受影響。
var fontScales = []float64{0.8, 0.9, 1.0, 1.1, 1.25}

const defaultFontScale = 1.0

type settings struct {
	BGMSource string  `json:"bgm_source"` // "fm"(預設)或 "mt32"
	LocaleID  string  `json:"locale_id"`  // BCP 47；不屬於戰役存檔
	FontScale float64 `json:"font_scale"` // 字級倍率；0 或不合法值視同 1.0
}

// normalizeFontScale 把設定值收斂到 fontScales 裡最接近的一格。舊的設定檔沒有
// 這個欄位，讀出來是 0，這裡會變成預設值。
func normalizeFontScale(v float64) float64 {
	if v <= 0 {
		return defaultFontScale
	}
	best, bestDiff := defaultFontScale, math.Inf(1)
	for _, candidate := range fontScales {
		if diff := math.Abs(candidate - v); diff < bestDiff {
			best, bestDiff = candidate, diff
		}
	}
	return best
}

// loadSettings 讀 fd2_settings.json；無檔或不合法時回重製端預設 FM。
func loadSettings() settings {
	s := settings{BGMSource: "fm", LocaleID: "zh-Hant", FontScale: defaultFontScale}
	if raw, err := os.ReadFile(settingsPath()); err == nil {
		json.Unmarshal(raw, &s)
	}
	s.FontScale = normalizeFontScale(s.FontScale)
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

// applyFontScale 把字級倍率套到所有以 TTF 繪製的字型。索引畫面用原版字模，不走
// 這裡，所以調字級不會動到與原版對拍的畫面。
func (g *Game) applyFontScale(scale float64) {
	if g == nil {
		return
	}
	g.fontScale = normalizeFontScale(scale)
	g.font.SetUserScale(g.fontScale)
	g.fontNm.SetUserScale(g.fontScale)
}

// cycleFontScale 依序切換字級並存進設定。英文與日文的譯文比中文長，同一個對話框
// 常常斷不完；調小一格可以多塞一行。
func (g *Game) cycleFontScale() {
	if g == nil {
		return
	}
	current := normalizeFontScale(g.fontScale)
	next := fontScales[0]
	for i, candidate := range fontScales {
		if candidate == current {
			next = fontScales[(i+1)%len(fontScales)]
			break
		}
	}
	g.applyFontScale(next)
	configured := loadSettings()
	configured.FontScale = next
	saveSettings(configured)
	g.msg = fmt.Sprintf("字級：%.0f%%", next*100)
}
