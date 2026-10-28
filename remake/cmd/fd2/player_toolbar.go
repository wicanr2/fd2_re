package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
)

const playerToolbarHeight = 24
const playerToolbarHideTicks = 90

var playerLocaleButton = image.Rect(240, 3, 336, 21)
var playerInterfaceButton = image.Rect(134, 3, 236, 21)
var playerFontButton = image.Rect(340, 3, 410, 21)
var playerSpeedButton = image.Rect(414, 3, 466, 21)
var playerPosterButton = image.Rect(470, 3, 540, 21)
var playerFullscreenButton = image.Rect(544, 3, 632, 21)

func (g *Game) playerTouchCoordinates(x, y int) (int, int, bool) {
	if y < g.playerViewportTop || g.playerToolbarTouchBlocked && y < playerToolbarHeight {
		return 0, 0, false
	}
	return x, y - g.playerViewportTop, true
}

func (p *modernHDPresentation) toolbarOffset() int {
	if p.toolbar && !p.floatingToolbar {
		return playerToolbarHeight
	}
	return 0
}

func (p *modernHDPresentation) toolbarVisible() bool {
	return p.toolbar && (!p.floatingToolbar || p.toolbarTicks > 0)
}

// Visibility uses presentation updates only; it never advances or pauses game time.
func (p *modernHDPresentation) stepToolbarVisibility(pointer image.Point, touches []image.Point) {
	if !p.toolbar || !p.floatingToolbar {
		return
	}
	zone := image.Rect(0, 0, logicalW, 4)
	if p.toolbarTicks > 0 {
		zone.Max.Y = playerToolbarHeight
	}
	active := pointer.In(zone)
	for _, touch := range touches {
		active = active || touch.In(zone)
	}
	if active {
		p.toolbarTicks = playerToolbarHideTicks * normalizePlayerSpeed(p.Game.playerSpeed)
	} else if p.toolbarTicks > 0 {
		p.toolbarTicks--
	}
}

func (p *modernHDPresentation) updateToolbarVisibility() {
	x, y := ebiten.CursorPosition()
	var touches []image.Point
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		tx, ty := ebiten.TouchPosition(id)
		touches = append(touches, image.Pt(tx, ty))
	}
	p.stepToolbarVisibility(image.Pt(x, y), touches)
}

var playerThemeButtons = [...]image.Rectangle{
	image.Rect(8, 3, 78, 21),
	image.Rect(82, 3, 130, 21),
}

func playerThemeAt(x, y int) (hd, ok bool) {
	for index, rect := range playerThemeButtons {
		if image.Pt(x, y).In(rect) {
			return index == 1, true
		}
	}
	return false, false
}

func (p *modernHDPresentation) requestToolbarTheme(hd bool) {
	if hd != (p.Game.modernStoryPortraits != nil) {
		p.Game.playerThemeRequested = true
	}
}

func (p *modernHDPresentation) queueToolbarInput() {
	if !p.toolbarVisible() {
		return
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		p.queueToolbarPoint(image.Pt(x, y))
		return
	}
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		p.queueToolbarPoint(image.Pt(x, y))
		return
	}
}

func (p *modernHDPresentation) queueToolbarPoint(point image.Point) {
	if hd, ok := playerThemeAt(point.X, point.Y); ok {
		p.requestToolbarTheme(hd)
		return
	}
	switch {
	case point.In(playerLocaleButton):
		p.Game.playerLocaleRequested = true
	case point.In(playerInterfaceButton):
		p.Game.playerInterfaceRequested = true
	case point.In(playerFontButton):
		p.Game.playerFontRequested = true
	case point.In(playerSpeedButton):
		p.Game.playerSpeedRequested = true
	case point.In(playerFullscreenButton):
		p.Game.playerFullscreenRequested = true
	}
}

func (p *modernHDPresentation) Layout(int, int) (int, int) {
	return logicalW, logicalH + p.toolbarOffset()
}

func (p *modernHDPresentation) Draw(screen *ebiten.Image) {
	if p.toolbarOffset() == 0 {
		p.Game.Draw(screen)
		return
	}
	if p.gameSurface == nil {
		p.gameSurface = ebiten.NewImage(logicalW, logicalH)
	}
	p.gameSurface.Clear()
	p.Game.Draw(p.gameSurface)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, float64(p.toolbarOffset()))
	screen.DrawImage(p.gameSurface, op)
}

func playerLocaleLabel(locale string) string {
	switch locale {
	case "en":
		return "English  F4"
	case "ja":
		return "日本語  F4"
	case "zh-Hans":
		return "简体中文  F4"
	default:
		return "繁體中文  F4"
	}
}

func playerThemeLabels(locale string) [3]string {
	switch locale {
	case "en":
		return [3]string{"Original", "HD", "F2: switch theme"}
	case "ja":
		return [3]string{"原版", "HD", "F2：テーマ切替"}
	case "zh-Hans":
		return [3]string{"原版", "HD", "F2：切换主题"}
	default:
		return [3]string{"原版", "HD", "F2：切換主題"}
	}
}

