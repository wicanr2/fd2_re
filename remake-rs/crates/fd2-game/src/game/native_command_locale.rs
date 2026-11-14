//! Go 來源：`remake/cmd/fd2/native_command_locale.go`（52 行）。
//!
//! 骨架：Go 檔的 2 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command_locale.go:17`，`func localizedSpellBook(entities *localization.EntityCatalog, source []battle.Spell) ([]battle.Spell, error)`
pub(crate) fn localized_spell_book(
    entities: Option<&localization::EntityCatalog>,
    source: &[battle::Spell],
) -> Result<Vec<battle::Spell>> {
    let Some(entities) = entities.filter(|_| source.len() == 36) else {
        return Err(game_err!("localized command catalog or spell table is unavailable"));
    };
    let mut result = source.to_vec();
    for spell in result.iter_mut() {
        if spell.id == 31 {
            spell.name = String::new();
            continue;
        }
        spell.name = entities.command_name(spell.id)?;
    }
    Ok(result)
}
