//! Go 來源：`remake/cmd/fd2/player_poster_controls.go`（190 行）。
//!
//! 海報選單、設定與獨立於遊戲速度的淡入。

use std::path::PathBuf;
use std::time::Instant;

use super::*;

pub const POSTER_SIDE_WIDTH: u32 = 132;
const MODES: [&str; 3] = ["off", "hd", "classic"];
const DURATIONS: [i64; 4] = [0, 300, 600, 1200];
const SETS: [&str; 3] = ["關閉", "HD 第一組", "原稿第二組"];
const MENU: Rectangle = rect(362, 26, 634, 157);
const SLIDER: Rectangle = rect(380, 84, 620, 104);
const REPLAY: Rectangle = rect(374, 134, 496, 154);
const CLOSE: Rectangle = rect(500, 134, 622, 154);

fn choices<const N: usize>(top: i64, height: i64, stride: i64) -> [Rectangle; N] {
    std::array::from_fn(|index| {
        let x = 374 + index as i64 * stride;
        rect(x, top, x + stride - 4, top + height)
    })
}

fn contains(r: Rectangle, (x, y): (i64, i64)) -> bool {
    (r.min.x..r.max.x).contains(&x) && (r.min.y..r.max.y).contains(&y)
}

#[derive(Clone, Copy)]
enum SliderInput {
    Mouse,
    Touch(u64),
}

pub(crate) struct PlayerPosters {
    settings: PlayerPosterSettings,
    pair: Option<PlayerPosterPair>,
    masked: Option<[Arc<Picture>; 2]>,
    started: Option<Instant>,
    open: bool,
    slider: Option<SliderInput>,
    failed: bool,
    pub(crate) native_scenery: Option<Arc<Picture>>,
    pub(crate) native_dialogue_visible: bool,
    hd_scenery: Option<PosterScenery>,
}

struct PosterScenery {
    base: Arc<Picture>,
    layers: Vec<ModernHDTerrainDraw>,
    clip: Rectangle,
    image: Arc<Picture>,
}

impl Default for PlayerPosters {
    fn default() -> Self {
        Self {
            settings: default_player_posters(),
            pair: None,
            masked: None,
            started: None,
            open: false,
            slider: None,
            failed: false,
            native_scenery: None,
            native_dialogue_visible: false,
            hd_scenery: None,
        }
    }
}

impl PlayerPosters {
    pub(crate) fn load() -> Self {
        let settings = normalize_player_posters(load_settings().poster.as_ref());
        let mut result = Self::default();
        (result.settings.edge, result.settings.fade_ms) = (settings.edge, settings.fade_ms);
        result.apply_mode(&settings.set, false);
        result
    }

    fn save(&self) {
        let mut settings = load_settings();
        // 冷啟動因素材缺失回退時，保留磁碟上的圖組偏好。
        let mut poster = self.settings.clone();
        if self.failed && self.pair.is_none() {
            poster.set = normalize_player_posters(settings.poster.as_ref()).set;
        }
        settings.poster = Some(poster);
        save_settings(&settings);
    }

    fn apply_mode(&mut self, mode: &str, persist: bool) {
        if self.settings.set == mode && !self.failed {
            return;
        }
        let pair = if mode == "off" {
            None
        } else {
            let root = std::env::var_os("FD2_MODERN_THEME_PACK")
                .filter(|path| !path.is_empty())
                .map_or_else(|| PathBuf::from(asset_path("assets/themes/modern")), PathBuf::from);
            match PlayerPosterPair::load(&PathBuf::from(asset_path("assets/themes/modern/catalog.json")), &root) {
                Ok(pair) => Some(pair),
                Err(err) => {
                    self.failed = true;
                    eprintln!("海報不可用：{err}");
                    return;
                }
            }
        };
        self.failed = false;
        self.pair = pair;
        if !self.active() {
            self.native_scenery = None;
            self.hd_scenery = None;
        }
        self.masked = None;
        self.settings.set = mode.to_owned();
        self.started = Some(Instant::now());
        if persist {
            self.save();
        }
    }

