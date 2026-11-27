//! Go 來源：`remake/cmd/fd2/action_overlay_runtime.go`（898 行）。
//!
//! 骨架：Go 檔的 28 個函式中產生 11 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`action_overlay_runtime.go:15`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeSystemEndTurnUIState {
    pub(crate) hd: Option<ModernHDSystemConfirmation>,
    pub(crate) source: Vec<u8>,
    pub(crate) dialogue: Vec<u8>,
    pub(crate) question: Vec<u8>,
    pub(crate) portraits: Vec<dato::Frame>,
    pub(crate) accepted: Vec<Vec<u8>>,
    pub(crate) canceled: Vec<Vec<u8>>,
    pub(crate) choice: i64,
    pub(crate) accepted_outcome: bool,
    pub(crate) exit_program: bool,
    pub(crate) group_march: bool,
    pub(crate) group_march_plan: battle::NativeSystemGroupMarchPlan,
    pub(crate) save_current: bool,
    pub(crate) save_path: String,
    pub(crate) save_stored: Vec<u8>,
    pub(crate) load_current: bool,
    pub(crate) load_candidate: Option<Box<Game>>,
    pub(crate) treasure: Option<NativeTreasurePrompt>,
    pub(crate) death_reward: Option<NativeDeathRewardUIState>,
    pub(crate) reward_message: Option<NativeDeathRewardMessageState>,
}

/// 0x17259：delay(0xC8 ms)，60 Hz 約十二幀。
///
/// Go：`action_overlay_runtime.go:12`
pub(crate) const NATIVE_SYSTEM_END_TURN_DELAY_FRAMES: i64 = 12;

pub(crate) const ACTION_OVERLAY_OPENING: &str = "opening";

pub(crate) const ACTION_OVERLAY_OPEN: &str = "open";

pub(crate) const ACTION_OVERLAY_CLOSING: &str = "closing";

impl Game {
    /// beginActionOverlayOpen starts the four presents recovered at 0x1741c.
    /// There is no delay call between native presents, so the remake assigns one
    /// presented Ebiten frame to each step without claiming an original duration.
    ///
    /// Go：`action_overlay_runtime.go:44`，`func (g *Game) beginActionOverlayOpen(selection int)`
    pub(crate) fn begin_action_overlay_open(&mut self, selection: i64) {
        self.ring = true;
        self.ring_sel = selection;
        self.action_overlay_phase = ACTION_OVERLAY_OPENING.to_string();
        self.action_overlay_frame = 0;
        self.action_overlay_after = None;
        self.action_overlay_drawn = false;
        self.action_overlay_shot_hold = false;
        self.action_overlay_blink = fdother::ActionOverlaySelectionBlink::default();
    }

