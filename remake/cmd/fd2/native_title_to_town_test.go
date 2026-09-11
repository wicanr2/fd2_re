package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// 從標題 START 一路走到第一關之後的羅德鎮：開場動畫 → 標題選單 START → 序章處理器
// 過場 → 第一關 → 勝利 → 戰後過場 → 城鎮。
//
// 這條不是逐幀對拍。它記下的是**原版證據說得出來的東西**：每一句原生對白的
// 來源（FDTXT 檔、字串索引、句序）、按了幾次 Enter、戰場開局的單位、四個回合
// 事件、援軍的放置點、勝利的回合、城鎮入口的狀態，最後同一個種子跑兩次比對
// 兩份紀錄是否完全相同。每一份紀錄都由 compareJourneyToEvidence 逐項對照
// docs/data/ui-traces/title-to-town-ch01-e1.json 收據裡的原版證據。
//
// 輸入一律交給正式的輸入 owner（標題 applyTitleMenuEvent、故事
// handleNativeStoryInput、戰場事件 handleBattleEventDialogueInput、勝敗
// confirmBattleResult），而且只在 owner 真的在等鍵的那一幀送——等於原版對拍的
// kbd_empty 閘門。
//
//	FD2_TITLE_TO_TOWN=<輸出目錄> go test ./cmd/fd2 -run TestPlayTitleStartThroughChapterOneToTown
const (
	journeySeed          = "20260911"
	journeyTitleFrames   = 20000
	journeyStoryFrames   = 400000
	journeyStallFrames   = 6000
	journeyTownFrames    = 3000
	journeyBattleNode    = "battle_ch01"
	journeyPostNode      = "story_ch02"
	journeyTownNode      = "town_ch02"
	journeyOpeningNode   = "story_ch00_handler"
	journeyTownSettleMax = 600
)

type journeyLine struct {
	Node      string `json:"node"`
	Phase     string `json:"phase"`
	Source    string `json:"source,omitempty"`
	String    int    `json:"string"`
	Utterance int    `json:"utterance"`
	Control   string `json:"control,omitempty"`
	Text      string `json:"text,omitempty"`
	Pages     int    `json:"pages"`
	Turn      int    `json:"turn,omitempty"`
	Presses   int    `json:"presses_before"`
}

type journeyUnit struct {
	Camp   int    `json:"camp"`
	Group  int    `json:"group"`
	Record [2]int `json:"record"`
	At     [2]int `json:"at"`
	HP     int    `json:"hp"`
	ID     int    `json:"identity"`
	HasID  bool   `json:"has_identity"`
}

type journeySpawn struct {
	Turn    int    `json:"turn"`
	Camp    int    `json:"camp"`
	Group   int    `json:"group"`
	Record  [2]int `json:"record"`
	First   [2]int `json:"first_seen"`
	Settled [2]int `json:"settled"`
	settled bool
	unit    *battle.Unit
}

type journeyMilestone struct {
	Presses int           `json:"presses"`
	Node    string        `json:"node"`
	Turn    int           `json:"turn"`
	Own     int           `json:"own_on_field"`
	Enemy   int           `json:"enemy_on_field"`
	Ally    int           `json:"ally_on_field"`
	Pending int           `json:"enemy_pending"`
	Units   []journeyUnit `json:"units,omitempty"`
}

type journeyTown struct {
	Node       string      `json:"node"`
	Town       string      `json:"town"`
	Selection  int         `json:"selection"`
	Gold       int         `json:"gold"`
	JoinOrder  []int       `json:"join_order"`
	Members    []int       `json:"members"`
	RosterHPMP [][3]int    `json:"roster_id_hp_mp"`
	RosterLv   [][4]int    `json:"roster_id_lv_exp_maxhp"`
	Presses    int         `json:"presses"`
	Variant    interface{} `json:"variant,omitempty"`
}

type journeyTrace struct {
	Seed           string            `json:"seed"`
	Nodes          []string          `json:"nodes"`
	Lines          []journeyLine     `json:"lines"`
	Presses        int               `json:"presses"`
	PressesByPhase map[string]int    `json:"presses_by_phase"`
	FirstMapLine   *journeyMilestone `json:"first_battlefield_line"`
	PlayerControl  *journeyMilestone `json:"player_control"`
	Spawns         []*journeySpawn   `json:"spawns"`
	Result         string            `json:"result"`
	ResultTurn     int               `json:"result_turn"`
	Town           *journeyTown      `json:"town"`
	Frames         int               `json:"frames"`

	lastNode    string
	lastLineSig string
	known       map[*battle.Unit]bool
}

