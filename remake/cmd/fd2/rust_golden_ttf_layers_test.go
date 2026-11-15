package main

// R5 階段 2 r5f 組（docs/goal/121）的 Go 局部黃金工具，只輸出資料。未設 FD2_GOLDEN_TTF_LAYERS_OUT／
// FD2_GOLDEN_TTF_METRICS_OUT 時略過，不改任何遊戲行為；由 tools/rust_r5f_ttf_layers.sh 在 Docker／Xvfb
// 內呼叫。
//
// 121 定案 TTF 只比文字位置與外框，像素差異寫成 KNOWN_DIFFERENCES。兩個測試：
//
//   - TestRustGoldenTTFLayers：照 TestRustGoldenFrames 的 chapter 模式原樣跑第一章受控重播（同一個
//     ch01FrameObserver 掛點、60 Hz 虛擬時鐘與 rand.Seed(4)），每個 pump 幀不呼叫整個 Draw，只照 Draw 的
//     分支順序（goldenMapLayersBranch）在走到尾段時執行 Draw 的 TTF 段（main.go:10136..10161 的逐字複本，
//     g.font.Draw 換成 r.fontDraw 以記錄每次繪字），畫到 Clear 的 640×400 畫布。比對鍵是每次繪字的文字、
//     座標、倍率、像素尺寸、ascent、顏色與 text.BoundString 外框（平移到畫面座標）；畫素統計（非透明像素外框、
//     覆蓋率、與外框重疊的裁切 RGBA）另存 renders，不進比對鍵。
//   - TestRustGoldenTTFMetrics：不跑遊戲，對 remake/assets/locales 全部字串的字元與 ASCII 列印 Face.Metrics、
//     GlyphBounds、Kern、GlyphIndex 與每個字串的 text.BoundString，供 Rust 的 TextMeasure 逐筆比對。
//
// Draw 開頭的三個本幀旗標照 Draw 清掉；選中單位中文名與舊對話框兩段在開場與第一章沒有走到（階段 0 覆蓋率
// 0 次），不在複本內。

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/sfnt"
)

type ttfLayerDraw struct {
	text   string
	x, y   float64
	scale  float64
	px     int
	asc    float64
	clr    color.RGBA
	bounds image.Rectangle // 畫面座標
}

type ttfLayerRender struct {
	Captures   int    `json:"captures"`
	Nonzero    int    `json:"nonzero"`
	Opaque     int    `json:"opaque"`
	AlphaSum   int    `json:"alpha_sum"`
	InkBox     [4]int `json:"ink_box"`
	OutsideBox int    `json:"outside_bounds"`
	RGBA       string `json:"rgba_sha256"`
	Crop       [4]int `json:"crop"`
	CropRGBA   string `json:"crop_rgba_base64"`
	Conflicts  int    `json:"conflicts"`
}

type ttfLayerRecorder struct {
	t        *testing.T
	probe    *goldenFrameProbe
	clock    goldenFrameRecorder // 只借用 pinClock（同一個虛擬時鐘）
	out      string
	screen   *ebiten.Image
	pix      []byte
	rows     []mapLayerRow
	seq      []byte
	capture  int
	branches map[string]int
	stats    map[string]int
	draws    []ttfLayerDraw
	renders  map[string]*ttfLayerRender
}

// fontDraw 取代複本裡的 g.font.Draw：先記下這次繪字與 BoundString 外框，再呼叫原本的 Font.Draw。
func (r *ttfLayerRecorder) fontDraw(f *Font, screen *ebiten.Image, s string, x, y, scale float64, clr color.RGBA) float64 {
	px := int(math.Round(f.base * f.effectiveScale(scale)))
	face, asc := f.faceFor(px)
	d := ttfLayerDraw{text: s, x: x, y: y, scale: scale, px: px, asc: asc, clr: clr}
	if face != nil {
		d.bounds = text.BoundString(face, s).Add(image.Pt(int(math.Round(x)), int(math.Round(y)+asc)))
	}
	r.draws = append(r.draws, d)
	return f.Draw(screen, s, x, y, scale, clr)
}

