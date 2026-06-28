package battle

// Reachable 回傳單位可移動到的格(flood-fill BFS,MV 步內,避開其他單位與邊界)。
// M1:每格成本 1;地形成本(doc 11)留待接入地形屬性後加權。
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

// MoveCost 進入該格的移動成本(M1:全平地=1;之後接地形屬性差異化)。
func (s *State) MoveCost(x, y int) int { return 1 }

// InAttackRange 目標是否在攻擊範圍(M1:相鄰 4 格的近戰;遠程之後加)。
func (s *State) InAttackRange(u *Unit, tx, ty int) bool {
	dx, dy := tx-u.X, ty-u.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx+dy == 1
}
