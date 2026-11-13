//! Go 來源：`remake/internal/battle/native_ai_path.go`（255 行）。

use super::*;

/// Go 原式：`iota`
pub const NATIVE_PATH_DOWN: u8 = 0;

pub const NATIVE_PATH_LEFT: u8 = 1;

pub const NATIVE_PATH_UP: u8 = 2;

pub const NATIVE_PATH_RIGHT: u8 = 3;

/// SelectNativeMovementDestination mirrors 0x14b78's 0x14dc5..0x14e56
/// reachable-cell ranking. It first minimizes Manhattan distance to intended,
/// then minimizes the absolute X/Y distance imbalance. Exact ties preserve the
/// caller's existing candidate order.
///
/// Go：`native_ai_path.go:16`，`func SelectNativeMovementDestination(candidates []Cell, intended Cell) (Cell, bool)`
pub fn select_native_movement_destination(candidates: &[Cell], intended: Cell) -> (Cell, bool) {
    let mut best = intended;
    let (mut best_distance, mut best_imbalance) = (0xff, 0xff);
    let mut found = false;
    for &candidate in candidates {
        let (distance, imbalance) = native_movement_destination_metrics(candidate, intended);
        if distance < best_distance || distance == best_distance && imbalance < best_imbalance {
            best = candidate;
            (best_distance, best_imbalance) = (distance, imbalance);
            found = true;
        }
    }
    (best, found)
}

/// NativePathBlockedCoordinate reproduces the mode-2 use of 0x4e1a6 in
/// 0x14121. Unlike modes 0/1, mode 2 enters 0x40 cells, records their
/// coordinates, and keeps traversing. Every later accepted blocked cell
/// overwrites the prior result, so the answer follows the original DFS order
/// rather than a nearest-distance rule.
///
/// Go：`native_ai_path.go:36`，`func NativePathBlockedCoordinate( w, h int, start Cell, initialBudget int, flags, terrainMoveCodes, costRow []byte, ) (Cell, bool, error)`
pub fn native_path_blocked_coordinate(
    w: i64,
    h: i64,
    start: Cell,
    initial_budget: i64,
    flags: &[u8],
    terrain_move_codes: &[u8],
    cost_row: &[u8],
) -> Result<(Cell, bool)> {
    if w <= 0
        || h <= 0
        || start.x < 0
        || start.y < 0
        || start.x >= w
        || start.y >= h
        || !(0..=0xff).contains(&initial_budget)
        || flags.len() as i64 != w.wrapping_mul(h)
        || terrain_move_codes.len() as i64 != w.wrapping_mul(h)
        || cost_row.len() as i64 != NATIVE_MOVEMENT_COST_ROW_SIZE
    {
        return Err(battle_err!("native blocked-coordinate path inputs are malformed"));
    }
    for (cell, &code) in terrain_move_codes.iter().enumerate() {
        if code as usize >= cost_row.len() {
            return Err(battle_err!("native blocked-coordinate terrain code {code} at cell {cell} is out of bounds"));
        }
    }
    let mut budgets = vec![0xffu8; (w * h) as usize];
    budgets[(start.y * w + start.x) as usize] = initial_budget as u8;

    struct Walk<'a> {
        w: i64,
        h: i64,
        flags: &'a [u8],
        terrain_move_codes: &'a [u8],
        cost_row: &'a [u8],
        budgets: Vec<u8>,
        result: Cell,
        found: bool,
    }
    fn walk(ctx: &mut Walk<'_>, cell: Cell, budget: u8) {
        for neighbour in NATIVE_PATH_NEIGHBOURS {
            let next = Cell { x: cell.x + neighbour.dx, y: cell.y + neighbour.dy };
            if next.x < 0 || next.y < 0 || next.x >= ctx.w || next.y >= ctx.h {
                continue;
            }
            let index = (next.y * ctx.w + next.x) as usize;
            let cost = ctx.cost_row[ctx.terrain_move_codes[index] as usize];
            if cost > budget {
                continue;
            }
            let next_budget = budget - cost;
            if next_budget as i8 <= ctx.budgets[index] as i8 {
                continue;
            }
            ctx.budgets[index] = next_budget;
            if ctx.flags[index] & NATIVE_COMMAND_GRID_BLOCKED != 0 {
                (ctx.result, ctx.found) = (next, true);
            }
            walk(ctx, next, next_budget);
        }
    }
    let mut ctx = Walk { w, h, flags, terrain_move_codes, cost_row, budgets, result: Cell::default(), found: false };
    walk(&mut ctx, start, initial_budget as u8);
    Ok((ctx.result, ctx.found))
}

