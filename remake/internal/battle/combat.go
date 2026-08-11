// combat.go — 戰鬥結算 + 敵方 AI + 勝負(M1)。
//
// 傷害公式對映青衫/反組譯(doc 02 §4.1、doc 11、doc 27 checklist):
//
//	命中率 = (攻方HIT − 守方EV)%
//	暴擊時 DP = 守方DP/2(取整)
//	AP = AP×(1+攻方地形AP%)、DP = DP×(1+守方地形DP%)(取整,terrain.go)
//	最大傷害 = AP − DP;實際傷害 = 最大傷害×0.9 ～ 最大傷害-1(亂數,magic.go randomizeAmount)
//
// AI normalized approximation：舊 doc11 的 0x15140 地址已由 canonical recheck 撤回；
// 此處只保留 remake-owned 估值與 dmg≤2 相容行為，不宣稱 native AI parity。
// 演出動畫(FIGANI/移動)後補;此處先把邏輯層做對,讓第一關可玩。
package battle

import "math/rand"

// AttackResult 一次近戰攻擊的完整結算結果(doc02 §4.1)。
type AttackResult struct {
	Amount int // 實際傷害;Miss 時為 0
	Missed bool
	Crit   bool

	ExpGained float64        // 攻方本次取得的經驗值(doc02 §4.5「攻擊」列;僅 Own/Ally 攻方會 >0,見 growth.go)
	LevelUps  []LevelUpEvent // 攻方因本次經驗值連續升級的事件(通常 0 或 1 筆,經驗值夠大可多筆)
}

// Attack 舊版相容介面(main.go 目前呼叫此簽名):結算一次近戰攻擊,回傳實際傷害
// (Miss 時回 0)。內部呼叫 AttackWithRNG,用 magic.go 共用的 engineRand。
// 測試/需要確定性結果一律走 AttackWithRNG 並自行注入 *rand.Rand(同 magic.go Cast/CastArea 慣例)。
func (s *State) Attack(a, d *Unit) int {
	return s.AttackWithRNG(a, d, engineRand).Amount
}

// AttackWithRNG 近戰攻擊完整結算(doc02 §4.1、doc27 checklist、doc11 地形修正)。
// 命中率、暴擊、地形% 修正、傷害隨機化皆對照青衫攻略 notes.md 逐條實作,詳見檔頭註解與
// terrain.go/model.go EffectiveHIT/EffectiveEV。恆標記已行動,不論命中與否
// (原版「攻擊」是一個已耗用的行動,miss 不退還行動權)。
func (s *State) AttackWithRNG(a, d *Unit, rng *rand.Rand) AttackResult {
	a.Acted = true

	// 命中率 = (攻方HIT − 守方EV)%;含風行術 HIT/EV 加成(EffectiveHIT/EffectiveEV)。
	hitPct := a.EffectiveHIT() - d.EffectiveEV()
	if !rollsHitPct(hitPct, rng) {
		return AttackResult{Missed: true}
	}

	crit := a.CritPct > 0 && rng.Intn(100) < a.CritPct

	// AP/DP 含輔助法術 Buff(魔刃/魔鎧,doc02 §6.4);暴擊先讓 DP 減半,再套地形% —
	// notes.md 公式順序:「暴擊時 DP=守方DP/2」在「DP=DP×(1+地形%)」之前。
	ap := a.EffectiveAP()
	dp := d.EffectiveDP()
	if crit {
		dp /= 2
	}
	atkAPPct, _ := s.TerrainAPDPPct(a.X, a.Y)
	_, defDPPct := s.TerrainAPDPPct(d.X, d.Y)
	ap = ap * (100 + atkAPPct) / 100
	dp = dp * (100 + defDPPct) / 100

	max := ap - dp
	dmg := randomizeAmount(max, rng)
	// 青衫「dmg≤2」是 AI「不值得打」門檻(doc11),非玩家攻擊下限;玩家命中至少造成 1。
	if dmg < 1 {
		dmg = 1
	}
	d.ApplyHPDamage(dmg)

	// 經驗值(doc02 §4.5「攻擊」列,growth.go AttackExp):致死視同傷害HP=總HP。
	// 只有 Own/Ally 攻方才計算/回報經驗值(見 growth.go 檔頭說明);Enemy 攻方 ExpGained
	// 恆為 0,不是先算出來又被 GainExp 悄悄丟棄。
	var exp float64
	var levelUps []LevelUpEvent
	if a.Camp == Own || a.Camp == Ally {
		dmgForExp := dmg
		if d.HP == 0 {
			dmgForExp = d.MaxHP
		}
		exp = AttackExp(a.Lv, d.Lv, dmgForExp, d.MaxHP, d.ExpPerLevel)
		levelUps = s.GainExp(a, exp, rng)
	}

	return AttackResult{Amount: dmg, Crit: crit, ExpGained: exp, LevelUps: levelUps}
}

