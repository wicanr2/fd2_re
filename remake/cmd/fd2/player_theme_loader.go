package main

import "errors"

// 第一次按 F2（美術主題）或 F11（介面）切到 HD 時要載入完整現代主題包：正式包的 modern/ 約
// 1.1 萬檔、3.6 GB，同步解碼要兩到四分鐘。v.1.1.11 在 Update 裡同步載入，整段時間畫面不更新、
// 輸入沒有反應，Windows 會把視窗標成「沒有回應」，玩家回報為卡死（2026-11-14）。
//
// 這裡把載入移到背景 goroutine。loadModernStoryPortraitSet 只讀檔、雜湊與解碼成 image 資料，
// 不碰 Game，也不建立 GPU 圖片（GPU 圖片在繪製端按需建立），所以可以離開 Update 執行緒。
// 載入期間 F2／F11 只顯示進度訊息、不切換；完成後由 Update 收回結果寫進 modernThemeCache，
// 再補做玩家按下的那一次切換，之後的切換沿用快取，和原本相同。

// errModernThemeLoading 表示現代主題包正在背景載入，這次切換延後到載入完成。
var errModernThemeLoading = errors.New("現代主題載入中")

const (
	modernThemePendingArt       = "art"
	modernThemePendingInterface = "interface"
)

type modernThemeLoad struct {
	done    chan struct{}
	set     *modernStoryPortraitSet
	err     error
	pending string // 載入完成後要補做的切換
}

// modernThemeLoadMessage 是載入期間顯示的訊息。
const modernThemeLoadMessage = "HD 主題素材載入中，完成後自動切換……"

// requestModernThemeLoad 在沒有快取時啟動背景載入並記下待補做的切換，回傳 true 表示呼叫端
// 這次不要切換。已經在載入時不重複啟動，只更新待補做的切換。
func (g *Game) requestModernThemeLoad(pending string) bool {
	if g.modernThemeCache != nil {
		return false
	}
	if g.modernThemeLoading == nil {
		load := &modernThemeLoad{done: make(chan struct{})}
		g.modernThemeLoading = load
		loader := g.modernThemeLoader
		if loader == nil {
			loader = loadPlayerModernTheme
		}
		go func() {
			load.set, load.err = loader()
			close(load.done)
		}()
	}
	g.modernThemeLoading.pending = pending
	g.msg = modernThemeLoadMessage
	return true
}

// pollModernThemeLoad 在每次 Update 開頭收回已完成的背景載入，成功就寫進快取並補做切換。
func (g *Game) pollModernThemeLoad() {
	load := g.modernThemeLoading
	if load == nil {
		return
	}
	select {
	case <-load.done:
	default:
		return
	}
	g.modernThemeLoading = nil
	if load.err != nil {
		g.msg = "現代主題不可用：" + load.err.Error()
		return
	}
	g.modernThemeCache = load.set
	switch load.pending {
	case modernThemePendingArt:
		if g.modernStoryPortraits == nil {
			g.cycleTheme()
		}
	case modernThemePendingInterface:
		if !g.modernHDInterfaceEnabled() {
			g.cycleInterfaceMode()
		}
	}
}
