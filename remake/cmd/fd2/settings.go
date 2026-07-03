// settings.go — 全域系統設定(跨存檔,獨立於 fd2_save.json)。
// 目前:音源選擇(MT-32 / Sound Blaster FM)——還原原版 SETSOUND.EXE 選音效卡的體驗。
// FD2 出廠預設是 Sound Blaster(FM/OPL),多數玩家(含使用者童年)聽到的就是這個磅礡版;
// MT-32 是選配的「高級」音源,音色偏圓潤。兩套 OGG 並存,執行期可切換。
package main

import (
	"encoding/json"
	"os"
)

const settingsPath = "fd2_settings.json"

// bgmSources 可切換音源(對應 assets/music_<id>/ 資料夾)。
var bgmSources = []string{"fm", "mt32"}

var bgmSourceName = map[string]string{
	"fm":   "Sound Blaster (FM)",
	"mt32": "Roland MT-32",
}

type settings struct {
	BGMSource string `json:"bgm_source"` // "fm"(預設)或 "mt32"
}

// loadSettings 讀 fd2_settings.json;無檔/不合法回預設(fm=出廠 Sound Blaster)。
func loadSettings() settings {
	s := settings{BGMSource: "fm"}
	if raw, err := os.ReadFile(settingsPath); err == nil {
		json.Unmarshal(raw, &s)
	}
	if bgmSourceName[s.BGMSource] == "" {
		s.BGMSource = "fm"
	}
	return s
}

func saveSettings(s settings) {
	if raw, err := json.MarshalIndent(s, "", " "); err == nil {
		os.WriteFile(settingsPath, raw, 0o644)
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
	saveSettings(settings{BGMSource: g.bgmSource})
	g.msg = "音源:" + bgmSourceName[g.bgmSource]
	// 強制重播目前曲(繞過同曲不重播:清 bgmCur)
	if cur := g.bgmCur; cur != "" {
		g.bgmCur = ""
		g.playBGM(cur)
	}
}
