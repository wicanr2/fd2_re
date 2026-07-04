package battle

// Reachable 回傳單位可移動到的格(flood-fill BFS,MV 步內,避開其他單位與邊界)。
// 成本依 MoveCost 查地形表(worklist 第 8 輪接上;無資料時退回全平地=1)。
func (s *State) Reachable(u *Unit) map[Cell]bool {
	res := map[Cell]bool{{u.X, u.Y}: true}
	cost := map[Cell]int{{u.X, u.Y}: 0}
	q := []Cell{{u.X, u.Y}}
	for len(q) > 0 {
		c := q[0]
		q = q[1:]
		if cost[c] >= u.MV {
			continue
		}
		for _, d := range [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
			nx, ny := c.X+d[0], c.Y+d[1]
			if nx < 0 || ny < 0 || nx >= s.W || ny >= s.H {
				continue
			}
			nc := Cell{nx, ny}
			if _, seen := cost[nc]; seen {
				continue
			}
			if o := s.UnitAt(nx, ny); o != nil && o != u { // 他人擋路(同陣營也擋,簡化)
				continue
			}
			nco := cost[c] + s.MoveCost(nx, ny)
			if nco <= u.MV {
				cost[nc] = nco
				res[nc] = true
				q = append(q, nc)
			}
		}
	}
	return res
}

// MoveCost 進入該格的移動成本。查 s.Cost(Load 從 map.json "cost" 陣列接上,doc01 §5
// 地形控制表換算;worklist 第 8 輪「地形屬性接線」);無地形資料(s.Cost==nil,如舊測試
// 直接手寫 State{})或座標越界一律回 1(平地)。不可通行地形回一個遠大於任何 MV 的值
// (export_engine_assets.py 的 BLOCKED_COST=99),Reachable/Path 的 `nco <= u.MV` 判斷
// 天然把它篩掉,不需要另外特判「牆」。
func (s *State) MoveCost(x, y int) int {
	if s.Cost == nil || x < 0 || y < 0 || x >= s.W || y >= s.H {
		return 1
	}
	return s.Cost[y*s.W+x]
}

// InAttackRange 目標是否在攻擊範圍(曼哈頓距離落在 [AtkMin,AtkMax] 內;doc32 依武器
// type 決定,如騎士槍type3=[1,2]。AtkMin/AtkMax 未設(0)一律視為預設 1,等同舊版
// 「只查相鄰 4 格」行為不變)。
func (s *State) InAttackRange(u *Unit, tx, ty int) bool {
	dx, dy := tx-u.X, ty-u.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	d := dx + dy
	min, max := u.AtkMin, u.AtkMax
	if min == 0 {
		min = 1
	}
	if max == 0 {
		max = 1
	}
	return d >= min && d <= max
}

// Path 回傳 u 走到 (tx,ty) 的逐格路徑(含起點;BFS,同 Reachable 規則)。不可達回 nil。
func (s *State) Path(u *Unit, tx, ty int) []Cell {
	start := Cell{X: u.X, Y: u.Y}
	goal := Cell{X: tx, Y: ty}
	if start == goal {
		return []Cell{start}
	}
	cost := map[Cell]int{start: 0}
	par := map[Cell]Cell{}
	q := []Cell{start}
	for len(q) > 0 {
		c := q[0]
		q = q[1:]
		if cost[c] >= u.MV {
			continue
		}
		for _, d := range [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
			nx, ny := c.X+d[0], c.Y+d[1]
			if nx < 0 || ny < 0 || nx >= s.W || ny >= s.H {
				continue
			}
			nc := Cell{X: nx, Y: ny}
			if _, seen := cost[nc]; seen {
				continue
			}
			if o := s.UnitAt(nx, ny); o != nil && o != u {
				continue
			}
			nco := cost[c] + s.MoveCost(nx, ny)
			if nco <= u.MV {
				cost[nc] = nco
				par[nc] = c
				q = append(q, nc)
			}
		}
	}
	if _, ok := cost[goal]; !ok {
		return nil
	}
	path := []Cell{goal}
	for p := goal; p != start; {
		p = par[p]
		path = append([]Cell{p}, path...)
	}
	return path
}
