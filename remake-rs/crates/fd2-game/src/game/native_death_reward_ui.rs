//! Go 來源：`remake/cmd/fd2/native_death_reward_ui.go`（258 行）。
//!
//! Go 檔的 5 個函式都已移植；HD 的 `modernHDFullReward` 凍結與綁定在 R6（PORTING.md「D 組：HD 預凍結延後」）。

use super::*;

/// Go：`native_death_reward_ui.go:10`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct PendingNativeDeathReward {
    pub(crate) killer: Option<UnitIdx>,
    pub(crate) item: i64,
}

/// Go：`native_death_reward_ui.go:15`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Clone, Default)]
pub(crate) struct NativeDeathRewardUIState {
    pub(crate) killer: Option<UnitIdx>,
    pub(crate) previous_selection: Option<UnitIdx>,
    pub(crate) portrait: dato::Frame,
    pub(crate) item: i64,
    pub(crate) selected: i64,
    pub(crate) source: Vec<u8>,
    pub(crate) dialogue: Vec<u8>,
    pub(crate) hd: Option<ModernHDFullReward>,
}

/// Go：`native_death_reward_ui.go:24`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativeDeathRewardInput {
    pub(crate) up: bool,
    pub(crate) down: bool,
    pub(crate) left: bool,
    pub(crate) right: bool,
    pub(crate) confirm: bool,
    pub(crate) cancel: bool,
}

impl Game {
    /// runPendingNativeDeathRewards starts the first 0x1AA56 modal only at action
    /// completion, after the attack presentation. It preflights both the indexed
    /// question and the later item panel before exposing any interactive state.
    ///
    /// Go：`native_death_reward_ui.go:32`，`func (g *Game) runPendingNativeDeathRewards(then func()) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    ///
    /// 所有權：Go 讓 `g.nativeDeathRewardUI` 與 `nativeSystemEndTurnUI.deathReward` 指向同一個提示；Rust 在結束回合
    /// 狀態放一份複本。之後只有 `g.native_death_reward_ui` 的 `selected` 會改（`handle_native_death_reward_item_input`），
    /// 結束回合那份只被讀 `killer` 與「是否為死亡獎勵」，HD 的 `kind` 在 R6。
    pub(crate) fn run_pending_native_death_rewards(&mut self, then: Option<Box<dyn FnOnce(&mut Game) + Send>>) -> bool {
        if self.native_death_reward_ui.is_some() || self.pending_native_death_rewards.is_empty() {
            return false;
        }
        let pending = self.pending_native_death_rewards[0];
        let st = self.st.as_ref().expect("Go：pending.killer 指向 g.st.Units");
        let killer = pending.killer.map(|k| &st.units[k]);
        let valid = killer.is_some_and(|k| {
            k.inventory.len() == 8 && battle::validate_native_inventory_projection(Some(k)).is_ok() && k.has_battle_fig
        });
        if !valid
            || pending.item < 0
            || pending.item > 0xff
            || self.native_preparation_ui.is_none()
            || self.native_class_ui.is_none()
        {
            self.load_err = String::from("native death reward: full inventory or dialogue provenance unavailable");
            return true;
        }
        let killer_idx = pending.killer.unwrap();
        let killer = killer.unwrap();
        if killer.native_inventory_flags.iter().any(|&flag| flag & 0x80 != 0) {
            self.load_err = String::from("native death reward: full raw inventory contains an empty slot");
            return true;
        }
        let portraits = match load_native_separated_portrait(killer.battle_fig) {
            Ok(portraits) if portraits.len() > 3 => portraits,
            _ => {
                self.load_err = String::from("native death reward: killer portrait unavailable");
                return true;
            }
        };
        if let Err(err) = self.compose_native_map_frame() {
            self.load_err = format!("native death reward: {}", err.message());
            return true;
        }
        if !self.prepare_native_item_panel(killer_idx) {
            self.load_err = String::from("native death reward: eight-slot item panel unavailable");
            return true;
        }
        let composed = (|| -> Result<_> {
            let ui = self.native_preparation_ui.as_ref().unwrap();
            let source = self.native_map_vga.clone();
            let dialogue =
                campaign::compose_native_preparation_confirmation_dialogue(&source, &ui.dialogue, &portraits[0])?;
            // Go 把 `ui.status.Strings`／`Font` 直接傳下去，nil 時由 `ComposeNativeChurchTextAt` 回錯；Rust 的簽名要求
            // 已載入的文字與字型，缺件在這裡回同一層錯誤。
            let strings = ui.status.strings.as_ref().ok_or_else(|| game_err!("FDTXT strings unavailable"))?;
            let font = ui.status.font.as_ref().ok_or_else(|| game_err!("FDTXT font unavailable"))?;
            let question =
                campaign::compose_native_death_reward_discard_question(&dialogue, &portraits[0], strings, font)?;
            let canceled =
                campaign::native_death_reward_abandon_response_frames(&question, Some(strings), font, pending.item)?;
            Ok((source, dialogue, question, canceled))
        })();
        let (source, dialogue, question, canceled) = match composed {
            Ok(composed) => composed,
            Err(err) => {
                self.clear_native_item_panel();
                self.load_err = format!("native death reward: {}", err.message());
                return true;
            }
        };
        let ui = self.native_preparation_ui.as_ref().unwrap();
        let frames = match campaign::native_preparation_confirmation_opening_frames(
            &source,
            &dialogue,
            &question,
            &ui.choices,
        ) {
            Ok(frames) if frames.len() == 10 => frames,
            _ => {
                self.clear_native_item_panel();
                self.load_err = String::from("native death reward: prompt opening frames unavailable");
                return true;
            }
        };
        // Go 在這裡以 `freezeNativeCommand9MapInput`／`newModernHDCompoundMapScene`／`freezeModernHDDialogueChrome`／
        // `freezeModernHDConfirmation` 凍結 `prompt.hd`（錯誤只存進 `hd`，只有 HD 繪製與綁定讀）；Rust 留到 R6，
        // `hd` 是 `None`（PORTING.md「D 組：HD 預凍結延後」）。HD 介面在 `prepare_native_item_panel` 已停在 R6。
        if self.modern_story_portraits.is_some() {
            todo!("R6: native_death_reward_ui.go:111 prepareModernHDFullRewardWorld（HD 主題）")
        }
        let prompt = NativeDeathRewardUIState {
            killer: Some(killer_idx),
            previous_selection: self.sel,
            portrait: portraits[0].clone(),
            item: pending.item,
            selected: -1,
            source: source.clone(),
            dialogue: dialogue.clone(),
            hd: None,
        };
        self.pending_native_death_rewards.remove(0);
        self.native_death_reward_ui = Some(prompt.clone());
        self.native_death_reward_then = then;
        self.native_system_end_turn_ui = Some(Box::new(NativeSystemEndTurnUIState {
            portraits,
            source,
            dialogue,
            question,
            canceled,
            death_reward: Some(prompt),
            ..Default::default()
        }));
        self.native_system_end_turn_confirm = true;
        self.native_system_end_turn_delay = 0;
        self.ring = false;
        self.msg = String::new();
        self.reset_native_class_ui_pulse();
        self.native_class_ui_job = Some(NativeClassUIJob { frames, ..Default::default() });
        self.bind_native_system_end_turn_hd(FULL_REWARD_OPENING);
        true
    }