// rollsHitPct 物理攻擊命中率擲骰(doc02 §4.1「命中率=(攻方HIT-守方EV)%」)。
// 與 magic.go rollsHit 語意不同:那裡的 hit<=0 是資料矛盾下的「必中」特例(法術表 dump
// 值本身有衝突,見該檔案檔頭說明);這裡 pct<=0 是公式算出來的合法結果(HIT 追不上 EV),
// 依公式原意視為必定 miss,不套用那條特例。
func rollsHitPct(pct int, rng *rand.Rand) bool {
	if pct <= 0 {
		return false
	}
	if pct >= 100 {
		return true
	}
	return rng.Intn(100) < pct
}

// hostile 判斷 a 是否視 b 為攻擊對象(同一套 AI,依陣營;doc11)。
// 敵方(Enemy)打 玩家/友軍;友軍 NPC(Ally)打 敵方;玩家(Own)由人操作。
func hostile(a, b *Unit) bool {
	if a.Camp == Enemy {
		return b.Camp == Own || b.Camp == Ally
	}
	if a.Camp == Ally {
		return b.Camp == Enemy
	}
	return false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func manhattan(ax, ay, bx, by int) int { return abs(ax-bx) + abs(ay-by) }

// estDamage 是 remake normalized AI 估值；舊 doc11 0x15140 反組譯地址已撤回，
// 不把這個 helper 當作 native score proof：
//
//	myAP'  = myAP  × 地形AP%[u當下座標] / 100
//	tarDP' = tarDP × 地形DP%[t當下座標] / 100
//	估計傷害 = myAP' − tarDP'
//
// 只是選目標用的估值,不擲骰(不含命中率/暴擊/傷害隨機化——那些留給 AttackWithRNG 實際結算)。
func (s *State) estDamage(u, t *Unit) int {
	apPct, _ := s.TerrainAPDPPct(u.X, u.Y)
	_, dpPct := s.TerrainAPDPPct(t.X, t.Y)
	ap := u.AP * (100 + apPct) / 100
	dp := t.DP * (100 + dpPct) / 100
	return ap - dp
}

// aiTargets separates the original AI's attack candidate from its movement
// fallback.  This normalized compatibility rule ignores targets whose estimated
// damage is at most two; it is not proof that the withdrawn 0x15140 address has
// that native behavior. When every hostile target is below the threshold, the
// unit may still advance toward the nearest hostile but must not attack it.
func (s *State) aiTargets(u *Unit) (attack, move *Unit) {
	bestScore := -1 << 30
	bestDistance := 1 << 30
	for _, t := range s.Units {
		if !t.OnField || !t.Alive() || !hostile(u, t) {
			continue
		}
		distance := manhattan(u.X, u.Y, t.X, t.Y)
		if move == nil || distance < bestDistance {
			move, bestDistance = t, distance
		}
		dmg := s.estDamage(u, t)
		if dmg <= 2 {
			continue
		}
		score := dmg
		if dmg >= t.HP { // 可擊殺 → 最高優先(doc11 prio 0x12)
			score = dmg*2 + 1000
		}
		score = score*100 - distance
		if attack == nil || score > bestScore {
			attack, bestScore = t, score
		}
	}
	return attack, move
}

func (s *State) aiApproachPath(u, target *Unit) []Cell {
	reach := s.Reachable(u)
	dstX, dstY := u.X, u.Y
	bestD := manhattan(u.X, u.Y, target.X, target.Y)
	for c := range reach {
		if s.UnitAt(c.X, c.Y) != nil {
			continue
		}
		d := manhattan(c.X, c.Y, target.X, target.Y)
		if d < bestD {
			bestD = d
			dstX, dstY = c.X, c.Y
		}
	}
	return s.Path(u, dstX, dstY)
}

// aiActUnit 是重製端既有的正規化近似，不代表原版 0x14237 的完整候選、
// 地形、優先級與同分契約。
func (s *State) aiActUnit(u *Unit) {
	// 找重製端的近似攻擊目標；低傷害時仍保留最近單位作為移動目標。
	best, moveTarget := s.aiTargets(u)
	if moveTarget == nil {
		return
	}
	if best == nil {
		best = moveTarget
	}
	// 2. 已在攻擊範圍內(InAttackRange 依武器射程判定,doc32) → 直接打
	if s.InAttackRange(u, best.X, best.Y) && s.estDamage(u, best) > 2 {
		s.Attack(u, best)
		return
	}
	// 3. 移到「能攻擊到 best 的最近可達格」,再打
	reach := s.Reachable(u)
	var dstX, dstY = u.X, u.Y
	bestD := manhattan(u.X, u.Y, best.X, best.Y)
	for c := range reach {
		if s.UnitAt(c.X, c.Y) != nil {
			continue
		}
		d := manhattan(c.X, c.Y, best.X, best.Y)
		if d < bestD {
			bestD = d
			dstX, dstY = c.X, c.Y
		}
	}
	u.SetMapPlacement(dstX, dstY, u.Dir)
	if best != moveTarget && s.InAttackRange(u, best.X, best.Y) {
		s.Attack(u, best)
	}
	u.Acted = true
}

// AITurn 讓所有非玩家、已登場、未行動的單位(敵 + 友軍 NPC)各行動一次。
func (s *State) AITurn() {
	for _, u := range s.Units {
		if !u.OnField || !u.Alive() || u.Camp == Own || u.Acted || u.Paralyzed {
			continue
		}
		s.aiActUnit(u)
		u.Acted = true
	}
}

// Result 勝負判定。回傳 "win"/"lose"/""。
// 預設規則(可被 scenario 覆寫):敵全滅(且無待命援軍)→ win;指定要保護的單位死 → lose。
func (s *State) Result(protect string) string {
	if protect != "" {
		dead := true
		for _, u := range s.Units {
			if u.Name == protect && u.Alive() {
				dead = false
				break
			}
		}
		if dead {
			return "lose"
		}
	}
	if s.AliveCount(Enemy) == 0 && s.PendingCount(Enemy) == 0 {
		return "win"
	}
	return ""
}

// AIPlan 一個 AI 單位的行動計畫(決策與執行分離,供引擎逐單位播放移動動畫後才結算)。
type AIPlan struct {
	U       *Unit
	Path    []Cell // 含起點;len>=2 = 要移動(引擎播行走動畫)
	Target  *Unit  // 到位後攻擊目標(nil = 僅移動/待機)
	SpellID int    // 原版 spell command 的資料欄位；-1 表示本計畫不施法
	// NativeActionKind identifies a verified raw action route selected by
	// 0x14ef0.  None keeps the legacy planner contract; the other values are
	// only executable when the corresponding raw candidate, target and
	// movement provenance are complete.
	NativeActionKind        NativeAIActionKind
	NativeCommandID         int
	NativeItemSlot          int
	NativeItemID            int
	NativeActionDestination Cell
	NativeAI14EF0Route      NativeAI14EF0Tail
	NativeActionScore       int
	// NativeModeFallbackActive distinguishes a raw dispatcher fallback from a
	// normalized plan. NativeModeFallback is meaningful only when this flag is
	// true; its value is the original low nibble, not a gameplay name.
	NativeModeFallbackActive bool
	NativeModeFallback       byte
	NativeModeWriteByte5     bool
	NativeModeWriteRangeZero bool
	// NativeModeEventActive marks mode 5's exact 0x15df3 map-event lookup.
	// The event ID and destination are raw dispatcher operands; the executor
	// must revalidate the map/control bytes before mutating the unit.
	NativeModeEventActive      bool
	NativeModeEventID          byte
	NativeModeEventDestination Cell
	// NativeMode2Physical 標記原始 mode 2 物理候選窄切片；只有計畫確實使用
	// 脫離的 runtime record、0x4e555 移動表與已驗證的 0x14237 評分契約時才為真。
	// 它不代表前置 0x14ef0 路由或所有 native mode 已閉合。
	NativeMode2Physical bool
	// NativeError 是失敗即關閉的來源／執行期錯誤；非 nil 時，命令層 runner
	// 必須在消耗單位行動前停止。
	NativeError error
	// NativeScoredCommands preserves raw command indices that passed the
	// verified command-mask/+0x27/MP gates at 0x1598a. It is evidence only: planner code
	// must still resolve target, score, presentation, and execution separately.
	NativeScoredCommands []int
	// NativeMode11Stages preserves the direct mode-11 dispatcher as one
	// caller-owned sequence.  Each stage retains its raw route ordinal; the
	// command/physical plan or the optional 0x13FD4 decision is consumed only
	// after its predecessor has completed presentation.  A nil slice means the
	// plan is not mode 11.
	NativeMode11Stages []NativeAIMode11StagePlan
}

// NativeAIMode11StagePlan is the executable boundary for one direct mode-11
// stage.  Action is used by 0x15311/0x1548E and the raw 0x14121 movement path;
// Recovery is present only when 0x14121 returned zero and the 0x13FD4 raw HP
// gates accepted.  Keeping both optional is intentional: the original common
// tail can complete without a visible action when the recovery gates reject.
type NativeAIMode11StagePlan struct {
	Stage    NativeAIMode11Stage
	Action   *AIPlan
	Recovery *NativeAIIdleRecoveryDecision
}

// NativeAIActionKind is deliberately a route label, not a gameplay name.
// The command table's IDs and the item row's bytes remain the authoritative
// semantic layer until their presentation/effect names are independently
// proven.
type NativeAIActionKind uint8

const (
	NativeAIActionNone NativeAIActionKind = iota
	NativeAIActionPhysical
	NativeAIActionCommand
	NativeAIActionItem
)

func (s *State) nativeAIPlanScoredCommands(u *Unit) []int {
	if s == nil || len(s.NativeCommandBook) != 36 {
		return nil
	}
	return NativeAvailableAIScoredCommandIDs(u, s.NativeCommandBook)
}

// AIAvailableSpells mirrors the data portion of the native AI command scan:
// inventory commands are translated through State.AICommandSpell and then
// resolved against the injected EXE SpellBook. It deliberately does not pick
// a target or score a spell; those rules belong to the later 0x149f8/0x15b77
// decision layer.
func (s *State) AIAvailableSpells(u *Unit) []Spell {
	if s == nil || u == nil || len(s.AICommandSpell) == 0 || len(s.SpellBook) == 0 {
		return nil
	}
	byID := make(map[int]Spell, len(s.SpellBook))
	for _, spell := range s.SpellBook {
		byID[spell.ID] = spell
	}
	seen := make(map[int]bool)
	out := make([]Spell, 0)
	for _, itemID := range u.Inventory {
		spellID, ok := s.AICommandSpell[itemID]
		if !ok || seen[spellID] {
			continue
		}
		spell, ok := byID[spellID]
		if !ok {
			continue
		}
		seen[spellID] = true
		out = append(out, spell)
	}
	return out
}

// AISpellCandidates mirrors the family split visible in 0x15B77. It returns
// candidates in canonical runtime order only; the native score/priority layer
// is intentionally separate and not inferred here.
func (s *State) AISpellCandidates(caster *Unit, spell Spell) []*Unit {
	if s == nil || caster == nil {
		return nil
	}
	family := ""
	switch spell.ID {
	case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12:
		family = "attack"
	case 13, 14, 15, 16:
		family = "heal"
	case 17, 18, 19:
		family = "buff"
	case 20, 21:
		family = "cure"
	case 22, 26, 27:
		family = "status"
	case 25:
		family = "action"
	case 34:
		family = "buff"
	case 35:
		family = "status"
	default:
		return nil
	}
	out := make([]*Unit, 0)
	for _, target := range s.Units {
		if target == nil || !target.OnField || !target.Alive() {
			continue
		}
		// Own 與 Ally 是同一陣線；不可只用 Camp 相等，否則 Ally NPC
		// 會把 Own 誤當成攻擊法術目標。
		sameSide := !isEnemyOf(caster, target)
		switch family {
		case "attack":
			if !sameSide {
				out = append(out, target)
			}
		case "heal":
			if sameSide && target.HP < target.MaxHP {
				out = append(out, target)
			}
		case "buff":
			if sameSide {
				out = append(out, target)
			}
		case "action":
			if sameSide && target.Acted {
				out = append(out, target)
			}
		case "cure":
			if sameSide && ((spell.ID == 20 && target.Poisoned) || (spell.ID == 21 && target.Paralyzed)) {
				out = append(out, target)
			}
		case "status":
			if !sameSide {
				out = append(out, target)
			}
		}
	}
	return out
}

// aiSpellOptions 合併重製端兩種可編輯法術來源。原始命令紀錄仍走獨立 route；
// 只有原始 provenance 不可用時才會呼叫本函式。背包命令映射保持作者設定的順序，
// 之後才加入尚未出現的正規化（normalized）單位法術 ID。
func (s *State) aiSpellOptions(u *Unit) []Spell {
	if s == nil || u == nil {
		return nil
	}
	byID := make(map[int]Spell, len(s.SpellBook))
	for _, sp := range s.SpellBook {
		if _, exists := byID[sp.ID]; !exists {
			byID[sp.ID] = sp
		}
	}
	seen := make(map[int]bool)
	options := make([]Spell, 0)
	for _, sp := range s.AIAvailableSpells(u) {
		if seen[sp.ID] {
			continue
		}
		seen[sp.ID] = true
		options = append(options, sp)
	}
	for _, id := range u.Spells {
		if seen[id] {
			continue
		}
		sp, ok := byID[id]
		if !ok {
			continue
		}
		seen[id] = true
		options = append(options, sp)
	}
	return options
}

// aiSpellChoice 是正規化（normalized）的後備決策。它刻意只保存可編輯的
// spell／target／path 資料；不可將其誤讀成原版 0x1598A 的評分或選目標 ABI。
type aiSpellChoice struct {
	spell    Spell
	target   *Unit
	path     []Cell
	priority int
	score    int
	order    int
}

// aiSpellPriority 讓輔助行為可用，但不宣稱原版權重：先解除異常，再治療／行動／
// 輔助，最後才是攻擊／狀態法術。
func aiSpellPriority(sp Spell) int {
	switch sp.ID {
	case 20, 21:
		return 500
	case 13, 14, 15, 16:
		return 450
	case 25:
		return 400
	case 17, 18, 19, 34:
		return 350
	case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 22, 26, 27, 35:
		return 200
	default:
		return 0
	}
}

// aiSpellTargetScore 刻意保持可解釋且決定性（deterministic）。輔助法術優先處理
// 最需要的目標；攻擊／狀態法術偏好可擊殺或高傷害的敵方目標。這只是重製端後備，
// 不是對原版評分表的宣稱。
func (s *State) aiSpellTargetScore(caster, target *Unit, sp Spell) (int, bool) {
	if caster == nil || target == nil || !target.OnField || !target.Alive() {
		return 0, false
	}
	distance := manhattan(caster.X, caster.Y, target.X, target.Y)
	score := 0
	switch sp.ID {
	case 20:
		if !target.Poisoned {
			return 0, false
		}
		score = 10000
	case 21:
		if !target.Paralyzed {
			return 0, false
		}
		score = 10000
	case 13, 14, 15, 16:
		missing := target.MaxHP - target.HP
		if missing <= 0 {
			return 0, false
		}
		score = missing * 100
	case 25:
		if !target.Acted {
			return 0, false
		}
		score = 5000
	case 17, 18, 19, 34:
		if target.BuffTurns > 0 {
			return 0, false
		}
		score = 3000
	case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 22, 26, 27, 35:
		if !isEnemyOf(caster, target) {
			return 0, false
		}
		// 用可編輯法術的最大值當穩定估計；可擊殺者優先，其後為較低剩餘 HP
		// 與距離。這不是原版 score 的推論。
		score = sp.Dmg * 10
		if sp.Dmg >= target.HP {
			score += 10000
		}
		score += (target.MaxHP - target.HP) * 2
	default:
		return 0, false
	}
	return score - distance, true
}

// aiSpellPath 選擇可自其施放的最短可達空格。零移動施放保留起點；它與
// InCastRange 不同，允許以自己為目標的輔助法術。
func (s *State) aiSpellPath(u, target *Unit, sp Spell) []Cell {
	if s == nil || u == nil || target == nil {
		return nil
	}
	if manhattan(u.X, u.Y, target.X, target.Y) <= sp.Dist {
		return []Cell{{X: u.X, Y: u.Y}}
	}
	reach := s.Reachable(u)
	best := []Cell(nil)
	bestDistance, bestPathLen, bestX, bestY := 1<<30, 1<<30, 1<<30, 1<<30
	for cell := range reach {
		if cell.X == u.X && cell.Y == u.Y {
			continue
		}
		if s.UnitAt(cell.X, cell.Y) != nil {
			continue
		}
		if manhattan(cell.X, cell.Y, target.X, target.Y) > sp.Dist {
			continue
		}
		path := s.Path(u, cell.X, cell.Y)
		if len(path) == 0 {
			continue
		}
		distance := manhattan(cell.X, cell.Y, target.X, target.Y)
		if distance < bestDistance ||
			(distance == bestDistance && len(path) < bestPathLen) ||
			(distance == bestDistance && len(path) == bestPathLen &&
				(cell.Y < bestY || (cell.Y == bestY && cell.X < bestX))) {
			best, bestDistance, bestPathLen, bestX, bestY = path, distance, len(path), cell.X, cell.Y
		}
	}
	return best
}

// nextAISpellPlan 是敵方／友軍 NPC 可編輯法術的正規化（normalized）後備。它只會在
// 所有原始 route 正常未處理或缺少 provenance 後呼叫；原始 route 的錯誤仍會讓回合
// 失敗即關閉（fail-closed）。
func (s *State) nextAISpellPlan(u *Unit) *AIPlan {
	if s == nil || u == nil || u.Sealed || u.MP < 0 {
		return nil
	}
	options := s.aiSpellOptions(u)
	var best *aiSpellChoice
	for order, sp := range options {
		if sp.MP < 0 || u.MP < sp.MP || aiSpellPriority(sp) == 0 {
			continue
		}
		for _, target := range s.AISpellCandidates(u, sp) {
			score, ok := s.aiSpellTargetScore(u, target, sp)
			if !ok {
				continue
			}
			path := s.aiSpellPath(u, target, sp)
			if len(path) == 0 {
				continue
			}
			choice := &aiSpellChoice{spell: sp, target: target, path: path,
				priority: aiSpellPriority(sp), score: score, order: order}
			if best == nil || choice.priority > best.priority ||
				(choice.priority == best.priority && choice.score > best.score) ||
				(choice.priority == best.priority && choice.score == best.score && choice.order < best.order) {
				best = choice
			}
		}
	}
	if best == nil {
		return nil
	}
	return &AIPlan{
		U: u, Path: best.path, Target: best.target, SpellID: best.spell.ID,
		NativeScoredCommands: s.nativeAIPlanScoredCommands(u),
	}
}

// NextAIPlan 找下一個未行動的 AI 單位並產生重製端近似計畫
// （不執行、不設 Acted）；它不是原版 0x14237/0x1548e 的替代實作。
func (s *State) NextAIPlan() *AIPlan {
	for _, u := range s.Units {
		if !u.OnField || !u.Alive() || u.Camp == Own || u.Acted || u.Paralyzed {
			continue
		}
		// Mode 11 has its own direct 0x1598A→0x15311→0x14237 dispatcher;
		// it must be consumed before the separate 0x14EF0 bridge is considered.
		if nativePlan, handled, err := s.nextNativeAIMode11Plan(u); handled {
			if err != nil {
				return &AIPlan{U: u, SpellID: -1, NativeError: err}
			}
			if nativePlan != nil {
				return nativePlan
			}
		}
		if nativePlan, handled, err := s.nextNativeAI14EF0Plan(u); handled {
			if err != nil {
				// A failed 0x14ef0 producer is only allowed to continue through
				// the exact raw mode fallback.  If that fallback is unavailable,
				// preserve the original evidence error and stop before the
				// normalized planner can consume the unit.
				if fallback, fallbackHandled, fallbackErr := s.nextNativeAIModeFallbackPlan(u); fallbackHandled {
					if fallbackErr != nil {
						return &AIPlan{U: u, SpellID: -1, NativeError: fallbackErr}
					}
					if fallback != nil {
						return fallback
					}
				}
				return &AIPlan{U: u, SpellID: -1, NativeError: err}
			}
			if nativePlan != nil {
				return nativePlan
			}
		}
		if nativePlan, handled, err := s.nextNativeAIPhysicalPlan(u); handled {
			if err != nil {
				if fallback, fallbackHandled, fallbackErr := s.nextNativeAIModeFallbackPlan(u); fallbackHandled {
					if fallbackErr != nil {
						return &AIPlan{U: u, SpellID: -1, NativeError: fallbackErr}
					}
					if fallback != nil {
						return fallback
					}
				}
				return &AIPlan{U: u, SpellID: -1, NativeError: err}
			}
			if nativePlan != nil {
				return nativePlan
			}
		}
		if nativePlan, handled, err := s.nextNativeAIModeFallbackPlan(u); handled {
			if err != nil {
				return &AIPlan{U: u, SpellID: -1, NativeError: err}
			}
			if nativePlan != nil {
				return nativePlan
			}
		}
		if spellPlan := s.nextAISpellPlan(u); spellPlan != nil {
			return spellPlan
		}
		best, moveTarget := s.aiTargets(u)
		if moveTarget == nil {
			return &AIPlan{U: u, SpellID: -1, NativeScoredCommands: s.nativeAIPlanScoredCommands(u)}
		}
		if best == nil {
			return &AIPlan{U: u, Path: s.aiApproachPath(u, moveTarget), SpellID: -1, NativeScoredCommands: s.nativeAIPlanScoredCommands(u)}
		}
		if s.InAttackRange(u, best.X, best.Y) {
			return &AIPlan{U: u, Target: best, SpellID: -1, NativeScoredCommands: s.nativeAIPlanScoredCommands(u)}
		}
		reach := s.Reachable(u)
		dstX, dstY := u.X, u.Y
		bestD := manhattan(u.X, u.Y, best.X, best.Y)
		for c := range reach {
			if s.UnitAt(c.X, c.Y) != nil {
				continue
			}
			d := manhattan(c.X, c.Y, best.X, best.Y)
			if d < bestD {
				bestD = d
				dstX, dstY = c.X, c.Y
			}
		}
		p := &AIPlan{U: u, Path: s.Path(u, dstX, dstY), SpellID: -1, NativeScoredCommands: s.nativeAIPlanScoredCommands(u)}
		// 到位後若可攻擊 best,帶上目標(引擎走完動畫再結算)
		du, dv := dstX-best.X, dstY-best.Y
		if du < 0 {
			du = -du
		}
		if dv < 0 {
			dv = -dv
		}
		if du+dv == 1 {
			p.Target = best
		}
		return p
	}
	return nil
}
