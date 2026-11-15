package main

// R5 階段 0（docs/goal/121）的 Go 黃金畫格工具，只輸出資料。未設
// FD2_GOLDEN_FRAMES_OUT 時略過，不改任何遊戲行為；由 tools/rust_r5_golden_frames.sh
// 在 Docker／Xvfb 內呼叫。
//
// 兩種模式（FD2_GOLDEN_FRAMES_MODE）：
//
//   - chapter：照 TestChapterParityReplay 的第一章受控流程（建構槽、原版
//     original-full-r4、FD2_PARITY_ENTRY=start，環境變數由腳本設定）原樣重播，只掛上
//     共用 pump 既有的 ch01FrameObserver：每個 pump 幀在上一個 Update 之後、下一個
//     ackPresents 之前呼叫一次 g.Draw 到離屏 640×400 影像並 ReadPixels。重播邏輯本身
//     不動；pump 以外的迴圈（開場動畫、driveStory、settleTown、closeRing 等）沒有
//     鉤子，那些 Update 的畫面不擷取，跳過的 Update 數記在清單的 gaps。
//   - opening：自己驅動「標題開場動畫 → 選單 → START → 序章 → 進入 battle_ch01」，
//     步驟與 startFromNormalTitle、settleTown、driveStory 相同，每個 Update 之後擷取。
//
// 牆鐘：遊戲裡所有隨牆鐘前進的時間都經過 g.playerClock.sample（gameNow 與地圖合成的
// BIOS 取樣）。擷取時把 playerClock.wall 設在遠未來，sample 就只回傳 playerClock.game；
// 每次擷取再依 g.frame 的增量把 game 推進 1/60 秒（乘遊戲速度），等於 60 Hz 虛擬時鐘。
// chapter 模式只能在 pump 幀推進，pump 以外的 Update 期間時鐘不動，到下一個 pump 幀
// 才依 g.frame 補上。
//
// 亂數：遊戲的 g.rng 由 FD2_SEED 決定；嘴型動畫另用 math/rand 全域來源，這裡以 rand.Seed
// 固定（goldenFrameGlobalRandSeed）。

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// goldenFrameProbe 讓 Ebiten 進入執行狀態（ReadPixels 需要已啟動的 graphics driver），
// 並讓每次擷取都走完一個正常的 Ebiten 畫格：Draw＋ReadPixels 在 probe 的 Update 裡執行，
// 回傳後 Ebiten 做 EndFrame／BeginFrame（釋放 Draw 每幀新建的影像），到下一個 Update
// 開頭才放測試 goroutine 繼續。測試 goroutine 跑遊戲邏輯時，Ebiten 一律停在 Update 的
// select 上，兩邊不會同時使用 Ebiten。
//
// 只停在第一個 Update、從不結束畫格的做法會讓 Draw 新建的影像無法釋放（第一章重播約每次
// 擷取 3.6 MB，2048 次擷取後 RSS 8 GB 被砍）。
type goldenFrameProbe struct {
	started chan struct{}
	work    chan func()
	parked  chan struct{}
	stop    chan struct{}
	once    sync.Once
	pending bool
}

func (p *goldenFrameProbe) Update() error {
	p.once.Do(func() { close(p.started) })
	if p.pending {
		p.pending = false
		p.parked <- struct{}{}
	}
	select {
	case f := <-p.work:
		f()
		p.pending = true
		return nil
	case <-p.stop:
		return ebiten.Termination
	}
}

// run 在 Ebiten 的 Update 內執行 f，等這個畫格結束、下一個 Update 開始後才回傳。
func (p *goldenFrameProbe) run(f func()) {
	p.work <- f
	<-p.parked
}

func (p *goldenFrameProbe) Draw(*ebiten.Image)         {}
func (p *goldenFrameProbe) Layout(int, int) (int, int) { return logicalW, logicalH }

