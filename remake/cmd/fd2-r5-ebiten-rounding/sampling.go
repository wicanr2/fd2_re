package main

// R5 階段 1（docs/goal/121）補測：DrawImage 的最近鄰取樣與 Fill／SubImage／Set／NewImageFromImage 語意。
//
// 取樣實驗的來源圖以座標編碼：像素 (x, y) 為 R=x&255、G=y&255、B=0x80|((x>>8)<<3)|(y>>8)、A=255，
// 畫到全透明的 640×400 目標；讀回 alpha 0 是沒有覆蓋，255 是覆蓋，RGB 解回來源座標。GeoM 只有縮放與平移，
// 所以每個實驗先確認讀回可以拆成「目標欄 → 來源欄」與「目標列 → 來源列」兩張對應表，再拿每張表和候選規則
// 逐值比對。倍率與平移取自原版模式程式的呼叫點（見 samplingCases）。
//
// 語意實驗逐像素比對讀回值與預期公式，記不符數與前幾筆。

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	sampW = 640
	sampH = 400
)

type axisMap struct {
	Covered []int `json:"covered"` // [第一個, 最後一個] 覆蓋的目標座標；沒有覆蓋時為 [-1,-1]
	Index   []int `json:"index"`   // 覆蓋範圍內每個目標座標對到的來源座標（相對來源圖左上）
}

type axisCandidate struct {
	Name       string `json:"name"`
	Mismatches int    `json:"mismatches"`
	First      []int  `json:"first_mismatch_coords,omitempty"` // 不符的目標座標（最多 8 個）
}

type samplingResult struct {
	Name       string          `json:"name"`
	Source     string          `json:"source"`
	SrcW       int             `json:"src_w"`
	SrcH       int             `json:"src_h"`
	Ops        []string        `json:"geom_ops"` // 依序套用的 GeoM 操作
	GeoM       [6]float64      `json:"geom"`     // a, b, c, d, tx, ty（Element）
	RGBASHA256 string          `json:"actual_rgba_sha256"`
	Separable  bool            `json:"separable"`
	X          axisMap         `json:"x"`
	Y          axisMap         `json:"y"`
	ExactBoth  []string        `json:"exact"`
	CandX      []axisCandidate `json:"candidates_x"`
	CandY      []axisCandidate `json:"candidates_y"`
}

type geomOp struct {
	kind string // "scale" 或 "translate"
	x, y float64
}

type samplingCase struct {
	name   string
	source string
	w, h   int
	ops    []geomOp
}

func coordImage(w, h int) []byte {
	pix := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			pix[i] = byte(x)
			pix[i+1] = byte(y)
			pix[i+2] = byte(0x80 | ((x >> 8) << 3) | (y >> 8))
			pix[i+3] = 255
		}
	}
	return pix
}

func (c samplingCase) geoM() ebiten.GeoM {
	var g ebiten.GeoM
	for _, op := range c.ops {
		if op.kind == "scale" {
			g.Scale(op.x, op.y)
		} else {
			g.Translate(op.x, op.y)
		}
	}
	return g
}

func (c samplingCase) opStrings() []string {
	out := []string{}
	for _, op := range c.ops {
		out = append(out, fmt.Sprintf("%s(%v,%v)", op.kind, op.x, op.y))
	}
	return out
}

// 一軸的候選規則：給定 GeoM 的該軸係數（a、t，float64）與來源長度 n，回傳目標座标 p 的來源索引，
// 沒有覆蓋回 -1。
type axisRule struct {
	name string
	f    func(a, t float64, n, p int) int
}

// ebitenAdjust 是 Ebiten v2.6.6 internal/graphics/vertex.go:113 adjustDestinationPixel：
// 目標頂點對齊到約 1/3 像素（小數 <3/16 → 0、<8/16 → 5/16、<13/16 → 11/16、其餘 → 1）。
func ebitenAdjust(x float32) float32 {
	ix := float32(int(x))
	if x < 0 && x != ix {
		ix -= 1
	}
	frac := x - ix
	switch {
	case frac < 3.0/16.0:
		return ix
	case frac < 8.0/16.0:
		return ix + 5.0/16.0
	case frac < 13.0/16.0:
		return ix + 11.0/16.0
	default:
		return ix + 16.0/16.0
	}
}

