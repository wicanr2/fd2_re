//! Go 來源：`remake/cmd/fd2/player_toolbar.go`（324 行）。
//!
//! R5 階段 4（r5w）只移植觸控座標換算 `playerTouchCoordinates`：第一章的地圖觸控（`main.go:9095`）經過它。
//! 工具列本體（`modernHDPresentation` 的顯示、點擊、標籤與繪製）在 R6，見 `PORTING.md`「R5 階段 4：r5w」。

use super::*;

/// Go：`player_toolbar.go:15`，`const playerToolbarHeight = 24`。
pub(crate) const PLAYER_TOOLBAR_HEIGHT: i64 = 24;

#[derive(Clone, Copy)]
enum ToolbarAction {
    Art(bool),
    Interface,
    Audio,
    Locale,
    Music,
    Font,
    Speed,
    Fullscreen,
    Controls,
    Posters,
    Chapter,
}

const BUTTONS: [(i64, i64, ToolbarAction); 12] = [
    (8, 78, ToolbarAction::Art(false)),
    (82, 130, ToolbarAction::Art(true)),
    (134, 226, ToolbarAction::Interface),
    (230, 282, ToolbarAction::Audio),
    (286, 314, ToolbarAction::Locale),
    (318, 376, ToolbarAction::Font),
    (380, 424, ToolbarAction::Speed),
    (428, 482, ToolbarAction::Music),
    (486, 543, ToolbarAction::Fullscreen),
    (547, 582, ToolbarAction::Posters),
    (586, 609, ToolbarAction::Controls),
    (613, 636, ToolbarAction::Chapter),
];

const GUIDE_ROWS: usize = 7;
const GUIDE_DETAILS: Rectangle = rect(424, 44, 574, 69);
const GUIDE_PREVIOUS: Rectangle = rect(66, 324, 166, 350);
const GUIDE_NEXT: Rectangle = rect(172, 324, 272, 350);
const GUIDE_CLOSE: Rectangle = rect(474, 324, 574, 350);

impl Game {
    pub(crate) fn update_toolbar(&mut self, input: &TickInput, focused: bool) -> bool {
        if self.chapter_guide_open {
            self.update_chapter_guide(input, focused);
            return true;
        }
        let height = if self.toolbar_ticks > 0 { PLAYER_TOOLBAR_HEIGHT } else { 4 };
        let zone = |(x, y)| (0..LOGICAL_W).contains(&x) && (0..height).contains(&y);
        let near = focused && zone(input.cursor);
        if self.font.is_some()
            && (near || self.posters.menu_open() || input.touches.iter().any(|&(_, x, y)| zone((x, y))))
        {
            self.toolbar_ticks = 90 * self.effective_player_speed();
        } else {
            self.toolbar_ticks = (self.toolbar_ticks - 1).max(0);
        }
        self.player_toolbar_touch_blocked = self.toolbar_ticks > 0;
        if !self.player_toolbar_touch_blocked {
            return false;
        }
        let point = if focused && input.mouse_left_just_pressed {
            Some(input.cursor)
        } else {
            input.just_pressed_touches.iter().copied().find(|&(_, y)| (0..24).contains(&y))
        };
        if let Some((x, y)) = point
            && (3..21).contains(&y)
            && let Some((_, _, action)) = BUTTONS.iter().find(|&&(left, right, _)| (left..right).contains(&x))
        {
            self.activate_toolbar(*action);
        }
        focused && (0..LOGICAL_W).contains(&input.cursor.0) && (0..24).contains(&input.cursor.1)
    }

