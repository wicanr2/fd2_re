// fd2-golden-rules-int 輸出 internal/battle 跨組整合的黃金差分資料，供 Rust 改寫
// remake-rs/crates/fd2-rules 逐步比對（docs/goal/119-goal-rust-bevy-rewrite-20261113.md 的 R2
// 完成條件：以相同種子跑完整的 AI 戰鬥，Rust 與 Go 逐回合狀態相同）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 battle 的公開 API，不改任何遊戲行為。
//
// 真實輸入：remake/assets 的 30 份劇本 scenarios/ch*.json（各自載入 map<劇本.map>）與沒有劇本使用的
// 地圖（map30～map32），合計 33 張地圖；路徑排序後連同大小與 SHA-256 寫進黃金檔。另有 spells.json、
// 移動成本列、物品效果列、指令抗性、學招表、成長槽與 item.json（AICommandSpell 由 campaign 套件
// 算好後以資料寫進黃金檔，Rust 端直接使用）。
//
// 戰鬥重播（battles）：每個戰場以多個固定種子各跑一場。cmd/fd2 的 AI 回合（main.go 的 aiStep、
// endTurn、finishTurn、completeTurn 與 native_ai_*_execute.go）掛在 Ebiten 的逐幀狀態機與演出上，
// 無法在工具內直接執行；本工具依同一段程式的順序把其中改寫 battle 狀態的呼叫組成無頭驅動
// （headless driver），只用 battle 的公開 API。組裝方式與刻意省略的部分寫在 driver 型別的註解，
// Rust 端 tests/golden_rules_int.rs 依同一份規則重做。每個 AI 計畫、每個回合邊界都記一步：計畫的
// 投影、執行結果、改變的單位欄位（json.Marshal(Unit) 的鍵）、改變的戰場摘要、執行後 math/rand
// 序列的下一個 Int63（不消耗）與原版 RNG 狀態。戰鬥結束再存一份完整 StateSnapshot。
//
// 系統群行軍（march）：每個戰場（不分種子）以固定種子挑選目的格呼叫 PlanNativeSystemGroupMarch。
//
// combat.go 202、235、621、921 行在 reach（map）裡以嚴格 < 挑最近格，距離平手時取 Go 先迭代到的；
// 工具在 normalized 規劃器實際走到這幾個迴圈時另外計算平手，寫進該步的 tie 欄位（只供診斷，
// 不參與比對），Rust 端遇到分歧先查這一欄。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	randv2 "math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

type obj = map[string]any

const (
	spellsPath      = "assets/spells.json"
	costRowsPath    = "assets/data/native_movement_cost_rows.json"
	itemRowsPath    = "assets/data/native_item_effect_rows.json"
	resistancesPath = "assets/data/native_command_resistances.json"
	learnPath       = "assets/data/command_learn.json"
	learnSelPath    = "assets/data/class_change_growth.json"
	growthPath      = "assets/data/native_growth_slots.json"
	itemJSONPath    = "assets/data/item.json"
	controlsPath    = "assets/maps/native_turn_event_controls.json"
)

// 驅動的上限：回合數與單一階段的計畫數（防止無窮迴圈；碰到就記 stopped）。
const (
	maxTurns       = 24
	maxPhasePlans  = 400
	protectName    = "索爾"
	nativeRestPose = 0
)

// seeds 是每場戰鬥的 math/rand 種子與原版 RNG（0x627B8）初值。
var seeds = []struct {
	Rand   int64
	Native uint16
}{{20261114, 0x1234}, {7, 0xBEEF}, {-3, 0}}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func errv(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
}

