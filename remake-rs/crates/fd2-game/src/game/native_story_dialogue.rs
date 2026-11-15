//! Go 來源：`remake/cmd/fd2/native_story_dialogue.go`（586 行）。
//!
//! 骨架：Go 檔的 12 個函式中產生 9 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//!
//! R4 B 組：9 個函式移植完。逐字、翻頁、嘴型與等鍵由 `update` 每個邏輯幀推進（`step_native_story_dialogue_progress`、
//! `step_dialogue_mouth`），不讀牆鐘。
//!
//! R5 階段 2 r5d：繪圖函式 `drawNativeStoryDialogue`、`nativeStoryOpeningFrameIndex` 移植，呈現出口
//! `presentNativeStoryDialogueFrame`（Go 在 HD 檔 `modern_hd_timed_dialogue.go`）只移植忠實模式路徑；畫格擷取打開時
//! 嘴型改取 Go 全域亂數（`Game::go_global_rand`）。

use super::*;

/// R4 B 組新增：故事對白嘴型亂數的固定種子（Go 用套件層 `rand`，種子每次啟動不同；
/// KNOWN_DIFFERENCES `cosmetic-mouth-rng-fixed-seed`）。與遊戲亂數 `rng` 的預設種子錯開。
pub(crate) const STORY_MOUTH_RNG_SEED: i64 = VIRTUAL_CLOCK_ORIGIN.0 + 1;

impl Game {
    /// startNativeDialogueFrames 接回 0x165D6 的開框前聚焦；背景須在聚焦後保存。
    ///
    /// Go：`native_story_dialogue.go:16`，`func (g *Game) startNativeDialogueFrames() error`
    pub(crate) fn start_native_dialogue_frames(&mut self) -> Result<()> {
        if self.dialog.len() != 1 || self.dialog[0].native_dialogue.is_none() {
            return self.prepare_native_dialogue_frames();
        }
        let n = self.dialog[0].native_dialogue.clone().unwrap();
        if !n.has_motion_target_y {
            return Err(GameError::new("native story dialogue: focus provenance is unavailable"));
        }
        if n.motion_target_y == 0 {
            return self.prepare_native_dialogue_frames();
        }
        if self.st.is_none() && self.story_native_map_state.is_none() && self.story_native_map_source.is_some() {
            let source = self.story_native_map_source.clone();
            self.materialize_native_story_map_state(source.as_ref())?;
        }
        let from_battle = self.st.is_some();
        let candidates: &[battle::Unit] = match self.st.as_ref() {
            Some(st) => &st.units,
            None => &self.story_actors,
        };
        let mut speaker: Option<usize> = None;
        match n.control.as_str() {
            "FFED" | "FFEC" => {
                if n.operand >= 0 && (n.operand as usize) < candidates.len() {
                    speaker = Some(n.operand as usize);
                }
            }
            "FFEF" | "FFEE" => {
                for (i, u) in candidates.iter().enumerate() {
                    let (mut identity, mut known) = (u.native_identity, u.has_native_identity);
                    if u.has_native_record_byte8 {
                        (identity, known) = (u.native_record_byte8 as i64, true);
                    }
                    // 正在分派死亡效果的單位在原版還沒被標死（0x1B6B7 只收 +5 bit0 未設的
                    // 記錄），重製端則在扣到 0 時就設了 bit0，所以這一筆照樣算在場。
                    // Go 比較 `u == g.deathProgramDead`：該指標只指向 `g.st.Units`。
                    let is_dying = from_battle && self.death_program_dead == Some(i);
                    let alive = u.has_native_record_byte5 && (u.native_record_byte5 & 1 == 0 || is_dying);
                    if known && identity == n.operand && alive {
                        speaker = Some(i);
                        break;
                    }
                }
            }
            _ => {}
        }
        let speaker = speaker.map(|i| &candidates[i]);
        let Some(speaker) = speaker.filter(|s| s.has_native_map_presentation && self.focus_job.is_none()) else {
            return Err(GameError::new("native story dialogue: focus speaker or owner is unavailable"));
        };
        let (target_x, target_y) = (speaker.native_map_presentation.x as i64, speaker.native_map_presentation.y as i64);
        if self.has_story_native_map_view {
            (self.cur_x, self.cur_y) = (self.story_native_map_view.cursor_x, self.story_native_map_view.cursor_y);
        } else if let Some(st) = self.st.as_ref().filter(|st| st.has_native_map_view_state) {
            (self.cur_x, self.cur_y) = (st.native_map_view_state.cursor_x, st.native_map_view_state.cursor_y);
        } else {
            return Err(GameError::new("native story dialogue: focus view is unavailable"));
        }
        self.native_focus_endpoint(target_x, target_y)?;
        let pending = std::mem::take(&mut self.dialog);
        self.focus_job = Some(FocusUnitJob {
            target_x,
            target_y,
            native_view: true,
            then: Some(Box::new(move |g: &mut Game| {
                g.dialog = pending;
                g.native_map_vga = Vec::new();
                (g.dlg_shown, g.dlg_phase, g.dlg_t) = (DLG_NONE, 0, 0);
                if let Err(err) = g.prepare_native_dialogue_frames() {
                    g.dialog = Vec::new();
                    g.load_err = format!("native story dialogue: {err}");
                }
            })),
            ..Default::default()
        });
        Ok(())
    }

