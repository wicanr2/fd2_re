//! Go 來源：`remake/cmd/fd2/native_preparation_input.go`（194 行）。
//!
//! 骨架：Go 檔的 1 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativePreparationInput 是鍵盤與決定性玩家路徑共用的整備輸入事件。
/// 它只描述單次按鍵邊界，不直接攜帶節點、名冊或交易結果。
///
/// Go：`native_preparation_input.go:7`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativePreparationInput {
    pub(crate) left: bool,
    pub(crate) right: bool,
    pub(crate) up: bool,
    pub(crate) down: bool,
    pub(crate) enter: bool,
    pub(crate) escape: bool,
}

impl Game {
    /// handleNativePreparationInput 擁有 preparation prompt、選人與最終確認三態。
    /// 所有 campaign mutation 都延後到既有 indexed closing／restore continuation；
    /// 深層畫面不可用時則沿既有失敗即關閉後備邊界同步執行相同 continuation。
    ///
    /// Go：`native_preparation_input.go:17`，`func (g *Game) handleNativePreparationInput(input nativePreparationInput) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn handle_native_preparation_input(&mut self, input: NativePreparationInput) -> bool {
        todo!("native_preparation_input.go:handleNativePreparationInput")
    }
}
