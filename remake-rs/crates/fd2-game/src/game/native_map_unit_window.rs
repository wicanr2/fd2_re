//! Go 來源：`remake/cmd/fd2/native_map_unit_window.go`（126 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// nativeMapSelectionOverlay 決定這一幀比 0x11CAC 多畫什麼：
    ///   - 玩家選了自己還沒行動的單位、正在挑目的地（移動範圍亮著、還沒移動、指令環
    ///     沒開）：0x18B84 的重繪，多一個單位資訊視窗（0x18C6D）；
    ///   - 指令環開著：0x1741C 把四張 FDOTHER#2 圖示貼在游標四周（穩態用 0x179D5 的
    ///     停留位移，開合中用各自的動畫位移），被選方向依 [0x53C13] 閃爍。
    ///
    /// 回傳 nil 表示照 0x11CAC。
    ///
    /// Go：`native_map_unit_window.go:26`，`func (g *Game) nativeMapSelectionOverlay(state *battle.State) (func([]byte, int, int) error, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_map_selection_overlay(
        &mut self,
        state: Option<&mut battle::State>,
    ) -> Result<Option<Box<dyn FnMut(&mut Game, Vec<u8>, i64, i64) -> Result<()> + Send>>> {
        todo!("native_map_unit_window.go:nativeMapSelectionOverlay")
    }

    /// nativeMapActionOverlay 把指令環貼進工作緩衝，與 drawNativeActionOverlay 同一組
    /// 位移、格子索引與閃爍規則，只是目的地換成 indexed 畫面。
    ///
    /// Go：`native_map_unit_window.go:58`，`func (g *Game) nativeMapActionOverlay(state *battle.State) func([]byte, int, int) error`
    pub(crate) fn native_map_action_overlay(
        &mut self,
        state: &mut battle::State,
    ) -> Option<Box<dyn FnMut(&mut Game, Vec<u8>, i64, i64) -> Result<()> + Send>> {
        todo!("native_map_unit_window.go:nativeMapActionOverlay")
    }
}
