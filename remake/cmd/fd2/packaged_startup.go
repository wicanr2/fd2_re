package main

import "os"

const defaultPlayerCampaign = "assets/scenarios/campaign_full.json"

// applyPackagedPlayerDefaults 只補真正未設定的玩家預設值。明確空值是測試與工具
// 用來停用 campaign 的既有契約，不可被正式預設覆蓋。
func applyPackagedPlayerDefaults() {
	if _, configured := os.LookupEnv("FD2_CAMPAIGN"); !configured {
		_ = os.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	}
}
