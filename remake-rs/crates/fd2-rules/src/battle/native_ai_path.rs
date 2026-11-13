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
    todo!("battle/native_ai_path.go:SelectNativeMovementDestination")
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
    todo!("battle/native_ai_path.go:NativePathBlockedCoordinate")
}

/// Go：`native_ai_path.go:88`，`func nativeMovementDestinationMetrics(candidate, intended Cell) (distance, imbalance int)`
/// Go 具名回傳值依序為：distance、imbalance。
pub(crate) fn native_movement_destination_metrics(candidate: Cell, intended: Cell) -> (i64, i64) {
    todo!("battle/native_ai_path.go:nativeMovementDestinationMetrics")
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
    todo!("battle/native_ai_path.go:SelectNativeNearestOppositeCoordinate")
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
    flags: &mut [u8],
    terrain_move_codes: &mut [u8],
    cost_row: &mut [u8],
) -> Result<(Vec<u8>, bool)> {
    todo!("battle/native_ai_path.go:NativePathDirections")
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
        todo!("battle/native_ai_path.go:nativePathSearch.walk")
    }
}

/// Go：`native_ai_path.go:236`，`func nativePathRunMetric(path []byte) byte`
pub(crate) fn native_path_run_metric(path: &[u8]) -> u8 {
    todo!("battle/native_ai_path.go:nativePathRunMetric")
}

impl NativePathSearch {
    /// Go：`native_ai_path.go:249`，`func (search *nativePathSearch) considerDestination(cell Cell)`
    pub(crate) fn consider_destination(&mut self, cell: Cell) {
        todo!("battle/native_ai_path.go:nativePathSearch.considerDestination")
    }
}

/// Go 版此處是匿名 struct（static）；Rust 需要具名型別，名稱由骨架產生。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePathNeighbour {
    pub(crate) dx: i64,
    pub(crate) dy: i64,
    pub(crate) code: u8,
}
