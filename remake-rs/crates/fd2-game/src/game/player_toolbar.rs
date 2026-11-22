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
    Music,
    Font,
    Speed,
    Fullscreen,
    Controls,
    Posters,
}

const BUTTONS: [(i64, i64, ToolbarAction); 10] = [
    (8, 78, ToolbarAction::Art(false)),
    (82, 130, ToolbarAction::Art(true)),
    (134, 226, ToolbarAction::Interface),
    (230, 314, ToolbarAction::Audio),
    (318, 376, ToolbarAction::Font),
    (380, 424, ToolbarAction::Speed),
    (428, 482, ToolbarAction::Music),
    (486, 543, ToolbarAction::Fullscreen),
    (547, 582, ToolbarAction::Posters),
    (586, 632, ToolbarAction::Controls),
];

impl Game {
    pub(crate) fn update_toolbar(&mut self, input: &TickInput, focused: bool) -> bool {
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
            ToolbarAction::Font => {
                let previous = self.font_scale;
                let index = FONT_SCALES.iter().position(|&scale| scale == previous).unwrap_or(2);
                self.apply_font_scale(FONT_SCALES[(index + 1) % FONT_SCALES.len()]);
                if self.hd_title_menu.is_some() && self.enable_hd_title_interface().is_err() {
                    self.apply_font_scale(previous);
                    return;
                }
                let mut settings = load_settings();
                settings.font_scale = self.font_scale;
                save_settings(&settings);
            }
            ToolbarAction::Speed => {
                let speed = normalize_player_speed(self.player_speed) % 4 + 1;
                self.set_player_speed_at(speed, self.virtual_now);
                let mut settings = load_settings();
                settings.game_speed = speed;
                save_settings(&settings);
            }
            ToolbarAction::Fullscreen => self.fullscreen_toggle_requested = true,
            ToolbarAction::Controls => self.open_controls_panel(),
            ToolbarAction::Posters if self.player_poster_controls_allowed() => self.posters.toggle_menu(),
            ToolbarAction::Posters => {}
        }
    }

    fn toolbar_labels(&self) -> [String; 10] {
        let interface = if self.hd_title_menu.is_some() { "HD" } else { "原版" };
        let audio = if self.bgm_source == "mt32" { "MT-32" } else { "FM" };
        [
            "原版".into(),
            if self.modern_theme_loading.is_some() { "載入中".into() } else { "HD".into() },
            format!("介面 {interface}"),
            format!("音源 {audio}"),
            format!("字級 {:.0}%", self.font_scale * 100.0),
            format!("{}×", self.player_speed),
            if self.current_native_system_options().music_enabled() { "音樂 開" } else { "音樂 關" }.into(),
            "全螢幕".into(),
            "海報".into(),
            "按鍵".into(),
        ]
    }

    pub(crate) fn toolbar_render_key(&self) -> String {
        format!(
            "toolbar {:?} {} {}",
            self.toolbar_labels(),
            self.modern_story_portraits.is_some(),
            self.posters.render_key()
        )
    }

    pub(crate) fn draw_toolbar(&mut self, surface: &mut Picture) {
        let labels = self.toolbar_labels();
        let Some(mut font) = self.font.clone() else { return };
        font.set_user_scale(1.0);
        surface.sub_image_mut(rect(0, 0, LOGICAL_W * 3, 72)).fill(ColorRgba { r: 17, g: 24, b: 37, a: 255 });
        for ((left, right, action), label) in BUTTONS.into_iter().zip(labels) {
            let selected = matches!(action, ToolbarAction::Art(hd) if hd == self.modern_story_portraits.is_some())
                || matches!(action, ToolbarAction::Posters if self.posters.menu_open());
            let color = if selected {
                ColorRgba { r: 57, g: 77, b: 108, a: 255 }
            } else {
                ColorRgba { r: 31, g: 45, b: 66, a: 255 }
            };
            surface.sub_image_mut(rect(left * 3, 9, right * 3, 63)).fill(color);
            let width = font.width(&label, 1.6);
            let x = ((left + right) as f64 * 3.0 - width) / 2.0;
            font.draw(surface, &label, x, 18.0, 1.6, ColorRgba { r: 245, g: 245, b: 245, a: 255 });
        }
        if self.posters.menu_open() {
            self.posters.draw_menu(surface, &mut font);
        }
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
