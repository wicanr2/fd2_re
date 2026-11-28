//! Go 來源：`remake/cmd/fd2/native_story_dialogue.go`（586 行）。
//!
//! 骨架：Go 檔的 12 個函式中產生 9 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//!
//! R4 B 組：9 個函式移植完。逐字、翻頁、嘴型與等鍵由 `update` 每個邏輯幀推進（`step_native_story_dialogue_progress`、
//! `step_dialogue_mouth`），不讀牆鐘。
//!
//! R5 階段 2 r5d：繪圖函式 `drawNativeStoryDialogue`、`nativeStoryOpeningFrameIndex` 移植，呈現出口
//! `presentNativeStoryDialogueFrame` 依 `modern_hd_timed_dialogue.go` 分流普通對白與凍結背景的計時對白。畫格擷取打開時
//! 嘴型改取 Go 全域亂數（`Game::go_global_rand`）。

use super::*;

/// R4 B 組新增：故事對白嘴型亂數的固定種子（Go 用套件層 `rand`，種子每次啟動不同；
/// KNOWN_DIFFERENCES `cosmetic-mouth-rng-fixed-seed`）。與遊戲亂數 `rng` 的預設種子錯開。
pub(crate) const STORY_MOUTH_RNG_SEED: i64 = VIRTUAL_CLOCK_ORIGIN.0 + 1;

const DIALOGUE_FRAME_SIZE: usize = 320 * 200;

pub(crate) fn legacy_dialogue_lines(line: &battle::DialogLine) -> Vec<String> {
    // Go `dlgWrap` reserves the existing portrait rectangle before counting glyphs.
    let width = if line.speaker >= 32 { 640.0 - 16.0 - 80.0 * 2.1 - 8.0 - 32.0 } else { 10.0 + 620.0 - 16.0 - 216.0 };
    let per_line = (width / (FONT_SIZE * 1.7)) as usize;
    let text: Vec<_> = std::iter::once('『')
        .chain(line.text.chars().map(|ch| match ch {
            ',' => '，',
            '!' => '！',
            ':' => '：',
            '?' => '？',
            ';' => '；',
            '.' => '。',
            '(' => '（',
            ')' => '）',
            _ => ch,
        }))
        .chain(std::iter::once('』'))
        .collect();
    text.chunks(per_line).map(|row| row.iter().collect()).collect()
}