// ebitenAxis32：Ebiten 送進 GPU 的頂點。GeoM 存 a-1，elements32 是 float32(a-1)+1；
// QuadVertices 以 float32 算 a*n+t 再對齊。
func ebitenAxis32(a, t float64, n int) (x0, x1 float32) {
	a32 := float32(a-1) + 1
	t32 := float32(t)
	ax := float32(a32 * float32(n))
	return ebitenAdjust(t32), ebitenAdjust(float32(ax + t32))
}

var axisRules = []axisRule{
	{"Ebiten 對齊頂點＋像素中心 float64 內插 floor（左含右不含）", func(a, t float64, n, p int) int {
		x0, x1 := ebitenAxis32(a, t, n)
		c := float64(p) + 0.5
		if !(c >= float64(x0) && c < float64(x1)) {
			return -1
		}
		u := (c - float64(x0)) * float64(n) / (float64(x1) - float64(x0))
		return int(math.Floor(u))
	}},
	{"Ebiten 對齊頂點＋像素中心 float32 內插 floor", func(a, t float64, n, p int) int {
		x0, x1 := ebitenAxis32(a, t, n)
		c := float32(p) + 0.5
		if !(c >= x0 && c < x1) {
			return -1
		}
		u := float32(float32(c-x0)*float32(n)) / float32(x1-x0)
		return int(math.Floor(float64(u)))
	}},
	{"Ebiten 對齊頂點、內插用未對齊的 float64 頂點", func(a, t float64, n, p int) int {
		x0, x1 := ebitenAxis32(a, t, n)
		c := float64(p) + 0.5
		if !(c >= float64(x0) && c < float64(x1)) {
			return -1
		}
		u := (c - t) / a
		return int(math.Floor(u))
	}},
	{"不對齊：像素中心 (p+0.5-t)/a floor，覆蓋 0≤u<n", func(a, t float64, n, p int) int {
		u := (float64(p) + 0.5 - t) / a
		if !(u >= 0 && u < float64(n)) {
			return -1
		}
		return int(math.Floor(u))
	}},
	{"不對齊：像素左緣 (p-t)/a 四捨五入", func(a, t float64, n, p int) int {
		u := math.Floor((float64(p)-t)/a + 0.5)
		if !(u >= 0 && u < float64(n)) {
			return -1
		}
		return int(u)
	}},
}