// ttfSection 是 Draw 尾段 `if g.font != nil {` 裡的 TTF 戰場短訊與勝負字樣。
func (r *ttfLayerRecorder) ttfSection(g *Game, screen *ebiten.Image) {
	if g.font != nil {
		// ---- 逐字複本開始：main.go:10136..10161（g.font.Draw( 換成 r.fontDraw(g.font, ） ----
		if g.shouldDrawTransientBattleMessage() { // 攻擊等短訊(無對話框時)
			r.fontDraw(g.font, screen, g.msg, 8, float64(logicalH)-30, 1.2, color.RGBA{0xff, 0xf0, 0xb4, 0xff})
		}
		if g.result != "" { // 勝負(中央大字)
			t, ok := g.localeMessage("battle.result.win")
			if !ok {
				return
			}
			c := color.RGBA{0xff, 0xdc, 0x50, 0xff}
			if g.result == "lose" {
				t, ok = g.localeMessage("battle.result.lose")
				if !ok {
					return
				}
				c = color.RGBA{0xff, 0x70, 0x70, 0xff}
			}
			r.fontDraw(g.font, screen, t, float64(logicalW)/2-78, float64(logicalH)/2-30, 3.0, c)
			if g.camp != nil {
				continueText, ok := g.localeMessage("battle.result.continue")
				if !ok {
					return
				}
				r.fontDraw(g.font, screen, continueText, float64(logicalW)/2-70, float64(logicalH)/2+36, 1.0,
					color.RGBA{0xe0, 0xe0, 0xe0, 0xff})
			}
		}
		// ---- 逐字複本結束：main.go:10136..10161 ----
	}
}

func ttfLayerDrawKey(d ttfLayerDraw) string {
	// %+q（strconv.QuoteToASCII）：非 ASCII 一律 \u 跳脫，Rust 端不必重做 Go 的 IsPrint 表（例如 U+3000）。
	return fmt.Sprintf("text=%+q;x=%.3f;y=%.3f;scale=%.3f;px=%d;asc=%d;clr=%02x%02x%02x%02x;bound=%d,%d,%d,%d",
		d.text, d.x, d.y, d.scale, d.px, int(d.asc), d.clr.R, d.clr.G, d.clr.B, d.clr.A,
		d.bounds.Min.X, d.bounds.Min.Y, d.bounds.Max.X, d.bounds.Max.Y)
}

func (r *ttfLayerRecorder) render(key string) {
	union := image.Rectangle{}
	for _, d := range r.draws {
		union = union.Union(d.bounds)
	}
	crop := union.Inset(-2).Intersect(image.Rect(0, 0, logicalW, logicalH))
	rd := &ttfLayerRender{Captures: 1, InkBox: [4]int{logicalW, logicalH, 0, 0},
		Crop: [4]int{crop.Min.X, crop.Min.Y, crop.Max.X, crop.Max.Y}}
	for y := 0; y < logicalH; y++ {
		for x := 0; x < logicalW; x++ {
			a := int(r.pix[(y*logicalW+x)*4+3])
			if a == 0 {
				continue
			}
			rd.Nonzero++
			rd.AlphaSum += a
			if a == 0xff {
				rd.Opaque++
			}
			rd.InkBox = [4]int{min(rd.InkBox[0], x), min(rd.InkBox[1], y), max(rd.InkBox[2], x+1), max(rd.InkBox[3], y+1)}
			inside := false
			for _, d := range r.draws {
				if image.Pt(x, y).In(d.bounds) {
					inside = true
				}
			}
			if !inside {
				rd.OutsideBox++
			}
		}
	}
	rd.RGBA = mapLayerSHA(r.pix)
	if prev, ok := r.renders[key]; ok {
		prev.Captures++
		if prev.RGBA != rd.RGBA {
			prev.Conflicts++
		}
		return
	}
	cropped := make([]byte, 0, crop.Dx()*crop.Dy()*4)
	for y := crop.Min.Y; y < crop.Max.Y; y++ {
		cropped = append(cropped, r.pix[(y*logicalW+crop.Min.X)*4:(y*logicalW+crop.Max.X)*4]...)
	}
	rd.CropRGBA = base64.StdEncoding.EncodeToString(cropped)
	r.renders[key] = rd
}

