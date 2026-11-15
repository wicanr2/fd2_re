// fd2-r5-ebiten-rounding 實測 Ebiten v2.6.6 在目前 GPU 驅動（Docker／Xvfb 下為 Mesa llvmpipe）
// 上的半透明 Fill＋DrawImage 與 ColorScale 取整，供 R5 的 CPU 合成選定逐值相同的公式
// （docs/goal/121-goal-rust-r5-original-art-shell-20261115.md 階段 0）。
//
// Go 版凍結；本工具只呼叫 Ebiten 公開 API，不碰任何遊戲程式。所有輸入由下列固定清單決定：
//   - fill_over：640×400 以外的 256×256 離屏影像。底圖 R=x、G=y、B=(3x+y)&255、A=255，
//     以 Fill 填好的同尺寸色塊 DrawImage（預設 source-over）疊上；色塊色取自原版模式程式的
//     半透明 Fill 常數（main.go、title.go、ending_preview.go），含 R/G/B 大於 A 的非預乘值。
//   - alpha_sweep：第 y 列底圖灰階 x、色塊 alpha=y；黑色淡出 (0,0,0,a)、白閃 (a,a,a,a)。
//   - color_scale：來源 R=x、G=y、B=x^y、A=255，ColorScale.Scale(sr,sg,sb,1) 畫到不透明黑底；
//     係數取 main.go:11545 的 0.55,0.55,0.6 與 title.go 的 publisher 亮度、scroll 淡入值。
//   - scale_with_color：來源第 x 欄為 (x,x,x,x)（白字的覆蓋率），底圖灰階 y，
//     ColorScale.ScaleWithColor(clr)（font.go:188 的 TTF 文字路徑）。
//
// 讀回值與每個候選公式逐值比對，輸出不符數與前幾筆不符；完全相同的公式列在 exact。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image/color"
	"math"
	"os"
	"runtime"
	"runtime/debug"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

const n = 256

type mismatch struct {
	Inputs []int `json:"inputs"`
	Actual int   `json:"actual"`
	Want   int   `json:"formula"`
}

type candidateResult struct {
	Name       string     `json:"name"`
	Mismatches int        `json:"mismatches"`
	First      []mismatch `json:"first_mismatches,omitempty"`
}

type experiment struct {
	Name        string            `json:"name"`
	Kind        string            `json:"kind"`
	Params      map[string]any    `json:"params"`
	InputsDoc   string            `json:"inputs"`
	Vectors     int               `json:"vectors"`
	RGBASHA256  string            `json:"actual_rgba_sha256"`
	Exact       []string          `json:"exact"`
	Candidates  []candidateResult `json:"candidates"`
	ActualFirst [][]int           `json:"actual_rows_sample"`
}

// 單一通道的候選公式。source-over：s=色塊通道、d=底圖通道、a=色塊 alpha（皆 0..255）。
type overFormula struct {
	name string
	f    func(s, d, a int) int
}

