package main

// R5 階段 2 r5m 組（docs/goal/121）的 Go 局部黃金工具，只輸出資料。未設
// FD2_GOLDEN_MAP_LAYERS_OUT 時略過，不改任何遊戲行為；由 tools/rust_r5_map_layers.sh
// 在 Docker／Xvfb 內呼叫。
//
// chapter 模式的整張畫格混有 HUD、TTF 短訊、指令環等別組的圖層，不能拿來單獨驗地圖與
// 單位組的函式。這裡照 TestRustGoldenFrames 的 chapter 模式原樣跑第一章受控重播（同一個
// ch01FrameObserver 掛點、同一個 60 Hz 虛擬時鐘與 rand.Seed(4)），但每個 pump 幀不呼叫整個
// Draw，只照 Draw 的分支順序呼叫本組的函式，各自畫到空白畫布再讀回：
//
//   - spawn：Draw 走到 spawnIntroTransition 分支時，Fill 黑再 drawNativeSpawnIntro（640×400）。
//   - tail：Draw 走到尾段時
//     - world：正規化世界層，照 Draw 本體 main.go:9817..9909 的圖塊迴圈（tileAt）、單位迴圈
//       （mapSpriteGroup＋drawUnitSprite，含已行動單位的 ColorScale）與 storyActors 迴圈，畫到
//       Fill 黑的目標（storyBG／戰場事件時是 320×200 storyView，否則是 640×400）；移動範圍格、
//       storyZoom 放大、游標與 HUD 是 Draw 本體與別組的，不畫。
//     - map：nativeMapFrameAdmission 通過時，Fill 黑再 drawNativeMapFrame（640×400），另以
//       nativeMapFrameImage 再取一次 320×200 索引畫格與調色盤的雜湊。
//
// 每個 pump 幀開頭照 Draw 開頭清 nativeMapPresented／movementPanelDraw／
// nativeInterfaceHUDVisible（忠實模式下 Draw 開頭其餘欄位都是 nil）。Draw 裡其他分支的流程
// 旗標照 R4 由 ackPresents 承認，與 Rust 無畫面重播相同。

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/color"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type mapLayerRow struct {
	Capture     int    `json:"capture"`
	Count       int    `json:"count"`
	UpdateFrame int    `json:"update_frame"`
	Node        string `json:"node"`
	Key         string `json:"key"`
}

type mapLayerRecorder struct {
	t        *testing.T
	probe    *goldenFrameProbe
	clock    goldenFrameRecorder // 只借用 pinClock（同一個虛擬時鐘）
	out      string
	screen   *ebiten.Image
	view     *ebiten.Image
	pix      []byte
	viewPix  []byte
	rows     []mapLayerRow
	seq      []byte
	capture  int
	branches map[string]int
	stats    map[string]int
}

// goldenMapLayersBranch 是 Draw（main.go:9395..9797）的 early-return 順序，只讀欄位：
// spawn＝spawnIntroTransition 分支，tail＝走到尾段，early＝前面任一分支先 return。
func goldenMapLayersBranch(g *Game) string {
	switch {
	case g.startupBlocked, g.nativeDefeat != nil, g.titlePhase != "", g.nativeEnding != nil,
		g.nativeCh20SkyKey != nil, g.nativeCh23Loop != nil, g.native2189A != nil, g.nativeUnitPresent != nil,
		g.nativeCh28PostPresent != nil, g.transitionReveal != nil, g.m == nil, g.indexedTransition != nil,
		g.nativeHealPresentation != nil, g.nativeModifierPresentation != nil, g.nativeAICommandModifier != nil,
		g.nativeCmd0Presentation != nil, g.nativeCmd6Presentation != nil, g.nativeCmd1Presentation != nil,
		g.nativeCmd2Presentation != nil, g.nativeCmd3Presentation != nil, g.nativeCmd5Presentation != nil,
		g.nativeCmd7Presentation != nil, g.nativeCmd8Presentation != nil, g.nativeCmd9Player != nil,
		g.nativeCmd9AIPresentation != nil, g.nativeCmd1012 != nil, g.nativeCmd24Presentation != nil,
		g.nativeCmd29Presentation != nil, g.nativeCmd32Presentation != nil, g.nativeCmd33Presentation != nil,
		g.nativeCmd34Presentation != nil, g.nativeCmd35Presentation != nil, g.nativeAIItemPresentation != nil,
		g.nativeTownDeparture != nil, g.nativePaletteRamp != nil, g.nativePalettePulse != nil:
		return "early"
	case g.spawnIntroTransition != nil:
		return "spawn"
	case g.nativeTurnStaging != nil && g.nativeTurnStaging.indexed, g.nativeFieldEvent61 != nil,
		g.nativeEndTurnRecovery != nil, g.nativeAIIdleRecovery != nil, g.atk != nil:
		return "early"
	}
	return "tail"
}