func samplingCases() []samplingCase {
	cs := []samplingCase{
		{"native_scale2", "main.go:12442、12460 原生 320×200 索引整幀 GeoM.Scale(2,2)", 320, 200, []geomOp{{"scale", 2, 2}}},
		{"story_zoom2", "main.go:9908 storyView 320×200 GeoM.Scale(storyZoom=2)", 320, 200, []geomOp{{"scale", 2, 2}}},
	}
	for _, sy := range []float64{-3, -6, -1.5} {
		cs = append(cs, samplingCase{fmt.Sprintf("title_scroll_%g", sy), "title.go:752 Scale(2,2)＋Translate(0,-scrollY*2)，scrollY 每 tick 減 1.5", 320, 200,
			[]geomOp{{"scale", 2, 2}, {"translate", 0, sy * 2}}})
	}
	// 正規化圖層：圖塊與 sprite 以 float64(cx*tw)-camX、+OffX、y-4 平移到 storyView（倍率 1）；camX 線性內插
	// （main.go:1230）、OffX 依 frac（main.go:1531、6503），平移可為任意小數。
	fracs := []float64{0, 0.0625, 0.1875, 0.25, 0.3, 0.3125, 0.4999, 0.5, 0.5001, 0.6875, 0.75, 0.8125, 0.9, 0.999, 1.0 / 3, 24.0 / 7}
	for i, fx := range fracs {
		fy := fracs[(i+5)%len(fracs)]
		cs = append(cs, samplingCase{fmt.Sprintf("frac_translate_%02d", i), "main.go:9840、11542 倍率 1 的小數平移（camX、OffX）", 48, 40,
			[]geomOp{{"translate", 100 + fx, 50 + fy}}})
		cs = append(cs, samplingCase{fmt.Sprintf("frac_translate_neg_%02d", i), "同上，負平移（左上出界）", 48, 40,
			[]geomOp{{"translate", -7 - fx, -5 - fy}}})
	}
	for _, camX := range []float64{13.714285714285714, 37.5, 8.333333333333334, 100.25} {
		cs = append(cs, samplingCase{fmt.Sprintf("tile_cam_%g", camX), "main.go:9840 tile Translate(cx*tw-camX, cy*th-camY)，tw=th=24", 24, 24,
			[]geomOp{{"translate", 148 - camX, 60 - camX/3}}})
	}
	for k := 1; k <= 7; k++ {
		sc := 1 - float64(k)/7
		if k == 7 {
			sc = 1
		}
		for _, by := range []float64{198, 4} {
			cs = append(cs, samplingCase{fmt.Sprintf("dialog_k%d_by%g", k, by), "main.go:10011 舊對話框 Scale(2,2*sc)＋Translate(10,by+(1-sc)*99)（開場與第一章沒走到）", 310, 99,
				[]geomOp{{"scale", 2, 2 * sc}, {"translate", 10, by + (1-sc)*99}}})
		}
	}
	cs = append(cs,
		samplingCase{"portrait_lower", "main.go:10053 頭像 Scale(2.1)，下框 hx=16、hy=by+(198-80*2.1)/2（沒走到）", 80, 80,
			[]geomOp{{"scale", 2.1, 2.1}, {"translate", 16, 198 + (198-80*2.1)/2}}},
		samplingCase{"portrait_upper", "main.go:10057 頭像上框 hx=640-16-80*2.1、by=4（沒走到）", 80, 80,
			[]geomOp{{"scale", 2.1, 2.1}, {"translate", float64(640) - 16 - 80*2.1, 4 + (198-80*2.1)/2}}},
		samplingCase{"dialog_gradient_lower", "main.go:10049 1×198 漸層 Scale(604,182/198)＋Translate(bx+8,by+8)（沒走到）", 1, 198,
			[]geomOp{{"scale", 620 - 16, 182.0 / 198.0}, {"translate", 18, 206}}},
		samplingCase{"dialog_gradient_upper", "同上，上框 by=4", 1, 198,
			[]geomOp{{"scale", 620 - 16, 182.0 / 198.0}, {"translate", 18, 12}}},
	)
	for t := 13; t <= 42; t++ {
		p := float64(t-12) / 30
		sc := 2 * (3 - 2*p)
		cs = append(cs, samplingCase{fmt.Sprintf("logo_zoom_t%02d", t), "title.go:800 標題縮放 Translate(-160,-100)＋Scale(sc)＋Translate(320,200)，sc=2*(3-2p)（開場與第一章沒走到）", 320, 200,
			[]geomOp{{"translate", -160, -100}, {"scale", sc, sc}, {"translate", 320, 200}}})
	}
	return cs
}

