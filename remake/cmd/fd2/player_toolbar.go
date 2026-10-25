package main

import (
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

var playerLocaleButton = image.Rect(304, 3, 464, 21)

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
		p.toolbarTicks = playerToolbarHideTicks
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
	image.Rect(8, 3, 144, 21),
	image.Rect(152, 3, 288, 21),
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
		if hd, ok := playerThemeAt(x, y); ok {
			p.requestToolbarTheme(hd)
		} else if image.Pt(x, y).In(playerLocaleButton) {
			p.Game.playerLocaleRequested = true
		}
		return
	}
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		if hd, ok := playerThemeAt(x, y); ok {
			p.requestToolbarTheme(hd)
			return
		}
		if image.Pt(x, y).In(playerLocaleButton) {
			p.Game.playerLocaleRequested = true
			return
		}
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
		return [3]string{"Original pixels", "HD artwork", "F2: switch theme"}
	case "ja":
		return [3]string{"オリジナル", "HDイラスト", "F2：テーマ切替"}
	case "zh-Hans":
		return [3]string{"原版像素", "HD手绘", "F2：切换主题"}
	default:
		return [3]string{"原版像素", "HD手繪", "F2：切換主題"}
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
		selected := (index == 1) == (p.Game.modernStoryPortraits != nil)
		fill, border := color.RGBA{30, 41, 58, 255}, color.RGBA{88, 104, 126, 255}
		if selected {
			fill, border = color.RGBA{67, 57, 38, 255}, color.RGBA{220, 185, 112, 255}
		}
		x, y, w, h := float32(rect.Min.X*factor), float32(rect.Min.Y*factor), float32(rect.Dx()*factor), float32(rect.Dy()*factor)
		vector.DrawFilledRect(surface, x, y, w, h, border, false)
		vector.DrawFilledRect(surface, x+3, y+3, w-6, h-6, fill, false)
		p.drawToolbarLabel(surface, labels[index], (rect.Min.X+8)*factor, 5*factor)
	}
	rect := playerLocaleButton
	vector.DrawFilledRect(surface, float32(rect.Min.X*factor), float32(rect.Min.Y*factor), float32(rect.Dx()*factor), float32(rect.Dy()*factor), color.RGBA{88, 104, 126, 255}, false)
	vector.DrawFilledRect(surface, float32((rect.Min.X+1)*factor), float32((rect.Min.Y+1)*factor), float32((rect.Dx()-2)*factor), float32((rect.Dy()-2)*factor), color.RGBA{30, 41, 58, 255}, false)
	p.drawToolbarLabel(surface, playerLocaleLabel(p.Game.localeID), (rect.Min.X+8)*factor, 5*factor)
	p.drawToolbarLabel(surface, "F2 / F4", 520*factor, 5*factor)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(1.0/factor, 1.0/factor)
	op.GeoM.Concat(geometry)
	screen.DrawImage(surface, op)
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
