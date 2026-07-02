// font.go — TTF 中文字型渲染(doc 18 字型現代化)。
//
// 設計決策:台詞/物品名都已解碼成 UTF-8(extracted/story/*),remake 用現代 TTF
// (Noto Sans CJK)直接渲染 UTF-8,清晰可縮放、支援任意字,不受原版 16×16 點陣字限制。
// 繁體 codepoint 在 Noto CJK 都有 glyph;原版自製「特殊代號字模」(機器人篇,KB 有記錄)未來再特例。
package main

import (
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// Font 包一個 TTF face(ebiten 2.6 text API)。
type Font struct {
	face font.Face
	asc  float64 // ascent(baseline 偏移,y=頂部 → +asc 才是 baseline)
}

// 字型搜尋路徑:打包用 assets 優先,否則用系統 Noto CJK(桌面)。
var fontPaths = []string{
	"assets/fonts/cjk.ttc",
	"assets/fonts/cjk.otf",
	"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/truetype/arphic/uming.ttc",
}

const fontSize = 18.0

// loadFont 載入 CJK TTF/ttc(預設 18px)。失敗回 nil。
func loadFont() *Font { return loadFontSized(fontSize) }

// loadFontSized 以指定像素尺寸建 face:目標尺寸直接 rasterize + scale 1.0 繪製,
// 避免非整數 GeoM 縮放造成的重採樣模糊(狀態欄名字/數字用)。
func loadFontSized(size float64) *Font {
	var data []byte
	for _, p := range fontPaths {
		if d, e := os.ReadFile(p); e == nil {
			data = d
			break
		}
	}
	if data == nil {
		return nil
	}
	opts := &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull}
	var face font.Face
	if coll, err := opentype.ParseCollection(data); err == nil { // ttc
		if sf, e := coll.Font(0); e == nil {
			face, _ = opentype.NewFace(sf, opts)
		}
	}
	if face == nil {
		if sf, err := opentype.Parse(data); err == nil { // 單 face
			face, _ = opentype.NewFace(sf, opts)
		}
	}
	if face == nil {
		return nil
	}
	m := face.Metrics()
	return &Font{face: face, asc: float64(m.Ascent.Round())}
}

// Draw 畫文字(支援 \n)。scale 相對 fontSize 放大;clr 上色。回傳起始 y。
func (f *Font) Draw(dst *ebiten.Image, s string, x, y, scale float64, clr color.RGBA) float64 {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y+f.asc*scale) // y=頂部 → 移到 baseline
	op.ColorScale.ScaleWithColor(clr)
	text.DrawWithOptions(dst, s, f.face, op)
	return y
}

// Width 估算一行寬(像素)。
func (f *Font) Width(s string, scale float64) float64 {
	return float64(text.BoundString(f.face, s).Dx()) * scale
}

// LineHeight 一行高(像素,含行距)。
func (f *Font) LineHeight(scale float64) float64 { return fontSize * 1.3 * scale }