func clamp8(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

func roundHalfUp64(x float64) int { return int(math.Floor(x + 0.5)) }
func roundHalfUp32(x float32) int { return int(float32(math.Floor(float64(x + 0.5)))) }

// llvmpipeMul255 是 Mesa llvmpipe 8 位元混色的乘法近似：x/255 ≈ (x + (x>>8) + 0x80) >> 8。
func llvmpipeMul255(x int) int { return (x + (x >> 8) + 0x80) >> 8 }

// llvmpipeUnormToFloat：取樣 8 位元貼圖時 v → float32(v/256)*float32(256/255)。
func llvmpipeUnormToFloat(v int) float32 {
	return float32(float32(v)/256) * float32(256.0/255.0)
}

// llvmpipeFloatToUnorm：著色器輸出轉回 8 位元，x*(255/256) 加 32768 後取 float32 尾數低 8 位
// （float32 加法以最近偶數取整），x≤0 為 0、x≥1 為 255。
func llvmpipeFloatToUnorm(x float32) int {
	if !(x > 0) {
		return 0
	}
	if x >= 1 {
		return 255
	}
	// 明寫 float32 轉換，禁止編譯器把乘加融合成 FMA（Go 規格允許跨敘述融合）。
	y := float32(x * float32(255.0/256.0))
	z := float32(y + 32768)
	return int(math.Float32bits(z) & 0xff)
}

var overFormulas = []overFormula{
	{"llvmpipe 整數 s+mul255(d*(255-a))，mul255(x)=(x+(x>>8)+0x80)>>8，上限 255", func(s, d, a int) int {
		return clamp8(s + llvmpipeMul255(d*(255-a)))
	}},
	{"float64 s+d*(1-a) 四捨五入", func(s, d, a int) int {
		return clamp8(roundHalfUp64((float64(s)/255 + float64(d)/255*(1-float64(a)/255)) * 255))
	}},
	{"float32 s+d*(1-a) 四捨五入", func(s, d, a int) int {
		v := float32(s)/255 + float32(d)/255*(1-float32(a)/255)
		return clamp8(roundHalfUp32(v * 255))
	}},
	{"float32 s+d*(1-a) 截斷", func(s, d, a int) int {
		v := float32(s)/255 + float32(d)/255*(1-float32(a)/255)
		return clamp8(int(v * 255))
	}},
	{"整數 s+(d*(255-a)+127)/255", func(s, d, a int) int { return clamp8(s + (d*(255-a)+127)/255) }},
	{"整數 s+d*(255-a)/255 截斷", func(s, d, a int) int { return clamp8(s + d*(255-a)/255) }},
	{"整數 s+(d*(256-a))>>8", func(s, d, a int) int { return clamp8(s + (d*(256-a))>>8) }},
	{"整數 s+(d*(255-a)+128+((d*(255-a)+128)>>8))>>8", func(s, d, a int) int {
		x := d*(255-a) + 128
		return clamp8(s + (x+(x>>8))>>8)
	}},
	{"整數 s+(d*(255-a)+255)>>8", func(s, d, a int) int { return clamp8(s + (d*(255-a)+255)>>8) }},
}

type scaleFormula struct {
	name string
	f    func(v int, s float32) int
}

// keff 是 Ebiten v2.6.6 ColorScale 實際帶進著色器的係數：ColorScale 存 scale-1，
// Scale(s) 算 (0+1)*s-1，apply 再 +1，兩步都是 float32。
func keff(s float32) float32 {
	r1 := float32(1)*s - 1
	return r1 + 1
}

var scaleFormulas = []scaleFormula{
	{"llvmpipe float→unorm(unorm→float(v)*keff(s))", func(v int, s float32) int {
		return llvmpipeFloatToUnorm(llvmpipeUnormToFloat(v) * keff(s))
	}},
	{"llvmpipe float→unorm(unorm→float(v)*s)（不經 keff）", func(v int, s float32) int {
		return llvmpipeFloatToUnorm(llvmpipeUnormToFloat(v) * s)
	}},
	{"float→unorm 用 llvmpipe、unorm→float 用 float32(v)/255", func(v int, s float32) int {
		return llvmpipeFloatToUnorm(float32(v) / 255 * keff(s))
	}},
	{"float32 v/255*keff(s)*255 四捨五入", func(v int, s float32) int {
		return clamp8(roundHalfUp32(float32(v) / 255 * keff(s) * 255))
	}},
	{"float64(v)*float64(keff(s)) 四捨五入", func(v int, s float32) int { return clamp8(roundHalfUp64(float64(v) * float64(keff(s)))) }},
	{"float32 v/255*keff(s)*255 截斷", func(v int, s float32) int { return clamp8(int(float32(v) / 255 * keff(s) * 255)) }},
	{"float32 v/255*s*255 四捨五入", func(v int, s float32) int { return clamp8(roundHalfUp32(float32(v) / 255 * s * 255)) }},
	{"float32 v*s 四捨五入", func(v int, s float32) int { return clamp8(roundHalfUp32(float32(v) * s)) }},
	{"float64(v)*float64(float32 s) 四捨五入", func(v int, s float32) int { return clamp8(roundHalfUp64(float64(v) * float64(s))) }},
	{"float64 v*十進位 s 四捨五入", func(v int, s float32) int {
		return clamp8(roundHalfUp64(float64(v) * shortDecimal(s)))
	}},
	{"float32 v/255*s*255 截斷", func(v int, s float32) int { return clamp8(int(float32(v) / 255 * s * 255)) }},
	{"float64 v*s 截斷", func(v int, s float32) int { return clamp8(int(float64(v) * float64(s))) }},
	{"四捨五入到偶數 float64(v)*float64(s)", func(v int, s float32) int { return clamp8(int(math.RoundToEven(float64(v) * float64(s)))) }},
	{"整數 (v*round(s*255)+127)/255", func(v int, s float32) int {
		k := roundHalfUp64(float64(s) * 255)
		return clamp8((v*k + 127) / 255)
	}},
}

// shortDecimal 把 float32 係數還原成最短十進位表示（0.55 → 0.55）的 float64。
func shortDecimal(s float32) float64 {
	var v float64
	fmt.Sscanf(fmt.Sprint(s), "%g", &v)
	return v
}

func newFilled(w, h int, c color.RGBA) *ebiten.Image {
	img := ebiten.NewImage(w, h)
	img.Fill(c)
	return img
}

func read(img *ebiten.Image) []byte {
	b := img.Bounds()
	pix := make([]byte, 4*b.Dx()*b.Dy())
	img.ReadPixels(pix)
	return pix
}

// rawDir 非空時，每個實驗的讀回 RGBA 另存成 <rawDir>/<實驗名>.rgba（256 寬），供離線分析。
var rawDir string

func dumpRaw(name string, pix []byte) {
	if rawDir == "" {
		return
	}
	if err := os.WriteFile(rawDir+"/"+name+".rgba", pix, 0o644); err != nil {
		panic(err)
	}
}

func sha(pix []byte) string { s := sha256.Sum256(pix); return hex.EncodeToString(s[:]) }

func rowsSample(pix []byte, ch int) [][]int {
	rows := [][]int{}
	for _, y := range []int{0, 1, 64, 102, 127, 128, 200, 255} {
		row := make([]int, n)
		for x := 0; x < n; x++ {
			row[x] = int(pix[(y*n+x)*4+ch])
		}
		rows = append(rows, row)
	}
	return rows
}

type sample struct {
	inputs []int
	actual int
}

func scoreOver(samples []sample, s func(sample) (int, int, int)) ([]string, []candidateResult) {
	exact := []string{}
	results := []candidateResult{}
	for _, f := range overFormulas {
		r := candidateResult{Name: f.name}
		for _, sm := range samples {
			sv, dv, av := s(sm)
			if want := f.f(sv, dv, av); want != sm.actual {
				r.Mismatches++
				if len(r.First) < 8 {
					r.First = append(r.First, mismatch{Inputs: sm.inputs, Actual: sm.actual, Want: want})
				}
			}
		}
		if r.Mismatches == 0 {
			exact = append(exact, f.name)
		}
		results = append(results, r)
	}
	return exact, results
}

func scoreScale(samples []sample, s func(sample) (int, float32)) ([]string, []candidateResult) {
	exact := []string{}
	results := []candidateResult{}
	for _, f := range scaleFormulas {
		r := candidateResult{Name: f.name}
		for _, sm := range samples {
			v, k := s(sm)
			if want := f.f(v, k); want != sm.actual {
				r.Mismatches++
				if len(r.First) < 8 {
					r.First = append(r.First, mismatch{Inputs: sm.inputs, Actual: sm.actual, Want: want})
				}
			}
		}
		if r.Mismatches == 0 {
			exact = append(exact, f.name)
		}
		results = append(results, r)
	}
	return exact, results
}

func dstPattern() *ebiten.Image {
	pix := make([]byte, n*n*4)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := (y*n + x) * 4
			pix[i], pix[i+1], pix[i+2], pix[i+3] = byte(x), byte(y), byte((3*x+y)&255), 255
		}
	}
	img := ebiten.NewImage(n, n)
	img.WritePixels(pix)
	return img
}

