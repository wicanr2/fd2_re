package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var playerSpeedMenu = image.Rect(300, 26, 634, 166)
var playerGeneralSpeedChoices = [...]image.Rectangle{
	image.Rect(376, 54, 434, 74), image.Rect(438, 54, 496, 74), image.Rect(500, 54, 558, 74), image.Rect(562, 54, 622, 74),
}
var playerBattleSpeedChoices = [...]image.Rectangle{
	image.Rect(376, 86, 438, 106), image.Rect(442, 86, 484, 106), image.Rect(488, 86, 530, 106), image.Rect(534, 86, 576, 106), image.Rect(580, 86, 622, 106),
}
var playerSpeedClose = image.Rect(312, 140, 622, 160)

func (p *modernHDPresentation) queueSpeedPoint(point image.Point) bool {
	if !p.Game.playerPosterControlsAllowed() {
		return point.In(playerSpeedButton)
	}
	if point.In(playerSpeedButton) {
		p.speedMenuOpen = !p.speedMenuOpen
		p.poster.menuOpen = false
		p.poster.consumeInput = true
		return true
	}
	if !p.speedMenuOpen {
		return false
	}
	if !point.In(playerSpeedMenu) {
		p.speedMenuOpen = false
		if point.In(image.Rect(0, 0, logicalW, playerToolbarHeight)) {
			return false
		}
		p.poster.consumeInput = true
		return true
	}
	p.poster.consumeInput = true
	for i, rect := range playerGeneralSpeedChoices {
		if point.In(rect) {
			p.Game.chooseGeneralSpeed(i + 1)
			return true
		}
	}
	for i, rect := range playerBattleSpeedChoices {
		if point.In(rect) {
			p.Game.setBattleAnimationSpeed(i)
			return true
		}
	}
	if point.In(playerSpeedClose) {
		p.speedMenuOpen = false
	}
	return true
}

func (p *modernHDPresentation) stepSpeedMenuInput() {
	if p.speedMenuOpen && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.speedMenuOpen = false
		p.poster.consumeInput = true
	}
}

func (p *modernHDPresentation) drawSpeedMenu(screen ebiten.FinalScreen, geometry ebiten.GeoM) {
	if !p.speedMenuOpen || !p.toolbarVisible() {
		return
	}
	if p.speedMenuSurface == nil {
		p.speedMenuSurface = ebiten.NewImage(logicalW*3, playerSpeedMenu.Max.Y*3)
	}
	surface := p.speedMenuSurface
	surface.Clear()
	r := playerSpeedMenu
	vector.DrawFilledRect(surface, float32(r.Min.X*3), float32(r.Min.Y*3), float32(r.Dx()*3), float32(r.Dy()*3), color.RGBA{194, 158, 89, 255}, false)
	vector.DrawFilledRect(surface, float32(r.Min.X*3+3), float32(r.Min.Y*3+3), float32(r.Dx()*3-6), float32(r.Dy()*3-6), color.RGBA{17, 30, 48, 255}, false)
	title, game, battle, follow, note, close := "速度 · F8", "遊戲", "戰鬥", "跟隨", "音樂與音效維持原速", "收合"
	switch p.Game.localeID {
	case "en":
		title, game, battle, follow, note, close = "Speed · F8", "Game", "Battle", "Follow", "Music and sound keep their original rate", "Close"
	case "ja":
		title, game, battle, follow, note, close = "速度 · F8", "全体", "戦闘", "連動", "音楽と効果音の速度は変わりません", "閉じる"
	case "zh-Hans":
		title, game, battle, follow, note, close = "速度 · F8", "游戏", "战斗", "跟随", "音乐与音效保持原速", "收起"
	}
	p.drawCompactToolbarButton(surface, image.Rect(312, 29, 622, 49), title, false)
	p.drawCompactToolbarButton(surface, image.Rect(312, 54, 370, 74), game, false)
	p.drawCompactToolbarButton(surface, image.Rect(312, 86, 370, 106), battle, false)
	for i, rect := range playerGeneralSpeedChoices {
		p.drawCompactToolbarButton(surface, rect, fmt.Sprintf("%d×", i+1), normalizePlayerSpeed(p.Game.playerSpeed) == i+1)
	}
	for i, rect := range playerBattleSpeedChoices {
		label := follow
		if i != 0 {
			label = fmt.Sprintf("%d×", i)
		}
		p.drawCompactToolbarButton(surface, rect, label, normalizeBattleAnimationSpeed(p.Game.playerBattleSpeed) == i)
	}
	p.drawCompactToolbarButton(surface, image.Rect(312, 113, 622, 135), note, false)
	p.drawCompactToolbarButton(surface, playerSpeedClose, close, false)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(1.0/3, 1.0/3)
	op.GeoM.Concat(geometry)
	screen.DrawImage(surface, op)
}
