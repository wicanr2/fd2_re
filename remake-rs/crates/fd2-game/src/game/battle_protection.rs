//! Go 來源：`remake/cmd/fd2/battle_protection.go`（39 行）。
//!
//! 骨架：Go 檔的 1 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// protect 保存作者使用的來源姓名；原生角色不必把姓名複製到執行期 record。
    ///
    /// Go：`battle_protection.go:6`，`func (g *Game) protectedBattleUnitAlive(protect string) (bool, error)`
    pub(crate) fn protected_battle_unit_alive(&mut self, protect: &str) -> Result<bool> {
        if protect.is_empty() {
            return Ok(true);
        }
        let count = self.st.as_ref().expect("Go：g.st 為 nil 時解參考 panic").units.len();
        for i in 0..count {
            let u = &self.st.as_ref().unwrap().units[i];
            if !u.alive() {
                continue;
            }
            let mut name = u.name.clone();
            let (has_byte8, byte8, has_identity, identity) =
                (u.has_native_record_byte8, u.native_record_byte8, u.has_native_identity, u.native_identity);
            if has_byte8 || has_identity {
                if self.source_battle_entities.is_none() {
                    let catalog = localization::load_official_entities(&asset_path("assets/locales"), "zh-Hant")?;
                    self.source_battle_entities = Some(catalog);
                }
                let entities = self.source_battle_entities.as_ref().unwrap();
                name = if has_byte8 {
                    entities.battle_name(i64::from(byte8))?
                } else {
                    // 舊版來源化角色保存 persistent identity，未另存 runtime +8。
                    entities.character_name(identity)?
                };
            }
            if name == protect {
                return Ok(true);
            }
        }
        Ok(false)
    }
}
