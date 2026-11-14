//! Go 來源：`remake/cmd/fd2/native_event_dialogue.go`（96 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 4 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativeBattleDialogueAdvanceInput 回報這次鍵盤事件是否推進戰鬥對白。
/// 原版戰鬥事件與回合起手對白的 dosgolem 三臂收據都證實 ESC 與 Enter 同義；
/// Space 沿用既有桌面便利鍵，呼叫端把 Enter／Space 合併成 enterOrSpace。
///
/// Go：`native_event_dialogue.go:14`，`func nativeBattleDialogueAdvanceInput(enterOrSpace, escape bool) bool`
pub(crate) fn native_battle_dialogue_advance_input(enter_or_space: bool, escape: bool) -> bool {
    enter_or_space || escape
}

impl Game {
    /// Go：`native_event_dialogue.go:18`，`func (g *Game) startNativeEventDialogue(action battle.Action) error`
    pub(crate) fn start_native_event_dialogue(&mut self, action: &battle::Action) -> Result<()> {
        let Some(r#ref) = action.native_dialogue_ref.as_ref().filter(|r| {
            !action.native_source.is_empty()
                && action.native_text_index == Some(r.string_index)
                && r.scene_index >= 0
                && r.line >= 0
        }) else {
            return Err(GameError::new("戰鬥對話缺少一致的來源"));
        };
        let layout = campaign::NativeDialogueLayout {
            source_dat: r#ref.source_dat.clone(),
            string_index: r#ref.string_index,
            utterance: r#ref.utterance,
            control: r#ref.control.clone(),
            operand: r#ref.operand,
            pages: r#ref.pages.clone(),
            glyph_pages: r#ref.glyph_pages.clone(),
            glyph_ids: r#ref.glyph_ids.clone(),
        };
        layout.validate()?;
        let lines = load_story_script_with_identity_at(&r#ref.script, "", Some(&r#ref.scene_index))?;
        if r#ref.line >= lines.len() as i64 {
            return Err(game_err!("戰鬥對話來源句 {} 不存在", r#ref.line));
        }
        let upper = r#ref.control == "FFED" || r#ref.control == "FFEF";
        if self.cutscene_log {
            // FD2_CUTSCENE_LOG：印回合／死亡事件的每一句，對原版 0x15F84 呼叫序列比對
            let event = action.native_event_id.unwrap_or(-1);
            eprintln!(
                "[cutscene] event dialogue source={} event={} text={} scene={} line={}",
                action.native_source, event, r#ref.string_index, r#ref.scene_index, r#ref.line
            );
        }
        let line_source = &lines[r#ref.line as usize];
        if !self.native_battle_dialogue_available() {
            // 這一章的戰場還沒有原生視圖與 HUD 狀態，它自己的回合事件也用一般對白框；
            // 死亡台詞照同一種呈現，文字與說話者取自同一行故事腳本，不另造內容。
            let line = self.resolve_campaign_dialog_line(line_source, Some(upper), None)?;
            self.dialog = vec![line];
            (self.dlg_page, self.dlg_scroll_t, self.dlg_shown, self.dlg_phase, self.dlg_t) = (0, 0, DLG_NONE, 0, 0);
            return Ok(());
        }
        let line = self.resolve_campaign_dialog_line(line_source, Some(upper), Some(&layout))?;
        self.dialog = vec![line];
        (self.dlg_page, self.dlg_scroll_t, self.dlg_shown, self.dlg_phase, self.dlg_t) = (0, 0, DLG_NONE, 0, 0);
        self.start_native_dialogue_frames()
    }

    /// handleBattleEventDialogueInput 與故事對話共用逐字／收框閘門，續行仍歸戰鬥事件。
    ///
    /// Go：`native_event_dialogue.go:69`，`func (g *Game) handleBattleEventDialogueInput(advance bool)`
    pub(crate) fn handle_battle_event_dialogue_input(&mut self, advance: bool) {
        if !advance || self.dialog.is_empty() || self.native_dialogue_closing_live {
            return;
        }
        let current = self.dialog.last().unwrap();
        if current.native_dialogue.is_some() && self.dlg_page + 1 >= dlg_page_count(current) {
            if !self.native_story_dialogue_at_input_wait() {
                return;
            }
            if !self.begin_native_story_dialogue_closing() {
                self.finish_battle_event_with_error("原生對話收框不可用");
            }
            return;
        }
        if self.dlg_advance() && self.dialog.is_empty() {
            self.advance_battle_event();
        }
    }

    /// nativeBattleDialogueAvailable 是原生戰場對白（開框、聚焦說話者、逐字、收框）需要的
    /// 狀態：原生視圖、HUD 輸入與分離素材都在。
    ///
    /// Go：`native_event_dialogue.go:90`，`func (g *Game) nativeBattleDialogueAvailable() bool`
    pub(crate) fn native_battle_dialogue_available(&mut self) -> bool {
        if !self.st.as_ref().is_some_and(|st| st.has_native_map_view_state)
            || !native_map_assets_available(self.native_map_assets.as_ref())
        {
            return false;
        }
        let (_, ok) = self.native_map_hud_input();
        ok
    }
}

#[cfg(test)]
mod native_event_dialogue_test;
