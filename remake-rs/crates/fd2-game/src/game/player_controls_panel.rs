//! Go 來源：`remake/cmd/fd2/player_controls_panel.go`（678 行，Go v.1.1.15 `7c710b34`）。R5 階段 4（r5i）移植。
//!
//! F12「手把與自訂按鍵」設定面板（版面依 mm3 使用者核可的分組表格）。
//!
//! 開啟：F12（邏輯鍵，可改綁）、手把 Start（預設綁 F12）；工具列「按鍵」在 R6（使用者定案工具列延到 R6）。開著時遊戲
//! Update 暫停、遊戲時鐘凍結，與能力設定相同。面板自己的操作讀實體鍵與手把固定按鈕，不經玩家綁定，改壞綁定也不會被鎖
//! 在面板外：
//!
//! ```text
//! 鍵盤：↑↓ 選列、PageUp／PageDown、Enter 指定、Delete 清除此列、R 恢復此列預設、
//!       Esc 取消或返回、F12 關閉、←→ 調整防飄移（選到滑桿列時）。
//! 手把：十字鍵或左搖桿選列、A 指定、B 取消或返回、X 清除、Y 預設、Start 關閉。
//! 滑鼠：點選列（再點一次＝指定）、滾輪捲動、拖曳滑桿、右鍵＝返回、「全部恢復預設」按鈕。
//! ```
//!
//! 指定時再按要用的鍵盤鍵或手把按鈕；已被別列使用就先提示，確認後才取代，別列改成未設定。每次變更立即寫
//! `controls-settings.json`。滑鼠游標速度不提供：FD2 沒有遊戲內的模擬游標，滑鼠直接用系統指標，指向格子與選單。
//!
//! # 繪製
//!
//! Go 在 `DrawFinalScreen` 把面板畫在 1920×1200 的疊層（`controlsSurface`），再以線性濾鏡縮 1/3 疊到內容區。Rust 的
//! [`Game::draw_controls_panel_surface`] 在 CPU 畫同一張 1920×1200 疊層（填色是整數像素的不透明矩形，文字照 Ebiten
//! `text.Draw`），縮放與疊合由外殼在 GPU 做（`fd2-app/src/video.rs`，KNOWN_DIFFERENCES `controls-panel-final-filter`）。

use super::*;
use crate::go_ebiten_keys::{self as ek, EbitenKey};

/// Go：`player_controls_panel.go:31`，`type controlsConflict struct`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ControlsConflict {
    pub(crate) action: usize,
    pub(crate) other: usize,
    pub(crate) reset: bool,
    pub(crate) gamepad: bool,
    pub(crate) key: EbitenKey,
    pub(crate) button: ControlPadButton,
}

/// Go：`player_controls_panel.go:38`，`type controlsPanel struct`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ControlsPanel {
    pub(crate) open: bool,
    /// 0..len(CONTROL_ACTIONS)-1 是動作；len(CONTROL_ACTIONS) 是防飄移滑桿
    pub(crate) selected: usize,
    /// 第一條可見的行（行含分組標題）
    pub(crate) scroll: i64,
    pub(crate) capture: bool,
    pub(crate) conflict: Option<ControlsConflict>,
    pub(crate) message: String,
    pub(crate) dragging: bool,
    /// 面板內的搖桿連發計數（上、下、左、右）
    pub(crate) stick_held: [i64; 4],
    pub(crate) text: Option<ControlsLocale>,
    pub(crate) text_locale: String,
}

/// 一次 Update 收到的面板操作；實體鍵盤、滑鼠與手把都先換成這個，測試直接餵這個結構。
///
/// Go：`player_controls_panel.go:53`，`type controlsPanelInput struct`
#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub(crate) struct ControlsPanelInput {
    pub(crate) up: bool,
    pub(crate) down: bool,
    pub(crate) left: bool,
    pub(crate) right: bool,
    pub(crate) page_up: bool,
    pub(crate) page_down: bool,
    pub(crate) confirm: bool,
    pub(crate) back: bool,
    pub(crate) clear: bool,
    pub(crate) reset: bool,
    pub(crate) close: bool,
    pub(crate) capture_key: EbitenKey,
    pub(crate) capture_button: ControlPadButton,
    pub(crate) capture_cancel: bool,
    pub(crate) click: bool,
    pub(crate) drag: bool,
    pub(crate) point: (i64, i64),
    pub(crate) wheel: i64,
}

/// Go：`player_controls_panel.go:64`，`func noControlsPanelInput() controlsPanelInput`
pub(crate) fn no_controls_panel_input() -> ControlsPanelInput {
    ControlsPanelInput {
        up: false,
        down: false,
        left: false,
        right: false,
        page_up: false,
        page_down: false,
        confirm: false,
        back: false,
        clear: false,
        reset: false,
        close: false,
        capture_key: KEY_UNBOUND,
        capture_button: PAD_NONE,
        capture_cancel: false,
        click: false,
        drag: false,
        point: (0, 0),
        wheel: 0,
    }
}

