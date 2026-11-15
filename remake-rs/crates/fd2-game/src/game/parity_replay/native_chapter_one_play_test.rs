//! Go `remake/cmd/fd2/native_chapter_one_play_test.go` 的驅動端工具：`pump`、`closeRing`、`ackPresents`、
//! `answerNativeTreasurePrompt`、`ch01Blockers`、`pendingOwn`。`TestPlayChapterOneToVictory` 本身（自行挑格子的
//! 自動打法）不是章重播的一部分，R8 再移植。

use super::*;

/// Go `native_chapter_one_play_test.go:30`：`ch01FrameBudget`。
pub(crate) const CH01_FRAME_BUDGET: i64 = 4000;

/// 無畫面驅動端：持有 `Game` 與虛擬牆鐘。Go 的 `pump(t, g, …)`、`closeRing(t, g, …)` 等收 `*Game`，
/// 這裡改成 `Driver` 的方法，好在每次 `Update` 前推進同一個 [`VirtualFrameClock`]。
pub(crate) struct Driver {
    pub(crate) g: Game,
    pub(crate) clock: VirtualFrameClock,
    /// R5 階段 1：Go `ch01FrameObserver`（`native_chapter_one_play_test.go:300`）在 R5 黃金畫格的掛法；
    /// `None` 時（預設）與 R4 相同。
    pub(crate) frame_capture: Option<FrameCapture>,
    /// R5 階段 2 r5m：地圖與單位組的局部黃金（`rust_golden_map_layers_test.rs`），同一個掛點；預設 `None`。
    pub(crate) map_layer_capture: Option<MapLayerCapture>,
    /// R5 階段 2 r5d：對話與文字組的局部黃金（`rust_golden_dialogue_layers_test.rs`），同一個掛點；預設 `None`。
    pub(crate) dialogue_layer_capture: Option<DialogueLayerCapture>,
    /// R5 階段 2 r5p：轉場與調色盤＋Draw 本體兩段的局部黃金（`rust_golden_r5p_layers_test.rs`），同一個掛點；
    /// 預設 `None`。
    pub(crate) r5p_layer_capture: Option<R5pLayerCapture>,
    /// R5 階段 2 r5c：城鎮與設施組的局部黃金（`rust_golden_r5c_layers_test.rs`），同一個掛點另加 gap 擷取點；
    /// 預設 `None`。
    pub(crate) r5c_layer_capture: Option<R5cLayerCapture>,
    /// R5 階段 2 r5f：TTF 戰場短訊與勝負字樣的局部黃金（`rust_golden_ttf_layers_test.rs`），同一個掛點；預設 `None`。
    pub(crate) ttf_layer_capture: Option<TtfLayerCapture>,
}

/// Go `native_chapter_one_play_test.go:101`：`pendingOwn`，我方在場、存活、還沒行動的單位。
pub(crate) fn pending_own(g: &Game) -> Vec<UnitIdx> {
    let Some(st) = g.st.as_ref() else {
        return Vec::new();
    };
    st.units
        .iter()
        .enumerate()
        .filter(|(_, u)| u.camp == battle::OWN && u.on_field && u.alive() && !u.acted)
        .map(|(i, _)| i)
        .collect()
}

impl Driver {
    /// Go `native_chapter_one_play_test.go:302`：`pump`。每一幀先交給畫格擷取（Go `ch01FrameObserver`，章重播
    /// 預設沒有），再問 `done`，承認呈現、回答取寶提示、按掉敵方回合的掉落訊息與升級對話、按戰場事件對白，
    /// 最後 `parityUpdateGame`。
    pub(crate) fn pump(&mut self, budget: i64, mut done: impl FnMut(&mut Game) -> bool) -> bool {
        for _ in 0..budget {
            if let Some(capture) = self.frame_capture.as_mut() {
                capture.observe(&mut self.g);
            }
            if let Some(capture) = self.map_layer_capture.as_mut() {
                capture.observe(&mut self.g);
            }
            if let Some(capture) = self.dialogue_layer_capture.as_mut() {
                capture.observe(&mut self.g);
            }
            if let Some(capture) = self.r5p_layer_capture.as_mut() {
                capture.observe(&mut self.g);
            }
            if let Some(capture) = self.r5c_layer_capture.as_mut() {
                capture.observe(&mut self.g);
            }
            if let Some(capture) = self.ttf_layer_capture.as_mut() {
                capture.observe(&mut self.g);
            }
            if done(&mut self.g) {
                return true;
            }
            let g = &mut self.g;
            ack_presents(g);
            answer_native_treasure_prompt(g);
            // 敵方回合裡我方反擊擊倒有掉落時，0x1AA1D 的訊息會等鍵（原版側驅動端在 await 裡看到
            // dialogue 就送 enter）；玩家回合的訊息由攻擊步驟自己在檢查點之後按。
            if g.ai_busy && g.native_death_reward_message_awaiting_key() {
                g.acknowledge_native_death_reward_message();
            }
            if g.ai_busy && g.native_level_up_dialogue.is_some() {
                g.step_native_level_up_dialogue(true);
            }
            // 戰場事件的對白要玩家按 enter 才會往下走（正式路徑是 handleBattleEventDialogueInput）。
            if (g.battle_event.is_some() || g.native_turn_staging.is_some()) && !g.dialog.is_empty() {
                g.handle_battle_event_dialogue_input(true);
            }
            if let Err(err) = parity_update_game(self) {
                panic!("Update：{err}");
            }
            let g = &self.g;
            if !g.load_err.is_empty() {
                let Some(st) = g.st.as_ref().filter(|_| g.m.is_some()) else {
                    panic!("執行期錯誤：{}；節點={:?}，戰場尚未建立", g.load_err, node_id(g));
                };
                let view = &st.native_map_view_state;
                panic!(
                    "執行期錯誤：{}\n視圖：鏡頭({},{}) 游標({},{}) 可見({},{})\n阻塞：{}",
                    g.load_err,
                    view.camera_x,
                    view.camera_y,
                    view.cursor_x,
                    view.cursor_y,
                    view.visible_cursor_x,
                    view.visible_cursor_y,
                    ch01_blockers(g)
                );
            }
        }
        done(&mut self.g)
    }

