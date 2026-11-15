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
                    if state.treasure.is_some() {
                        todo!("R8: action_overlay_runtime.go:719 finishNativeTreasurePrompt");
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
                            todo!("R8: action_overlay_runtime.go:740 startNextNativeSystemGroupMarchStep");
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
    pub(crate) fn draw_native_system_end_turn(&mut self, screen: &mut Picture) -> bool {
        todo!("r5:action_overlay_runtime.go:drawNativeSystemEndTurn")
    }
}
