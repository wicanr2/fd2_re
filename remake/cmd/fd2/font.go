// font.go — TTF 中文字型渲染(doc 18 字型現代化)。
//
// 設計決策:台詞/物品名都已解碼成 UTF-8(extracted/story/*),remake 用現代 TTF
// (Noto Sans CJK)直接渲染 UTF-8，清晰可縮放，且不受原版 16×16 點陣字索引限制。
// 實際字形覆蓋仍由載入的字型檔決定；不可假設所有繁體 codepoint 必然存在。
// 原版自製「特殊代號字模」（機器人篇，知識庫有記錄）仍須逐字建立明確特例。
//
// 銳利度:Draw(scale) 不做 GeoM 縮放(非整數縮放重採樣=糊字根因,狀態欄名字踩過),
// 而是把 scale 換算成目標像素尺寸,用 per-尺寸 face 快取直接 rasterize(scale 1.0 繪製)。
package main

import (
	"image/color"
	"math"
	"os"
	"strings"
	"unicode"

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
	// userScale 是玩家在設定裡選的字級倍率，套在每一次 Draw／Width 的 scale 上。
	// 英文與日文譯文比中文長得多，同一個對話框裡常常斷不完；把字級調小一格就能
	// 多塞一行。0 視同 1.0。索引畫面用的是原版字模，不經過這裡，所以調整字級不會
	// 動到與原版對拍的那些畫面。
	userScale float64
}

// SetUserScale 設定字級倍率。超出合理範圍的值會被夾住——字太小讀不到、太大反而
// 更容易溢出版面。
func (f *Font) SetUserScale(scale float64) {
	if f == nil {
		return
	}
	if scale < 0.6 {
		scale = 0.6
	}
	if scale > 1.6 {
		scale = 1.6
	}
	f.userScale = scale
}

// effectiveScale 把呼叫端要的 scale 乘上玩家設定。
func (f *Font) effectiveScale(scale float64) float64 {
	if f == nil || f.userScale <= 0 {
		return scale
	}
	return scale * f.userScale
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
	px := int(math.Round(f.base * f.effectiveScale(scale)))
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
	px := int(math.Round(f.base * f.effectiveScale(scale)))
	face, _ := f.faceFor(px)
	if face == nil {
		return 0
	}
	return float64(text.BoundString(face, s).Dx())
}

// Wrap 依實際 glyph 寬度切行。斷點以 textSegments 為單位：繁中每個字都能斷，
// 英文與日文譯文裡的拉丁單字整串搬，不會斷在單字中間。明示換行仍保留。這是
// ending／說明頁與商店訊息使用的顯示工具，不改劇本文字。
func (f *Font) Wrap(s string, scale, maxWidth float64) []string {
	return wrapTextByWidth(s, maxWidth, func(line string) float64 {
		return f.Width(line, scale)
	})
}

// runeBreakable 判斷這個字元自己就能當一個斷行單位。中日韓文字與全形標點
// 每個字都可以斷，拉丁字母與數字不行——那會把英文單字切成兩半。
func runeBreakable(r rune) bool {
	if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) {
		return true
	}
	switch {
	case r >= 0x3000 && r <= 0x303F: // 全形標點
		return true
	case r >= 0xFF00 && r <= 0xFFEF: // 全形英數與半形片假名
		return true
	}
	return false
}

// textSegments 把一段文字切成不可拆的顯示單位。空白是斷點，黏在前一段的尾巴——
// 留著不砍，把每一行接回去才會等於原文（逐字表示與分頁都靠這個性質）；斷在那裡
// 時它落在行尾，畫出來本來就看不見。
func textSegments(paragraph string) []string {
	var segments []string
	var current strings.Builder
	flush := func() {
		if current.Len() > 0 {
			segments = append(segments, current.String())
			current.Reset()
		}
	}
	for _, r := range paragraph {
		switch {
		case r == ' ' || r == '\t':
			if current.Len() == 0 && len(segments) > 0 {
				segments[len(segments)-1] += string(r)
			} else {
				current.WriteRune(r)
				flush()
			}
		case runeBreakable(r):
			flush()
			segments = append(segments, string(r))
		default:
			current.WriteRune(r)
		}
	}
	flush()
	return segments
}

func wrapTextByWidth(s string, maxWidth float64, width func(string) float64) []string {
	var lines []string
	for _, paragraph := range strings.Split(s, "\n") {
		line := ""
		for _, segment := range textSegments(paragraph) {
			if line != "" && width(line+segment) > maxWidth {
				lines = append(lines, line)
				line = ""
			}
			if line == "" && width(segment) > maxWidth {
				// 這一段自己就放不下：逐字元硬拆，總比整行溢出好。
				for _, r := range segment {
					if line != "" && width(line+string(r)) > maxWidth {
						lines = append(lines, line)
						line = ""
					}
					line += string(r)
				}
				continue
			}
			line += segment
		}
		lines = append(lines, line)
	}
	return lines
}

// LineHeight 一行高(像素,含行距)。
func (f *Font) LineHeight(scale float64) float64 { return f.base * 1.3 * scale }
