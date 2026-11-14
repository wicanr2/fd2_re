//! Go 來源：`remake/cmd/fd2/native_story_input.go`（66 行）。
//!
//! 骨架：Go 檔的 2 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativeStoryInput 是鍵盤與決定性玩家路徑共用的故事輸入事件。
/// 它只保存單次推進鍵邊界，不提供直接清除對白或改寫節點的捷徑。
///
/// Go：`native_story_input.go:7`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativeStoryInput {
    pub(crate) enter: bool,
    pub(crate) escape: bool,
}

impl NativeStoryInput {
    /// advance 回報這次事件是否構成一次原版故事推進。
    ///
    /// 原版開場的故事等待對 ESC 與 Enter 反應完全相同：同一份控制序列只換按鍵，
    /// 19 個控制邊界的畫面逐格 bit-identical，而完全不送鍵時畫面停住不動
    /// （收據 docs/data/ui-traces/fd2-opening-esc-20260909.json）。所以 ESC 在
    /// 原版不是「跳過整段」的捷徑，而是與 Enter 同義的推進鍵。
    ///
    /// Go：`native_story_input.go:18`，`func (i nativeStoryInput) advance() bool`
    /// Go 值接收者：本體對接收者的改寫只影響副本。
    pub(crate) fn advance(&self) -> bool {
        todo!("native_story_input.go:advance")
    }
}

impl Game {
    /// handleNativeStoryInput 擁有一般 story 與處理器 cutscene 的確認鍵消費。
    /// 原生對話必須完成逐字與已證實收框後，才可交還既有 beat continuation。
    ///
    /// Go：`native_story_input.go:24`，`func (g *Game) handleNativeStoryInput(n *campaign.Node, input nativeStoryInput) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn handle_native_story_input(
        &mut self,
        n: Option<&mut campaign::Node>,
        input: NativeStoryInput,
    ) -> bool {
        todo!("native_story_input.go:handleNativeStoryInput")
    }
}