// 面板幾何（640×400 畫布座標）。Go：`player_controls_panel.go:69`..`82`
pub(crate) const CONTROLS_PANEL_RECT: Rectangle = r(16, 10, 624, 390);
pub(crate) const CONTROLS_PAD_CHIP: Rectangle = r(436, 20, 610, 44);
pub(crate) const CONTROLS_HEADER_BAND: Rectangle = r(24, 62, 616, 82);
pub(crate) const CONTROLS_LIST_RECT: Rectangle = r(24, 82, 616, 282);
pub(crate) const CONTROLS_SLIDER_ROW: Rectangle = r(24, 286, 616, 312);
pub(crate) const CONTROLS_SLIDER_TRACK: Rectangle = r(236, 296, 456, 302);
pub(crate) const CONTROLS_RESET_ALL_RECT: Rectangle = r(478, 338, 610, 364);
pub(crate) const CONTROLS_COLUMN_KEYS: i64 = 296;
pub(crate) const CONTROLS_COLUMN_PAD: i64 = 412;
pub(crate) const CONTROLS_LINE_HEIGHT: i64 = 20;
pub(crate) const CONTROLS_VISIBLE_LINES: i64 = 10;
pub(crate) const CONTROLS_SLIDER_SELECTS: usize = CONTROL_ACTIONS.len();

const fn r(x0: i64, y0: i64, x1: i64, y1: i64) -> Rectangle {
    Rectangle { min: Point { x: x0, y: y0 }, max: Point { x: x1, y: y1 } }
}

fn point_in(p: (i64, i64), rr: Rectangle) -> bool {
    rr.min.x <= p.0 && p.0 < rr.max.x && rr.min.y <= p.1 && p.1 < rr.max.y
}

/// 清單的一行：分組標題（`action = -1`）或動作列。
///
/// Go：`player_controls_panel.go:85`，`type controlsLine struct`
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct ControlsLine {
    pub(crate) group: ControlActionGroup,
    pub(crate) action: i64,
}

/// Go：`player_controls_panel.go:90`，`func controlsLines() []controlsLine`
pub(crate) fn controls_lines() -> Vec<ControlsLine> {
    let mut lines = Vec::new();
    for group in 0..CONTROL_GROUP_COUNT {
        lines.push(ControlsLine { group, action: -1 });
        for (i, action) in CONTROL_ACTIONS.iter().enumerate() {
            if action.group == group {
                lines.push(ControlsLine { group, action: i as i64 });
            }
        }
    }
    lines
}

/// Go：`player_controls_panel.go:103`，`func controlsLineOf(action int) int`
pub(crate) fn controls_line_of(action: usize) -> i64 {
    controls_lines().iter().position(|line| line.action == action as i64).map_or(0, |i| i as i64)
}

fn txt(c: &ControlsPanel, key: &str) -> String {
    ControlsLocale::text(c.text.as_ref(), key, &[])
}

impl Game {
    /// Go：`player_controls_panel.go:112`，`func (g *Game) openControlsPanel()`
    pub(crate) fn open_controls_panel(&mut self) {
        if self.player_input.is_none() {
            let (b, status) = load_controls_settings();
            self.player_input = Some(PlayerInput::new(b, status));
        }
        let text = match load_controls_locale(&self.locale_id) {
            Ok(text) => Some(text),
            Err(err) => {
                self.load_err = format!("controls locale: {}", err.message());
                None
            }
        };
        self.controls = ControlsPanel { open: true, text, text_locale: self.locale_id.clone(), ..Default::default() };
        let key = match self.player_input.as_ref().unwrap().status {
            ControlsLoadStatus::Corrupt => "controls.msg.corrupt",
            ControlsLoadStatus::Partial => "controls.msg.partial",
            _ => "controls.msg.ready",
        };
        self.controls.message = txt(&self.controls, key);
        // Go 另呼叫 `recordInputAuditState`（除錯觀測器，沒有移植）。
    }

    /// Go：`player_controls_panel.go:133`，`func (g *Game) closeControlsPanel()`
    pub(crate) fn close_controls_panel(&mut self) {
        self.controls = ControlsPanel::default();
    }
}

impl ControlsPanel {
    /// Go：`player_controls_panel.go:137`，`func (c *controlsPanel) actionName(action int) string`
    pub(crate) fn action_name(&self, action: usize) -> String {
        txt(self, &format!("controls.action.{}", CONTROL_ACTIONS[action].id))
    }

    /// Go：`player_controls_panel.go:141`，`func (c *controlsPanel) ensureVisible()`
    pub(crate) fn ensure_visible(&mut self) {
        if self.selected >= CONTROLS_SLIDER_SELECTS {
            return;
        }
        let mut line = controls_line_of(self.selected);
        if line > 0 && controls_lines()[(line - 1) as usize].action < 0 {
            line -= 1; // 分組第一列連同標題一起露出
        }
        if line < self.scroll {
            self.scroll = line;
        }
        let last = controls_line_of(self.selected);
        if last >= self.scroll + CONTROLS_VISIBLE_LINES {
            self.scroll = last - CONTROLS_VISIBLE_LINES + 1;
        }
        self.clamp_scroll();
    }

    /// Go：`player_controls_panel.go:159`，`func (c *controlsPanel) clampScroll()`
    pub(crate) fn clamp_scroll(&mut self) {
        let max_scroll = controls_lines().len() as i64 - CONTROLS_VISIBLE_LINES;
        if self.scroll > max_scroll {
            self.scroll = max_scroll;
        }
        if self.scroll < 0 {
            self.scroll = 0;
        }
    }

