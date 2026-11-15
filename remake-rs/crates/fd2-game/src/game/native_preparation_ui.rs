//! Go 來源：`remake/cmd/fd2/native_preparation_ui.go`（510 行）。
//!
//! 骨架：Go 檔的 26 個函式中產生 7 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! E 組另補移植 `stepNativePreparationCycleTick`（`r8`，`stepNativePreparationUILifecycle` 選人時每幀呼叫）。

use super::*;

/// Go：`native_preparation_ui.go:15`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Clone, Default)]
pub(crate) struct NativePreparationUIAssets {
    pub(crate) roster: Option<fdother::NativePreparationAssets>,
    pub(crate) status: battle::NativeItemPanelDataAssets,
    pub(crate) choices: Vec<fdother::RawCell>,
    pub(crate) dialogue: Vec<fdother::RawCell>,
    pub(crate) portrait: dato::Frame,
    pub(crate) portraits: Vec<dato::Frame>,
    pub(crate) slots_box: fdother::LMI1Entry,
}

impl Game {
    /// Go：`native_preparation_ui.go:25`，`func (g *Game) stepNativePreparationCycleTick(rawTick int)`（`r8`，E 組隨
    /// `stepNativePreparationUILifecycle` 補移植）
    pub(crate) fn step_native_preparation_cycle_tick(&mut self, raw_tick: i64) {
        let state = fdicon::advance_native_map_sprite_cycles(
            fdicon::NativeMapSpriteCycleState {
                idle: self.prep_idle_cycle,
                last_timer_tick: self.prep_last_tick,
                ..Default::default()
            },
            raw_tick,
        );
        self.prep_idle_cycle = state.idle;
        self.prep_last_tick = state.last_timer_tick;
    }

    /// Go：`native_preparation_ui.go:36`，`func (g *Game) stepNativePreparationUILifecycle(now time.Time)`
    pub(crate) fn step_native_preparation_ui_lifecycle(&mut self, now: GameInstant) {
        if self.camp.is_none() || self.native_preparation_ui.is_none() || !self.prep_selecting {
            return;
        }
        let Some(n) = self.camp.as_ref().unwrap().node() else {
            return;
        };
        if n.r#type != "preparation" {
            return;
        }
        if self.prep_shot_cycle_frozen {
            return;
        }
        let tick = self.prep_clock.sample(now);
        self.step_native_preparation_cycle_tick(tick);
    }
}

/// Go：`native_preparation_ui.go:50`，`func loadNativePreparationUIAssets() (*nativePreparationUIAssets, error)`
pub(crate) fn load_native_preparation_ui_assets() -> Result<Option<NativePreparationUIAssets>> {
    let roster = fdother::load_native_preparation_assets(separated_asset_path(""))?;
    let status = battle::load_native_item_panel_data_assets(&separated_asset_path(""))?;
    let choices = fdother::load_separated_action_cells(separated_asset_path("ui"))?;
    let shared_ui = fdother::load_separated_item_panel_entries(separated_asset_path("ui"))?;
    // Go `make([]fdother.RawCell, 20)` 後只填 1..19；第 0 格保持零值。
    let mut dialogue = vec![fdother::RawCell::default(); 20];
    for index in 1..=19 {
        let Some(cell) = shared_ui.raw.get(&index) else {
            return Err(game_err!("native preparation UI: separated dialogue is incomplete"));
        };
        dialogue[index as usize] = cell.clone();
    }
    let portraits = load_native_separated_portrait(0x4b)?;
    if portraits.len() <= 3 {
        return Err(game_err!("native preparation UI: DATO#75 mouth frames are incomplete"));
    }
    let slots_box = fdother::load_separated_load_slots_frame(separated_asset_path("ui"))?;
    Ok(Some(NativePreparationUIAssets {
        roster: Some(roster),
        status,
        choices,
        dialogue,
        portrait: portraits[0].clone(),
        portraits,
        slots_box,
    }))
}