func runSamplingCase(c samplingCase) samplingResult {
	src := ebiten.NewImage(c.w, c.h)
	src.WritePixels(coordImage(c.w, c.h))
	dst := ebiten.NewImage(sampW, sampH)
	g := c.geoM()
	dst.DrawImage(src, &ebiten.DrawImageOptions{GeoM: g})
	pix := make([]byte, sampW*sampH*4)
	dst.ReadPixels(pix)
	src.Dispose()
	dst.Dispose()
	dumpRaw("sampling_"+c.name, pix)
	res := samplingResult{Name: c.name, Source: c.source, SrcW: c.w, SrcH: c.h, Ops: c.opStrings(), RGBASHA256: sha(pix),
		GeoM: [6]float64{g.Element(0, 0), g.Element(0, 1), g.Element(1, 0), g.Element(1, 1), g.Element(0, 2), g.Element(1, 2)}}
	// 解碼並拆成兩軸。
	mapX := make([]int, sampW)
	mapY := make([]int, sampH)
	for i := range mapX {
		mapX[i] = -2 // 尚未決定
	}
	for i := range mapY {
		mapY[i] = -2
	}
	separable := true
	set := func(m []int, k, v int) {
		if m[k] == -2 {
			m[k] = v
		} else if m[k] != v {
			separable = false
		}
	}
	for y := 0; y < sampH; y++ {
		for x := 0; x < sampW; x++ {
			i := (y*sampW + x) * 4
			switch pix[i+3] {
			case 0:
				if pix[i] != 0 || pix[i+1] != 0 || pix[i+2] != 0 {
					separable = false
				}
				continue
			case 255:
			default:
				separable = false
				continue
			}
			b := int(pix[i+2])
			if b&0x80 == 0 {
				separable = false
				continue
			}
			sx := int(pix[i]) | ((b>>3)&3)<<8
			sy := int(pix[i+1]) | (b&3)<<8
			set(mapX, x, sx)
			set(mapY, y, sy)
		}
	}
	// 覆蓋集合必須是兩軸的直積。
	for y := 0; y < sampH; y++ {
		for x := 0; x < sampW; x++ {
			covered := pix[(y*sampW+x)*4+3] == 255
			if covered != (mapX[x] >= 0 && mapY[y] >= 0) {
				separable = false
			}
		}
	}
	res.Separable = separable
	toAxis := func(m []int) ([]int, axisMap) {
		full := make([]int, len(m))
		am := axisMap{Covered: []int{-1, -1}, Index: []int{}}
		for k, v := range m {
			if v < 0 {
				v = -1
			}
			full[k] = v
			if v >= 0 {
				if am.Covered[0] < 0 {
					am.Covered[0] = k
				}
				am.Covered[1] = k
			}
		}
		if am.Covered[0] >= 0 {
			for k := am.Covered[0]; k <= am.Covered[1]; k++ {
				am.Index = append(am.Index, full[k])
			}
		}
		return full, am
	}
	fullX, ax := toAxis(mapX)
	fullY, ay := toAxis(mapY)
	res.X, res.Y = ax, ay
	if ax.Covered[0] < 0 || ay.Covered[0] < 0 {
		// 一軸完全沒有覆蓋時另一軸的對應表觀察不到，實驗本身無效。
		panic("取樣實驗 " + c.name + " 沒有覆蓋任何目標像素")
	}
	a, d, tx, ty := g.Element(0, 0), g.Element(1, 1), g.Element(0, 2), g.Element(1, 2)
	score := func(full []int, a, t float64, n int) []axisCandidate {
		out := []axisCandidate{}
		for _, r := range axisRules {
			ac := axisCandidate{Name: r.name}
			for p := range full {
				if r.f(a, t, n, p) != full[p] {
					ac.Mismatches++
					if len(ac.First) < 8 {
						ac.First = append(ac.First, p)
					}
				}
			}
			out = append(out, ac)
		}
		return out
	}
	res.CandX = score(fullX, a, tx, c.w)
	res.CandY = score(fullY, d, ty, c.h)
	res.ExactBoth = []string{}
	if separable {
		for i, r := range axisRules {
			if res.CandX[i].Mismatches == 0 && res.CandY[i].Mismatches == 0 {
				res.ExactBoth = append(res.ExactBoth, r.name)
			}
		}
	}
	return res
}

// ---------------- 語意實驗 ----------------