func mapLayerSHA(pix []byte) string {
	sum := sha256.Sum256(pix)
	return hex.EncodeToString(sum[:])
}

// mapLayerPaletteSHA：每色 color.RGBA() 的高 8 位元（與 NewImageFromImage 寫進貼圖的值相同）。
func mapLayerPaletteSHA(p color.Palette) string {
	raw := make([]byte, 0, len(p)*4)
	for _, c := range p {
		r, g, b, a := c.RGBA()
		raw = append(raw, byte(r>>8), byte(g>>8), byte(b>>8), byte(a>>8))
	}
	return mapLayerSHA(raw)
}

// drawWorld 照 main.go:9817..9909 畫正規化世界層（圖塊、單位、storyActors）。
func (r *mapLayerRecorder) drawWorld(g *Game, target *ebiten.Image, viewW, viewH int) (units, acted, fallback int) {
	tw, th := g.m.TileW, g.m.TileH
	x0 := int(g.camX) / tw
	y0 := int(g.camY) / th
	x1 := (int(g.camX)+viewW)/tw + 1
	y1 := (int(g.camY)+viewH)/th + 1
	for cy := y0; cy <= y1 && cy < g.m.H; cy++ {
		for cx := x0; cx <= x1 && cx < g.m.W; cx++ {
			if cy < 0 || cx < 0 {
				continue
			}
			t := g.tileAt(g.m.Tiles[cy*g.m.W+cx])
			if t == nil {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(cx*tw)-g.camX, float64(cy*th)-g.camY)
			target.DrawImage(t, op)
		}
	}
	count := func(u *battle.Unit, group int) {
		units++
		if len(g.sprites[group]) == 0 {
			fallback++
		} else if u.Acted {
			acted++
		}
	}
	if g.st != nil {
		for _, u := range g.st.Units {
			if !u.OnField || !u.Alive() {
				continue
			}
			ux := float64(u.X*tw) - g.camX
			uy := float64(u.Y*th) - g.camY
			if ux < -float64(tw) || ux > float64(viewW) || uy < -float64(th) || uy > float64(viewH) {
				continue
			}
			group := g.mapSpriteGroup(u)
			count(u, group)
			g.drawUnitSprite(target, ux, uy, float64(tw), float64(th), u, group)
		}
	}
	for i := range g.storyActors {
		u := &g.storyActors[i]
		if !u.OnField || !u.Alive() {
			continue
		}
		ux := float64(u.X*tw) - g.camX
		uy := float64(u.Y*th) - g.camY
		count(u, u.Fig)
		g.drawUnitSprite(target, ux, uy, float64(tw), float64(th), u, u.Fig)
	}
	return units, acted, fallback
}