/// Go：`native_ai_path.go:88`，`func nativeMovementDestinationMetrics(candidate, intended Cell) (distance, imbalance int)`
/// Go 具名回傳值依序為：distance、imbalance。
pub(crate) fn native_movement_destination_metrics(candidate: Cell, intended: Cell) -> (i64, i64) {
    let dx = abs_int(candidate.x - intended.x);
    let dy = abs_int(candidate.y - intended.y);
    (dx + dy, abs_int(dx - dy))
}

/// SelectNativeNearestOppositeCoordinate mirrors 0x13e9c's final movement
/// fallback. selector remains raw: zero scans records with record+6 != 0;
/// nonzero scans record+6 == 0. The routine does not apply active/dead gates
/// and retains the first runtime record on an exact Manhattan-distance tie.
///
/// Go：`native_ai_path.go:98`，`func SelectNativeNearestOppositeCoordinate( records []byte, count, actor, selector int, ) (Cell, bool, error)`
pub fn select_native_nearest_opposite_coordinate(
    records: &[u8],
    count: i64,
    actor: i64,
    selector: i64,
) -> Result<(Cell, bool)> {
    if count < 0 || count > records.len() as i64 / NATIVE_RECORD_SIZE || actor < 0 || actor >= count {
        return Err(battle_err!("native nearest-coordinate roster/actor is malformed"));
    }
    let actor_record = &records[(actor * NATIVE_RECORD_SIZE) as usize..];
    let origin = Cell { x: actor_record[0] as i64, y: actor_record[1] as i64 };
    let mut best_distance = 0xffff;
    let mut best = Cell::default();
    let mut found = false;
    for unit in 0..count {
        let record = &records[(unit * NATIVE_RECORD_SIZE) as usize..];
        if !native_ai_opposite_selector_group(record[6], selector) {
            continue;
        }
        let candidate = Cell { x: record[0] as i64, y: record[1] as i64 };
        let distance = abs_int(origin.x - candidate.x) + abs_int(origin.y - candidate.y);
        if distance >= best_distance {
            continue;
        }
        (best, best_distance, found) = (candidate, distance, true);
    }
    Ok((best, found))
}

/// NativePathDirections reproduces 0x4e1a6 modes 0 and 1. It returns the
/// original direction bytes consumed by 0x13488. Mode 0 keeps only a strictly
/// better remaining budget per cell. Mode 1 may replace an equal-budget visit
/// when the current route has more direction runs, matching 0x4e35b..0x4e373.
///
/// flags is FDFIELD entry+2 and terrainMoveCodes is the already validated
/// FDSHAP record+1 projection used by the original cost-row lookup.
///
/// Go：`native_ai_path.go:132`，`func NativePathDirections( w, h int, start, destination Cell, initialBudget, mode int, flags, terrainMoveCodes, costRow []byte, ) ([]byte, bool, error)`
pub fn native_path_directions(
    w: i64,
    h: i64,
    start: Cell,
    destination: Cell,
    initial_budget: i64,
    mode: i64,
    flags: &[u8],
    terrain_move_codes: &[u8],
    cost_row: &[u8],
) -> Result<(Vec<u8>, bool)> {
    if w <= 0
        || h <= 0
        || start.x < 0
        || start.y < 0
        || start.x >= w
        || start.y >= h
        || destination.x < 0
        || destination.y < 0
        || destination.x >= w
        || destination.y >= h
        || !(0..=0xff).contains(&initial_budget)
        || (mode != 0 && mode != 1)
        || flags.len() as i64 != w.wrapping_mul(h)
        || terrain_move_codes.len() as i64 != w.wrapping_mul(h)
        || cost_row.len() as i64 != NATIVE_MOVEMENT_COST_ROW_SIZE
    {
        return Err(battle_err!("native path inputs are malformed"));
    }
    for (cell, &code) in terrain_move_codes.iter().enumerate() {
        if code as usize >= cost_row.len() {
            return Err(battle_err!("native path terrain code {code} at cell {cell} is out of bounds"));
        }
    }

    let mut search = NativePathSearch {
        w,
        h,
        destination,
        mode,
        flags: flags.to_vec(),
        terrain_move_codes: terrain_move_codes.to_vec(),
        cost_row: cost_row.to_vec(),
        budgets: vec![0xff; (w * h) as usize],
        run_metrics: vec![0; (w * h) as usize],
        path: Vec::new(),
        best_path: Vec::new(),
        best_length: 0xff,
    };
    search.budgets[(start.y * w + start.x) as usize] = initial_budget as u8;
    search.consider_destination(start);
    search.walk(start, initial_budget as u8);
    if search.best_length == 0xff {
        return Ok((Vec::new(), false));
    }
    Ok((search.best_path.clone(), true))
}