    /// Go `native_chapter_one_play_test.go:346`：`closeRing`。推完指令環的收合動畫；開合掛在「這一幀被畫過」，
    /// 離屏不走 Draw，所以每幀先合成原生整幀，再以 `markActionOverlayDrawn` 承認。
    pub(crate) fn close_ring(&mut self, then: Box<dyn FnOnce(&mut Game) + Send>) -> bool {
        self.g.begin_action_overlay_close(Some(then));
        let mut frame = 0;
        while frame < 120 && self.g.ring {
            if let Err(err) = self.g.compose_native_map_frame() {
                panic!("組不出整幀：{err}");
            }
            self.g.mark_action_overlay_drawn();
            if let Err(err) = parity_update_game(self) {
                panic!("Update：{err}");
            }
            frame += 1;
        }
        !self.g.ring
    }
}

/// Go `native_chapter_one_play_test.go:485`：`ch01Blockers`。Go 以反射列出所有非零欄位；Rust 沒有反射，
/// 列出重播停住時要看的流程欄位（只用在失敗訊息）。
pub(crate) fn ch01_blockers(g: &Game) -> String {
    let mut parts: Vec<String> = Vec::new();
    let mut flag = |name: &str, on: bool| {
        if on {
            parts.push(name.to_string());
        }
    };
    flag("atk", g.atk.is_some());
    flag("walk", g.walk.is_some());
    flag("ring", g.ring);
    flag("aiBusy", g.ai_busy);
    flag("battleEvent", g.battle_event.is_some());
    flag("nativeTurnStaging", g.native_turn_staging.is_some());
    flag("nativeClassUIJob", g.native_class_ui_job.is_some());
    flag("spawnIntroTransition", g.spawn_intro_transition.is_some());
    flag("indexedTransition", g.indexed_transition.is_some());
    flag("nativeUnitPresent", g.native_unit_present.is_some());
    flag("actJob", g.act_job.is_some());
    flag("camPan", g.cam_pan.is_some());
    flag("focusJob", g.focus_job.is_some());
    flag("nativePlayerFocus", g.native_player_focus.is_some());
    flag("nativeEndTurnRecovery", g.native_end_turn_recovery.is_some());
    flag("nativePalettePulse", g.native_palette_pulse.is_some());
    flag("nativePaletteRamp", g.native_palette_ramp.is_some());
    flag("transitionReveal", g.transition_reveal.is_some());
    flag("nativeAIIdleRecovery", g.native_ai_idle_recovery.is_some());
    flag("nativeLevelUpDialogue", g.native_level_up_dialogue.is_some());
    flag("nativeSystemEndTurnUI", g.native_system_end_turn_ui.is_some());
    flag("nativeFieldEventPending", g.native_field_event_pending.is_some());
    flag("fade", g.fade.is_some());
    flag("sel", g.sel.is_some());
    parts.push(format!("bannerT={}", g.banner_t));
    parts.push(format!("dialog={}", g.dialog.len()));
    parts.push(format!("result={:?}", g.result));
    parts.push(format!("node={:?}", node_id(g)));
    parts.push(format!("titlePhase={:?}", g.title_phase));
    parts.push(format!("loadErr={:?}", g.load_err));
    if let Some(st) = g.st.as_ref() {
        parts.push(format!("turn={}", st.turn));
    }
    parts.join(" ")
}