// fillOver：固定色塊疊到三通道各自掃過 0..255 的底圖。
func fillOver(name string, c color.RGBA, source string) experiment {
	dst := dstPattern()
	dst.DrawImage(newFilled(n, n, c), &ebiten.DrawImageOptions{})
	pix := read(dst)
	dumpRaw(name, pix)
	src := [3]int{int(c.R), int(c.G), int(c.B)}
	samples := []sample{}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			d := [3]int{x, y, (3*x + y) & 255}
			for ch := 0; ch < 3; ch++ {
				samples = append(samples, sample{inputs: []int{ch, src[ch], d[ch], int(c.A)}, actual: int(pix[(y*n+x)*4+ch])})
			}
			if a := int(pix[(y*n+x)*4+3]); a != 255 {
				panic(fmt.Sprintf("%s: 不透明底圖疊色後 alpha=%d", name, a))
			}
		}
	}
	exact, cands := scoreOver(samples, func(s sample) (int, int, int) { return s.inputs[1], s.inputs[2], s.inputs[3] })
	return experiment{Name: name, Kind: "fill_over", Params: map[string]any{"fill_rgba": []int{int(c.R), int(c.G), int(c.B), int(c.A)}, "source": source},
		InputsDoc: "inputs=[通道 0R/1G/2B, 色塊通道值 s, 底圖通道值 d, 色塊 alpha a]；底圖 R=x、G=y、B=(3x+y)&255、A=255",
		Vectors:   len(samples), RGBASHA256: sha(pix), Exact: exact, Candidates: cands, ActualFirst: rowsSample(pix, 0)}
}