    /// Go：`player_controls_panel.go:390`，`func (c *controlsPanel) padLabel(button controlPadButton) string`
    pub(crate) fn pad_label(&self, button: ControlPadButton) -> String {
        match button {
            PAD_NONE => String::new(),
            PAD_UP => txt(self, "controls.dpad") + "↑",
            PAD_DOWN => txt(self, "controls.dpad") + "↓",
            PAD_LEFT => txt(self, "controls.dpad") + "←",
            PAD_RIGHT => txt(self, "controls.dpad") + "→",
            _ => PAD_FACE_LABELS[button as usize].to_string(),
        }
    }
}

impl Game {
    fn panel_input(&self) -> &PlayerInput {
        self.player_input.as_ref().expect("面板開著時 g.input 一定有值（openControlsPanel 建立）")
    }

    /// 套用並立即保存；保存失敗時綁定仍生效，訊息告訴玩家。
    ///
    /// Go：`player_controls_panel.go:170`，`func (g *Game) commitControls(b controlBindings, messageKey string)`
    pub(crate) fn commit_controls(&mut self, b: ControlBindings, message_key: &str) {
        let input = self.player_input.as_mut().expect("面板開著時 g.input 一定有值");
        input.set_bindings(b);
        input.status = ControlsLoadStatus::Ok;
        if let Err(err) = save_controls_settings(&input.bindings) {
            self.controls.message = ControlsLocale::text(
                self.controls.text.as_ref(),
                "controls.msg.save_failed",
                &[ControlsArg::S(err.to_string())],
            );
            return;
        }
        self.controls.message = txt(&self.controls, message_key);
    }

    /// Go：`player_controls_panel.go:181`，`func (g *Game) stepControlsPanel(in controlsPanelInput)`
    pub(crate) fn step_controls_panel(&mut self, input: ControlsPanelInput) {
        if !self.controls.open || self.player_input.is_none() {
            return;
        }
        if input.close {
            self.close_controls_panel();
            return;
        }
        if self.controls.conflict.is_some() {
            if input.back {
                self.controls.conflict = None;
                self.controls.message = txt(&self.controls, "controls.msg.ready");
            } else if input.confirm || input.click {
                self.apply_controls_conflict();
            }
            return;
        }
        if self.controls.capture {
            if input.capture_cancel || input.back || input.click {
                self.controls.capture = false;
                self.controls.message = txt(&self.controls, "controls.msg.ready");
            } else if input.capture_key != KEY_UNBOUND {
                self.capture_controls_key(input.capture_key);
            } else if input.capture_button != PAD_NONE {
                self.capture_controls_button(input.capture_button);
            }
            return;
        }
        if input.back {
            self.close_controls_panel();
            return;
        }
        let last = CONTROLS_SLIDER_SELECTS;
        let c = &mut self.controls;
        if input.up && c.selected > 0 {
            c.selected -= 1;
            c.ensure_visible();
        } else if input.down && c.selected < last {
            c.selected += 1;
            c.ensure_visible();
        } else if input.page_up {
            c.selected = (c.selected as i64 - CONTROLS_VISIBLE_LINES + 1).max(0) as usize;
            c.ensure_visible();
        } else if input.page_down {
            c.selected = (c.selected + CONTROLS_VISIBLE_LINES as usize - 1).min(last);
            c.ensure_visible();
        }
        if input.wheel != 0 {
            c.scroll -= input.wheel;
            c.clamp_scroll();
        }
        if c.selected == last && (input.left || input.right) {
            let mut b = self.panel_input().bindings.clone();
            if input.left {
                b.deadzone = MIN_STICK_DEADZONE.max(b.deadzone - STICK_DEADZONE_STEP);
            } else {
                b.deadzone = MAX_STICK_DEADZONE.min(b.deadzone + STICK_DEADZONE_STEP);
            }
            self.commit_controls(b, "controls.msg.assigned");
        }
        if input.click || input.drag {
            self.controls_panel_point(input.point, input.click);
            return;
        }
        let selected = self.controls.selected;
        if selected >= last {
            return;
        }
        if input.confirm {
            self.controls.capture = true;
            self.controls.message = txt(&self.controls, "controls.msg.capture");
        } else if input.clear {
            let mut b = self.panel_input().bindings.clone();
            b.clear_action(selected);
            self.commit_controls(b, "controls.msg.cleared");
        } else if input.reset {
            let other = self.panel_input().bindings.reset_conflict(selected);
            if other >= 0 {
                // resetConflict 先看鍵盤再看手把；提示顯示被拿走的那一個。
                let mut label = self.controls.pad_label(CONTROL_ACTIONS[selected].default_pad);
                let key = CONTROL_ACTIONS[selected].logical;
                if self.panel_input().bindings.keyboard_owner(key) == other {
                    label = control_key_label(key);
                }
                let other = other as usize;
                self.controls.conflict =
                    Some(ControlsConflict { action: selected, other, reset: true, ..Default::default() });
                let name = self.controls.action_name(other);
                self.controls.message = ControlsLocale::text(
                    self.controls.text.as_ref(),
                    "controls.msg.conflict",
                    &[ControlsArg::S(label), ControlsArg::S(name)],
                );
                return;
            }
            let mut b = self.panel_input().bindings.clone();
            b.reset_action(selected);
            self.commit_controls(b, "controls.msg.reset_row");
        }
    }