    /// beginActionOverlayClose starts the independent four-present sequence from
    /// 0x176b4. The selected action is deferred until all four close frames have
    /// been presented; it must not appear beneath an overlay that native code was
    /// still closing.
    ///
    /// Go：`action_overlay_runtime.go:59`，`func (g *Game) beginActionOverlayClose(after func())`
    /// Go 的 func 參數 `after` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_action_overlay_close(&mut self, after: Option<Box<dyn FnOnce(&mut Game) + Send>>) {
        if !self.ring {
            if let Some(after) = after {
                after(self);
            }
            return;
        }
        self.action_overlay_phase = ACTION_OVERLAY_CLOSING.to_string();
        self.action_overlay_frame = 0;
        self.action_overlay_after = after;
        self.action_overlay_drawn = false;
        self.action_overlay_shot_hold = false;
    }

    /// Go：`action_overlay_runtime.go:73`，`func (g *Game) actionOverlayBlocksInput() bool`
    pub(crate) fn action_overlay_blocks_input(&self) -> bool {
        self.action_overlay_phase == ACTION_OVERLAY_OPENING || self.action_overlay_phase == ACTION_OVERLAY_CLOSING
    }

    /// Go：`action_overlay_runtime.go:78`，`func (g *Game) actionOverlayRenderState() (frame int, closing bool)`
    pub(crate) fn action_overlay_render_state(&self) -> (i64, bool) {
        match self.action_overlay_phase.as_str() {
            ACTION_OVERLAY_OPENING => (self.action_overlay_frame, false),
            ACTION_OVERLAY_CLOSING => (self.action_overlay_frame, true),
            _ => (3, false),
        }
    }

    /// stepActionOverlayLifecycle runs once near the start of Update. A sequence is
    /// initialized later in an input Update, so frame zero is drawn before the next
    /// call advances it. The callback similarly runs only after close frame three
    /// was available to Draw for a complete update interval.
    ///
    /// Go：`action_overlay_runtime.go:93`，`func (g *Game) stepActionOverlayLifecycle()`
    pub(crate) fn step_action_overlay_lifecycle(&mut self) {
        if self.action_overlay_shot_hold {
            return;
        }
        if self.action_overlay_blocks_input() && !self.action_overlay_drawn {
            return;
        }
        match self.action_overlay_phase.as_str() {
            ACTION_OVERLAY_OPENING => {
                if self.action_overlay_frame < 3 {
                    self.action_overlay_frame += 1;
                    self.action_overlay_drawn = false;
                    return;
                }
                self.action_overlay_phase = ACTION_OVERLAY_OPEN.to_string();
                self.action_overlay_drawn = false;
            }
            ACTION_OVERLAY_OPEN => {
                // sub_17898 的等待迴圈每次都重新取樣 0x46C 再決定要不要翻相位；
                // 開合動畫(sub_1741C／sub_176B4)不含這個加法，故只在此推進。
                if let (tick, true) = self.native_map_clock.current() {
                    self.action_overlay_blink.advance(tick);
                }
            }
            ACTION_OVERLAY_CLOSING => {
                if self.action_overlay_frame < 3 {
                    self.action_overlay_frame += 1;
                    self.action_overlay_drawn = false;
                    return;
                }
                let after = self.action_overlay_after.take();
                self.action_overlay_phase = String::new();
                self.action_overlay_frame = 0;
                self.action_overlay_drawn = false;
                self.ring = false;
                if let Some(after) = after {
                    after(self);
                }
            }
            _ => {}
        }
    }

    /// Go：`action_overlay_runtime.go:133`，`func (g *Game) markActionOverlayDrawn()`
    pub(crate) fn mark_action_overlay_drawn(&mut self) {
        // Go 由 Draw 呼叫（`main.go:10159` 與 `drawRing`）；Rust 由呈現回報呼叫，見 PORTING.md「C 組：指令環的呈現回報」。
        if self.action_overlay_blocks_input() {
            self.action_overlay_drawn = true;
        }
    }

    /// Go：`action_overlay_runtime.go:139`，`func (g *Game) resetActionOverlayLifecycle()`
    pub(crate) fn reset_action_overlay_lifecycle(&mut self) {
        self.cancel_native_ai_command_modifier_presentation();
        self.cancel_native_command32_presentation();
        self.cancel_native_command33_presentation();
        self.cancel_native_command34_presentation();
        self.cancel_native_command35_presentation();
        self.cancel_native_ai_item_presentation();
        self.ring = false;
        self.native_system_cursor_overlay = false;
        self.native_system_nested_open = false;
        self.native_system_options_open = false;
        self.native_system_info_ui = None;
        self.native_system_end_turn_confirm = false;
        self.native_system_end_turn_delay = 0;
        self.native_system_exit_requested = false;
        self.native_system_group_march = None;
        self.native_system_group_march_step = 0;
        self.pending_native_death_rewards = Vec::new();
        self.pending_native_reward_msgs = Vec::new();
        self.native_death_reward_ui = None;
        self.native_death_reward_then = None;
        if self.native_system_end_turn_ui.is_some() {
            self.native_class_ui_job = None;
        }
        self.native_system_end_turn_ui = None;
        self.action_overlay_phase = String::new();
        self.action_overlay_frame = 0;
        self.action_overlay_after = None;
        self.action_overlay_drawn = false;
        self.action_overlay_shot_hold = false;
    }

    /// Go：`action_overlay_runtime.go:171`，`func (g *Game) currentNativeSystemOptions() fdother.NativeSystemOptions`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn current_native_system_options(&self) -> fdother::NativeSystemOptions {
        if let Some(options) = self.native_system_options {
            return options;
        }
        fdother::default_native_system_options()
    }

    /// Go：`action_overlay_runtime.go:181`，`func (g *Game) nativeSystemOptionsReady() bool`
    pub(crate) fn native_system_options_ready(&self) -> bool {
        if self.current_native_system_options().action_overlay_state().is_err() {
            return false;
        }
        [54usize, 57, 60, 63, 66, 69, 72, 75]
            .iter()
            .all(|&index| self.native_action_cells.get(index).is_some_and(Option::is_some))
    }

    /// 系統選項（0x1728C）切換一格；音樂與 HUD 顯示閘門 A 有立即效果。
    ///
    /// Go：`action_overlay_runtime.go:193`，`func (g *Game) toggleNativeSystemOption(selector int) bool`
    pub(crate) fn toggle_native_system_option(&mut self, selector: i64) -> bool {
        let Ok(next) = self.current_native_system_options().toggle(selector) else {
            return false;
        };
        self.native_system_options = Some(next);
        match selector {
            0 => {
                if next.music_enabled() {
                    if !self.native_system_bgm_track.is_empty() {
                        let track = std::mem::take(&mut self.native_system_bgm_track);
                        self.play_bgm(&track);
                    }
                } else {
                    self.mute_native_system_bgm();
                }
            }
            3 => {
                self.restore_native_map_hud_gate_a(next.raw_51aab);
                if let Some(st) = self.st.as_mut().filter(|st| st.has_native_map_hud_state) {
                    st.native_map_hud_state.display_gate_a = next.raw_51aab;
                }
            }
            _ => {}
        }
        true
    }

    /// 系統確認框共用的建立程序（Go 各個 `beginNative…` 逐一重複同一段）：先在目前地圖畫面上建好對話框、
    /// 問句、接受與取消兩組回覆和 10 格開場，全部成功才收起命令框並交給 `after_close`。任何素材缺失回
    /// false，命令框保持原狀。
    ///
    /// HD 確認框（`freezeModernHDSystemConfirmation`，R6）尚未移植：`hd` 留空，HD 介面下也畫原版索引畫格。
    #[allow(clippy::too_many_arguments)]
    fn begin_native_system_confirmation<QE, RE>(
        &mut self,
        portrait: dato::Frame,
        portraits: Vec<dato::Frame>,
        question: impl Fn(&[u8], &dato::Frame, &fdtxt::Strings, &fdtxt::Font) -> std::result::Result<Vec<u8>, QE>,
        response: impl Fn(
            &[u8],
            Option<&fdtxt::Strings>,
            Option<&fdtxt::Font>,
            bool,
        ) -> std::result::Result<Vec<Vec<u8>>, RE>,
        mut state: NativeSystemEndTurnUIState,
        after_close: impl FnOnce(&mut Game) + Send + 'static,
    ) -> bool {
        let Some(ui) = self.native_preparation_ui.as_ref() else {
            return false;
        };
        let (Some(strings), Some(font)) = (ui.status.strings.as_ref(), ui.status.font.as_ref()) else {
            return false;
        };
        let source = self.native_map_vga.clone();
        let Ok(dialogue) = campaign::compose_native_preparation_confirmation_dialogue(&source, &ui.dialogue, &portrait)
        else {
            return false;
        };
        let Ok(question) = question(&dialogue, &portrait, strings, font) else {
            return false;
        };
        let (Ok(accepted), Ok(canceled)) = (
            response(&question, Some(strings), Some(font), true),
            response(&question, Some(strings), Some(font), false),
        ) else {
            return false;
        };
        let frames = match campaign::native_preparation_confirmation_opening_frames(
            &source,
            &dialogue,
            &question,
            &ui.choices,
        ) {
            Ok(frames) if frames.len() == 10 => frames,
            _ => return false,
        };
        (state.portraits, state.source, state.dialogue, state.question) = (portraits, source, dialogue, question);
        (state.accepted, state.canceled) = (accepted, canceled);
        // 狀態含不能跨執行緒的欄位，先暫存在 Game，收框延續再取出（Go 由閉包捕捉指標）。
        self.native_system_confirmation_pending = Some(Box::new(state));
        self.begin_action_overlay_close(Some(Box::new(move |g: &mut Game| {
            after_close(g);
            g.native_system_end_turn_confirm = true;
            g.native_system_end_turn_ui = g.native_system_confirmation_pending.take();
            g.reset_native_class_ui_pulse();
            g.native_class_ui_job = Some(NativeClassUIJob { frames, ..Default::default() });
        })));
        true
    }

    /// 共用 0x117E7→0x16F55 的 Down→END：在收起命令框前建好 0x1956B→0x19953 的整份確認畫面。
    ///
    /// Go：`action_overlay_runtime.go:241`，`func (g *Game) beginNativeSystemEndTurn() bool`
    pub(crate) fn begin_native_system_end_turn(&mut self) -> bool {
        if !self.native_system_cursor_overlay || !self.native_system_confirmation_context(3) {
            return false;
        }
        let ui = self.native_preparation_ui.as_ref().unwrap();
        let (portrait, portraits) = (ui.portrait.clone(), ui.portraits.clone());
        self.begin_native_system_confirmation(
            portrait,
            portraits,
            campaign::compose_native_battle_end_turn_question,
            campaign::native_battle_end_turn_response_frames,
            NativeSystemEndTurnUIState::default(),
            |g| g.native_system_cursor_overlay = false,
        )
    }

    /// 巢狀選單 selector3：YES 回覆、停音樂、延遲與收框都結束後才發布離開要求。
    ///
    /// Go：`action_overlay_runtime.go:296`，`func (g *Game) beginNativeNestedSystemExit() bool`
    pub(crate) fn begin_native_nested_system_exit(&mut self) -> bool {
        if !self.native_system_cursor_overlay
            || !self.native_system_nested_open
            || !self.native_system_confirmation_context(3)
        {
            return false;
        }
        let ui = self.native_preparation_ui.as_ref().unwrap();
        let (portrait, portraits) = (ui.portrait.clone(), ui.portraits.clone());
        self.begin_native_system_confirmation(
            portrait,
            portraits,
            campaign::compose_native_battle_exit_question,
            campaign::native_battle_exit_response_frames,
            NativeSystemEndTurnUIState { exit_program: true, ..Default::default() },
            |g| (g.native_system_nested_open, g.native_system_cursor_overlay) = (false, false),
        )
    }

    /// 巢狀選單 selector2：讀回目前戰況存檔。Go 先以 `prepareNativeContinueFromCurrentSnapshot` 建好整份候選
    /// 狀態再開確認框；那條還原鏈（`native_continue_handoff.go`）尚未移植，這裡失敗即關閉，戰場保持不變。
    ///
    /// Go：`action_overlay_runtime.go:418`，`func (g *Game) beginNativeNestedCurrentLoad() bool`
    pub(crate) fn begin_native_nested_current_load(&mut self) -> bool {
        false
    }

    /// 外層 selector1 全軍移動：收命令框前預演逐單位尋路與事件，任一未知事件整批不動。
    ///
    /// Go：`action_overlay_runtime.go:487`，`func (g *Game) beginNativeSystemGroupMarch() bool`
    pub(crate) fn begin_native_system_group_march(&mut self) -> bool {
        if !self.native_system_cursor_overlay || self.m.is_none() || !self.native_system_confirmation_context(1) {
            return false;
        }
        let cell = battle::Cell { x: self.cur_x, y: self.cur_y };
        let st = self.st.as_mut().unwrap();
        let Some(leader_fig) = st.units.first().filter(|unit| unit.has_battle_fig).map(|unit| unit.battle_fig) else {
            return false;
        };
        let Ok(plan) = st.plan_native_system_group_march(cell) else {
            return false;
        };
        if !self.preflight_native_system_group_march_events(&plan) {
            return false;
        }
        let portraits = match load_native_separated_portrait(leader_fig) {
            Ok(portraits) if portraits.len() > 3 => portraits,
            _ => return false,
        };
        self.begin_native_system_confirmation(
            portraits[0].clone(),
            portraits,
            campaign::compose_native_battle_group_march_question,
            campaign::native_battle_group_march_response_frames,
            NativeSystemEndTurnUIState { group_march: true, group_march_plan: plan, ..Default::default() },
            |g| g.native_system_cursor_overlay = false,
        )
    }

    /// Go 各確認入口共同的前置條件（環開著、方向正確、不在 AI 回合、戰場未結束、介面與地圖畫面都在）。
    fn native_system_confirmation_context(&self, direction: i64) -> bool {
        self.ring
            && self.ring_sel == direction
            && self.st.is_some()
            && !self.ai_busy
            && self.result.is_empty()
            && self.native_class_ui.is_some()
            && self.native_map_vga.len() == 320 * 200
            && self.native_preparation_ui.as_ref().is_some_and(|ui| ui.portraits.len() > 3)
    }

    /// 同一方向兩層不同 owner：外層 0x16F55 selector1 是全軍移動；巢狀 0x19DF7 selector1 是存檔。
    ///
    /// Go：`action_overlay_runtime.go:552`，`func (g *Game) activateNativeSystemDirectionOne() bool`
    ///
    /// 巢狀存檔在有原版目前戰況基底時走 `beginNativeNestedCurrentSave`（`buildNativeCurrentSaveStored` 尚未移植，
    /// 先回報失敗）；一般劇情進戰沒有基底，改存重製端戰場快照到第一槽（56「#303 戰場JSON快照」）。
    pub(crate) fn activate_native_system_direction_one(&mut self) -> bool {
        if !self.native_system_cursor_overlay || !self.ring || self.ring_sel != 1 {
            return false;
        }
        if self.native_system_nested_open {
            if native_current_save_path().is_empty() || self.native_current_save_plain.is_empty() {
                self.msg.clear();
                if let Err(err) = self.save_game_to_slot_from(0, true)
                    && self.msg.is_empty()
                {
                    self.msg = format!("戰場存檔失敗：{err}");
                }
                return true;
            }
            self.msg = String::from("原版目前戰況存檔來源或資產不完整，未寫入檔案");
            return true;
        }
        if !self.begin_native_system_group_march() {
            self.msg = String::from("原版全軍移動來源或事件不完整，未執行");
        }
        true
    }

    /// 全軍移動的事件預檢：只有 Game 層取得的演出素材在這裡補驗，避免先移動才發現缺素材。
    ///
    /// Go：`action_overlay_runtime.go:578`，`func (g *Game) preflightNativeSystemGroupMarchEvents(plan battle.NativeSystemGroupMarchPlan) bool`
    ///
    /// 事件 61 的對白與演出（`event61DialogueActions`）尚未移植，含事件 61 的計畫失敗即關閉。
    fn preflight_native_system_group_march_events(&mut self, plan: &battle::NativeSystemGroupMarchPlan) -> bool {
        for step in &plan.steps {
            for event in &step.events {
                match event.event_id {
                    75 => {
                        if !event75_dialogue_preflight(self, event.text_index) {
                            return false;
                        }
                    }
                    _ => return false,
                }
            }
        }
        true
    }

    /// Go：`action_overlay_runtime.go:613`，`func (g *Game) confirmNativeSystemEndTurn()`
    pub(crate) fn confirm_native_system_end_turn(&mut self) {
        if self.native_system_end_turn_confirm {
            self.finish_native_system_end_turn_choice(true);
        }
    }

    /// Go：`action_overlay_runtime.go:620`，`func (g *Game) cancelNativeSystemEndTurn()`
    pub(crate) fn cancel_native_system_end_turn(&mut self) {
        if self.native_system_end_turn_confirm {
            self.finish_native_system_end_turn_choice(false);
        }
    }

    /// 確認框收合 4 格 → 回覆畫格 → 停留延遲（`step_native_system_end_turn` 收框後執行結果）。
    ///
    /// Go：`action_overlay_runtime.go:627`，`func (g *Game) finishNativeSystemEndTurnChoice(accepted bool)`
    ///
    /// 寶箱問句走 `finish_native_treasure_choice`；死亡獎勵與原版目前戰況存檔兩支的入口尚未移植，不會走到這裡。
    fn finish_native_system_end_turn_choice(&mut self, accepted: bool) {
        let (Some(state), Some(ui)) = (self.native_system_end_turn_ui.as_ref(), self.native_preparation_ui.as_ref())
        else {
            self.native_system_end_turn_confirm = false;
            return;
        };
        let frames = match campaign::native_class_confirmation_closing_frames(&state.question, &ui.choices) {
            Ok(frames) if frames.len() == 4 => frames,
            _ => return,
        };
        self.native_system_end_turn_confirm = false;
        self.native_class_ui_job = Some(NativeClassUIJob {
            frames,
            after: Some(Box::new(move |g: &mut Game| {
                let Some(state) = g.native_system_end_turn_ui.as_mut() else {
                    return;
                };
                state.accepted_outcome = accepted;
                let response = if accepted { state.accepted.clone() } else { state.canceled.clone() };
                g.native_class_ui_job = Some(NativeClassUIJob {
                    frames: response,
                    after: Some(Box::new(move |g: &mut Game| {
                        if accepted && g.native_system_end_turn_ui.as_ref().is_some_and(|state| state.exit_program) {
                            g.stop_bgm();
                        }
                        g.native_system_end_turn_delay = NATIVE_SYSTEM_END_TURN_DELAY_FRAMES;
                    })),
                    ..Default::default()
                });
            })),
            ..Default::default()
        });
    }

    /// 全軍移動逐單位走路；每段走完提交該單位，最後結束回合。
    ///
    /// Go：`action_overlay_runtime.go:752`，`func (g *Game) startNextNativeSystemGroupMarchStep()`
    pub(crate) fn start_next_native_system_group_march_step(&mut self) {
        while self.walk.is_none() {
            let Some(plan) = self.native_system_group_march.as_ref() else {
                return;
            };
            if self.native_system_group_march_step as usize >= plan.steps.len() {
                self.native_system_group_march = None;
                self.native_system_group_march_step = 0;
                self.end_turn();
                return;
            }
            let step = plan.steps[self.native_system_group_march_step as usize].clone();
            let units = self.st.as_ref().map_or(0, |st| st.units.len());
            if step.unit_index < 0 || step.unit_index as usize >= units {
                self.load_err = String::from("native system group-march runtime unit disappeared");
                self.native_system_group_march = None;
                return;
            }
            if step.path.len() < 2 {
                if !self.commit_native_system_group_march_step(&step) {
                    return;
                }
                continue;
            }
            let (unit, path, events) = (step.unit_index as UnitIdx, step.path.clone(), step.events.clone());
            self.walk = Some(WalkAnim {
                u: Some(unit),
                path,
                then: Some(Box::new(move |g: &mut Game| {
                    if g.commit_native_system_group_march_step(&step) {
                        g.start_next_native_system_group_march_step();
                    }
                })),
                native_event_selector: Some(1),
                native_group_march_events: events,
                ..Default::default()
            });
        }
    }

    fn commit_native_system_group_march_step(&mut self, step: &battle::NativeSystemGroupMarchStep) -> bool {
        if let Err(err) = self.st.as_mut().unwrap().commit_native_system_group_march_step(step) {
            self.load_err = format!("native system group-march commit: {err}");
            self.native_system_group_march = None;
            return false;
        }
        self.native_system_group_march_step += 1;
        true
    }

    /// Go：`action_overlay_runtime.go:701`，`func (g *Game) stepNativeSystemEndTurn()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn step_native_system_end_turn(&mut self) {
        if self.native_system_end_turn_delay <= 0 {
            return;
        }
        self.native_system_end_turn_delay -= 1;
        if self.native_system_end_turn_delay == 0 {
            let Some(state) = self.native_system_end_turn_ui.as_ref() else {
                return;
            };
            let frames = match campaign::native_class_list_closing_frames(&state.source, &state.dialogue) {
                Ok(frames) if frames.len() == 5 => frames,
                _ => return,
            };
            let restore = state.source.clone();
            let kind = if state
                .death_reward
                .as_ref()
                .and_then(|reward| reward.hd.as_ref())
                .is_some_and(|hd| hd.kind == FULL_REWARD_CANCEL_OPENING)
            {
                FULL_REWARD_CANCEL_CLOSING
            } else {
                FULL_REWARD_BLANK_CLOSING
            };
            // Go 的延續捕捉 `state` 指標，並在延續開頭把 `g.nativeSystemEndTurnUI` 設成 nil；收合期間沒有
            // 別的寫入端替換這個欄位，Rust 在延續執行時才從 `Game` 取出同一份狀態。
            self.native_class_ui_job = Some(NativeClassUIJob {
                frames,
                restore,
                redraw_native_map_after_restore: true,
                after: Some(Box::new(|g: &mut Game| {
                    let Some(mut state) = g.native_system_end_turn_ui.take() else {
                        return;
                    };
                    let accepted = state.accepted_outcome;
                    if let Some(prompt) = state.treasure.take() {
                        g.finish_native_treasure_prompt(prompt);
                        return;
                    }
                    if state.death_reward.is_some() {
                        todo!("R8: action_overlay_runtime.go:723 finishNativeDeathRewardUI");
                    }
                    if accepted {
                        if state.load_current {
                            let Some(candidate) = state.load_candidate.take() else {
                                g.load_err = String::from("原版目前戰況讀檔：私有候選狀態遺失");
                                return;
                            };
                            *g = *candidate;
                        } else if state.exit_program {
                            g.native_system_exit_requested = true;
                        } else if state.group_march {
                            g.native_system_group_march = Some(std::mem::take(&mut state.group_march_plan));
                            g.native_system_group_march_step = 0;
                            g.start_next_native_system_group_march_step();
                        } else if !state.save_current {
                            g.end_turn();
                        }
                    }
                })),
                ..Default::default()
            });
            self.bind_native_system_end_turn_hd(kind);
        }
    }

    /// applyNativeOverlayDirectionInput 重現 sub_177FC 於 `0x17835..0x17897` 的四個
    /// 方向分支：掃描碼只有在目前 caller 的 availability 為零時才寫入 `[0x53c57]`；
    /// 不可用的方向按下去原版完全沒有反應，等待迴圈繼續。0x16F55、0x1728C、
    /// 0x18D8C、0x19DF7 與 0x1BBDC 五個 chooser 共用這個閘門，各自帶自己的表。
    ///
    /// Go：`action_overlay_runtime.go:879`，`func (g *Game) applyNativeOverlayDirectionInput()`
    pub(crate) fn apply_native_overlay_direction_input(&mut self) {
        let state = self.native_action_overlay_state();
        for (direction, key) in [Key::ArrowUp, Key::ArrowLeft, Key::ArrowRight, Key::ArrowDown].into_iter().enumerate()
        {
            if !self.tick_input.is_just_pressed(key) {
                continue;
            }
            if fdother::action_overlay_accepts_direction(state.availability, direction as i64) {
                self.ring_sel = direction as i64;
            }
        }
        self.apply_pointer_to_overlay(state.availability); // Go `action_overlay_runtime.go:890`（r5i）
    }

    /// beginBattleActionOverlay 依 sub_173E7（0x18E53／0x18ED6 各呼叫一次）挑起始
    /// 方向：第一個 availability 為零的方向。硬寫 1 會讓起始選擇落在原版根本不會
    /// 停留的不可用格上，方向鍵閘門一旦成立就再也移不開。
    ///
    /// Go：`action_overlay_runtime.go:896`，`func (g *Game) beginBattleActionOverlay()`
    pub(crate) fn begin_battle_action_overlay(&mut self) {
        let availability = self.action_overlay_availability();
        self.begin_action_overlay_open(fdother::action_overlay_initial_direction(availability));
    }

    /// Go `bindModernHDFullReward(kind)` 與 `bindModernHDSystemConfirmation(kind)`（`modern_hd_full_reward.go:62`、
    /// `modern_hd_system_confirmation.go:329`，兩者都標 `r8`）：把 HD 外框接到同一個 `nativeClassUIJob`。兩者只在
    /// HD 狀態存在時動作，忠實模式下 `hd` 是 nil、沒有效果。Go 以指標共用 job，Rust 的 job 不能共用，HD 分支在 R6。
    fn bind_native_system_end_turn_hd(&mut self, _kind: FullRewardPhase) {
        if self.native_death_reward_ui.as_ref().is_some_and(|p| p.hd.is_some()) {
            todo!("R6: modern_hd_full_reward.go:62 bindModernHDFullReward");
        }
        if self.native_system_end_turn_ui.as_ref().is_some_and(|s| s.hd.is_some()) {
            todo!("R6: modern_hd_system_confirmation.go:329 bindModernHDSystemConfirmation");
        }
    }

    /// Go：`action_overlay_runtime.go:225`，`func (g *Game) nativeSystemOverlayReady() bool`（`r8`，隨 `confirm`
    /// 補移植）。
    pub(crate) fn native_system_overlay_ready(&self) -> bool {
        if self.st.is_none() || self.m.is_none() || self.ai_busy || !self.result.is_empty() {
            return false;
        }
        let state = fdother::native_continue_action_overlay_state();
        for direction in 0..4 {
            match state.cell_index(direction) {
                Ok(index)
                    if index >= 0
                        && (index as usize) < self.native_action_cells.len()
                        && self.native_action_cells[index as usize].is_some() => {}
                _ => return false,
            }
        }
        true
    }
}

