package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// controlsPanelShot 是一張設定面板擷圖：先跑幾次正式 Update 讓畫面穩定，再開面板，
// 把 Draw 與 DrawFinalScreen 畫進指定大小的影像存成 PNG。
type controlsPanelShot struct {
	name          string
	p             *modernHDPresentation
	width, height int
}

type controlsPanelShotProbe struct {
	shots    []controlsPanelShot
	out      string
	index    int
	ticks    int
	saved    []string
	err      error
	captured bool
}

func (q *controlsPanelShotProbe) Layout(int, int) (int, int) { return logicalW, logicalH }

func (q *controlsPanelShotProbe) Update() error {
	if q.err != nil || q.index >= len(q.shots) {
		return ebiten.Termination
	}
	shot := q.shots[q.index]
	if q.captured {
		q.index, q.ticks, q.captured = q.index+1, 0, false
		return nil
	}
	q.ticks++
	if q.ticks <= 4 {
		q.err = shot.p.Update()
		return nil
	}
	if !shot.p.Game.controls.open {
		shot.p.Game.openControlsPanel()
		shot.p.Game.controls.selected = controlActionIndex("confirm")
		shot.p.Game.controls.ensureVisible()
	}
	return nil
}

func (q *controlsPanelShotProbe) Draw(screen *ebiten.Image) {
	if q.index < len(q.shots) {
		q.shots[q.index].p.Draw(screen)
	}
}

func (q *controlsPanelShotProbe) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	screen.DrawImage(offscreen, &ebiten.DrawImageOptions{GeoM: geometry})
	if q.err != nil || q.index >= len(q.shots) || q.captured || !q.shots[q.index].p.Game.controls.open {
		return
	}
	shot := q.shots[q.index]
	target := ebiten.NewImage(shot.width, shot.height)
	scaled := ebiten.GeoM{}
	scaled.Scale(float64(shot.width)/float64(logicalW), float64(shot.height)/float64(logicalH))
	shot.p.DrawFinalScreen(target, offscreen, scaled)
	pixels := make([]byte, 4*shot.width*shot.height)
	target.ReadPixels(pixels)
	rgba := &image.RGBA{Pix: pixels, Stride: 4 * shot.width, Rect: image.Rect(0, 0, shot.width, shot.height)}
	path := filepath.Join(q.out, "controls-panel-"+shot.name+".png")
	file, err := os.Create(path)
	if err == nil {
		err = png.Encode(file, rgba)
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}
	if err != nil {
		q.err = fmt.Errorf("%s: %w", shot.name, err)
		return
	}
	q.saved = append(q.saved, path)
	q.captured = true
}

// TestControlsPanelScreenshotGPU 在 Xvfb 下存兩張設定面板（原版美術＋原版介面、HD 美術＋HD 介面，
// 手把未連接）。需要正式 HD 包：FD2_TEST_FACILITY_CATALOG／FD2_TEST_FACILITY_PACK，輸出目錄
// FD2_CONTROLS_PANEL_SHOT。只證明面板的實際繪製，不是玩家路徑驗收。
func TestControlsPanelScreenshotGPU(t *testing.T) {
	out := os.Getenv("FD2_CONTROLS_PANEL_SHOT")
	if out == "" {
		t.Skip("dedicated screenshot process")
	}
	previous := userDataDirCached
	userDataDirCached = t.TempDir()
	t.Cleanup(func() { userDataDirCached = previous })
	set := facilityHDTestSet(t)
	build := func(hd bool) *modernHDPresentation {
		g := facilityHDTestGame(t, set)
		variant := 0
		g.camp = campaign.NewRunner(&campaign.Campaign{Start: "town", Nodes: map[string]*campaign.Node{"town": {Type: "town", NativeTownVariant: &variant}}})
		g.nativeShopMode, g.nativeShopUIJob = "", nil
		g.playerSpeed = 1
		g.applyFontScale(1)
		if hd {
			g.interfaceMode = playerInterfaceHD
		} else {
			g.modernStoryPortraits = nil
			g.interfaceMode = playerInterfaceOriginal
		}
		g.input = newPlayerInput(defaultControlBindings(), controlsLoadDefault)
		return &modernHDPresentation{Game: g, toolbar: true, floatingToolbar: true, faithfulWidth: 1280, faithfulHeight: 800}
	}
	q := &controlsPanelShotProbe{out: out, shots: []controlsPanelShot{
		{name: "original-art", p: build(false), width: 1280, height: 800},
		{name: "hd-interface", p: build(true), width: 1920, height: 1200},
	}}
	ebiten.SetWindowSize(1280, 800)
	if err := ebiten.RunGame(q); err != nil {
		t.Fatal(err)
	}
	if q.err != nil || len(q.saved) != len(q.shots) {
		t.Fatal("screenshots", q.err, q.saved)
	}
	for _, shot := range q.shots {
		if shot.p.Game.input.padCount != 0 {
			t.Fatal("screenshot expects no gamepad")
		}
	}
	t.Log("saved", q.saved)
}