impl Game {
    /// Go：`native_preparation_ui.go:187`，`func (g *Game) nativePreparationPromptActive() bool`
    pub(crate) fn native_preparation_prompt_active(&self) -> bool {
        let Some(camp) = self.camp.as_ref() else {
            return false;
        };
        if self.prep_selecting || self.prep_confirm || self.prep_record_slots {
            return false;
        }
        camp.node().is_some_and(|node| node.r#type == "preparation")
    }

    /// Go：`native_preparation_ui.go:195`，`func (g *Game) composeNativePreparationPromptDialogue() ([]byte, bool)`
    pub(crate) fn compose_native_preparation_prompt_dialogue(&self) -> (Vec<u8>, bool) {
        if !self.native_preparation_prompt_active() || self.prep_prompt_source.len() != 320 * 200 {
            return (Vec::new(), false);
        }
        let Some(ui) = self.native_preparation_ui.as_ref() else {
            return (Vec::new(), false);
        };
        match campaign::compose_native_preparation_confirmation_dialogue(
            &self.prep_prompt_source,
            &ui.dialogue,
            &ui.portrait,
        ) {
            Ok(frame) => (frame, true),
            Err(_) => (Vec::new(), false),
        }
    }

    /// Go：`native_preparation_ui.go:208`，`func (g *Game) composeNativePreparationPromptQuestion() ([]byte, bool)`
    pub(crate) fn compose_native_preparation_prompt_question(&self) -> (Vec<u8>, bool) {
        if !self.native_preparation_prompt_active() || self.prep_prompt_source.len() != 320 * 200 {
            return (Vec::new(), false);
        }
        let Some(ui) = self.native_preparation_ui.as_ref() else {
            return (Vec::new(), false);
        };
        // `native_preparation_prompt_active` 成功表示節點存在。
        let node = self.camp.as_ref().unwrap().node().unwrap();
        let (strings, font) = (ui.status.strings.as_ref(), ui.status.font.as_ref());
        let result = if let Some(identity) = self.prep_required_missing {
            campaign::compose_native_preparation_required_message(
                &self.prep_prompt_source,
                &ui.dialogue,
                &ui.portrait,
                strings,
                font,
                identity,
            )
        } else if !node.cancel.is_empty() {
            campaign::compose_native_preparation_departure_question(
                &self.prep_prompt_source,
                &ui.dialogue,
                &ui.portrait,
                strings,
                font,
            )
        } else {
            campaign::compose_native_preparation_record_question(
                &self.prep_prompt_source,
                &ui.dialogue,
                &ui.portrait,
                strings,
                font,
            )
        };
        match result {
            Ok(frame) => (frame, true),
            Err(_) => (Vec::new(), false),
        }
    }

    /// Go：`native_preparation_ui.go:299`，`func (g *Game) beginNativePreparationPromptOpening() bool`
    pub(crate) fn begin_native_preparation_prompt_opening(&mut self) -> bool {
        if self.prep_prompt_source.len() != 320 * 200 {
            return false;
        }
        let (dialogue, ok) = self.compose_native_preparation_prompt_dialogue();
        if !ok {
            return false;
        }
        let (question, ok) = self.compose_native_preparation_prompt_question();
        if !ok {
            return false;
        }
        // 兩個 compose 成功表示 `native_preparation_ui` 存在。
        let choices = &self.native_preparation_ui.as_ref().unwrap().choices;
        let frames = match campaign::native_preparation_confirmation_opening_frames(
            &self.prep_prompt_source,
            &dialogue,
            &question,
            choices,
        ) {
            Ok(frames) if frames.len() == 10 => frames,
            _ => return false,
        };
        self.reset_native_class_ui_pulse();
        let preparation_text = self.snapshot_native_preparation_transition(&self.prep_prompt_source, false, 6, false);
        self.native_class_ui_job = Some(NativeClassUIJob { frames, preparation_text, ..Default::default() });
        true
    }

    /// Go：`native_preparation_ui.go:323`，`func (g *Game) beginNativePreparationPromptClosing(after func()) bool`
    /// Go 的 func 參數 `after` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    ///
    /// 回傳 `false` 時 Go 沒有存下 `after`（呼叫端接著自己呼叫它）；Rust 的 `after` 已移入本函式並被丟棄，
    /// 呼叫端要重建一份再呼叫（見 `native_preparation_input.rs` 的 `close_then`）。
    pub(crate) fn begin_native_preparation_prompt_closing(
        &mut self,
        after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> bool {
        if self.prep_prompt_source.len() != 320 * 200 {
            return false;
        }
        let (dialogue, ok) = self.compose_native_preparation_prompt_dialogue();
        if !ok {
            return false;
        }
        let (question, ok) = self.compose_native_preparation_prompt_question();
        if !ok {
            return false;
        }
        let choices = &self.native_preparation_ui.as_ref().unwrap().choices;
        let frames = match campaign::native_preparation_confirmation_closing_frames(
            &self.prep_prompt_source,
            &dialogue,
            &question,
            choices,
        ) {
            Ok(frames) if frames.len() == 9 => frames,
            _ => return false,
        };
        let preparation_text = self.snapshot_native_preparation_transition(&self.prep_prompt_source, false, 0, true);
        self.native_class_ui_job = Some(NativeClassUIJob {
            frames,
            restore: self.prep_prompt_source.clone(),
            after,
            preparation_text,
            ..Default::default()
        });
        true
    }
}

#[cfg(test)]
mod native_preparation_cycle_test;

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// Go：`native_preparation_ui.go:244`，`func (g *Game) composeNativePreparationPromptFrame() ([]byte, bool)`
    ///
    /// R5 階段 2 r5c。`composeNativePreparationPromptQuestion` 成功表示 `nativePreparationUI` 存在。
    pub(crate) fn compose_native_preparation_prompt_frame(&mut self) -> (Vec<u8>, bool) {
        let (question, ok) = self.compose_native_preparation_prompt_question();
        if !ok {
            return (Vec::new(), false);
        }
        if self.prep_required_missing.is_some() {
            return (question, true);
        }
        let ui = self.native_preparation_ui.as_ref().unwrap();
        let (question, ok) = self.compose_native_confirmation_mouth(&question, &ui.portraits);
        if !ok {
            return (Vec::new(), false);
        }
        match campaign::compose_native_confirmation_choices(
            &question,
            &ui.choices,
            self.prep_confirm_sel,
            self.native_class_ui_pulse / 2,
        ) {
            Ok(frame) => (frame, true),
            Err(_) => (Vec::new(), false),
        }
    }

    /// Go：`native_preparation_ui.go:429`，`func (g *Game) drawNativePreparation(screen *ebiten.Image) bool`
    ///
    /// R5 階段 2 r5c。HD 路徑（`beginModernHDPreparationText` 的擷取、`drawModernHDPreparationRoster`、
    /// `drawModernHDPreparationBackground`，R6）在忠實模式都不成立：擷取要 `modernHDText`，名冊要 HD 主題或 HD 介面，
    /// 背景要 `modernFacility` 場景或 `modernHDText`；任一成立時 `todo!("r6:…")`。`appendModernHDPreparationStableLayers`
    /// 在忠實模式只留下 `appendModernHDPreparationDialogue` 開頭的 `modernHDOverlay = nil`（人數與狀態面板都要
    /// `modernHDText`），這裡照做。`beginModernHDPreparationText` 的收尾在忠實模式不做事。
    pub(crate) fn draw_native_preparation(&mut self, screen: &mut Picture) -> bool {
        if self.draw_native_class_ui_job(screen) {
            return true;
        }
        if self.facility_hd_draw_active() {
            todo!(
                "r6:modern_hd_facility_text.go:beginModernHDPreparationText、modern_hd_preparation_roster.go:drawModernHDPreparationRoster、modern_hd_preparation_background.go:drawModernHDPreparationBackground"
            );
        }
        let (frame, ok) = self.compose_native_preparation_stable_frame();
        if !ok {
            if self.prep_record_slots {
                self.load_err = String::from("preparation record slot assets unavailable");
            }
            return false;
        }
        self.present_native_class_frame(screen, &frame);
        self.modern_hd_overlay = None;
        true
    }

    /// Go：`native_preparation_ui.go:458`，`func (g *Game) composeNativePreparationStableFrame() ([]byte, bool)`
    pub(crate) fn compose_native_preparation_stable_frame(&mut self) -> (Vec<u8>, bool) {
        self.compose_native_preparation_stable_frame_layers(true)
    }

    /// Go：`native_preparation_ui.go:462`，`func (g *Game) composeNativePreparationStableFrameLayers(drawUnits bool) ([]byte, bool)`
    pub(crate) fn compose_native_preparation_stable_frame_layers(&mut self, draw_units: bool) -> (Vec<u8>, bool) {
        self.compose_native_preparation_stable_frame_parts(draw_units, true)
    }

    /// Go：`native_preparation_ui.go:466`，`func (g *Game) composeNativePreparationStableFrameParts(drawUnits, drawCursor bool) ([]byte, bool)`
    ///
    /// R5 階段 2 r5c。第一章只走到出擊前提示（`composeNativePreparationPromptFrame`）；四槽列表、出擊確認與名冊
    /// （`composeNativePreparationRecordSlotsFrame`、`composeNativePreparationConfirmationFrame`、
    /// `composeNativePreparationFrameParts`）在 crate 裡還沒有，`todo!("r8:…")`。
    pub(crate) fn compose_native_preparation_stable_frame_parts(
        &mut self,
        draw_units: bool,
        draw_cursor: bool,
    ) -> (Vec<u8>, bool) {
        if self.prep_record_slots {
            todo!("r8:native_preparation_ui.go:composeNativePreparationRecordSlotsFrame（開場與第一章沒有走到）");
        }
        if self.native_preparation_prompt_active() {
            return self.compose_native_preparation_prompt_frame();
        }
        if self.prep_confirm {
            todo!("r8:native_preparation_ui.go:composeNativePreparationConfirmationFrame（開場與第一章沒有走到）");
        }
        let _ = (draw_units, draw_cursor);
        todo!("r8:native_preparation_ui.go:composeNativePreparationFrameParts（整備名冊；開場與第一章沒有走到）")
    }
}
