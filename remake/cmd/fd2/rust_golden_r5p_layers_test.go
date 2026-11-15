package main

// R5 階段 2 r5p 組（docs/goal/121，轉場與調色盤＋Draw 本體兩段）的 Go 局部黃金工具，只輸出資料。
// 未設 FD2_GOLDEN_R5P_LAYERS_OUT 時略過，不改任何遊戲行為；由 tools/rust_r5p_layers.sh 在
// Docker／Xvfb 內呼叫。
//
// 兩種模式（FD2_GOLDEN_R5P_LAYERS_MODE）的掛點、步驟、60 Hz 虛擬時鐘與 rand.Seed(4) 都與
// TestRustGoldenFrames 相同：chapter 掛共用 pump 的 ch01FrameObserver；opening 照
// goldenFramesOpening 的四段迴圈（開場到選單、START、settleTown、driveStory 到 battle_ch01），
// 每個 Update 之後擷取。每次擷取不呼叫整個 Draw，依 Draw 的分支順序只畫本組的圖層：
//
//   - ramp：Draw 走到 nativePaletteRamp 分支時呼叫整個 g.Draw（該分支是 Fill 黑、
//     drawNativePaletteRamp、defer 的 DAC 白／黑 Fill，沒有別組圖層），記整張 640×400 雜湊，
//     可與階段 0 chapter 黃金同一擷取點直接比。
//   - tail：Draw 走到尾段時，640×400 Fill 黑後畫正規化世界層（main.go:9816..9919 的逐字複本：
//     storyView 離屏、圖塊、移動範圍格、施法／攻擊射程格、單位、storyActors、storyZoom 放大），
//     記雜湊（world）；g.fade 非 nil 時再於同一張畫布疊舊 fade 黑色疊層（main.go:10174..10183
//     的逐字複本），記雜湊（fade）。游標與左下 HUD（HUD 組）與其後各層不畫；原生整幀（r5m）照 Draw
//     畫到另一張畫布，只為了留下它對 nativeMapVGA 的重建（之後的調色盤斜坡從那裡取底圖），鍵只記成功與否。
//
// 每次擷取開頭照 Draw 開頭清 nativeMapPresented／movementPanelDraw／nativeInterfaceHUDVisible。
//
// 兩段逐字複本以「逐字複本開始／結束」標記包住，tools/rust_r5p_layers.sh 每次執行先核對它們與
// main.go 對應行逐字相同（Go 凍結，main.go 不改）。Rust 端呼叫正式的
// Game::draw_normalized_world_layer／draw_story_fade_overlay。

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
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type r5pLayerRow struct {
	Capture     int    `json:"capture"`
	Count       int    `json:"count"`
	UpdateFrame int    `json:"update_frame"`
	Node        string `json:"node"`
	Key         string `json:"key"`
}

type r5pLayerRecorder struct {
	t         *testing.T
	probe     *goldenFrameProbe
	clock     goldenFrameRecorder // 只借用 pinClock（同一個虛擬時鐘）
	out       string
	screen    *ebiten.Image
	mapScreen *ebiten.Image
	pix       []byte
	rows      []r5pLayerRow
	seq       []byte
	capture   int
	branches  map[string]int
	stats     map[string]int
}

// r5pLayerBranch：Draw（main.go:9395..9797）的 early-return 順序，只讀欄位。nativePaletteRamp
// 之前的分支先 return 時是 early；其後同 goldenMapLayersBranch（spawn／early／tail）。
func r5pLayerBranch(g *Game) string {
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
		g.nativeTownDeparture != nil:
		return "early"
	case g.nativePaletteRamp != nil:
		return "ramp"
	}
	return goldenMapLayersBranch(g)
}

