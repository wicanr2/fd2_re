//! Go `remake/cmd/fd2/chapter_parity_hp_lock_test.go`：`parityUpdateGame` 與修改路徑的鎖血政策。
//! 第一章沒有 `hp_lock` 動作（`parityHPPolicies[g]` 是 nil），鎖血政策在 R8 移植。

use super::*;

/// Go `chapter_parity_hp_lock_test.go:66`：`parityUpdateGame`。沒有鎖血政策時就是 `g.Update()`；
/// Rust 在 `update` 之前把虛擬時鐘推進一個邏輯幀（Ebiten 主迴圈的一次 `Update`）。
pub(crate) fn parity_update_game(d: &mut Driver) -> Result<()> {
    d.g.advance_virtual_frame(&mut d.clock);
    d.g.update()
}

/// Go `chapter_parity_hp_lock_test.go:126`：`parityPreservesEnemy`。沒有鎖血政策時 Go 要求 `indices` 為空、
/// 回傳 false；有政策的章在 R8 移植。
pub(crate) fn parity_preserves_enemy(indices: &[i64]) -> bool {
    if !indices.is_empty() {
        todo!("R8: chapter_parity_hp_lock_test.go:126 parityPreservesEnemy（鎖血政策下保留敵方）")
    }
    false
}