    fn activate_toolbar(&mut self, action: ToolbarAction) {
        match action {
            ToolbarAction::Art(hd) => {
                if hd != self.modern_story_portraits.is_some() {
                    self.cycle_theme_art();
                }
            }
            ToolbarAction::Interface => self.cycle_hd_title_interface(),
            ToolbarAction::Audio => {
                self.bgm_source = if self.bgm_source == "mt32" { "fm" } else { "mt32" }.into();
                let mut settings = load_settings();
                settings.bgm_source = self.bgm_source.clone();
                save_settings(&settings);
                if !self.bgm_cur.is_empty() {
                    let track = std::mem::take(&mut self.bgm_cur);
                    self.play_bgm(&track);
                }
            }
            ToolbarAction::Music => {
                let Ok(options) = self.current_native_system_options().toggle(0) else { return };
                self.native_system_options = Some(options);
                if options.music_enabled() {
                    let track = std::mem::take(&mut self.native_system_bgm_track);
                    self.play_bgm(&track);
                } else {
                    let track = self.bgm_cur.clone();
                    self.stop_bgm();
                    self.native_system_bgm_track = track;
                }
            }
            ToolbarAction::Font => self.cycle_font_scale(),
            ToolbarAction::Speed => self.cycle_player_speed(),
            ToolbarAction::Fullscreen => self.fullscreen_toggle_requested = true,
            ToolbarAction::Controls => self.open_controls_panel(),
            ToolbarAction::Locale => {
                if self.dialog.is_empty() && self.player_poster_controls_allowed() {
                    self.cycle_player_locale();
                }
            }
            ToolbarAction::Posters if self.player_poster_controls_allowed() => self.posters.toggle_menu(),
            ToolbarAction::Posters => {}
            ToolbarAction::Chapter => match self.chapter_toolbar_kind() {
                "battle" => {
                    self.chapter_guide_open = true;
                    self.chapter_guide_detail = false;
                    self.chapter_guide_page = 0;
                }
                "town" if self.camp.as_mut().is_some_and(|camp| camp.confirm_native_town_secret(5)) => {
                    self.enter_node();
                }
                _ => {}
            },
        }
    }

    fn toolbar_labels(&self) -> [String; 12] {
        let interface = if self.hd_title_menu.is_some() { "HD" } else { "原版" };
        let audio = if self.bgm_source == "mt32" { "MT-32" } else { "FM" };
        [
            "原版".into(),
            if self.modern_theme_loading.is_some() { "載入中".into() } else { "HD".into() },
            format!("介面 {interface}"),
            format!("音源 {audio}"),
            match self.locale_id.as_str() {
                "zh-Hans" => "简中",
                "ja" => "日本語",
                "en" => "EN",
                _ => "繁中",
            }
            .into(),
            format!("字級 {:.0}%", self.font_scale * 100.0),
            format!("{}×", self.player_speed),
            if self.current_native_system_options().music_enabled() { "音樂 開" } else { "音樂 關" }.into(),
            "全螢幕".into(),
            "海報".into(),
            "按鍵".into(),
            match self.chapter_toolbar_kind() {
                "battle" => "事件",
                "town" => "祕店",
                _ => "",
            }
            .into(),
        ]
    }

    pub(crate) fn toolbar_render_key(&self) -> String {
        format!(
            "toolbar {:?} {} {} {}",
            self.toolbar_labels(),
            self.modern_story_portraits.is_some(),
            self.posters.render_key(),
            if self.chapter_guide_open {
                format!("{} {}", self.chapter_guide_detail, self.chapter_guide_page)
            } else {
                String::new()
            }
        )
    }

    pub(crate) fn draw_toolbar(&mut self, surface: &mut Picture) {
        let labels = self.toolbar_labels();
        let Some(mut font) = self.font.clone() else { return };
        font.set_user_scale(1.0);
        surface.sub_image_mut(rect(0, 0, LOGICAL_W * 3, 72)).fill(ColorRgba { r: 17, g: 24, b: 37, a: 255 });
        for ((left, right, action), label) in BUTTONS.into_iter().zip(labels) {
            if label.is_empty() {
                continue;
            }
            let selected = matches!(action, ToolbarAction::Art(hd) if hd == self.modern_story_portraits.is_some())
                || matches!(action, ToolbarAction::Posters if self.posters.menu_open());
            let color = if selected {
                ColorRgba { r: 57, g: 77, b: 108, a: 255 }
            } else {
                ColorRgba { r: 31, g: 45, b: 66, a: 255 }
            };
            surface.sub_image_mut(rect(left * 3, 9, right * 3, 63)).fill(color);
            let scale = if right - left < 26 { 1.45 } else { 1.6 };
            let width = font.width(&label, scale);
            let x = ((left + right) as f64 * 3.0 - width) / 2.0;
            font.draw(surface, &label, x, 18.0, scale, ColorRgba { r: 245, g: 245, b: 245, a: 255 });
        }
        if self.posters.menu_open() {
            self.posters.draw_menu(surface, &mut font);
        }
    }