func (r *ttfLayerRecorder) layers(g *Game) string {
	g.nativeMapPresented = false
	g.movementPanelDraw = nil
	g.nativeInterfaceHUDVisible = false
	branch := goldenMapLayersBranch(g)
	r.branches[branch]++
	if branch != "tail" {
		return branch
	}
	if g.modernStoryPortraits != nil {
		r.t.Fatal("局部黃金只涵蓋忠實模式（modernStoryPortraits 必須是 nil）")
	}
	r.draws = r.draws[:0]
	r.screen.Clear()
	r.ttfSection(g, r.screen)
	if len(r.draws) == 0 {
		return "tail|ttf=0"
	}
	parts := make([]string, 0, len(r.draws))
	for _, d := range r.draws {
		parts = append(parts, ttfLayerDrawKey(d))
	}
	key := fmt.Sprintf("tail|ttf=%d|%s", len(r.draws), strings.Join(parts, "|"))
	r.stats["ttf_captures"]++
	r.stats[fmt.Sprintf("ttf_draws_%d", len(r.draws))]++
	if g.shouldDrawTransientBattleMessage() {
		r.stats["ttf_msg"]++
	}
	if g.result != "" {
		r.stats["ttf_result"]++
	}
	r.screen.ReadPixels(r.pix)
	r.render(key)
	return key
}

func (r *ttfLayerRecorder) observe(g *Game) {
	r.clock.pinClock(g)
	if r.capture%goldenFrameGCInterval == 0 && r.capture > 0 {
		runtime.GC()
	}
	var key string
	r.probe.run(func() { key = r.layers(g) })
	r.seq = append(r.seq, key...)
	r.seq = append(r.seq, '\n')
	index := r.capture
	r.capture++
	if n := len(r.rows); n > 0 && r.rows[n-1].Key == key {
		r.rows[n-1].Count++
		return
	}
	r.rows = append(r.rows, mapLayerRow{Capture: index, Count: 1, UpdateFrame: g.frame, Node: goldenFrameNode(g), Key: key})
}

func (r *ttfLayerRecorder) finish(fontPath string) {
	seq := sha256.Sum256(r.seq)
	manifest := map[string]any{
		"schema":           "fd2_rust_r5_ttf_layers/1",
		"plan":             "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
		"producer":         "remake/cmd/fd2/rust_golden_ttf_layers_test.go（tools/rust_r5f_ttf_layers.sh）",
		"go_version":       runtime.Version(),
		"font":             fontPath,
		"clock":            r.clock.clockMode,
		"global_rand_seed": goldenFrameGlobalRandSeed,
		"virtual_frames":   r.clock.virtualFrames,
		"captures":         r.capture,
		"changed_rows":     len(r.rows),
		"key_sequence":     "每次擷取一行比對鍵（rows[].key），串接後的 SHA-256",
		"sequence_sha256":  hex.EncodeToString(seq[:]),
		"branches":         r.branches,
		"stats":            r.stats,
		"rows":             r.rows,
		"renders":          r.renders,
	}
	raw, err := json.MarshalIndent(manifest, "", " ")
	if err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.out, "ttf-layers.json"), append(raw, '\n'), 0o644); err != nil {
		r.t.Fatal(err)
	}
	r.t.Logf("ttf layers: captures=%d rows=%d renders=%d branches=%v stats=%v sequence=%x",
		r.capture, len(r.rows), len(r.renders), r.branches, r.stats, seq)
}

// goldenTTFFontPath 回傳 loadFont 實際讀到的字型路徑（fontPaths 第一個讀得到的），沒有字型時測試失敗。
func goldenTTFFontPath(t *testing.T) string {
	for _, p := range fontPaths {
		if _, err := os.Stat(assetPath(p)); err == nil {
			return assetPath(p)
		}
	}
	t.Fatal("沒有 TTF 字型（fontPaths 都讀不到）")
	return ""
}