func (p *modernHDPresentation) drawThemeToolbar(screen ebiten.FinalScreen, geometry ebiten.GeoM) {
	const factor = 3
	if p.toolbarSurface == nil {
		p.toolbarSurface = ebiten.NewImage(logicalW*factor, playerToolbarHeight*factor)
	}
	surface := p.toolbarSurface
	surface.Fill(color.RGBA{17, 24, 37, 255})
	vector.DrawFilledRect(surface, 0, playerToolbarHeight*factor-3, logicalW*factor, 3, color.RGBA{87, 105, 127, 255}, false)
	labels := playerThemeLabels(p.Game.localeID)
	for index, rect := range playerThemeButtons {
		p.drawCompactToolbarButton(surface, rect, labels[index], (index == 1) == (p.Game.modernStoryPortraits != nil))
	}
	mode := "原版"
	if p.Game.modernHDInterfaceEnabled() {
		mode = "HD"
	}
	label := "介面：" + mode + " F11"
	switch p.Game.localeID {
	case "en":
		if mode != "HD" {
			mode = "Original"
		}
		label = "UI: " + mode + " F11"
	case "ja":
		label = "UI：" + mode + " F11"
	case "zh-Hans":
		label = "界面：" + mode + " F11"
	}
	p.drawCompactToolbarButton(surface, playerInterfaceButton, label, p.Game.modernHDInterfaceEnabled())
	p.drawCompactToolbarButton(surface, playerLocaleButton, playerLocaleLabel(p.Game.localeID), false)
	p.drawCompactToolbarButton(surface, playerFontButton, fmt.Sprintf("A+ %.0f%%", normalizeFontScale(p.Game.fontScale)*100), false)
	p.drawCompactToolbarButton(surface, playerSpeedButton, fmt.Sprintf("%d× F8", normalizePlayerSpeed(p.Game.playerSpeed)), p.Game.playerSpeed > 1)
	full := "全螢幕"
	switch p.Game.localeID {
	case "en":
		full = "Full"
	case "ja":
		full = "全画面"
	case "zh-Hans":
		full = "全屏"
	}
	p.drawCompactToolbarButton(surface, playerFullscreenButton, full, ebiten.IsFullscreen())
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(1.0/factor, 1.0/factor)
	op.GeoM.Concat(geometry)
	screen.DrawImage(surface, op)
}

func playerToolbarLabelLayout(f *Font, label string, rect image.Rectangle) (font.Face, string, image.Point) {
	box := image.Rect(rect.Min.X*3+12, rect.Min.Y*3+6, rect.Max.X*3-12, rect.Max.Y*3-6)
	var face font.Face
	var ink image.Rectangle
	for size := 36; size >= 24; size-- {
		face, _ = f.faceFor(size)
		if face == nil {
			continue
		}
		ink = text.BoundString(face, label)
		if ink.Dx() <= box.Dx() && ink.Dy() <= box.Dy() {
			break
		}
	}
	if face == nil {
		return nil, "", image.Point{}
	}
	for text.BoundString(face, label).Dx() > box.Dx() {
		runes := []rune(label)
		if len(runes) <= 1 {
			label = ""
			break
		}
		if runes[len(runes)-1] == '…' {
			runes = runes[:len(runes)-1]
		}
		label = string(runes[:len(runes)-1]) + "…"
	}
	ink = text.BoundString(face, label)
	return face, label, image.Pt(box.Min.X+(box.Dx()-ink.Dx())/2-ink.Min.X, box.Min.Y+(box.Dy()-ink.Dy())/2-ink.Min.Y)
}

func (p *modernHDPresentation) drawCompactToolbarButton(surface *ebiten.Image, rect image.Rectangle, label string, selected bool) {
	fill, border := color.RGBA{30, 41, 58, 255}, color.RGBA{88, 104, 126, 255}
	if selected {
		fill, border = color.RGBA{67, 57, 38, 255}, color.RGBA{220, 185, 112, 255}
	}
	x, y, w, h := float32(rect.Min.X*3), float32(rect.Min.Y*3), float32(rect.Dx()*3), float32(rect.Dy()*3)
	vector.DrawFilledRect(surface, x, y, w, h, border, false)
	vector.DrawFilledRect(surface, x+3, y+3, w-6, h-6, fill, false)
	f := p.Game.font
	if f == nil {
		return
	}
	if regional := f.hdFonts[p.Game.localeID]; regional != nil {
		f = regional
	}
	face, caption, at := playerToolbarLabelLayout(f, label, rect)
	if face != nil {
		text.Draw(surface, caption, face, at.X, at.Y, color.RGBA{242, 240, 226, 255})
	}
}

func (p *modernHDPresentation) drawToolbarLabel(surface *ebiten.Image, label string, x, y int) {
	f := p.Game.font
	if f == nil {
		ebitenutil.DebugPrintAt(surface, label, x, y)
		return
	}
	if regional := f.hdFonts[p.Game.localeID]; regional != nil {
		f = regional
	}
	face, _ := f.faceFor(36)
	if face != nil {
		text.Draw(surface, label, face, x, playerToolbarTextBaseline(face, label), color.RGBA{242, 240, 226, 255})
	}
}

func playerToolbarTextBaseline(face font.Face, label string) int {
	ink, _ := font.BoundString(face, label)
	return playerToolbarHeight*3/2 - (ink.Min.Y.Floor()+ink.Max.Y.Ceil())/2
}
