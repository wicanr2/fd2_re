package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
)

// 第一次按 F2（美術主題）或 F11（介面）切到 HD 時要載入完整現代主題包：正式包的 modern/ 約
// 1.1 萬檔、3.6 GB，同步解碼要兩到四分鐘。v.1.1.11 在 Update 裡同步載入，整段時間畫面不更新、
// 輸入沒有反應，Windows 會把視窗標成「沒有回應」，玩家回報為卡死（2026-11-14）。
//
// 這裡把載入移到背景 goroutine。loadModernStoryPortraitSet 只讀檔、雜湊與解碼成 image 資料，
// 不碰 Game，也不建立 GPU 圖片（GPU 圖片在繪製端按需建立），所以可以離開 Update 執行緒。
// 載入期間 F2／F11 只顯示進度訊息、不切換；完成後由 Update 收回結果寫進 modernThemeCache，
// 再補做玩家按下的切換，之後的切換沿用快取，和原本相同。
//
// 設定已存成 HD 的玩家啟動時也走同一條路（#318）：loadGame 先以原版素材進入，背景載入完成
// 後補做美術與介面兩個切換。載入期間存設定時保留原本的 HD 設定，載入失敗則留在原版並顯示原因。

// errModernThemeLoading 表示現代主題包正在背景載入，這次切換延後到載入完成。
var errModernThemeLoading = errors.New("現代主題載入中")

const (
	modernThemePendingArt       = "art"
	modernThemePendingInterface = "interface"
	// modernThemePendingInterfaceAuto 是設定沒有指定介面時的啟動行為：主題包有 HD 素材才切到 HD 介面。
	modernThemePendingInterfaceAuto = "interface-auto"
)

type modernThemeLoad struct {
	done chan struct{}
	set  *modernStoryPortraitSet
	err  error
	// 載入完成後要補做的切換；可以同時有美術與介面。
	pendingArt, pendingInterface, pendingInterfaceAuto bool
	// 啟動時延後載入的設定值；載入期間 saveDisplaySettings 寫回這些值，不把 HD 設定蓋成原版。
	startup                         bool
	startupThemeID, startupIfaceSet string
	progressDone, progressTotal     atomic.Int64
	artMessage                      string        // 上一幀補做美術切換時的訊息，與介面切換的訊息合併顯示
	warming                         chan struct{} // 發布前在背景解碼目前地圖的地形；關閉表示完成
}

// modernThemeLoadMessage 是載入期間顯示的訊息；有進度時改為 modernThemeLoadProgressMessage。
const modernThemeLoadMessage = "HD 主題素材載入中，完成後自動切換……"

func modernThemeLoadProgressMessage(done, total int64) string {
	return fmt.Sprintf("HD 主題素材載入中（%d%%），完成後自動切換……", done*100/total)
}

// progress 可由多個載入流同時呼叫；顯示的已完成數只增不減。
func (load *modernThemeLoad) progress(done, total int) {
	load.progressTotal.Store(int64(total))
	for {
		old := load.progressDone.Load()
		if int64(done) <= old || load.progressDone.CompareAndSwap(old, int64(done)) {
			return
		}
	}
}

func (load *modernThemeLoad) addPending(pending string) {
	switch pending {
	case modernThemePendingArt:
		load.pendingArt = true
	case modernThemePendingInterface:
		load.pendingInterface = true
	case modernThemePendingInterfaceAuto:
		load.pendingInterfaceAuto = true
	}
}

// requestModernThemeLoad 在沒有快取時啟動背景載入並記下待補做的切換，回傳 true 表示呼叫端
// 這次不要切換。已經在載入時不重複啟動，只加上待補做的切換。
func (g *Game) requestModernThemeLoad(pending string) bool {
	if g.modernThemeCache != nil {
		return false
	}
	if g.modernThemeLoading == nil {
		load := &modernThemeLoad{done: make(chan struct{})}
		g.modernThemeLoading = load
		userDataDir() // assetPath 會讀這個快取；先在主執行緒填好，背景只讀
		loader := g.modernThemeLoader
		if loader == nil {
			loader = func() (*modernStoryPortraitSet, error) { return loadPlayerModernThemeWithProgress(load.progress) }
		}
		currentMap := -1
		if g.nativeMapAssets != nil {
			currentMap = g.nativeMapAssets.MapIndex
		}
		go func() {
			load.set, load.err = loader()
			if load.err == nil && currentMap >= 0 {
				// 地形圖按需解碼；先在背景解好目前這張地圖，補做切換的那一幀就不用等（#317 後續）。
				// 地圖若在載入期間換了，切換時照常按需解碼。錯誤留給切換時的正式路徑回報。
				_, _ = load.set.terrainHDMap(currentMap)
			}
			close(load.done)
		}()
	}
	g.modernThemeLoading.addPending(pending)
	g.msg = modernThemeLoadMessage
	return true
}