    /// Go：`player_controls_panel.go:276`，`func (g *Game) controlsPanelPoint(point image.Point, click bool)`
    pub(crate) fn controls_panel_point(&mut self, point: (i64, i64), click: bool) {
        if click {
            self.controls.dragging = false; // 新的一次按下重新判斷位置
        }
        if self.controls.dragging || click && point_in(point, CONTROLS_SLIDER_ROW) {
            (self.controls.selected, self.controls.dragging) = (CONTROLS_SLIDER_SELECTS, true);
            let span = CONTROLS_SLIDER_TRACK.max.x - CONTROLS_SLIDER_TRACK.min.x;
            let ratio = (point.0 - CONTROLS_SLIDER_TRACK.min.x) as f64 / span as f64;
            // Go `int(...)`：向零截斷。
            let mut value =
                MIN_STICK_DEADZONE + (ratio * (MAX_STICK_DEADZONE - MIN_STICK_DEADZONE) as f64 + 0.5) as i64;
            value = MIN_STICK_DEADZONE.max(MAX_STICK_DEADZONE.min(value));
            value -= value % STICK_DEADZONE_STEP;
            if value != self.panel_input().bindings.deadzone {
                let mut b = self.panel_input().bindings.clone();
                b.deadzone = value;
                self.commit_controls(b, "controls.msg.assigned");
            }
            return;
        }
        if !click {
            return;
        }
        if point_in(point, CONTROLS_RESET_ALL_RECT) {
            self.commit_controls(default_control_bindings(), "controls.msg.reset_all");
            return;
        }
        if !point_in(point, CONTROLS_LIST_RECT) {
            return;
        }
        // Go 的整數除法向零截斷；point 在清單內，被除數不小於 0。
        let line = self.controls.scroll + (point.1 - CONTROLS_LIST_RECT.min.y) / CONTROLS_LINE_HEIGHT;
        let lines = controls_lines();
        if line < 0 || line >= lines.len() as i64 || lines[line as usize].action < 0 {
            return;
        }
        let action = lines[line as usize].action as usize;
        if action == self.controls.selected {
            self.controls.capture = true;
            self.controls.message = txt(&self.controls, "controls.msg.capture");
            return;
        }
        self.controls.selected = action;
    }

    /// Go：`player_controls_panel.go:318`，`func (g *Game) captureControlsKey(k ebiten.Key)`
    pub(crate) fn capture_controls_key(&mut self, k: EbitenKey) {
        if !bindable_key(k) {
            self.controls.message = txt(&self.controls, "controls.msg.reserved");
            return;
        }
        self.controls.capture = false;
        let selected = self.controls.selected;
        let owner = self.panel_input().bindings.keyboard_owner(k);
        if owner >= 0 && owner as usize != selected {
            let other = owner as usize;
            self.controls.conflict = Some(ControlsConflict { action: selected, other, key: k, ..Default::default() });
            let name = self.controls.action_name(other);
            self.controls.message = ControlsLocale::text(
                self.controls.text.as_ref(),
                "controls.msg.conflict",
                &[ControlsArg::S(control_key_label(k)), ControlsArg::S(name)],
            );
            return;
        }
        let mut b = self.panel_input().bindings.clone();
        b.set_keyboard(selected, k);
        self.commit_controls(b, "controls.msg.assigned");
    }

    /// Go：`player_controls_panel.go:335`，`func (g *Game) captureControlsButton(button controlPadButton)`
    pub(crate) fn capture_controls_button(&mut self, button: ControlPadButton) {
        self.controls.capture = false;
        let selected = self.controls.selected;
        let owner = self.panel_input().bindings.gamepad_owner(button);
        if owner >= 0 && owner as usize != selected {
            let other = owner as usize;
            self.controls.conflict =
                Some(ControlsConflict { action: selected, other, gamepad: true, button, ..Default::default() });
            let label = self.controls.pad_label(button);
            let name = self.controls.action_name(other);
            self.controls.message = ControlsLocale::text(
                self.controls.text.as_ref(),
                "controls.msg.conflict",
                &[ControlsArg::S(label), ControlsArg::S(name)],
            );
            return;
        }
        let mut b = self.panel_input().bindings.clone();
        b.set_gamepad(selected, button);
        self.commit_controls(b, "controls.msg.assigned");
    }

    /// Go：`player_controls_panel.go:348`，`func (g *Game) applyControlsConflict()`
    pub(crate) fn apply_controls_conflict(&mut self) {
        let Some(conflict) = self.controls.conflict.take() else {
            return;
        };
        let mut b = self.panel_input().bindings.clone();
        if conflict.reset {
            b.reset_action(conflict.action);
            self.commit_controls(b, "controls.msg.reset_row");
        } else if conflict.gamepad {
            b.set_gamepad(conflict.action, conflict.button);
            self.commit_controls(b, "controls.msg.assigned");
        } else {
            b.set_keyboard(conflict.action, conflict.key);
            self.commit_controls(b, "controls.msg.assigned");
        }
    }
}

/// 鍵盤格顯示的鍵名。
///
/// Go：`player_controls_panel.go:367`，`func controlKeyLabel(k ebiten.Key) string`
pub(crate) fn control_key_label(k: EbitenKey) -> String {
    match k {
        KEY_UNBOUND => String::new(),
        ek::KEY_ARROW_UP => String::from("↑"),
        ek::KEY_ARROW_DOWN => String::from("↓"),
        ek::KEY_ARROW_LEFT => String::from("←"),
        ek::KEY_ARROW_RIGHT => String::from("→"),
        ek::KEY_ESCAPE => String::from("Esc"),
        _ => {
            let name = ek::key_string(k);
            name.strip_prefix("Digit").unwrap_or(name).to_string()
        }
    }
}