    pub(crate) fn active(&self) -> bool {
        self.pair.is_some()
    }

    pub(crate) fn scenery(&mut self, hd: Option<&ModernHDMapFrame>, native: bool) -> Option<(Arc<Picture>, Rectangle)> {
        if !self.active() {
            return None;
        }
        if let Some(frame) = hd
            && let Some(base) = &frame.poster_base
        {
            let clip = if rect_empty(&frame.viewport) { modern_hd_map_viewport() } else { frame.viewport };
            if self
                .hd_scenery
                .as_ref()
                .is_none_or(|cache| cache.clip != clip || cache.base != *base || cache.layers != frame.terrain)
            {
                self.hd_scenery = Some(PosterScenery {
                    base: base.clone(),
                    layers: frame.terrain.clone(),
                    clip,
                    image: Arc::new(compose_poster_terrain(base, &frame.terrain, clip)),
                });
            }
            return Some((self.hd_scenery.as_ref()?.image.clone(), clip));
        }
        if native {
            return Some((self.native_scenery.as_ref()?.clone(), rect(4, 4, 316, 196)));
        }
        None
    }

    pub(crate) fn sides(&mut self) -> Option<(&[Arc<Picture>; 2], f32)> {
        let pair = self.pair.as_ref()?;
        let mask = self.masked.get_or_insert_with(|| {
            let pictures = &pair.sides[usize::from(self.settings.set == "classic")];
            std::array::from_fn(|side| Arc::new(mask_player_poster(&pictures[side], side == 1, self.settings.edge)))
        });
        let alpha = self.started.map_or(1.0, |start| {
            if self.settings.fade_ms == 0 {
                return 1.0;
            }
            let t = (start.elapsed().as_secs_f32() * 1000.0 / self.settings.fade_ms as f32).min(1.0);
            t * t * (3.0 - 2.0 * t)
        });
        Some((mask, alpha))
    }

    fn set_edge(&mut self, x: i64) {
        let x = x.clamp(SLIDER.min.x, SLIDER.max.x - 1) - SLIDER.min.x;
        let range = SLIDER.max.x - SLIDER.min.x - 1;
        let edge = (96 * x + range / 2) / range;
        if self.settings.edge != edge {
            self.settings.edge = edge;
            self.masked = None;
        }
    }

    fn close(&mut self) {
        if self.slider.take().is_some() {
            self.save();
        }
        self.open = false;
    }

    pub(crate) fn toggle_menu(&mut self) {
        if self.open {
            self.close();
        } else {
            self.open = true;
        }
    }

    pub(crate) fn menu_open(&self) -> bool {
        self.open
    }

    pub(crate) fn menu_rect(&self) -> Rectangle {
        if self.open { MENU } else { Rectangle::default() }
    }

    /// 返回此幀是否由選單消費。上方其餘工具列按鈕繼續由工具列處理。
    pub(crate) fn update_menu(&mut self, input: &TickInput, focused: bool, allowed: bool) -> bool {
        if !self.open {
            return false;
        }
        if !allowed || input.is_just_pressed(Key::Escape) {
            self.close();
            return input.is_just_pressed(Key::Escape);
        }
        if let Some(slider) = self.slider {
            let x = match slider {
                SliderInput::Mouse if focused && input.mouse_left_held => Some(input.cursor.0),
                SliderInput::Mouse => None,
                SliderInput::Touch(id) => input.touches.iter().find(|&&touch| touch.0 == id).map(|touch| touch.1),
            };
            if let Some(x) = x {
                self.set_edge(x);
            } else {
                self.slider = None;
                self.save();
            }
            return true;
        }
        let point = if focused && input.mouse_left_just_pressed {
            Some((input.cursor, SliderInput::Mouse))
        } else {
            input.just_pressed_touches.iter().find_map(|&(x, y)| {
                input.touches.iter().find(|&&(_, tx, ty)| (x, y) == (tx, ty)).map(|t| ((x, y), SliderInput::Touch(t.0)))
            })
        };
        let Some((point, slider)) = point else { return false };
        if (0..PLAYER_TOOLBAR_HEIGHT).contains(&point.1) {
            return false;
        }
        if !contains(MENU, point) || contains(CLOSE, point) {
            self.close();
        } else if let Some(index) = choices::<3>(49, 19, 84).iter().position(|&r| contains(r, point)) {
            self.apply_mode(MODES[index], true);
        } else if contains(SLIDER, point) {
            self.set_edge(point.0);
            self.slider = Some(slider);
        } else if let Some(index) = choices::<4>(109, 20, 63).iter().position(|&r| contains(r, point)) {
            self.settings.fade_ms = DURATIONS[index];
            self.started = Some(Instant::now());
            self.save();
        } else if contains(REPLAY, point) {
            self.started = Some(Instant::now());
        }
        true
    }

