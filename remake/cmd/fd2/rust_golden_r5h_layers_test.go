package main

// R5 階段 2 r5h 組（docs/goal/121，HUD 與選單）的 Go 局部黃金工具，只輸出資料。未設
// FD2_GOLDEN_R5H_LAYERS_OUT 時略過，不改任何遊戲行為；由 tools/rust_r5h_layers.sh 在 Docker／Xvfb 內呼叫。
//
// 掛點、60 Hz 虛擬時鐘與 rand.Seed(4) 同 TestRustGoldenFrames 的 chapter 模式（共用 pump 的
// ch01FrameObserver）。每次擷取不呼叫整個 Draw，依 Draw 的分支順序只畫本組的圖層：
//
//   - recovery：Draw 走到 nativeEndTurnRecovery 分支時呼叫整個 g.Draw（該分支是 Fill 黑、
//     drawNativeEndTurnRecovery、defer 的 DAC 白／黑 Fill，沒有別組圖層），記整張 640×400 雜湊，可與階段 0
//     chapter 黃金同一擷取點直接比。
//   - tail：三段各畫到一張 Clear 的 640×400 畫布，記雜湊（畫布全透明時記 0）：
//     1. 游標白框與左下 HUD（main.go:9925..9941 的逐字複本）；
//     2. 照 Draw 把原生整幀（r5m）畫到另一張畫布，取 nativeMapPresented 與它對狀態的改動，鍵只記成功與否；
//        再畫移動面板與回合橫幅（main.go:9964..9972 的逐字複本）；
//     3. TTF 段在語系缺鍵時 return 出 Draw（main.go:10137..10146），照同一條件判斷；沒有 return 時畫
//        設施／班級工作、END 對話、指令環、指令格、法術與物品選單、系統資訊與角色狀態（main.go:10156..10174
//        的逐字複本）。
//     正規化世界層（r5p）、對白（r5d）、TTF 短訊（r5f）與 drawCampaignUI（r5c）是別組的圖層，不畫。
//
// 每次擷取開頭照 Draw 開頭清 nativeMapPresented／movementPanelDraw／nativeInterfaceHUDVisible。
//
// 三段逐字複本以「逐字複本開始／結束」標記包住，tools/rust_r5h_layers.sh 每次執行先核對它們與 main.go
// 對應行逐字相同（Go 凍結，main.go 不改；行號是 v.1.1.14 之後的現行 main.go）。Rust 端呼叫從 draw_field 拆出的
// Game::draw_field_cursor_hud／draw_field_map_overlays／draw_field_command_ui。

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type r5hLayerRow struct {
	Capture     int    `json:"capture"`
	Count       int    `json:"count"`
	UpdateFrame int    `json:"update_frame"`
	Node        string `json:"node"`
	Key         string `json:"key"`
}

type r5hLayerRecorder struct {
	t         *testing.T
	probe     *goldenFrameProbe
	clock     goldenFrameRecorder // 只借用 pinClock（同一個虛擬時鐘）
	out       string
	screen    *ebiten.Image
	mapScreen *ebiten.Image
	pix       []byte
	rows      []r5hLayerRow
	seq       []byte
	capture   int
	branches  map[string]int
	stats     map[string]int
}

// r5hPreSpawnEarly：Draw 在 spawnIntroTransition 之前 return 的分支（main.go:9399..9738）。
func r5hPreSpawnEarly(g *Game) bool {
	return g.startupBlocked || g.nativeDefeat != nil || g.titlePhase != "" || g.nativeEnding != nil ||
		g.nativeCh20SkyKey != nil || g.nativeCh23Loop != nil || g.native2189A != nil || g.nativeUnitPresent != nil ||
		g.nativeCh28PostPresent != nil || g.transitionReveal != nil || g.m == nil || g.indexedTransition != nil ||
		g.nativeHealPresentation != nil || g.nativeModifierPresentation != nil || g.nativeAICommandModifier != nil ||
		g.nativeCmd0Presentation != nil || g.nativeCmd6Presentation != nil || g.nativeCmd1Presentation != nil ||
		g.nativeCmd2Presentation != nil || g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil ||
		g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil || g.nativeCmd9Player != nil ||
		g.nativeCmd9AIPresentation != nil || g.nativeCmd1012 != nil || g.nativeCmd24Presentation != nil ||
		g.nativeCmd29Presentation != nil || g.nativeCmd32Presentation != nil || g.nativeCmd33Presentation != nil ||
		g.nativeCmd34Presentation != nil || g.nativeCmd35Presentation != nil || g.nativeAIItemPresentation != nil ||
		g.nativeTownDeparture != nil || g.nativePaletteRamp != nil || g.nativePalettePulse != nil
}

