package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
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

// 設定存成 HD 時 loadGame 不同步載入主題包（#318）：先以原版進入、背景載入，載入期間存設定
// 不把 HD 蓋成原版；載入失敗時留在原版、顯示原因，不讓遊戲停在錯誤畫面。
func TestStartupHDSettingsLoadInBackgroundAndFailSafe(t *testing.T) {
	isolateThemeSettings(t)
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	t.Setenv("FD2_MODERN_THEME_PACK", t.TempDir()) // 空包：背景載入必定失敗
	s := loadSettings()
	s.ThemeID, s.InterfaceMode = modernHandpaintedThemeID, playerInterfaceHD
	saveSettings(s)
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	load := g.modernThemeLoading
	if load == nil || !load.pendingArt || !load.pendingInterface || !load.startup {
		t.Fatal("HD settings did not start a background load with both pending switches")
	}
	if g.modernStoryPortraits != nil || g.interfaceMode != playerInterfaceOriginal || g.modernThemeCache != nil {
		t.Fatalf("startup published HD before the pack loaded: iface=%s", g.interfaceMode)
	}
	g.saveDisplaySettings() // 例如載入期間按了 F7
	if saved := loadSettings(); saved.ThemeID != modernHandpaintedThemeID || saved.InterfaceMode != playerInterfaceHD {
		t.Fatalf("saving during the startup load overwrote HD settings: %+v", saved)
	}
	waitModernThemeLoad(t, g)
	if g.modernStoryPortraits != nil || g.interfaceMode != playerInterfaceOriginal || g.loadErr != "" ||
		!strings.HasPrefix(g.msg, "現代主題不可用：") {
		t.Fatalf("failed startup load did not stay in original mode with a message: msg=%q err=%q", g.msg, g.loadErr)
	}
	if saved := loadSettings(); saved.ThemeID != modernHandpaintedThemeID || saved.InterfaceMode != playerInterfaceHD {
		t.Fatalf("failed startup load erased HD settings: %+v", saved)
	}
}

// 以環境變數指定主題或介面、截圖與結局預覽等工具路徑仍同步載入，輸出不受背景載入影響。
func TestStartupThemeToolPathsStaySynchronous(t *testing.T) {
	names := []string{"FD2_THEME", "FD2_INTERFACE_MODE", "FD2_SHOT", "FD2_SHOT_SERIES", "FD2_ENDING_PREFIX"}
	for _, name := range names {
		for _, other := range names {
			t.Setenv(other, "")
		}
		if !modernThemeStartupDeferred() {
			t.Fatal("plain startup is not deferred")
		}
		t.Setenv(name, "x")
		if modernThemeStartupDeferred() {
			t.Fatalf("%s did not keep the synchronous load", name)
		}
	}
}

// 啟動時兩個切換都等載入完成才做：美術套用並寫入設定，介面依結果切換或留原版；
// 自動介面只在主題包確實有 HD 素材時才嘗試切換。
func TestStartupPendingSwitchesApplyAfterLoad(t *testing.T) {
	for _, hasHD := range []bool{false, true} {
		isolateThemeSettings(t)
		release := make(chan struct{})
		set := &modernStoryPortraitSet{hasHD: hasHD}
		g := &Game{m: &MapData{TileW: 24, TileH: 24, Cols: 2}, interfaceMode: playerInterfaceOriginal,
			baseMapTileset: image.NewRGBA(image.Rect(0, 0, 48, 24)),
			modernThemeLoader: func() (*modernStoryPortraitSet, error) {
				<-release
				return set, nil
			}}
		g.deferStartupModernTheme(true, modernThemePendingInterfaceAuto, modernHandpaintedThemeID, "")
		load := g.modernThemeLoading
		if load == nil || !load.pendingArt || !load.pendingInterfaceAuto || g.modernStoryPortraits != nil {
			t.Fatal("startup did not defer both switches")
		}
		g.pollModernThemeLoad()
		if g.modernStoryPortraits != nil {
			t.Fatal("switched before load finished")
		}
		close(release)
		waitModernThemeLoad(t, g)
		if g.modernStoryPortraits != set || g.modernThemeCache != set {
			t.Fatal("art switch not applied after startup load")
		}
		if saved := loadSettings(); saved.ThemeID != modernHandpaintedThemeID {
			t.Fatalf("applied art not saved: %+v", saved)
		}
		if hasHD {
			// 介面切換排在下一次 Update，不和美術擠在同一幀。
			if g.modernThemeLoading == nil || strings.Contains(g.msg, "介面") {
				t.Fatalf("interface switch was not deferred to the next update: msg=%q", g.msg)
			}
			g.pollModernThemeLoad()
		}
		if g.modernThemeLoading != nil {
			t.Fatal("startup load still pending after both switches")
		}
		// 測試用 Game 沒有 HD 字型，HD 介面預檢必定失敗：只要有試、失敗時留原版即可。
		tried := strings.Contains(g.msg, "介面切換失敗")
		if hasHD != tried || g.interfaceMode != playerInterfaceOriginal {
			t.Fatalf("auto interface hasHD=%v tried=%v iface=%s msg=%q", hasHD, tried, g.interfaceMode, g.msg)
		}
	}
}