// deferStartupModernTheme 讓 loadGame 不同步載入設定指定的 HD 主題：遊戲先以原版素材進入，
// 背景載入完成後由 pollModernThemeLoad 補做切換。themeID／ifaceSetting 是設定檔原本的值。
func (g *Game) deferStartupModernTheme(art bool, iface string, themeID, ifaceSetting string) {
	if !art && iface == "" {
		return
	}
	if art {
		g.requestModernThemeLoad(modernThemePendingArt)
	}
	if iface != "" {
		g.requestModernThemeLoad(iface)
	}
	if load := g.modernThemeLoading; load != nil {
		load.startup, load.startupThemeID, load.startupIfaceSet = true, themeID, ifaceSetting
	}
}

// modernThemeStartupDeferred 判斷 loadGame 能不能把設定指定的 HD 主題改成背景載入。截圖、
// 結局預覽與以環境變數指定主題／介面的工具路徑維持同步載入，輸出不受影響。
func modernThemeStartupDeferred() bool {
	for _, name := range []string{"FD2_THEME", "FD2_INTERFACE_MODE", "FD2_SHOT", "FD2_SHOT_SERIES", "FD2_ENDING_PREFIX"} {
		if os.Getenv(name) != "" {
			return false
		}
	}
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
		if total := load.progressTotal.Load(); total > 0 && strings.HasPrefix(g.msg, "HD 主題素材載入中") {
			g.msg = modernThemeLoadProgressMessage(load.progressDone.Load(), total)
		}
		return
	}
	if load.err == nil && g.modernThemeCache == nil && g.warmModernThemeTerrain(load) {
		return // 目前這張地圖的地形還在背景解碼；解完才切換，切換那一幀不必等（#317 後續）
	}
	g.modernThemeLoading = nil
	if load.err != nil {
		g.msg = "現代主題不可用：" + load.err.Error()
		return
	}
	g.modernThemeCache = load.set
	wantInterface := load.pendingInterface || (load.pendingInterfaceAuto && load.set != nil && load.set.hasHD)
	if load.pendingArt && g.modernStoryPortraits == nil {
		g.cycleTheme()
		if wantInterface {
			// 兩個切換分在相鄰兩次 Update，避免同一幀做兩次完整的主題預備。
			next := &modernThemeLoad{done: load.done, set: load.set, pendingInterface: true, artMessage: g.msg}
			g.modernThemeLoading = next
			return
		}
	}
	if wantInterface && !g.modernHDInterfaceEnabled() {
		g.cycleInterfaceMode()
		if strings.HasPrefix(g.msg, "介面：") && strings.HasPrefix(load.artMessage, "主題：") {
			g.msg = load.artMessage + "　" + g.msg
		}
	}
}

// warmModernThemeTerrain 在主題集合發布前，於背景解碼目前這張地圖的地形（地形按需解碼）；回傳 true
// 表示還在解碼，這次 Update 先不切換。集合尚未發布，背景 goroutine 是唯一的使用者。
func (g *Game) warmModernThemeTerrain(load *modernThemeLoad) bool {
	if load.warming == nil {
		if load.set == nil || g.nativeMapAssets == nil || load.set.terrainHDMapReady(g.nativeMapAssets.MapIndex) {
			return false
		}
		warming, set, mapID := make(chan struct{}), load.set, g.nativeMapAssets.MapIndex
		load.warming = warming
		go func() {
			_, _ = set.terrainHDMap(mapID) // 錯誤留給切換時的正式路徑回報
			close(warming)
		}()
	}
	select {
	case <-load.warming:
		return false
	default:
		return true
	}
}

// startupThemeSettings 回傳啟動延後載入期間要寫回設定檔的主題與介面值；ok 為 false 表示沒有。
func (g *Game) startupThemeSettings() (themeID, iface string, ok bool) {
	load := g.modernThemeLoading
	if load == nil || !load.startup {
		return "", "", false
	}
	return load.startupThemeID, load.startupIfaceSet, true
}