func TestRustGoldenTTFLayers(t *testing.T) {
	out := os.Getenv("FD2_GOLDEN_TTF_LAYERS_OUT")
	if out == "" {
		t.Skip("需要 FD2_GOLDEN_TTF_LAYERS_OUT（R5 r5f 局部黃金，只由 tools/rust_r5f_ttf_layers.sh 執行）")
	}
	parity := os.Getenv("FD2_PARITY_OUT")
	if parity == "" || os.Getenv("FD2_PARITY_CAPTURE_FRAMES") != "0" || os.Getenv("FD2_PARITY_ENTRY") != "start" ||
		os.Getenv("FD2_PARITY_CHAPTER") != "1" {
		t.Fatal("需要與 tools/rust_r4_ch01_coverage.sh 相同的 FD2_PARITY_* 設定（第一章、START、不擷取索引畫格）")
	}
	fontPath := goldenTTFFontPath(t)
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	rand.Seed(goldenFrameGlobalRandSeed) //nolint:staticcheck // 與 TestRustGoldenFrames 相同
	probe := &goldenFrameProbe{started: make(chan struct{}), stop: make(chan struct{}),
		work: make(chan func()), parked: make(chan struct{})}
	ebiten.SetVsyncEnabled(false)
	ebiten.SetTPS(ebiten.SyncWithFPS)
	runErr := make(chan error, 1)
	go func() { runErr <- ebiten.RunGame(probe) }()
	select {
	case <-probe.started:
	case err := <-runErr:
		t.Fatalf("Ebiten 未啟動：%v", err)
	case <-time.After(60 * time.Second):
		t.Fatal("Ebiten 60 秒內未進入 Update")
	}
	defer func() {
		close(probe.stop)
		select {
		case <-runErr:
		case <-time.After(30 * time.Second):
			t.Error("Ebiten 未在 30 秒內結束")
		}
	}()
	rec := &ttfLayerRecorder{t: t, probe: probe, clock: goldenFrameRecorder{t: t}, out: out,
		screen: ebiten.NewImage(logicalW, logicalH), pix: make([]byte, logicalW*logicalH*4),
		branches: map[string]int{}, stats: map[string]int{}, renders: map[string]*ttfLayerRender{}}
	saved := ch01FrameObserver
	ch01FrameObserver = rec.observe
	defer func() { ch01FrameObserver = saved }()
	TestChapterParityReplay(t)
	if t.Failed() {
		return
	}
	rec.finish(fontPath)
}