// alphaSweep：第 y 列底圖灰階 x、色塊 alpha=y，色塊 RGB 由 rgb(a) 給出。
func alphaSweep(name string, rgb func(a int) [3]int, doc string) experiment {
	dst := ebiten.NewImage(n, n)
	pix := make([]byte, n*n*4)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := (y*n + x) * 4
			pix[i], pix[i+1], pix[i+2], pix[i+3] = byte(x), byte(x), byte(x), 255
		}
	}
	dst.WritePixels(pix)
	for a := 0; a < n; a++ {
		c := rgb(a)
		row := newFilled(n, 1, color.RGBA{uint8(c[0]), uint8(c[1]), uint8(c[2]), uint8(a)})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, float64(a))
		dst.DrawImage(row, op)
	}
	out := read(dst)
	dumpRaw(name, out)
	samples := []sample{}
	for a := 0; a < n; a++ {
		c := rgb(a)
		for x := 0; x < n; x++ {
			for ch := 0; ch < 3; ch++ {
				samples = append(samples, sample{inputs: []int{ch, c[ch], x, a}, actual: int(out[(a*n+x)*4+ch])})
			}
		}
	}
	exact, cands := scoreOver(samples, func(s sample) (int, int, int) { return s.inputs[1], s.inputs[2], s.inputs[3] })
	return experiment{Name: name, Kind: "alpha_sweep", Params: map[string]any{"fill": doc},
		InputsDoc: "inputs=[通道, 色塊通道值 s, 底圖灰階 d, 色塊 alpha a]；第 a 列是 alpha=a 的 256×1 色塊",
		Vectors:   len(samples), RGBASHA256: sha(out), Exact: exact, Candidates: cands, ActualFirst: rowsSample(out, 0)}
}

func colorScale(name string, sr, sg, sb float32, source string) experiment {
	src := ebiten.NewImage(n, n)
	pix := make([]byte, n*n*4)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := (y*n + x) * 4
			pix[i], pix[i+1], pix[i+2], pix[i+3] = byte(x), byte(y), byte(x^y), 255
		}
	}
	src.WritePixels(pix)
	dst := newFilled(n, n, color.RGBA{0, 0, 0, 255})
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.Scale(sr, sg, sb, 1)
	dst.DrawImage(src, op)
	out := read(dst)
	dumpRaw(name, out)
	k := [3]float32{sr, sg, sb}
	samples := []sample{}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			v := [3]int{x, y, x ^ y}
			for ch := 0; ch < 3; ch++ {
				samples = append(samples, sample{inputs: []int{ch, v[ch]}, actual: int(out[(y*n+x)*4+ch])})
			}
		}
	}
	exact, cands := scoreScale(samples, func(s sample) (int, float32) { return s.inputs[1], k[s.inputs[0]] })
	return experiment{Name: name, Kind: "color_scale", Params: map[string]any{"scale_rgb": []float32{sr, sg, sb}, "source": source},
		InputsDoc: "inputs=[通道, 來源通道值 v]；來源 R=x、G=y、B=x^y、A=255，畫到不透明黑底",
		Vectors:   len(samples), RGBASHA256: sha(out), Exact: exact, Candidates: cands, ActualFirst: rowsSample(out, 0)}
}

