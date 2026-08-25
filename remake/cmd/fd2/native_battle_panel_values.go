package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type nativeBattlePanelValueKey struct {
	Level, HP, MaxHP, MP, MaxMP int
}

// drawNativeBattlePanelValues 將既有0x18C6D indexed核心接到普通全螢幕攻擊。
// caller仍只提供E1可見值與已選panel原點；它不冒充raw +6／+8的完整轉接器。
func (g *Game) drawNativeBattlePanelValues(
	screen *ebiten.Image,
	x, y float64,
	values battle.NativeBattlePanelValues,
) bool {
	if g == nil || g.nativeBattlePanel == nil || len(g.nativeUIPalette) != 256 {
		return false
	}
	key := nativeBattlePanelValueKey{
		Level: values.Level, HP: values.HP, MaxHP: values.MaxHP,
		MP: values.MP, MaxMP: values.MaxMP,
	}
	if g.nativeBattleValues == nil {
		g.nativeBattleValues = make(map[nativeBattlePanelValueKey]*ebiten.Image)
	}
	panel := g.nativeBattleValues[key]
	if panel == nil {
		indexed := make([]byte, 320*200)
		if err := battle.RenderNativeBattlePanelValuesAt(
			*g.nativeBattlePanel, indexed, 0, 0, values,
		); err != nil {
			return false
		}
		paletted := image.NewPaletted(image.Rect(0, 0, 149, 42), g.nativeUIPalette)
		for row := 0; row < 42; row++ {
			copy(
				paletted.Pix[row*paletted.Stride:row*paletted.Stride+149],
				indexed[row*320:row*320+149],
			)
		}
		panel = ebiten.NewImageFromImage(paletted)
		g.nativeBattleValues[key] = panel
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(x, y)
	screen.DrawImage(panel, op)
	return true
}