// 背景載入期間訊息顯示進度，但不覆寫其他訊息。
func TestModernThemeLoadShowsProgress(t *testing.T) {
	load := &modernThemeLoad{done: make(chan struct{})}
	g := &Game{modernThemeLoading: load, msg: modernThemeLoadMessage}
	g.pollModernThemeLoad()
	if g.msg != modernThemeLoadMessage {
		t.Fatalf("progress shown before total known: %q", g.msg)
	}
	load.progress(5, 20)
	g.pollModernThemeLoad()
	if !strings.Contains(g.msg, "25%") {
		t.Fatalf("progress not shown: %q", g.msg)
	}
	g.msg = "攻擊！"
	load.progress(10, 20)
	g.pollModernThemeLoad()
	if g.msg != "攻擊！" {
		t.Fatalf("progress overwrote another message: %q", g.msg)
	}
}

// 地形按需解碼：主題包載入完成後、發布前，先在背景解好目前這張地圖的地形，再補做切換；載入期間
// 換了地圖也一樣（#317 後續）。
func TestModernThemeLoadWarmsCurrentMapTerrainBeforePublishing(t *testing.T) {
	isolateThemeSettings(t)
	root := t.TempDir()
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, terrainTestImage(color.RGBA{30, 60, 90, 255})); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tile.png"), encoded.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(encoded.Bytes()))
	set := &modernStoryPortraitSet{}
	for _, mapID := range []int{5, 6} {
		asset := modernThemeAsset{Role: "native_terrain_hd_set", Status: "runtime_candidate", ConsumerContract: "fdshap_native_hd_sparse_v1",
			MapID: mapID, Width: 144, Height: 144, AlphaContract: "native_mask_nearest_6", CyclePolicy: "native_selector", FrameCount: 1,
			FrameIDs: []int{0}, Files: []string{"tile.png"}, FrameSHA256: []string{hash}, Semantics: []string{"terrain"}}
		if err := loadModernTerrainHDAsset(set, asset, root); err != nil {
			t.Fatal(err)
		}
	}
	release := make(chan struct{})
	g := &Game{nativeMapAssets: &nativeMapAssets{MapIndex: 5},
		modernThemeLoader: func() (*modernStoryPortraitSet, error) { <-release; return set, nil }}
	g.requestModernThemeLoad(modernThemePendingInterfaceAuto)
	g.nativeMapAssets = &nativeMapAssets{MapIndex: 6} // 載入期間換了地圖
	load := g.modernThemeLoading
	close(release)
	<-load.done
	if !set.terrainHDMapReady(5) || set.terrainHDMapReady(6) {
		t.Fatal("request-time map was not decoded in the loader goroutine")
	}
	g.pollModernThemeLoad()
	if g.modernThemeCache != nil || load.warming == nil {
		t.Fatal("published before the current map terrain was decoded")
	}
	select {
	case <-load.warming:
	case <-time.After(10 * time.Second):
		t.Fatal("terrain warm-up did not finish")
	}
	g.pollModernThemeLoad()
	if g.modernThemeCache != set || g.modernThemeLoading != nil || !set.terrainHDMapReady(6) || set.terrainHD[6][0].Image == nil {
		t.Fatal("warmed theme was not published")
	}
}