impl Game {
    /// Go `draw` 的舊式可編輯對白。章界讀檔會經劇本的正式 setup 走到此處。
    pub(crate) fn draw_legacy_story_dialogue(&mut self, screen: &mut Picture) {
        let Some(line) = self.dialog.last() else { return };
        let upper = self.dlg_upper.unwrap_or(self.dlg_shown >= 32);
        let by = if upper { 4.0 } else { 198.0 };
        let scale = match self.dlg_phase {
            1 => 1.0 - self.dlg_t as f64 / DLG_ANIM_FRAMES as f64,
            2 => self.dlg_t as f64 / DLG_ANIM_FRAMES as f64,
            _ => 1.0,
        };
        if scale <= 0.0 {
            return;
        }
        let top = by + (1.0 - scale) * 99.0;
        let mut op = DrawImageOptions::default();
        if let Some(box_image) = &self.dlg_box {
            op.geo_m.scale(2.0, 2.0 * scale);
            op.geo_m.translate(10.0, top);
            screen.draw_image(box_image, &op);
        } else {
            let mut panel = Picture::new_image(1, 1);
            panel.fill(ColorRgba { r: 0x2c, g: 0x44, b: 0x84, a: 0xf2 });
            op.geo_m.scale(620.0, 198.0 * scale);
            op.geo_m.translate(10.0, top);
            screen.draw_image(&panel, &op);
        }
        if self.hd_title_menu.is_some()
            && let Some(chrome) = self.native_dialogue_chrome.as_mut()
        {
            let bottom = top + 198.0 * scale;
            let size = Point { x: 620 * 3, y: (bottom * 3.0).round() as i64 - (top * 3.0).round() as i64 };
            if let Some(border) = chrome.gold_border_for_size(size) {
                let extent = rect(10, top.round() as i64, 630, bottom.round() as i64);
                self.window_dialogue_border = Some((border, extent));
            }
        }
        if self.dlg_phase != 0 {
            return;
        }
        let gradient = self.dlg_grad.get_or_insert_with(|| {
            let mut picture = Picture::new_image(1, 198);
            for y in 0..198 {
                let d = (16 * y / 197) as u8;
                picture.set(0, y, ColorRgba { r: 40 + d, g: 69 + d, b: 138 + d, a: 255 });
            }
            Arc::new(picture)
        });
        let mut op = DrawImageOptions::default();
        op.geo_m.scale(604.0, 182.0 / 198.0);
        op.geo_m.translate(18.0, by + 8.0);
        screen.draw_image(gradient, &op);

        let modern = self.modern_story_portraits.as_ref().and_then(|art| art.portraits.get(&line.speaker));
        let original = self.portraits.get(&line.speaker).and_then(|frames| {
            let index = if self.mouth_open && frames.len() > 3 { 3 } else { 0 };
            frames.get(index).and_then(Option::as_ref)
        });
        let portrait = (if self.modern_story_portraits.is_some() {
            modern.and_then(|frame| frame.image.as_ref())
        } else {
            original
        })
        .filter(|image| image.width > 0 && image.height > 0);
        if self.modern_story_portraits.is_some() && portrait.is_none() {
            return;
        }
        let hx = if upper { 456.0 } else { 16.0 };
        let hy = by + 15.0;
        if let Some(portrait) = portrait {
            let mirrored;
            let picture = if upper {
                portrait.as_ref()
            } else {
                let columns = portrait.width as usize;
                let pixels: Vec<_> = portrait
                    .read_pixels()
                    .as_chunks::<4>()
                    .0
                    .chunks_exact(columns)
                    .flat_map(|row| row.iter().rev().copied())
                    .collect();
                let mut flipped = Picture::new_image(portrait.width, portrait.height);
                flipped.write_pixels(pixels.as_flattened());
                mirrored = flipped;
                &mirrored
            };
            let mut op = DrawImageOptions::default();
            op.geo_m.scale(2.1, 2.1);
            op.geo_m.translate(hx, hy);
            screen.draw_image(picture, &op);
            if let Some(frame) = modern.filter(|frame| frame.hd_image.is_some()) {
                self.modern_hd_overlay = Some(ModernHDOverlay {
                    portrait: Some(frame.clone()),
                    rect: rect(hx as i64, hy as i64, hx as i64 + 168, hy as i64 + 168),
                    mirror_x: !upper,
                });
            }
        }
        let tx = if upper || portrait.is_none() { 32.0 } else { 216.0 };
        let ty = by + if upper { 46.0 } else { 24.0 };
        let lines = legacy_dialogue_lines(line);
        let Some(font) = self.font.as_mut() else { return };
        let mut draw_page = |page: i64, dy: f64| {
            for (row, line) in lines.iter().skip((page.max(0) * 3) as usize).take(3).enumerate() {
                let y = ty + row as f64 * 38.0 + dy;
                if (by + 8.0..=by + 184.0).contains(&y) {
                    font.draw(screen, line, tx, y, 1.7, ColorRgba { r: 0xf0, g: 0xf4, b: 0xff, a: 255 });
                }
            }
        };
        if self.dlg_scroll_t > 0 {
            let progress = 1.0 - self.dlg_scroll_t as f64 / DLG_SCROLL_FRAMES as f64;
            draw_page(self.dlg_scroll_from, -progress * 114.0);
            draw_page(self.dlg_page, (1.0 - progress) * 114.0);
        } else {
            draw_page(self.dlg_page, 0.0);
        }
    }
}

#[derive(Debug)]
pub(crate) struct ModernStoryDialogueMasks {
    opening: Vec<Vec<bool>>,
    closing: Vec<Vec<bool>>,
    pages: Vec<Vec<bool>>,
}

fn dialogue_marker_masks(compose: impl Fn(&[u8]) -> Result<Vec<Vec<u8>>>) -> Result<Vec<Vec<bool>>> {
    let low = compose(&vec![0; DIALOGUE_FRAME_SIZE])?;
    let high = compose(&vec![255; DIALOGUE_FRAME_SIZE])?;
    if low.is_empty() || low.len() != high.len() {
        return Err(game_err!("modern HD dialogue: marker replay changed frame count"));
    }
    low.into_iter()
        .zip(high)
        .map(|(a, b)| {
            if a.len() != DIALOGUE_FRAME_SIZE || b.len() != DIALOGUE_FRAME_SIZE {
                return Err(game_err!("modern HD dialogue: marker replay frame is incomplete"));
            }
            Ok(a.into_iter().zip(b).map(|(a, b)| a == b).collect())
        })
        .collect()
}

