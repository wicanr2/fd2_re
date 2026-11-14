//! Go 來源：`remake/cmd/fd2/native_command_locale.go`（52 行）。
//!
//! 骨架：Go 檔的 2 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command_locale.go:17`，`func localizedSpellBook(entities *localization.EntityCatalog, source []battle.Spell) ([]battle.Spell, error)`
pub(crate) fn localized_spell_book(
    entities: Option<&localization::EntityCatalog>,
    source: &[battle::Spell],
) -> Result<Vec<battle::Spell>> {
    todo!("native_command_locale.go:localizedSpellBook")
}
