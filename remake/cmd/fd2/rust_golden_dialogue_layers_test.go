package main

// R5 階段 2 r5d 組（docs/goal/121）的 Go 局部黃金工具，只輸出資料。未設
// FD2_GOLDEN_DIALOGUE_LAYERS_OUT 時略過，不改任何遊戲行為；由 tools/rust_r5_dialogue_layers.sh
// 在 Docker／Xvfb 內呼叫。
//
// 第一章 chapter 畫格的對白段多半疊在正規化地圖層與 HUD 上（別組），整張畫格不能單獨驗對話與文字組的
// 函式。做法同 rust_golden_map_layers_test.go：照 TestRustGoldenFrames 的 chapter 模式原樣跑第一章
// 受控重播（同一個 ch01FrameObserver 掛點、同一個 60 Hz 虛擬時鐘與 rand.Seed(4)），每個 pump 幀
// 不呼叫整個 Draw，只照 Draw 的分支順序（goldenMapLayersBranch）在走到尾段時呼叫
// drawNativeStoryDialogue，畫到 Fill 黑的 640×400 畫布再讀回：
//
//   - 每一次擷取都記嘴型狀態（故事對白 mouthState 的 Open／Countdown、確認框嘴型的擁有者／Open／
//     Countdown）。兩者都取自 math/rand 全域來源，逐幀相同才代表 Rust 的全域亂數取用順序與 Go 相同。
//   - drawNativeStoryDialogue 回 true 時另記對白狀態（收框、開框、穩定頁、頁、進度、說話幀、等鍵箭頭
//     計數）、loadErr 與畫布的 RGBA SHA-256。
//
// 每個 pump 幀開頭照 Draw 開頭清 nativeMapPresented／movementPanelDraw／nativeInterfaceHUDVisible（忠實
// 模式下 Draw 開頭其餘欄位都是 nil）。Draw 裡其他分支的流程旗標照 R4 由 ackPresents 承認，與 Rust 無畫面
// 重播相同。

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
)

type dialogueLayerRecorder struct {
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
}

// goldenDialogueMouthKey 是每次擷取都記的嘴型狀態（全域亂數的消費者）。
func goldenDialogueMouthKey(g *Game) string {
	return fmt.Sprintf("mouth=%v:%d|confirm=%v:%v:%d", g.mouthState.Open, g.mouthState.Countdown,
		g.nativeConfirmationMouthOwner != nil, g.nativeConfirmationMouth.Open, g.nativeConfirmationMouth.Countdown)
}

// goldenDialogueStage 照 drawNativeStoryDialogue 的分支順序分類這一幀（只讀欄位），並回傳穩定頁的三個
// 合成條件：說話嘴型（ComposeNativeStoryDialogueMouthFrame）、等鍵張嘴（nativeDialogueMouthOpen）、
// 等鍵箭頭（ComposeNativeStoryDialogueWaitArrow）。
func goldenDialogueStage(g *Game) (stage string, speaking, mouthOpen, arrow bool) {
	switch {
	case g.nativeDialogueClosingLive:
		return "closing", false, false, false
	case g.dlgPhase == 2:
		return "opening", false, false, false
	case g.dlgPhase != 0:
		return "phase", false, false, false
	case g.dlgPage < 0 || g.dlgPage >= len(g.nativeDialogueProgressive) || len(g.nativeDialogueProgressive[g.dlgPage]) == 0:
		return "empty", false, false, false
	}
	frames := g.nativeDialogueProgressive[g.dlgPage]
	progress := g.nativeDialogueProgress
	if progress < 0 {
		progress = 0
	}
	if progress >= len(frames) {
		progress = len(frames) - 1
	}
	speaking = g.localeID == "zh-Hant" && progress > 0 && progress < len(frames)-1 &&
		g.nativeDialogueSpeakingFrame > 0 && len(g.nativeDialoguePortraits) == 4
	mouthOpen = g.mouthOpen && progress == len(frames)-1 && g.dlgPage < len(g.nativeDialogueMouthOpen)
	lastPageArrow := g.nativeLevelUpDialogue != nil && g.nativeLevelUpDialogue.awaitLastKey &&
		g.nativeLevelUpDialogue.closeTimer == 0
	arrow = g.nativeStoryDialogueAtInputWait() && (g.dlgPage+1 < len(g.nativeDialogueProgressive) || lastPageArrow)
	return "steady", speaking, mouthOpen, arrow
}

