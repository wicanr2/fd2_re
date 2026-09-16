package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// 章重播（111 章工作單元的重製側）：從同一份建構槽由標題 LOAD 進城，照原版側
// 驅動器留下的語意動作（actions.jsonl）走同一組決定——選誰、走到哪、打誰、待機、
// END、清場、進哪棟建築——每一步寫出與 oracle 同形的 checkpoint（單位、回合、金幣、
// 亂數字組、節點）與 320×200 索引畫面。四個 gate 由 tools/verify_chapter_parity.py 判。
//
// 亂數：0x627B8 是行程級狀態，從標題畫面起就被各種演出消耗，兩側不可能逐 tick
// 對齊。所以在每個會擲骰的決定點（攻擊確認、END 換手）把原版當下的字組同步進
// 重製端，比的是「同一個字組下的結果」；同步之間的漂移由 verifier 另外列出。
//
// 環境變數：
//   FD2_PARITY_CHAPTER      章號（例：4）
//   FD2_PARITY_SLOT         建構槽 FD2.SAV（同一份餵給 dosgolem）
//   FD2_PARITY_ORACLE_RUN   dosgolem 輸出目錄（actions.jsonl、checkpoint-*.json）
//   FD2_PARITY_OUT          重製側輸出目錄

type parityAction struct {
	Kind    string   `json:"kind"`
	Seq     int      `json:"seq"`
	Round   int      `json:"round"`
	RNGWord *int     `json:"rng_word"`
	Gold    *int     `json:"gold"`
	At      []int    `json:"at"`
	From    []int    `json:"frm"`
	To      []int    `json:"to"`
	Target  []int    `json:"target"`
	Index   *int     `json:"index"`
	UI      string   `json:"ui"`
	Move    string   `json:"move"`
	Probe   *int     `json:"probe_index"`
	Cleared *int     `json:"cleared"`
	Slot    *int     `json:"slot"`
	Moves   []string `json:"moves"`
	Key     string   `json:"key"`
	Label   string   `json:"label"`
}

type parityUnit struct {
	Camp     int `json:"camp"`
	X        int `json:"x"`
	Y        int `json:"y"`
	HP       int `json:"hp"`
	Identity int `json:"identity"`
	Acted    int `json:"acted"`
	// PX／PY 是原版 record +0／+1（NativeMapPresentation），AI 用的是這一組座標。
	PX int `json:"px"`
	PY int `json:"py"`
}

type parityCheckpoint struct {
	Index     int          `json:"index"`
	Kind      string       `json:"kind"`
	OracleSeq int          `json:"oracle_seq"`
	Node      string       `json:"node"`
	UI        string       `json:"ui"`
	Round     int          `json:"round"`
	Gold      int          `json:"gold"`
	RNGWord   int          `json:"rng_word"`
	RNGSynced bool         `json:"rng_synced"`
	Cursor    []int        `json:"cursor"`
	Camera    []int        `json:"camera,omitempty"` // 原生地圖視圖的 camera（與 oracle view.camera_x/y 同義）
	Units     []parityUnit `json:"units"`
	Frame     string       `json:"frame,omitempty"` // 相位 0；同名 -pK 為其他相位
	FrameHash string       `json:"indexed_sha256,omitempty"`
	Note      string       `json:"note,omitempty"`
}

type parityReplay struct {
	t          *testing.T
	g          *Game
	out        string
	log        *os.File
	index      int
	battle     string
	town       string // LOAD 進的城鎮（本章戰前）
	townAfter  string // 戰後城鎮（原版 chapter 已推進，城鎮記錄是下一章的）
	battleDone bool
	// aiEntries 是原版側 eip-trace 的 0x13A9F 入口（每個 AI 單位行動一筆，含 rng_word）；
	// 重播在每個 AI 計畫產生後、執行前依序對齊 RNG。等待迴圈（死亡訊息、對白）會依
	// 虛擬時間消耗 0x4E893，回合內的 RNG 沒有辦法逐次對上，所以以「決策點」對齊。
	aiEntries          []parityAIEntry
	aiCursor           int
	aiOrderDivergences int
	// growthEntries 是 eip-trace 的 0x1E54A 入口（升級成長每一次會擲骰的欄位一筆）。
	// 升級訊息的等待迴圈同樣依時序吃亂數，所以每一次成長擲骰前都對齊一次。
	growthEntries []parityAIEntry
	growthCursor  int
}

// parityAIEntry 是 eip-trace.jsonl 裡一筆 0x13A9F 入口：unit 是 record 索引（堆疊第一個引數）。
type parityAIEntry struct {
	ControlSeq int
	Unit       int
	RNGWord    int
}

