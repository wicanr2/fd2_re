// combat.go — 戰鬥結算 + 敵方 AI + 勝負(M1)。
//
// 傷害公式對映青衫/反組譯(doc 02/11/27):dmg = AP' − DP'(地形% 之後接);
// AI 評分式(doc 11 0x15140):選能對最有價值目標造成最大傷害的落點,dmg≤2 略過。
// 演出動畫(FIGANI/移動)後補;此處先把邏輯層做對,讓第一關可玩。
package battle

// Attack 近戰結算:dmg = max(1, AP-DP)。扣血、標記已行動。回傳傷害。
// 註:青衫「dmg≤2」是 AI「不值得打」門檻(doc11),非玩家不能打;玩家攻擊至少造成 1。
func (s *State) Attack(a, d *Unit) int {
	dmg := a.AP - d.DP
	if dmg < 1 {
		dmg = 1
	}
	d.HP -= dmg
	if d.HP < 0 {
		d.HP = 0
	}
	a.Acted = true
	return dmg
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

// aiActUnit 一個 AI 單位的行動:挑最佳目標,移到相鄰可達格,能打就打(doc11 評分式簡化版)。
func (s *State) aiActUnit(u *Unit) {
	// 1. 找最佳目標:可造成最大傷害者(dmg×擊殺加成);dmg≤2 視為不值得(但仍可能當移動目標)
	var best *Unit
	bestScore := -1 << 30
	for _, t := range s.Units {
		if !t.OnField || !t.Alive() || !hostile(u, t) {
			continue
		}
		dmg := u.AP - t.DP
		score := dmg
		if dmg >= t.HP { // 可擊殺 → 最高優先(doc11 prio 0x12)
			score = dmg*2 + 1000
		}
		// 距離越近越優先(同分時)
		score = score*100 - manhattan(u.X, u.Y, t.X, t.Y)
		if score > bestScore {
			bestScore = score
			best = t
		}
	}
	if best == nil {
		return
	}
	// 2. 已相鄰 → 直接打
	if s.InAttackRange(u, best.X, best.Y) {
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
	if s.InAttackRange(u, best.X, best.Y) {
		s.Attack(u, best)
	}
	u.Acted = true
}

// AITurn 讓所有非玩家、已登場、未行動的單位(敵 + 友軍 NPC)各行動一次。
func (s *State) AITurn() {
	for _, u := range s.Units {
		if !u.OnField || !u.Alive() || u.Camp == Own || u.Acted {
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