    /// Go：`native_story_dialogue.go:93`，`func (g *Game) prepareNativeDialogueFrames() error`
    ///
    /// R4 B 組：HD 介面開啟時的兩個分支（`prepareModernHDStoryDialogueChrome`、`freezeModernHDDialogueChrome`）
    /// 停在 `todo!("R6: …")`；非繁中語系的逐字合成（`composeLocalizedNativeDialogueProgressiveFrames`，依 TTF 量寬）
    /// 與第 28 章戰後的視圖繼承是 `r8`。
    pub(crate) fn prepare_native_dialogue_frames(&mut self) -> Result<()> {
        self.native_dialogue_frames = Vec::new();
        self.native_dialogue_progressive = Vec::new();
        self.native_dialogue_mouth_open = Vec::new();
        self.native_dialogue_progress = -1;
        self.native_dialogue_opening = Vec::new();
        self.native_dialogue_closing = Vec::new();
        self.native_dialogue_closing_t = 0;
        self.native_dialogue_closing_live = false;
        self.reset_native_story_dialogue_mouth();
        self.native_dialogue_modern_portrait = None;
        self.native_dialogue_speaking_frame = 0;
        self.native_dialogue_portraits = Vec::new();
        self.native_dialogue_layout = None;
        self.native_dialogue_chrome = None;
        self.native_dialogue_chrome_bounds = BTreeMap::new();
        self.native_dialogue_glyph_steps = Vec::new();
        if self.dialog.last().is_none_or(|d| d.native_dialogue.is_none()) {
            if !self.dialog.is_empty() && self.modern_hd_interface_enabled() {
                todo!("R6: modern_hd_dialogue_chrome.go:prepareModernHDStoryDialogueChrome（HD 介面）")
            }
            return Ok(());
        }
        if self.dialog.len() != 1 {
            return Err(GameError::new("native story dialogue: one typed utterance per beat is required"));
        }
        if self.native_class_ui.is_none() {
            self.native_class_ui = load_native_class_ui_assets()?;
        }
        if self.locale_id == "zh-Hant" && (self.native_battle_font.is_none() || self.native_battle_glyphs.is_empty()) {
            // Go 比較 `g.nativeBattleGlyphs == nil`；Rust 的 map 不分 nil 與空。
            let (font, glyphs) = load_native_battle_name_assets()?;
            (self.native_battle_font, self.native_battle_glyphs) = (font, glyphs);
        }
        let native0 = self.dialog[0].native_dialogue.as_ref().unwrap();
        let ch28_inherited = self.camp.as_ref().is_some_and(|c| c.node_id() == "postbattle_ch28_persist")
            && native0.source_dat == "FDTXT_028"
            && native0.string_index == 7;
        if self.st.as_ref().is_some_and(|st| !st.has_native_map_view_state) && ch28_inherited {
            todo!("R8: native_story_dialogue.go:materializeInheritedNativeDialogueView（第一章沒有執行到）")
        }
        if self.st.is_none() && self.story_native_map_state.is_none() && self.story_native_map_source.is_some() {
            let source = self.story_native_map_source.clone();
            self.materialize_native_story_map_state(source.as_ref())?;
        }
        if self.st.is_none() && self.story_native_map_state.is_some() {
            self.compose_native_story_map_frame()?;
        } else if self.native_map_vga.len() != 320 * 200 {
            self.compose_native_map_frame()?;
        }
        if self.native_map_vga.len() != 320 * 200 {
            return Err(GameError::new("native story dialogue: indexed map/font/frame assets are unavailable"));
        }
        let dl = self.dialog[0].clone();
        if let Some(set) = self.modern_story_portraits.as_ref() {
            let Some(portrait) = set.portraits.get(&dl.speaker) else {
                return Err(game_err!(
                    "native story dialogue: modern portrait for speaker {} is unavailable",
                    dl.speaker
                ));
            };
            self.native_dialogue_modern_portrait = Some(portrait.clone());
        }
        let portraits = load_native_separated_portrait(dl.speaker);
        let portraits = match portraits {
            Ok(p) if p.len() >= 4 => p,
            Ok(p) => {
                // Go 的 `%v` 印 nil error 是 `<nil>`。
                return Err(game_err!(
                    "native story dialogue: speaker portrait {} is unavailable (frames={}): <nil>",
                    dl.speaker,
                    p.len()
                ));
            }
            Err(err) => {
                return Err(game_err!(
                    "native story dialogue: speaker portrait {} is unavailable (frames=0): {err}",
                    dl.speaker
                ));
            }
        };
        let native = dl.native_dialogue.as_ref().unwrap();
        let layout = campaign::NativeDialogueLayout {
            source_dat: native.source_dat.clone(),
            string_index: native.string_index,
            utterance: native.utterance,
            control: native.control.clone(),
            operand: native.operand,
            pages: native.pages.clone(),
            glyph_ids: native.glyph_ids.clone(),
            glyph_pages: native.glyph_pages.clone(),
        };
        let want_upper = layout.control == "FFEF" || layout.control == "FFED";
        if dl.upper != Some(want_upper) {
            return Err(GameError::new("native story dialogue: control and upper/lower binding disagree"));
        }
        let class_ui = self.native_class_ui.as_ref().expect("Go：loadNativeClassUIAssets 成功時不回 nil");
        let opening =
            campaign::compose_native_story_dialogue_opening_frames(&self.native_map_vga, &class_ui.dialogue, &layout)
                .map_err(|err| game_err!("native story dialogue: opening frames: {err}"))?;
        if !native.has_motion_target_y {
            return Err(GameError::new("native story dialogue: closing motion target provenance is unavailable"));
        }
        let mut motion_target_y = native.motion_target_y;
        // raw ch27_post 沒有留下五句各自的說話者螢幕座標；沿用玩家游標會把
        // 下框滑到無關甚至越界的位置。99% 玩家可見模式保留五階段收框與背景
        // 還原，只省略這段無來源的額外滑動。
        if ch28_inherited {
            motion_target_y = 0;
        }
        let (mut visible_cursor_x, mut visible_cursor_y) = (0, 0);
        if motion_target_y != 0 {
            if let Some(st) = self.st.as_ref().filter(|st| st.has_native_map_view_state) {
                visible_cursor_x = st.native_map_view_state.visible_cursor_x;
                visible_cursor_y = st.native_map_view_state.visible_cursor_y;
            } else if self.has_story_native_map_view {
                visible_cursor_x = self.story_native_map_view.visible_cursor_x;
                visible_cursor_y = self.story_native_map_view.visible_cursor_y;
            } else {
                return Err(GameError::new("native story dialogue: closing cursor/view provenance is unavailable"));
            }
        }
        let closing = campaign::compose_native_story_dialogue_closing_frames(
            &self.native_map_vga,
            &class_ui.dialogue,
            &layout,
            motion_target_y,
            visible_cursor_x,
            visible_cursor_y,
        )
        .map_err(|err| game_err!("native story dialogue: closing frames: {err}"))?;
        let pages = layout.pages.len();
        let mut frames: Vec<Vec<u8>> = vec![Vec::new(); pages];
        let mut progressive: Vec<Vec<Vec<u8>>> = vec![Vec::new(); pages];
        let mut mouth_open: Vec<Vec<u8>> = vec![Vec::new(); pages];
        let mut glyph_steps: Vec<Vec<bool>> = vec![Vec::new(); pages];
        let zh_hant = self.locale_id == "zh-Hant";
        for page in 0..pages {
            if !zh_hant {
                todo!(
                    "R8: native_story_dialogue_locale.go:composeLocalizedNativeDialogueProgressiveFrames（非繁中，依 TTF 量寬）"
                )
            }
            progressive[page] = campaign::compose_native_story_dialogue_progressive_frames(
                &self.native_map_vga,
                &class_ui.dialogue,
                &portraits[0],
                class_ui.font.as_ref(),
                Some(&self.native_battle_glyphs),
                &layout,
                page as i64,
            )
            .map_err(|err| game_err!("native story dialogue: page {} progressive frames: {err}", page))?;
            match campaign::native_story_dialogue_glyph_steps(&layout, page as i64) {
                Ok(steps) if steps.len() == progressive[page].len() => glyph_steps[page] = steps,
                Ok(_) => {
                    return Err(GameError::new("native story dialogue: glyph timeline mismatch: <nil>"));
                }
                Err(err) => {
                    return Err(game_err!("native story dialogue: glyph timeline mismatch: {err}"));
                }
            }
            frames[page] = progressive[page].last().cloned().expect("Go：progressive 為空時索引 panic");
            mouth_open[page] =
                campaign::compose_native_story_dialogue_mouth_frame(&frames[page], &portraits[3], &layout)
                    .map_err(|err| game_err!("native story dialogue: page {} mouth frame: {err}", page))?;
        }
        // Freeze the native grid geometry once per utterance, including cursor-only
        // closing tails. No timer, glyph progression or save state is advanced here.
        // Go 走 map 字面值（順序隨機）；結果是以 phase 為鍵的 map，只有多個錯誤同時成立時回報的那一個可能不同。
        let mut bounds: BTreeMap<campaign::NativeDialogueFramePhase, Vec<Rectangle>> = BTreeMap::new();
        for (phase, count) in [
            (campaign::NATIVE_DIALOGUE_OPENING, opening.len()),
            (campaign::NATIVE_DIALOGUE_STEADY, 1),
            (campaign::NATIVE_DIALOGUE_CLOSING, closing.len()),
        ] {
            for index in 0..count {
                let rect = campaign::native_story_dialogue_frame_bounds(
                    &class_ui.dialogue,
                    &layout.control,
                    phase,
                    index as i64,
                )?;
                bounds.entry(phase).or_default().push(rect);
            }
        }
        if self.modern_hd_interface_enabled() {
            todo!("R6: modern_hd_dialogue_chrome.go:freezeModernHDDialogueChrome（HD 介面）")
        }
        (self.native_dialogue_chrome, self.native_dialogue_chrome_bounds) = (None, bounds);
        self.native_dialogue_frames = frames;
        self.native_dialogue_progressive = progressive;
        self.native_dialogue_mouth_open = mouth_open;
        self.native_dialogue_opening = opening;
        self.native_dialogue_closing = closing;
        self.native_dialogue_portraits = portraits;
        self.native_dialogue_layout = Some(layout);
        self.native_dialogue_glyph_steps = glyph_steps;
        Ok(())
    }
}