// scaleWithColor：白字覆蓋率 (v,v,v,v) 乘 ScaleWithColor(clr) 後 source-over 到灰階底。
// 候選：先以 scale 公式算出預乘來源 (s, a)，再以 source-over 公式疊合。
func scaleWithColor(name string, clr color.RGBA) experiment {
	src := ebiten.NewImage(n, 1)
	pix := make([]byte, n*4)
	for x := 0; x < n; x++ {
		pix[x*4], pix[x*4+1], pix[x*4+2], pix[x*4+3] = byte(x), byte(x), byte(x), byte(x)
	}
	src.WritePixels(pix)
	dst := ebiten.NewImage(n, n)
	bg := make([]byte, n*n*4)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := (y*n + x) * 4
			bg[i], bg[i+1], bg[i+2], bg[i+3] = byte(y), byte(y), byte(y), 255
		}
	}
	dst.WritePixels(bg)
	for y := 0; y < n; y++ {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, float64(y))
		op.ColorScale.ScaleWithColor(clr)
		dst.DrawImage(src, op)
	}
	out := read(dst)
	dumpRaw(name, out)
	// ScaleWithColor 的係數（Ebiten v2.6.6 colorscale.go:110）：直接用預乘的 RGBA()/0xffff，
	// 再經 keff 的 scale-1 存法。
	ca := keff(float32(uint32(clr.A)*0x101) / 0xffff)
	var k [3]float32
	for ch, v := range []uint8{clr.R, clr.G, clr.B} {
		k[ch] = keff(float32(uint32(v)*0x101) / 0xffff)
	}
	type combo struct {
		name string
		f    func(v, d, ch int) int
	}
	combos := []combo{
		{"llvmpipe：來源 rgb=f2u(u2f(v)*k)、a=f2u(u2f(v)*ca)，再 s+mul255(d*(255-a))", func(v, d, ch int) int {
			sv := llvmpipeFloatToUnorm(llvmpipeUnormToFloat(v) * k[ch])
			av := llvmpipeFloatToUnorm(llvmpipeUnormToFloat(v) * ca)
			return clamp8(sv + llvmpipeMul255(d*(255-av)))
		}},
		{"float32 全程（src=v/255*k，src-over 不中途取整）四捨五入", func(v, d, ch int) int {
			s := float32(v) / 255 * k[ch]
			a := float32(v) / 255 * ca
			return clamp8(roundHalfUp32((s + float32(d)/255*(1-a)) * 255))
		}},
		{"來源先取整到 8 位元再 src-over（整數 +127）", func(v, d, ch int) int {
			s := roundHalfUp32(float32(v) / 255 * k[ch] * 255)
			a := roundHalfUp32(float32(v) / 255 * ca * 255)
			return clamp8(s + (d*(255-a)+127)/255)
		}},
	}
	exact := []string{}
	results := []candidateResult{}
	total := 0
	for _, c := range combos {
		r := candidateResult{Name: c.name}
		total = 0
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				for ch := 0; ch < 3; ch++ {
					total++
					got := int(out[(y*n+x)*4+ch])
					if want := c.f(x, y, ch); want != got {
						r.Mismatches++
						if len(r.First) < 8 {
							r.First = append(r.First, mismatch{Inputs: []int{ch, x, y}, Actual: got, Want: want})
						}
					}
				}
			}
		}
		if r.Mismatches == 0 {
			exact = append(exact, c.name)
		}
		results = append(results, r)
	}
	return experiment{Name: name, Kind: "scale_with_color", Params: map[string]any{"color_rgba": []int{int(clr.R), int(clr.G), int(clr.B), int(clr.A)}},
		InputsDoc: "inputs=[通道, 覆蓋率 v（來源 (v,v,v,v)）, 底圖灰階 d]",
		Vectors:   total, RGBASHA256: sha(out), Exact: exact, Candidates: results, ActualFirst: rowsSample(out, 0)}
}

type prober struct {
	out  string
	done bool
	err  error
}

func (p *prober) Layout(int, int) (int, int) { return 640, 400 }
func (p *prober) Draw(*ebiten.Image)         {}
func (p *prober) Update() error {
	if p.done {
		return ebiten.Termination
	}
	p.done = true
	p.err = p.run()
	return ebiten.Termination
}

