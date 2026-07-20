// combat.go — 戰鬥結算 + 敵方 AI + 勝負(M1)。
//
// 傷害公式對映青衫/反組譯(doc 02 §4.1、doc 11、doc 27 checklist):
//
//	命中率 = (攻方HIT − 守方EV)%
//	暴擊時 DP = 守方DP/2(取整)
//	AP = AP×(1+攻方地形AP%)、DP = DP×(1+守方地形DP%)(取整,terrain.go)
//	最大傷害 = AP − DP;實際傷害 = 最大傷害×0.9 ～ 最大傷害-1(亂數,magic.go randomizeAmount)
//
// AI 評分式(doc 11 0x15140):選能對最有價值目標造成最大傷害的落點,dmg≤2 略過,
// 現含地形% 修正(doc42 gap-audit 第 5 項)。
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
	d.HP -= dmg
	if d.HP < 0 {
		d.HP = 0
	}

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
		levelUps = GainExp(a, exp, rng)
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

// estDamage AI 評分用的預估傷害(doc11 0x15140 反組譯公式):
//
//	myAP'  = myAP  × 地形AP%[u當下座標] / 100
//	tarDP' = tarDP × 地形DP%[t當下座標] / 100
//	估計傷害 = myAP' − tarDP'
//
// 只是選目標用的估值,不擲骰(不含命中率/暴擊/傷害隨機化——那些留給 AttackWithRNG 實際結算,
// doc42 gap-audit 第 5 項只要求 AI 評分補上地形%,未要求 AI 決策也模擬命中/暴擊機率)。
func (s *State) estDamage(u, t *Unit) int {
	apPct, _ := s.TerrainAPDPPct(u.X, u.Y)
	_, dpPct := s.TerrainAPDPPct(t.X, t.Y)
	ap := u.AP * (100 + apPct) / 100
	dp := t.DP * (100 + dpPct) / 100
	return ap - dp
}

// aiTargets separates the original AI's attack candidate from its movement
// fallback.  The 0x15140 scorer ignores targets whose estimated damage is at
// most two; when every hostile target is below that threshold, the unit may
// still advance toward the nearest hostile but must not attack it.
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

// aiActUnit 一個 AI 單位的行動:挑最佳目標,移到攻擊範圍內可達格,能打就打(doc11 評分式簡化版)。
func (s *State) aiActUnit(u *Unit) {
	// 1. 找最佳攻擊目標；dmg≤2 略過，但保留最近敵人作為移動目標。
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
	u.X, u.Y = dstX, dstY
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
	U      *Unit
	Path   []Cell // 含起點;len>=2 = 要移動(引擎播行走動畫)
	Target *Unit  // 到位後攻擊目標(nil = 僅移動/待機)
}

// NextAIPlan 找下一個未行動的 AI 單位並產生行動計畫(不執行、不設 Acted);
// 全部動完回 nil。決策邏輯同 aiActUnit(doc11 評分式)。
func (s *State) NextAIPlan() *AIPlan {
	for _, u := range s.Units {
		if !u.OnField || !u.Alive() || u.Camp == Own || u.Acted || u.Paralyzed {
			continue
		}
		best, moveTarget := s.aiTargets(u)
		if moveTarget == nil {
			return &AIPlan{U: u}
		}
		if best == nil {
			return &AIPlan{U: u, Path: s.aiApproachPath(u, moveTarget)}
		}
		if s.InAttackRange(u, best.X, best.Y) {
			return &AIPlan{U: u, Target: best}
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
		p := &AIPlan{U: u, Path: s.Path(u, dstX, dstY)}
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
