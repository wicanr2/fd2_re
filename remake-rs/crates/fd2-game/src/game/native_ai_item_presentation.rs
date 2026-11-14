//! Go 來源：`remake/cmd/fd2/native_ai_item_presentation.go`（646 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_ai_item_presentation.go:14`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeAIItemPresentationJob {
    pub(crate) transaction: Option<NativeAIItemTransaction>,
    pub(crate) frames: Vec<NativeCompoundPresentedFrame>,
    pub(crate) frame: i64,
    pub(crate) repeat: i64,
    pub(crate) publish_at: i64,
    pub(crate) published: bool,
    pub(crate) drawn: bool,
    pub(crate) holding: bool,
    pub(crate) hold: i64,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) post_transaction_work: Vec<u8>,
    pub(crate) post_vga: Vec<u8>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

impl Game {
    /// Go：`native_ai_item_presentation.go:531`，`func (g *Game) cancelNativeAIItemPresentation()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn cancel_native_ai_item_presentation(&mut self) {
        todo!("native_ai_item_presentation.go:cancelNativeAIItemPresentation")
    }

    /// Go：`native_ai_item_presentation.go:573`，`func (g *Game) stepNativeAIItemPresentation()`
    pub(crate) fn step_native_ai_item_presentation(&mut self) {
        todo!("native_ai_item_presentation.go:stepNativeAIItemPresentation")
    }
}