    fn chapter_toolbar_kind(&self) -> &'static str {
        if !self.title_phase.is_empty() || !self.load_err.is_empty() {
            return "";
        }
        let Some(camp) = &self.camp else { return "" };
        let Some(node) = camp.node() else { return "" };
        if node.r#type == "battle" && self.st.is_some() && self.sc.is_some() {
            return "battle";
        }
        if node.r#type == "town"
            && self.fade.is_none()
            && self.native_town_departure.is_none()
            && let Some(gate) = &node.native_secret_gate
            && camp.c.as_ref().and_then(|graph| graph.nodes.get(&gate.to)).is_some_and(|shop| shop.r#type == "shop")
            && self.native_shop_ui.as_ref().is_some_and(|ui| ui.shops.contains_key(&5) && ui.portraits.contains_key(&5))
        {
            return "town";
        }
        ""
    }

    fn chapter_guide_lines(&self) -> Vec<String> {
        let Some(sc) = &self.sc else { return Vec::new() };
        let fired: BTreeSet<_> = sc.fired_events().into_iter().map(|event| event.index).collect();
        let mut rows = Vec::new();
        for (index, event) in sc.events.iter().enumerate() {
            let kind = if event.r#do.iter().any(|action| action.r#type == "spawn_group") {
                "增援登場"
            } else {
                match event.trigger.as_str() {
                    "on_battle_start" => "戰鬥開始",
                    "on_turn_end" => "回合事件",
                    "on_unit_death" => "擊倒事件",
                    _ => "戰場事件",
                }
            };
            let status = if fired.contains(&(index as i64)) {
                "已觸發"
            } else if event.once {
                "待觸發"
            } else {
                "可重複"
            };
            rows.push(format!("{status}　{kind}"));
            if self.chapter_guide_detail {
                let mut details = Vec::new();
                let mut positions = BTreeSet::new();
                if let Some(when) = &event.when {
                    if when.turn > 0 {
                        details.push(format!("第 {} 回合", when.turn));
                    }
                    if !when.unit_dead.is_empty() {
                        details.push(format!("{}倒下", when.unit_dead));
                    }
                }
                for action in &event.r#do {
                    if let Some([x, y]) = action.grid {
                        details.push(format!("地圖格 ({x},{y})"));
                    }
                    if action.r#type == "join_party" {
                        details.push("角色加入".into());
                    }
                    if action.r#type == "spawn_group"
                        && let Some(state) = &self.st
                    {
                        for unit in state.units.iter().chain(&state.roster) {
                            if action.groups.contains(&unit.group) && unit.has_native_position_record {
                                let pos = unit.native_position_record;
                                let line = format!("{}　預定登場格 ({},{})", unit.cls_name, pos.x_word, pos.y_word);
                                if positions.insert(line.clone()) {
                                    details.push(line);
                                }
                            }
                        }
                    }
                }
                rows.extend(details.into_iter().map(|detail| format!("　{detail}")));
            }
        }
        if self.chapter_guide_detail
            && let Some(treasures) = self.st.as_ref().and_then(|state| state.treasures.as_ref())
        {
            for (cell, reward) in treasures {
                let name = match reward.kind.as_str() {
                    "gold" => Some(format!("{} 金幣", reward.value)),
                    "item" => self.locale_entities.as_ref().and_then(|catalog| catalog.item_name(reward.value).ok()),
                    _ => None,
                };
                if let Some(name) = name {
                    rows.push(format!("寶物 ({},{})　{name}", cell.x, cell.y));
                }
            }
        }
        rows
    }

    fn update_chapter_guide(&mut self, input: &TickInput, focused: bool) {
        if self.chapter_toolbar_kind() != "battle" || input.is_just_pressed(Key::Escape) {
            self.chapter_guide_open = false;
            return;
        }
        let point = if focused && input.mouse_left_just_pressed {
            Some(input.cursor)
        } else {
            input.just_pressed_touches.first().copied()
        };
        let Some((x, y)) = point else { return };
        let hit = |area: Rectangle| (area.min.x..area.max.x).contains(&x) && (area.min.y..area.max.y).contains(&y);
        if hit(GUIDE_CLOSE) {
            self.chapter_guide_open = false;
        } else if hit(GUIDE_DETAILS) {
            self.chapter_guide_detail = !self.chapter_guide_detail;
            self.chapter_guide_page = 0;
        } else if hit(GUIDE_PREVIOUS) {
            self.chapter_guide_page = self.chapter_guide_page.saturating_sub(1);
        } else if hit(GUIDE_NEXT) && (self.chapter_guide_page + 1) * GUIDE_ROWS < self.chapter_guide_lines().len() {
            self.chapter_guide_page += 1;
        }
    }

    pub(crate) fn draw_chapter_guide(&self, surface: &mut Picture) {
        let (Some(sc), Some(mut font)) = (&self.sc, self.font.clone()) else { return };
        font.set_user_scale(1.0);
        let ink = ColorRgba { r: 245, g: 245, b: 245, a: 255 };
        surface.sub_image_mut(rect(144, 114, 1776, 1098)).fill(ColorRgba { r: 17, g: 24, b: 37, a: 255 });
        font.draw(surface, &format!("第 {} 章　關卡事件", sc.chapter), 204.0, 126.0, 2.5, ink);
        let goal = if sc.native_result_initial_code.is_some() {
            "目標：完成本關指定事件"
        } else {
            "目標：擊倒所有已上場的敵軍"
        };
        font.draw(surface, goal, 204.0, 266.0, 2.0, ink);
        let hint = if sc.native_result_rules.is_empty() {
            "提前清場可能略過未觸發的事件"
        } else {
            "本關另有特殊結算條件"
        };
        font.draw(surface, hint, 204.0, 332.0, 2.0, ink);
        let rows = self.chapter_guide_lines();
        let start = self.chapter_guide_page * GUIDE_ROWS;
        for (i, row) in rows.iter().skip(start).take(GUIDE_ROWS).enumerate() {
            let caption: String = row.chars().take(36).collect();
            font.draw(surface, &caption, 204.0, 408.0 + i as f64 * 74.0, 2.0, ink);
        }
        let mut button = |area: Rectangle, caption: &str| {
            surface
                .sub_image_mut(rect(area.min.x * 3, area.min.y * 3, area.max.x * 3, area.max.y * 3))
                .fill(ColorRgba { r: 57, g: 77, b: 108, a: 255 });
            let x = ((area.min.x + area.max.x) as f64 * 3.0 - font.width(caption, 1.8)) / 2.0;
            font.draw(surface, caption, x, (area.min.y * 3 + 18) as f64, 1.8, ink);
        };
        button(GUIDE_DETAILS, if self.chapter_guide_detail { "收起細節" } else { "顯示細節" });
        if start > 0 {
            button(GUIDE_PREVIOUS, "上一頁");
        }
        if start + GUIDE_ROWS < rows.len() {
            button(GUIDE_NEXT, "下一頁");
        }
        button(GUIDE_CLOSE, "關閉");
    }

    /// Go：`player_toolbar.go:26`，`func (g *Game) playerTouchCoordinates(x, y int) (int, int, bool)`
    pub(crate) fn player_touch_coordinates(&self, x: i64, y: i64) -> (i64, i64, bool) {
        let x = x - self.player_viewport_left;
        let block = &self.player_overlay_block;
        // Go `image.Pt(x, y).In(r)`：Min ≤ p < Max（空矩形不含任何點）。
        let in_block = block.min.x <= x && x < block.max.x && block.min.y <= y && y < block.max.y;
        // Go `x < 0 || x >= logicalW || y < top || y >= top+logicalH`。
        if !(0..LOGICAL_W).contains(&x)
            || !(self.player_viewport_top..self.player_viewport_top + LOGICAL_H).contains(&y)
            || self.player_toolbar_touch_blocked && y < PLAYER_TOOLBAR_HEIGHT
            || in_block
        {
            return (0, 0, false);
        }
        (x, y - self.player_viewport_top, true)
    }
}

#[cfg(test)]
#[path = "player_toolbar/player_toolbar_test.rs"]
mod player_toolbar_test;