func (p *prober) run() error {
	exps := []experiment{}
	fills := []struct {
		c      color.RGBA
		source string
	}{
		{color.RGBA{0x40, 0x80, 0xff, 0x66}, "main.go:9844 移動範圍格"},
		{color.RGBA{0xa0, 0x50, 0xe0, 0x5c}, "main.go:9852"},
		{color.RGBA{0xe0, 0x30, 0x30, 0x5c}, "main.go:9870 攻擊範圍格"},
		{color.RGBA{0x2c, 0x44, 0x84, 0xf2}, "main.go:10018 舊對話框底"},
		{color.RGBA{0xff, 0xa8, 0x20, 0x50}, "main.go:10252"},
		{color.RGBA{0x10, 0x1c, 0x40, 0xee}, "main.go:10543／10593"},
		{color.RGBA{0x10, 0x1c, 0x40, 0xe8}, "main.go:10645、ending_preview.go:834"},
		{color.RGBA{0x08, 0x14, 0x30, 0xf0}, "title.go:852"},
		{color.RGBA{0x30, 0x60, 0x90, 0x80}, "合成的合法預乘色（對照）"},
	}
	for i, f := range fills {
		exps = append(exps, fillOver(fmt.Sprintf("fill_over_%02d", i), f.c, f.source))
	}
	exps = append(exps,
		alphaSweep("alpha_sweep_black", func(int) [3]int { return [3]int{} }, "(0,0,0,a)：main.go:10180 舊 fade 疊層"),
		alphaSweep("alpha_sweep_white", func(a int) [3]int { return [3]int{a, a, a} }, "(a,a,a,a)：title.go:812 白閃"),
		alphaSweep("alpha_sweep_premul", func(a int) [3]int { return [3]int{a, a / 2, a / 3} }, "(a,a/2,a/3,a)：合法預乘色"),
	)
	scales := []struct {
		r, g, b float32
		source  string
	}{{0.55, 0.55, 0.6, "main.go:11545 已行動單位變暗"}}
	for tick := 0; tick < 8; tick++ {
		s := float32(tick+1) / 8
		scales = append(scales, struct {
			r, g, b float32
			source  string
		}{s, s, s, fmt.Sprintf("title.go titlePublisherBrightness(%d)", tick)})
	}
	for _, scrollY := range []float64{535, 520, 500, 495, 480} {
		fade := float32((535 - scrollY) / 60)
		if fade >= 1 {
			continue
		}
		scales = append(scales, struct {
			r, g, b float32
			source  string
		}{fade, fade, fade, fmt.Sprintf("title.go:790 scroll 淡入 scrollY=%g", scrollY)})
	}
	for i, s := range scales {
		exps = append(exps, colorScale(fmt.Sprintf("color_scale_%02d", i), s.r, s.g, s.b, s.source))
	}
	for i, c := range []color.RGBA{{0xff, 0xff, 0xff, 0xff}, {0xff, 0xff, 0x00, 0xff}, {0xf0, 0xe0, 0xa0, 0xff}, {0x80, 0x80, 0x80, 0x80}} {
		exps = append(exps, scaleWithColor(fmt.Sprintf("scale_with_color_%02d", i), c))
	}
	summary := map[string][]string{}
	for _, e := range exps {
		key := e.Kind
		if len(e.Exact) == 0 {
			summary[key] = append(summary[key], e.Name+"：無")
		}
	}
	ebitenVersion := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/hajimehoshi/ebiten/v2" {
				ebitenVersion = dep.Version
			}
		}
	}
	// 每種實驗：所有案例都逐值相同的公式。
	common := map[string][]string{}
	kinds := []string{}
	for _, e := range exps {
		if _, ok := common[e.Kind]; !ok {
			kinds = append(kinds, e.Kind)
			common[e.Kind] = append([]string{}, e.Exact...)
			continue
		}
		keep := []string{}
		for _, name := range common[e.Kind] {
			for _, x := range e.Exact {
				if x == name {
					keep = append(keep, name)
				}
			}
		}
		common[e.Kind] = keep
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		fmt.Printf("ROUNDING %s 全部案例逐值相同：%v\n", k, common[k])
	}
	for _, e := range exps {
		fmt.Printf("CASE %s %s vectors=%d exact=%v\n", e.Name, e.Kind, e.Vectors, e.Exact)
	}
	raw, err := json.MarshalIndent(map[string]any{
		"schema":         "fd2_rust_r5_ebiten_rounding/1",
		"plan":           "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md 階段 0",
		"producer":       "remake/cmd/fd2-r5-ebiten-rounding",
		"go_version":     runtime.Version(),
		"ebiten_version": ebitenVersion,
		"note":           "ReadPixels 讀回的是預乘 alpha 值；底圖不透明，讀回即最終 RGB。",
		"common_exact":   common,
		"cases_no_exact": summary,
		"experiments":    exps,
	}, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.out, append(raw, '\n'), 0o644)
}

func main() {
	out := flag.String("out", "", "輸出 JSON")
	flag.StringVar(&rawDir, "raw", "", "另存每個實驗讀回 RGBA 的目錄（可省略）")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "需要 -out")
		os.Exit(2)
	}
	ebiten.SetWindowSize(640, 400)
	p := &prober{out: *out}
	if err := ebiten.RunGame(p); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if p.err != nil {
		fmt.Fprintln(os.Stderr, p.err)
		os.Exit(1)
	}
}
