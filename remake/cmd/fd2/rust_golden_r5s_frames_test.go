package main

// R5 階段 3（docs/goal/121，代號 r5s）城鎮與設施的整張畫格黃金工具，只輸出資料。未設
// FD2_GOLDEN_R5S_FRAMES_OUT 時略過，不改任何遊戲行為；由 tools/rust_r5_stage3_verify.sh 在 Docker／Xvfb 內呼叫。
//
// 階段 0 的 chapter 模式只在 pump 擷取，城鎮、商店、酒店、整備多半不經 pump。r5c 的局部黃金
// （rust_golden_r5c_layers_test.go）在戰後迴圈、戰後 driveStory 與 settleTown 的每個 Update 之後、城鎮游標移動之後、
// 酒店每次輸入之後另加 gap 擷取點，但只畫本組圖層。本檔用同一批擷取點，每次呼叫整個 g.Draw：
//
//   - pump：同 TestRustGoldenFrames chapter 模式，掛在共用 pump 的 ch01FrameObserver，推進同一個 60 Hz 虛擬時鐘。
//   - gap：本檔末尾「逐字複本」區段與 r5c 相同（TestChapterParityReplay、ensureTown、ensurePostbattleNode、
//     driveStory、settleTown、townProbe、townSave），只換接收者並插入標 `// r5s-insert` 的擷取行，不推進虛擬時鐘。
//     tools/rust_r5c_layers_compare.py --profile r5s --verbatim 每次先核對複本與原檔逐字相同。
//
// 每次擷取先照 Draw 開頭清 nativeMapPresented／movementPanelDraw／nativeInterfaceHUDVisible，以
// goldenMapLayersBranch 記分支；尾段時另把 Draw 的 TTF 段（r5f 的 ttfLayerRecorder.ttfSection，main.go 的逐字複本）
// 畫到 Clear 的畫布，記每次繪字的鍵（ttfLayerDrawKey）與墨跡外框，再 Clear 畫布呼叫整個 g.Draw 取雜湊。gap 擷取與
// 有 TTF 繪字的畫格以雜湊為檔名寫 zlib 壓縮的 RGBA（frames/<sha>.rgba.z），供比對程式逐像素核對差異是否只在
// TTF 墨跡內；其餘 pump 畫格的像素已在階段 0 黃金裡。
//
// 亂數與時鐘同階段 0：rand.Seed(4)、pump 擷取時推進 60 Hz 虛擬時鐘。gap 擷取的 Draw 會照 Draw 改流程旗標，所以比對
// 同時要求兩邊的 checkpoints.jsonl 與 R4 黃金相同。

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type r5sFrameRow struct {
	Capture     int    `json:"capture"`
	Count       int    `json:"count"`
	Pump        int    `json:"pump_capture"`
	UpdateFrame int    `json:"update_frame"`
	Node        string `json:"node"`
	SHA256      string `json:"rgba_sha256"`
	Key         string `json:"key"`
}

type r5sFrameRecorder struct {
	t        *testing.T
	probe    *goldenFrameProbe
	clock    goldenFrameRecorder // 只借用 pinClock（同一個虛擬時鐘）
	ttf      ttfLayerRecorder    // 只借用 ttfSection／fontDraw（r5f 的 TTF 段逐字複本）
	out      string
	screen   *ebiten.Image
	ttfImage *ebiten.Image
	pix      []byte
	ttfPix   []byte
	rows     []r5sFrameRow
	seq      []byte
	capture  int
	pumps    int
	branches map[string]int
	stats    map[string]int
	dumped   map[string]bool
}