// goldenTTFStrings 收集 locales 目錄下每個 JSON 檔的全部字串葉節點（排序、去重）。
func goldenTTFStrings(t *testing.T, dir string) []string {
	seen := map[string]bool{}
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case string:
			seen[x] = true
		case []any:
			for _, e := range x {
				walk(e)
			}
		case map[string]any:
			for _, e := range x {
				walk(e)
			}
		}
	}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".json") {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return err
		}
		walk(v)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		if s != "" {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// TestRustGoldenTTFMetrics 輸出 TextMeasure 要對齊的 x/image／Ebiten 度量。
func TestRustGoldenTTFMetrics(t *testing.T) {
	out := os.Getenv("FD2_GOLDEN_TTF_METRICS_OUT")
	if out == "" {
		t.Skip("需要 FD2_GOLDEN_TTF_METRICS_OUT（R5 r5f 度量黃金，只由 tools/rust_r5f_ttf_layers.sh 執行）")
	}
	fontPath := goldenTTFFontPath(t)
	f := loadFont()
	if f == nil {
		t.Fatal("loadFont 失敗")
	}
	strs := goldenTTFStrings(t, assetPath("assets/locales"))
	runeSet := map[rune]bool{0x10ffff: true, 0xfffd: true}
	for r := rune(0x20); r < 0x7f; r++ {
		runeSet[r] = true
	}
	pairSet := map[[2]rune]bool{}
	for _, s := range strs {
		prev := rune(-1)
		for _, r := range s {
			runeSet[r] = true
			if prev >= 0 {
				pairSet[[2]rune{prev, r}] = true
			}
			prev = r
		}
	}
	for a := rune(0x20); a < 0x7f; a++ {
		for b := rune(0x20); b < 0x7f; b++ {
			pairSet[[2]rune{a, b}] = true
		}
	}
	runes := make([]rune, 0, len(runeSet))
	for r := range runeSet {
		runes = append(runes, r)
	}
	sort.Slice(runes, func(i, j int) bool { return runes[i] < runes[j] })
	pairs := make([][2]rune, 0, len(pairSet))
	for p := range pairSet {
		pairs = append(pairs, p)
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i][0] != pairs[j][0] {
			return pairs[i][0] < pairs[j][0]
		}
		return pairs[i][1] < pairs[j][1]
	})
	sizes := []int{6, 12, 14, 18, 22, 28, 31, 54}
	type metric struct {
		Px      int `json:"px"`
		Height  int `json:"height"`
		Ascent  int `json:"ascent"`
		Descent int `json:"descent"`
		AscPx   int `json:"ascent_round"`
	}
	var metrics []metric
	glyphs := map[string][]int{} // 字元 → 每個尺寸 [minX minY maxX maxY advance ok]，加上 GlyphIndex
	var buf sfnt.Buffer
	for _, px := range sizes {
		face, asc := f.faceFor(px)
		m := face.Metrics()
		metrics = append(metrics, metric{Px: px, Height: int(m.Height), Ascent: int(m.Ascent), Descent: int(m.Descent), AscPx: int(asc)})
	}
	for _, r := range runes {
		index, err := f.sf.GlyphIndex(&buf, r)
		if err != nil {
			index = 0
		}
		row := []int{int(index)}
		for _, px := range sizes {
			face, _ := f.faceFor(px)
			b, a, ok := face.GlyphBounds(r)
			okv := 0
			if ok {
				okv = 1
			}
			row = append(row, int(b.Min.X), int(b.Min.Y), int(b.Max.X), int(b.Max.Y), int(a), okv)
		}
		glyphs[fmt.Sprintf("%d", r)] = row
	}
	face18, _ := f.faceFor(18)
	kerns := map[string]int{}
	for _, p := range pairs {
		if k := face18.Kern(p[0], p[1]); k != 0 {
			kerns[fmt.Sprintf("%d,%d", p[0], p[1])] = int(k)
		}
	}
	boundSizes := []int{18, 22}
	bounds := make([][]int, 0, len(strs))
	for _, s := range strs {
		row := []int{}
		for _, px := range boundSizes {
			face, _ := f.faceFor(px)
			b := text.BoundString(face, s)
			row = append(row, b.Min.X, b.Min.Y, b.Max.X, b.Max.Y)
		}
		bounds = append(bounds, row)
	}
	strRunes := make([]int, len(runes))
	for i, r := range runes {
		strRunes[i] = int(r)
	}
	manifest := map[string]any{
		"schema":       "fd2_rust_r5_ttf_metrics/1",
		"plan":         "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
		"producer":     "remake/cmd/fd2/rust_golden_ttf_layers_test.go TestRustGoldenTTFMetrics（tools/rust_r5f_ttf_layers.sh）",
		"go_version":   runtime.Version(),
		"font":         fontPath,
		"sizes":        sizes,
		"metrics":      metrics,
		"runes":        strRunes,
		"glyphs":       glyphs,
		"kern_px":      18,
		"kern_pairs":   len(pairs),
		"kern_nonzero": kerns,
		"bound_sizes":  boundSizes,
		"strings":      strs,
		"bounds":       bounds,
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "ttf-metrics.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("ttf metrics: runes=%d pairs=%d nonzero_kern=%d strings=%d", len(runes), len(pairs), len(kerns), len(strs))
}