fn freeze_story_dialogue_masks(
    pages: usize,
    opening: impl Fn(&[u8]) -> Result<Vec<Vec<u8>>>,
    closing: impl Fn(&[u8]) -> Result<Vec<Vec<u8>>>,
    page: impl Fn(&[u8], usize) -> Result<Vec<Vec<u8>>>,
) -> Result<ModernStoryDialogueMasks> {
    let opening = dialogue_marker_masks(opening)?;
    let closing = dialogue_marker_masks(closing)?;
    let mut page_masks = Vec::with_capacity(pages);
    for index in 0..pages {
        let masks = dialogue_marker_masks(|base| page(base, index))?;
        let mut union = vec![false; DIALOGUE_FRAME_SIZE];
        for mask in masks {
            for (saved, covered) in union.iter_mut().zip(mask) {
                *saved |= covered;
            }
        }
        page_masks.push(union);
    }
    Ok(ModernStoryDialogueMasks { opening, closing, pages: page_masks })
}

fn masked_dialogue_picture(frame: &[u8], mask: &[bool], palette: &[ColorRgba]) -> Result<Picture> {
    if frame.len() != DIALOGUE_FRAME_SIZE || mask.len() != frame.len() || palette.len() != 256 {
        return Err(game_err!("modern HD dialogue: incomplete pixels"));
    }
    let pixels: Vec<_> = frame
        .iter()
        .zip(mask)
        .map(|(&index, &covered)| {
            if covered {
                let c = palette[index as usize];
                [c.r, c.g, c.b, 255]
            } else {
                [0; 4]
            }
        })
        .collect();
    let mut picture = Picture::new_image(320, 200);
    picture.write_pixels(pixels.as_flattened());
    Ok(picture)
}

