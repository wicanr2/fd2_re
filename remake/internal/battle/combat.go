// combat.go — 重製端的**近似**戰鬥結算 + 正規化 AI + 勝負(M1)。
//
// ⚠ 這個檔案裡的傷害公式**不是原版行為**，正式路徑也不再走它。玩家攻擊與原版
// AI 走 native_physical_attack.go 的 AttackNativePhysicalWithExperience，那條照
// `sub_29F72` 的順序、含武器暴擊加成、沒有傷害下限，而且會結算反擊。
//
// 這裡留下來的 Attack／AttackWithRNG 只有正規化 AI（AIStep／AITurn）與它們的
// 測試在用，兩者都沒有產品消費者。差別有三處，別把這裡的行為當成原版：
// 暴擊減半在地形修正**之前**（原版相反）、暴擊率只有職業表沒有武器加成、
// 玩家命中硬給至少 1 點傷害。
//
// 以下是這條近似路徑自己的來源（青衫/反組譯 doc 02 §4.1、doc 11、doc 27）:
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

	// Counter 是守方的反擊結果，nil 表示沒有反擊。原版一次物理攻擊含兩次結算
	// （`sub_28A6C` 的兩次 `sub_2939D`），反擊不給守方經驗，所以這一筆的
	// ExpGained 恆為 0。條件見 NativeCounterattackEligible。
	Counter *AttackResult
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
	return s.attackWithExperience(a, d, rng, nil)
}