/// Go：`native_story_dialogue.go:309`，`func loadNativeSeparatedPortrait(speaker int) ([]dato.Frame, error)`
pub(crate) fn load_native_separated_portrait(speaker: i64) -> Result<Vec<dato::Frame>> {
    Ok(dato::load_separated_resource(separated_asset_path("portraits"), speaker)?)
}

impl Game {
    /// Go：`native_story_dialogue.go:353`，`func (g *Game) stepNativeStoryDialogueProgress()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn step_native_story_dialogue_progress(&mut self) {
        if self.native_dialogue_closing_live {
            if self.native_dialogue_closing_t < self.native_dialogue_closing.len() as i64 - 1 {
                self.native_dialogue_closing_t += 1;
                return;
            }
            self.finish_native_story_dialogue_closing();
            return;
        }
        if self.dlg_phase != 0
            || self.dialog.last().is_none_or(|d| d.native_dialogue.is_none())
            || self.dlg_scroll_t > 0
            || self.dlg_page < 0
            || self.dlg_page >= self.native_dialogue_progressive.len() as i64
        {
            return;
        }
        let page = self.dlg_page as usize;
        let frames = self.native_dialogue_progressive[page].len() as i64;
        if frames > 0 && self.native_dialogue_progress < frames - 1 {
            self.native_dialogue_progress += 1;
            let progress = self.native_dialogue_progress;
            let glyph_step = self
                .native_dialogue_glyph_steps
                .get(page)
                .and_then(|steps| steps.get(progress as usize))
                .copied()
                .unwrap_or(false);
            if glyph_step {
                self.native_dialogue_speaking_half += 1;
                if self.native_dialogue_speaking_half == 2 {
                    self.native_dialogue_speaking_half = 0;
                    self.native_dialogue_speaking_cycle = (self.native_dialogue_speaking_cycle + 1) % 4;
                    self.native_dialogue_speaking_frame = self.native_dialogue_speaking_cycle;
                    if self.native_dialogue_speaking_frame == 3 {
                        self.native_dialogue_speaking_frame = 1;
                    }
                }
            }
        }
    }

    /// Go：`native_story_dialogue.go:386`，`func (g *Game) beginNativeStoryDialogueClosing() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn begin_native_story_dialogue_closing(&mut self) -> bool {
        if self.native_dialogue_closing_live
            || self.dialog.last().is_none_or(|d| d.native_dialogue.is_none())
            || self.native_dialogue_closing.is_empty()
        {
            return false;
        }
        self.native_dialogue_closing_live = true;
        self.reset_native_story_dialogue_mouth();
        // campInput 位於本幀一般 step 之後；立即發布 frame0，避免先出現空白幀。
        self.native_dialogue_closing_t = 0;
        true
    }

    /// Go：`native_story_dialogue.go:398`，`func (g *Game) resetNativeStoryDialogueMouth()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn reset_native_story_dialogue_mouth(&mut self) {
        self.native_dialogue_mouth_ready = false;
        self.native_dialogue_arrow_ticks = 0;
        self.mouth_state = dato::MouthState::default();
        (self.mouth_open, self.mouth_timer) = (false, 0);
    }

    /// Go：`native_story_dialogue.go:408`，`func (g *Game) nativeStoryDialogueAtInputWait() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_story_dialogue_at_input_wait(&self) -> bool {
        if self.native_dialogue_closing_live
            || self.dlg_phase != 0
            || self.dlg_scroll_t != 0
            || self.dialog.last().is_none_or(|d| d.native_dialogue.is_none())
            || self.dlg_page < 0
            || self.dlg_page >= self.native_dialogue_progressive.len() as i64
            || self.dlg_page >= self.native_dialogue_mouth_open.len() as i64
            || self.native_dialogue_mouth_open[self.dlg_page as usize].len() != 320 * 200
        {
            return false;
        }
        let frames = &self.native_dialogue_progressive[self.dlg_page as usize];
        !frames.is_empty() && self.native_dialogue_progress >= frames.len() as i64 - 1
    }

    /// Go：`native_story_dialogue.go:419`，`func (g *Game) stepDialogueMouth()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    ///
    /// R4 B 組：Go 的 `rand.Intn(30)` 改抽 `story_mouth_rng`（見 [`Game::story_mouth_intn30`]）；推進節奏只看
    /// 邏輯幀 `self.frame` 的奇偶，與牆鐘無關。
    pub(crate) fn step_dialogue_mouth(&mut self) {
        let Some(last) = self.dialog.last() else {
            self.reset_native_story_dialogue_mouth();
            return;
        };
        if last.native_dialogue.is_none() {
            if self.frame % 2 != 0 {
                return;
            }
            let mut random_mod30 = 0;
            if self.mouth_state.open {
                random_mod30 = self.story_mouth_intn30();
            }
            if let Ok(next) = self.mouth_state.tick(random_mod30) {
                (self.mouth_state, self.mouth_open, self.mouth_timer) = (next, next.frame_index() == 3, next.countdown);
            }
            return;
        }
        if !self.native_story_dialogue_at_input_wait() {
            self.reset_native_story_dialogue_mouth();
            return;
        }
        if !self.native_dialogue_mouth_ready {
            self.mouth_state = dato::MouthState { countdown: self.story_mouth_intn30() + 2, ..Default::default() };
            self.native_dialogue_mouth_ready = true;
            (self.mouth_open, self.mouth_timer) = (false, self.mouth_state.countdown);
            return;
        }
        self.native_dialogue_arrow_ticks += 1;
        if self.frame % 2 != 0 {
            return;
        }
        let mut random_mod30 = 0;
        if self.mouth_state.open {
            random_mod30 = self.story_mouth_intn30();
        }
        if let Ok(next) = self.mouth_state.tick(random_mod30) {
            (self.mouth_state, self.mouth_open, self.mouth_timer) = (next, next.frame_index() == 3, next.countdown);
        }
    }

    /// R4 B 組新增：Go `rand.Intn(30)`（套件層全域來源）。惰性以 [`STORY_MOUTH_RNG_SEED`] 建立
    /// `story_mouth_rng`，與遊戲亂數 `rng` 分開（Go 的全域來源也不碰 `g.rng`）。
    ///
    /// R5 階段 2 r5d：`go_global_rand` 有值（畫格擷取打開）時改取它的 `Intn(30)`，與 Go 全域
    /// `rand.Intn` 同一序列（Go 1.22 的全域 `Intn` 是 `(*Rand).Intn`，`Seed(4)` 後的來源與
    /// `NewSource(4)` 相同）。
    pub(crate) fn story_mouth_intn30(&mut self) -> i64 {
        if let Some(global) = self.go_global_rand.as_mut() {
            return global.intn(30);
        }
        self.story_mouth_rng.get_or_insert_with(|| battle::GoMathRand::new(STORY_MOUTH_RNG_SEED)).intn(30)
    }

    /// Go：`native_story_dialogue.go:460`，`func (g *Game) finishNativeStoryDialogueClosing()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn finish_native_story_dialogue_closing(&mut self) {
        if !self.native_dialogue_closing_live {
            return;
        }
        self.native_dialogue_closing_live = false;
        self.reset_native_story_dialogue_mouth();
        self.native_dialogue_closing_t = 0;
        self.dialog.pop();
        (self.dlg_page, self.dlg_scroll_t, self.dlg_scroll_from, self.native_dialogue_progress) = (0, 0, 0, -1);
        (self.dlg_shown, self.dlg_phase, self.dlg_t) = (DLG_NONE, 0, 0);
        if self.native_level_up_dialogue.is_some() {
            self.finish_native_level_up_dialogue();
            return;
        }
        if self.battle_event.is_some() {
            self.advance_battle_event();
            return;
        }
        if self.camp.as_ref().and_then(|c| c.node()).is_some_and(|n| n.r#type == "cutscene") {
            self.beat_advance();
        }
    }
}