impl Game {
    fn story_dialogue_mask(&self, phase: campaign::NativeDialogueFramePhase, index: i64) -> Option<&[bool]> {
        let masks = self.native_dialogue_hd_masks.as_ref()?;
        let (frames, index) = match phase {
            campaign::NATIVE_DIALOGUE_OPENING => (&masks.opening, index),
            campaign::NATIVE_DIALOGUE_CLOSING => (&masks.closing, index),
            campaign::NATIVE_DIALOGUE_STEADY if self.dlg_scroll_t == 0 => (&masks.pages, self.dlg_page),
            _ => return None,
        };
        frames.get(index as usize).map(Vec::as_slice)
    }
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
    /// 停在 `todo!("R6: …")`。R6 非繁中對白依 TTF 字寬排版，沿共用索引畫格逐字顯示。第 28 章戰後的視圖繼承是 `r8`。
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
        self.native_dialogue_hd_masks = None;
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
            self.materialize_inherited_native_dialogue_view()?;
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
            progressive[page] = if zh_hant {
                let frames = campaign::compose_native_story_dialogue_progressive_frames(
                    &self.native_map_vga,
                    &class_ui.dialogue,
                    &portraits[0],
                    class_ui.font.as_ref(),
                    Some(&self.native_battle_glyphs),
                    &layout,
                    page as i64,
                )?;
                let steps = campaign::native_story_dialogue_glyph_steps(&layout, page as i64)?;
                if steps.len() != frames.len() {
                    return Err(GameError::new("native story dialogue: glyph timeline mismatch"));
                }
                glyph_steps[page] = steps;
                frames
            } else {
                compose_localized_native_dialogue_progressive_frames(
                    &self.native_map_vga,
                    &class_ui.dialogue,
                    &portraits[0],
                    self.font.as_mut(),
                    &layout,
                    page,
                )?
            };
            frames[page] = progressive[page]
                .last()
                .cloned()
                .ok_or_else(|| game_err!("native story dialogue: empty page {page}"))?;
            mouth_open[page] = if zh_hant {
                campaign::compose_native_story_dialogue_mouth_frame(&frames[page], &portraits[3], &layout)
                    .map_err(|err| game_err!("native story dialogue: page {} mouth frame: {err}", page))?
            } else {
                frames[page].clone()
            };
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
        self.native_dialogue_chrome_bounds = bounds;
        self.native_dialogue_frames = frames;
        self.native_dialogue_progressive = progressive;
        self.native_dialogue_mouth_open = mouth_open;
        self.native_dialogue_opening = opening;
        self.native_dialogue_closing = closing;
        self.native_dialogue_portraits = portraits;
        self.native_dialogue_layout = Some(layout);
        self.native_dialogue_mask_motion = (motion_target_y, visible_cursor_x, visible_cursor_y);
        self.native_dialogue_glyph_steps = glyph_steps;
        if self.modern_story_portraits.as_ref().is_some_and(|theme| theme.has_hd) {
            self.native_dialogue_hd_masks = self.build_current_story_dialogue_masks().ok().flatten();
        }
        Ok(())
    }

    pub(crate) fn build_current_story_dialogue_masks(&self) -> Result<Option<ModernStoryDialogueMasks>> {
        let Some(layout) = &self.native_dialogue_layout else {
            return Ok(None);
        };
        let ui = self.native_class_ui.as_ref().ok_or_else(|| game_err!("HD dialogue: missing interface"))?;
        let portrait =
            self.native_dialogue_portraits.first().ok_or_else(|| game_err!("HD dialogue: missing portrait"))?;
        let (motion_y, cursor_x, cursor_y) = self.native_dialogue_mask_motion;
        let masks = freeze_story_dialogue_masks(
            layout.pages.len(),
            |base| Ok(campaign::compose_native_story_dialogue_opening_frames(base, &ui.dialogue, layout)?),
            |base| {
                Ok(campaign::compose_native_story_dialogue_closing_frames(
                    base,
                    &ui.dialogue,
                    layout,
                    motion_y,
                    cursor_x,
                    cursor_y,
                )?)
            },
            |base, page| {
                if self.locale_id != "zh-Hant" {
                    return compose_localized_native_dialogue_progressive_frames(
                        base,
                        &ui.dialogue,
                        portrait,
                        self.font.clone().as_mut(),
                        layout,
                        page,
                    );
                }
                Ok(campaign::compose_native_story_dialogue_progressive_frames(
                    base,
                    &ui.dialogue,
                    portrait,
                    ui.font.as_ref(),
                    Some(&self.native_battle_glyphs),
                    layout,
                    page as i64,
                )?)
            },
        )?;
        Ok(Some(masks))
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
            if progress == frames - 1 && self.cutscene_log && self.locale_id != "zh-Hant" {
                eprintln!("[cutscene] locale page ready: {} page={page}", self.locale_id);
            }
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
    /// `validate` 回同一個錯誤）代入。穩定頁可疊現代靜態頭像。
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
        let portrait = self.native_dialogue_modern_portrait.clone();
        let portrait_rect = if portrait.is_some() {
            let layout = self.dialog.last().unwrap().native_dialogue.as_ref().unwrap();
            let Ok(rect) = campaign::native_story_dialogue_portrait_rect(&layout.control) else { return true };
            rect
        } else {
            Rectangle::default()
        };
        if let Err(err) = self.present_native_story_dialogue_frame(
            screen,
            &frame,
            campaign::NATIVE_DIALOGUE_STEADY,
            0,
            portrait.as_ref().and_then(|frame| frame.image.clone()),
            portrait_rect,
        ) {
            self.load_err = err.to_string();
            return true;
        }
        if portrait.as_ref().is_some_and(|frame| frame.hd_image.is_some()) {
            let Rectangle { min, max } = portrait_rect;
            self.modern_hd_overlay = Some(ModernHDOverlay {
                portrait,
                rect: rect(min.x * 2, min.y * 2, max.x * 2, max.y * 2),
                mirror_x: false,
            });
        }
        // Go `if g.modernHDText != nil { runs = append(runs, hdText...) }`：上面已對 HD 停下。
        true
    }

    /// Go：`modern_hd_timed_dialogue.go:65`，`func (g *Game) presentNativeStoryDialogueFrame(screen *ebiten.Image, frame []byte, phase campaign.NativeDialogueFramePhase, index int, portrait image.Image, rect image.Rectangle) error`
    ///
    /// R5 階段 2 r5d 新增（Go 在 HD 檔，`PORTING.json` 狀態仍是 `r6`）：`drawNativeStoryDialogue` 的唯一呈現出口。
    /// 原生索引背景透過 `presentNativeClassFrame` 呈現，再疊上候選主題提供的 80×80 頭像。
    pub(crate) fn present_native_story_dialogue_frame(
        &mut self,
        screen: &mut Picture,
        frame: &[u8],
        phase: campaign::NativeDialogueFramePhase,
        index: i64,
        portrait: Option<Arc<Picture>>,
        rect: Rectangle,
    ) -> Result<()> {
        if let Some(portrait) = portrait.as_ref()
            && (portrait.width != rect.max.x - rect.min.x || portrait.height != rect.max.y - rect.min.y)
        {
            return Err(game_err!("native RGBA overlay geometry is invalid"));
        }
        let hd_world = self.modern_hd_world_visible && self.modern_hd_map_gpu.is_some();
        let timed = self
            .native_level_up_dialogue
            .as_ref()
            .filter(|dialogue| hd_world && dialogue.hd_world.is_some() && self.modern_hd_timed_dialogue_eligible());
        let mask = if let Some(dialogue) = timed {
            let mask = dialogue
                .hd_coverage
                .get(&phase)
                .and_then(|masks| masks.get(index as usize))
                .ok_or_else(|| game_err!("modern HD dialogue: coverage unavailable"))?;
            Some(mask.as_slice())
        } else if hd_world {
            self.story_dialogue_mask(phase, index)
        } else {
            None
        };
        if let (Some(mask), Some(ui)) = (mask, &self.native_class_ui) {
            let picture = masked_dialogue_picture(frame, mask, &ui.palette)?;
            if let Some(text) = &mut self.modern_hd_text {
                text.runs.clear();
            }
            let mut op = DrawImageOptions::default();
            op.geo_m.scale(2.0, 2.0);
            screen.draw_image(&picture, &op);
            if timed.is_some() {
                self.modern_hd_timed_dialogue_visible = true;
                self.modern_hd_timed_dialogue_phase = phase;
                self.modern_hd_timed_dialogue_index = index;
            }
        } else {
            self.present_native_class_frame(screen, frame);
        }
        if let Some(portrait) = portrait {
            let mut op = DrawImageOptions::default();
            op.geo_m.scale(2.0, 2.0);
            op.geo_m.translate((rect.min.x * 2) as f64, (rect.min.y * 2) as f64);
            screen.draw_image(&portrait, &op);
        }
        if self.hd_title_menu.is_some()
            && self.native_dialogue_layout.is_some()
            && let (Some(area), Some(chrome)) = (
                self.native_dialogue_chrome_bounds.get(&phase).and_then(|bounds| bounds.get(index as usize)),
                self.native_dialogue_chrome.as_mut(),
            )
            && let Some(border) = chrome.gold_border(*area)
        {
            let extent = canvas::rect(area.min.x * 2, area.min.y * 2, area.max.x * 2, area.max.y * 2);
            self.window_dialogue_border = Some((border, extent));
        }
        if self.modern_hd_text.is_some() && self.native_dialogue_layout.is_some() {
            todo!("r6:modern_hd_timed_dialogue.go:presentNativeStoryDialogueFrame（HD 對白框線面板）");
        }
        Ok(())
    }
}