func newJourneyTrace() *journeyTrace {
	return &journeyTrace{
		Seed: journeySeed, PressesByPhase: map[string]int{},
		known: map[*battle.Unit]bool{},
	}
}

func journeyPhase(g *Game) string {
	switch {
	case g.titlePhase != "":
		return "title"
	case g.camp == nil || g.camp.Node() == nil:
		return "none"
	}
	switch g.camp.NodeID() {
	case journeyOpeningNode:
		return "opening"
	case journeyBattleNode:
		return "battle"
	case journeyPostNode:
		return "postbattle"
	}
	return g.camp.Node().Type
}

func (j *journeyTrace) press(phase string) {
	j.Presses++
	j.PressesByPhase[phase]++
}

func journeyTurn(g *Game) int {
	if g.st == nil {
		return 0
	}
	return g.st.Turn
}

// observe 每一幀呼叫一次：節點轉換、新出現的對白行、新登場的單位。
func (j *journeyTrace) observe(g *Game) {
	j.Frames++
	node := ""
	if g.camp != nil {
		node = g.camp.NodeID()
		if n := g.camp.Node(); n != nil && node != j.lastNode {
			j.Nodes = append(j.Nodes, node+":"+n.Type)
		}
	}
	j.lastNode = node

	if len(g.dialog) == 0 {
		j.lastLineSig = ""
	} else {
		top := g.dialog[len(g.dialog)-1]
		line := journeyLine{Node: node, Phase: journeyPhase(g), Presses: j.Presses, Turn: journeyTurn(g)}
		if top.NativeDialogue != nil {
			nd := top.NativeDialogue
			line.Source, line.String, line.Utterance = nd.SourceDAT, nd.StringIndex, nd.Utterance
			line.Control, line.Pages = nd.Control, len(nd.Pages)
		} else {
			line.String, line.Utterance, line.Text, line.Pages = -1, -1, top.Text, dlgPageCount(top)
		}
		sig := fmt.Sprintf("%s|%d|%d|%s|%d|%d|%s", node, g.beatIdx, len(g.dialog),
			line.Source, line.String, line.Utterance, line.Text)
		if sig != j.lastLineSig {
			j.Lines = append(j.Lines, line)
			if j.FirstMapLine == nil && line.Source == "FDTXT_001" {
				j.FirstMapLine = j.milestone(g, false)
			}
		}
		j.lastLineSig = sig
	}

	if g.st != nil {
		for _, u := range g.st.Units {
			if u == nil || j.known[u] || !u.OnField {
				continue
			}
			j.known[u] = true
			if node != journeyBattleNode || j.PlayerControl == nil {
				continue // 開局的單位記在 player_control，不算援軍
			}
			j.Spawns = append(j.Spawns, &journeySpawn{
				Turn: g.st.Turn, Camp: int(u.Camp), Group: u.Group,
				Record: [2]int{int(u.NativePositionRecord.XWord), int(u.NativePositionRecord.YWord)},
				First:  [2]int{u.X, u.Y}, unit: u,
			})
		}
	}
}

// settleSpawns 在玩家重新取得操作權時補記援軍最後站定的格（進場 ACT 走位之後）。
func (j *journeyTrace) settleSpawns() {
	for _, s := range j.Spawns {
		if !s.settled && s.unit != nil {
			s.Settled, s.settled = [2]int{s.unit.X, s.unit.Y}, true
		}
	}
}

func (j *journeyTrace) milestone(g *Game, withUnits bool) *journeyMilestone {
	m := &journeyMilestone{Presses: j.Presses, Turn: journeyTurn(g)}
	if g.camp != nil {
		m.Node = g.camp.NodeID()
	}
	if g.st == nil {
		return m
	}
	for _, u := range g.st.Units {
		if u == nil || !u.OnField || !u.Alive() {
			continue
		}
		switch u.Camp {
		case battle.Own:
			m.Own++
		case battle.Enemy:
			m.Enemy++
		case battle.Ally:
			m.Ally++
		}
		if withUnits {
			m.Units = append(m.Units, journeyUnit{
				Camp: int(u.Camp), Group: u.Group,
				Record: [2]int{int(u.NativePositionRecord.XWord), int(u.NativePositionRecord.YWord)},
				At:     [2]int{u.X, u.Y}, HP: u.HP, ID: u.NativeIdentity, HasID: u.HasNativeIdentity,
			})
		}
	}
	m.Pending = g.st.PendingCount(battle.Enemy)
	return m
}