    /// 物品選擇器在滿欄丟棄時不分戰鬥使用（`battleUse=false`），Enter 先以複本試換，成功才記下選擇並收合面板。
    ///
    /// Go：`native_death_reward_ui.go:137`，`func (g *Game) handleNativeDeathRewardItemInput(input nativeDeathRewardInput) bool`
    pub(crate) fn handle_native_death_reward_item_input(&mut self, input: NativeDeathRewardInput) -> bool {
        let Some(prompt) = self.native_death_reward_ui.as_ref() else {
            return false;
        };
        let (killer, item) = (prompt.killer, prompt.item);
        if !self.item_open || self.sel != killer || self.native_item_panel.is_none() {
            return false;
        }
        // 建構時 `killer` 一定有值；`sel` 等於它又通過上一行，所以這裡不會是 `None`。
        let Some(killer) = killer else {
            return false;
        };
        let scan_code = if input.up {
            72
        } else if input.down {
            80
        } else if input.left {
            75
        } else if input.right {
            77
        } else if input.confirm {
            28
        } else if input.cancel {
            1
        } else {
            0
        };
        if scan_code == 0 {
            return true;
        }
        let (selected, result) = match battle::advance_native_item_selector(self.item_sel, 8, scan_code, false, 0) {
            Ok(advanced) => advanced,
            Err(err) => {
                self.load_err = format!("native death reward: {err}");
                return true;
            }
        };
        if selected != self.item_sel {
            self.item_sel = selected;
            if !self.refresh_native_item_panel(killer) {
                self.load_err = String::from("native death reward: item panel refresh failed");
            }
        }
        if result == battle::NATIVE_ITEM_SELECTOR_CONFIRM {
            let mut probe = self.st.as_ref().unwrap().units[killer].clone();
            if let Err(err) = battle::replace_native_full_inventory_reward(Some(&mut probe), self.item_sel, item) {
                self.load_err = format!("native death reward: {err}");
                return true;
            }
            self.native_death_reward_ui.as_mut().unwrap().selected = self.item_sel;
            self.item_closing = true;
            self.item_anim_step = 0;
        } else if result == battle::NATIVE_ITEM_SELECTOR_CANCEL {
            self.native_death_reward_ui.as_mut().unwrap().selected = -1;
            self.item_closing = true;
            self.item_anim_step = 0;
        }
        true
    }