type goldenFrameRow struct {
	Capture     int    `json:"capture"`
	Count       int    `json:"count"`
	UpdateFrame int    `json:"update_frame"`
	Checkpoints int    `json:"checkpoints_written"`
	SHA256      string `json:"rgba_sha256"`
	File        string `json:"file"`
	Node        string `json:"node"`
	NodeType    string `json:"node_type"`
	TitlePhase  string `json:"title_phase,omitempty"`
	Turn        int    `json:"turn"`
	Dialog      int    `json:"dialog"`
	AIBusy      bool   `json:"ai_busy,omitempty"`
	Ring        bool   `json:"ring,omitempty"`
	Fade        bool   `json:"fade,omitempty"`
	Path        string `json:"draw_path"`
}

type goldenFrameGap struct {
	AfterCapture int    `json:"after_capture"`
	Updates      int    `json:"uncaptured_updates"`
	FromFrame    int    `json:"from_update_frame"`
	ToFrame      int    `json:"to_update_frame"`
	Node         string `json:"node"`
}

type goldenFrameRecorder struct {
	t       *testing.T
	probe   *goldenFrameProbe
	out     string
	screen  *ebiten.Image
	pix     []byte
	rows    []goldenFrameRow
	gaps    []goldenFrameGap
	seq     []byte // 每次擷取的 RGBA SHA-256 串接，算整段序列雜湊
	capture int

	pinned        bool
	clockMode     string
	lastFrame     int
	virtualStart  time.Time
	virtualFrames int

	paths map[string]int // 每次擷取的 Draw 主路徑計數（goldenFrameDrawPath）

	cpPath  string
	cpFile  *os.File
	cpLines int
	cpBuf   []byte
}

// goldenFrameFarFuture 讓 playerGameClock.sample 的 delta 恆為負，只回傳 game。
var goldenFrameFarFuture = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

const goldenFrameTick = time.Second / 60

// goldenFrameGlobalRandSeed 固定 math/rand 全域來源（嘴型動畫）；取 FD2_SEED 同值。
const goldenFrameGlobalRandSeed = 4

func (r *goldenFrameRecorder) pinClock(g *Game) {
	if !r.pinned {
		r.pinned = true
		r.lastFrame = g.frame
		if g.playerClock.wall.IsZero() {
			// 還沒有任何取樣：從固定原點開始，絕對值也可重現。
			g.playerClock.game = time.Unix(1<<30, 0).UTC()
			r.clockMode = "virtual-60hz-fixed-origin"
		} else {
			// 已有牆鐘取樣（chapter 模式在開場動畫之後才拿得到 g）：沿用當時的 game，
			// 各 BIOS 時鐘只看差值，之後的差值固定。
			r.clockMode = "virtual-60hz-from-first-pump"
		}
		r.virtualStart = g.playerClock.game
		g.playerClock.wall = goldenFrameFarFuture
		return
	}
	d := g.frame - r.lastFrame
	r.lastFrame = g.frame
	if d < 0 {
		r.t.Fatalf("g.frame 倒退：%d", d)
	}
	r.virtualFrames += d
	g.playerClock.wall = goldenFrameFarFuture
	g.playerClock.game = g.playerClock.game.Add(time.Duration(d) * goldenFrameTick *
		time.Duration(normalizePlayerSpeed(g.playerClock.speed)))
}

func (r *goldenFrameRecorder) checkpointsWritten() int {
	if r.cpPath == "" {
		return 0
	}
	if r.cpFile == nil {
		f, err := os.Open(r.cpPath)
		if err != nil {
			return 0
		}
		r.cpFile = f
	}
	if r.cpBuf == nil {
		r.cpBuf = make([]byte, 64<<10)
	}
	for {
		n, err := r.cpFile.Read(r.cpBuf)
		for _, b := range r.cpBuf[:n] {
			if b == '\n' {
				r.cpLines++
			}
		}
		if err == io.EOF || n == 0 {
			break
		}
		if err != nil {
			r.t.Fatal(err)
		}
	}
	return r.cpLines
}