/// Go `native_chapter_one_play_test.go:535`：`ackPresents`。承認演出工作的「這一幀已呈現」。Go 在三種情況呼叫
/// 正式 `g.Draw`（城鎮出發、攻擊演出、END 回收），這裡改呼叫呈現回報 [`Game::acknowledge_present`]；其餘照 Go
/// 只設 `drawn`。
pub(crate) fn ack_presents(g: &mut Game) {
    if g.native_town_departure.is_some() {
        g.acknowledge_present();
    }
    if let Some(scene) = g.atk.as_ref().and_then(|a| a.native_scene.as_ref())
        && (scene.prelude_frame < scene.prelude_images.len() as i64
            || scene.lead_frame < scene.lead_images.len() as i64
            || scene.body.is_some())
    {
        // 同一攻擊owner在body後仍持有兩段palette ramp；依正式Draw優先序確認。
        g.acknowledge_present();
    }
    if g.native_end_turn_recovery.is_some() {
        g.acknowledge_present();
        // 離屏測試以明示虛擬時間跨過 initial BIOS tick；兩個零 tick 發布仍由正式 Draw/Update 分別確認。
        if let Some(job) = g.native_end_turn_recovery.as_ref()
            && job.frame == 0
        {
            let ready_at = job.ready_at;
            g.step_native_end_turn_recovery(ready_at);
        }
    }
    // 正式Draw會重建被對白或record_bytes失效的地圖快取；離屏pump也要走同一合成器（#80）。
    if g.native_map_vga.is_empty()
        && g.st.as_ref().is_some_and(|st| st.has_native_map_view_state)
        && g.m.is_some()
        && native_map_assets_available(g.native_map_assets.as_ref())
        && let Err(err) = g.compose_native_map_frame()
    {
        g.load_err = format!("離屏原生畫面重建：{err}");
    }
    if let Some(j) = g.spawn_intro_transition.as_mut() {
        j.drawn = true;
    }
    if let Some(j) = g.indexed_transition.as_mut() {
        j.drawn = true;
    }
    if let Some(j) = g.native_palette_ramp.as_mut() {
        j.drawn = true;
    }
    if let Some(j) = g.native_palette_pulse.as_mut() {
        j.drawn = true;
    }
    if let Some(j) = g.transition_reveal.as_mut() {
        j.drawn = true;
    }
    if let Some(j) = g.native_unit_present.as_mut() {
        j.drawn = true;
    }
    if let Some(j) = g.native2189_a.as_mut() {
        j.drawn = true;
    }
    if g.native_ch23_loop.is_some() {
        // #144：Go 走正式 drawNativeCh23Loop 再交出回呼（第二十四章舞台）。
        todo!("R8: native_chapter_one_play_test.go:604 drawNativeCh23Loop（第二十四章舞台）")
    }
    if let Some(j) = g.native_class_ui_job.as_mut() {
        j.drawn = true;
    }
    // 死亡程式的 0x35822 閃白與 0x35BBA 接的 0x1DB65 呈現也掛在 drawn 上。
    if let Some(j) = g.native_turn_staging.as_mut() {
        j.drawn = true;
    }
    if let Some(j) = g.native_ch28_post_present.as_mut() {
        j.drawn = true;
    }
    // 指令／道具演出（玩家與敵方 0x15311 路線共用的 owner）也逐 Draw 推進。
    macro_rules! ack {
        ($($field:ident),* $(,)?) => {
            $(if let Some(j) = g.$field.as_mut() {
                j.drawn = true;
            })*
        };
    }
    ack!(
        native_cmd0_presentation,
        native_cmd1_presentation,
        native_cmd2_presentation,
        native_cmd3_presentation,
        native_cmd5_presentation,
        native_cmd6_presentation,
        native_cmd7_presentation,
        native_cmd8_presentation,
        native_cmd9_player,
        native_cmd9_ai_presentation,
        native_cmd1012,
        native_cmd24_presentation,
        native_cmd29_presentation,
        native_cmd32_presentation,
        native_cmd33_presentation,
        native_cmd34_presentation,
        native_cmd35_presentation,
        native_heal_presentation,
        native_modifier_presentation,
        native_ai_command_modifier,
        native_ai_item_presentation,
        native_ai_idle_recovery,
        native_field_event61,
    );
}

/// Go `native_chapter_one_play_test.go:692`：`answerNativeTreasurePrompt`。只回答寶物提示。
pub(crate) fn answer_native_treasure_prompt(g: &mut Game) {
    let class_job = g.native_class_ui_job.is_some();
    let Some(state) = g.native_system_end_turn_ui.as_mut() else {
        return;
    };
    let Some(treasure) = state.treasure.as_mut() else {
        return;
    };
    if class_job {
        return;
    }
    if treasure.await_ack {
        treasure.await_ack = false;
        g.native_system_end_turn_delay = 1;
        return;
    }
    if g.native_system_end_turn_confirm {
        todo!("R8: action_overlay_runtime.go:614 confirmNativeSystemEndTurn（寶物提示；第一章重播沒有待機在寶物格）")
    }
}
