// font.go — TTF 中文字型渲染(doc 18 字型現代化)。
//
// 設計決策:台詞/物品名都已解碼成 UTF-8(extracted/story/*),remake 用現代 TTF
// (Noto Sans CJK)直接渲染 UTF-8,清晰可縮放、支援任意字,不受原版 16×16 點陣字限制。
// 繁體 codepoint 在 Noto CJK 都有 glyph;原版自製「特殊代號字模」(機器人篇,KB 有記錄)未來再特例。
//
// 銳利度:Draw(scale) 不做 GeoM 縮放(非整數縮放重採樣=糊字根因,狀態欄名字踩過),
// 而是把 scale 換算成目標像素尺寸,用 per-尺寸 face 快取直接 rasterize(scale 1.0 繪製)。
package main

import (
	"image/color"
	"math"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// Font 包一個 TTF,依目標像素尺寸 lazily 建 face(銳利,無縮放重採樣)。
type Font struct {
	sf    *sfnt.Font
	base  float64 // Draw 的 scale=1.0 對應的像素尺寸
	faces map[int]font.Face
	ascs  map[int]float64
}

// 字型搜尋路徑:打包用 assets 優先,否則用系統 Noto CJK(桌面)。
var fontPaths = []string{
	"assets/fonts/cjk.ttc",
	"assets/fonts/cjk.otf",
	"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/truetype/arphic/uming.ttc",
}

const fontSize = 18.0

// loadFont 載入 CJK TTF/ttc(scale=1 → 18px)。失敗回 nil。
func loadFont() *Font { return loadFontSized(fontSize) }

// loadFontSized 指定 scale=1 的像素尺寸(狀態欄等固定尺寸用)。
func loadFontSized(size float64) *Font {
	var data []byte
	for _, p := range fontPaths {
		if d, e := os.ReadFile(assetPath(p)); e == nil {
			data = d
			break
		}
	}
	if data == nil {
		return nil
	}
	var sf *sfnt.Font
	if coll, err := sfnt.ParseCollection(data); err == nil {
		if f0, e := coll.Font(0); e == nil {
			sf = f0
		}
	}
	if sf == nil {
		if f0, err := sfnt.Parse(data); err == nil {
			sf = f0
		}
	}
	if sf == nil {
		return nil
	}
	return &Font{sf: sf, base: size, faces: map[int]font.Face{}, ascs: map[int]float64{}}
}

// faceFor 取(或建)目標像素尺寸的 face。
func (f *Font) faceFor(px int) (font.Face, float64) {
	if px < 6 {
		px = 6
	}
	if fc, ok := f.faces[px]; ok {
		return fc, f.ascs[px]
	}
	fc, err := opentype.NewFace(f.sf, &opentype.FaceOptions{Size: float64(px), DPI: 72, Hinting: font.HintingFull})
	if err != nil || fc == nil {
		return nil, 0
	}
	asc := float64(fc.Metrics().Ascent.Round())
	f.faces[px] = fc
	f.ascs[px] = asc
	return fc, asc
}

// Draw 畫文字(支援 \n)。scale 相對 base 尺寸;內部用整數尺寸 face 直接 rasterize(銳利)。
func (f *Font) Draw(dst *ebiten.Image, s string, x, y, scale float64, clr color.RGBA) float64 {
	px := int(math.Round(f.base * scale))
	face, asc := f.faceFor(px)
	if face == nil {
		return y
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(x), math.Round(y)+asc) // y=頂部 → baseline;座標取整對齊像素格
	op.ColorScale.ScaleWithColor(clr)
	text.DrawWithOptions(dst, s, face, op)
	return y
}

// Width 估算一行寬(像素)。
func (f *Font) Width(s string, scale float64) float64 {
	px := int(math.Round(f.base * scale))
	face, _ := f.faceFor(px)
	if face == nil {
		return 0
	}
	return float64(text.BoundString(face, s).Dx())
}

// Wrap 依實際 glyph 寬度切行。繁中句子多半沒有空白，因此以 rune 為最小
// 邊界；明示換行仍保留。這是 ending／說明頁使用的顯示工具，不改劇本文字。
func (f *Font) Wrap(s string, scale, maxWidth float64) []string {
	return wrapTextByWidth(s, maxWidth, func(line string) float64 {
		return f.Width(line, scale)
	})
}

func wrapTextByWidth(s string, maxWidth float64, width func(string) float64) []string {
	var lines []string
	for _, paragraph := range strings.Split(s, "\n") {
		line := ""
		for _, r := range paragraph {
			candidate := line + string(r)
			if line != "" && width(candidate) > maxWidth {
				lines = append(lines, line)
				line = string(r)
			} else {
				line = candidate
			}
		}
		lines = append(lines, line)
	}
	return lines
}

// LineHeight 一行高(像素,含行距)。
func (f *Font) LineHeight(scale float64) float64 { return f.base * 1.3 * scale }
