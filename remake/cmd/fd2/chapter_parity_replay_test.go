package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
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
	t.Setenv("FD2_NATIVE_SAVE", slot)
	t.Setenv("FD2_CUTSCENE_LOG", "1")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })

	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	r := &parityReplay{t: t, g: g, out: out,
		battle:    fmt.Sprintf("battle_ch%02d", chapter),
		town:      fmt.Sprintf("town_ch%02d", chapter),
		townAfter: fmt.Sprintf("town_ch%02d", chapter+1)}
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
	for _, action := range actions {
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
			r.endTurn(action)
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
		out = append(out, parityUnit{Camp: nativeCampCode(u.Camp), X: u.X, Y: u.Y, HP: u.HP, Identity: identity, Acted: acted})
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
		t.Fatalf("attack(seq %d)：原版可攻擊、重製端指令環攻擊不可選 availability=%v", action.Seq, available)
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
		if g.ring && nativeActionSelectable(g.actionOverlayAvailability(), 3) {
			g.ringSel = 3
			closeRing(t, g, g.finishSelectedWait)
			pump(t, g, ch01FrameBudget, func() bool { return actor.Acted || actor.HP <= 0 })
		}
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

func (r *parityReplay) endTurn(action parityAction) {
	t, g := r.t, r.g
	if g.camp.NodeID() != r.battle || g.result != "" {
		r.note(action, "end_turn 時已不在戰場")
		return
	}
	extra := []string{}
	if r.syncRNG(action) {
		extra = append(extra, "rng_synced")
	}
	before := g.st.Turn
	g.endTurn()
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
	// 原版側接著送 END（會另有 end_turn 動作）；清場後勝負判定在回合結束時發生。
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
	case n.Type == "preparation":
		g.camp.Advance("cancel")
		g.enterNode()
	case n.Type == "town":
	default:
		g.camp.Advance("cancel")
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
	note := "重製端槽存檔為自有格式（非原版 FD2.SAV 槽 bytes）"
	if _, err := os.Stat(saveSlotPath(slot)); err != nil {
		note = "重製端槽存檔未寫出：" + err.Error()
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