// observe 在一個 Update 之後呼叫：推進虛擬時鐘、Draw、讀回、去重寫檔。
func (r *goldenFrameRecorder) observe(g *Game) {
	prev := r.lastFrame
	r.pinClock(g)
	if r.capture > 0 && g.frame-prev > 1 {
		r.gaps = append(r.gaps, goldenFrameGap{AfterCapture: r.capture - 1, Updates: g.frame - prev - 1,
			FromFrame: prev, ToFrame: g.frame, Node: goldenFrameNode(g)})
	}
	if r.capture%goldenFrameGCInterval == 0 && r.capture > 0 {
		// Draw 每幀新建的 ebiten.Image 只在 Go GC 跑 finalizer 後才釋放 GPU（llvmpipe 是
		// 主機記憶體）端的貼圖；Go 堆積小，GC 很少觸發。定期 GC 只影響釋放時點，不碰遊戲狀態。
		runtime.GC()
		if r.capture%(goldenFrameGCInterval*32) == 0 {
			r.t.Logf("golden frames: capture=%d update_frame=%d rss=%s", r.capture, g.frame, goldenFrameRSS())
		}
	}
	r.probe.run(func() {
		r.screen.Clear()
		g.Draw(r.screen)
		r.screen.ReadPixels(r.pix)
	})
	path := goldenFrameDrawPath(g)
	if r.paths == nil {
		r.paths = map[string]int{}
	}
	r.paths[path]++
	sum := sha256.Sum256(r.pix)
	r.seq = append(r.seq, sum[:]...)
	hash := hex.EncodeToString(sum[:])
	index := r.capture
	r.capture++
	if n := len(r.rows); n > 0 && r.rows[n-1].SHA256 == hash {
		r.rows[n-1].Count++
		return
	}
	row := goldenFrameRow{Capture: index, Count: 1, UpdateFrame: g.frame, Checkpoints: r.checkpointsWritten(),
		SHA256: hash, Node: goldenFrameNode(g), TitlePhase: g.titlePhase, Dialog: len(g.dialog),
		AIBusy: g.aiBusy, Ring: g.ring, Fade: g.fade != nil, Path: path}
	if g.camp != nil && g.camp.Node() != nil {
		row.NodeType = g.camp.Node().Type
	}
	if g.st != nil {
		row.Turn = g.st.Turn
	}
	row.File = fmt.Sprintf("frames/%06d.png", index)
	f, err := os.Create(filepath.Join(r.out, row.File))
	if err != nil {
		r.t.Fatal(err)
	}
	img := &image.RGBA{Pix: r.pix, Stride: logicalW * 4, Rect: image.Rect(0, 0, logicalW, logicalH)}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		r.t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		r.t.Fatal(err)
	}
	r.rows = append(r.rows, row)
}

const goldenFrameGCInterval = 32

func goldenFrameRSS() string {
	raw, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return "?"
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "VmRSS:"))
		}
	}
	return "?"
}

// goldenFrameDrawPath 在 Draw 之後依 Game 狀態分類這一幀走的主路徑，只讀欄位：
// title（drawTitle）、native-map（drawNativeMapFrame 已蓋上整幀索引畫面）、
// story-viewport（storyBG 或戰場事件：320×200 正規化圖塊／sprite 離屏放大）、
// battle-normalized（戰場但沒有原生整幀：正規化圖塊、單位與 RGBA 疊層可見）、
// fullscreen-native（Draw 前段 early-return 的原生全螢幕呈現，或非戰場節點）。
// 後綴 +ttf-msg 表示 Draw 尾段以 TTF 畫了 g.msg，+fade 表示舊 fade 黑色疊層。
func goldenFrameDrawPath(g *Game) string {
	path := "fullscreen-native"
	battleView := g.camp == nil || (g.camp.Node() != nil && g.camp.Node().Type == "battle")
	switch {
	case g.titlePhase != "":
		path = "title"
	case g.nativeMapPresented:
		path = "native-map"
	case g.m != nil && g.atk == nil && (g.storyBG || g.battleEvent != nil) && g.nativeEndTurnRecovery == nil &&
		g.nativeUnitPresent == nil && g.indexedTransition == nil && g.transitionReveal == nil:
		path = "story-viewport"
	case g.m != nil && g.atk == nil && battleView && g.nativeEndTurnRecovery == nil && g.nativeUnitPresent == nil &&
		g.indexedTransition == nil && g.transitionReveal == nil:
		path = "battle-normalized"
	}
	if g.titlePhase == "" && g.font != nil && g.shouldDrawTransientBattleMessage() && strings.TrimSpace(g.msg) != "" {
		path += "+ttf-msg"
	}
	if g.titlePhase == "" && g.fade != nil {
		path += "+fade"
	}
	return path
}