func (r *dialogueLayerRecorder) layers(g *Game) string {
	g.nativeMapPresented = false
	g.movementPanelDraw = nil
	g.nativeInterfaceHUDVisible = false
	mouth := goldenDialogueMouthKey(g)
	branch := goldenMapLayersBranch(g)
	r.branches[branch]++
	if branch != "tail" {
		return branch + "|" + mouth
	}
	if g.modernStoryPortraits != nil {
		r.t.Fatal("局部黃金只涵蓋忠實模式（modernStoryPortraits 必須是 nil）")
	}
	stage, speaking, mouthOpen, arrow := goldenDialogueStage(g)
	state := fmt.Sprintf("stage=%s|closing_t=%d|dlg_t=%d|page=%d|progress=%d|speaking=%d:%v|mouth_open=%v|arrow=%v:%d",
		stage, g.nativeDialogueClosingT, g.dlgT, g.dlgPage, g.nativeDialogueProgress,
		g.nativeDialogueSpeakingFrame, speaking, mouthOpen, arrow, g.nativeDialogueArrowTicks)
	errBefore := g.loadErr
	r.screen.Clear()
	r.screen.Fill(color.RGBA{A: 0xff})
	drawn := g.drawNativeStoryDialogue(r.screen)
	if !drawn {
		return "tail|dialogue=false|" + mouth
	}
	r.stats["dialogue_drawn"]++
	r.stats["dialogue_"+stage]++
	if speaking {
		r.stats["dialogue_speaking_mouth"]++
	}
	if mouthOpen {
		r.stats["dialogue_wait_mouth_open"]++
	}
	if arrow {
		r.stats["dialogue_wait_arrow"]++
	}
	loadErr := ""
	if g.loadErr != errBefore {
		loadErr = g.loadErr
		r.stats["dialogue_load_err"]++
	}
	r.screen.ReadPixels(r.pix)
	return fmt.Sprintf("tail|dialogue=true|%s|err=%q|rgba=%s|%s", state, loadErr, mapLayerSHA(r.pix), mouth)
}

func (r *dialogueLayerRecorder) observe(g *Game) {
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

func (r *dialogueLayerRecorder) finish() {
	seq := sha256.Sum256(r.seq)
	manifest := map[string]any{
		"schema":           "fd2_rust_r5_dialogue_layers/1",
		"plan":             "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
		"producer":         "remake/cmd/fd2/rust_golden_dialogue_layers_test.go（tools/rust_r5_dialogue_layers.sh）",
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
	if err := os.WriteFile(filepath.Join(r.out, "dialogue-layers.json"), append(raw, '\n'), 0o644); err != nil {
		r.t.Fatal(err)
	}
	r.t.Logf("dialogue layers: captures=%d rows=%d branches=%v stats=%v sequence=%x",
		r.capture, len(r.rows), r.branches, r.stats, seq)
}

func TestRustGoldenDialogueLayers(t *testing.T) {
	out := os.Getenv("FD2_GOLDEN_DIALOGUE_LAYERS_OUT")
	if out == "" {
		t.Skip("需要 FD2_GOLDEN_DIALOGUE_LAYERS_OUT（R5 r5d 局部黃金，只由 tools/rust_r5_dialogue_layers.sh 執行）")
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
	rec := &dialogueLayerRecorder{t: t, probe: probe, clock: goldenFrameRecorder{t: t}, out: out,
		screen: ebiten.NewImage(logicalW, logicalH), pix: make([]byte, logicalW*logicalH*4),
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