    /// 面板收完：有選擇就把獎勵換進那一格並結束，否則開「放棄」對話（0x1AB1F）。
    ///
    /// Go：`native_death_reward_ui.go:195`，`func (g *Game) finishNativeDeathRewardItemClose()`
    pub(crate) fn finish_native_death_reward_item_close(&mut self) {
        let Some(prompt) = self.native_death_reward_ui.as_ref() else {
            return;
        };
        if prompt.selected >= 0 {
            let (killer, selected, item) = (prompt.killer, prompt.selected, prompt.item);
            let unit = killer.map(|k| &mut self.st.as_mut().unwrap().units[k]);
            if let Err(err) = battle::replace_native_full_inventory_reward(unit, selected, item) {
                self.load_err = format!("native death reward: {err}");
                return;
            }
            self.finish_native_death_reward_ui();
            return;
        }
        let prompt = prompt.clone();
        self.begin_native_death_reward_selector_cancel(Some(prompt));
    }

    /// Go：`native_death_reward_ui.go:211`，`func (g *Game) beginNativeDeathRewardSelectorCancel(prompt *nativeDeathRewardUIState)`
    pub(crate) fn begin_native_death_reward_selector_cancel(&mut self, prompt: Option<NativeDeathRewardUIState>) {
        let (Some(prompt), Some(ui)) = (prompt, self.native_preparation_ui.as_ref()) else {
            self.load_err = String::from("native death reward: selector-cancel dialogue unavailable");
            return;
        };
        let abandon = match ui.status.font.as_ref() {
            Some(font) => campaign::compose_native_death_reward_selector_cancel(
                &prompt.dialogue,
                &prompt.portrait,
                ui.status.strings.as_ref(),
                font,
                prompt.item,
            )
            .map_err(|err| err.to_string()),
            None => Err(String::from("FDTXT font unavailable")),
        };
        let abandon = match abandon {
            Ok(abandon) => abandon,
            Err(err) => {
                self.load_err = format!("native death reward: {err}");
                return;
            }
        };
        let frames = match campaign::native_class_list_opening_frames(&prompt.source, &abandon) {
            Ok(frames) if frames.len() == 6 => frames,
            _ => {
                self.load_err = String::from("native death reward: selector-cancel opening frames unavailable");
                return;
            }
        };
        self.native_system_end_turn_ui = Some(Box::new(NativeSystemEndTurnUIState {
            source: prompt.source.clone(),
            dialogue: abandon.clone(),
            question: abandon.clone(),
            canceled: vec![abandon],
            death_reward: Some(prompt),
            ..Default::default()
        }));
        self.native_system_end_turn_confirm = false;
        self.native_system_end_turn_delay = 0;
        self.native_class_ui_job = Some(NativeClassUIJob {
            frames,
            after: Some(Box::new(|g: &mut Game| {
                g.native_system_end_turn_delay = NATIVE_SYSTEM_END_TURN_DELAY_FRAMES;
            })),
            ..Default::default()
        });
        self.bind_native_system_end_turn_hd(FULL_REWARD_CANCEL_OPENING);
    }

    /// Go：`native_death_reward_ui.go:241`，`func (g *Game) finishNativeDeathRewardUI()`
    pub(crate) fn finish_native_death_reward_ui(&mut self) {
        let Some(prompt) = self.native_death_reward_ui.take() else {
            return;
        };
        let then = self.native_death_reward_then.take();
        self.sel = prompt.previous_selection;
        self.native_system_end_turn_ui = None;
        self.native_system_end_turn_confirm = false;
        self.native_system_end_turn_delay = 0;
        self.native_class_ui_job = None;
        self.clear_native_item_panel();
        if let Some(then) = then {
            then(self);
        }
    }
}

#[cfg(test)]
#[path = "native_death_reward_ui/native_death_reward_ui_test.rs"]
mod native_death_reward_ui_test;