// storyEnterReady 回報故事對白是否停在等鍵：逐字寫完、沒有在收框、沒有在翻頁。
func storyEnterReady(g *Game) bool {
	if len(g.dialog) == 0 || g.dlgPhase != 0 || g.dlgScrollT != 0 || g.nativeDialogueClosingLive {
		return false
	}
	top := g.dialog[len(g.dialog)-1]
	if top.NativeDialogue == nil {
		return len(g.storyWalks) == 0 && g.fade == nil
	}
	return g.dlgPage >= 0 && g.dlgPage < len(g.nativeDialogueProgressive) &&
		len(g.nativeDialogueProgressive[g.dlgPage]) > 0 &&
		g.nativeDialogueProgress >= len(g.nativeDialogueProgressive[g.dlgPage])-1
}

func storySignature(g *Game) string {
	node := ""
	if g.camp != nil {
		node = g.camp.NodeID()
	}
	return fmt.Sprintf("%s|%d|%d|%d|%d|%v", node, g.beatIdx, len(g.dialog), g.dlgPage,
		g.nativeDialogueProgress, g.nativeDialogueClosingLive)
}

// progressSignature 只用來偵測「卡住」：幾千幀都沒變就停下來列阻塞欄位。
func progressSignature(g *Game) string {
	s := storySignature(g)
	if g.st != nil {
		s += fmt.Sprintf("|t%d|ai%v|r%s", g.st.Turn, g.aiBusy, g.result)
	}
	return fmt.Sprintf("%s|%s|%d|%v|%v", s, g.titlePhase, g.titleFlash,
		g.battleEvent != nil, g.nativeTurnStaging != nil)
}

