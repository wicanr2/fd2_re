//! Go 來源：`remake/cmd/fd2/modern_hd_preparation_transition.go`（156 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_preparation_transition.go:9`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativePreparationQuestionKind(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_PREPARATION_RECORD_QUESTION: NativePreparationQuestionKind = NativePreparationQuestionKind(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_PREPARATION_DEPARTURE_QUESTION: NativePreparationQuestionKind =
    NativePreparationQuestionKind(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_PREPARATION_CONFIRM_QUESTION: NativePreparationQuestionKind = NativePreparationQuestionKind(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_PREPARATION_REQUIRED_QUESTION: NativePreparationQuestionKind = NativePreparationQuestionKind(3);

/// This recipe belongs to the job, independently of the next selection or
/// confirmation. Indexed animation buffers remain the faithful source.
///
/// Go：`modern_hd_preparation_transition.go:20`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Clone, Default)]
pub(crate) struct NativePreparationTransitionText {
    pub(crate) source: Vec<u8>,
    pub(crate) assets: NativePreparationUIAssets,
    pub(crate) kind: NativePreparationQuestionKind,
    pub(crate) identity: i64,
    pub(crate) start: i64,
    pub(crate) closing: bool,
    pub(crate) town: Option<NativeTownFrameSource>,
}

impl Game {
    /// Go：`modern_hd_preparation_transition.go:30`，`func (g *Game) snapshotNativePreparationTransition(source []byte, confirmation bool, start int, closing bool) *nativePreparationTransitionText`
    pub(crate) fn snapshot_native_preparation_transition(
        &self,
        source: &mut [u8],
        confirmation: bool,
        start: i64,
        closing: bool,
    ) -> Option<NativePreparationTransitionText> {
        todo!("modern_hd_preparation_transition.go:snapshotNativePreparationTransition")
    }
}