func mustJSON(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func decodeObj(data []byte) obj {
	var m obj
	if err := json.Unmarshal(data, &m); err != nil {
		panic(err)
	}
	return m
}

// ---- 資產 ----

type assets struct {
	book     []battle.NativeCommandRecord
	spells   []battle.Spell
	costRows [][]byte
	itemRows []byte
	resist   map[int]int
	learn    map[int][]battle.CommandLearnEntry
	learnSel map[int]int
	growth   map[int]battle.GrowthRow
	aiCmd    map[int]int
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func loadAssets() *assets {
	return &assets{
		book:     must(battle.LoadNativeCommandRecords(spellsPath)),
		spells:   must(battle.LoadSpells(spellsPath)),
		costRows: must(battle.LoadNativeMovementCostRows(costRowsPath)),
		itemRows: must(battle.LoadNativeItemEffectRowPrefix(itemRowsPath)),
		resist:   must(battle.LoadNativeCommandResistances(resistancesPath)),
		learn:    must(battle.LoadCommandLearn(learnPath)),
		learnSel: must(battle.LoadCommandLearnSelectors(learnSelPath)),
		growth:   must(battle.LoadNativeGrowthSlots(growthPath)),
		aiCmd:    must(campaign.LoadAICommandSpellMap(itemJSONPath)),
	}
}

// rawify 補齊 NativeAIScoringRecords 需要、劇本隊伍沒有的原版 provenance（cmd/fd2 由章節存檔
// 與 handler 建構，本工具沒有那一層）。規則固定、不用亂數，Rust 端照做；已有的欄位不動。
func rawify(units []*battle.Unit) {
	for _, u := range units {
		if !u.HasNativeMapPresentation {
			u.NativeMapPresentation = battle.NativeMapPresentationState{X: byte(u.X), Y: byte(u.Y)}
			u.HasNativeMapPresentation = true
		}
		if !u.HasNativeRecordByte5 {
			u.NativeRecordByte5 = 0
			if !u.OnField || u.HP <= 0 {
				u.NativeRecordByte5 = 1
			}
			u.HasNativeRecordByte5 = true
		}
		if !u.HasNativeRecordByte6 {
			u.NativeRecordByte6 = []byte{2, 1, 0}[int(u.Camp)%3]
			u.HasNativeRecordByte6 = true
		}
		if !u.HasBattleFig {
			u.BattleFig, u.HasBattleFig = 4, true
		}
		if !u.HasNativeRecordByte34 {
			u.NativeRecordByte34, u.HasNativeRecordByte34 = 0, true
		}
		if !u.HasNativeRecordByte35 {
			u.NativeRecordByte35, u.HasNativeRecordByte35 = 0, true
		}
		if !u.HasNativeRecordByte36 {
			u.NativeRecordByte36, u.HasNativeRecordByte36 = 0, true
		}
		if !u.HasNativeRecordWord42 {
			u.NativeRecordWord42, u.HasNativeRecordWord42 = uint16(u.MaxHP), true
		}
		if !u.HasNativeRecordWord46 {
			u.NativeRecordWord46, u.HasNativeRecordWord46 = uint16(u.MaxMP), true
		}
	}
}

// setup 載入地圖、綁定資產（cmd/fd2 main.go 3904～3920、11826～11883 的綁定）、套用劇本，再 rawify。
func setup(a *assets, mapPath, scenarioPath string) (*battle.State, *battle.Scenario, obj) {
	st, err := battle.Load(mapPath)
	if err != nil {
		panic(fmt.Sprintf("%s: %v", mapPath, err))
	}
	info := obj{}
	info["bind_items"] = errv(st.BindNativeFutureItemRows(a.itemRows))
	info["bind_cost"] = errv(st.BindNativeMovementCostRows(a.costRows))
	st.NativeCommandBook = append([]battle.NativeCommandRecord(nil), a.book...)
	st.NativeCommandResistances = a.resist
	st.CommandLearn, st.CommandLearnSelectors = a.learn, a.learnSel
	st.NativeGrowthRows = a.growth
	st.SpellBook = append([]battle.Spell(nil), a.spells...)
	st.AICommandSpell = a.aiCmd
	var sc *battle.Scenario
	if scenarioPath != "" {
		sc, err = battle.LoadScenario(scenarioPath)
		if err != nil {
			panic(fmt.Sprintf("%s: %v", scenarioPath, err))
		}
		if len(sc.NativeDeathPrograms) != 0 {
			st.NativeDeathExpCancel = sc.NativeDeathCancelsExp
		}
		lines, err := sc.SetupChecked(st)
		info["setup_err"], info["setup_dialogues"] = errv(err), len(lines)
		if err != nil {
			panic(fmt.Sprintf("%s: %v", scenarioPath, err))
		}
	}
	rawify(st.Units)
	return st, sc, info
}

// ---- 投影 ----

func index(st *battle.State, u *battle.Unit) int {
	if u == nil {
		return -1
	}
	for i, x := range st.Units {
		if x == u {
			return i
		}
	}
	return -2
}

func cell(c battle.Cell) []int { return []int{c.X, c.Y} }

func cells(cs []battle.Cell) [][]int {
	out := make([][]int, 0, len(cs))
	for _, c := range cs {
		out = append(out, cell(c))
	}
	return out
}

func recoveryDecision(d *battle.NativeAIIdleRecoveryDecision) any {
	if d == nil {
		return nil
	}
	return []any{d.Unit, d.Accepted, d.CurrentHP, d.MaximumHP, d.NextHP, d.RawGate25, d.RawGate26}
}

func ints(b []byte) []int {
	out := make([]int, len(b))
	for i, v := range b {
		out[i] = int(v)
	}
	return out
}

// planProj 是一個 AIPlan 的完整投影（*Unit 寫成 State.Units 索引，nil 為 -1）。
func planProj(st *battle.State, p *battle.AIPlan) any {
	if p == nil {
		return nil
	}
	o := obj{
		"u": index(st, p.U), "path": cells(p.Path), "target": index(st, p.Target), "spell": p.SpellID,
		"kind": int(p.NativeActionKind), "cmd": p.NativeCommandID, "slot": p.NativeItemSlot, "item": p.NativeItemID,
		"item_targets": ints(p.NativeItemTargetIndices), "dest": cell(p.NativeActionDestination),
		"intended": cell(p.NativeModeIntended), "cands": cells(p.NativeModeCandidates),
		"blocked": cell(p.NativeModeBlockedCell), "blocked_found": p.NativeModeBlockedFound,
		"opposite": cells(p.NativeModeOpposite), "route": int(p.NativeAI14EF0Route), "score": p.NativeActionScore,
		"fb_active": p.NativeModeFallbackActive, "fb": int(p.NativeModeFallback), "wb5": p.NativeModeWriteByte5,
		"rz": p.NativeModeWriteRangeZero, "focus": p.NativeModeFocusActor, "ev": p.NativeModeEventActive,
		"ev_id": int(p.NativeModeEventID), "ev_dest": cell(p.NativeModeEventDestination), "m2p": p.NativeMode2Physical,
		"idle": recoveryDecision(p.NativeIdleRecovery), "fb_idle": recoveryDecision(p.NativeFallbackIdleRecovery),
		"err": errv(p.NativeError), "scored": p.NativeScoredCommands,
	}
	// nil 與空切片在執行端相同（aiStep 只看長度），只在非空時寫出。
	if len(p.NativeMode11Stages) > 0 {
		stages := make([]any, 0, len(p.NativeMode11Stages))
		for _, s := range p.NativeMode11Stages {
			stages = append(stages, obj{"ord": s.Stage.Ordinal, "route": int(s.Stage.Route),
				"action": planProj(st, s.Action), "rec": recoveryDecision(s.Recovery)})
		}
		o["m11"] = stages
	}
	return o
}

func levelUps(ev []battle.LevelUpEvent) [][]int {
	out := make([][]int, 0, len(ev))
	for _, e := range ev {
		row := []int{e.NewLv, e.ApGain, e.DpGain, e.DxGain, e.HpGain, e.MpGain}
		out = append(out, append(row, e.LearnedCommandIDs...))
	}
	return out
}

func attackProj(a *battle.AttackResult) any {
	if a == nil {
		return nil
	}
	return []any{a.Amount, a.Missed, a.Crit, a.ExpGained, levelUps(a.LevelUps)}
}

// extras 是單位以外、每一步都比對的戰場欄位（只在改變時寫出）。
func extras(st *battle.State) obj {
	flags := []string{}
	for k, v := range st.Flags {
		if v {
			flags = append(flags, k)
		}
	}
	sort.Strings(flags)
	pending := []int{}
	for g, v := range st.PendingGroups {
		if v {
			pending = append(pending, g)
		}
	}
	sort.Ints(pending)
	opened := []int{}
	for k, v := range st.OpenedTreasure {
		if v {
			opened = append(opened, k)
		}
	}
	sort.Ints(opened)
	runtime := make([]string, 0, len(st.NativeRuntimeRecords))
	for _, r := range st.NativeRuntimeRecords {
		runtime = append(runtime, hex.EncodeToString(r.Raw[:]))
	}
	controls := make([][]int, 0, 16)
	for _, row := range st.NativeTurnEventControls {
		controls = append(controls, []int{int(row.Turn), int(row.EventID), int(row.RawCamp)})
	}
	chests := make([][]int, 0, 16)
	for _, c := range st.NativeChestControls {
		chests = append(chests, []int{int(c.RawType), int(c.Value)})
	}
	return obj{
		"turn": st.Turn, "round": st.NativeRoundCounter, "range": st.NativeMapRangeMode,
		"has_range": st.HasNativeMapRangeModeState, "flags": flags, "event_state": ints(st.NativeEventState[:]),
		"field_raw": hex.EncodeToString(st.NativeFieldControlRaw), "pending": pending, "units": len(st.Units),
		"roster": len(st.Roster), "opened": opened, "exp_acc": st.NativeExperienceAccumulator,
		"runtime": runtime, "controls": controls, "chests": chests, "treasures": len(st.Treasures),
		"view": st.HasNativeMapViewState, "proj": st.HasNativeRuntimeUnitProjection,
	}
}

// ---- 可窺看的 math/rand 來源 ----

// countingSource 包住 rand.NewSource(seed)，記下已消耗的步數；peek 以另一份同種子來源前進到
// 同一步再取下一個 Int63，不影響驅動用的序列。Rand 的取樣只呼叫 Int63／Uint64，兩者各前進一步。
type countingSource struct {
	src        rand.Source64
	n          int64
	shadow     rand.Source64
	shadowN    int64
	peekAt     int64
	peekVal    int64
	peekCached bool
}

func newCountingSource(seed int64) *countingSource {
	return &countingSource{src: rand.NewSource(seed).(rand.Source64), shadow: rand.NewSource(seed).(rand.Source64)}
}

func (c *countingSource) Int63() int64   { c.n++; return c.src.Int63() }
func (c *countingSource) Uint64() uint64 { c.n++; return c.src.Uint64() }
func (c *countingSource) Seed(int64)     { panic("countingSource 不重設種子") }

func (c *countingSource) peek() int64 {
	if c.peekCached && c.peekAt == c.n {
		return c.peekVal
	}
	for c.shadowN < c.n {
		c.shadow.Int63()
		c.shadowN++
	}
	v := c.shadow.Int63()
	c.shadowN++
	c.peekAt, c.peekVal, c.peekCached = c.n, v, true
	return v
}

// ---- 無頭驅動 ----

// driver 依 cmd/fd2 的順序組出一場只有 AI 行動的戰鬥（我方每回合直接 END）：
//
//	玩家階段開始：checkResult（main.go finishNativeTransientPlayerPhaseInput）。
//	END（main.go endTurn）：ApplyNativeEndTurnRecovery → selector 1 回合事件 → 暫時狀態掃描 selector 1
//	  → ResetNativeAIScan → 友軍遍 NextAllyAIPlan（aiStep，handled 且非 nil 才執行）。
//	友軍遍結束：全部單位清 +5 bit7 與 Acted（0x13536）→ selector 0 回合事件 → 掃描 selector 0
//	  → 敵軍兩遍 NextAIPlan 直到 nil。
//	finishTurn：沒有 selector 的 on_turn_end 事件 → completeTurn（Turn++、回合計數、Acted、bit7、
//	  TickStatus）→ 掃描 selector 2 → selector 2 回合事件。
//
// 每個計畫照 aiStep 的分支執行：mode 8 備援跳過；NativeError 停止整場；mode 11 逐段；0x13FD4 回復；
// 走路（stepBattleWalkSegment 的逐格 SetMapPose／SetNativeMapGridMotion／FinishNativeMapGridStep，
// 抵達時 SetMapPlacement 靜止姿態）；法術（executeAISpell 的前置檢查與 CastArea）；原生指令 13～16
// （NativeAICommandHealTargets、ExecuteNativeAICommandHealNative）；物理攻擊
// （AttackNativePhysicalWithExperience，接 checkResult）；收尾 finishSuccessfulUnitAction 的
// MarkNativeDeadRecords、全部靜止姿態與 Acted。
//
// 刻意省略（全屬 cmd/fd2 的演出或 campaign 套件，Rust 端照樣省略）：對白、鏡頭、音效；死亡獎勵與
// 死亡程式；格子事件；event 76／79 與 raw camp 0／2 回合事件；帶演出的出場（spawn_group_with_intro）；
// 其他原生指令（0～12、17～35）與道具的效果演出（只設姿態、清經驗累計並收尾）；暫時狀態到期時的裝備
// 重算（campaign.NativeShopEquipmentRecordForUnit）。保護角色以名稱「索爾」或 NativeIdentity 0 的我方
// 單位判定（cmd/fd2 經語系檔解析名稱）。
type driver struct {
	st      *battle.State
	sc      *battle.Scenario
	src     *countingSource
	rng     *rand.Rand
	nrng    uint16
	steps   []obj
	prev    []obj
	prevExt string
	result  string
	stopped string
	actions int
	counts  map[string]int
}

func (d *driver) unitObjs() []obj {
	out := make([]obj, len(d.st.Units))
	for i, u := range d.st.Units {
		out[i] = decodeObj(mustJSON(u))
	}
	return out
}

// emit 寫一步：body 加上改變的單位欄位、改變的戰場摘要、下一個 Int63 與原版 RNG。
func (d *driver) emit(kind string, body obj) {
	if body == nil {
		body = obj{}
	}
	body["k"] = kind
	cur := d.unitObjs()
	chg := obj{}
	for i, u := range cur {
		if i >= len(d.prev) {
			chg[fmt.Sprint(i)] = obj{"new": u}
			continue
		}
		set, del := obj{}, []string{}
		for k, v := range u {
			if !reflect.DeepEqual(d.prev[i][k], v) {
				set[k] = v
			}
		}
		for k := range d.prev[i] {
			if _, ok := u[k]; !ok {
				del = append(del, k)
			}
		}
		sort.Strings(del)
		if len(set) != 0 || len(del) != 0 {
			entry := obj{}
			if len(set) != 0 {
				entry["set"] = set
			}
			if len(del) != 0 {
				entry["del"] = del
			}
			chg[fmt.Sprint(i)] = entry
		}
	}
	if len(chg) != 0 {
		body["chg"] = chg
	}
	d.prev = cur
	ext := extras(d.st)
	if text := string(mustJSON(ext)); text != d.prevExt {
		body["ext"] = ext
		d.prevExt = text
	}
	body["i63"] = d.src.peek()
	body["nr"] = d.nrng
	if d.result != "" {
		body["res"] = d.result
	}
	if d.stopped != "" {
		body["stop"] = d.stopped
	}
	d.steps = append(d.steps, body)
	d.counts["step/"+kind]++
}

func (d *driver) stop(reason string) {
	if d.stopped == "" {
		d.stopped = reason
	}
}

// checkResult 是 main.go checkResult 去掉演出等待後的判定。
func (d *driver) checkResult() {
	if d.result != "" || d.sc == nil {
		return
	}
	if len(d.sc.NativeResultRules) != 0 {
		code, err := battle.NativeInitialResultCode(d.st, d.sc)
		if err != nil {
			d.stop("result: " + err.Error())
			return
		}
		for _, rule := range d.sc.NativeResultRules {
			matched, err := rule.Match(d.st)
			if err != nil {
				d.stop("result " + rule.ID + ": " + err.Error())
				return
			}
			if matched {
				code = rule.Code
			}
		}
		d.result = map[int]string{1: "lose", 2: "win"}[code]
		return
	}
	alive := false
	for _, u := range d.st.Units {
		if u.Alive() && (u.Name == protectName || (u.Camp == battle.Own && u.HasNativeIdentity && u.NativeIdentity == 0)) {
			alive = true
			break
		}
	}
	if !alive {
		d.result = "lose"
		return
	}
	if d.sc.NativeResultCode1(d.st) {
		d.result = "lose"
		return
	}
	d.result = d.st.Result("")
}

func manhattan(ax, ay, bx, by int) int {
	dx, dy := ax-bx, ay-by
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

func dirToward(ax, ay, tx, ty int) int {
	dx, dy := tx-ax, ty-ay
	adx, ady := dx, dy
	if adx < 0 {
		adx = -adx
	}
	if ady < 0 {
		ady = -ady
	}
	if adx > ady {
		if dx > 0 {
			return 3
		}
		return 1
	}
	if dy > 0 {
		return 0
	}
	return 2
}

// walk 是 stepBattleWalkSegment／stepBattleWalk 對 battle 狀態的寫入（像素位移屬演出，不寫）。
func (d *driver) walk(u *battle.Unit, path []battle.Cell) {
	for seg := 0; seg+1 < len(path); seg++ {
		a, b := path[seg], path[seg+1]
		pose := dirToward(a.X, a.Y, b.X, b.Y)
		for tick := 1; tick <= 6; tick++ {
			u.X, u.Y = a.X, a.Y
			u.SetMapPose(pose)
			u.SetNativeMapGridMotion(pose, tick)
		}
		if !u.FinishNativeMapGridStep(pose, b.X, b.Y) {
			u.SetMapPlacement(b.X, b.Y, pose)
		}
	}
	last := path[len(path)-1]
	u.SetMapPlacement(last.X, last.Y, nativeRestPose)
}

// finishAction 是 finishSuccessfulUnitAction 對 AI 行動者的狀態寫入。
func (d *driver) finishAction(u *battle.Unit) {
	d.st.MarkNativeDeadRecords()
	for _, x := range d.st.Units {
		if x.HasNativeMapPresentation {
			x.SetMapPose(nativeRestPose)
		}
	}
	u.Acted = true
}

// idleRecovery 是 beginNativeAIIdleRecovery 與其提交（native_ai_idle_recovery.go）。
func (d *driver) idleRecovery(u *battle.Unit, decision battle.NativeAIIdleRecoveryDecision) obj {
	records, err := battle.NativeAIScoringRecords(d.st.Units)
	if err != nil {
		d.stop("idle recovery records: " + err.Error())
		return obj{"err": err.Error()}
	}
	current, err := battle.PlanNativeAIIdleRecovery(records, len(d.st.Units), index(d.st, u))
	if err != nil || current != decision {
		d.stop("idle recovery preflight")
		return obj{"err": errv(err), "changed": current != decision}
	}
	presentation, err := battle.BuildNativeAIIdleRecoveryPresentation(decision)
	if err != nil {
		d.stop("idle recovery presentation: " + err.Error())
		return obj{"err": err.Error()}
	}
	u.HP = int(presentation.AfterHP)
	d.st.NativeMapRangeMode, d.st.HasNativeMapRangeModeState = int(presentation.AfterGlobalWrite.Value), true
	d.counts["idle_recovery"]++
	return obj{"before": presentation.BeforeHP, "after": presentation.AfterHP}
}

func (d *driver) attack(u, tgt *battle.Unit) obj {
	u.SetMapPose(dirToward(u.X, u.Y, tgt.X, tgt.Y))
	result, err := d.st.AttackNativePhysicalWithExperience(u, tgt, d.nrng, d.rng)
	if err != nil {
		d.stop("attack: " + err.Error())
		return obj{"err": err.Error()}
	}
	d.nrng = result.RNGState
	d.counts["attack"]++
	return obj{"atk": attackProj(&result.Attack), "strikes": len(result.AttackStrikes), "ctr": attackProj(result.Counter),
		"ctr_strikes": len(result.CounterStrikes), "ctr_exp": result.CounterExpGained, "ctr_lv": levelUps(result.CounterLevelUps)}
}

// spell 是 executeAISpell（ai_spell_execute.go）。
func (d *driver) spell(plan *battle.AIPlan) obj {
	actor, target := plan.U, plan.Target
	if actor == nil || target == nil {
		d.stop("spell: context")
		return obj{"err": "context"}
	}
	if !actor.OnField || !actor.Alive() || actor.Acted || actor.Sealed || !target.OnField || !target.Alive() {
		d.stop("spell: actor or target invalid")
		return obj{"err": "invalid"}
	}
	var sp *battle.Spell
	for i := range d.st.SpellBook {
		if d.st.SpellBook[i].ID == plan.SpellID {
			sp = &d.st.SpellBook[i]
			break
		}
	}
	if sp == nil || actor.MP < sp.MP || sp.MP < 0 {
		d.stop("spell: row or MP")
		return obj{"err": "row"}
	}
	valid := false
	for _, c := range d.st.AISpellCandidates(actor, *sp) {
		if c == target {
			valid = true
		}
	}
	if !valid || manhattan(actor.X, actor.Y, target.X, target.Y) > sp.Dist {
		d.stop("spell: target")
		return obj{"err": "target"}
	}
	results := d.st.CastArea(actor, target.X, target.Y, *sp, d.rng)
	if len(results) == 0 {
		actor.MP += sp.MP
		d.stop("spell: no result")
		return obj{"err": "no result"}
	}
	out := make([]any, 0, len(results))
	for _, r := range results {
		out = append(out, []any{index(d.st, r.Target), r.Amount, r.Missed, r.ExpGained, levelUps(r.LevelUps)})
	}
	actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
	d.finishAction(actor)
	d.checkResult()
	d.counts["spell"]++
	return obj{"cast": out}
}

// nativeAction 是 executeNativeAIActionWithContinuation：指令 13～16 照做，其餘只收尾。
func (d *driver) nativeAction(plan *battle.AIPlan) obj {
	actor, target := plan.U, plan.Target
	if target == nil {
		d.stop("native action: no target")
		return obj{"err": "no target"}
	}
	out := obj{}
	if plan.NativeActionKind == battle.NativeAIActionCommand && plan.NativeCommandID >= 13 && plan.NativeCommandID <= 16 {
		origin := plan.NativeActionDestination
		if _, err := d.st.NativeAICommandHealTargets(actor, origin, plan.NativeCommandID); err != nil {
			d.stop("heal targets: " + err.Error())
			return obj{"err": err.Error()}
		}
		results, next, err := d.st.ExecuteNativeAICommandHealNative(actor, origin, plan.NativeCommandID, d.nrng)
		if err != nil {
			d.stop("heal: " + err.Error())
			return obj{"err": err.Error()}
		}
		d.nrng = next
		heals := make([][]int, 0, len(results))
		for _, r := range results {
			heals = append(heals, []int{index(d.st, r.Target), r.Restore.Actual})
		}
		out["heal"] = heals
		d.counts["heal"]++
	} else {
		out["skipped"] = true
		d.counts[fmt.Sprintf("native_unreplayed/%d/%d", plan.NativeActionKind, plan.NativeCommandID)]++
	}
	d.st.NativeExperienceAccumulator = 0
	actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
	d.finishAction(actor)
	d.checkResult()
	return out
}

// act 是 aiStep 拿到計畫之後的分支。
func (d *driver) act(plan *battle.AIPlan) obj {
	if plan.NativeModeFallbackActive && plan.NativeModeFallback == 8 && plan.NativeError == nil {
		d.counts["skip8"]++
		return obj{"o": "skip8"}
	}
	if plan.NativeError != nil {
		d.stop("native error: " + plan.NativeError.Error())
		return obj{"o": "native_error"}
	}
	d.actions++
	if len(plan.NativeMode11Stages) > 0 {
		d.counts["mode11"]++
		stages := []any{}
		for _, stage := range plan.NativeMode11Stages {
			if d.stopped != "" {
				break
			}
			if d.result != "" {
				break
			}
			if stage.Recovery != nil {
				// runNativeAIMode11Stage：回復完直接進下一段，不收尾。
				stages = append(stages, d.idleRecovery(plan.U, *stage.Recovery))
				continue
			}
			action := stage.Action
			if action == nil {
				stages = append(stages, "none")
				continue
			}
			if action.NativeModeWriteRangeZero {
				d.st.NativeMapRangeMode, d.st.HasNativeMapRangeModeState = 0, true
			}
			if len(action.Path) >= 2 {
				d.walk(action.U, action.Path)
			}
			switch action.NativeActionKind {
			case battle.NativeAIActionCommand, battle.NativeAIActionItem:
				stages = append(stages, d.nativeAction(action))
			case battle.NativeAIActionPhysical:
				if action.Target == nil || !action.Target.Alive() {
					d.stop("mode 11 physical: target")
					stages = append(stages, "bad target")
					break
				}
				r := d.attack(action.U, action.Target)
				d.checkResult()
				if d.stopped == "" {
					d.finishAction(action.U)
				}
				stages = append(stages, r)
			default:
				d.finishAction(action.U)
				stages = append(stages, "finish")
			}
		}
		if d.stopped == "" {
			d.finishAction(plan.U)
		}
		return obj{"o": "mode11", "stages": stages}
	}
	if plan.NativeIdleRecovery != nil {
		r := d.idleRecovery(plan.U, *plan.NativeIdleRecovery)
		if d.stopped == "" {
			d.finishAction(plan.U)
		}
		return obj{"o": "idle", "r": r}
	}
	u := plan.U
	out := obj{"o": "act"}
	if len(plan.Path) >= 2 {
		d.walk(u, plan.Path)
		d.counts["walk"]++
	} else if plan.NativeModeFocusActor && plan.Target == nil && plan.NativeModeWriteRangeZero && plan.NativeModeFallback != 3 {
		d.st.NativeMapRangeMode, d.st.HasNativeMapRangeModeState = 0, true
	}
	if plan.NativeModeWriteRangeZero {
		d.st.NativeMapRangeMode, d.st.HasNativeMapRangeModeState = 0, true
	}
	if plan.SpellID >= 0 {
		out["spell"] = d.spell(plan)
		return out
	}
	if plan.NativeActionKind == battle.NativeAIActionCommand || plan.NativeActionKind == battle.NativeAIActionItem {
		out["native"] = d.nativeAction(plan)
		return out
	}
	finish := func() {
		if plan.NativeModeEventActive {
			cues := [][]int{}
			err := d.st.ApplyNativeAIMode5EventWithAudioCue(u, plan.NativeModeEventID, plan.NativeModeEventDestination,
				func(c battle.NativeAIMode5AudioCue) {
					cues = append(cues, []int{int(c.HandleLinearAddress), c.ResourceID, c.Index, c.LoopCount})
				})
			out["mode5"] = obj{"err": errv(err), "cues": cues}
			if err != nil {
				d.stop("mode5: " + err.Error())
				return
			}
			d.counts["mode5"]++
		}
		if plan.NativeFallbackIdleRecovery != nil {
			out["fb_idle"] = d.idleRecovery(u, *plan.NativeFallbackIdleRecovery)
			if d.stopped != "" {
				return
			}
		}
		if plan.NativeModeWriteByte5 {
			u.NativeRecordByte5, u.HasNativeRecordByte5 = 1, true
		}
		d.finishAction(u)
	}
	if plan.Target != nil && plan.Target.Alive() {
		out["attack"] = d.attack(u, plan.Target)
		if d.stopped != "" {
			return out
		}
		d.checkResult()
	}
	finish()
	return out
}

// events 是 runEditableTurnEvents(selector)（selector 為 -1 時是 finishTurn 沒有 selector 的事件）
// 加上 advanceBattleEvent 對每個動作的分派。
func (d *driver) events(selector int) {
	if d.sc == nil || d.stopped != "" {
		return
	}
	actions := d.sc.TriggerActionsWhere(d.st, "on_turn_end", "", func(e *battle.Event) bool {
		return e.NativeTurnPhaseSelector() == selector
	})
	if selector >= 0 {
		for _, event := range d.sc.NativeTurnActionEventsAt(d.st, byte(selector)) {
			actions = append(actions, event.Actions...)
		}
	}
	if len(actions) == 0 {
		return
	}
	outcomes := make([]any, 0, len(actions))
	for _, action := range actions {
		outcome := d.runAction(action)
		outcomes = append(outcomes, []any{action.Type, outcome})
		if d.stopped != "" {
			break
		}
	}
	rawify(d.st.Units)
	d.emit("events", obj{"sel": selector, "acts": outcomes})
}

func (d *driver) runAction(action battle.Action) string {
	if action.NativeWhen != nil {
		ok, err := action.NativeWhen.Match(d.st)
		if err != nil {
			d.stop("when: " + err.Error())
			return "when_err"
		}
		if !ok {
			return "when_false"
		}
	}
	if action.Type == "dialogue" && action.NativeDialogueRef != nil {
		return "dialogue"
	}
	switch action.Type {
	case "native_death_op":
		op := action.NativeDeathOp
		if op == nil || action.NativeSource == "" {
			d.stop("death op source")
			return "death_op_err"
		}
		if op.Op == "staging" || op.Op == "clear_hp_from" || op.Op == "reward" {
			return "death_op_presented"
		}
		if err := d.st.ApplyNativeDeathOp(*op); err != nil {
			d.stop("death op: " + err.Error())
			return "death_op_err"
		}
		return "death_op"
	case "reset_pose":
		for _, u := range d.st.Units {
			u.SetMapPose(0)
		}
		return "reset_pose"
	case "redraw", "pan", "delay", "native_acting", "native_staging_present":
		return "presented"
	case "native_range_zero", "native_range_one":
		if action.NativeSource == "" {
			d.stop("range source")
			return "range_err"
		}
		value := 0
		if action.Type == "native_range_one" {
			value = 1
		}
		d.st.NativeMapRangeMode, d.st.HasNativeMapRangeModeState = value, true
		return "range"
	}
	for _, call := range action.NativeSpawns {
		if call.Via == "spawn_group_with_intro" {
			return "intro"
		}
	}
	dl, isDialogue, err := d.sc.ExecuteActionChecked(d.st, action)
	if err != nil {
		d.stop("action: " + err.Error())
		return "err"
	}
	d.sc.TakePartyJoins()
	if isDialogue {
		return fmt.Sprint("dialogue_line:", dl.Speaker)
	}
	return "exec"
}

func (d *driver) transient(selector byte) {
	if d.stopped != "" || len(d.st.Units) == 0 {
		return
	}
	if !d.st.HasNativeRuntimeUnitProjection {
		for _, u := range d.st.Units {
			if !u.HasNativeRecordByte5 || !u.HasNativeRecordByte6 {
				return
			}
		}
	}
	result, err := d.st.AdvanceNativeTransientPhaseTyped(selector)
	body := obj{"sel": int(selector), "err": errv(err)}
	if err != nil {
		d.stop("transient: " + err.Error())
	} else {
		dmg := [][]int{}
		for _, x := range result.Damage {
			dmg = append(dmg, []int{index(d.st, x.Unit), x.Slot, x.Amount, x.Before, x.After})
		}
		exp := [][]int{}
		for _, x := range result.Expired {
			exp = append(exp, []int{index(d.st, x.Unit), x.Offset})
		}
		body["dmg"], body["expired"], body["inactive"] = dmg, exp, result.InactiveSlots
	}
	d.emit("transient", body)
}

// phase 跑一遍 AI：ally 為真時是 NextAllyAIPlan，否則 NextAIPlan。
func (d *driver) phase(ally bool) {
	for n := 0; ; n++ {
		if d.stopped != "" || d.result != "" {
			return
		}
		if n >= maxPhasePlans {
			d.stop("phase plan limit")
			d.emit("limit", nil)
			return
		}
		var plan *battle.AIPlan
		handled := true
		if ally {
			plan, handled = d.st.NextAllyAIPlan()
			if !handled || plan == nil {
				d.emit("ally_done", obj{"h": handled})
				return
			}
		} else {
			plan = d.st.NextAIPlan()
			if plan == nil {
				d.emit("enemy_done", nil)
				return
			}
		}
		body := obj{"ally": ally, "p": planProj(d.st, plan)}
		if tie := d.reachTie(plan); tie != nil {
			body["tie"] = tie
			d.counts["reach_tie"]++
		}
		body["a"] = d.act(plan)
		d.emit("ai", body)
	}
}

// reachTie 只供診斷：normalized 規劃器走到 combat.go 202／235／921 的最近格迴圈時，
// 計算與所選格同距離的其他候選數（Go map 迭代順序決定挑哪一格）。在 act 之前呼叫。
func (d *driver) reachTie(plan *battle.AIPlan) any {
	if plan == nil || plan.U == nil || plan.NativeActionKind != battle.NativeAIActionNone ||
		plan.NativeModeFallbackActive || plan.NativeMode2Physical || plan.SpellID >= 0 || len(plan.Path) < 2 {
		return nil
	}
	u := plan.U
	dst := plan.Path[len(plan.Path)-1]
	reach := d.st.Reachable(u)
	ties := 0
	for _, t := range d.st.Units {
		hostile := (u.Camp == battle.Enemy && (t.Camp == battle.Own || t.Camp == battle.Ally)) ||
			(u.Camp == battle.Ally && t.Camp == battle.Enemy)
		if !t.OnField || !t.Alive() || !hostile {
			continue
		}
		best := manhattan(dst.X, dst.Y, t.X, t.Y)
		minimal, others := true, 0
		for c := range reach {
			if d.st.UnitAt(c.X, c.Y) != nil || (c.X == dst.X && c.Y == dst.Y) {
				continue
			}
			switch dd := manhattan(c.X, c.Y, t.X, t.Y); {
			case dd < best:
				minimal = false
			case dd == best:
				others++
			}
		}
		if minimal && others > 0 {
			ties += others
		}
	}
	if ties == 0 {
		return nil
	}
	return ties
}

func (d *driver) run(turns int) {
	d.checkResult()
	d.emit("init_result", nil)
	for turn := 1; turn <= turns; turn++ {
		if d.result != "" || d.stopped != "" {
			return
		}
		recs := [][]int{}
		for _, r := range d.st.ApplyNativeEndTurnRecovery() {
			recs = append(recs, []int{index(d.st, r.Unit), r.Before, r.After, r.MaxHP})
		}
		d.emit("end", obj{"turn": turn, "rec": recs})
		d.events(1)
		d.transient(1)
		if d.stopped != "" {
			return
		}
		d.st.ResetNativeAIScan()
		d.phase(true)
		if d.stopped != "" || d.result != "" {
			return
		}
		for _, u := range d.st.Units {
			if u.HasNativeRecordByte5 {
				u.NativeRecordByte5 &= 0x7f
			}
			u.Acted = false
		}
		d.emit("clear_bit7", nil)
		d.events(0)
		d.transient(0)
		d.phase(false)
		if d.stopped != "" || d.result != "" {
			return
		}
		d.events(-1)
		d.st.Turn++
		if d.st.NativeRoundCounter > 0 {
			d.st.NativeRoundCounter++
		}
		for _, u := range d.st.Units {
			u.Acted = false
			if d.st.HasNativeMapViewState && u.HasNativeRecordByte5 {
				u.NativeRecordByte5 &= 0x7f
			}
			u.TickStatus()
		}
		d.emit("complete_turn", nil)
		d.transient(2)
		d.events(2)
		d.checkResult()
		d.emit("player_phase", nil)
	}
}

// ---- main ----

func recordFiles(paths []string) []obj {
	files := []obj{}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		files = append(files, obj{"path": path, "size": len(raw), "sha256": digest(raw)})
	}
	return files
}

type field struct {
	name, mapPath, scenarioPath string
}

func fields() ([]field, []string) {
	scenarios, err := filepath.Glob("assets/scenarios/ch*.json")
	if err != nil || len(scenarios) == 0 {
		panic("找不到 assets/scenarios/ch*.json")
	}
	sort.Strings(scenarios)
	maps, err := filepath.Glob("assets/maps/map*/map*_units.json")
	if err != nil || len(maps) == 0 {
		panic("找不到 assets/maps/map*/map*_units.json")
	}
	sort.Strings(maps)
	used := map[string]bool{}
	out := []field{}
	for _, path := range scenarios {
		probe := must(battle.LoadScenario(path))
		mapPath := fmt.Sprintf("assets/maps/map%d/map%d_units.json", probe.Map, probe.Map)
		used[mapPath] = true
		out = append(out, field{strings.TrimSuffix(filepath.Base(path), ".json"), mapPath, path})
	}
	for _, path := range maps {
		if !used[path] {
			out = append(out, field{strings.TrimSuffix(filepath.Base(path), "_units.json"), path, ""})
		}
	}
	inputs := []string{spellsPath, costRowsPath, itemRowsPath, resistancesPath, learnPath, learnSelPath,
		growthPath, itemJSONPath, controlsPath}
	inputs = append(inputs, scenarios...)
	for _, path := range maps {
		inputs = append(inputs, path)
		if _, err := os.Stat(filepath.Join(filepath.Dir(path), "map.json")); err == nil {
			inputs = append(inputs, filepath.Join(filepath.Dir(path), "map.json"))
		}
	}
	return out, inputs
}

func battleRun(a *assets, f field, seedIndex int, counts map[string]int) obj {
	st, sc, info := setup(a, f.mapPath, f.scenarioPath)
	seed := seeds[seedIndex]
	src := newCountingSource(seed.Rand)
	d := &driver{st: st, sc: sc, src: src, rng: rand.New(src), nrng: seed.Native, counts: counts}
	init := obj{"units": d.unitObjs(), "ext": extras(st), "info": info, "i63": src.peek()}
	d.prev = init["units"].([]obj)
	d.prevExt = string(mustJSON(init["ext"]))
	d.run(maxTurns)
	end := obj{"result": d.result, "stopped": d.stopped, "turn": st.Turn, "actions": d.actions}
	if snap, err := st.Snapshot(); err != nil {
		end["snapshot_err"] = err.Error()
	} else {
		end["snapshot"] = mustJSON(snap)
	}
	counts["battle"]++
	counts["result/"+d.result]++
	if d.stopped != "" {
		counts["stopped"]++
		counts["stop/"+d.stopped]++
	}
	return obj{"name": f.name, "map": f.mapPath, "scenario": f.scenarioPath, "seed": seed.Rand, "native_rng": seed.Native,
		"init": init, "steps": d.steps, "end": end}
}

func marchCases(a *assets, f field, rng *randv2.Rand, counts map[string]int) []obj {
	st, _, _ := setup(a, f.mapPath, f.scenarioPath)
	before := string(mustJSON(st.Units))
	out := []obj{}
	dests := []battle.Cell{{X: -1, Y: 0}, {X: st.W, Y: 0}}
	for k := 0; k < 10; k++ {
		dests = append(dests, battle.Cell{X: rng.IntN(st.W), Y: rng.IntN(st.H)})
	}
	for _, dest := range dests {
		plan, err := st.PlanNativeSystemGroupMarch(dest)
		steps := []any{}
		for _, s := range plan.Steps {
			events := [][]any{}
			for _, e := range s.Events {
				events = append(events, []any{e.PathIndex, e.EventID, e.TextIndex, e.Presentation})
			}
			steps = append(steps, obj{"u": s.UnitIndex, "path": cells(s.Path), "events": events})
		}
		after := string(mustJSON(st.Units))
		out = append(out, obj{"field": f.name, "map": f.mapPath, "scenario": f.scenarioPath, "dest": cell(dest),
			"err": errv(err), "plan_dest": cell(plan.Destination), "steps": steps, "units_changed": after != before})
		counts["march"]++
		if err == nil {
			counts["march_ok"]++
		}
	}
	return out
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	only := flag.String("only", "", "只跑名稱相符的戰場（診斷用）")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "需要 -out")
		os.Exit(2)
	}
	a := loadAssets()
	list, inputs := fields()
	counts := map[string]int{}
	battles := []obj{}
	march := []obj{}
	mrng := randv2.New(randv2.NewPCG(20261114, 0x494e54))
	for _, f := range list {
		if *only != "" && f.name != *only {
			continue
		}
		march = append(march, marchCases(a, f, mrng, counts)...)
		for k := range seeds {
			battles = append(battles, battleRun(a, f, k, counts))
		}
	}
	aiCmd := [][]int{}
	for k, v := range a.aiCmd {
		aiCmd = append(aiCmd, []int{k, v})
	}
	sort.Slice(aiCmd, func(i, j int) bool { return aiCmd[i][0] < aiCmd[j][0] })
	doc := obj{"schema": "fd2-golden-rules-int/1", "inputs": recordFiles(inputs), "ai_command_spell": aiCmd,
		"max_turns": maxTurns, "max_phase_plans": maxPhasePlans, "battles": battles, "march": march}
	data := mustJSON(doc)
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		panic(err)
	}
	names := []string{}
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("%s=%d\n", name, counts[name])
	}
	fmt.Printf("bytes=%d\n", len(data))
}