// driveStory 推過一個故事／處理器過場節點，直到節點換成 stop 回報 true。
func driveStory(t *testing.T, g *Game, j *journeyTrace, stop func() bool) {
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

// playerHasControl：戰場上沒有任何演出、對白、事件或 AI 在跑，而且有我方單位
// 還沒行動。
func playerHasControl(g *Game) bool {
	return g.camp != nil && g.camp.NodeID() == journeyBattleNode && g.st != nil &&
		g.result == "" && !g.aiBusy && g.battleEvent == nil && g.nativeTurnStaging == nil &&
		len(g.dialog) == 0 && g.walk == nil && g.atk == nil && !g.ring &&
		g.nativeClassUIJob == nil && g.spawnIntroTransition == nil && g.indexedTransition == nil &&
		g.nativeUnitPresent == nil && g.actJob == nil && g.camPan == nil && g.focusJob == nil &&
		len(pendingOwn(g)) > 0
}

// resultInputReachable 近似 Update 走到 campInput 之前的那些提早 return：
// 演出、事件、對白、指令環與 AI 都停了，Enter 才會落到勝敗確認上。
func resultInputReachable(g *Game) bool {
	return g.result != "" && !g.aiBusy && g.atk == nil && g.walk == nil &&
		g.battleEvent == nil && g.nativeTurnStaging == nil && len(g.dialog) == 0 &&
		!g.ring && g.nativeClassUIJob == nil && g.spawnIntroTransition == nil &&
		g.nativeUnitPresent == nil && g.nativeAIIdleRecovery == nil &&
		g.nativeHealPresentation == nil && g.nativeModifierPresentation == nil &&
		g.nativeAICommandModifier == nil && g.nativeAIItemPresentation == nil &&
		g.nativeCmd0Presentation == nil && g.nativeCmd1Presentation == nil
}

func runTitleToTownJourney(t *testing.T, out string) *journeyTrace {
	t.Helper()
	t.Setenv("FD2_TITLE", "1")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	t.Setenv("FD2_SHOT_AI", "1")
	t.Setenv("FD2_SEED", journeySeed)

	j := newJourneyTrace()
	ch01FrameObserver = j.observe
	defer func() { ch01FrameObserver = nil }()

	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if g.titlePhase != "cutscene" {
		t.Fatalf("沒有從開場動畫開始：titlePhase=%q", g.titlePhase)
	}

	// 1. 開場動畫整段播完，進標題選單。不送鍵：原版開場可以用鍵中斷當前一幕，
	//    但這一條要的是「看完開場再選 START」。
	for frame := 0; g.titlePhase != "menu"; frame++ {
		if frame > journeyTitleFrames {
			t.Fatalf("開場動畫 %d 幀內沒有進選單：phase=%q", journeyTitleFrames, g.titlePhase)
		}
		j.observe(g)
		if err := g.Update(); err != nil {
			t.Fatalf("Update：%v", err)
		}
	}
	if g.titleSel != 0 {
		t.Fatalf("標題選單的初始選項不是 START：%d", g.titleSel)
	}
	if !g.applyTitleMenuEvent(TitleMenuConfirm) {
		t.Fatal("START 沒有被標題輸入 owner 接收")
	}
	j.press("title")
	for frame := 0; g.titlePhase != ""; frame++ {
		if frame > 600 {
			t.Fatalf("START 之後標題沒有收掉：phase=%q", g.titlePhase)
		}
		j.observe(g)
		if err := g.Update(); err != nil {
			t.Fatalf("Update：%v", err)
		}
	}
	if g.camp == nil || g.camp.NodeID() != journeyOpeningNode {
		t.Fatalf("START 之後不是序章：%v", g.camp)
	}

	// 2. 序章處理器過場，直到第一關的戰場節點。
	driveStory(t, g, j, func() bool { return g.camp.NodeID() == journeyBattleNode })

	// 3. 戰場上的開局對白與事件，推到玩家取得操作權。
	if !pump(t, g, journeyStoryFrames, func() bool {
		if playerHasControl(g) {
			return true
		}
		if (g.battleEvent != nil || g.nativeTurnStaging != nil) && len(g.dialog) > 0 && storyEnterReady(g) {
			before := storySignature(g)
			g.handleBattleEventDialogueInput(true)
			if storySignature(g) != before {
				j.press("battle-opening")
			}
		}
		return false
	}) {
		t.Fatalf("第一關沒有交出操作權\n阻塞：%s", ch01Blockers(g))
	}
	j.PlayerControl = j.milestone(g, true)

	// 4. 打到分出勝負。戰場事件的對白由 pump 逐幀代按（與單獨跑第一關相同），
	//    這裡不把那些 Enter 算進開場的按鍵數。
	log, err := os.Create(filepath.Join(out, "rounds.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	rec := &ch01Recorder{out: out, log: log}
	rec.note(g, "battle-start")
	settle := func(g *Game) {
		j.observe(g)
		if playerHasControl(g) {
			j.settleSpawns()
		}
	}
	ch01FrameObserver = settle
	playChapterOneRounds(t, g, rec)
	rec.note(g, "final")
	j.Result, j.ResultTurn = g.result, journeyTurn(g)
	if g.result != "win" {
		t.Fatalf("第一關沒有打贏：result=%q", g.result)
	}

	// 5. 勝負已分：等攻擊演出與事件都收掉（Update 的輸入分派到得了 campInput），
	//    再在正式的 Enter 邊界確認，交給戰後節點。
	for frame := 0; frame < 3000 && g.camp.NodeID() == journeyBattleNode; frame++ {
		j.observe(g)
		ackPresents(g)
		if resultInputReachable(g) && g.confirmBattleResult() {
			j.press("result")
			break
		}
		if err := g.Update(); err != nil {
			t.Fatalf("Update：%v", err)
		}
	}
	if g.loadErr != "" {
		t.Fatalf("勝利確認之後執行期錯誤：%s", g.loadErr)
	}

	// 6. 戰後過場，直到城鎮。
	driveStory(t, g, j, func() bool { return g.camp.NodeID() == journeyTownNode })

	// 7. 城鎮入口：等開場演出收掉再記狀態。
	for frame := 0; frame < journeyTownSettleMax; frame++ {
		j.observe(g)
		if g.nativeClassUIJob == nil && g.fade == nil && g.transitionReveal == nil &&
			g.indexedTransition == nil && frame > 10 {
			break
		}
		ackPresents(g)
		if err := g.Update(); err != nil {
			t.Fatalf("Update：%v", err)
		}
		if g.loadErr != "" {
			t.Fatalf("城鎮入口執行期錯誤：%s", g.loadErr)
		}
	}
	n := g.camp.Node()
	town := &journeyTown{Node: g.camp.NodeID(), Selection: g.campSel, Gold: g.gold,
		JoinOrder: append([]int(nil), g.partyJoinOrder...), Presses: j.Presses}
	if n != nil {
		town.Town = n.Town
		if n.NativeTownVariant != nil {
			town.Variant = *n.NativeTownVariant
		}
	}
	for _, id := range g.partyJoinOrder {
		if g.partyMembers[id] {
			town.Members = append(town.Members, id)
		}
		if u, ok := g.partyRoster[id]; ok {
			town.RosterHPMP = append(town.RosterHPMP, [3]int{id, u.HP, u.MP})
			town.RosterLv = append(town.RosterLv, [4]int{id, u.Lv, int(u.Exp), u.MaxHP})
		}
	}
	j.Town = town
	return j
}

func (j *journeyTrace) digest() string {
	frames := j.Frames
	j.Frames = 0
	raw, _ := json.Marshal(j)
	j.Frames = frames
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func writeJourney(t *testing.T, path string, j *journeyTrace) {
	t.Helper()
	raw, err := json.MarshalIndent(j, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// journeyEvidence 是 docs/data/ui-traces/title-to-town-ch01-e1.json 的 expected
// 區段：每一項都出自原版（處理器位元組碼、原版 FD2.SAV 的快照與章節槽、呼叫點前的
// push imm），出處寫在收據的 original_sources。
type journeyEvidence struct {
	Expected struct {
		OpeningCalls      [][2]any `json:"opening_calls"`
		OpeningUtterances int      `json:"opening_utterances"`
		Turn1Runtime      []struct {
			X, Y     int
			Identity int `json:"identity"`
			RawCamp  int `json:"raw_camp"`
			Byte5    int `json:"byte5"`
			HP       int `json:"hp"`
		} `json:"turn1_runtime"`
		BattleEventStrings [][2]int `json:"battle_event_strings"`
		Postbattle         struct {
			Source     string `json:"source"`
			String     int    `json:"string"`
			Utterances int    `json:"utterances"`
		} `json:"postbattle"`
		Town struct {
			Node            string         `json:"node"`
			Name            string         `json:"name"`
			Gold            int            `json:"gold"`
			JoinOrder       []int          `json:"join_order"`
			UnlevelledMaxHP map[string]int `json:"unlevelled_max_hp"`
		} `json:"town"`
		OriginalKeyReadsToControl [2]int `json:"original_key_reads_to_control"`
	} `json:"expected"`
}

// compareJourneyToEvidence 逐項對原版證據。這裡只斷言原版說得出來的東西；戰局
// （誰陣亡、第幾回合勝、誰升級）取決於測試自己的戰術，不拿來比。
func compareJourneyToEvidence(t *testing.T, j *journeyTrace) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "data", "ui-traces", "title-to-town-ch01-e1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ev journeyEvidence
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatal(err)
	}
	want := ev.Expected

	// 1. 序章：19 次對白呼叫的 (FDTXT, 字串) 順序與 97 句。
	var calls [][2]any
	utterances := 0
	var post []journeyLine
	var battleCalls [][2]int
	for _, l := range j.Lines {
		switch l.Node {
		case journeyOpeningNode:
			utterances++
			if n := len(calls); n == 0 || calls[n-1][0] != l.Source || calls[n-1][1] != float64(l.String) {
				calls = append(calls, [2]any{l.Source, float64(l.String)})
			}
		case journeyBattleNode:
			if n := len(battleCalls); n == 0 || battleCalls[n-1] != [2]int{l.Turn, l.String} {
				battleCalls = append(battleCalls, [2]int{l.Turn, l.String})
			}
		case journeyPostNode:
			post = append(post, l)
		}
	}
	if fmt.Sprint(calls) != fmt.Sprint(want.OpeningCalls) {
		t.Errorf("序章對白呼叫順序與原版處理器不同：\n重製 %v\n原版 %v", calls, want.OpeningCalls)
	}
	if utterances != want.OpeningUtterances {
		t.Errorf("序章 %d 句，原版 %d 句", utterances, want.OpeningUtterances)
	}

	// 2. 第一關取得操作權時的單位：與原版 FD2.SAV 第 1 回合快照逐筆比。
	pc := j.PlayerControl
	own, enemy := 0, 0
	for _, r := range want.Turn1Runtime {
		if r.Byte5&1 != 0 {
			continue // 不啟用的列：原版留在陣列裡，重製端不上場
		}
		if r.RawCamp == 2 {
			own++
		} else {
			enemy++
		}
		found := false
		for _, u := range pc.Units {
			if u.At == [2]int{r.X, r.Y} && u.HP == r.HP &&
				(r.RawCamp != 2 || (u.HasID && u.ID == r.Identity)) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("原版第 1 回合的單位 (%d,%d) 身分%d hp%d 在重製端找不到", r.X, r.Y, r.Identity, r.HP)
		}
	}
	if pc.Own != own || pc.Enemy != enemy || pc.Ally != 0 {
		t.Errorf("開局 我方%d 敵方%d 友軍%d，原版 我方%d 敵方%d 友軍0", pc.Own, pc.Enemy, pc.Ally, own, enemy)
	}

	// 3. 四個回合事件的 (回合, 字串) 順序。
	if fmt.Sprint(battleCalls) != fmt.Sprint(want.BattleEventStrings) {
		t.Errorf("戰場事件對白與原版不同：重製 %v，原版 %v", battleCalls, want.BattleEventStrings)
	}

	// 4. 戰後過場：FDTXT_001 字串 9、13 句。
	if len(post) != want.Postbattle.Utterances {
		t.Errorf("戰後 %d 句，原版 %d 句", len(post), want.Postbattle.Utterances)
	}
	for _, l := range post {
		if l.Source != want.Postbattle.Source || l.String != want.Postbattle.String {
			t.Errorf("戰後出現 %s 字串%d，原版只有 %s 字串%d", l.Source, l.String,
				want.Postbattle.Source, want.Postbattle.String)
			break
		}
	}

	// 5. 城鎮入口：節點、名稱、金幣、隊伍順序、沒升級的人的 MaxHP。
	town := j.Town
	if town.Node != want.Town.Node || town.Town != want.Town.Name || town.Gold != want.Town.Gold ||
		fmt.Sprint(town.JoinOrder) != fmt.Sprint(want.Town.JoinOrder) {
		t.Errorf("城鎮入口 %s/%s 金幣%d 隊伍%v，原版 %s/%s 金幣%d 隊伍%v",
			town.Node, town.Town, town.Gold, town.JoinOrder,
			want.Town.Node, want.Town.Name, want.Town.Gold, want.Town.JoinOrder)
	}
	for _, row := range town.RosterLv {
		if row[1] != 1 {
			continue
		}
		if maxHP, ok := want.Town.UnlevelledMaxHP[fmt.Sprint(row[0])]; ok && maxHP != row[3] {
			t.Errorf("身分%d 沒升級，MaxHP %d，原版 %d", row[0], row[3], maxHP)
		}
	}

	t.Logf("取得操作權前按了 %d 次 Enter；原版開場長度收據在 HUD 出現時讀了 %d..%d 次鍵（原版那一臂開場動畫也在送鍵，數字只能比範圍）",
		pc.Presses, want.OriginalKeyReadsToControl[0], want.OriginalKeyReadsToControl[1])
}

func TestPlayTitleStartThroughChapterOneToTown(t *testing.T) {
	out := os.Getenv("FD2_TITLE_TO_TOWN")
	if out == "" {
		t.Skip("需要 FD2_TITLE_TO_TOWN 指向輸出目錄")
	}
	var digests []string
	for run := 1; run <= 2; run++ {
		dir := filepath.Join(out, fmt.Sprintf("run%d", run))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		j := runTitleToTownJourney(t, dir)
		writeJourney(t, filepath.Join(dir, "journey.json"), j)
		compareJourneyToEvidence(t, j)
		digests = append(digests, j.digest())
		t.Logf("第 %d 次：%d 幀、%d 次 Enter、%d 句對白、結果 %s（第 %d 回合）、城鎮 %s",
			run, j.Frames, j.Presses, len(j.Lines), j.Result, j.ResultTurn, j.Town.Town)
	}
	if digests[0] != digests[1] {
		t.Fatalf("同一個種子跑兩次，紀錄不一樣：%s ≠ %s（見 run1/run2 的 journey.json）",
			digests[0][:16], digests[1][:16])
	}
}