/// Go：`player_controls_panel.go:385`，`var padFaceLabels`（十字鍵四格由 `padLabel` 另組）。
pub(crate) const PAD_FACE_LABELS: [&str; PAD_BUTTON_COUNT] =
    ["A", "B", "X", "Y", "LB", "RB", "LT", "RT", "Back", "Start", "L3", "R3", "", "", "", ""];

// ---- 實體輸入 → controlsPanelInput ----

/// Go：`player_controls_panel.go:408`，`func repeatTick(d int) bool`
pub(crate) fn repeat_tick(d: i64) -> bool {
    d == 1 || d >= 12 && (d - 12) % 5 == 0
}

impl Game {
    /// Go：`player_controls_panel.go:412`，`func (p *modernHDPresentation) readControlsPanelInput() controlsPanelInput`。
    /// Go 直接查 `inpututil` 與 Ebiten 的手把；Rust 讀同一次 `Update` 的實體 [`TickInput`] 與 [`DeviceTick`]。游標扣掉
    /// 海報與工具列位移（R5 沒有兩者，位移是 0）。
    pub(crate) fn read_controls_panel_input(&mut self, physical: &TickInput, dev: &DeviceTick) -> ControlsPanelInput {
        let mut input = no_controls_panel_input();
        let raw = |k: EbitenKey| physical_key_duration(physical, k);
        let just = |k: EbitenKey| raw(k) == 1;
        (input.up, input.down) = (repeat_tick(raw(ek::KEY_ARROW_UP)), repeat_tick(raw(ek::KEY_ARROW_DOWN)));
        (input.left, input.right) = (repeat_tick(raw(ek::KEY_ARROW_LEFT)), repeat_tick(raw(ek::KEY_ARROW_RIGHT)));
        (input.page_up, input.page_down) = (just(ek::KEY_PAGE_UP), just(ek::KEY_PAGE_DOWN));
        input.confirm = just(ek::KEY_ENTER) || just(ek::KEY_NUMPAD_ENTER);
        input.back = just(ek::KEY_ESCAPE) || dev.right_just;
        input.clear = just(ek::KEY_DELETE);
        input.reset = just(ek::KEY_R);
        input.close = just(ek::KEY_F12);
        let capture = self.controls.capture;
        if capture {
            if let Some(&k) = physical_just_pressed_keys(physical).first() {
                if k == ek::KEY_ESCAPE {
                    input.capture_cancel = true;
                } else {
                    input.capture_key = k;
                }
            }
            input.close = false;
        }
        let mut stick = [false; 4];
        let deadzone = self.player_input.as_ref().map_or(DEFAULT_STICK_DEADZONE, |i| i.bindings.deadzone);
        for (id, durations) in &dev.pad_durations {
            let Some(pad) = dev.snapshot.pads.iter().find(|p| p.id == *id) else {
                continue;
            };
            let just = |b: ControlPadButton| durations[b as usize] == 1;
            let hold = |b: ControlPadButton| durations[b as usize];
            if capture {
                for button in 0..PAD_BUTTON_COUNT as ControlPadButton {
                    if just(button) && input.capture_button == PAD_NONE {
                        if button == PAD_B {
                            input.capture_cancel = true;
                        } else {
                            input.capture_button = button;
                        }
                    }
                }
                continue;
            }
            input.up = input.up || repeat_tick(hold(PAD_UP));
            input.down = input.down || repeat_tick(hold(PAD_DOWN));
            input.left = input.left || repeat_tick(hold(PAD_LEFT));
            input.right = input.right || repeat_tick(hold(PAD_RIGHT));
            input.confirm = input.confirm || just(PAD_A);
            input.back = input.back || just(PAD_B);
            input.clear = input.clear || just(PAD_X);
            input.reset = input.reset || just(PAD_Y);
            input.close = input.close || just(PAD_START);
            match stick_direction(pad.stick_x, pad.stick_y, deadzone) {
                ek::KEY_ARROW_UP => stick[0] = true,
                ek::KEY_ARROW_DOWN => stick[1] = true,
                ek::KEY_ARROW_LEFT => stick[2] = true,
                ek::KEY_ARROW_RIGHT => stick[3] = true,
                _ => {}
            }
        }
        let c = &mut self.controls;
        for (i, &held) in stick.iter().enumerate() {
            if held {
                c.stick_held[i] += 1;
            } else {
                c.stick_held[i] = 0;
            }
        }
        input.up = input.up || repeat_tick(c.stick_held[0]);
        input.down = input.down || repeat_tick(c.stick_held[1]);
        input.left = input.left || repeat_tick(c.stick_held[2]);
        input.right = input.right || repeat_tick(c.stick_held[3]);
        input.point = (physical.cursor.0 - self.player_viewport_left, physical.cursor.1 - self.player_viewport_top);
        input.click = physical.mouse_left_just_pressed;
        if !physical.mouse_left_held {
            c.dragging = false;
        }
        input.drag = c.dragging && !input.click;
        let wheel = dev.snapshot.wheel_y;
        if wheel >= 1.0 {
            input.wheel = 1;
        } else if wheel <= -1.0 {
            input.wheel = -1;
        }
        input
    }