    pub(crate) fn render_key(&self) -> String {
        if self.open {
            format!("{} {} {} {}", self.settings.set, self.settings.edge, self.settings.fade_ms, self.failed)
        } else {
            String::new()
        }
    }

    pub(crate) fn draw_menu(&self, surface: &mut Picture, font: &mut Font) {
        let fill = |surface: &mut Picture, r: Rectangle, color| {
            surface.sub_image_mut(rect(r.min.x * 3, r.min.y * 3, r.max.x * 3, r.max.y * 3)).fill(color);
        };
        let ink = ColorRgba { r: 245, g: 245, b: 245, a: 255 };
        let button = |surface: &mut Picture, font: &mut Font, r: Rectangle, label: &str, selected| {
            let color = if selected {
                ColorRgba { r: 57, g: 77, b: 108, a: 255 }
            } else {
                ColorRgba { r: 31, g: 45, b: 66, a: 255 }
            };
            fill(surface, r, color);
            let x = ((r.min.x + r.max.x) as f64 * 3.0 - font.width(label, 1.4)) / 2.0;
            font.draw(surface, label, x, (r.min.y * 3 + 9) as f64, 1.4, ink);
        };
        fill(surface, MENU, ColorRgba { r: 88, g: 104, b: 126, a: 255 });
        fill(surface, rect(363, 27, 633, 156), ColorRgba { r: 17, g: 24, b: 37, a: 255 });
        let title = if self.failed { "海報素材不可用" } else { "海報" };
        button(surface, font, rect(374, 28, 622, 46), title, false);
        for (index, r) in choices::<3>(49, 19, 84).into_iter().enumerate() {
            button(surface, font, r, SETS[index], self.settings.set == MODES[index]);
        }
        let edge = self.settings.edge;
        button(surface, font, rect(374, 69, 622, 84), &format!("內緣漸層 {edge}"), false);
        fill(surface, rect(380, 93, 620, 94), ColorRgba { r: 110, g: 124, b: 142, a: 255 });
        let x = 380 + edge * 239 / 96;
        fill(surface, rect(x - 1, 87, x + 2, 102), ColorRgba { r: 220, g: 185, b: 112, a: 255 });
        for (index, r) in choices::<4>(109, 20, 63).into_iter().enumerate() {
            let duration = DURATIONS[index];
            button(surface, font, r, &format!("{:.1}s", duration as f64 / 1000.0), self.settings.fade_ms == duration);
        }
        button(surface, font, REPLAY, "重播淡入", false);
        button(surface, font, CLOSE, "收合", false);
    }
}

impl Game {
    /// Go：`player_poster_controls.go:23`，`func (g *Game) playerPosterControlsAllowed() bool`
    pub(crate) fn player_poster_controls_allowed(&self) -> bool {
        !self.startup_blocked
            && self.load_err.is_empty()
            && self.native_defeat.is_none()
            && self.native_ending.is_none()
            && !self.help_visible
            && !self.transient_ui
            && !native_modifier_held(&self.tick_input)
            && self.native_system_end_turn_ui.as_ref().is_none_or(|ui| ui.reward_message.is_none())
    }
}