type semanticsResult struct {
	Name       string  `json:"name"`
	Question   string  `json:"question"`
	Expected   string  `json:"expected"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	RGBASHA256 string  `json:"actual_rgba_sha256"`
	Vectors    int     `json:"vectors"`
	Mismatches int     `json:"mismatches"`
	First      [][]int `json:"first_mismatches,omitempty"` // [x, y, 通道, 讀回, 預期]
	Params     any     `json:"params,omitempty"`
}

func patternPix(w, h int) []byte {
	pix := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			pix[i], pix[i+1], pix[i+2], pix[i+3] = byte(x*3), byte(y*5), byte(x^y), 255
		}
	}
	return pix
}

func compare(name, question, expected string, w, h int, actual, want []byte, params any) semanticsResult {
	r := semanticsResult{Name: name, Question: question, Expected: expected, Width: w, Height: h, RGBASHA256: sha(actual), Vectors: len(actual), Params: params}
	for i := range actual {
		if actual[i] != want[i] {
			r.Mismatches++
			if len(r.First) < 8 {
				p := i / 4
				r.First = append(r.First, []int{p % w, p / w, i % 4, int(actual[i]), int(want[i])})
			}
		}
	}
	dumpRaw("semantics_"+name, actual)
	return r
}

func readAll(img *ebiten.Image) []byte {
	b := img.Bounds()
	pix := make([]byte, 4*b.Dx()*b.Dy())
	img.ReadPixels(pix)
	return pix
}

var fillAlphas = []int{0, 1, 2, 0x50, 0x5c, 0x66, 0x80, 0xe8, 0xee, 0xf0, 0xf2, 0xfe, 0xff}

func mul255(x int) int { return (x + (x >> 8) + 0x80) >> 8 }

func semanticsCases() []func() semanticsResult {
	return []func() semanticsResult{
		func() semanticsResult {
			// 1×1 SubImage 逐格 Fill：值是否原樣寫入（含 R/G/B 大於 A 的非預乘值）。
			w, h := 256, len(fillAlphas)
			dst := ebiten.NewImage(w, h)
			dst.WritePixels(patternPix(w, h))
			want := make([]byte, w*h*4)
			for j, a := range fillAlphas {
				for x := 0; x < w; x++ {
					c := color.RGBA{uint8(x), uint8(255 - x), uint8(x ^ 0x55), uint8(a)}
					dst.SubImage(image.Rect(x, j, x+1, j+1)).(*ebiten.Image).Fill(c)
					i := (j*w + x) * 4
					want[i], want[i+1], want[i+2], want[i+3] = c.R, c.G, c.B, c.A
				}
			}
			out := readAll(dst)
			dst.Dispose()
			return compare("fill_values", "Fill 半透明或非預乘色時是覆蓋還是疊合；寫入值是否原樣", "覆蓋：讀回等於 color.RGBA 的四個位元組", w, h, out, want, map[string]any{"alphas": fillAlphas, "color": "(x, 255-x, x^0x55, a)"})
		},
		func() semanticsResult {
			w, h := 64, 64
			dst := ebiten.NewImage(w, h)
			dst.WritePixels(patternPix(w, h))
			c := color.RGBA{0x40, 0x80, 0xff, 0x66}
			dst.Fill(c)
			want := make([]byte, w*h*4)
			for i := 0; i < w*h; i++ {
				want[i*4], want[i*4+1], want[i*4+2], want[i*4+3] = c.R, c.G, c.B, c.A
			}
			out := readAll(dst)
			dst.Dispose()
			return compare("fill_whole", "整張不透明影像 Fill (0x40,0x80,0xff,0x66)", "覆蓋整張", w, h, out, want, nil)
		},
		func() semanticsResult {
			w, h := 64, 64
			dst := ebiten.NewImage(w, h)
			pat := patternPix(w, h)
			dst.WritePixels(pat)
			dst.Clear()
			want := make([]byte, w*h*4)
			out := readAll(dst)
			dst.Dispose()
			return compare("clear", "不透明影像 Clear", "全部 0", w, h, out, want, nil)
		},
		func() semanticsResult {
			w, h := 64, 64
			dst := ebiten.NewImage(w, h)
			pat := patternPix(w, h)
			dst.WritePixels(pat)
			c1 := color.RGBA{0xe0, 0x30, 0x30, 0x5c}
			c2 := color.RGBA{0x10, 0x1c, 0x40, 0xee}
			r1 := image.Rect(5, 7, 20, 30)
			r2 := image.Rect(50, 50, 80, 80) // 與影像相交成 (50,50)-(64,64)
			dst.SubImage(r1).(*ebiten.Image).Fill(c1)
			dst.SubImage(r2).(*ebiten.Image).Fill(c2)
			sub := dst.SubImage(image.Rect(30, 2, 45, 12)).(*ebiten.Image)
			sub.SubImage(image.Rect(25, 0, 40, 8)).(*ebiten.Image).Fill(c1) // 子影像的子影像：(30,2)-(40,8)
			want := append([]byte(nil), pat...)
			fill := func(r image.Rectangle, c color.RGBA) {
				r = r.Intersect(image.Rect(0, 0, w, h))
				for y := r.Min.Y; y < r.Max.Y; y++ {
					for x := r.Min.X; x < r.Max.X; x++ {
						i := (y*w + x) * 4
						want[i], want[i+1], want[i+2], want[i+3] = c.R, c.G, c.B, c.A
					}
				}
			}
			fill(r1, c1)
			fill(r2, c2)
			fill(image.Rect(30, 2, 40, 8), c1)
			out := readAll(dst)
			dst.Dispose()
			return compare("subimage_fill", "SubImage 後 Fill 的範圍（含出界矩形與子影像的子影像）", "只覆蓋與父影像相交的矩形；座標是父影像座標", w, h, out, want, map[string]any{"rects": [][]int{{5, 7, 20, 30}, {50, 50, 80, 80}, {30, 2, 45, 12, 25, 0, 40, 8}}})
		},
		func() semanticsResult {
			w, h := 64, 64
			dst := ebiten.NewImage(w, h)
			pat := patternPix(w, h)
			dst.WritePixels(pat)
			src := ebiten.NewImage(w, h)
			cp := coordImage(w, h)
			src.WritePixels(cp)
			sub := dst.SubImage(image.Rect(10, 12, 40, 44)).(*ebiten.Image)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(3, 4)
			sub.DrawImage(src, op)
			want := append([]byte(nil), pat...)
			for y := 12; y < 44; y++ {
				for x := 10; x < 40; x++ {
					s := ((y-4)*w + (x - 3)) * 4
					copy(want[(y*w+x)*4:(y*w+x)*4+4], cp[s:s+4])
				}
			}
			out := readAll(dst)
			dst.Dispose()
			src.Dispose()
			return compare("subimage_dst_draw", "DrawImage 畫到 SubImage：平移原點與裁切範圍", "GeoM 以父影像座標計算，只寫入子影像矩形", w, h, out, want, map[string]any{"rect": []int{10, 12, 40, 44}, "translate": []int{3, 4}})
		},
		func() semanticsResult {
			w, h := 64, 64
			src := ebiten.NewImage(w, h)
			cp := coordImage(w, h)
			src.WritePixels(cp)
			dst := ebiten.NewImage(w, h)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(30, 20)
			dst.DrawImage(src.SubImage(image.Rect(8, 9, 24, 30)).(*ebiten.Image), op)
			want := make([]byte, w*h*4)
			for j := 0; j < 21; j++ {
				for i := 0; i < 16; i++ {
					s := ((9+j)*w + 8 + i) * 4
					d := ((20+j)*w + 30 + i) * 4
					copy(want[d:d+4], cp[s:s+4])
				}
			}
			out := readAll(dst)
			dst.Dispose()
			src.Dispose()
			return compare("subimage_src_draw", "DrawImage 的來源是 SubImage", "來源矩形左上對到 GeoM 原點", w, h, out, want, map[string]any{"rect": []int{8, 9, 24, 30}, "translate": []int{30, 20}})
		},
		func() semanticsResult {
			w, h := 256, 4
			dst := ebiten.NewImage(w, h)
			want := make([]byte, w*h*4)
			for x := 0; x < w; x++ {
				cs := []color.Color{
					color.RGBA{uint8(x), uint8(255 - x), uint8(x ^ 0x55), 0x80},
					color.RGBA{uint8(x), uint8(255 - x), uint8(x ^ 0x55), 0xff},
					color.NRGBA{uint8(x), uint8(255 - x), uint8(x ^ 0x55), 0x80},
					color.NRGBA{uint8(x), uint8(255 - x), uint8(x ^ 0x55), uint8(x)},
				}
				for row, c := range cs {
					dst.Set(x, row, c)
					r, g, b, a := c.RGBA()
					i := (row*w + x) * 4
					want[i], want[i+1], want[i+2], want[i+3] = byte(r>>8), byte(g>>8), byte(b>>8), byte(a>>8)
				}
			}
			out := readAll(dst)
			dst.Dispose()
			return compare("set_values", "Set 寫入 color.RGBA 與 color.NRGBA", "寫入 clr.RGBA()>>8（NRGBA 先預乘）", w, h, out, want, map[string]any{"rows": []string{"RGBA a=0x80", "RGBA a=0xff", "NRGBA a=0x80", "NRGBA a=x"}})
		},
		func() semanticsResult {
			w, h := 16, 16
			pal := make(color.Palette, 256)
			for i := range pal {
				switch i % 3 {
				case 0:
					pal[i] = color.RGBA{uint8(i), uint8(i / 2), uint8(255 - i), 255}
				case 1:
					pal[i] = color.NRGBA{uint8(i), uint8(255 - i), uint8(i / 3), uint8(i)}
				default:
					pal[i] = color.RGBA{uint8(i / 2), uint8(i / 3), uint8(i / 4), uint8(i)}
				}
			}
			img := image.NewPaletted(image.Rect(0, 0, w, h), pal)
			for k := range img.Pix {
				img.Pix[k] = uint8(k)
			}
			e := ebiten.NewImageFromImage(img)
			out := readAll(e)
			e.Dispose()
			want := make([]byte, w*h*4)
			for k := 0; k < w*h; k++ {
				c := color.RGBAModel.Convert(pal[k]).(color.RGBA)
				want[k*4], want[k*4+1], want[k*4+2], want[k*4+3] = c.R, c.G, c.B, c.A
			}
			return compare("new_from_paletted", "NewImageFromImage(*image.Paletted)", "每像素 color.RGBAModel.Convert(palette[index])", w, h, out, want, map[string]any{"palette": "i%3==0: RGBA(i,i/2,255-i,255)；1: NRGBA(i,255-i,i/3,i)；2: RGBA(i/2,i/3,i/4,i)；Pix[k]=k"})
		},
		func() semanticsResult {
			w, h := 256, 256
			img := image.NewNRGBA(image.Rect(0, 0, w, h))
			want := make([]byte, w*h*4)
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					i := (y*w + x) * 4
					s := []uint8{uint8(x), uint8(255 - x), uint8((x * 7) & 255), uint8(y)}
					copy(img.Pix[i:i+4], s)
					sa := uint32(s[3]) * 0x101
					want[i] = uint8(uint32(s[0]) * sa / 0xff >> 8)
					want[i+1] = uint8(uint32(s[1]) * sa / 0xff >> 8)
					want[i+2] = uint8(uint32(s[2]) * sa / 0xff >> 8)
					want[i+3] = uint8(sa >> 8)
				}
			}
			e := ebiten.NewImageFromImage(img)
			out := readAll(e)
			e.Dispose()
			return compare("new_from_nrgba", "NewImageFromImage(*image.NRGBA)（PNG 解碼結果）", "image/draw drawNRGBASrc 預乘：uint8((v*a*0x101/0xff)>>8)", w, h, out, want, map[string]any{"pixel": "(x, 255-x, 7x&255, y)"})
		},
		func() semanticsResult {
			w, h := 256, 256
			pix := make([]byte, w*h*4)
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					i := (y*w + x) * 4
					pix[i], pix[i+1], pix[i+2], pix[i+3] = byte(x), byte(y), byte(x^y), byte(x+y)
				}
			}
			e := ebiten.NewImage(w, h)
			e.WritePixels(pix)
			out := readAll(e)
			e.Dispose()
			return compare("write_read_roundtrip", "WritePixels 後 ReadPixels（含 R/G/B 大於 A）", "原樣讀回", w, h, out, pix, nil)
		},
		func() semanticsResult {
			w, h := 256, 256
			pix := make([]byte, w*h*4)
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					i := (y*w + x) * 4
					pix[i], pix[i+1], pix[i+2], pix[i+3] = byte(x*y/255), byte((255-x)*y/255), byte(y/2), byte(y)
				}
			}
			dst := ebiten.NewImage(w, h)
			dst.WritePixels(pix)
			c := color.RGBA{0x40, 0x80, 0xff, 0x66}
			dst.DrawImage(newFilled(w, h, c), nil)
			out := readAll(dst)
			dst.Dispose()
			want := make([]byte, w*h*4)
			s := []int{int(c.R), int(c.G), int(c.B), int(c.A)}
			for i := range want {
				want[i] = byte(clamp8(s[i%4] + mul255(int(pix[i])*(255-int(c.A)))))
			}
			return compare("over_translucent_dst", "source-over 疊到半透明底（四個通道含 alpha）", "每通道 min(255, s+mul255(d*(255-a)))", w, h, out, want, map[string]any{"dst": "(xy/255, (255-x)y/255, y/2, y)", "fill": []int{0x40, 0x80, 0xff, 0x66}})
		},
	}
}

func ebitenModuleVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/hajimehoshi/ebiten/v2" {
				return dep.Version
			}
		}
	}
	return ""
}

// samplingSteps 回傳階段 1 實驗的步驟：每個取樣實驗、每個語意實驗各一步，最後一步寫 JSON。
func samplingSteps(out string) []func() error {
	var samp []samplingResult
	var sem []semanticsResult
	steps := []func() error{}
	for _, c := range samplingCases() {
		c := c
		steps = append(steps, func() error { samp = append(samp, runSamplingCase(c)); return nil })
	}
	for _, f := range semanticsCases() {
		f := f
		steps = append(steps, func() error { sem = append(sem, f()); return nil })
	}
	steps = append(steps, func() error {
		// 全部取樣實驗都逐值相同、且讀回可拆成兩軸的規則。
		common := []string{}
		for i, r := range axisRules {
			ok := true
			for _, s := range samp {
				if !s.Separable || s.CandX[i].Mismatches != 0 || s.CandY[i].Mismatches != 0 {
					ok = false
				}
			}
			if ok {
				common = append(common, r.name)
			}
		}
		nonSep := []string{}
		for _, s := range samp {
			if !s.Separable {
				nonSep = append(nonSep, s.Name)
			}
			fmt.Printf("SAMPLING %s separable=%v exact=%v\n", s.Name, s.Separable, s.ExactBoth)
		}
		failed := []string{}
		for _, s := range sem {
			if s.Mismatches != 0 {
				failed = append(failed, s.Name)
			}
			fmt.Printf("SEMANTICS %s mismatches=%d/%d\n", s.Name, s.Mismatches, s.Vectors)
		}
		fmt.Printf("SAMPLING 全部案例逐值相同：%v；不可拆成兩軸：%v\n", common, nonSep)
		fmt.Printf("SEMANTICS 與預期不同：%v\n", failed)
		raw, err := json.MarshalIndent(map[string]any{
			"schema":           "fd2_rust_r5_ebiten_sampling/1",
			"plan":             "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md 階段 1",
			"producer":         "remake/cmd/fd2-r5-ebiten-rounding -sampling-out",
			"go_version":       runtime.Version(),
			"ebiten_version":   ebitenModuleVersion(),
			"target":           []int{sampW, sampH},
			"source_encoding":  "來源像素 (x,y)：R=x&255、G=y&255、B=0x80|((x>>8)<<3)|(y>>8)、A=255；目標起始全透明",
			"sampling_common":  common,
			"sampling_nonsep":  nonSep,
			"semantics_failed": failed,
			"sampling":         samp,
			"semantics":        sem,
		}, "", " ")
		if err != nil {
			return err
		}
		return os.WriteFile(out, append(raw, '\n'), 0o644)
	})
	return steps
}