// layers 在 Ebiten 的 Update 內執行：回傳這一幀的比對鍵（Rust 端以同一格式組字串）。
func (r *mapLayerRecorder) layers(g *Game) string {
	g.nativeMapPresented = false
	g.movementPanelDraw = nil
	g.nativeInterfaceHUDVisible = false
	branch := goldenMapLayersBranch(g)
	r.branches[branch]++
	switch branch {
	case "spawn":
		r.screen.Clear()
		r.screen.Fill(color.Black)
		ok := g.drawNativeSpawnIntro(r.screen)
		r.screen.ReadPixels(r.pix)
		if ok {
			r.stats["spawn_drawn"]++
		}
		return fmt.Sprintf("spawn|ok=%t|rgba=%s", ok, mapLayerSHA(r.pix))
	case "tail":
		if g.modernStoryPortraits != nil {
			r.t.Fatal("局部黃金只涵蓋忠實模式（modernStoryPortraits 必須是 nil）")
		}
		campaignBattleView := g.camp == nil || (g.camp.Node() != nil && g.camp.Node().Type == "battle")
		legacyViewport := g.storyBG || g.battleEvent != nil
		target, viewW, viewH, pix := r.screen, logicalW, logicalH, r.pix
		if legacyViewport {
			target, viewW, viewH, pix = r.view, logicalW/storyZoom, logicalH/storyZoom, r.viewPix
		}
		target.Clear()
		target.Fill(color.RGBA{0, 0, 0, 0xff})
		units, acted, fallback := r.drawWorld(g, target, viewW, viewH)
		target.ReadPixels(pix)
		r.stats["world_units"] += units
		r.stats["world_acted_units"] += acted
		r.stats["world_fallback_units"] += fallback
		key := fmt.Sprintf("tail|legacy=%t|world=%dx%d:%s|units=%d|acted=%d|fallback=%d",
			legacyViewport, viewW, viewH, mapLayerSHA(pix), units, acted, fallback)
		if !g.nativeMapFrameAdmission(g.storyBG, campaignBattleView) {
			return key + "|admitted=false"
		}
		r.stats["map_admitted"]++
		r.screen.Clear()
		r.screen.Fill(color.RGBA{0, 0, 0, 0xff})
		ok := g.drawNativeMapFrame(r.screen)
		r.screen.ReadPixels(r.pix)
		key += fmt.Sprintf("|admitted=true|map_ok=%t|map=%s|hud=%t", ok, mapLayerSHA(r.pix), g.nativeInterfaceHUDVisible)
		if !ok {
			return key
		}
		r.stats["map_drawn"]++
		if step := g.phaseBannerStep(); step >= 0 {
			r.stats["map_banner_mosaic"]++
		}
		if _, visible := g.phaseBannerSlideOffset(); visible {
			r.stats["map_banner_glyphs"]++
		}
		img, ok := g.nativeMapFrameImage()
		if !ok {
			r.t.Fatal("drawNativeMapFrame 成功後 nativeMapFrameImage 失敗")
		}
		return key + fmt.Sprintf("|index=%s|palette=%s", mapLayerSHA(img.Pix), mapLayerPaletteSHA(img.Palette))
	}
	return branch
}

func (r *mapLayerRecorder) observe(g *Game) {
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

func (r *mapLayerRecorder) finish() {
	seq := sha256.Sum256(r.seq)
	manifest := map[string]any{
		"schema":           "fd2_rust_r5_map_layers/1",
		"plan":             "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
		"producer":         "remake/cmd/fd2/rust_golden_map_layers_test.go（tools/rust_r5_map_layers.sh）",
		"go_version":       runtime.Version(),
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
	}
	raw, err := json.MarshalIndent(manifest, "", " ")
	if err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.out, "map-layers.json"), append(raw, '\n'), 0o644); err != nil {
		r.t.Fatal(err)
	}
	r.t.Logf("map layers: captures=%d rows=%d branches=%v stats=%v sequence=%x",
		r.capture, len(r.rows), r.branches, r.stats, seq)
}

func TestRustGoldenMapLayers(t *testing.T) {
	out := os.Getenv("FD2_GOLDEN_MAP_LAYERS_OUT")
	if out == "" {
		t.Skip("需要 FD2_GOLDEN_MAP_LAYERS_OUT（R5 r5m 局部黃金，只由 tools/rust_r5_map_layers.sh 執行）")
	}
	parity := os.Getenv("FD2_PARITY_OUT")
	if parity == "" || os.Getenv("FD2_PARITY_CAPTURE_FRAMES") != "0" || os.Getenv("FD2_PARITY_ENTRY") != "start" ||
		os.Getenv("FD2_PARITY_CHAPTER") != "1" {
		t.Fatal("需要與 tools/rust_r4_ch01_coverage.sh 相同的 FD2_PARITY_* 設定（第一章、START、不擷取索引畫格）")
	}
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
	rec := &mapLayerRecorder{t: t, probe: probe, clock: goldenFrameRecorder{t: t}, out: out,
		screen: ebiten.NewImage(logicalW, logicalH), view: ebiten.NewImage(logicalW/storyZoom, logicalH/storyZoom),
		pix: make([]byte, logicalW*logicalH*4), viewPix: make([]byte, logicalW/storyZoom*logicalH/storyZoom*4),
		branches: map[string]int{}, stats: map[string]int{}}
	saved := ch01FrameObserver
	ch01FrameObserver = rec.observe
	defer func() { ch01FrameObserver = saved }()
	TestChapterParityReplay(t)
	if t.Failed() {
		return
	}
	rec.finish()
}
