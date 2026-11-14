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
    todo!("native_event_dialogue.go:nativeBattleDialogueAdvanceInput")
}

impl Game {
    /// Go：`native_event_dialogue.go:18`，`func (g *Game) startNativeEventDialogue(action battle.Action) error`
    pub(crate) fn start_native_event_dialogue(&mut self, action: &battle::Action) -> Result<()> {
        todo!("native_event_dialogue.go:startNativeEventDialogue")
    }

    /// handleBattleEventDialogueInput 與故事對話共用逐字／收框閘門，續行仍歸戰鬥事件。
    ///
    /// Go：`native_event_dialogue.go:69`，`func (g *Game) handleBattleEventDialogueInput(advance bool)`
    pub(crate) fn handle_battle_event_dialogue_input(&mut self, advance: bool) {
        todo!("native_event_dialogue.go:handleBattleEventDialogueInput")
    }

    /// nativeBattleDialogueAvailable 是原生戰場對白（開框、聚焦說話者、逐字、收框）需要的
    /// 狀態：原生視圖、HUD 輸入與分離素材都在。
    ///
    /// Go：`native_event_dialogue.go:90`，`func (g *Game) nativeBattleDialogueAvailable() bool`
    pub(crate) fn native_battle_dialogue_available(&mut self) -> bool {
        todo!("native_event_dialogue.go:nativeBattleDialogueAvailable")
    }
}
