//! Go 來源：`remake/cmd/fd2/player_balance.go`（629 行）。
//!
//! 骨架：Go 檔的 17 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`player_balance.go:20`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct BalanceEditor {
    pub(crate) names: BTreeMap<i64, String>,
    pub(crate) open: bool,
    pub(crate) ack: i64,
    pub(crate) tab: i64,
    pub(crate) person: i64,
    pub(crate) command: i64,
    pub(crate) field: i64,
    pub(crate) all: bool,
    pub(crate) draft: Option<battle::BalanceOptions>,
    pub(crate) values: [i64; 5],
    pub(crate) input: String,
    pub(crate) message: String,
}

impl Game {
    /// Go：`player_balance.go:252`，`func (g *Game) syncBalanceState()`
    pub(crate) fn sync_balance_state(&mut self) {
        todo!("player_balance.go:syncBalanceState")
    }
}