// attackWithExperience 是近似路徑的結算：二手公式，只打一次，沒有反擊。
// 正式路徑見 native_physical_attack.go；兩者的差異列在本檔檔頭。
func (s *State) attackWithExperience(a, d *Unit, rng *rand.Rand, nativeEXP *nativePhysicalExperiencePlan) AttackResult {
	a.Acted = true
	if nativeEXP != nil && a.HasNativeRecordByte5 {
		a.NativeRecordByte5 |= 0x80
	}

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
		if nativeEXP != nil {
			exp = float64(min(nativeEXP.award(dmg, d.HP == 0), 99))
		}
		if s.killCancelsExp(d) {
			exp = 0
		}
		exp, levelUps = s.AwardExp(a, exp, rng)
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
// 預設規則(可被 scenario 覆寫):敵全滅 → win;指定要保護的單位死 → lose。
//
// 「敵全滅」只看已登場的單位，照原版 default handler 0x205b4／0x205be 的三值規則：
// 它只掃記錄表 [0x53a45]，還沒被回合事件 0x10B4E 追加進表的援軍群組不存在於判定裡。
// 第五章收據（docs/data/ui-traces/parity-ch05.json）在第 7 回合的 group 3 還沒登場時
// 清場，原版當回合就進戰後對白。
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
	if s.AliveCount(Enemy) == 0 {
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
	NativeActionKind NativeAIActionKind
	NativeCommandID  int
	NativeItemSlot   int
	NativeItemID     int
	// NativeItemTargetIndices preserves the complete raw roster-order list
	// scored by 0x1567e.  The 0x15055 owner consumes the whole list after
	// movement; Target is only the first target used by the movement planner.
	NativeItemTargetIndices []byte
	NativeActionDestination Cell
	// NativeModeIntended／NativeModeCandidates 只作對拍診斷：mode 備援的
	// 0x14121／0x13E9C 目標格與 0x14B16 候選（列優先順序）。不參與執行。
	NativeModeIntended     Cell
	NativeModeCandidates   []Cell
	NativeModeBlockedCell  Cell
	NativeModeBlockedFound bool
	NativeModeOpposite     []Cell
	NativeAI14EF0Route     NativeAI14EF0Tail
	NativeActionScore      int
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
	// NativeIdleRecovery 保存非 mode11 分派器（dispatcher）抵達的已接受 0x13FD4
	// 原始決策。已證實 fallback 上的 nil 表示恢復閘門（recovery gate）拒絕，
	// 共用收尾仍須完成該單位。
	NativeIdleRecovery *NativeAIIdleRecoveryDecision
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
	if plan, handled := s.nextNativeScannedAIPlan(3); handled {
		return plan
	}
	for _, u := range s.Units {
		if !u.OnField || !u.Alive() || u.Camp == Own || u.Acted || u.Paralyzed {
			continue
		}
		if plan := s.nextAIPlanForUnit(u); plan != nil {
			return plan
		}
	}
	return nil
}

// nativeAIScanState 是 0x1D80B／0x1D8BA／0x1D988 三遍掃描的游標：pass 0 是友軍
// （raw +6==1）單遍；pass 1 是敵軍（+6==0）預選遍，只有 0x1598A 的 [0x53C23]
// 或 0x1567E 的 [0x53C33] 有號 >= 6 的單位才在這一遍行動；pass 2 是敵軍第二遍。
// 每一遍都依 record 順序走一次，被跳過的單位不會在同一遍回頭。
type nativeAIScanState struct {
	active bool
	pass   int
	index  int
}

// ResetNativeAIScan 在敵方階段開始時重設掃描游標。
func (s *State) ResetNativeAIScan() {
	if s != nil {
		s.nativeAIScan = nativeAIScanState{}
	}
}

// NextAllyAIPlan 只跑 0x1D80B 那一遍（pass 0，raw +6==1 的友軍）。0x1A30B 在回合
// 橫幅之前先跑友軍 AI，橫幅與 0x13536 之後才是敵軍兩遍；pass 0 跑完回 (nil, true)，
// 游標停在 pass 1，之後 NextAIPlan 從敵軍第一遍接下去。沒有 raw +6 provenance 的
// 名冊回 handled=false，呼叫端直接進橫幅。
func (s *State) NextAllyAIPlan() (*AIPlan, bool) {
	return s.nextNativeScannedAIPlan(1)
}

// nextNativeScannedAIPlan 依原版三遍順序挑下一個行動單位，只掃 pass < passLimit
// 的那幾遍。沒有 raw +6 provenance 的名冊回 handled=false，交回舊的單遍迴圈。
func (s *State) nextNativeScannedAIPlan(passLimit int) (*AIPlan, bool) {
	if s == nil || len(s.Units) == 0 {
		return nil, false
	}
	for _, u := range s.Units {
		if u != nil && u.OnField && u.Alive() && !u.HasNativeRecordByte6 {
			return nil, false
		}
	}
	if !s.nativeAIScan.active {
		s.nativeAIScan = nativeAIScanState{active: true}
	}
	for pass := s.nativeAIScan.pass; pass < passLimit; pass++ {
		start := 0
		if pass == s.nativeAIScan.pass {
			start = s.nativeAIScan.index
		}
		for i := start; i < len(s.Units); i++ {
			s.nativeAIScan = nativeAIScanState{active: true, pass: pass, index: i + 1}
			u := s.Units[i]
			if !s.nativeAIScanEligible(u, pass) {
				continue
			}
			if pass == 1 && !s.nativeAIPreselected(u) {
				continue
			}
			if plan := s.nextAIPlanForUnit(u); plan != nil {
				if plan.NativeError != nil {
					// 失敗即關閉會中止這個階段；下一次呼叫從頭掃。
					s.nativeAIScan = nativeAIScanState{}
				}
				return plan, true
			}
		}
		s.nativeAIScan = nativeAIScanState{active: true, pass: pass + 1}
	}
	if passLimit < 3 {
		// 只跑了前面幾遍：游標留在下一遍的起點，敵軍兩遍稍後接著掃。
		s.nativeAIScan = nativeAIScanState{active: true, pass: passLimit}
		return nil, true
	}
	s.nativeAIScan = nativeAIScanState{}
	return nil, true
}

// nativeAIScanEligible 是 0x1D80B（pass 0，+6==1）與 0x1D8BA／0x1D988
// （pass 1／2，+6==0）共用的入場條件：`(+5 & 0x81)==0` 且 `+0x26==0`。
func (s *State) nativeAIScanEligible(u *Unit, pass int) bool {
	if u == nil || !u.OnField || !u.Alive() || u.Acted || u.Paralyzed || u.Camp == Own {
		return false
	}
	want := byte(0)
	if pass == 0 {
		want = 1
	}
	if u.NativeRecordByte6 != want {
		return false
	}
	if u.HasNativeRecordByte5 && u.NativeRecordByte5&0x81 != 0 {
		return false
	}
	if transient, ok := u.NativeTransientDuration(0x26); ok && transient != 0 {
		return false
	}
	return true
}

// nativeAIPreselected 重現 0x1D8BA 預選遍的門檻：0x1598A(unit,0) 與 0x1567E(unit,0)
// 之後，[0x53C23] >= 6 或 [0x53C33] >= 6（有號比較）才在第一遍行動。評分來源
// 不齊時視為未入選，留到第二遍由既有失敗即關閉路徑處理。
func (s *State) nativeAIPreselected(u *Unit) bool {
	actor, selector, records, _, baseFlags, costRow, err := s.nativeAIModeRuntimeContext(u)
	if err != nil || selector != 0 {
		return false
	}
	command, err := ScoreNativeAI1598A(
		s.W, s.H, records, len(s.Units), actor, selector, u,
		s.NativeCommandBook, baseFlags, s.NativeTerrainMoveCodes, costRow, nil,
	)
	if err != nil {
		return false
	}
	if command.MaxScore >= 6 {
		return true
	}
	item, err := ScoreNativeAI1567E(
		s.W, s.H, records, len(s.Units), actor, selector,
		s.nativeFutureItemRows, s.NativeCommandBook, baseFlags,
	)
	if err != nil {
		return false
	}
	return item.MaxScore >= 6
}

// nextAIPlanForUnit 是原本 NextAIPlan 迴圈本體：依 mode 11 → 0x14EF0 → mode 2
// 物理 → mode 備援 → 法術 → 正規化規劃器的順序替一個單位產生計畫。
func (s *State) nextAIPlanForUnit(u *Unit) *AIPlan {
	{
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
				// mode 2 的 0x14EF0 生產端可能因 0x14237 沒有候選而無法建立
				// target provenance；原版 caller 此時會消費 0x13FD4，不是進入
				// 通用 mode fallback。先交回已閉合的物理／恢復橋，仍失敗才
				// 保留失敗即關閉。
				if physical, physicalHandled, physicalErr := s.nextNativeAIPhysicalPlan(u); physicalHandled {
					if physicalErr != nil {
						return &AIPlan{U: u, SpellID: -1, NativeError: physicalErr}
					}
					if physical != nil {
						return physical
					}
				}
				// 其他 mode 只允許進入各自的原始分派後備；若沒有正式
				// consumer，保留證據錯誤，不讓正規化規劃器接手。
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
}