func goldenFrameNode(g *Game) string {
	if g.camp == nil {
		return ""
	}
	return g.camp.NodeID()
}

func (r *goldenFrameRecorder) finish(mode string, extra map[string]any) {
	r.t.Helper()
	if r.cpFile != nil {
		r.cpFile.Close()
	}
	seq := sha256.Sum256(r.seq)
	distinct := map[string]bool{}
	for _, row := range r.rows {
		distinct[row.SHA256] = true
	}
	ebitenVersion := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/hajimehoshi/ebiten/v2" {
				ebitenVersion = dep.Version
			}
		}
	}
	gapUpdates := 0
	for _, gap := range r.gaps {
		gapUpdates += gap.Updates
	}
	manifest := map[string]any{
		"schema":            "fd2_rust_r5_golden_frames/1",
		"mode":              mode,
		"plan":              "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
		"producer":          "remake/cmd/fd2/rust_golden_frames_test.go（tools/rust_r5_golden_frames.sh）",
		"go_version":        runtime.Version(),
		"ebiten_version":    ebitenVersion,
		"logical_size":      []int{logicalW, logicalH},
		"pixel_format":      "RGBA8，Ebiten ReadPixels 的預乘 alpha 值（畫面不透明時與一般 RGBA 相同）",
		"clock":             r.clockMode,
		"global_rand_seed":  goldenFrameGlobalRandSeed,
		"virtual_frames":    r.virtualFrames,
		"captures":          r.capture,
		"changed_frames":    len(r.rows),
		"distinct_frames":   len(distinct),
		"sequence_sha256":   hex.EncodeToString(seq[:]),
		"uncaptured_gaps":   len(r.gaps),
		"uncaptured_update": gapUpdates,
		"draw_paths":        r.paths,
		"draw_path_note":    "goldenFrameDrawPath 依 Draw 後的 Game 狀態推定，是近似分類；+ttf-msg 只代表條件成立，Draw 尾段是否真的畫到畫面以覆蓋率為準",
		"rows":              r.rows,
		"gaps":              r.gaps,
	}
	for k, v := range extra {
		manifest[k] = v
	}
	raw, err := json.MarshalIndent(manifest, "", " ")
	if err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.out, "frames.json"), append(raw, '\n'), 0o644); err != nil {
		r.t.Fatal(err)
	}
	r.t.Logf("golden frames %s: captures=%d changed=%d distinct=%d gaps=%d/%d sequence=%x",
		mode, r.capture, len(r.rows), len(distinct), len(r.gaps), gapUpdates, seq)
}

