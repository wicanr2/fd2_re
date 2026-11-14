//! Go 來源：`remake/cmd/fd2/battle_protection.go`（39 行）。
//!
//! 骨架：Go 檔的 1 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// protect 保存作者使用的來源姓名；原生角色不必把姓名複製到執行期 record。
    ///
    /// Go：`battle_protection.go:6`，`func (g *Game) protectedBattleUnitAlive(protect string) (bool, error)`
    pub(crate) fn protected_battle_unit_alive(&mut self, protect: &str) -> Result<bool> {
        todo!("battle_protection.go:protectedBattleUnitAlive")
    }
}