// ttfKey 是尾段的 TTF 段：畫到 Clear 的畫布，回傳繪字鍵與墨跡外框（非透明像素的外框，右下不含；沒有墨跡時
// ink=none）。不是尾段回 ttf=-。
func (r *r5sFrameRecorder) ttfKey(g *Game, branch string) string {
	if branch != "tail" {
		return "ttf=-"
	}
	r.ttf.draws = r.ttf.draws[:0]
	r.ttfImage.Clear()
	r.ttf.ttfSection(g, r.ttfImage)
	if len(r.ttf.draws) == 0 {
		return "ttf=0"
	}
	r.stats["ttf_captures"]++
	r.ttfImage.ReadPixels(r.ttfPix)
	ink := [4]int{logicalW, logicalH, 0, 0}
	for y := 0; y < logicalH; y++ {
		for x := 0; x < logicalW; x++ {
			if r.ttfPix[(y*logicalW+x)*4+3] != 0 {
				ink = [4]int{min(ink[0], x), min(ink[1], y), max(ink[2], x+1), max(ink[3], y+1)}
			}
		}
	}
	inkText := "none"
	if ink[2] != 0 {
		inkText = fmt.Sprintf("%d,%d,%d,%d", ink[0], ink[1], ink[2], ink[3])
	}
	parts := make([]string, 0, len(r.ttf.draws))
	for _, d := range r.ttf.draws {
		parts = append(parts, ttfLayerDrawKey(d))
	}
	return fmt.Sprintf("ttf=%d|%s|ink=%s", len(r.ttf.draws), strings.Join(parts, "|"), inkText)
}

// frame 在 Ebiten 的 Update 內執行：清三個本幀旗標、記分支與 TTF 鍵，再 Clear 畫布呼叫整個 g.Draw。
// 第三個回傳值是這次有沒有 TTF 繪字。
func (r *r5sFrameRecorder) frame(g *Game) (string, string, bool) {
	g.nativeMapPresented = false
	g.movementPanelDraw = nil
	g.nativeInterfaceHUDVisible = false
	branch := goldenMapLayersBranch(g)
	r.branches[branch]++
	if branch == "tail" && (g.modernStoryPortraits != nil || g.modernHDText != nil) {
		r.t.Fatal("整張畫格黃金只涵蓋忠實模式（modernStoryPortraits、modernHDText 必須是 nil）")
	}
	ttf := r.ttfKey(g, branch)
	r.screen.Clear()
	g.Draw(r.screen)
	r.screen.ReadPixels(r.pix)
	sha := mapLayerSHA(r.pix)
	return fmt.Sprintf("%s|%s|rgba=%s", branch, ttf, sha), sha, ttf != "ttf=-" && ttf != "ttf=0"
}

func (r *r5sFrameRecorder) dump(sha string) {
	if r.dumped[sha] {
		return
	}
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(r.pix); err != nil {
		r.t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		r.t.Fatal(err)
	}
	dir := filepath.Join(r.out, "frames")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, sha+".rgba.z"), buf.Bytes(), 0o644); err != nil {
		r.t.Fatal(err)
	}
	r.dumped[sha] = true
}

func (r *r5sFrameRecorder) record(g *Game, key, sha string, pump int) {
	r.seq = append(r.seq, key...)
	r.seq = append(r.seq, '\n')
	index := r.capture
	r.capture++
	if n := len(r.rows); n > 0 && r.rows[n-1].Key == key {
		r.rows[n-1].Count++
		return
	}
	r.rows = append(r.rows, r5sFrameRow{Capture: index, Count: 1, Pump: pump, UpdateFrame: g.frame,
		Node: goldenFrameNode(g), SHA256: sha, Key: key})
}

func (r *r5sFrameRecorder) gc() {
	if r.capture%goldenFrameGCInterval == 0 && r.capture > 0 {
		runtime.GC()
	}
}

// observe 是 pump 擷取點（ch01FrameObserver）。
func (r *r5sFrameRecorder) observe(g *Game) {
	r.clock.pinClock(g)
	r.gc()
	var key, sha string
	var ttf bool
	r.probe.run(func() { key, sha, ttf = r.frame(g) })
	if ttf {
		r.dump(sha)
	}
	pump := r.pumps
	r.pumps++
	r.stats["point_pump"]++
	r.record(g, "pump|"+key, sha, pump)
}

// observeGap 是逐字複本裡插入的擷取點：不推進虛擬時鐘。
func (r *r5sFrameRecorder) observeGap(g *Game, point string) {
	r.gc()
	var key, sha string
	r.probe.run(func() { key, sha, _ = r.frame(g) })
	r.dump(sha)
	r.stats["point_"+point]++
	r.record(g, "gap:"+point+"|"+key, sha, -1)
}