// r5pDrawNormalizedWorld：Draw 的 main.go:9812..9814 與 9816..9919（!modernHDRequested 時）。
func r5pDrawNormalizedWorld(g *Game, screen *ebiten.Image) {
	target, viewW, viewH := screen, logicalW, logicalH
	legacyViewport := g.storyBG || g.battleEvent != nil
	tw, th := g.m.TileW, g.m.TileH
	{
		// ---- 逐字複本開始：main.go:9816..9919 ----
		if legacyViewport {
			if g.storyView == nil {
				g.storyView = ebiten.NewImage(logicalW/storyZoom, logicalH/storyZoom)
			}
			g.storyView.Fill(color.RGBA{0, 0, 0, 0xff})
			target, viewW, viewH = g.storyView, logicalW/storyZoom, logicalH/storyZoom
		}
		// 只畫可見範圍
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
		// 移動範圍高亮(已選單位:藍色半透明格)
		if g.sel != nil {
			hl := ebiten.NewImage(tw, th)
			hl.Fill(color.RGBA{0x40, 0x80, 0xff, 0x66})
			for c := range g.reach {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(c.X*tw)-g.camX, float64(c.Y*th)-g.camY)
				target.DrawImage(hl, op)
			}
			if g.castSp != nil { // 施法射程高亮(紫)
				ch := ebiten.NewImage(tw, th)
				ch.Fill(color.RGBA{0xa0, 0x50, 0xe0, 0x5c})
				for y := 0; y < g.m.H; y++ {
					for x := 0; x < g.m.W; x++ {
						if g.st.InCastRange(g.sel, *g.castSp, x, y) {
							op := &ebiten.DrawImageOptions{}
							op.GeoM.Translate(float64(x*tw)-g.camX, float64(y*th)-g.camY)
							target.DrawImage(ch, op)
						}
					}
				}
			}
			// 原生物品／command 0 目標 modal 只由完整的 0x11CAC 索引畫面
			// 呈現。缺少 raw target field、LUT 或 range sprite 時，不畫沒有
			// 原版證據的 RGBA 色塊作後備，避免把 renderer 失敗偽裝成成功。
			// 攻擊射程高亮(紅;已移動、選攻擊、尚未選中目標的階段,doc32 武器攻擊距離接線 —
			// 沒有這格高亮,槍兵2格射程會「打得到但畫面看不出範圍」)
			if g.castSp == nil && g.moved && !g.ring && !g.spellOpen {
				ah := ebiten.NewImage(tw, th)
				ah.Fill(color.RGBA{0xe0, 0x30, 0x30, 0x5c})
				for y := 0; y < g.m.H; y++ {
					for x := 0; x < g.m.W; x++ {
						if g.st.InAttackRange(g.sel, x, y) {
							op := &ebiten.DrawImageOptions{}
							op.GeoM.Translate(float64(x*tw)-g.camX, float64(y*th)-g.camY)
							target.DrawImage(ah, op)
						}
					}
				}
			}
		}
		// 單位層(M1:FIGANI 待機動畫 sprite + 陣營腳標 + HP bar;無 sprite 退回色塊)
		if g.st != nil {
			for _, u := range g.st.Units {
				if !u.OnField || !u.Alive() { // 待命(未進場)單位不畫
					continue
				}
				ux := float64(u.X*tw) - g.camX
				uy := float64(u.Y*th) - g.camY
				if ux < -float64(tw) || ux > float64(viewW) || uy < -float64(th) || uy > float64(viewH) {
					continue
				}
				g.drawUnitSprite(target, ux, uy, float64(tw), float64(th), u, g.mapSpriteGroup(u))
			}
		}
		// storyBG 場景靜態角色(doc23 §4:王座廳國王/王后/主角等 cutscene 擺位,同一 sprite 繪法無戰鬥邏輯)
		for i := range g.storyActors {
			u := &g.storyActors[i]
			if !u.OnField || !u.Alive() {
				continue
			}
			ux := float64(u.X*tw) - g.camX
			uy := float64(u.Y*th) - g.camY
			g.drawUnitSprite(target, ux, uy, float64(tw), float64(th), u, u.Fig)
		}
		if legacyViewport { // 離屏世界層放大貼回畫布(48px/格,原版取景)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(storyZoom, storyZoom)
			screen.DrawImage(g.storyView, op)
			if g.unitLabels { // FD2_UNIT_LABELS:cutscene sprite 左上標 [idx]名(x,y)dDir,協助回報/對映原版 slot
				for i := range g.storyActors {
					u := &g.storyActors[i]
					sx := (float64(u.X*tw) - g.camX + u.OffX) * storyZoom
					sy := (float64(u.Y*th) - g.camY + u.OffY) * storyZoom
					ebitenutil.DebugPrintAt(screen, fmt.Sprintf("[%d]f%d(%d,%d)d%d", i, u.Fig, u.X, u.Y, u.Dir), int(sx), int(sy)-14)
				}
			}
		}
		// ---- 逐字複本結束：main.go:9816..9919 ----
	}
}

