package main

// R5 階段 2 r5b 組（docs/goal/121，戰鬥演出）的 Go 局部黃金工具，只輸出資料。
// 未設 FD2_GOLDEN_R5B_LAYERS_OUT 時略過，不改任何遊戲行為；由 tools/rust_r5b_layers.sh 在
// Docker／Xvfb 內呼叫。
//
// 只有 chapter 模式（本組的函式在 opening 模式沒有執行到）：掛點、60 Hz 虛擬時鐘與 rand.Seed(4)
// 同 TestRustGoldenFrames chapter 模式（共用 pump 的 ch01FrameObserver）。每次擷取不一律呼叫整個
// Draw，依 Draw（main.go:9398..9805）的 early-return 順序分支：
//
//   - ch28／idle／atk：nativeCh28PostPresent、nativeAIIdleRecovery、atk 三個分支。這三個分支只有
//     Fill 黑、本組的 drawNativeCh28PostPresent／drawNativeAIIdleRecovery／drawBattleScene（第一章
//     都在 drawNativePhysicalPrelude 返回）與 defer 的 DAC 白／黑 Fill，沒有別組圖層，所以呼叫
//     **整個** g.Draw，記整張 640×400 雜湊與 Draw 之後的工作狀態，可與階段 0 chapter 黃金同一
//     擷取點直接比。
//   - ramp、spawn：調色盤斜坡（r5p）與登場轉場（r5m）也是只有自己一層的分支，同樣呼叫整個 Draw，
//     只為了照 Draw 留下狀態改動（鍵另記雜湊）。
//   - tail：照 Draw 把原生整幀（r5m）畫到另一張畫布，只為了留下 composeNativeMapFrame 對
//     nativeMapVGA／nativeMapDAC 的重建（AI 閒置回收的調色盤取自 nativeMapDAC）；鍵只記成功與否。
//     其他圖層（HUD、選單、對白、TTF）屬別組，不畫。
//   - early：其餘 early-return 分支（含 HUD 組的 END 回收），不畫。
//
// 每次擷取開頭照 Draw 開頭清 nativeMapPresented／movementPanelDraw／nativeInterfaceHUDVisible。

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
)

type r5bLayerRow struct {
	Capture     int    `json:"capture"`
	Count       int    `json:"count"`
	UpdateFrame int    `json:"update_frame"`
	Node        string `json:"node"`
	Key         string `json:"key"`
}

type r5bLayerRecorder struct {
	t         *testing.T
	probe     *goldenFrameProbe
	clock     goldenFrameRecorder // 只借用 pinClock（同一個虛擬時鐘）
	out       string
	screen    *ebiten.Image
	mapScreen *ebiten.Image
	pix       []byte
	rows      []r5bLayerRow
	seq       []byte
	capture   int
	branches  map[string]int
	stats     map[string]int
}

// r5bLayerBranch：Draw（main.go:9398..9805）的 early-return 順序，只讀欄位。
func r5bLayerBranch(g *Game) string {
	switch {
	case g.startupBlocked, g.nativeDefeat != nil, g.titlePhase != "", g.nativeEnding != nil,
		g.nativeCh20SkyKey != nil, g.nativeCh23Loop != nil, g.native2189A != nil, g.nativeUnitPresent != nil:
		return "early"
	case g.nativeCh28PostPresent != nil:
		return "ch28"
	case g.transitionReveal != nil, g.m == nil, g.indexedTransition != nil,
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
	case g.nativePalettePulse != nil:
		return "early"
	case g.spawnIntroTransition != nil:
		return "spawn"
	case g.nativeTurnStaging != nil && g.nativeTurnStaging.indexed, g.nativeFieldEvent61 != nil,
		g.nativeEndTurnRecovery != nil:
		return "early"
	case g.nativeAIIdleRecovery != nil:
		return "idle"
	case g.atk != nil:
		return "atk"
	}
	return "tail"
}

func r5bSHA(pix []byte) string {
	sum := sha256.Sum256(pix)
	return hex.EncodeToString(sum[:])
}

// r5bDraw：清空後呼叫整個 Draw，回傳整張 640×400 的雜湊。
func (r *r5bLayerRecorder) r5bDraw(g *Game) string {
	r.screen.Clear()
	g.Draw(r.screen)
	r.screen.ReadPixels(r.pix)
	return r5bSHA(r.pix)
}

// r5bAtkState：Draw 之後的物理演出狀態（Rust 端以同一格式組字串）。
func r5bAtkState(g *Game) string {
	a := g.atk
	if a == nil {
		return "atk=nil"
	}
	s := a.nativeScene
	if s == nil {
		return "scene=nil"
	}
	body := "none"
	if p := s.body; p != nil {
		body = fmt.Sprintf("%d/%d,drawn=%t,image=%t,last=%d,wait=%t,ramp=%t",
			p.index, len(p.jobs), p.drawn, p.image != nil, p.lastPresent, p.returnWaitMillis > 0, p.returnRampStarted)
	}
	return fmt.Sprintf("prelude=%d/%d|prelude_drawn=%t|lead=%d/%d|lead_drawn=%t|counter=%t|body=%s",
		s.preludeFrame, len(s.preludeImages), s.preludeDrawn, s.leadFrame, len(s.leadImages), s.leadDrawn,
		s.counterActive, body)
}