// r5hLayerBranch：Draw（main.go:9399..9800）的 early-return 順序，只讀欄位。除了 recovery 以外與
// goldenMapLayersBranch 相同。
func r5hLayerBranch(g *Game) string {
	switch {
	case r5hPreSpawnEarly(g):
		return "early"
	case g.spawnIntroTransition != nil:
		return "spawn"
	case g.nativeTurnStaging != nil && g.nativeTurnStaging.indexed, g.nativeFieldEvent61 != nil:
		return "early"
	case g.nativeEndTurnRecovery != nil:
		return "recovery"
	case g.nativeAIIdleRecovery != nil, g.atk != nil:
		return "early"
	}
	return "tail"
}

// r5hCursorHUD：Draw 的 main.go:9925..9941（!modernHDRequested 區塊內，正規化世界層之後）。
func r5hCursorHUD(g *Game, screen *ebiten.Image, tw, th int, legacyViewport, campaignBattleView bool) {
	{
		// ---- 逐字複本開始：main.go:9925..9941 ----
		curPx := float64(g.curX*tw) - g.camX
		curPy := float64(g.curY*th) - g.camY
		if !g.ring && !g.spellOpen && !legacyViewport && campaignBattleView {
			drawCursor(screen, curPx, curPy, float64(tw), float64(th))
		}
		// HUD(對照原版 orig_04/08):游標單位資訊=左下面板(非常駐頂列);回合切換=中央大字橫幅。
		if g.st != nil && g.font != nil && !legacyViewport {
			if u := g.st.UnitAt(g.curX, g.curY); u != nil { // 左下單位面板(orig 樣式)
				if !g.drawNativeMapHUD(screen) {
					g.drawUnitHUD(screen, u)
				}
			}
			if g.debug { // F3:詳細除錯(回合/戰況/座標)
				ebitenutil.DebugPrintAt(screen, fmt.Sprintf("T%d own%d ally%d enemy%d cur(%d,%d)",
					g.st.Turn, g.st.AliveCount(battle.Own), g.st.AliveCount(battle.Ally), g.st.AliveCount(battle.Enemy), g.curX, g.curY), 6, 4)
			}
		}
		// ---- 逐字複本結束：main.go:9925..9941 ----
	}
}

// r5hMapOverlays：Draw 的 main.go:9964..9972（原生整幀之後）。
func r5hMapOverlays(g *Game, screen *ebiten.Image, nativeMapPresented, legacyViewport, campaignBattleView bool) {
	// ---- 逐字複本開始：main.go:9964..9972 ----
	if nativeMapPresented {
		g.drawNativeMovementPanel(screen)
	}
	// 回合橫幅(PLAYER/ENEMY PHASE，transient)必須畫在原生整幀之後：
	// drawNativeMapFrame 是覆蓋全畫面的 320×200 blit，畫在它前面等於沒畫。
	// 這一層是重製端的近似呈現，原版自己的回合字樣仍未解出，不宣稱一致。
	if !legacyViewport && campaignBattleView {
		g.drawPhaseBanner(screen)
	}
	// ---- 逐字複本結束：main.go:9964..9972 ----
}