#[cfg(test)]
mod native_story_dialogue_test;

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// Go：`native_story_dialogue.go:485`，`func (g *Game) drawNativeStoryDialogue(screen *ebiten.Image) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    ///
    /// R5 階段 2 r5d：逐行照 Go。`g.nativeClassUI` 為 nil 時 Go 解參考 panic，Rust 同樣 panic；`g.nativeDialogueLayout`
    /// 為 nil 時 Go 的 `Validate` 回「lacks source provenance」，Rust 以空的 `NativeDialogueLayout`（`source_dat` 為空，
    /// `validate` 回同一個錯誤）代入。HD 分支（`modernHDStoryTextFrame`、`nativeDialogueModernPortrait`、`modernHDText`）在
    /// 忠實模式不成立，HD 時 `todo!("r6:…")`。
    pub(crate) fn draw_native_story_dialogue(&mut self, screen: &mut Picture) -> bool {
        if self.dialog.last().is_none_or(|d| d.native_dialogue.is_none()) {
            return false;
        }
        if self.native_dialogue_closing_live {
            let mut index = self.native_dialogue_closing_t;
            if index >= self.native_dialogue_closing.len() as i64 {
                index = self.native_dialogue_closing.len() as i64 - 1;
            }
            if index >= 0 {
                let frame = self.native_dialogue_closing[index as usize].clone();
                if let Err(err) = self.present_native_story_dialogue_frame(
                    screen,
                    &frame,
                    campaign::NATIVE_DIALOGUE_CLOSING,
                    index,
                    None,
                    Rectangle::default(),
                ) {
                    self.load_err = err.to_string();
                }
            }
            return true;
        }
        // caller-specific 開框、穩定頁與收框都只呈現完整預建的 indexed frame；
        // 任一階段缺失時不回退到正規化對話介面。
        if self.dlg_phase == 2 {
            let index = native_story_opening_frame_index(self.dlg_t, self.native_dialogue_opening.len() as i64);
            if index >= 0 {
                let frame = self.native_dialogue_opening[index as usize].clone();
                if let Err(err) = self.present_native_story_dialogue_frame(
                    screen,
                    &frame,
                    campaign::NATIVE_DIALOGUE_OPENING,
                    index,
                    None,
                    Rectangle::default(),
                ) {
                    self.load_err = err.to_string();
                }
            }
            return true;
        }
        if self.dlg_phase != 0 {
            return true;
        }
        if self.dlg_page < 0 || self.dlg_page >= self.native_dialogue_progressive.len() as i64 {
            return true;
        }
        let page = self.dlg_page as usize;
        let count = self.native_dialogue_progressive[page].len() as i64;
        if count == 0 {
            return true;
        }
        let mut progress = self.native_dialogue_progress;
        if progress < 0 {
            progress = 0;
        }
        if progress >= count {
            progress = count - 1;
        }
        let mut frame = self.native_dialogue_progressive[page][progress as usize].clone();
        // Go 的 nil `*NativeDialogueLayout`：`Validate` 對 nil 與空 SourceDAT 回同一個錯誤。
        let no_layout = campaign::NativeDialogueLayout::default();
        if self.locale_id == "zh-Hant"
            && progress > 0
            && progress < count - 1
            && self.native_dialogue_speaking_frame > 0
            && self.native_dialogue_portraits.len() == 4
        {
            let layout = self.native_dialogue_layout.as_ref().unwrap_or(&no_layout);
            match campaign::compose_native_story_dialogue_mouth_frame(
                &frame,
                &self.native_dialogue_portraits[self.native_dialogue_speaking_frame as usize],
                layout,
            ) {
                Ok(composed) => frame = composed,
                Err(err) => {
                    self.load_err = format!("native story dialogue: {err}");
                    return true;
                }
            }
        }
        if self.mouth_open && progress == count - 1 && self.dlg_page < self.native_dialogue_mouth_open.len() as i64 {
            frame = self.native_dialogue_mouth_open[page].clone();
        }
        // 最後一頁的箭頭：0x1E292 最後一欄的 FFFD 也等鍵（第七章 r4 seq 972 MHP 那一頁有箭頭）。
        let last_page_arrow =
            self.native_level_up_dialogue.as_ref().is_some_and(|d| d.await_last_key && d.close_timer == 0);
        if self.native_story_dialogue_at_input_wait()
            && (self.dlg_page + 1 < self.native_dialogue_progressive.len() as i64 || last_page_arrow)
        {
            let cells = &self.native_class_ui.as_ref().expect("drawNativeStoryDialogue：nativeClassUI 為 nil").dialogue;
            let layout = self.native_dialogue_layout.as_ref().unwrap_or(&no_layout);
            match campaign::compose_native_story_dialogue_wait_arrow(
                &frame,
                cells,
                layout,
                (self.native_dialogue_arrow_ticks / 6) % 2,
            ) {
                Ok(composed) => frame = composed,
                Err(err) => {
                    self.load_err = format!("native story dialogue: {err}");
                    return true;
                }
            }
        }
        // Go `frame, hdText := g.modernHDStoryTextFrame(frame, progress)`：modernHDText 為 nil 時原樣回傳、沒有 HD 文字。
        if self.modern_hd_text.is_some() {
            todo!("r6:modern_hd_story_text.go:modernHDStoryTextFrame（HD 故事文字層）");
        }
        if self.native_dialogue_modern_portrait.is_some() {
            todo!(
                "r6:native_story_dialogue.go:drawNativeStoryDialogue nativeDialogueModernPortrait（HD 頭像疊層與 modernHDOverlay）"
            );
        }
        if let Err(err) = self.present_native_story_dialogue_frame(
            screen,
            &frame,
            campaign::NATIVE_DIALOGUE_STEADY,
            0,
            None,
            Rectangle::default(),
        ) {
            self.load_err = err.to_string();
            return true;
        }
        // Go `if g.modernHDText != nil { runs = append(runs, hdText...) }`：上面已對 HD 停下。
        true
    }

    /// Go：`modern_hd_timed_dialogue.go:65`，`func (g *Game) presentNativeStoryDialogueFrame(screen *ebiten.Image, frame []byte, phase campaign.NativeDialogueFramePhase, index int, portrait image.Image, rect image.Rectangle) error`
    ///
    /// R5 階段 2 r5d 新增（Go 在 HD 檔，`PORTING.json` 狀態仍是 `r6`）：`drawNativeStoryDialogue` 的唯一呈現出口。
    /// 只移植忠實模式的路徑：`modernHDWorldVisible` 為 false 時 `presentNativeClassFrame`；`portrait` 只有 HD 頭像會傳，
    /// `modernHDText` 只在 HD 有值。HD 計時對白（`modernHDWorldVisible`）、RGBA 頭像疊層與 HD 框線面板 `todo!("r6:…")`。
    pub(crate) fn present_native_story_dialogue_frame(
        &mut self,
        screen: &mut Picture,
        frame: &[u8],
        phase: campaign::NativeDialogueFramePhase,
        index: i64,
        portrait: Option<Arc<Picture>>,
        rect: Rectangle,
    ) -> Result<()> {
        // phase、index 只給 HD 計時對白與框線面板用（下面 r6 分支）。
        let _ = (phase, index);
        if let Some(portrait) = portrait.as_ref()
            && (portrait.width != rect.max.x - rect.min.x || portrait.height != rect.max.y - rect.min.y)
        {
            return Err(game_err!("native RGBA overlay geometry is invalid"));
        }
        if self.modern_hd_world_visible {
            todo!(
                "r6:modern_hd_timed_dialogue.go:presentNativeStoryDialogueFrame（HD 計時對白 modernHDTimedDialogueEligible）"
            );
        }
        self.present_native_class_frame(screen, frame);
        if portrait.is_some() {
            todo!("r6:modern_hd_timed_dialogue.go:presentNativeStoryDialogueFrame（RGBA 頭像疊層）");
        }
        if self.modern_hd_text.is_some() && self.native_dialogue_layout.is_some() {
            todo!("r6:modern_hd_timed_dialogue.go:presentNativeStoryDialogueFrame（HD 對白框線面板）");
        }
        Ok(())
    }
}

/// Go：`native_story_dialogue.go:339`，`func nativeStoryOpeningFrameIndex(tick, count int) int`
pub(crate) fn native_story_opening_frame_index(tick: i64, count: i64) -> i64 {
    if count <= 0 {
        return -1;
    }
    let mut index = tick - 1;
    if index < 0 {
        index = 0;
    }
    if index >= count {
        index = count - 1;
    }
    index
}