/// Go：`native_ai_path.go:177`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativePathSearch {
    pub(crate) w: i64,
    pub(crate) h: i64,
    pub(crate) destination: Cell,
    pub(crate) mode: i64,
    pub(crate) flags: Vec<u8>,
    pub(crate) terrain_move_codes: Vec<u8>,
    pub(crate) cost_row: Vec<u8>,
    pub(crate) budgets: Vec<u8>,
    pub(crate) run_metrics: Vec<u8>,
    pub(crate) path: Vec<u8>,
    pub(crate) best_path: Vec<u8>,
    pub(crate) best_length: i64,
}

/// Go：`native_ai_path.go:191`
pub(crate) static NATIVE_PATH_NEIGHBOURS: &[NativePathNeighbour] = &[
    NativePathNeighbour { dx: 1, dy: 0, code: 3 },
    NativePathNeighbour { dx: -1, dy: 0, code: 1 },
    NativePathNeighbour { dx: 0, dy: 1, code: 0 },
    NativePathNeighbour { dx: 0, dy: -1, code: 2 },
];

impl NativePathSearch {
    /// Go：`native_ai_path.go:201`，`func (search *nativePathSearch) walk(cell Cell, budget byte)`
    pub(crate) fn walk(&mut self, cell: Cell, budget: u8) {
        for neighbour in NATIVE_PATH_NEIGHBOURS {
            let next = Cell { x: cell.x + neighbour.dx, y: cell.y + neighbour.dy };
            if next.x < 0 || next.y < 0 || next.x >= self.w || next.y >= self.h {
                continue;
            }
            let index = (next.y * self.w + next.x) as usize;
            let cost = self.cost_row[self.terrain_move_codes[index] as usize];
            if cost > budget {
                continue;
            }
            let mut next_budget = budget - cost;
            let old_budget = self.budgets[index];
            self.path.push(neighbour.code);
            let run_metric = native_path_run_metric(&self.path);
            let mut accept = next_budget as i8 > old_budget as i8;
            if next_budget as i8 == old_budget as i8 && self.mode == 1 {
                accept = run_metric > self.run_metrics[index];
            }
            if accept {
                // 0x4e37a updates the path-run metadata before testing flags.
                self.run_metrics[index] = run_metric;
                if self.flags[index] & NATIVE_COMMAND_GRID_BLOCKED == 0 {
                    if self.flags[index] & NATIVE_COMMAND_GRID_ZERO_BUDGET != 0 {
                        next_budget = 0;
                    }
                    self.budgets[index] = next_budget;
                    self.consider_destination(next);
                    self.walk(next, next_budget);
                }
            }
            self.path.pop();
        }
    }
}

/// Go：`native_ai_path.go:236`，`func nativePathRunMetric(path []byte) byte`
pub(crate) fn native_path_run_metric(path: &[u8]) -> u8 {
    if path.is_empty() {
        return 0;
    }
    let mut runs: i64 = 1;
    for index in 1..path.len() {
        if path[index] != path[index - 1] {
            runs += 1;
        }
    }
    (runs << 2) as u8
}

impl NativePathSearch {
    /// Go：`native_ai_path.go:249`，`func (search *nativePathSearch) considerDestination(cell Cell)`
    pub(crate) fn consider_destination(&mut self, cell: Cell) {
        if cell != self.destination || self.path.len() as i64 > self.best_length {
            return;
        }
        self.best_length = self.path.len() as i64;
        self.best_path.clone_from(&self.path);
    }
}

/// Go 版此處是匿名 struct（static）；Rust 需要具名型別，名稱由骨架產生。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePathNeighbour {
    pub(crate) dx: i64,
    pub(crate) dy: i64,
    pub(crate) code: u8,
}

#[cfg(test)]
mod native_ai_path_test;