// r5pDrawFade：Draw 的 main.go:10174..10183。
func r5pDrawFade(g *Game, screen *ebiten.Image) {
	// ---- 逐字複本開始：main.go:10174..10183 ----
	if g.fade != nil {
		frac := float64(g.fade.t) / float64(g.fade.total)
		alpha := frac
		if !g.fade.out {
			alpha = 1 - frac
		}
		ov := ebiten.NewImage(logicalW, logicalH)
		ov.Fill(color.RGBA{0, 0, 0, uint8(alpha * 0xff)})
		screen.DrawImage(ov, &ebiten.DrawImageOptions{})
	}
	// ---- 逐字複本結束：main.go:10174..10183 ----
}

// r5pFadeAlpha 是 main.go:10175..10181 的 uint8(alpha * 0xff)，只供比對鍵顯示。
func r5pFadeAlpha(g *Game) uint8 {
	frac := float64(g.fade.t) / float64(g.fade.total)
	alpha := frac
	if !g.fade.out {
		alpha = 1 - frac
	}
	return uint8(alpha * 0xff)
}

func r5pSHA(pix []byte) string {
	sum := sha256.Sum256(pix)
	return hex.EncodeToString(sum[:])
}

// layers 在 Ebiten 的 Update 內執行：回傳這一幀的比對鍵（Rust 端以同一格式組字串）。
func (r *r5pLayerRecorder) layers(g *Game) string {
	// 照 Draw 開頭清三個本幀旗標（同 rust_golden_map_layers_test.go；忠實模式下其餘欄位都是 nil）。
	g.nativeMapPresented = false
	g.movementPanelDraw = nil
	g.nativeInterfaceHUDVisible = false
	branch := r5pLayerBranch(g)
	r.branches[branch]++
	switch branch {
	case "ramp":
		j := g.nativePaletteRamp
		hd := j.hdBattleSources != nil
		r.screen.Clear()
		g.Draw(r.screen)
		r.screen.ReadPixels(r.pix)
		r.stats["ramp_frames"]++
		if j.drawn {
			r.stats["ramp_drawn"]++
		}
		return fmt.Sprintf("ramp|step=%d|drawn=%t|hd_sources=%t|dac_white=%t|dac_black=%t|help=%t|rgba=%s",
			j.step, j.drawn, hd, g.nativeFullDACWhite, g.nativeFullDACBlack, g.helpVisible, r5pSHA(r.pix))
	case "tail":
		if g.modernStoryPortraits != nil {
			r.t.Fatal("局部黃金只涵蓋忠實模式（modernStoryPortraits 必須是 nil）")
		}
		if g.unitLabels {
			r.t.Fatal("FD2_UNIT_LABELS 不在局部黃金範圍")
		}
		legacyViewport := g.storyBG || g.battleEvent != nil
		sel := g.sel != nil
		reach := 0
		if sel {
			reach = len(g.reach)
		}
		cast := sel && g.castSp != nil
		attack := sel && g.castSp == nil && g.moved && !g.ring && !g.spellOpen
		r.screen.Clear()
		r.screen.Fill(color.RGBA{0, 0, 0, 0xff})
		r5pDrawNormalizedWorld(g, r.screen)
		r.screen.ReadPixels(r.pix)
		r.stats["world_frames"]++
		if legacyViewport {
			r.stats["world_legacy"]++
		}
		if sel {
			r.stats["world_sel"]++
			r.stats["world_reach_cells"] += reach
		}
		if cast {
			r.stats["world_cast"]++
		}
		if attack {
			r.stats["world_attack"]++
		}
		key := fmt.Sprintf("tail|legacy=%t|sel=%t|reach=%d|cast=%t|attack=%t|world=%s",
			legacyViewport, sel, reach, cast, attack, r5pSHA(r.pix))
		// Draw 的下一步：原生整幀（r5m 的函式，局部黃金已比過像素）。這裡照畫到另一張畫布，只為了留下
		// Draw 對狀態的同一個改動（composeNativeMapFrame 依當下時間重建 nativeMapVGA，之後的調色盤
		// 斜坡從它取底圖）；鍵只記成功與否。
		campaignBattleView := g.camp == nil || (g.camp.Node() != nil && g.camp.Node().Type == "battle")
		if g.nativeMapFrameAdmission(g.storyBG, campaignBattleView) {
			r.mapScreen.Clear()
			key += fmt.Sprintf("|map_ok=%t", g.drawNativeMapFrame(r.mapScreen))
		}
		if g.fade == nil {
			return key + "|fade=false"
		}
		alpha := r5pFadeAlpha(g)
		r5pDrawFade(g, r.screen)
		r.screen.ReadPixels(r.pix)
		r.stats["fade_frames"]++
		return key + fmt.Sprintf("|fade=true|fade_out=%t|fade_t=%d/%d|fade_alpha=%d|fade_rgba=%s",
			g.fade.out, g.fade.t, g.fade.total, alpha, r5pSHA(r.pix))
	}
	return branch
}