    /// 面板開著就吃掉這一次 Update。
    ///
    /// Go：`player_controls_panel.go:510`，`func (p *modernHDPresentation) updateControlsPanel() bool`
    pub(crate) fn update_controls_panel(&mut self, now: GameInstant, physical: &TickInput, dev: &DeviceTick) -> bool {
        if !self.controls.open {
            return false;
        }
        self.player_clock.wall = now; // 與能力設定相同：面板開著時遊戲時鐘不走
        let input = self.read_controls_panel_input(physical, dev);
        self.step_controls_panel(input);
        true
    }
}

// ---- 繪製（1920×1200 疊層，同能力設定的三倍畫布）----

// Go：`player_controls_panel.go:523`..`534`
pub(crate) const CONTROLS_COLOR_PANEL: ColorRgba = ColorRgba { r: 24, g: 30, b: 43, a: 255 };
pub(crate) const CONTROLS_COLOR_BORDER: ColorRgba = ColorRgba { r: 70, g: 82, b: 104, a: 255 };
pub(crate) const CONTROLS_COLOR_BAND: ColorRgba = ColorRgba { r: 36, g: 43, b: 58, a: 255 };
pub(crate) const CONTROLS_COLOR_GROUP: ColorRgba = ColorRgba { r: 30, g: 36, b: 50, a: 255 };
pub(crate) const CONTROLS_COLOR_SELECTED: ColorRgba = ColorRgba { r: 60, g: 53, b: 36, a: 255 };
pub(crate) const CONTROLS_COLOR_GOLD: ColorRgba = ColorRgba { r: 226, g: 194, b: 122, a: 255 };
pub(crate) const CONTROLS_COLOR_TEXT: ColorRgba = ColorRgba { r: 232, g: 235, b: 241, a: 255 };
pub(crate) const CONTROLS_COLOR_MUTED: ColorRgba = ColorRgba { r: 158, g: 168, b: 186, a: 255 };
pub(crate) const CONTROLS_COLOR_CHIP: ColorRgba = ColorRgba { r: 30, g: 40, b: 56, a: 255 };
pub(crate) const CONTROLS_COLOR_CHIP_LINE: ColorRgba = ColorRgba { r: 112, g: 128, b: 154, a: 255 };

/// 疊層大小（Go `ebiten.NewImage(1920, 1200)`）。
pub const CONTROLS_SURFACE_W: i64 = 1920;
pub const CONTROLS_SURFACE_H: i64 = 1200;

/// 面板的一次 `text.Draw`（局部黃金比對鍵）：文字、疊層座標的原點（Go `text.Draw` 的 `x, y`，基線）、字級（Go
/// `faceFor(size)`）、顏色與 `text.BoundString(face, caption)`（相對原點）。
#[derive(Debug, Clone, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub(crate) struct ControlsTextDraw {
    pub(crate) text: String,
    pub(crate) x: i64,
    pub(crate) y: i64,
    pub(crate) size: i64,
    pub(crate) color: [u8; 4],
    pub(crate) bound: [i64; 4],
}

/// 畫面板用的工具：Go `drawControlsPanel` 裡的 `rect`、`frame`、`label`、`measure`、`chip` 五個閉包。
struct PanelPainter<'a> {
    s: &'a mut Picture,
    f: Option<Font>,
    record: Option<&'a mut Vec<ControlsTextDraw>>,
    draw_text: bool,
}

impl PanelPainter<'_> {
    /// `vector.DrawFilledRect(s, x, y, w, h, fill, false)`：整數座標、不透明色，等於覆蓋像素中心落在矩形內的像素。
    fn fill_px(&mut self, x: i64, y: i64, w: i64, h: i64, fill: ColorRgba) {
        debug_assert_eq!(fill.a, 255, "面板只用不透明色，source-over 等於覆蓋");
        if w <= 0 || h <= 0 {
            return;
        }
        self.s.sub_image_mut(rect(x, y, x + w, y + h)).fill(fill);
    }

    fn rect(&mut self, rr: Rectangle, fill: ColorRgba) {
        let (dx, dy) = (rr.max.x - rr.min.x, rr.max.y - rr.min.y);
        self.fill_px(rr.min.x * 3, rr.min.y * 3, dx * 3, dy * 3, fill);
    }

    fn frame(&mut self, rr: Rectangle, fill: ColorRgba, line: ColorRgba) {
        self.rect(rr, line);
        let (dx, dy) = (rr.max.x - rr.min.x, rr.max.y - rr.min.y);
        self.fill_px(rr.min.x * 3 + 2, rr.min.y * 3 + 2, dx * 3 - 4, dy * 3 - 4, fill);
    }

    fn face(&mut self, size: i64) -> Option<(Arc<FontFace>, Arc<dyn TextMeasure>)> {
        let f = self.f.as_mut()?;
        let (face, _) = f.face_for(size);
        let face = face?;
        let measure = f.measure.0.clone().expect("face_for 成功時已注入 TextMeasure");
        Some((face, measure))
    }

    fn label(&mut self, caption: &str, x: i64, baseline: i64, size: i64, ink: ColorRgba) -> i64 {
        if self.f.is_none() || caption.is_empty() {
            return 0;
        }
        let Some((face, measure)) = self.face(size) else {
            return 0;
        };
        let bound = measure.bound_string(&face, caption);
        if let Some(record) = self.record.as_deref_mut() {
            record.push(ControlsTextDraw {
                text: caption.to_string(),
                x: x * 3,
                y: baseline * 3,
                size,
                color: [ink.r, ink.g, ink.b, ink.a],
                bound: [bound.min.x, bound.min.y, bound.max.x, bound.max.y],
            });
        }
        if self.draw_text {
            // Ebiten `text.Draw(dst, text, face, x, y, clr)`：`GeoM.Translate(x, y)`、`ColorScale.ScaleWithColor(clr)`。
            let mut op = DrawImageOptions::default();
            op.geo_m.translate((x * 3) as f64, (baseline * 3) as f64);
            op.color_scale.scale_with_color(ink);
            measure.draw_text(&face, self.s, caption, &op);
        }
        rect_dx(&bound) / 3
    }

    fn measure(&mut self, caption: &str, size: i64) -> i64 {
        if self.f.is_none() || caption.is_empty() {
            return 0;
        }
        let Some((face, measure)) = self.face(size) else {
            return 0;
        };
        (rect_dx(&measure.bound_string(&face, caption)) + 2) / 3
    }

    fn chip(&mut self, caption: &str, x: i64, top: i64, ink: ColorRgba) -> i64 {
        let w = self.measure(caption, 33) + 12;
        self.frame(rect(x, top, x + w, top + 16), CONTROLS_COLOR_CHIP, CONTROLS_COLOR_CHIP_LINE);
        self.label(caption, x + 6, top + 12, 33, ink);
        w
    }
}

