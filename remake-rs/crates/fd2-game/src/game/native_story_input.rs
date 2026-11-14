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
        self.enter || self.escape
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
        let Some(n) = n else {
            return false;
        };
        if self.camp.is_none() {
            return false;
        }
        let advance = input.advance();
        match n.r#type.as_str() {
            "story" => {
                if advance
                    && self.fade.is_none()
                    && self.story_walks.is_empty()
                    && self.dlg_advance()
                    && self.dialog.is_empty()
                {
                    self.advance_story_node(n);
                }
                true
            }
            "cutscene" => {
                if self.approximate_postbattle {
                    if advance {
                        todo!("R8: main.go:continueApproximatePostbattle（第一章沒有執行到）")
                    }
                    return true;
                }
                if advance && !self.dialog.is_empty() && self.native_dialogue_closing_live {
                    return true;
                }
                if advance && let Some(current) = self.dialog.last() {
                    let page = self.dlg_page;
                    if current.native_dialogue.is_some()
                        && page + 1 >= dlg_page_count(current)
                        && page >= 0
                        && page < self.native_dialogue_progressive.len() as i64
                        && !self.native_dialogue_progressive[page as usize].is_empty()
                        && self.native_dialogue_progress
                            >= self.native_dialogue_progressive[page as usize].len() as i64 - 1
                    {
                        if !self.begin_native_story_dialogue_closing() {
                            self.load_err =
                                String::from("native story dialogue: verified closing frames are unavailable");
                        }
                        return true;
                    }
                    if self.dlg_advance() && self.dialog.is_empty() {
                        self.beat_advance();
                    }
                }
                true
            }
            _ => false,
        }
    }
}

#[cfg(test)]
mod native_story_input_test;
