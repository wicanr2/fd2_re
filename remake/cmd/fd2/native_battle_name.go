package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

const (
	nativeBattleNameOriginX = 5
	nativeBattleNameOriginY = 4
)

// nativeBattleNameAssets 是全螢幕戰鬥狀態欄的原版姓名字模來源。
// 原版 0x18c6d→0x15f84 走 FDOTHER#4 的 16×16 1bpp glyph，不與對話／現代
// UTF-8 字型共用；索引表仍是可編輯資產，未知字元一律拒絕 native 路徑。
func loadNativeBattleNameAssets() (*fdtxt.Font, map[string]int, error) {
	font, err := fdtxt.LoadSeparatedFont(separatedAssetPath("fonts"))
	if err != nil {
		return nil, nil, err
	}
	indexRaw, err := os.ReadFile(assetPath("assets/fonts/unicode_to_glyph.json"))
	if err != nil {
		return nil, nil, err
	}
	encoded := make(map[string]json.RawMessage)
	if err := json.Unmarshal(indexRaw, &encoded); err != nil {
		return nil, nil, err
	}
	index := make(map[string]int, len(encoded))
	for key, value := range encoded {
		if key == "_comment" {
			continue
		}
		var glyph int
		if err := json.Unmarshal(value, &glyph); err != nil {
			return nil, nil, fmt.Errorf("native battle name glyph %q: %w", key, err)
		}
		index[key] = glyph
	}
	for key, glyph := range index {
		if glyph < 0 || glyph >= font.GlyphCount() {
			return nil, nil, fmt.Errorf("native battle name glyph %q=%d is outside font", key, glyph)
		}
	}
	return font, index, nil
}

// renderNativeBattleNameIndexed reproduces the visible 0x4ea2a glyph writes in
// a small indexed surface. The one-pixel left margin is intentional: native
// shadow writes use x-1 and the next row, so the caller must not clip it.
func renderNativeBattleNameIndexed(font *fdtxt.Font, index map[string]int, name string, foreground, shadow byte) ([]byte, int, int, error) {
	if font == nil || index == nil || name == "" {
		return nil, 0, 0, fmt.Errorf("native battle name assets are unavailable")
	}
	keys := make([]string, 0, len([]rune(name)))
	for _, r := range name {
		key := string(r)
		glyph, ok := index[key]
		if !ok {
			return nil, 0, 0, fmt.Errorf("native battle name glyph %q is unavailable", key)
		}
		if glyph < 0 || glyph >= font.GlyphCount() {
			return nil, 0, 0, fmt.Errorf("native battle name glyph %q=%d is invalid", key, glyph)
		}
		keys = append(keys, key)
	}
	stride := len(keys)*fdtxt.GlyphWidth + 2
	height := fdtxt.GlyphHeight + 1 // native shadow may land on the following row
	indexed := make([]byte, stride*height)
	for i, key := range keys {
		if err := font.BlitNativeGlyph(indexed, stride, 1+i*fdtxt.GlyphWidth, index[key], fdtxt.NativeGlyphStyle{
			Foreground: foreground,
			Shadow:     shadow,
		}); err != nil {
			return nil, 0, 0, err
		}
	}
	return indexed, stride, height, nil
}

func nativePaletteColor(palette color.Palette, index byte, fallback color.RGBA) color.RGBA {
	if int(index) >= len(palette) {
		return fallback
	}
	r, g, b, a := palette[index].RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

// nativeBattleNameImage converts the indexed native glyph surface to an
// alpha image for the existing 2× logical canvas. Palette indices are kept
// explicit; zero remains transparent as in 0x4ea2a's destination-preserving
// glyph blit.
func nativeBattleNameImage(font *fdtxt.Font, index map[string]int, palette color.Palette, name string) (*ebiten.Image, bool) {
	indexed, stride, height, err := renderNativeBattleNameIndexed(font, index, name, 0xcd, 0x4c)
	if err != nil {
		return nil, false
	}
	img := image.NewNRGBA(image.Rect(0, 0, stride, height))
	foreground := nativePaletteColor(palette, 0xcd, color.RGBA{0xe0, 0xee, 0xff, 0xff})
	shadow := nativePaletteColor(palette, 0x4c, color.RGBA{0x20, 0x30, 0x60, 0xff})
	for y := 0; y < height; y++ {
		for x := 0; x < stride; x++ {
			switch indexed[y*stride+x] {
			case 0xcd:
				img.SetNRGBA(x, y, color.NRGBA(foreground))
			case 0x4c:
				img.SetNRGBA(x, y, color.NRGBA(shadow))
			}
		}
	}
	return ebiten.NewImageFromImage(img), true
}

func (g *Game) drawNativeBattleName(screen *ebiten.Image, x, y float64, name string) bool {
	if g == nil {
		return false
	}
	img, ok := nativeBattleNameImage(g.nativeBattleFont, g.nativeBattleGlyphs, g.nativeUIPalette, name)
	if !ok {
		return false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	// 索引輔助函式為 x-1 陰影保留一個原生像素。0x18C6D 傳給 0x15F84 的
	// 座標是 panel+(5,4)，因此暫存面從原生 x=4 開始，第一個前景像素仍在 x=5。
	op.GeoM.Translate(x+(nativeBattleNameOriginX-1)*2, y+nativeBattleNameOriginY*2)
	screen.DrawImage(img, op)
	return true
}
