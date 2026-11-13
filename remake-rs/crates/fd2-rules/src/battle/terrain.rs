//! Go 來源：`remake/internal/battle/terrain.go`（41 行）。
//!
//! terrain.go — FDSHAP control byte+1 的原版地形 AP/DP 修正。0x1acf3 以該 byte
//! 索引 0x51a12 (AP) 和 0x51a2a (DP)；NativeTerrainMoveCodes 是匯出 map 的逐格
//! 原始索引，不以歸一化的 MoveCost 猜測地形。

use super::*;

/// NativeTerrainAPDPPct is the verified six-entry static table.  The bool is
/// false for an unrecognised control value so callers can fail closed.
///
/// Go：`terrain.go:8`，`func NativeTerrainAPDPPct(moveCode byte) (apPct, dpPct int, ok bool)`
/// Go 具名回傳值依序為：apPct、dpPct、ok。
pub fn native_terrain_apdp_pct(move_code: u8) -> (i64, i64, bool) {
    match move_code {
        0 => (5, 0, true),
        1 | 5 => (0, 0, true),
        2 | 3 => (-5, 10, true),
        4 => (-5, -5, true),
        _ => (0, 0, false),
    }
}

impl State {
    /// TerrainAPDPPct returns a cell's combat modifiers.  Complete native map
    /// exports take precedence.  Old/incomplete maps retain the historical Cost
    /// fallback so loading them remains compatible; that fallback is deliberately
    /// not evidence of original terrain semantics.
    ///
    /// Go：`terrain.go:27`，`func (s *State) TerrainAPDPPct(x, y int) (apPct, dpPct int)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 具名回傳值依序為：apPct、dpPct。
    pub fn terrain_apdp_pct(&self, x: i64, y: i64) -> (i64, i64) {
        if x >= 0
            && x < self.w
            && y >= 0
            && y < self.h
            && self.native_terrain_move_codes.len() as i64 == self.w.wrapping_mul(self.h)
        {
            let (ap_pct, dp_pct, ok) =
                native_terrain_apdp_pct(self.native_terrain_move_codes[(y * self.w + x) as usize]);
            if ok {
                return (ap_pct, dp_pct);
            }
        }
        match self.move_cost(x, y) {
            // 沼澤(export_engine_assets.py:move_code 4 -> cost 2)
            2 => (-5, -5),
            // 正常(含目前無法分辨的森林,見檔頭與本函式註解;s.Cost==nil/越界也落此分支)
            1 => (5, -5),
            // 不可通行(cost>=BLOCKED_COST)等其他值:不套地形修正
            _ => (0, 0),
        }
    }
}

#[cfg(test)]
mod native_combat_tables_test;

#[cfg(test)]
mod terrain_native_test;