// readParityAIEntries 讀 oracle 輸出目錄的 eip-trace.jsonl（沒有就回空）。
func readParityAIEntries(t *testing.T, path, eip string) []parityAIEntry {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []parityAIEntry
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var row struct {
			ControlSeq int      `json:"control_seq"`
			EIP        string   `json:"eip"`
			RNGWord    *int     `json:"rng_word"`
			Stack      []string `json:"stack"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("eip-trace.jsonl：%v：%s", err, line)
		}
		if row.EIP != eip || row.RNGWord == nil || len(row.Stack) < 2 {
			continue
		}
		unit, err := strconv.ParseInt(strings.TrimPrefix(row.Stack[1], "0x"), 16, 32)
		if err != nil {
			t.Fatalf("eip-trace.jsonl 堆疊引數：%v：%s", err, line)
		}
		out = append(out, parityAIEntry{ControlSeq: row.ControlSeq, Unit: int(unit), RNGWord: *row.RNGWord})
	}
	return out
}

// observeGrowthRoll 在重製端每一次升級成長擲骰前把原版側下一筆 0x1E54A 的 rng_word
// 抄過來；原版側沒有更多收據就照重製端目前的狀態擲。
func (r *parityReplay) observeGrowthRoll(u *battle.Unit, state uint16) uint16 {
	if r.growthCursor >= len(r.growthEntries) {
		return state
	}
	entry := r.growthEntries[r.growthCursor]
	r.growthCursor++
	return uint16(entry.RNGWord)
}

// observeAIPlan 在重製端每個 AI 單位計畫產生後對齊 RNG：原版側同一順序的 0x13A9F
// 入口若是同一個 record 索引就把 rng_word 抄過來；不是就記一筆 ai_order 分岔並
// 往後找同一單位的入口（找到才對齊）。
func (r *parityReplay) observeAIPlan(plan *battle.AIPlan) {
	if plan == nil || plan.U == nil || r.aiCursor >= len(r.aiEntries) {
		return
	}
	index := -1
	for i, u := range r.g.st.Units {
		if u == plan.U {
			index = i
			break
		}
	}
	entry := r.aiEntries[r.aiCursor]
	if entry.Unit == index {
		r.aiCursor++
		r.g.nativeRNGState = uint16(entry.RNGWord)
		return
	}
	r.aiOrderDivergences++
	r.checkpoint("ai_order", entry.ControlSeq, r.ui(), false,
		fmt.Sprintf("divergence: 原版第 %d 個 AI 行動是 record %d，重製端是 record %d", r.aiCursor, entry.Unit, index))
	for j := r.aiCursor + 1; j < len(r.aiEntries) && j < r.aiCursor+8; j++ {
		if r.aiEntries[j].Unit == index {
			r.aiCursor = j + 1
			r.g.nativeRNGState = uint16(r.aiEntries[j].RNGWord)
			return
		}
	}
}

func TestChapterParityReplay(t *testing.T) {
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
	actions := readParityActions(t, filepath.Join(run, "actions.jsonl"))
	if len(actions) == 0 {
		t.Fatal("actions.jsonl 是空的")
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("FD2_TITLE", "1")
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
	r := &parityReplay{t: t, g: g, out: out,
		battle:        fmt.Sprintf("battle_ch%02d", chapter),
		town:          fmt.Sprintf("town_ch%02d", chapter),
		townAfter:     fmt.Sprintf("town_ch%02d", chapter+1),
		aiEntries:     readParityAIEntries(t, filepath.Join(run, "eip-trace.jsonl"), "0x13A9F"),
		growthEntries: readParityAIEntries(t, filepath.Join(run, "eip-trace.jsonl"), "0x1E54A")}
	g.aiPlanObserver = r.observeAIPlan
	g.growthRollObserver = r.observeGrowthRoll
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
	if g.camp.NodeID() != r.town || g.loadErr != "" {
		t.Fatalf("LOAD 後 node=%q err=%q", g.camp.NodeID(), g.loadErr)
	}
	r.settleTown()

	// 依原版側的語意動作重播。戰場上遇到分岔（原版選得到的格重製端沒單位、原版
	// 接受的移動重製端拒絕、攻擊目標格沒人）不中止：記一筆 divergence checkpoint，
	// 讓這個單位待機，跳過它後面的動作，繼續到 END／清場／城鎮，四個 gate 才都有資料。
	var actor, target *battle.Unit
	var moveTo [2]int
	skipUnit := false
	for i, action := range actions {
		if skipUnit {
			switch action.Kind {
			case "move", "stay", "attack", "attack_result", "wait", "cancel":
				r.note(action, "skipped after divergence")
				continue
			}
			skipUnit = false
		}
		switch action.Kind {
		case "mark":
			r.mark(action)
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
		case "cancel":
			r.cancelUnit(action, actor)
		case "end_turn":
			nextIsForceClear := i+1 < len(actions) && actions[i+1].Kind == "force_enemy_clear"
			r.endTurn(action, nextIsForceClear)
		case "force_enemy_clear":
			r.forceEnemyClear(action)
		case "town_enter":
			r.townProbe(action)
		case "town_save":
			r.townSave(action)
		case "shop_sell":
			r.shopSell(action)
		case "secret_shop":
			r.secretShop(action)
		default:
			r.note(action, "unhandled action kind")
		}
		if g.loadErr != "" {
			t.Fatalf("動作 %s(seq %d) 之後執行期錯誤：%s\n阻塞：%s", action.Kind, action.Seq, g.loadErr, ch01Blockers(g))
		}
	}
	r.checkpoint("end", 0, r.ui(), true)
}

// mark 對應驅動端的 mark：同一個節點兩側各取一張畫面。標記本身不送鍵，但重製端
// 要在這裡把節點推到同一處（出口提示、戰場交出操作權、戰後城鎮）。
func (r *parityReplay) mark(action parityAction) {
	t, g := r.t, r.g
	switch action.Label {
	case "town_loaded":
		r.settleTown()
		r.checkpoint("town_loaded", action.Seq, "town", true)
	case "departure_prompt":
		for g.campSel != 2 {
			if !g.moveNativeTownSelection(1) {
				t.Fatal("無法走到出口")
			}
		}
		r.enterTownOption(2)
		if g.camp.Node() == nil || g.camp.Node().Type != "preparation" {
			t.Fatalf("出口之後不是整備節點：%q", g.camp.NodeID())
		}
		// 提示的開啟動畫跑完、owner 開始收鍵之後才算同一個節點（原版側等 14 格）。
		if !pump(t, g, 600, func() bool { return !g.nativeClassUIBlocksInput() }) {
			t.Fatal("整備提示的開啟動畫沒有結束")
		}
		r.checkpoint("departure_prompt", action.Seq, "preparation", true)
	case "battle_start":
		if g.camp.Node() != nil && g.camp.Node().Type == "preparation" {
			if !g.handleNativePreparationInput(nativePreparationInput{enter: true}) {
				t.Fatal("整備 YES 未被正式 owner 消費")
			}
			if !pump(t, g, 600, func() bool { return g.camp.Node() != nil && g.camp.Node().Type != "preparation" }) {
				t.Fatalf("YES 之後沒有離開整備節點：%q（prepSelecting=%v prepConfirm=%v prepIDs=%d prepLimit=%d job=%v err=%q）",
					g.camp.NodeID(), g.prepSelecting, g.prepConfirm, len(g.prepIDs), g.prepLimit, g.nativeClassUIJob != nil, g.loadErr)
			}
		}
		j := newJourneyTrace()
		driveStory(t, g, j, func() bool { return g.camp.NodeID() == r.battle })
		if !pump(t, g, journeyStoryFrames, func() bool {
			if r.playerHasControl() {
				return true
			}
			if (g.battleEvent != nil || g.nativeTurnStaging != nil) && len(g.dialog) > 0 && storyEnterReady(g) {
				g.handleBattleEventDialogueInput(true)
			}
			return false
		}) {
			t.Fatalf("戰場沒有交出操作權\n阻塞：%s", ch01Blockers(g))
		}
		r.checkpoint("battle_start", action.Seq, "cursor", true)
	case "town_after_battle":
		r.ensureTown()
		r.checkpoint("town_after_battle", action.Seq, "town", true)
	default:
		r.checkpoint("mark:"+action.Label, action.Seq, r.ui(), true)
	}
}

func readParityActions(t *testing.T, path string) []parityAction {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []parityAction
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var a parityAction
		if err := json.Unmarshal([]byte(line), &a); err != nil {
			t.Fatalf("actions.jsonl：%v：%s", err, line)
		}
		out = append(out, a)
	}
	return out
}

func (r *parityReplay) playerHasControl() bool {
	g := r.g
	return g.camp != nil && g.camp.NodeID() == r.battle && g.st != nil &&
		g.result == "" && !g.aiBusy && g.battleEvent == nil && g.nativeTurnStaging == nil &&
		len(g.dialog) == 0 && g.walk == nil && g.atk == nil && !g.ring &&
		g.nativeClassUIJob == nil && g.spawnIntroTransition == nil && g.indexedTransition == nil &&
		g.nativeUnitPresent == nil && g.actJob == nil && g.camPan == nil && g.focusJob == nil
}

func (r *parityReplay) settleTown() {
	g := r.g
	for frame := 0; frame < journeyTownSettleMax; frame++ {
		if g.nativeClassUIJob == nil && g.fade == nil && g.transitionReveal == nil &&
			g.indexedTransition == nil && frame > 10 {
			return
		}
		ackPresents(g)
		if err := g.Update(); err != nil {
			r.t.Fatalf("Update：%v", err)
		}
	}
}

func (r *parityReplay) enterTownOption(selection int) {
	g := r.g
	if source, ok := g.composeNativeTownFrame(); ok {
		g.prepPromptSource = append([]byte(nil), source...)
	}
	g.camp.Advance(fmt.Sprintf("opt%d", selection))
	g.enterNode()
}

func (r *parityReplay) ui() string {
	g := r.g
	n := g.camp.Node()
	if n == nil {
		return "unknown"
	}
	switch n.Type {
	case "town":
		return "town"
	case "shop":
		return "shop"
	case "church":
		return "church"
	case "preparation":
		return "preparation"
	case "battle":
		switch {
		case len(g.dialog) > 0:
			return "dialogue"
		case g.ring:
			return "ring"
		case g.sel != nil:
			return "target"
		case g.aiBusy:
			return "enemy"
		default:
			return "cursor"
		}
	case "story", "cutscene":
		return "dialogue"
	}
	return n.Type
}

func (r *parityReplay) units() []parityUnit {
	g := r.g
	if g.st == nil {
		return nil
	}
	var out []parityUnit
	for _, u := range g.st.Units {
		if u == nil || !u.OnField || u.HP <= 0 {
			continue
		}
		acted := 0
		if u.Acted {
			acted = 1
		}
		identity := -1
		if u.HasNativeIdentity {
			identity = u.NativeIdentity
		}
		out = append(out, parityUnit{Camp: nativeCampCode(u.Camp), X: u.X, Y: u.Y, HP: u.HP, Identity: identity, Acted: acted,
			PX: int(u.NativeMapPresentation.X), PY: int(u.NativeMapPresentation.Y)})
	}
	return out
}

// nativeCampCode 對應 raw `+6`：0 敵方、1 友軍、2 我方（與 dosgolem 收據一致）。
func nativeCampCode(camp battle.Camp) int {
	switch camp {
	case battle.Own:
		return 2
	case battle.Ally:
		return 1
	default:
		return 0
	}
}

// frame 寫出這一點的 320×200 索引畫面。精靈的待機循環（sub_1297d 的 Idle 0..3）與
// 城鎮游標脈衝相位是 BIOS tick 決定的，兩側不可能逐 tick 對齊；所以每一點寫出
// 全部相位的變體（remake-NNNN-pK.png），verifier 取差異最小的那一張。回傳主檔名
// 與主檔雜湊（相位 0）。
func (r *parityReplay) frame(kind string) (string, string) {
	g := r.g
	variants := [][]byte{}
	var palette color.Palette
	switch {
	case g.camp != nil && g.camp.Node() != nil && g.camp.Node().Type == "battle" && g.st != nil:
		saved := g.st.NativeMapCycleState
		for idle := 0; idle < 4; idle++ {
			g.st.NativeMapCycleState.Idle = idle
			if err := g.composeNativeMapFrame(); err != nil {
				g.st.NativeMapCycleState = saved
				var prov []string
				for i, u := range g.st.Units {
					if u == nil {
						continue
					}
					prov = append(prov, fmt.Sprintf("%d:camp=%d (%d,%d) fig=%d on=%v hp=%d pres=%v slot=%v key=%v b5=%v id=%v/%d",
						i, u.Camp, u.X, u.Y, u.BattleFig, u.OnField, u.HP, u.HasNativeMapPresentation, u.HasMapSelectorSlot, u.HasMapSelectorKey,
						u.HasNativeRecordByte5, u.HasNativeIdentity, u.NativeIdentity))
				}
				r.t.Fatalf("%s 組不出戰場整幀：%v\n%s", kind, err, strings.Join(prov, "\n"))
			}
			variants = append(variants, append([]byte(nil), g.nativeMapVGA...))
		}
		g.st.NativeMapCycleState = saved
		palette = g.nativeMapAssets.Palette
		if len(g.nativeMapDAC) == 256*3 {
			if p, e := fdother.VGAPaletteFromDAC(g.nativeMapDAC); e == nil {
				palette = p
			}
		}
	case g.camp != nil && g.camp.Node() != nil && g.camp.Node().Type == "town":
		saved := g.nativeTownUIPulse
		for pulse := 0; pulse < 4; pulse++ {
			g.nativeTownUIPulse = pulse
			source, ok := g.composeNativeTownFrame()
			if !ok {
				g.nativeTownUIPulse = saved
				return "", ""
			}
			variants = append(variants, append([]byte(nil), source...))
		}
		g.nativeTownUIPulse = saved
		palette = g.nativeClassUI.palette
	case g.camp != nil && g.camp.Node() != nil && g.camp.Node().Type == "preparation":
		source, ok := g.composeNativePreparationPromptFrame()
		if !ok {
			return "", ""
		}
		variants, palette = append(variants, source), g.nativeClassUI.palette
	case g.camp != nil && g.camp.Node() != nil && g.camp.Node().Type == "shop":
		saved := g.nativeShopUIPulse
		for pulse := 0; pulse < 4; pulse++ {
			g.nativeShopUIPulse = pulse
			source, ok := g.composeNativeShopServiceMenu()
			if !ok {
				break
			}
			variants = append(variants, append([]byte(nil), source...))
		}
		g.nativeShopUIPulse = saved
		palette = g.nativeClassUI.palette
	default:
		return "", ""
	}
	if len(variants) == 0 || len(palette) == 0 {
		return "", ""
	}
	var mainName, mainHash string
	for k, pix := range variants {
		if len(pix) != 320*200 {
			r.t.Fatalf("%s 相位 %d 的畫面不是 320×200：%d", kind, k, len(pix))
		}
		pic := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
		copy(pic.Pix, pix)
		name := fmt.Sprintf("remake-%04d-p%d.png", r.index, k)
		f, err := os.Create(filepath.Join(r.out, name))
		if err != nil {
			r.t.Fatal(err)
		}
		if err := png.Encode(f, pic); err != nil {
			f.Close()
			r.t.Fatal(err)
		}
		f.Close()
		if k == 0 {
			mainName, mainHash = name, fmt.Sprintf("%x", sha256.Sum256(pix))
		}
	}
	return mainName, mainHash
}

func (r *parityReplay) checkpoint(kind string, seq int, ui string, withFrame bool, extra ...string) parityCheckpoint {
	g := r.g
	cp := parityCheckpoint{Index: r.index, Kind: kind, OracleSeq: seq, Node: g.camp.NodeID(), UI: ui,
		Gold: g.gold, RNGWord: int(g.nativeRNGState), Cursor: []int{g.curX, g.curY}, Units: r.units()}
	if g.st != nil && g.st.HasNativeMapViewState {
		cp.Camera = []int{g.st.NativeMapViewState.CameraX, g.st.NativeMapViewState.CameraY}
	}
	for _, e := range extra {
		if e == "rng_synced" {
			cp.RNGSynced = true
		} else {
			cp.Note = e
		}
	}
	if g.st != nil {
		cp.Round = g.st.Turn
	}
	if withFrame {
		cp.Frame, cp.FrameHash = r.frame(kind)
	}
	r.write(cp)
	return cp
}

func (r *parityReplay) write(cp parityCheckpoint) {
	encoded, _ := json.Marshal(cp)
	fmt.Fprintf(r.log, "%s\n", encoded)
	r.index++
}

func (r *parityReplay) note(action parityAction, text string) {
	r.write(parityCheckpoint{Index: r.index, Kind: action.Kind, OracleSeq: action.Seq, Node: r.g.camp.NodeID(),
		UI: r.ui(), Gold: r.g.gold, RNGWord: int(r.g.nativeRNGState), Note: text})
}

func (r *parityReplay) syncRNG(action parityAction) bool {
	if action.RNGWord == nil {
		return false
	}
	r.g.nativeRNGState = uint16(*action.RNGWord)
	return true
}

func (r *parityReplay) selectUnit(action parityAction) *battle.Unit {
	t, g := r.t, r.g
	if !pump(t, g, journeyStoryFrames, r.playerHasControl) {
		t.Fatalf("select(seq %d)：玩家沒有操作權\n阻塞：%s", action.Seq, ch01Blockers(g))
	}
	if !g.positionScreenshotCursor(action.At[0], action.At[1]) {
		t.Fatalf("select(seq %d)：游標移不到 (%d,%d)", action.Seq, action.At[0], action.At[1])
	}
	g.confirm()
	if g.sel == nil || g.sel.X != action.At[0] || g.sel.Y != action.At[1] {
		r.checkpoint("select", action.Seq, r.ui(), true,
			fmt.Sprintf("divergence: 原版在 (%d,%d) 選到單位，重製端沒有（err=%q）", action.At[0], action.At[1], g.loadErr))
		if g.sel != nil {
			r.releaseSelection()
		}
		if g.nativeSystemCursorOverlay || g.ring {
			// 空格上的 enter 開的是系統選單（0x117E7→0x16F55）；等同原版側送 esc 退回地圖游標。
			pump(t, g, 240, func() bool { return !g.actionOverlayBlocksInput() })
			g.resetActionOverlayLifecycle()
		}
		return nil
	}
	r.checkpoint("select", action.Seq, "target", true)
	return g.sel
}

// releaseSelection 把誤選的單位放掉（等同原版在 target 模式送 esc）。
func (r *parityReplay) releaseSelection() {
	g := r.g
	g.clearNativePlayerMovement()
	g.sel, g.reach, g.moved = nil, nil, false
}

// standDown 讓目前選中的單位就地待機，分岔之後用它把回合推得動。
func (r *parityReplay) standDown(action parityAction, actor *battle.Unit) {
	t, g := r.t, r.g
	if actor == nil || g.sel != actor {
		return
	}
	if !g.ring {
		g.confirm()
		pump(t, g, 240, func() bool { return g.ring })
	}
	if !g.ring || !nativeActionSelectable(g.actionOverlayAvailability(), 3) {
		return
	}
	g.ringSel = 3
	closeRing(t, g, g.finishSelectedWait)
	pump(t, g, ch01FrameBudget, func() bool { return actor.Acted || actor.HP <= 0 })
	r.checkpoint("stand_down", action.Seq, "cursor", false, "divergence 之後就地待機")
}

func (r *parityReplay) moveUnit(action parityAction, actor *battle.Unit) bool {
	t, g := r.t, r.g
	if actor == nil {
		t.Fatalf("move(seq %d)：沒有先 select", action.Seq)
	}
	if !g.positionScreenshotCursor(action.To[0], action.To[1]) {
		t.Fatalf("move(seq %d)：游標移不到 (%d,%d)", action.Seq, action.To[0], action.To[1])
	}
	g.confirm()
	if g.walk == nil {
		r.checkpoint("move", action.Seq, r.ui(), true,
			fmt.Sprintf("divergence: (%d,%d)→(%d,%d) 原版接受、重製端拒絕（err=%q）",
				action.From[0], action.From[1], action.To[0], action.To[1], g.loadErr))
		if !g.positionScreenshotCursor(actor.X, actor.Y) {
			t.Fatalf("move(seq %d)：分岔後游標移不回 (%d,%d)", action.Seq, actor.X, actor.Y)
		}
		r.standDown(action, actor)
		return false
	}
	if !pump(t, g, ch01FrameBudget, func() bool { return g.walk == nil }) {
		t.Fatalf("move(seq %d)：移動動畫沒有結束", action.Seq)
	}
	if !pump(t, g, 240, func() bool { return g.ring }) {
		t.Fatalf("move(seq %d)：走到 (%d,%d) 之後沒有開指令環", action.Seq, action.To[0], action.To[1])
	}
	r.checkpoint("move", action.Seq, "ring", true)
	return true
}

func (r *parityReplay) stayUnit(action parityAction, actor *battle.Unit) {
	t, g := r.t, r.g
	if actor == nil {
		t.Fatalf("stay(seq %d)：沒有先 select", action.Seq)
	}
	g.confirm()
	if !pump(t, g, 240, func() bool { return g.ring }) {
		t.Fatalf("stay(seq %d)：原地確認之後沒有開指令環", action.Seq)
	}
	r.checkpoint("stay", action.Seq, "ring", true)
}

func (r *parityReplay) attack(action parityAction, actor *battle.Unit) *battle.Unit {
	t, g := r.t, r.g
	if actor == nil {
		t.Fatalf("attack(seq %d)：沒有先 select", action.Seq)
	}
	available := g.actionOverlayAvailability()
	if !nativeActionSelectable(available, 0) {
		// 位置早已分岔時，原版打得到的格在重製端可能沒有敵人；這是行為 gate
		// 的分岔點，不是重播的執行期錯誤。記下來、改待機，讓後面的節點繼續對。
		r.checkpoint("attack_armed", action.Seq, r.ui(), true,
			fmt.Sprintf("divergence: 原版可攻擊 (%d,%d)→(%d,%d)，重製端指令環攻擊不可選 availability=%v",
				actor.X, actor.Y, action.Target[0], action.Target[1], available))
		r.waitInsteadOfAttack(actor)
		return nil
	}
	g.ringSel = 0
	if !closeRing(t, g, func() {}) {
		t.Fatalf("attack(seq %d)：指令環沒有收合", action.Seq)
	}
	if !g.positionScreenshotCursor(action.Target[0], action.Target[1]) {
		t.Fatalf("attack(seq %d)：游標移不到目標 (%d,%d)", action.Seq, action.Target[0], action.Target[1])
	}
	var target *battle.Unit
	for _, u := range g.st.Units {
		if u != nil && u.OnField && u.HP > 0 && u.X == action.Target[0] && u.Y == action.Target[1] {
			target = u
		}
	}
	if target == nil {
		r.checkpoint("attack_armed", action.Seq, r.ui(), true,
			fmt.Sprintf("divergence: 原版攻擊目標格 (%d,%d) 重製端沒有單位", action.Target[0], action.Target[1]))
		// 收掉目標選擇回到指令環，改待機。
		if g.positionScreenshotCursor(actor.X, actor.Y) {
			g.confirm()
			pump(t, g, 240, func() bool { return g.ring })
		}
		r.waitInsteadOfAttack(actor)
		return nil
	}
	extra := []string{}
	if r.syncRNG(action) {
		extra = append(extra, "rng_synced")
	}
	r.checkpoint("attack_armed", action.Seq, "target", true, extra...)
	g.confirm()
	hp := target.HP
	pump(t, g, ch01FrameBudget, func() bool {
		return g.atk == nil && g.walk == nil && !g.ring && (target.HP != hp || target.HP <= 0 || actor.Acted)
	})
	if !pump(t, g, ch01FrameBudget, func() bool { return actor.Acted || actor.HP <= 0 }) {
		t.Fatalf("attack(seq %d)：攻擊之後沒有行動完畢\n阻塞：%s", action.Seq, ch01Blockers(g))
	}
	return target
}

// waitInsteadOfAttack 在攻擊已分岔時讓單位待機，維持「每個單位都行動完」的回合結構。
func (r *parityReplay) waitInsteadOfAttack(actor *battle.Unit) {
	t, g := r.t, r.g
	if g.ring && nativeActionSelectable(g.actionOverlayAvailability(), 3) {
		g.ringSel = 3
		closeRing(t, g, g.finishSelectedWait)
		pump(t, g, ch01FrameBudget, func() bool { return actor.Acted || actor.HP <= 0 })
	}
}

func (r *parityReplay) checkpointAttackResult(action parityAction, _ *battle.Unit, _ *battle.Unit, _ [2]int) {
	g := r.g
	// 攻擊收尾之後可能接死亡對白、升級訊息；推到玩家再度有操作權或戰鬥結束。
	pump(r.t, g, ch01FrameBudget, func() bool { return r.playerHasControl() || g.result != "" || g.camp.NodeID() != r.battle })
	r.checkpoint("attack_result", action.Seq, r.ui(), true)
}

func (r *parityReplay) waitUnit(action parityAction, actor *battle.Unit) {
	t, g := r.t, r.g
	if actor == nil {
		t.Fatalf("wait(seq %d)：沒有先 select", action.Seq)
	}
	available := g.actionOverlayAvailability()
	if !nativeActionSelectable(available, 3) {
		t.Fatalf("wait(seq %d)：待機不可選 availability=%v", action.Seq, available)
	}
	g.ringSel = 3
	if !closeRing(t, g, g.finishSelectedWait) {
		t.Fatalf("wait(seq %d)：指令環沒有收合", action.Seq)
	}
	if !pump(t, g, ch01FrameBudget, func() bool { return actor.Acted || actor.HP <= 0 }) {
		t.Fatalf("wait(seq %d)：待機之後沒有行動完畢\n阻塞：%s", action.Seq, ch01Blockers(g))
	}
	r.checkpoint("wait", action.Seq, "cursor", true)
}

func (r *parityReplay) cancelUnit(action parityAction, actor *battle.Unit) {
	g := r.g
	// 原版側在指令環上 esc：取消這次行動，單位退回移動前那一格（ringInput 的 esc 分支）。
	closeRing(r.t, g, func() {
		g.msg = ""
		if g.sel == nil {
			return
		}
		if g.sel.X == g.selOrigX && g.sel.Y == g.selOrigY {
			g.clearNativePlayerMovement()
			g.sel, g.reach, g.moved = nil, nil, false
			return
		}
		g.sel.SetMapPlacement(g.selOrigX, g.selOrigY, g.sel.Dir)
		g.moved = false
		g.reach = g.st.Reachable(g.sel)
		g.restorePlayerMovementCursor()
	})
	pump(r.t, g, ch01FrameBudget, func() bool { return !g.ring && g.walk == nil })
	r.checkpoint("cancel", action.Seq, r.ui(), true)
}

// endTurn 送 END。stopBeforeAI 為真時只推到敵方回合開始（0x1A30B 的回復與 selector 1
// 事件之後、第一個 AI 行動之前）就停：原版側的 force_enemy_clear 是在 END 之後 20M
// 指令注入的，那時敵方還沒有任何一個單位行動（r9 收據 seq 934→935 之間沒有 0x13A9F
// 入口），所以那一回合的敵方 AI 在原版根本沒跑。
func (r *parityReplay) endTurn(action parityAction, stopBeforeAI bool) {
	t, g := r.t, r.g
	if g.camp.NodeID() != r.battle || g.result != "" {
		r.note(action, "end_turn 時已不在戰場")
		return
	}
	// 全員行動完原版會自己換手（0x13565）：那時敵方回合可能還在跑，原版側的 END
	// 是在下一回合才按下去的。先等到玩家重新拿到操作權，再照原版送 END。
	pump(t, g, ch01FrameBudget*6, func() bool { return g.result != "" || r.playerHasControl() })
	if g.result != "" {
		r.note(action, "end_turn 前戰鬥已分出勝負")
		return
	}
	extra := []string{}
	if r.syncRNG(action) {
		extra = append(extra, "rng_synced")
	}
	before := g.st.Turn
	g.endTurn()
	if stopBeforeAI {
		if !pump(t, g, ch01FrameBudget*6, func() bool { return g.result != "" || g.aiBusy }) {
			t.Fatalf("end_turn(seq %d)：敵方回合沒有開始\n阻塞：%s", action.Seq, ch01Blockers(g))
		}
		r.checkpoint("enemy_phase_start", action.Seq, r.ui(), true, extra...)
		return
	}
	if !pump(t, g, ch01FrameBudget*6, func() bool {
		return g.result != "" || (!g.aiBusy && g.nativeTurnStaging == nil && g.st.Turn > before && r.playerHasControl())
	}) {
		t.Fatalf("end_turn(seq %d)：敵方回合沒有結束（turn %d→%d aiBusy=%v）\n阻塞：%s",
			action.Seq, before, g.st.Turn, g.aiBusy, ch01Blockers(g))
	}
	r.checkpoint("after_enemy_phase", action.Seq, r.ui(), true, extra...)
}

func (r *parityReplay) forceEnemyClear(action parityAction) {
	g := r.g
	// 與 oracle 的 force_enemy_clear 相同的注入：把場上 camp 0 的 HP 寫成 0。
	// 這是明示的修改路徑；收據會標記。
	cleared := 0
	for _, u := range g.st.Units {
		if u != nil && u.OnField && u.Camp == battle.Enemy && u.HP > 0 {
			u.HP = 0
			cleared++
		}
	}
	r.checkpoint("force_enemy_clear", action.Seq, r.ui(), false, fmt.Sprintf("cleared=%d（修改路徑）", cleared))
	// 原版在下一個勝負檢查點（單位行動收尾／回合邊界）發現敵方全滅就直接進戰後；
	// 重製端的同一檢查是 checkResult，這裡沒有行動可掛，所以直接呼叫。敵方回合
	// 正要開始時（見 endTurn 的 stopBeforeAI），先把 HP 歸零的記錄標成 +5＝1，
	// AI 掃描才不會讓屍體行動；沒有存活敵人的回合會自己收掉。
	g.st.MarkNativeDeadRecords()
	g.checkResult()
	if g.aiBusy && g.result == "" {
		pump(r.t, g, ch01FrameBudget*6, func() bool { return g.result != "" || r.playerHasControl() })
		g.checkResult()
	}
}

// currentTown 回報現在該在哪個城鎮節點：戰鬥還沒打就是本章城鎮，打完就是下一章的。
func (r *parityReplay) currentTown() string {
	if r.battleDone {
		return r.townAfter
	}
	return r.town
}

func (r *parityReplay) ensureTown() {
	t, g := r.t, r.g
	if g.camp.NodeID() == r.currentTown() {
		return
	}
	if g.camp.NodeID() == r.townAfter {
		r.battleDone = true
		return
	}
	// 戰鬥結束：等演出收掉，在正式 Enter 邊界確認，走戰後過場到城鎮。
	for frame := 0; frame < 3000 && g.camp.NodeID() == r.battle; frame++ {
		ackPresents(g)
		if resultInputReachable(g) && g.confirmBattleResult() {
			break
		}
		if (g.battleEvent != nil || g.nativeTurnStaging != nil) && len(g.dialog) > 0 {
			g.handleBattleEventDialogueInput(true)
		}
		if err := g.Update(); err != nil {
			t.Fatalf("Update：%v", err)
		}
	}
	if g.camp.NodeID() == r.battle {
		t.Fatalf("戰鬥沒有結束：result=%q\n阻塞：%s", g.result, ch01Blockers(g))
	}
	r.checkpoint("battle_result", 0, r.ui(), true)
	j := newJourneyTrace()
	driveStory(t, g, j, func() bool { return g.camp.NodeID() == r.townAfter })
	r.battleDone = true
	r.settleTown()
}

func (r *parityReplay) townProbe(action parityAction) {
	t, g := r.t, r.g
	r.ensureTown()
	delta := 1
	if action.Move == "right" {
		delta = -1
	}
	if !g.moveNativeTownSelection(delta) {
		t.Fatalf("town_enter(seq %d)：城鎮選擇無法移動", action.Seq)
	}
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

func (r *parityReplay) townSave(action parityAction) {
	g := r.g
	r.ensureTown()
	slot := 0
	if action.Slot != nil {
		slot = *action.Slot
	}
	g.saveGameToSlot(slot)
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
	r.checkpoint("town_save", action.Seq, "town", true, note)
}

// shopSell 對應驅動端的 shop_sell：武器店（選項 1）賣掉第一位隊員的第一件物品。
// 走的是 town-shop 三章收據同一組正式 owner。
func (r *parityReplay) shopSell(action parityAction) {
	t, g := r.t, r.g
	r.ensureTown()
	for g.campSel != 1 {
		if !g.moveNativeTownSelection(1) {
			t.Fatalf("shop_sell(seq %d)：無法走到武器店", action.Seq)
		}
	}
	r.enterTownOption(1)
	if g.nativeShopMode != "menu" || g.loadErr != "" {
		t.Fatalf("shop_sell(seq %d)：武器店 mode=%q node=%q err=%q", action.Seq, g.nativeShopMode, g.camp.NodeID(), g.loadErr)
	}
	r.checkpoint("shop_menu", action.Seq, "shop", false)
	g.nativeShopUIJob = nil
	if !g.setupNativeShopSellRoster() || !g.setupNativeShopSellItems() {
		t.Fatalf("shop_sell(seq %d)：正式 sell roster／item owner 無法建立", action.Seq)
	}
	g.nativeShopMode, g.nativeShopSellConfirmSel = "sell_confirm", 0
	if !g.beginNativeShopSellSuccess() || g.nativeShopUIJob == nil {
		t.Fatalf("shop_sell(seq %d)：正式 sell success owner 無法建立", action.Seq)
	}
	for i := 0; i < 2; i++ {
		after := g.nativeShopUIJob.after
		g.nativeShopUIJob = nil
		if after == nil {
			t.Fatalf("shop_sell(seq %d)：sell callback %d 缺失", action.Seq, i)
		}
		after()
	}
	g.leaveShop()
	pump(t, g, 120, func() bool { return g.camp.NodeID() == r.currentTown() })
	r.checkpoint("shop_sell", action.Seq, "town", true)
}

// secretShop 對應驅動端的 secret_shop：切到指定建築、送功能鍵 chord、enter 進店。
func (r *parityReplay) secretShop(action parityAction) {
	t, g := r.t, r.g
	r.ensureTown()
	for _, move := range action.Moves {
		delta := 1
		if move == "right" {
			delta = -1
		}
		g.moveNativeTownSelection(delta)
	}
	n := g.camp.Node()
	if n == nil || n.NativeSecretGate == nil {
		t.Fatalf("secret_shop(seq %d)：節點沒有祕密商店 gate", action.Seq)
	}
	if !g.revealNativeTownSecret(n.NativeSecretGate.ScanCode) || !g.camp.ConfirmNativeTownSecret(g.campSel) {
		r.checkpoint("secret_shop", action.Seq, "town", true, fmt.Sprintf("gate 未開：selection=%d scan=%02X", g.campSel, n.NativeSecretGate.ScanCode))
		return
	}
	g.enterNode()
	pump(t, g, 120, func() bool { return g.nativeClassUIJob == nil && g.fade == nil })
	ui := "shop"
	if g.nativeShopMode == "" {
		ui = r.ui()
	}
	r.checkpoint("secret_shop", action.Seq, ui, true)
}
