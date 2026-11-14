//! Go 來源：`remake/internal/campaign/menu_state.go`（49 行）。

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
        if self.count < 0 {
            self.count = 0;
        }
        if self.selection < 0 {
            self.selection = 0;
        }
        if self.count == 0 {
            self.selection = 0;
        } else if self.selection >= self.count {
            self.selection = self.count - 1;
        }
        match event {
            MENU_UP => {
                if self.selection > 0 {
                    self.selection -= 1;
                }
            }
            MENU_DOWN => {
                if self.selection + 1 < self.count {
                    self.selection += 1;
                }
            }
            MENU_CONFIRM if self.count > 0 => return (self.selection, true),
            _ => {}
        }
        (self.selection, false)
    }
}

// 測試：Go `remake/internal/campaign/menu_state_test.go` 逐條移植。
#[cfg(test)]
mod menu_state_test;
