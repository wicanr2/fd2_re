package main

import (
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/editorcanonical"
)

// canonicalCampaignReference 讓 FD2_CAMPAIGN 指向編輯器的 canonical bundle 而不是
// 某個 JSON 檔。編輯器改的是 canonical 文件；玩家路徑要是繼續讀 legacy JSON，
// 編輯的結果就永遠進不了遊戲，而兩邊看起來都正常。
const canonicalCampaignReference = "canonical"

// legacyPlayerCampaign 是 canonical 文件的來源，現在只用來守住兩者不漂移。
const legacyPlayerCampaign = "assets/scenarios/campaign_full.json"

const defaultPlayerCampaign = canonicalCampaignReference

// applyPackagedPlayerDefaults 只補真正未設定的玩家預設值。明確空值是測試與工具
// 用來停用 campaign 的既有契約，不可被正式預設覆蓋。
func applyPackagedPlayerDefaults() {
	if _, configured := os.LookupEnv("FD2_CAMPAIGN"); !configured {
		_ = os.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	}
}

// loadPlayerCampaign 依 FD2_CAMPAIGN 的值取得戰役圖。
//
// `canonical` 走編輯器 bundle：編譯回 legacy 形狀之後交給 campaign.Decode，與直接
// 讀 JSON 走的是同一道驗證。其餘值仍是明確路徑，測試與離屏 oracle 靠它指定自己
// 的戰役檔，那個契約不動。
func loadPlayerCampaign(reference string) (*campaign.Campaign, error) {
	if reference != canonicalCampaignReference {
		return campaign.Load(assetPath(reference))
	}
	compiled, err := editorcanonical.CompileCampaignJSON(assetPath("assets/editor-canonical"))
	if err != nil {
		return nil, err
	}
	return campaign.Decode(compiled)
}
