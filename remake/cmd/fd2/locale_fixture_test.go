package main

import "testing"

// attachOfficialLocale 讓手工組出來的測試 Game 具備正式路徑一定會有的語系資料。
//
// `loadGame()` 一定會載入 catalog／content／entities 並設定 localeID；手工
// `&Game{}` 不會，於是 `localeID` 是空字串。空字串不等於繁中，本地化分支會
// 要求 `localeEntities` 與字型，缺了就失敗即關閉——測試因此在「語系缺件」上
// 停住，而不是在它要驗的那條功能路徑上。
func attachOfficialLocale(t *testing.T, g *Game) {
	t.Helper()
	if g == nil {
		t.Fatal("attachOfficialLocale 需要非 nil 的 Game")
	}
	const localeID = "zh-Hant"
	catalog, err := loadOfficialLocale(localeID)
	if err != nil {
		t.Fatalf("載入官方語系 %s：%v", localeID, err)
	}
	content, err := loadOfficialLocaleContent(localeID)
	if err != nil {
		t.Fatalf("載入官方全量內容 %s：%v", localeID, err)
	}
	entities, err := loadOfficialLocaleEntities(localeID)
	if err != nil {
		t.Fatalf("載入官方實體名稱 %s：%v", localeID, err)
	}
	g.localeID, g.localeCatalog, g.localeContent, g.localeEntities = localeID, catalog, content, entities
}