// layers 在 Ebiten 的 Update 內執行：回傳這一幀的比對鍵。
func (r *r5bLayerRecorder) layers(g *Game) string {
	g.nativeMapPresented = false
	g.movementPanelDraw = nil
	g.nativeInterfaceHUDVisible = false
	branch := r5bLayerBranch(g)
	r.branches[branch]++
	switch branch {
	case "ch28":
		if g.modernStoryPortraits != nil {
			r.t.Fatal("局部黃金只涵蓋忠實模式（modernStoryPortraits 必須是 nil）")
		}
		j := g.nativeCh28PostPresent
		frame, frames := j.frame, len(j.vgaFrames)
		rgba := r.r5bDraw(g)
		r.stats["ch28_frames"]++
		state := "job=nil"
		if j := g.nativeCh28PostPresent; j != nil {
			state = fmt.Sprintf("drawn=%t|presented=%d|hd=%t", j.drawn, j.presentedFrame, j.presentedHD)
		}
		return fmt.Sprintf("ch28|frame=%d/%d|%s|load_err=%t|rgba=%s", frame, frames, state, g.loadErr != "", rgba)
	case "idle":
		j := g.nativeAIIdleRecovery
		frame, frames := j.frame, len(j.frames)
		dac := len(g.nativeMapDAC)
		rgba := r.r5bDraw(g)
		r.stats["idle_frames"]++
		state := "job=nil"
		if j := g.nativeAIIdleRecovery; j != nil {
			state = fmt.Sprintf("drawn=%t", j.drawn)
		}
		return fmt.Sprintf("idle|frame=%d/%d|dac=%d|%s|load_err=%t|rgba=%s", frame, frames, dac, state, g.loadErr != "", rgba)
	case "atk":
		if g.modernStoryPortraits != nil {
			r.t.Fatal("局部黃金只涵蓋忠實模式（modernStoryPortraits 必須是 nil）")
		}
		before := r5bAtkState(g)
		rgba := r.r5bDraw(g)
		after := r5bAtkState(g)
		r.stats["atk_frames"]++
		if s := g.atk.nativeScene; s != nil {
			if s.preludeFrame < len(s.preludeImages) {
				r.stats["atk_prelude"]++
			} else if s.body != nil {
				r.stats["atk_body"]++
			}
		}
		return fmt.Sprintf("atk|before=%s|after=%s|load_err=%t|rgba=%s", before, after, g.loadErr != "", rgba)
	case "ramp", "spawn":
		return branch + "|rgba=" + r.r5bDraw(g)
	case "tail":
		campaignBattleView := g.camp == nil || (g.camp.Node() != nil && g.camp.Node().Type == "battle")
		if g.nativeMapFrameAdmission(g.storyBG, campaignBattleView) {
			r.mapScreen.Clear()
			return fmt.Sprintf("tail|map_ok=%t", g.drawNativeMapFrame(r.mapScreen))
		}
		return "tail"
	}
	return branch
}

func (r *r5bLayerRecorder) observe(g *Game) {
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
	r.rows = append(r.rows, r5bLayerRow{Capture: index, Count: 1, UpdateFrame: g.frame, Node: goldenFrameNode(g), Key: key})
}

func (r *r5bLayerRecorder) finish() {
	seq := sha256.Sum256(r.seq)
	manifest := map[string]any{
		"schema":           "fd2_rust_r5b_layers/1",
		"mode":             "chapter",
		"plan":             "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
		"producer":         "remake/cmd/fd2/rust_golden_r5b_layers_test.go（tools/rust_r5b_layers.sh）",
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
	if err := os.WriteFile(filepath.Join(r.out, "r5b-layers.json"), append(raw, '\n'), 0o644); err != nil {
		r.t.Fatal(err)
	}
	r.t.Logf("r5b layers: captures=%d rows=%d branches=%v stats=%v sequence=%x",
		r.capture, len(r.rows), r.branches, r.stats, seq)
}

func TestRustGoldenR5bLayers(t *testing.T) {
	out := os.Getenv("FD2_GOLDEN_R5B_LAYERS_OUT")
	if out == "" {
		t.Skip("需要 FD2_GOLDEN_R5B_LAYERS_OUT（R5 r5b 局部黃金，只由 tools/rust_r5b_layers.sh 執行）")
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
	rec := &r5bLayerRecorder{t: t, probe: probe, clock: goldenFrameRecorder{t: t}, out: out,
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
