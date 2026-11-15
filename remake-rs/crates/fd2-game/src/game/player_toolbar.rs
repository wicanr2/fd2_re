//! Go 來源：`remake/cmd/fd2/player_toolbar.go`（324 行）。
//!
//! R5 階段 4（r5w）只移植觸控座標換算 `playerTouchCoordinates`：第一章的地圖觸控（`main.go:9095`）經過它。
//! 工具列本體（`modernHDPresentation` 的顯示、點擊、標籤與繪製）在 R6，見 `PORTING.md`「R5 階段 4：r5w」。

use super::*;

/// Go：`player_toolbar.go:15`，`const playerToolbarHeight = 24`。
pub(crate) const PLAYER_TOOLBAR_HEIGHT: i64 = 24;

impl Game {
    /// Go：`player_toolbar.go:26`，`func (g *Game) playerTouchCoordinates(x, y int) (int, int, bool)`
    pub(crate) fn player_touch_coordinates(&self, x: i64, y: i64) -> (i64, i64, bool) {
        let x = x - self.player_viewport_left;
        let block = &self.player_overlay_block;
        // Go `image.Pt(x, y).In(r)`：Min ≤ p < Max（空矩形不含任何點）。
        let in_block = block.min.x <= x && x < block.max.x && block.min.y <= y && y < block.max.y;
        // Go `x < 0 || x >= logicalW || y < top || y >= top+logicalH`。
        if !(0..LOGICAL_W).contains(&x)
            || !(self.player_viewport_top..self.player_viewport_top + LOGICAL_H).contains(&y)
            || self.player_toolbar_touch_blocked && y < PLAYER_TOOLBAR_HEIGHT
            || in_block
        {
            return (0, 0, false);
        }
        (x, y - self.player_viewport_top, true)
    }
}

#[cfg(test)]
#[path = "player_toolbar/player_toolbar_test.rs"]
mod player_toolbar_test;
