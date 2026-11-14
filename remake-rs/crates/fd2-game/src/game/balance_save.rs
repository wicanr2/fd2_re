//! Go 來源：`remake/cmd/fd2/balance_save.go`（174 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 5 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`balance_save.go:13`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct BalanceSavedState {
    /// Go 標籤：`json:"options,omitempty"`
    pub(crate) options: Option<battle::BalanceOptions>,
    /// Go 標籤：`json:"enemy_bases,omitempty"`
    pub(crate) enemy_bases: BTreeMap<i64, battle::BalanceStats>,
}

/// Go：`balance_save.go:17`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct BalanceSidecar {
    /// Go 標籤：`json:"version"`
    pub(crate) version: i64,
    /// Go 標籤：`json:"entries"`
    pub(crate) entries: BTreeMap<String, BalanceSavedState>,
}

/// Go：`balance_save.go:22`，`func balanceSaveKey(plain []byte, slot int) (string, error)`
pub(crate) fn balance_save_key(plain: &[u8], slot: i64) -> Result<String> {
    todo!("balance_save.go:balanceSaveKey")
}

/// Go：`balance_save.go:36`，`func readBalanceSidecar(path string) (balanceSidecar, error)`
pub(crate) fn read_balance_sidecar(path: &str) -> Result<BalanceSidecar> {
    todo!("balance_save.go:readBalanceSidecar")
}

/// Go：`balance_save.go:79`，`func loadBalanceSavedState(path string, plain []byte, slot int) (balanceSavedState, error)`
pub(crate) fn load_balance_saved_state(path: &str, plain: &[u8], slot: i64) -> Result<BalanceSavedState> {
    todo!("balance_save.go:loadBalanceSavedState")
}

impl Game {
    /// Preserve both old and new fingerprints before replacing the native SAV.
    /// A crash at either rename still pairs a save only with its own options.
    ///
    /// Go：`balance_save.go:93`，`func (g *Game) replaceNativeSaveWithBalance(path string, stored []byte, slot int) error`
    pub(crate) fn replace_native_save_with_balance(&mut self, path: &str, stored: &[u8], slot: i64) -> Result<()> {
        todo!("balance_save.go:replaceNativeSaveWithBalance")
    }
}

/// Go：`balance_save.go:155`，`func fileExistsBalance(path string) bool`
pub(crate) fn file_exists_balance(path: &str) -> bool {
    todo!("balance_save.go:fileExistsBalance")
}