// r5hCommandUI：Draw 的 main.go:10156..10174。回傳 false 表示複本裡的 return（transientUI 的班級介面工作）。
func r5hCommandUI(g *Game, screen *ebiten.Image, nativeMapPresented bool) (completed bool) {
	// ---- 逐字複本開始：main.go:10156..10174 ----
	if g.transientUI && g.drawNativeClassUIJob(screen) {
		return
	}
	if !g.drawNativeSystemEndTurn(screen) {
		if g.nativeActionOverlayInMapFrame(nativeMapPresented) {
			// The admitted indexed frame already owns icons then cursor unit.
			// A second drawRing changes their placement and covers that unit.
			g.markActionOverlayDrawn()
		} else {
			g.drawRing(screen)
		}
		g.drawNativeCommandGrid(screen)
		g.drawSpellMenu(screen)
		g.drawItemMenu(screen)
	}

	// 原版戰場資訊是 320×200 indexed 全畫面，最後覆蓋戰場與選單層。
	g.drawNativeSystemInfoUI(screen)
	g.drawNativePlayerStatus(screen)
	// ---- 逐字複本結束：main.go:10156..10174 ----
	return true
}

// r5hTTFReturns：Draw 的 TTF 段（main.go:10137..10146）在語系缺鍵時 return 出 Draw 的同一條件（不畫字）。
func r5hTTFReturns(g *Game) bool {
	if g.font == nil || g.result == "" {
		return false
	}
	if _, ok := g.localeMessage("battle.result.win"); !ok {
		return true
	}
	if g.result == "lose" {
		if _, ok := g.localeMessage("battle.result.lose"); !ok {
			return true
		}
	}
	if g.camp != nil {
		if _, ok := g.localeMessage("battle.result.continue"); !ok {
			return true
		}
	}
	return false
}

// r5hLayerSHA：畫布全為 0（全透明，本層沒有畫）時回 0，否則 RGBA 的 SHA-256。
func r5hLayerSHA(pix []byte) string {
	for _, b := range pix {
		if b != 0 {
			sum := sha256.Sum256(pix)
			return hex.EncodeToString(sum[:])
		}
	}
	return "0"
}

// layer：本層畫布清成全透明、畫、取雜湊；有畫的層另記一次統計。
func (r *r5hLayerRecorder) layer(name string, draw func(screen *ebiten.Image)) string {
	r.screen.Clear()
	draw(r.screen)
	r.screen.ReadPixels(r.pix)
	sha := r5hLayerSHA(r.pix)
	if sha != "0" {
		r.stats[name+"_drawn"]++
	}
	return sha
}