func (r *r5sFrameRecorder) finish() {
	seq := sha256.Sum256(r.seq)
	manifest := map[string]any{
		"schema":           "fd2_rust_r5s_frames/1",
		"plan":             "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
		"producer":         "remake/cmd/fd2/rust_golden_r5s_frames_test.go（tools/rust_r5_stage3_verify.sh）",
		"go_version":       runtime.Version(),
		"pixel_format":     "RGBA8，預乘 alpha（Ebiten ReadPixels）；frames/<sha>.rgba.z 是 zlib 壓縮",
		"clock":            r.clock.clockMode,
		"global_rand_seed": goldenFrameGlobalRandSeed,
		"virtual_frames":   r.clock.virtualFrames,
		"captures":         r.capture,
		"pump_captures":    r.pumps,
		"changed_rows":     len(r.rows),
		"dumped_frames":    len(r.dumped),
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
	if err := os.WriteFile(filepath.Join(r.out, "r5s-frames.json"), append(raw, '\n'), 0o644); err != nil {
		r.t.Fatal(err)
	}
	r.t.Logf("r5s frames: captures=%d pump=%d rows=%d dumped=%d branches=%v stats=%v sequence=%x",
		r.capture, r.pumps, len(r.rows), len(r.dumped), r.branches, r.stats, seq)
}

func TestRustGoldenR5sFrames(t *testing.T) {
	out := os.Getenv("FD2_GOLDEN_R5S_FRAMES_OUT")
	if out == "" {
		t.Skip("需要 FD2_GOLDEN_R5S_FRAMES_OUT（R5 階段 3 整張畫格黃金，只由 tools/rust_r5_stage3_verify.sh 執行）")
	}
	parity := os.Getenv("FD2_PARITY_OUT")
	if parity == "" || os.Getenv("FD2_PARITY_CAPTURE_FRAMES") != "0" || os.Getenv("FD2_PARITY_ENTRY") != "start" ||
		os.Getenv("FD2_PARITY_CHAPTER") != "1" {
		t.Fatal("需要與 tools/rust_r4_ch01_coverage.sh 相同的 FD2_PARITY_* 設定（第一章、START、不擷取索引畫格）")
	}
	goldenTTFFontPath(t)
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
	rec := &r5sFrameRecorder{t: t, probe: probe, clock: goldenFrameRecorder{t: t}, ttf: ttfLayerRecorder{t: t}, out: out,
		screen: ebiten.NewImage(logicalW, logicalH), ttfImage: ebiten.NewImage(logicalW, logicalH),
		pix: make([]byte, logicalW*logicalH*4), ttfPix: make([]byte, logicalW*logicalH*4),
		branches: map[string]int{}, stats: map[string]int{}, dumped: map[string]bool{}}
	saved := ch01FrameObserver
	ch01FrameObserver = rec.observe
	defer func() { ch01FrameObserver = saved }()
	rec.chapterParityReplay(t)
	if t.Failed() {
		return
	}
	rec.finish()
}

// mark：同 r5c（rust_golden_r5c_layers_test.go 的 r5cLayerRecorder.mark）。第一章第一次走 ensureTown 的是
// town_after_battle 這個 mark；這裡先走逐字複本的 ensureTown（插了擷取點），原本的 r.mark 之後再呼叫
// r.ensureTown() 時節點已是目前城鎮，直接返回，檢查點順序與內容不變。
func (rec *r5sFrameRecorder) mark(r *parityReplay, action parityAction) {
	if action.Label == "town_after_battle" {
		rec.ensureTown(r)
	}
	r.mark(action)
}

// ---- 逐字複本（tools/rust_r5c_layers_compare.py --profile r5s --verbatim 核對；只換名稱並插入 // r5s-insert 行）----
// TestChapterParityReplay：chapter_parity_replay_test.go 的逐字複本。
func (rec *r5sFrameRecorder) chapterParityReplay(t *testing.T) {
	chapterText := os.Getenv("FD2_PARITY_CHAPTER")
	slot := os.Getenv("FD2_PARITY_SLOT")
	run := os.Getenv("FD2_PARITY_ORACLE_RUN")
	out := os.Getenv("FD2_PARITY_OUT")
	if chapterText == "" || slot == "" || run == "" || out == "" {
		t.Skip("需要 FD2_PARITY_CHAPTER／FD2_PARITY_SLOT／FD2_PARITY_ORACLE_RUN／FD2_PARITY_OUT")
	}
	chapter, err := strconv.Atoi(chapterText)
	if err != nil || chapter < 1 || chapter > 30 {
		t.Fatalf("FD2_PARITY_CHAPTER=%q", chapterText)
	}
	entryMode := os.Getenv("FD2_PARITY_ENTRY")
	if entryMode == "" {
		entryMode = "load"
	}
	if entryMode != "load" && (entryMode != "start" || chapter != 1) {
		t.Fatalf("FD2_PARITY_ENTRY=%q；START只適用首章", entryMode)
	}
	actions := readParityActions(t, filepath.Join(run, "actions.jsonl"))
	if len(actions) == 0 {
		t.Fatal("actions.jsonl 是空的")
	}
	legacyENDSources, err := recoverParityLegacyENDSources(run, actions, readParityControlKeys(t, filepath.Join(run, "control-history.jsonl")))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	frameSetting := os.Getenv("FD2_PARITY_CAPTURE_FRAMES")
	if frameSetting != "" && frameSetting != "0" && frameSetting != "1" {
		t.Fatalf("FD2_PARITY_CAPTURE_FRAMES=%q；只允許0或1", frameSetting)
	}

	t.Setenv("FD2_TITLE", "1")
	t.Setenv("FD2_SEED", "4")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	t.Setenv("FD2_SHOT_AI", "1")
	battle.DebugAI = log.Printf
	defer func() { battle.DebugAI = nil }()
	// 建構槽是唯讀輸入；酒店存檔（0x30012）要寫回同一份 FD2.SAV，所以在輸出目錄
	// 放一份複本當可寫覆蓋層，與 oracle 的 -state 覆蓋層同一個角色。
	slotCopy := filepath.Join(out, "FD2.SAV")
	if raw, err := os.ReadFile(slot); err != nil {
		t.Fatal(err)
	} else if err := os.WriteFile(slotCopy, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_NATIVE_SAVE", slotCopy)
	t.Setenv("FD2_CUTSCENE_LOG", "1")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })

	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	hpPolicy := &parityHPLockPolicy{}
	for _, action := range actions {
		if action.Kind == "hp_lock" {
			parityHPPolicies[g] = hpPolicy
			break
		}
	}
	t.Cleanup(func() { delete(parityHPPolicies, g) })
	settings, err := json.MarshalIndent(map[string]interface{}{
		"go_seed":               4,
		"native_rng_initial":    g.nativeRNGState,
		"method":                "AI／攻擊確認／END／升級比較點承接原版受控 RNG word",
		"legacy_end_sources":    legacyENDSources,
		"capture_frames":        frameSetting != "0",
		"modified_path_hp_lock": parityHPPolicies[g] != nil,
		"entry_mode":            entryMode,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "replay-settings.json"), append(settings, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &parityReplay{t: t, g: g, out: out, run: run,
		battle:           fmt.Sprintf("battle_ch%02d", chapter),
		town:             fmt.Sprintf("town_ch%02d", chapter),
		townAfter:        fmt.Sprintf("town_ch%02d", chapter+1),
		preparationAfter: fmt.Sprintf("preparation_ch%02d", chapter+1),
		aiEntries:        readParityAIEntries(t, filepath.Join(run, "eip-trace.jsonl"), "0x13A9F"),
		growthEntries:    readParityAIEntries(t, filepath.Join(run, "eip-trace.jsonl"), "0x1E54A"),
		skipFrames:       frameSetting == "0",
		keys:             readParityControlKeys(t, filepath.Join(run, "control-history.jsonl"))}
	g.aiPlanObserver = r.observeAIPlan
	g.growthRollObserver = r.observeGrowthRoll
	if chapter == 29 {
		r.turnPairEntries = readParityAIEntries(t, filepath.Join(run, "eip-trace.jsonl"), "0x35EE6")
		g.nativeTurnPairRNGInput = r.observeTurnPairRNG
	}
	logFile, err := os.Create(filepath.Join(out, "checkpoints.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()
	r.log = logFile
	// 執行期失敗即關閉（loadErr）會讓 pump 直接 Fatalf；離開前把它記成最後一筆
	// checkpoint，verifier 才知道這一章是在哪個動作後被重製端的錯誤擋住。
	defer func() {
		if g.loadErr != "" {
			r.write(parityCheckpoint{Index: r.index, Kind: "runtime_error", Node: g.camp.NodeID(), UI: r.ui(),
				Gold: g.gold, RNGWord: int(g.nativeRNGState), Units: r.units(), Note: "loadErr: " + g.loadErr})
		}
	}()

	// 標題 LOAD：與原版側計畫一樣走 LOAD → 四槽 → 槽 0。開場動畫直接跳到選單，
	// 那一段不在本章的比較範圍。
	if entryMode == "start" {
		r.startFromNormalTitle()
	} else {
		g.titlePhase, g.titleSel = "menu", 0
		if !g.applyTitleMenuEvent(TitleMenuDown) || !g.applyTitleMenuEvent(TitleMenuConfirm) {
			t.Fatal("標題 LOAD 未由正式 menu owner 消費")
		}
		for i := 0; i < 24; i++ {
			g.applyTitleMenuEvent(TitleMenuTick)
		}
		if !g.applyTitleSlotEvent(TitleSlotConfirm) {
			t.Fatalf("LOAD 槽 0 失敗：%s", g.msg)
		}
		loadNode := g.camp.Node()
		standalone := loadNode != nil && loadNode.Type == "preparation" && loadNode.Cancel == "" &&
			g.camp.NodeID() == fmt.Sprintf("preparation_ch%02d", chapter)
		if (g.camp.NodeID() != r.town && !standalone) || g.loadErr != "" {
			t.Fatalf("LOAD 後 node=%q err=%q", g.camp.NodeID(), g.loadErr)
		}
	}
	r.settleTown()

	// 依原版側的語意動作重播。戰場上遇到分岔（原版選得到的格重製端沒單位、原版
	// 接受的移動重製端拒絕、攻擊目標格沒人）不中止：記一筆 divergence checkpoint，
	// 讓這個單位待機，跳過它後面的動作，繼續到 END／清場／城鎮，四個 gate 才都有資料。
	var actor, target *battle.Unit
	var moveTo [2]int
	skipUnit := false
	traceSeq := 0
	if path := os.Getenv("FD2_FOCUS_TRACE_OUT"); path != "" {
		trace, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		defer trace.Close()
		battle.NativeFocusTraceHook = func(x, y int, view battle.NativeMapViewState) {
			round := 0
			if g.st != nil {
				round = g.st.Turn
			}
			fmt.Fprintf(trace, "%d r%d (%d,%d) from cur(%d,%d) cam(%d,%d)\n", traceSeq, round, x, y,
				view.CursorX, view.CursorY, view.CameraX, view.CameraY)
		}
		defer func() { battle.NativeFocusTraceHook = nil }()
	}
	for i, action := range actions {
		traceSeq = action.Seq
		if skipUnit {
			switch action.Kind {
			case "move", "stay", "attack", "attack_result", "wait", "cancel":
				r.note(action, "skipped after divergence")
				continue
			}
			skipUnit = false
		}
		switch action.Kind {
		case "hp_lock":
			r.updateHPLocks(action)
		case "mark":
			rec.mark(r, action)
		case "select":
			actor = r.selectUnit(action)
			if actor == nil {
				skipUnit = true
			}
		case "move":
			moveTo = [2]int{action.To[0], action.To[1]}
			if !r.moveUnit(action, actor) {
				skipUnit = true
			}
		case "stay":
			moveTo = [2]int{action.At[0], action.At[1]}
			r.stayUnit(action, actor)
		case "attack":
			target = r.attack(action, actor)
			if target == nil {
				skipUnit = true
			}
		case "attack_result":
			r.checkpointAttackResult(action, actor, target, moveTo)
		case "wait":
			r.waitUnit(action, actor)
		case "item_use":
			r.useSelfRestoreItem(action, actor)
		case "cancel":
			r.cancelUnit(action, actor)
		case "end_turn":
			stopBeforeAI := false
			if i+1 < len(actions) && actions[i+1].Kind == "force_enemy_clear" {
				var err error
				stopBeforeAI, err = parityClearBeforeAI(action, actions[i+1])
				if err != nil {
					t.Fatal(err)
				}
			}
			r.endTurn(action, stopBeforeAI)
		case "force_enemy_clear":
			var next *parityAction
			if i+1 < len(actions) {
				next = &actions[i+1]
			}
			r.forceEnemyClear(action, next)
		case "town_enter":
			rec.townProbe(r, action)
		case "town_save":
			rec.townSave(r, action)
		case "preparation_save":
			r.preparationSave(action)
		case "ending_terminal":
			r.finishEnding(action)
		case "shop_sell":
			r.shopSell(action)
		case "shop_buy":
			r.shopBuy(action)
		case "secret_shop":
			r.secretShop(action)
		default:
			r.note(action, "unhandled action kind")
		}
		if g.loadErr != "" {
			t.Fatalf("動作 %s(seq %d) 之後執行期錯誤：%s\n阻塞：%s", action.Kind, action.Seq, g.loadErr, ch01Blockers(g))
		}
		r.prevSeq = action.Seq
		if requested := os.Getenv("FD2_PARITY_RING_GPU_SEQ"); requested != "" && requested == strconv.Itoa(action.Seq) {
			if skipUnit || (action.Kind != "move" && action.Kind != "stay") || !g.ring {
				t.Fatal("GPU prefix lacks the requested normal ring owner")
			}
			r.verifyRingGPU(action)
			return // Explicit prefix receipt; never publish a complete chapter receipt.
		}
	}
	t.Logf("AI 入口比較：原版 %d 筆、已消費 %d 筆、順序分岔 %d 筆", len(r.aiEntries), r.aiCursor, r.aiOrderDivergences)
	if r.turnPairCursor != len(r.turnPairEntries) {
		t.Fatalf("event79原版入口%d筆，重製只消費%d筆", len(r.turnPairEntries), r.turnPairCursor)
	}
	if chapter == 29 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(settings, &metadata); err != nil {
			t.Fatal(err)
		}
		metadata["event79_rng_inputs"] = r.turnPairInputs
		metadata["event79_rng_method"] = "正式consumer入口承接dosgolem35EE6原始rng_word；以入口前checkpoint核對原始回合，非整段骰序一致。"
		raw, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(out, "replay-settings.json"), append(raw, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if len(r.aiEntries) > 0 && r.aiCursor != len(r.aiEntries) {
		r.checkpoint("ai_order", 0, r.ui(), false,
			fmt.Sprintf("divergence: 原版 AI 入口 %d 筆，重製只消費 %d 筆", len(r.aiEntries), r.aiCursor))
	}
	if os.Getenv("FD2_PARITY_DEFEAT_TAIL") == "1" {
		r.verifyObservedDefeatTail()
	}
	r.checkpoint("end", 0, r.ui(), g.titlePhase == "")
}

// ensureTown：chapter_parity_replay_test.go 的逐字複本。
func (rec *r5sFrameRecorder) ensureTown(r *parityReplay) {
	g := r.g
	if g.camp.NodeID() == r.currentTown() {
		return
	}
	if g.camp.NodeID() == r.townAfter {
		r.battleDone = true
		return
	}
	rec.ensurePostbattleNode(r, r.townAfter)
}

// ensurePostbattleNode：chapter_parity_replay_test.go 的逐字複本。
func (rec *r5sFrameRecorder) ensurePostbattleNode(r *parityReplay, target string) {
	t, g := r.t, r.g
	// 戰鬥結束：等演出收掉，在正式 Enter 邊界確認，走戰後過場到城鎮。
	for frame := 0; frame < 3000 && g.camp.NodeID() == r.battle; frame++ {
		ackPresents(g)
		if resultInputReachable(g) && g.confirmBattleResult() {
			break
		}
		if (g.battleEvent != nil || g.nativeTurnStaging != nil) && len(g.dialog) > 0 {
			g.handleBattleEventDialogueInput(true)
		}
		if err := parityUpdateGame(g); err != nil {
			t.Fatalf("Update：%v", err)
		}
		rec.observeGap(g, "postbattle") // r5s-insert
	}
	if g.camp.NodeID() == r.battle {
		t.Fatalf("戰鬥沒有結束：result=%q\n阻塞：%s", g.result, ch01Blockers(g))
	}
	r.checkpoint("battle_result", 0, r.ui(), true)
	j := newJourneyTrace()
	defer func() {
		if !strings.Contains(g.loadErr, "invalid radial LUT remap geometry") {
			return
		}
		t.Logf("戰後徑向幾何：story view=%+v", g.storyNativeMapView)
		if hud, ok := g.nativeMapHUDInput(); ok {
			if in, err := buildNativeMapFrameInput(g.nativeMapAssets, g.m, g.st, nativeMapFrameRuntime{HUD: hud}); err == nil {
				t.Logf("戰後徑向幾何：native frame camera=(%d,%d)", in.Frame.CameraX, in.Frame.CameraY)
			}
		}
		for _, slot := range []int{10, 16} {
			if unit := g.handlerUnitAt(slot); unit != nil {
				t.Logf("戰後徑向幾何：slot%d XY=(%d,%d) presentation=%+v", slot, unit.X, unit.Y, unit.NativeMapPresentation)
			}
		}
	}()
	rec.driveStory(t, g, j, func() bool { return g.camp.NodeID() == target })
	r.battleDone = true
	rec.settleTown(r)
}

// driveStory：native_title_to_town_test.go 的逐字複本。
func (rec *r5sFrameRecorder) driveStory(t *testing.T, g *Game, j *journeyTrace, stop func() bool) {
	t.Helper()
	lastSig, still := "", 0
	for frame := 0; frame < journeyStoryFrames; frame++ {
		j.observe(g)
		if stop() {
			return
		}
		if n := g.camp.Node(); n != nil && (n.Type == "story" || n.Type == "cutscene") && storyEnterReady(g) {
			before := storySignature(g)
			g.handleNativeStoryInput(n, nativeStoryInput{enter: true})
			if storySignature(g) != before {
				j.press(journeyPhase(g))
			}
		}
		ackPresents(g)
		if err := g.Update(); err != nil {
			t.Fatalf("Update：%v", err)
		}
		if g.loadErr != "" {
			t.Fatalf("故事節點 %s 執行期錯誤：%s\n阻塞：%s", g.camp.NodeID(), g.loadErr, ch01Blockers(g))
		}
		rec.observeGap(g, "story") // r5s-insert
		if sig := progressSignature(g); sig == lastSig {
			if still++; still > journeyStallFrames {
				t.Fatalf("故事節點 %s 卡住 %d 幀（beat=%d dialog=%d page=%d progress=%d）\n阻塞：%s",
					g.camp.NodeID(), still, g.beatIdx, len(g.dialog), g.dlgPage,
					g.nativeDialogueProgress, ch01Blockers(g))
			}
		} else {
			lastSig, still = sig, 0
		}
	}
	t.Fatalf("故事節點 %s 在 %d 幀內沒有結束", g.camp.NodeID(), journeyStoryFrames)
}

// settleTown：chapter_parity_replay_test.go 的逐字複本。
func (rec *r5sFrameRecorder) settleTown(r *parityReplay) {
	g := r.g
	for frame := 0; frame < journeyTownSettleMax; frame++ {
		if g.nativeClassUIJob == nil && g.fade == nil && g.transitionReveal == nil &&
			g.indexedTransition == nil && frame > 10 {
			return
		}
		ackPresents(g)
		if err := parityUpdateGame(g); err != nil {
			r.t.Fatalf("Update：%v", err)
		}
		rec.observeGap(g, "settle") // r5s-insert
	}
}

// townProbe：chapter_parity_replay_test.go 的逐字複本。
func (rec *r5sFrameRecorder) townProbe(r *parityReplay, action parityAction) {
	t, g := r.t, r.g
	rec.ensureTown(r)
	delta := 1
	if action.Move == "right" {
		delta = -1
	}
	if !g.moveNativeTownSelection(delta) {
		t.Fatalf("town_enter(seq %d)：城鎮選擇無法移動", action.Seq)
	}
	rec.observeGap(g, "town_move") // r5s-insert
	selection := g.campSel
	r.enterTownOption(selection)
	pump(t, g, 120, func() bool { return g.nativeClassUIJob == nil && g.fade == nil })
	r.checkpoint("town_enter", action.Seq, r.ui(), true)
	// 退回城鎮，與原版側的 esc 相同。
	n := g.camp.Node()
	switch {
	case n == nil:
	case n.Type == "shop":
		g.leaveShop()
	case n.Type == "church":
		g.leaveChurch()
	case n.Type == "hotel":
		g.leaveHotel()
	case n.Type == "preparation":
		// 出口提示答 NO：走正式輸入 owner（esc），與原版側送的鍵相同。
		g.handleNativePreparationInput(nativePreparationInput{escape: true})
		pump(t, g, 240, func() bool { return g.camp.NodeID() == r.currentTown() })
	case n.Type == "town":
	default:
		g.camp.Advance("cancel")
		g.nativeTownHubReturn = true
		g.enterNode()
	}
	pump(t, g, 120, func() bool { return g.camp.NodeID() == r.currentTown() })
	if g.camp.NodeID() != r.currentTown() {
		t.Fatalf("town_enter(seq %d)：離開選項 %d 之後沒有回到城鎮：%q", action.Seq, selection, g.camp.NodeID())
	}
}

// townSave：chapter_parity_replay_test.go 的逐字複本。
func (rec *r5sFrameRecorder) townSave(r *parityReplay, action parityAction) {
	t, g := r.t, r.g
	rec.ensureTown(r)
	slot := 0
	if action.Slot != nil {
		slot = *action.Slot
	}
	// 原版側：城鎮 hub 停在酒店（0 號）按 enter → 圖示選單 right 一格到存檔 → enter
	// 開四槽列表 → enter 確認第一槽 → 「記錄儲存完畢！」；重製端走同一套原生酒店 UI。
	for guard := 0; g.campSel != 0 && guard < 8; guard++ {
		if !g.moveNativeTownSelection(1) {
			t.Fatalf("town_save(seq %d)：城鎮選擇移不到酒店", action.Seq)
		}
		rec.observeGap(g, "town_move") // r5s-insert
	}
	r.enterTownOption(0)
	pump(t, g, 120, func() bool { return g.nativeClassUIJob == nil && g.fade == nil })
	if n := g.camp.Node(); n == nil || n.Type != "hotel" || g.nativeHotelMode != "menu" {
		t.Fatalf("town_save(seq %d)：沒有進到原生酒店選單（node=%q mode=%q）", action.Seq, g.camp.NodeID(), g.nativeHotelMode)
	}
	for step := 0; step < nativeHotelServiceSave; step++ {
		g.handleNativeHotelInput(nativeHotelInput{delta: 1})
		rec.observeGap(g, "hotel_input") // r5s-insert
	}
	g.handleNativeHotelInput(nativeHotelInput{enter: true})
	rec.observeGap(g, "hotel_input") // r5s-insert
	if g.nativeHotelMode != "slots" {
		t.Fatalf("town_save(seq %d)：存檔列表沒有開（mode=%q）", action.Seq, g.nativeHotelMode)
	}
	for g.nativeHotelSlotSel < slot {
		g.handleNativeHotelInput(nativeHotelInput{down: true})
		rec.observeGap(g, "hotel_input") // r5s-insert
	}
	g.handleNativeHotelInput(nativeHotelInput{enter: true})
	rec.observeGap(g, "hotel_input") // r5s-insert
	if g.nativeHotelMode != "saved" {
		t.Fatalf("town_save(seq %d)：存檔後沒有「記錄儲存完畢」（mode=%q）", action.Seq, g.nativeHotelMode)
	}
	// 收據記的是寫回後整份 FD2.SAV 的 sha256（與 oracle 的 save_sha256 同一個算法）；
	// 寫不出原版槽就把錯誤寫進 note，verifier 會判 save 項失敗。
	note := ""
	if g.nativeChapterSlotSaveErr != nil {
		note = "原版槽寫回失敗：" + g.nativeChapterSlotSaveErr.Error()
	} else if raw, err := os.ReadFile(nativeCurrentSavePath()); err != nil {
		note = "原版槽寫回後讀不到 FD2.SAV：" + err.Error()
	} else {
		note = fmt.Sprintf("save_sha256=%x", sha256.Sum256(raw))
	}
	if _, err := os.Stat(saveSlotPath(slot)); err != nil {
		note += "；重製端自有存檔未寫出：" + err.Error()
	}
	r.verifyChapterSlotSave(slot)
	r.checkpoint("town_save", action.Seq, "hotel", true, note)
	// 原版側接著 esc 三次回到城鎮：訊息 → 選單 → 離開。
	g.handleNativeHotelInput(nativeHotelInput{esc: true})
	rec.observeGap(g, "hotel_input") // r5s-insert
	g.handleNativeHotelInput(nativeHotelInput{esc: true})
	rec.observeGap(g, "hotel_input") // r5s-insert
	pump(t, g, 120, func() bool { return g.camp.NodeID() == r.currentTown() })
	if g.camp.NodeID() != r.currentTown() {
		t.Fatalf("town_save(seq %d)：離開酒店後沒有回到城鎮：%q", action.Seq, g.camp.NodeID())
	}
}