impl Game {
    /// materializeInheritedNativeDialogueView 保留戰鬥控制器的現行視圖。raw ch27_post 在呼叫對話前沒有另寫
    /// 鏡頭／游標，因此只有完整圖塊對齊的 controller 狀態可以成為 indexed source；這不是章節專用的座標後備值。
    ///
    /// Go：`native_story_dialogue.go:336`，`func (g *Game) materializeInheritedNativeDialogueView() error`
    pub(crate) fn materialize_inherited_native_dialogue_view(&mut self) -> Result<()> {
        let (Some(_), Some(m)) = (self.st.as_ref(), self.m.as_ref().filter(|m| m.tile_w > 0 && m.tile_h > 0)) else {
            return Err(game_err!("native story dialogue: inherited battle view is unavailable"));
        };
        if self.cam_x < 0.0 || self.cam_y < 0.0 {
            return Err(game_err!("native story dialogue: inherited battle camera is negative"));
        }
        // Go `int(g.camX)/g.m.TileW`：先截成整數再整除。
        let (camera_x, camera_y) = (self.cam_x as i64 / m.tile_w, self.cam_y as i64 / m.tile_h);
        if (camera_x * m.tile_w) as f64 != self.cam_x || (camera_y * m.tile_h) as f64 != self.cam_y {
            return Err(game_err!("native story dialogue: inherited battle camera is not tile-aligned"));
        }
        let view = battle::NativeMapViewState {
            camera_x,
            camera_y,
            cursor_x: self.cur_x,
            cursor_y: self.cur_y,
            visible_cursor_x: self.cur_x - camera_x,
            visible_cursor_y: self.cur_y - camera_y,
        };
        if let Err(err) = self.st.as_mut().unwrap().materialize_native_map_view_state(view) {
            return Err(game_err!("native story dialogue: inherited battle view: {err}"));
        }
        Ok(())
    }
}

/// Go：`native_story_dialogue.go:359`，`func nativeStoryOpeningFrameIndex(tick, count int) int`
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
