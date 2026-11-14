//! Go 來源：`remake/cmd/fd2/title_menu_state.go`（90 行）。
//!
//! 骨架：Go 檔的 2 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// TitleMenuAction is the observable result of a completed native-style title
/// menu confirmation. Keeping this transition pure gives the UI trace a stable
/// contract without pretending the JSON save format is FD2.SAV-compatible.
///
/// Go：`title_menu_state.go:6`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct TitleMenuAction(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const TITLE_MENU_NO_ACTION: TitleMenuAction = TitleMenuAction(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const TITLE_MENU_START: TitleMenuAction = TitleMenuAction(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const TITLE_MENU_LOAD_SLOTS: TitleMenuAction = TitleMenuAction(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const TITLE_MENU_CONTINUE: TitleMenuAction = TitleMenuAction(3);

/// Go：`title_menu_state.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct TitleMenuEvent(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const TITLE_MENU_TICK: TitleMenuEvent = TitleMenuEvent(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const TITLE_MENU_UP: TitleMenuEvent = TitleMenuEvent(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const TITLE_MENU_DOWN: TitleMenuEvent = TitleMenuEvent(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const TITLE_MENU_CONFIRM: TitleMenuEvent = TitleMenuEvent(3);

/// TitleMenuState models the three-entry main menu and its native confirmation
/// flash. Selection wraps; confirmation resolves only after FlashTicks reaches
/// zero, matching titleUpdate's visible confirmation delay.
///
/// Go：`title_menu_state.go:27`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct TitleMenuState {
    pub(crate) selection: i64,
    pub(crate) flash_ticks: i64,
}

impl TitleMenuState {
    /// Go：`title_menu_state.go:32`，`func (s *TitleMenuState) Step(event TitleMenuEvent) TitleMenuAction`
    pub(crate) fn step(&mut self, event: TitleMenuEvent) -> TitleMenuAction {
        todo!("title_menu_state.go:TitleMenuState.Step")
    }
}

/// Go：`title_menu_state.go:57`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct TitleSlotEvent(pub(crate) u8);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const TITLE_SLOT_UP: TitleSlotEvent = TitleSlotEvent(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const TITLE_SLOT_DOWN: TitleSlotEvent = TitleSlotEvent(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const TITLE_SLOT_CONFIRM: TitleSlotEvent = TitleSlotEvent(3);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const TITLE_SLOT_CANCEL: TitleSlotEvent = TitleSlotEvent(4);