func TestRustGoldenFrames(t *testing.T) {
	out := os.Getenv("FD2_GOLDEN_FRAMES_OUT")
	if out == "" {
		t.Skip("需要 FD2_GOLDEN_FRAMES_OUT（R5 黃金畫格，只由 tools/rust_r5_golden_frames.sh 執行）")
	}
	mode := os.Getenv("FD2_GOLDEN_FRAMES_MODE")
	if mode != "chapter" && mode != "opening" {
		t.Fatalf("FD2_GOLDEN_FRAMES_MODE=%q；只允許 chapter 或 opening", mode)
	}
	if err := os.MkdirAll(filepath.Join(out, "frames"), 0o755); err != nil {
		t.Fatal(err)
	}
	// 對白嘴型（native_story_dialogue.go stepDialogueMouth）與確認框嘴型的種子取自 math/rand
	// 的全域來源；Go 1.20 起它在行程啟動時隨機播種，正式程式沒有呼叫 rand.Seed，同輸入兩次
	// 重播的嘴型開合時點會不同。擷取前固定全域種子；章重播的檢查點不受影響（R4 黃金本來就
	// 在隨機種子下逐位元組相同）。
	rand.Seed(goldenFrameGlobalRandSeed) //nolint:staticcheck // 只有 rand.Seed 能固定全域來源
	probe := &goldenFrameProbe{started: make(chan struct{}), stop: make(chan struct{}),
		work: make(chan func()), parked: make(chan struct{})}
	// 擷取的畫格不需要等 60 Hz；遊戲邏輯的時間由虛擬時鐘決定，與 Ebiten 的 TPS 無關。
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
	rec := &goldenFrameRecorder{t: t, probe: probe, out: out, screen: ebiten.NewImage(logicalW, logicalH),
		pix: make([]byte, logicalW*logicalH*4)}
	switch mode {
	case "chapter":
		parity := os.Getenv("FD2_PARITY_OUT")
		if parity == "" || os.Getenv("FD2_PARITY_CAPTURE_FRAMES") != "0" || os.Getenv("FD2_PARITY_ENTRY") != "start" ||
			os.Getenv("FD2_PARITY_CHAPTER") != "1" {
			t.Fatal("chapter 模式需要與 tools/rust_r4_ch01_coverage.sh 相同的 FD2_PARITY_* 設定（第一章、START、不擷取索引畫格）")
		}
		rec.cpPath = filepath.Join(parity, "checkpoints.jsonl")
		saved := ch01FrameObserver
		ch01FrameObserver = rec.observe
		defer func() { ch01FrameObserver = saved }()
		TestChapterParityReplay(t)
		if t.Failed() {
			return
		}
		rec.finish(mode, map[string]any{
			"scope": "第一章受控重播（同 tools/rust_r4_ch01_coverage.sh）的每個 pump 幀；pump 以外迴圈的 Update 見 gaps",
		})
	case "opening":
		rec.finish(mode, goldenFramesOpening(t, rec))
	}
}

// goldenFramesOpening 與 TestChapterParityReplay 的 START 入口同設定、同步驟，但每個
// Update 之後都擷取；停在節點第一次變成 battle_ch01 的那一幀。
func goldenFramesOpening(t *testing.T, rec *goldenFrameRecorder) map[string]any {
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
	// 1. 開場動畫到標題選單（同 startFromNormalTitle）。
	for frame := 0; g.titlePhase != "menu"; frame++ {
		if frame >= journeyTitleFrames || g.titlePhase == "" {
			t.Fatal("正常開場動畫未進標題選單")
		}
		ackPresents(g)
		step()
	}
	mark("title_menu")
	// 2. START（選單第 0 項），等標題結束（同 startFromNormalTitle 的 pump(600)）。
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
	// 3. 同 parityReplay.settleTown。
	for frame := 0; frame < journeyTownSettleMax; frame++ {
		if g.nativeClassUIJob == nil && g.fade == nil && g.transitionReveal == nil &&
			g.indexedTransition == nil && frame > 10 {
			break
		}
		ackPresents(g)
		step()
	}
	// 4. 同 driveStory（停止條件：進入 battle_ch01）。
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
	nodes := strings.Join(j.Nodes, " → ")
	return map[string]any{
		"scope":          "標題開場動畫 → 選單 → START → 序章故事 → 第一次進入 battle_ch01；每個 Update 之後擷取（另含 loadGame 後一張）",
		"phase_captures": phases,
		"story_nodes":    nodes,
		"story_presses":  j.Presses,
	}
}
