//! Go 來源：`remake/cmd/fd2/modern_hd_full_reward.go`（376 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_full_reward.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct FullRewardPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const FULL_REWARD_OPENING: FullRewardPhase = FullRewardPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const FULL_REWARD_BLANK_CLOSING: FullRewardPhase = FullRewardPhase(3);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const FULL_REWARD_CANCEL_OPENING: FullRewardPhase = FullRewardPhase(5);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const FULL_REWARD_CANCEL_CLOSING: FullRewardPhase = FullRewardPhase(6);

/// Go：`modern_hd_full_reward.go:27`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ModernHDGlyphWrite {
    pub(crate) start: i64,
    pub(crate) clear: Rectangle,
    pub(crate) background: bool,
}

/// Go：`modern_hd_full_reward.go:33`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Clone, Default)]
pub(crate) struct ModernHDFullReward {
    pub(crate) scene: Option<ModernHDCompoundMapScene>,
    pub(crate) world: Option<ModernHDMapFrame>,
    pub(crate) theme: Option<ModernStoryPortraitSet>,
    pub(crate) ui: Option<NativePreparationUIAssets>,
    pub(crate) question: Vec<u8>,
    pub(crate) canceled: Vec<Vec<u8>>,
    pub(crate) seal: [u8; 32],
    pub(crate) kind: FullRewardPhase,
    pub(crate) chrome: Option<ModernHDDialogueChrome>,
    pub(crate) chrome_err: Option<GameError>,
    pub(crate) confirmation: Option<ModernHDConfirmation>,
    pub(crate) confirmation_err: Option<GameError>,
}