impl Game {
    /// Go：`player_controls_panel.go:536`，`func (p *modernHDPresentation) drawControlsPanel(screen ebiten.FinalScreen, geometry ebiten.GeoM)`
    /// 的疊層部分：在 1920×1200 的 `s` 上畫面板（Go 的 `p.controlsSurface`）。最後一步「以線性濾鏡縮 1/3 疊到內容區」在
    /// 外殼。`record` 收集每次 `text.Draw` 的參數；`draw_text` 為 false 時只畫矩形（局部黃金的矩形層）。面板沒開時不畫。
    pub(crate) fn draw_controls_panel_surface(
        &mut self,
        s: &mut Picture,
        record: Option<&mut Vec<ControlsTextDraw>>,
        draw_text: bool,
    ) {
        if !self.controls.open {
            return;
        }
        s.fill(ColorRgba { r: 0, g: 0, b: 0, a: 170 });
        // Go：`f := g.font`，HD 區域字型（`hdFonts[localeID]`）優先；R5 沒有載入區域字型（R6），`hd_fonts` 是空的。
        let f = self.font.as_ref().map(|f| f.hd_fonts.get(&self.locale_id).cloned().unwrap_or_else(|| f.clone()));
        let c = self.controls.clone();
        let mut p = PanelPainter { s, f, record, draw_text };

        p.frame(CONTROLS_PANEL_RECT, CONTROLS_COLOR_PANEL, CONTROLS_COLOR_BORDER);
        p.label(&txt(&c, "controls.title"), 30, 36, 54, CONTROLS_COLOR_GOLD);
        p.label(&txt(&c, "controls.subtitle"), 30, 54, 33, CONTROLS_COLOR_MUTED);
        let pad_count = self.player_input.as_ref().map_or(0, |i| i.pad_count);
        let mut status = txt(&c, "controls.pad.disconnected");
        if pad_count > 0 {
            status = ControlsLocale::text(c.text.as_ref(), "controls.pad.connected", &[ControlsArg::D(pad_count)]);
        }
        p.frame(CONTROLS_PAD_CHIP, CONTROLS_COLOR_CHIP, CONTROLS_COLOR_CHIP_LINE);
        p.label(&status, CONTROLS_PAD_CHIP.min.x + 8, CONTROLS_PAD_CHIP.min.y + 16, 33, CONTROLS_COLOR_TEXT);

        p.rect(CONTROLS_HEADER_BAND, CONTROLS_COLOR_BAND);
        p.label(&txt(&c, "controls.column.action"), 36, 77, 36, CONTROLS_COLOR_MUTED);
        p.label(&txt(&c, "controls.column.keyboard"), CONTROLS_COLUMN_KEYS, 77, 36, CONTROLS_COLOR_MUTED);
        p.label(&txt(&c, "controls.column.gamepad"), CONTROLS_COLUMN_PAD, 77, 36, CONTROLS_COLOR_MUTED);

        let bindings = self.player_input.as_ref().map_or_else(default_control_bindings, |i| i.bindings.clone());
        let lines = controls_lines();
        let mut row = 0;
        while row < CONTROLS_VISIBLE_LINES && c.scroll + row < lines.len() as i64 {
            let line = lines[(c.scroll + row) as usize];
            let top = CONTROLS_LIST_RECT.min.y + row * CONTROLS_LINE_HEIGHT;
            let band = rect(CONTROLS_LIST_RECT.min.x, top, CONTROLS_LIST_RECT.max.x - 8, top + CONTROLS_LINE_HEIGHT);
            row += 1;
            if line.action < 0 {
                p.rect(band, CONTROLS_COLOR_GROUP);
                p.label(&txt(&c, CONTROLS_GROUP_KEYS[line.group]), 36, top + 15, 33, CONTROLS_COLOR_GOLD);
                continue;
            }
            let action = line.action as usize;
            if action == c.selected {
                p.rect(band, CONTROLS_COLOR_SELECTED);
                p.rect(rect(band.min.x, band.min.y, band.min.x + 2, band.max.y), CONTROLS_COLOR_GOLD);
            }
            p.rect(rect(band.min.x, band.max.y - 1, band.max.x, band.max.y), CONTROLS_COLOR_GROUP);
            p.label(&c.action_name(action), 40, top + 15, 39, CONTROLS_COLOR_TEXT);
            let k = bindings.keyboard[action];
            if k != KEY_UNBOUND {
                p.chip(&control_key_label(k), CONTROLS_COLUMN_KEYS, top + 2, CONTROLS_COLOR_TEXT);
            } else {
                p.label(&txt(&c, "controls.unbound"), CONTROLS_COLUMN_KEYS, top + 15, 33, CONTROLS_COLOR_MUTED);
            }
            let mut x = CONTROLS_COLUMN_PAD;
            let button = bindings.gamepad[action];
            if button != PAD_NONE {
                x += p.chip(&c.pad_label(button), x, top + 2, CONTROLS_COLOR_TEXT) + 4;
            }
            if CONTROL_ACTIONS[action].group == CONTROL_GROUP_MOVEMENT {
                let arrow = control_key_label(CONTROL_ACTIONS[action].logical);
                p.chip(&(txt(&c, "controls.stick") + &arrow), x, top + 2, CONTROLS_COLOR_MUTED);
            } else if bindings.gamepad[action] == PAD_NONE {
                p.label(&txt(&c, "controls.unbound"), x, top + 15, 33, CONTROLS_COLOR_MUTED);
            }
        }
        // 捲軸
        let total = lines.len() as i64;
        let track = rect(
            CONTROLS_LIST_RECT.max.x - 6,
            CONTROLS_LIST_RECT.min.y + 2,
            CONTROLS_LIST_RECT.max.x - 3,
            CONTROLS_LIST_RECT.max.y - 2,
        );
        p.rect(track, CONTROLS_COLOR_GROUP);
        let track_dy = track.max.y - track.min.y;
        let thumb_h = track_dy * CONTROLS_VISIBLE_LINES / total;
        let thumb_y = track.min.y + (track_dy - thumb_h) * c.scroll / 1.max(total - CONTROLS_VISIBLE_LINES);
        p.rect(rect(track.min.x, thumb_y, track.max.x, thumb_y + thumb_h), CONTROLS_COLOR_CHIP_LINE);

        let slider = CONTROLS_SLIDER_ROW;
        if c.selected == CONTROLS_SLIDER_SELECTS {
            p.rect(slider, CONTROLS_COLOR_SELECTED);
            p.rect(rect(slider.min.x, slider.min.y, slider.min.x + 2, slider.max.y), CONTROLS_COLOR_GOLD);
        } else {
            p.rect(slider, CONTROLS_COLOR_BAND);
        }
        p.label(&txt(&c, "controls.deadzone"), 36, 304, 39, CONTROLS_COLOR_TEXT);
        let track = CONTROLS_SLIDER_TRACK;
        p.rect(track, CONTROLS_COLOR_CHIP_LINE);
        let ratio = (bindings.deadzone - MIN_STICK_DEADZONE) as f64 / (MAX_STICK_DEADZONE - MIN_STICK_DEADZONE) as f64;
        let mut fill = track;
        fill.max.x = fill.min.x + (ratio * (track.max.x - track.min.x) as f64) as i64;
        p.rect(fill, CONTROLS_COLOR_GOLD);
        let knob_x = fill.max.x;
        p.rect(rect(knob_x - 4, track.min.y - 5, knob_x + 4, track.max.y + 5), CONTROLS_COLOR_GOLD);
        p.label(&format!("{}%", bindings.deadzone), track.max.x + 14, 304, 39, CONTROLS_COLOR_GOLD);

        p.label(&c.message, 30, 330, 33, CONTROLS_COLOR_GOLD);
        p.label(&txt(&c, "controls.help.keyboard"), 30, 352, 30, CONTROLS_COLOR_MUTED);
        p.label(&txt(&c, "controls.help.gamepad"), 30, 370, 30, CONTROLS_COLOR_MUTED);
        let reset = CONTROLS_RESET_ALL_RECT;
        p.frame(reset, CONTROLS_COLOR_CHIP, CONTROLS_COLOR_CHIP_LINE);
        let reset_label = txt(&c, "controls.reset_all");
        let w = p.measure(&reset_label, 36);
        p.label(
            &reset_label,
            reset.min.x + (reset.max.x - reset.min.x - w) / 2,
            reset.min.y + 18,
            36,
            CONTROLS_COLOR_TEXT,
        );
    }
}

impl Game {
    /// Rust 新增：決定疊層內容的面板狀態（[`PlayerRunner::controls_panel_surface`] 狀態沒變就不重畫）。字級倍率不影響面板
    /// （`faceFor(size)` 直接用像素尺寸）。
    pub(crate) fn controls_panel_render_key(&self) -> String {
        let c = &self.controls;
        let input = self.player_input.as_ref().map(|i| (&i.bindings, i.pad_count));
        format!(
            "{:?}",
            (
                c.selected,
                c.scroll,
                c.capture,
                c.conflict,
                &c.message,
                c.text.is_some(),
                &c.text_locale,
                input,
                &self.locale_id
            )
        )
    }
}

#[cfg(test)]
#[path = "player_controls_panel/player_controls_panel_test.rs"]
mod player_controls_panel_test;

#[cfg(test)]
#[path = "player_controls_panel/rust_golden_r5i_test.rs"]
mod rust_golden_r5i_test;
