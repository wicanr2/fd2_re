//! Go 來源：`remake/cmd/fd2/native_battle_panel_locale.go`（135 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

pub(crate) const LOCALIZED_BATTLE_PANEL_NAME_WIDTH: i64 = 120;

pub(crate) const LOCALIZED_BATTLE_PANEL_NAME_HEIGHT: i64 = 16;

pub(crate) const LOCALIZED_BATTLE_PANEL_NAME_SCALE: f64 = 0.65;

impl Game {
    /// Go：`native_battle_panel_locale.go:18`，`func (g *Game) renderLocalizedNativeBattlePanel( assets battle.NativeItemPanelDataAssets, record, dst []byte, unitIndex, rawChapter int, ) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn render_localized_native_battle_panel(
        &mut self,
        assets: &battle::NativeItemPanelDataAssets,
        record: &[u8],
        dst: &mut [u8],
        unit_index: i64,
        raw_chapter: i64,
    ) -> Result<()> {
        if record.len() <= 8 {
            return Err(game_err!("localized native battle panel inputs are invalid"));
        }
        if self.locale_id == "zh-Hant" {
            return Ok(battle::render_native_battle_panel(assets, record, dst, unit_index, raw_chapter)?);
        }
        let (Some(entities), Some(_)) = (self.locale_entities.as_ref(), self.font.as_ref()) else {
            return Err(game_err!("localized native battle panel catalog or font is unavailable"));
        };
        let name = entities.battle_name(i64::from(record[8]))?;
        if self.font.as_mut().unwrap().width(&name, LOCALIZED_BATTLE_PANEL_NAME_SCALE)
            > LOCALIZED_BATTLE_PANEL_NAME_WIDTH as f64
        {
            return Err(game_err!(
                "localized battle name {} {} exceeds panel rectangle",
                record[8],
                battle::go_quote(&name)
            ));
        }
        let mut staged = dst.to_vec();
        battle::render_native_battle_panel_without_name(assets, record, &mut staged, unit_index, raw_chapter)?;
        let (x, y) = battle::native_battle_panel_origin(record, unit_index, raw_chapter)?;
        draw_indexed_localized_text(
            &mut staged,
            self.font.as_mut(),
            &name,
            x + 5,
            y + 4,
            LOCALIZED_BATTLE_PANEL_NAME_WIDTH,
            LOCALIZED_BATTLE_PANEL_NAME_HEIGHT,
            LOCALIZED_BATTLE_PANEL_NAME_SCALE,
            0xcd,
            0x4c,
        )?;
        dst.copy_from_slice(&staged);
        Ok(())
    }

    /// 0x276EC 一族的戰鬥底圖：BG.DAT 層在 (0,50)、狀態框、TAI 平台在 (164,157)，再疊呼叫端給的 FIGANI 畫格。
    /// 收集 HD 狀態框文字時先寫進暫存副本，整張底圖成功後才交回呼叫端。
    ///
    /// Go：`native_battle_panel_locale.go:103`，`func (g *Game) localizedNativeCommand24CapturedBase(background fdother.Frame, panelAssets battle.NativeItemPanelDataAssets, panelRecord []byte, unitIndex, rawChapter int, platform *fdother.Frame, panelText *[]*modernBattlePanelText, overlays ...figani.Frame) ([]byte, error)`
    #[allow(clippy::too_many_arguments)]
    pub(crate) fn localized_native_command24_captured_base(
        &mut self,
        background: &fdother::Frame,
        panel_assets: &battle::NativeItemPanelDataAssets,
        panel_record: &[u8],
        unit_index: i64,
        raw_chapter: i64,
        platform: Option<&fdother::Frame>,
        panel_text: Option<&mut Vec<ModernBattlePanelText>>,
        overlays: &[figani::Frame],
    ) -> Result<Vec<u8>> {
        let mut staged = panel_text.as_deref().cloned();
        let mut base = vec![0u8; 320 * 200];
        let mut background = background.clone();
        (background.x, background.y) = (0, 50);
        background.blit(&mut base, 320, -1)?;
        self.render_battle_panel_text(panel_assets, panel_record, &mut base, unit_index, raw_chapter, staged.as_mut())?;
        if let Some(platform) = platform {
            let mut frame = platform.clone();
            (frame.x, frame.y) = (164, 157);
            frame.blit(&mut base, 320, -1)?;
        }
        for overlay in overlays {
            overlay.blit_at(&mut base, 320)?;
        }
        if let (Some(panel_text), Some(staged)) = (panel_text, staged) {
            *panel_text = staged;
        }
        Ok(base)
    }
}