#[cfg(test)]
#[path = "action_overlay_runtime/action_overlay_runtime_test.rs"]
mod action_overlay_runtime_test;

#[cfg(test)]
#[path = "action_overlay_runtime/main_c_test.rs"]
mod main_c_test;

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// drawNativeSystemEndTurn 在 END 對話展開、等待輸入、顯示回覆與收合期間，
    /// 完整擁有320×200索引畫面，避免底下的重製指令層穿透。
    ///
    /// Go：`action_overlay_runtime.go:823`，`func (g *Game) drawNativeSystemEndTurn(screen *ebiten.Image) bool`
    ///
    /// R5 階段 2 r5h 逐行移植。第一章走到守門、`drawNativeClassUIJob`（開合幀 36 次）與死亡獎勵訊息的最後一幀
    /// （3 次）；確認框（`nativeSystemEndTurnConfirm`）與回覆延遲／寶箱等鍵兩支沒有走到，照 Go 移植，只由單元測試驗。
    /// HD 的三個 `drawModernHD*` 在忠實模式（`hdPresentationEnabled` 為 false，或對應的 HD 描述是 nil）都回 false；
    /// 會往下做 HD 呈現時 `todo!("r6:…")`。
    pub(crate) fn draw_native_system_end_turn(&mut self, screen: &mut Picture) -> bool {
        if self.native_system_end_turn_ui.is_none()
            || self.native_class_ui.is_none()
            || self.native_preparation_ui.is_none()
        {
            return false;
        }
        // Go `hdPresentationEnabled()`（`player_interface.go:32`，`r6`）：介面 HD 或美術 HD。
        let hd_presentation =
            self.modern_hd_interface_enabled() || self.modern_story_portraits.as_ref().is_some_and(|set| set.has_hd);
        if hd_presentation {
            let state = self.native_system_end_turn_ui.as_ref().unwrap();
            if state.hd.is_some() {
                todo!("r6:modern_hd_system_confirmation.go:drawModernHDSystemConfirmation");
            }
            if state.reward_message.is_some() && self.font.is_some() {
                todo!("r6:modern_hd_reward_message.go:drawModernHDRewardMessage");
            }
            if state.death_reward.as_ref().is_some_and(|p| p.hd.is_some()) {
                todo!("r6:modern_hd_full_reward.go:drawModernHDFullReward");
            }
        }
        if self.draw_native_class_ui_job(screen) {
            return true;
        }
        let state = self.native_system_end_turn_ui.as_ref().unwrap();
        let frame = if self.native_system_end_turn_confirm {
            let (question, ok) = self.compose_native_confirmation_mouth(&state.question, &state.portraits);
            if !ok {
                return false;
            }
            match campaign::compose_native_confirmation_choices(
                &question,
                &self.native_preparation_ui.as_ref().unwrap().choices,
                state.choice,
                self.native_class_ui_pulse / 2,
            ) {
                Ok(frame) => frame,
                Err(_) => return false,
            }
        } else if let Some(message) = state.reward_message.as_ref() {
            if !message.await_ack {
                return false;
            }
            message.r#final.clone()
        } else if self.native_system_end_turn_delay > 0 || state.treasure.as_ref().is_some_and(|t| t.await_ack) {
            let frames = if state.accepted_outcome { &state.accepted } else { &state.canceled };
            let Some(last) = frames.last() else {
                return false;
            };
            last.clone()
        } else {
            return false;
        };
        self.present_native_class_frame(screen, &frame);
        true
    }
}
