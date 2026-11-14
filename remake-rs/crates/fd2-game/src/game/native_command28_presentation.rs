//! Go 來源：`remake/cmd/fd2/native_command28_presentation.go`（413 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command28_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativeCommand28Event {
    pub(crate) prelude: i64,
    pub(crate) effect: i64,
    pub(crate) tick: i64,
    pub(crate) target: i64,
    pub(crate) idle: i64,
    pub(crate) impact: i64,
    pub(crate) mp: bool,
}

/// The shared strike owner retains input, F2, rollback and action completion.
/// This sequence supplies command28's mode0, fixed scenery and eight impacts.
///
/// Go：`native_command28_presentation.go:22`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeCommand28Sequence {
    pub(crate) frames: Vec<NativeCompoundPresentedFrame>,
    pub(crate) events: Vec<NativeCommand28Event>,
    pub(crate) index: i64,
}
