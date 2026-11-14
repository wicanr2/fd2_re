//! Go 來源：`remake/internal/campaign/menu_state.go`（49 行）。

use super::*;

/// MenuEvent is the input vocabulary shared by editable choice/town nodes.
/// It intentionally describes navigation only; option labels and their
/// transition targets remain campaign data.
///
/// Go：`menu_state.go:6`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct MenuEvent(pub u8);

/// Go 原式：`iota`
pub const MENU_TICK: MenuEvent = MenuEvent(0);

pub const MENU_UP: MenuEvent = MenuEvent(1);

pub const MENU_DOWN: MenuEvent = MenuEvent(2);

pub const MENU_CONFIRM: MenuEvent = MenuEvent(3);

/// MenuState is the bounded campaign-node selector used by the remake's
/// choice/town hub. It has no native service names or scene side effects.
///
/// Go：`menu_state.go:17`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct MenuState {
    pub selection: i64,
    pub count: i64,
}

impl MenuState {
    /// Go：`menu_state.go:22`，`func (s *MenuState) Step(event MenuEvent) (selected int, confirm bool)`
    /// Go 具名回傳值依序為：selected、confirm。
    pub fn step(&mut self, event: MenuEvent) -> (i64, bool) {
        todo!("campaign/menu_state.go:MenuState.Step")
    }
}
