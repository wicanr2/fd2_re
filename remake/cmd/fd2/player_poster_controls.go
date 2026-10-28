package main

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var playerPosterMenu = image.Rect(362, 26, 634, 158)
var playerPosterChoices = [3]image.Rectangle{image.Rect(374, 49, 454, 68), image.Rect(458, 49, 538, 68), image.Rect(542, 49, 622, 68)}
var playerPosterSlider = image.Rect(380, 84, 620, 104)
var playerPosterFadeChoices = [4]image.Rectangle{image.Rect(374, 109, 433, 129), image.Rect(437, 109, 496, 129), image.Rect(500, 109, 559, 129), image.Rect(563, 109, 622, 129)}
var playerPosterReplay = image.Rect(374, 134, 496, 154)
var playerPosterClose = image.Rect(500, 134, 622, 154)
var playerPosterDurations = [4]int{0, 300, 600, 1200}

func (g *Game) playerPosterControlsAllowed() bool {
	return !g.startupBlocked && g.loadErr == "" && g.nativeDefeat == nil && g.nativeEnding == nil &&
		!g.helpVisible && !g.transientUI && !nativeModifierHeld() &&
		(g.nativeSystemEndTurnUI == nil || g.nativeSystemEndTurnUI.rewardMessage == nil)
}

type playerPosterLabels struct {
	button, title, edge, replay, close, unavailable string
	sets                                            [3]string
}

func (p *modernHDPresentation) posterLabels() playerPosterLabels {
	switch p.Game.localeID {
	case "en":
		return playerPosterLabels{"Posters", "Posters · F10", "Blend", "Replay", "Close", "Posters unavailable: ", [3]string{"Off", "HD 1", "Classic 2"}}
	case "ja":
		return playerPosterLabels{"ポスター", "ポスター · F10", "境界", "再生", "閉じる", "ポスターを読めません：", [3]string{"なし", "HD 1", "原画 2"}}
	case "zh-Hans":
		return playerPosterLabels{"海报", "海报 · F10", "渐层", "重播", "收起", "海报不可用：", [3]string{"关闭", "HD 第一组", "原稿第二组"}}
	default:
		return playerPosterLabels{"海報", "海報 · F10", "漸層", "重播", "收合", "海報不可用：", [3]string{"關閉", "HD 第一組", "原稿第二組"}}
	}
}

func playerPosterEdgeAt(x int) int {
	value := math.Round(float64(x-playerPosterSlider.Min.X) * 96 / float64(playerPosterSlider.Dx()-1))
	return int(math.Max(0, math.Min(96, value)))
}

func (p *modernHDPresentation) queuePosterPoint(point image.Point) bool {
	if !p.Game.playerPosterControlsAllowed() {
		return point.In(playerPosterButton)
	}
	if point.In(playerPosterButton) {
		p.poster.menuOpen = !p.poster.menuOpen
		p.poster.consumeInput = true
		return true
	}
	if !p.poster.menuOpen {
		return false
	}
	if !point.In(playerPosterMenu) {
		if point.In(image.Rect(0, 0, logicalW, playerToolbarHeight)) {
			return false
		}
		p.poster.menuOpen = false
		p.poster.consumeInput = true
		return true
	}
	p.poster.consumeInput = true
	for i, rect := range playerPosterChoices {
		if point.In(rect) {
			if err := p.applyPosterMode([]string{"off", "hd", "classic"}[i], true); err != nil {
				p.Game.msg = p.posterLabels().unavailable + err.Error()
			}
			return true
		}
	}
	if point.In(playerPosterSlider) {
		p.poster.settings.Edge = playerPosterEdgeAt(point.X)
		p.poster.sliderMouse = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
		p.savePosterSettings()
	}
	for i, rect := range playerPosterFadeChoices {
		if point.In(rect) {
			p.poster.settings.FadeMS = playerPosterDurations[i]
			p.poster.started = p.posterNow()
			p.savePosterSettings()
		}
	}
	if point.In(playerPosterReplay) {
		p.poster.started = p.posterNow()
	}
	if point.In(playerPosterClose) {
		p.poster.menuOpen = false
	}
	return true
}

func (p *modernHDPresentation) stepPosterControls() {
	if !p.poster.menuOpen {
		p.poster.sliderMouse, p.poster.sliderTouch = false, false
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.poster.menuOpen = false
		p.poster.consumeInput = true
		return
	}
	if p.poster.sliderMouse {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			x, _ := ebiten.CursorPosition()
			p.poster.settings.Edge = playerPosterEdgeAt(x - p.posterOffset())
			p.poster.consumeInput = true
		} else {
			p.poster.sliderMouse = false
			p.savePosterSettings()
		}
	}
	if p.poster.sliderTouch {
		active := false
		for _, id := range ebiten.AppendTouchIDs(nil) {
			active = active || id == p.poster.touchID
		}
		if active {
			x, _ := ebiten.TouchPosition(p.poster.touchID)
			p.poster.settings.Edge = playerPosterEdgeAt(x - p.posterOffset())
			p.poster.consumeInput = true
		} else {
			p.poster.sliderTouch = false
			p.savePosterSettings()
		}
	}
}

func (p *modernHDPresentation) drawPosterMenu(screen ebiten.FinalScreen, geometry ebiten.GeoM) {
	if !p.poster.menuOpen || !p.toolbarVisible() {
		return
	}
	if p.posterMenuSurface == nil {
		p.posterMenuSurface = ebiten.NewImage(logicalW*3, playerPosterMenu.Max.Y*3)
	}
	surface := p.posterMenuSurface
	surface.Clear()
	r := playerPosterMenu
	vector.DrawFilledRect(surface, float32(r.Min.X*3), float32(r.Min.Y*3), float32(r.Dx()*3), float32(r.Dy()*3), color.RGBA{88, 104, 126, 255}, false)
	vector.DrawFilledRect(surface, float32(r.Min.X*3+3), float32(r.Min.Y*3+3), float32(r.Dx()*3-6), float32(r.Dy()*3-6), color.RGBA{17, 24, 37, 255}, false)
	l := p.posterLabels()
	p.drawCompactToolbarButton(surface, image.Rect(374, 28, 622, 46), l.title, false)
	for i, rect := range playerPosterChoices {
		p.drawCompactToolbarButton(surface, rect, l.sets[i], p.poster.settings.Set == []string{"off", "hd", "classic"}[i])
	}
	p.drawCompactToolbarButton(surface, image.Rect(374, 69, 622, 84), fmt.Sprintf("%s %d", l.edge, p.poster.settings.Edge), false)
	vector.DrawFilledRect(surface, float32(playerPosterSlider.Min.X*3), 93*3, float32(playerPosterSlider.Dx()*3), 3, color.RGBA{110, 124, 142, 255}, false)
	x := playerPosterSlider.Min.X + p.poster.settings.Edge*(playerPosterSlider.Dx()-1)/96
	vector.DrawFilledRect(surface, float32(x*3-4), 87*3, 9, 15*3, color.RGBA{220, 185, 112, 255}, false)
	for i, rect := range playerPosterFadeChoices {
		p.drawCompactToolbarButton(surface, rect, fmt.Sprintf("%.1fs", float64(playerPosterDurations[i])/1000), p.poster.settings.FadeMS == playerPosterDurations[i])
	}
	p.drawCompactToolbarButton(surface, playerPosterReplay, l.replay, false)
	p.drawCompactToolbarButton(surface, playerPosterClose, l.close, false)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(1.0/3, 1.0/3)
	op.GeoM.Concat(geometry)
	screen.DrawImage(surface, op)
}
