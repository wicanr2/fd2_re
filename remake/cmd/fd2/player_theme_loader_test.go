package main

import (
	"errors"
	"image"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// isolateThemeSettings 讓 cycleTheme 的 saveDisplaySettings 寫進暫存目錄，不留下 HD 設定影響後面的測試。
func isolateThemeSettings(t *testing.T) {
	t.Helper()
	old := userDataDirCached
	userDataDirCached = ""
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Cleanup(func() { userDataDirCached = old })
}

// waitModernThemeLoad 等背景載入結束（測試用；正式路徑由每次 Update 輪詢）。
func waitModernThemeLoad(t *testing.T, g *Game) {
	t.Helper()
	load := g.modernThemeLoading
	if load == nil {
		t.Fatal("no background theme load in progress")
	}
	select {
	case <-load.done:
	case <-time.After(10 * time.Second):
		t.Fatal("background theme load did not finish")
	}
	g.pollModernThemeLoad()
}

// 第一次 F2 不在 Update 執行緒同步載入：立即返回並顯示載入訊息，重按不重複載入，完成後
// 由輪詢補做這次切換（v.1.1.11 在這裡同步解碼整個 HD 包，畫面停住兩到四分鐘）。
func TestFirstThemeSwitchLoadsInBackgroundAndAppliesOnce(t *testing.T) {
	isolateThemeSettings(t)
	release := make(chan struct{})
	var calls atomic.Int32
	set := &modernStoryPortraitSet{hasHD: true}
	g := &Game{m: &MapData{TileW: 24, TileH: 24, Cols: 2},
		baseMapTileset: image.NewRGBA(image.Rect(0, 0, 48, 24)),
		modernThemeLoader: func() (*modernStoryPortraitSet, error) {
			calls.Add(1)
			<-release
			return set, nil
		}}
	g.cycleTheme()
	if g.modernStoryPortraits != nil || g.modernThemeLoading == nil || g.msg != modernThemeLoadMessage {
		t.Fatalf("first F2 did not defer to background load: msg=%q", g.msg)
	}
	g.cycleTheme() // 載入中再按一次
	g.pollModernThemeLoad()
	if g.modernStoryPortraits != nil || g.modernThemeLoading == nil {
		t.Fatal("theme switched before the background load finished")
	}
	close(release)
	waitModernThemeLoad(t, g)
	if calls.Load() != 1 {
		t.Fatalf("theme pack loaded %d times, want 1", calls.Load())
	}
	if g.modernThemeLoading != nil || g.modernThemeCache != set || g.modernStoryPortraits != set {
		t.Fatal("completed load was not applied as the pending F2")
	}
	g.cycleTheme()
	g.cycleTheme()
	if g.modernStoryPortraits != set || calls.Load() != 1 || g.modernThemeLoading != nil {
		t.Fatal("later F2 did not reuse the cached pack")
	}
}

func TestBackgroundThemeLoadFailureLeavesFaithfulMode(t *testing.T) {
	isolateThemeSettings(t)
	g := &Game{m: &MapData{TileW: 24, TileH: 24, Cols: 2},
		baseMapTileset: image.NewRGBA(image.Rect(0, 0, 48, 24)),
		modernThemeLoader: func() (*modernStoryPortraitSet, error) {
			return nil, errors.New("catalog missing")
		}}
	g.cycleTheme()
	waitModernThemeLoad(t, g)
	if g.modernStoryPortraits != nil || g.modernThemeCache != nil || !strings.Contains(g.msg, "catalog missing") {
		t.Fatalf("failed load changed theme or hid the error: msg=%q", g.msg)
	}
}

// F11 在沒有主題包時也走背景載入：setInterfaceMode 回 errModernThemeLoading，不改介面也不存設定。
func TestFirstInterfaceSwitchDefersWhileThemeLoads(t *testing.T) {
	release := make(chan struct{})
	g := &Game{interfaceMode: playerInterfaceOriginal,
		modernThemeLoader: func() (*modernStoryPortraitSet, error) {
			<-release
			return &modernStoryPortraitSet{}, nil
		}}
	if !g.requestModernThemeLoad(modernThemePendingInterface) || g.msg != modernThemeLoadMessage {
		t.Fatal("interface request did not start a background load")
	}
	if g.requestModernThemeLoad(modernThemePendingInterface) != true {
		t.Fatal("second request while loading should still defer")
	}
	close(release)
	<-g.modernThemeLoading.done
	g.modernThemeCache = g.modernThemeLoading.set
	g.modernThemeLoading = nil
	if g.requestModernThemeLoad(modernThemePendingInterface) {
		t.Fatal("cached pack still deferred the switch")
	}
}