func (r *r5pLayerRecorder) observe(g *Game) {
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
	r.rows = append(r.rows, r5pLayerRow{Capture: index, Count: 1, UpdateFrame: g.frame, Node: goldenFrameNode(g), Key: key})
}

func (r *r5pLayerRecorder) finish(mode string, extra map[string]any) {
	seq := sha256.Sum256(r.seq)
	manifest := map[string]any{
		"schema":           "fd2_rust_r5p_layers/1",
		"mode":             mode,
		"plan":             "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
		"producer":         "remake/cmd/fd2/rust_golden_r5p_layers_test.go（tools/rust_r5p_layers.sh）",
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
	for k, v := range extra {
		manifest[k] = v
	}
	raw, err := json.MarshalIndent(manifest, "", " ")
	if err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.out, "r5p-layers.json"), append(raw, '\n'), 0o644); err != nil {
		r.t.Fatal(err)
	}
	r.t.Logf("r5p layers %s: captures=%d rows=%d branches=%v stats=%v sequence=%x",
		mode, r.capture, len(r.rows), r.branches, r.stats, seq)
}

func TestRustGoldenR5pLayers(t *testing.T) {
	out := os.Getenv("FD2_GOLDEN_R5P_LAYERS_OUT")
	if out == "" {
		t.Skip("需要 FD2_GOLDEN_R5P_LAYERS_OUT（R5 r5p 局部黃金，只由 tools/rust_r5p_layers.sh 執行）")
	}
	mode := os.Getenv("FD2_GOLDEN_R5P_LAYERS_MODE")
	if mode != "chapter" && mode != "opening" {
		t.Fatalf("FD2_GOLDEN_R5P_LAYERS_MODE=%q；只允許 chapter 或 opening", mode)
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
	rec := &r5pLayerRecorder{t: t, probe: probe, clock: goldenFrameRecorder{t: t}, out: out,
		screen: ebiten.NewImage(logicalW, logicalH), mapScreen: ebiten.NewImage(logicalW, logicalH),
		pix:      make([]byte, logicalW*logicalH*4),
		branches: map[string]int{}, stats: map[string]int{}}
	switch mode {
	case "chapter":
		parity := os.Getenv("FD2_PARITY_OUT")
		if parity == "" || os.Getenv("FD2_PARITY_CAPTURE_FRAMES") != "0" || os.Getenv("FD2_PARITY_ENTRY") != "start" ||
			os.Getenv("FD2_PARITY_CHAPTER") != "1" {
			t.Fatal("chapter 模式需要與 tools/rust_r4_ch01_coverage.sh 相同的 FD2_PARITY_* 設定（第一章、START、不擷取索引畫格）")
		}
		saved := ch01FrameObserver
		ch01FrameObserver = rec.observe
		defer func() { ch01FrameObserver = saved }()
		TestChapterParityReplay(t)
		if t.Failed() {
			return
		}
		rec.finish(mode, map[string]any{"scope": "第一章受控重播的每個 pump 幀（同 TestRustGoldenFrames chapter 模式）"})
	case "opening":
		rec.finish(mode, r5pLayersOpening(t, rec))
	}
}

// r5pLayersOpening 是 goldenFramesOpening 的同款步驟（那個函式綁定 *goldenFrameRecorder，
// 不能換觀察者，所以照抄；兩者的差別只有 rec 的型別）。
func r5pLayersOpening(t *testing.T, rec *r5pLayerRecorder) map[string]any {
	slot := os.Getenv("FD2_PARITY_SLOT")
	if slot == "" {
		t.Fatal("opening 模式需要 FD2_PARITY_SLOT（與 chapter 模式同一個建構槽）")
	}
	t.Setenv("FD2_TITLE", "1")
	t.Setenv("FD2_SEED", "4")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	t.Setenv("FD2_SHOT_AI", "1")
	slotCopy := filepath.Join(t.TempDir(), "FD2.SAV")
	if raw, err := os.ReadFile(slot); err != nil {
		t.Fatal(err)
	} else if err := os.WriteFile(slotCopy, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_NATIVE_SAVE", slotCopy)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })

	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if g.titlePhase != "cutscene" {
		t.Fatalf("正常START未從開場動畫開始：%q", g.titlePhase)
	}
	step := func() {
		if err := parityUpdateGame(g); err != nil {
			t.Fatal(err)
		}
		if g.loadErr != "" {
			t.Fatalf("執行期錯誤：%s", g.loadErr)
		}
		rec.observe(g)
	}
	rec.observe(g) // loadGame 之後、第一個 Update 之前
	phases := map[string]int{}
	mark := func(name string) { phases[name] = rec.capture }
	for frame := 0; g.titlePhase != "menu"; frame++ {
		if frame >= journeyTitleFrames || g.titlePhase == "" {
			t.Fatal("正常開場動畫未進標題選單")
		}
		ackPresents(g)
		step()
	}
	mark("title_menu")
	if g.titleSel != 0 || !g.applyTitleMenuEvent(TitleMenuConfirm) {
		t.Fatal("正常START未由標題owner消費")
	}
	for frame := 0; g.titlePhase != ""; frame++ {
		if frame >= 600 {
			t.Fatal("START未正常離開標題")
		}
		ackPresents(g)
		answerNativeTreasurePrompt(g)
		step()
	}
	if g.camp == nil || g.camp.NodeID() != journeyOpeningNode {
		t.Fatal("START未正常進入序章")
	}
	mark("story_ch00")
	for frame := 0; frame < journeyTownSettleMax; frame++ {
		if g.nativeClassUIJob == nil && g.fade == nil && g.transitionReveal == nil &&
			g.indexedTransition == nil && frame > 10 {
			break
		}
		ackPresents(g)
		step()
	}
	j := newJourneyTrace()
	lastSig, still := "", 0
	for frame := 0; g.camp.NodeID() != "battle_ch01"; frame++ {
		if frame >= journeyStoryFrames {
			t.Fatalf("故事節點 %s 在 %d 幀內沒有結束", g.camp.NodeID(), journeyStoryFrames)
		}
		j.observe(g)
		if n := g.camp.Node(); n != nil && (n.Type == "story" || n.Type == "cutscene") && storyEnterReady(g) {
			before := storySignature(g)
			g.handleNativeStoryInput(n, nativeStoryInput{enter: true})
			if storySignature(g) != before {
				j.press(journeyPhase(g))
			}
		}
		ackPresents(g)
		step()
		if sig := progressSignature(g); sig == lastSig {
			if still++; still > journeyStallFrames {
				t.Fatalf("故事節點 %s 卡住 %d 幀", g.camp.NodeID(), still)
			}
		} else {
			lastSig, still = sig, 0
		}
	}
	mark("battle_ch01")
	return map[string]any{
		"scope":          "標題開場動畫 → 選單 → START → 序章故事 → 第一次進入 battle_ch01；每個 Update 之後擷取（另含 loadGame 後一張）",
		"phase_captures": phases,
		"story_nodes":    strings.Join(j.Nodes, " → "),
		"story_presses":  j.Presses,
	}
}
