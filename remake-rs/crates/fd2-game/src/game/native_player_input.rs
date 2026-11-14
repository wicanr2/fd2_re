//! Go 來源：`remake/cmd/fd2/native_player_input.go`（237 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 4 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// Go：`native_player_input.go:38`，`func (g *Game) beginNativePlayerFocus(u *battle.Unit) bool`
    pub(crate) fn begin_native_player_focus(&mut self, u: Option<UnitIdx>) -> bool {
        todo!("native_player_input.go:beginNativePlayerFocus")
    }

    /// Go：`native_player_input.go:53`，`func (g *Game) stepNativePlayerFocus()`
    pub(crate) fn step_native_player_focus(&mut self) {
        todo!("native_player_input.go:stepNativePlayerFocus")
    }

    /// inspectNativePlayerUnit 的回傳值表示已消費確認鍵，包括原版拒絕的特殊角色。
    ///
    /// Go：`native_player_input.go:99`，`func (g *Game) inspectNativePlayerUnit(u *battle.Unit, force bool) (bool, error)`
    pub(crate) fn inspect_native_player_unit(&mut self, u: UnitIdx, force: bool) -> Result<bool> {
        todo!("native_player_input.go:inspectNativePlayerUnit")
    }
}

/// Go：`native_player_input.go:114`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativePlayerStatusState {
    pub(crate) data: Option<NativeUnitStatusData>,
    pub(crate) text: Option<NativeStatusTransitionText>,
    pub(crate) source: Vec<u8>,
    pub(crate) status: Vec<u8>,
    pub(crate) commands: Vec<u8>,
    pub(crate) steady: Vec<u8>,
    pub(crate) frames: Vec<Vec<u8>>,
    pub(crate) frame: i64,
    pub(crate) phase: String,
    pub(crate) drawn: bool,
}

impl Game {
    /// Go：`native_player_input.go:152`，`func (g *Game) stepNativePlayerStatus()`
    pub(crate) fn step_native_player_status(&mut self) {
        todo!("native_player_input.go:stepNativePlayerStatus")
    }
}
