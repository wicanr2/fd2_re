package main

import (
	"errors"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// 以 ebiten.RunGame 跑正式的 modernHDPresentation 量第一次 F2／F11 與 HD 設定的啟動（#317、#318）。
// 離屏迴圈（hd_theme_load_measure_test.go）沒有真正的幀結束，Ebiten 的待處理繪圖與每幀新建的
// 圖片會一直留在 heap，載入越久 heap 越大；這裡讓每一幀照正式遊戲的方式結束。
// 每項用獨立程序跑：go test -run '^TestHDThemeLoadMeasureGUIF2$'。

type hdLoadGUIProbe struct {
	*modernHDPresentation
	pressAt    int // 第幾次 Update 按鍵；-1 表示啟動量測（不按鍵）
	press      func()
	applied    func() bool
	origin     time.Time // 量測起點：按鍵當下，或啟動量測的 loadGame 之前
	started    bool
	ticks      int
	switched   time.Duration
	afterTicks int
	firstDraw  time.Duration
	updateDur  time.Duration
	maxFrame   time.Duration // 一個 tick 的 Update＋Draw 最長時間（起點之後）
	maxGap     time.Duration // 相鄰兩次 Update 開始的最長間隔（起點之後）
	slow       [3]int        // Update＋Draw 超過 50／100／250 毫秒的 tick 數（起點之後）
	lastUpdate time.Time
	frames     int
	sampler    *hdLoadHeapSampler
	err        error
}

var _ ebiten.FinalScreenDrawer = (*hdLoadGUIProbe)(nil)

func (q *hdLoadGUIProbe) Update() error {
	now := time.Now()
	if q.started && !q.lastUpdate.IsZero() {
		if gap := now.Sub(q.lastUpdate); gap > q.maxGap {
			q.maxGap = gap
		}
	}
	q.lastUpdate = now
	if q.ticks == q.pressAt {
		q.sampler = startHDLoadHeapSampler()
		q.press()
		q.origin, q.started = now, true
	}
	q.ticks++
	err := q.modernHDPresentation.Update()
	q.updateDur = time.Since(now)
	if err != nil {
		q.err = err
		return ebiten.Termination
	}
	if q.started {
		q.frames++
		if q.switched == 0 && q.applied() {
			q.switched = time.Since(q.origin)
		}
		if q.switched != 0 {
			q.afterTicks++
			if q.afterTicks > 120 {
				return ebiten.Termination
			}
		}
		if time.Since(q.origin) > 600*time.Second {
			q.err = errors.New("切換超過 600 秒")
			return ebiten.Termination
		}
		if q.pressAt >= 0 && q.switched == 0 && q.Game.modernThemeLoading == nil && !q.applied() {
			q.err = errors.New("載入結束但沒有切換：" + q.Game.msg)
			return ebiten.Termination
		}
	}
	return nil
}

func (q *hdLoadGUIProbe) Draw(screen *ebiten.Image) {
	start := time.Now()
	q.modernHDPresentation.Draw(screen)
	if q.started {
		frame := q.updateDur + time.Since(start)
		if frame > q.maxFrame {
			q.maxFrame = frame
		}
		for i, limit := range []time.Duration{50 * time.Millisecond, 100 * time.Millisecond, 250 * time.Millisecond} {
			if frame > limit {
				q.slow[i]++
			}
		}
		if q.firstDraw == 0 {
			q.firstDraw = time.Since(q.origin)
		}
	}
}

func hdLoadGUIPresentation(g *Game) *modernHDPresentation {
	player := &modernHDPresentation{Game: g, toolbar: true, floatingToolbar: true, faithfulWidth: logicalW * 2, faithfulHeight: logicalH * 2}
	player.initPosters()
	if g.hdPresentationEnabled() {
		ebiten.SetWindowSize(modernHDWidth, player.presentationHeight(modernHDWidth))
	} else {
		ebiten.SetWindowSize(logicalW*2, player.presentationHeight(logicalW*2))
	}
	ebiten.SetTPS(60 * normalizePlayerSpeed(g.playerSpeed))
	return player
}

func hdLoadGUIToggle(t *testing.T, art bool) {
	out := hdLoadMeasureSetup(t)
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if g.modernStoryPortraits != nil || g.interfaceMode == playerInterfaceHD {
		t.Fatal("起點不是忠實／原版")
	}
	base := hdLoadLiveHeap()
	q := &hdLoadGUIProbe{modernHDPresentation: hdLoadGUIPresentation(g), pressAt: 200}
	if art {
		q.press = func() { g.playerThemeRequested = true }
		q.applied = func() bool { return g.modernStoryPortraits != nil }
	} else {
		q.press = func() { g.playerInterfaceRequested = true }
		q.applied = func() bool { return g.interfaceMode == playerInterfaceHD }
	}
	if err := ebiten.RunGame(q); err != nil {
		t.Fatal(err)
	}
	if q.err != nil {
		t.Fatal(q.err)
	}
	peak, peakAt := q.sampler.finish()
	live := hdLoadLiveHeap()
	name := "gui-f11"
	if art {
		name = "gui-f2"
	}
	hdLoadWriteHeapProfile(t, out, name)
	hdLoadWriteReceipt(t, out, name, map[string]any{
		"switch_seconds":         q.switched.Seconds(),
		"ticks_after_press":      q.frames,
		"max_frame_seconds":      q.maxFrame.Seconds(),
		"max_update_gap_seconds": q.maxGap.Seconds(),
		"ticks_over_50_100_250ms": q.slow,
		"heap_objects_base":      base,
		"heap_objects_peak":      peak,
		"heap_objects_peak_at_s": peakAt.Seconds(),
		"heap_objects_live":      live,
		"msg":                    g.msg,
	})
}

func TestHDThemeLoadMeasureGUIF2(t *testing.T)  { hdLoadGUIToggle(t, true) }
func TestHDThemeLoadMeasureGUIF11(t *testing.T) { hdLoadGUIToggle(t, false) }

// TestHDThemeLoadMeasureGUIStartup 量設定已存成 HD 時的啟動：loadGame 返回、第一幀畫出（可操作）
// 與 HD 實際套用的秒數，都從呼叫 loadGame 前起算。
func TestHDThemeLoadMeasureGUIStartup(t *testing.T) {
	out := hdLoadMeasureSetup(t)
	s := loadSettings()
	s.ThemeID, s.InterfaceMode = modernHandpaintedThemeID, playerInterfaceHD
	saveSettings(s)
	sampler := startHDLoadHeapSampler()
	origin := time.Now()
	g := loadGame()
	loaded := time.Since(origin)
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	q := &hdLoadGUIProbe{modernHDPresentation: hdLoadGUIPresentation(g), pressAt: -1, origin: origin, started: true, sampler: sampler,
		applied: func() bool { return g.modernStoryPortraits != nil && g.interfaceMode == playerInterfaceHD }}
	if err := ebiten.RunGame(q); err != nil {
		t.Fatal(err)
	}
	if q.err != nil {
		t.Fatal(q.err)
	}
	peak, peakAt := sampler.finish()
	live := hdLoadLiveHeap()
	saved := loadSettings()
	hdLoadWriteHeapProfile(t, out, "gui-startup")
	hdLoadWriteReceipt(t, out, "gui-startup", map[string]any{
		"load_game_seconds":      loaded.Seconds(),
		"operable_seconds":       q.firstDraw.Seconds(),
		"hd_applied_seconds":     q.switched.Seconds(),
		"ticks_until_applied":    q.frames - q.afterTicks,
		"max_frame_seconds":      q.maxFrame.Seconds(),
		"max_update_gap_seconds": q.maxGap.Seconds(),
		"ticks_over_50_100_250ms": q.slow,
		"heap_objects_peak":      peak,
		"heap_objects_peak_at_s": peakAt.Seconds(),
		"heap_objects_live":      live,
		"saved_theme":            saved.ThemeID,
		"saved_interface":        saved.InterfaceMode,
		"msg":                    g.msg,
	})
}
