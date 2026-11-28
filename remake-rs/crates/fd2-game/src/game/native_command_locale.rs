//! Go 來源：`remake/cmd/fd2/native_command_locale.go`（52 行）。
//!
//! Go 檔的 2 個函式都已移植。

use super::*;

/// Go：`native_command_locale.go:11`。Go 另有 `localizedNativeCommandNameWidth = 140`，只有商店語系測試讀，Rust 不設。
pub(crate) const LOCALIZED_NATIVE_COMMAND_DRAW_WIDTH: f64 = 138.0;
pub(crate) const LOCALIZED_NATIVE_COMMAND_MIN_SCALE: f64 = 0.6;

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

impl Game {
    /// 指令名稱以 1.0 倍起、每次縮 0.02 直到放進 138 像素寬的格子；縮到 0.6 倍以下仍放不進就回錯。
    ///
    /// Go：`native_command_locale.go:36`，`func (g *Game) localizedNativeCommandLabel(commandID int) (string, float64, error)`
    pub(crate) fn localized_native_command_label(&mut self, command_id: i64) -> Result<(String, f64)> {
        let (Some(entities), true) = (self.locale_entities.as_ref(), self.font.is_some()) else {
            return Err(game_err!("localized native command UI is unavailable"));
        };
        let name = entities.command_name(command_id)?;
        let font = self.font.as_mut().unwrap();
        let mut scale = 1.0;
        while scale >= LOCALIZED_NATIVE_COMMAND_MIN_SCALE
            && font.width(&name, scale) > LOCALIZED_NATIVE_COMMAND_DRAW_WIDTH
        {
            scale -= 0.02;
        }
        if scale < LOCALIZED_NATIVE_COMMAND_MIN_SCALE {
            return Err(game_err!(
                "localized command {command_id} {} exceeds native grid cell",
                battle::go_quote(&name)
            ));
        }
        Ok((name, scale))
    }
}