// layers 在 Ebiten 的 Update 內執行：回傳這一幀的比對鍵（Rust 端以同一格式組字串）。
func (r *r5hLayerRecorder) layers(g *Game) string {
	// 照 Draw 開頭清三個本幀旗標（同 rust_golden_map_layers_test.go；忠實模式下其餘欄位都是 nil）。
	g.nativeMapPresented = false
	g.movementPanelDraw = nil
	g.nativeInterfaceHUDVisible = false
	branch := r5hLayerBranch(g)
	r.branches[branch]++
	switch branch {
	case "recovery":
		frame := g.nativeEndTurnRecovery.frame
		r.screen.Clear()
		g.Draw(r.screen)
		r.screen.ReadPixels(r.pix)
		rgba := r5hLayerSHA(r.pix)
		alive, drawn := false, false
		if j := g.nativeEndTurnRecovery; j != nil {
			alive, drawn = true, j.drawn
		}
		r.stats["recovery_frames"]++
		return fmt.Sprintf("recovery|frame=%d|alive=%t|drawn=%t|rgba=%s", frame, alive, drawn, rgba)
	case "tail":
		if g.modernStoryPortraits != nil {
			r.t.Fatal("局部黃金只涵蓋忠實模式（modernStoryPortraits 必須是 nil）")
		}
		legacyViewport := g.storyBG || g.battleEvent != nil
		campaignBattleView := g.camp == nil || (g.camp.Node() != nil && g.camp.Node().Type == "battle")
		tw, th := g.m.TileW, g.m.TileH
		cursor := !g.ring && !g.spellOpen && !legacyViewport && campaignBattleView
		hudUnit := g.st != nil && g.st.UnitAt(g.curX, g.curY) != nil && g.font != nil && !legacyViewport
		cursorHUD := r.layer("cursor_hud", func(screen *ebiten.Image) {
			r5hCursorHUD(g, screen, tw, th, legacyViewport, campaignBattleView)
		})
		r.stats["tail_frames"]++
		if cursor {
			r.stats["cursor"]++
		}
		if hudUnit {
			r.stats["hud_unit"]++
		}
		key := fmt.Sprintf("tail|legacy=%t|cursor=%t|hud_unit=%t|cursor_hud=%s", legacyViewport, cursor, hudUnit, cursorHUD)
		// Draw 的下一步是原生整幀（r5m 的函式，局部黃金已比過像素）：畫到另一張畫布，取 nativeMapPresented 與
		// Draw 對狀態的同一個改動（composeNativeMapFrame 依當下時間重建 nativeMapVGA 等）；鍵只記成功與否。
		admitted := g.nativeMapFrameAdmission(g.storyBG, campaignBattleView)
		presented := false
		if admitted {
			r.mapScreen.Clear()
			presented = g.drawNativeMapFrame(r.mapScreen)
		}
		key += fmt.Sprintf("|map=%t:%t", admitted, presented)
		bannerFallback := g.bannerT > 0 && g.banner != "" && g.font != nil && !g.phaseBannerGlyphsAvailable() &&
			!legacyViewport && campaignBattleView
		overlays := r.layer("overlays", func(screen *ebiten.Image) {
			r5hMapOverlays(g, screen, presented, legacyViewport, campaignBattleView)
		})
		moved := g.movementPanelDraw != nil
		if moved {
			r.stats["move_panel"]++
		}
		key += fmt.Sprintf("|move=%t|banner_fallback=%t|overlays=%s", moved, bannerFallback, overlays)
		ttfReturn := r5hTTFReturns(g)
		key += fmt.Sprintf("|ttf_return=%t", ttfReturn)
		if ttfReturn {
			return key
		}
		transient := g.transientUI
		endturnUI := g.nativeSystemEndTurnUI != nil
		inmap := g.nativeActionOverlayInMapFrame(presented)
		info := g.nativeSystemInfoUI != nil
		status := g.nativePlayerStatus != nil
		completed := true
		command := r.layer("command", func(screen *ebiten.Image) {
			completed = r5hCommandUI(g, screen, presented)
		})
		if endturnUI {
			r.stats["endturn_ui"]++
		}
		if inmap {
			r.stats["overlay_inmap"]++
		}
		return key + fmt.Sprintf("|transient=%t|endturn_ui=%t|inmap=%t|info=%t|status=%t|command_ui=%t|overlay_drawn=%t|command=%s",
			transient, endturnUI, inmap, info, status, completed, g.actionOverlayDrawn, command)
	}
	return branch
}

func (r *r5hLayerRecorder) observe(g *Game) {
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
	r.rows = append(r.rows, r5hLayerRow{Capture: index, Count: 1, UpdateFrame: g.frame, Node: goldenFrameNode(g), Key: key})
}

func (r *r5hLayerRecorder) finish() {
	seq := sha256.Sum256(r.seq)
	manifest := map[string]any{
		"schema":           "fd2_rust_r5h_layers/1",
		"plan":             "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
		"producer":         "remake/cmd/fd2/rust_golden_r5h_layers_test.go（tools/rust_r5h_layers.sh）",
		"scope":            "第一章受控重播的每個 pump 幀（同 TestRustGoldenFrames chapter 模式）",
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
	if err := os.WriteFile(filepath.Join(r.out, "r5h-layers.json"), append(raw, '\n'), 0o644); err != nil {
		r.t.Fatal(err)
	}
	r.t.Logf("r5h layers: captures=%d rows=%d branches=%v stats=%v sequence=%x",
		r.capture, len(r.rows), r.branches, r.stats, seq)
}

func TestRustGoldenR5hLayers(t *testing.T) {
	out := os.Getenv("FD2_GOLDEN_R5H_LAYERS_OUT")
	if out == "" {
		t.Skip("需要 FD2_GOLDEN_R5H_LAYERS_OUT（R5 r5h 局部黃金，只由 tools/rust_r5h_layers.sh 執行）")
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
	rec := &r5hLayerRecorder{t: t, probe: probe, clock: goldenFrameRecorder{t: t}, out: out,
		screen: ebiten.NewImage(logicalW, logicalH), mapScreen: ebiten.NewImage(logicalW, logicalH),
		pix:      make([]byte, logicalW*logicalH*4),
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
