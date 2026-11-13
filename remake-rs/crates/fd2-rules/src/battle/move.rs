//! Go 來源：`remake/internal/battle/move.go`（117 行）。

use super::*;

impl State {
    /// Reachable 回傳單位可移動到的格(flood-fill BFS,MV 步內,避開其他單位與邊界)。
    /// 成本依 MoveCost 查地形表(worklist 第 8 輪接上;無資料時退回全平地=1)。
    ///
    /// Go：`move.go:5`，`func (s *State) Reachable(u *Unit) map[Cell]bool`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn reachable(&self, u: UnitIdx) -> BTreeMap<Cell, bool> {
        let unit = &self.units[u];
        let start = Cell { x: unit.x, y: unit.y };
        let mut res = BTreeMap::from([(start, true)]);
        let mut cost = BTreeMap::from([(start, 0i64)]);
        let mut q = std::collections::VecDeque::from([start]);
        while let Some(c) = q.pop_front() {
            let here = cost[&c];
            if here >= unit.mv {
                continue;
            }
            for (dx, dy) in [(0, -1), (0, 1), (-1, 0), (1, 0)] {
                let (nx, ny) = (c.x + dx, c.y + dy);
                if nx < 0 || ny < 0 || nx >= self.w || ny >= self.h {
                    continue;
                }
                let nc = Cell { x: nx, y: ny };
                if cost.contains_key(&nc) {
                    continue;
                }
                // 他人擋路(同陣營也擋,簡化)
                if self.unit_at(nx, ny).is_some_and(|o| o != u) {
                    continue;
                }
                let nco = here + self.move_cost(nx, ny);
                if nco <= unit.mv {
                    cost.insert(nc, nco);
                    res.insert(nc, true);
                    q.push_back(nc);
                }
            }
        }
        res
    }

    /// MoveCost 進入該格的移動成本。查 s.Cost(Load 從 map.json "cost" 陣列接上,doc01 §5
    /// 地形控制表換算;worklist 第 8 輪「地形屬性接線」);無地形資料(s.Cost==nil,如舊測試
    /// 直接手寫 State{})或座標越界一律回 1(平地)。不可通行地形回一個遠大於任何 MV 的值
    /// (export_engine_assets.py 的 BLOCKED_COST=99),Reachable/Path 的 `nco <= u.MV` 判斷
    /// 天然把它篩掉,不需要另外特判「牆」。
    ///
    /// Go：`move.go:43`，`func (s *State) MoveCost(x, y int) int`
    pub fn move_cost(&self, x: i64, y: i64) -> i64 {
        match &self.cost {
            Some(cost) if x >= 0 && y >= 0 && x < self.w && y < self.h => cost[(y * self.w + x) as usize],
            _ => 1,
        }
    }

    /// InAttackRange 目標是否在攻擊範圍(曼哈頓距離落在 [AtkMin,AtkMax] 內;doc32 依武器
    /// type 決定,如騎士槍type3=[1,2]。AtkMin/AtkMax 未設(0)一律視為預設 1,等同舊版
    /// 「只查相鄰 4 格」行為不變)。
    ///
    /// Go：`move.go:53`，`func (s *State) InAttackRange(u *Unit, tx, ty int) bool`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn in_attack_range(&self, u: UnitIdx, tx: i64, ty: i64) -> bool {
        let unit = &self.units[u];
        let d = (tx - unit.x).abs() + (ty - unit.y).abs();
        let min = if unit.atk_min == 0 { 1 } else { unit.atk_min };
        let max = if unit.atk_max == 0 { 1 } else { unit.atk_max };
        d >= min && d <= max
    }

    /// Path 回傳 u 走到 (tx,ty) 的逐格路徑(含起點;BFS,同 Reachable 規則)。不可達回 nil。
    ///
    /// Go：`move.go:73`，`func (s *State) Path(u *Unit, tx, ty int) []Cell`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn path(&self, u: UnitIdx, tx: i64, ty: i64) -> Vec<Cell> {
        let unit = &self.units[u];
        let start = Cell { x: unit.x, y: unit.y };
        let goal = Cell { x: tx, y: ty };
        if start == goal {
            return vec![start];
        }
        let mut cost = BTreeMap::from([(start, 0i64)]);
        let mut par: BTreeMap<Cell, Cell> = BTreeMap::new();
        let mut q = std::collections::VecDeque::from([start]);
        while let Some(c) = q.pop_front() {
            let here = cost[&c];
            if here >= unit.mv {
                continue;
            }
            for (dx, dy) in [(0, -1), (0, 1), (-1, 0), (1, 0)] {
                let (nx, ny) = (c.x + dx, c.y + dy);
                if nx < 0 || ny < 0 || nx >= self.w || ny >= self.h {
                    continue;
                }
                let nc = Cell { x: nx, y: ny };
                if cost.contains_key(&nc) {
                    continue;
                }
                if self.unit_at(nx, ny).is_some_and(|o| o != u) {
                    continue;
                }
                let nco = here + self.move_cost(nx, ny);
                if nco <= unit.mv {
                    cost.insert(nc, nco);
                    par.insert(nc, c);
                    q.push_back(nc);
                }
            }
        }
        if !cost.contains_key(&goal) {
            return Vec::new();
        }
        let mut path = vec![goal];
        let mut p = goal;
        while p != start {
            p = par[&p];
            path.push(p);
        }
        path.reverse();
        path
    }
}

#[cfg(test)]
mod move_test;
