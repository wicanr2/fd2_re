//! Go 來源：`remake/cmd/fd2/native_player_input.go`（237 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 4 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// 空游標時 Esc／Z／Numpad5：從 `native_next_player_index` 起依原生記錄順序找下一個可行動的我方單位，
    /// 把游標逐格移過去（`step_native_player_focus`）。記錄 +5 的 0x85 位元任一成立或 +6 不是 2 的跳過。
    ///
    /// Go：`native_player_input.go:12`，`func (g *Game) cycleNativePlayerUnit()`
    pub(crate) fn cycle_native_player_unit(&mut self) {
        let Some(st) = self.st.as_ref() else { return };
        if !st.has_native_map_view_state || self.sel.is_some() || st.units.is_empty() {
            return;
        }
        let records = match battle::native_ai_scoring_records(&st.units) {
            Ok(records) => records,
            Err(err) => {
                self.load_err = format!("native player cycle: {}", err.message());
                return;
            }
        };
        let count = st.units.len() as i64;
        if self.native_next_player_index < 0 || self.native_next_player_index >= count {
            self.load_err = String::from("native player cycle: index outside runtime records");
            return;
        }
        for offset in 0..count {
            let i = (self.native_next_player_index + offset) % count;
            let r = &records[(i * 0x50) as usize..((i + 1) * 0x50) as usize];
            if r[5] & 0x85 == 0 && r[6] == 2 {
                if self.begin_native_player_focus(Some(i as UnitIdx)) {
                    self.native_next_player_index = (i + 1) % count;
                }
                return;
            }
        }
    }

    /// Go：`native_player_input.go:38`，`func (g *Game) beginNativePlayerFocus(u *battle.Unit) bool`
    pub(crate) fn begin_native_player_focus(&mut self, u: Option<UnitIdx>) -> bool {
        let ok = self.st.as_ref().is_some_and(|st| st.has_native_map_view_state)
            && u.is_some_and(|u| self.st.as_ref().unwrap().units[u].has_native_map_presentation);
        if !ok {
            self.load_err = String::from("native player focus: raw position unavailable");
            return false;
        }
        let st = self.st.as_ref().unwrap();
        let p = st.units[u.unwrap()].native_map_presentation;
        if i64::from(p.x) >= st.w || i64::from(p.y) >= st.h {
            self.load_err = String::from("native player focus: position outside field");
            return false;
        }
        self.native_player_focus = Some(battle::Cell { x: i64::from(p.x), y: i64::from(p.y) });
        self.native_player_focus_started = false;
        true
    }

    /// Go：`native_player_input.go:53`，`func (g *Game) stepNativePlayerFocus()`
    pub(crate) fn step_native_player_focus(&mut self) {
        let Some(p) = self.native_player_focus else {
            return;
        };
        if self.st.is_none() {
            return;
        }
        self.step_native_player_focus_body(p);
        // Go 的 defer：任何一條 return 之後都執行。
        if self.native_player_focus.is_none() && self.native_gate_b_after_focus {
            self.restore_native_display_gate_b();
        }
    }

    /// `stepNativePlayerFocus` 在 `defer` 之後的本體（Rust 拆出，讓呼叫端在每條 return 後執行 defer）。
    fn step_native_player_focus_body(&mut self, p: battle::Cell) {
        if !self.native_player_focus_started {
            // 0x1A7AB 0x12D7B(0)→0x12CEA：開頭先 0x11CAC(0)。
            self.native_player_focus_started = true;
            let hook = battle::NATIVE_FOCUS_TRACE_HOOK.read().unwrap_or_else(|e| e.into_inner()).clone();
            if let Some(hook) = hook {
                hook(p.x, p.y, self.st.as_ref().unwrap().native_map_view_state);
            }
            self.redraw_native_map_hud();
        }
        let v = self.st.as_ref().unwrap().native_map_view_state;
        let (mut dx, mut dy) = (0, 0);
        if v.cursor_x > p.x {
            dx = -1;
        } else if v.cursor_x < p.x {
            dx = 1;
        } else if v.cursor_y > p.y {
            dy = -1;
        } else if v.cursor_y < p.y {
            dy = 1;
        } else {
            self.native_player_focus = None;
            return;
        }
        let (moved, ok) = self.st.as_mut().unwrap().move_native_map_cursor(dx, dy);
        if !ok || !moved {
            self.load_err = String::from("native player focus: cursor step rejected");
            self.native_player_focus = None;
            return;
        }
        self.sync_native_map_view();
        let after = self.st.as_ref().unwrap().native_map_view_state;
        self.native_cursor_step_hud(v, after);
        if self.cur_x == p.x && self.cur_y == p.y {
            self.native_player_focus = None;
        }
    }

    /// inspectNativePlayerUnit 的回傳值表示已消費確認鍵，包括原版拒絕的特殊角色。
    ///
    /// Go：`native_player_input.go:99`，`func (g *Game) inspectNativePlayerUnit(u *battle.Unit, force bool) (bool, error)`
    pub(crate) fn inspect_native_player_unit(&mut self, u: UnitIdx, force: bool) -> Result<bool> {
        let unit = self.st.as_ref().expect("Go：g.st 為 nil 時解參考 panic").units[u].clone();
        let records = match battle::native_ai_scoring_records(std::slice::from_ref(&unit)) {
            Ok(records) => records,
            Err(err) => return Err(GameError::from(err).wrap("native player status: ")),
        };
        let r = &records[..0x50];
        if r[7] == 0x79 || r[0x1f] == 10 {
            return Ok(true);
        }
        if !force && r[6] == 2 && r[5] & 0x80 == 0 && r[0x26] == 0 {
            return Ok(false);
        }
        self.begin_native_player_status(&unit)?;
        Ok(true)
    }

    /// 開戰場玩家狀態面板：合成能力值／指令兩頁，以目前地圖畫面為底建 12 幀開啟排程。
    ///
    /// Go：`native_player_input.go:124`，`func (g *Game) beginNativePlayerStatus(u *battle.Unit) error`
    pub(crate) fn begin_native_player_status(&mut self, u: &battle::Unit) -> Result<()> {
        let prepared = self.prepare_native_unit_status_data(Some(u));
        let (Some((data, status, commands)), 256) = (prepared, self.native_ui_palette.len()) else {
            return Err(game_err!("native player status: panel resources unavailable"));
        };
        if self.modern_hd_interface_enabled() {
            todo!("R6: modern_hd_item_panel_chrome.go:98 preflightModernHDItemPanel（HD 狀態面板）");
        }
        self.compose_native_map_frame().map_err(|err| err.wrap("native player status: "))?;
        let source = self.native_map_vga.clone();
        let frames = native_church_panel_frames(&source, &status, true)?;
        let steady = native_church_panel_frame(&source, &status, 0)?;
        let text = snapshot_native_status_transition(&source, Some(&data), NATIVE_STATUS_OPENING, false);
        self.native_player_status = Some(NativePlayerStatusState {
            source,
            status,
            commands,
            steady,
            data: Some(data),
            frames,
            phase: String::from("opening"),
            text,
            ..Default::default()
        });
        Ok(())
    }

    /// 確認鍵：能力值頁且有指令時捲到指令頁（下半部 7 步收起、7 步展開），否則關閉面板。演出期間不收。
    ///
    /// Go：`native_player_input.go:174`，`func (g *Game) advanceNativePlayerStatus()`
    pub(crate) fn advance_native_player_status(&mut self) {
        let Some(s) = self.native_player_status.as_ref() else {
            return;
        };
        if !s.frames.is_empty() {
            return;
        }
        if s.phase == "status" && !s.commands.is_empty() {
            let mut frames = Vec::with_capacity(14);
            let steps = (0..=6).map(|i| (&s.status, i)).chain((0..=6).rev().map(|i| (&s.commands, i)));
            for (panel, i) in steps {
                match native_church_bottom_panel_frame(&s.source, panel, i) {
                    Ok(frame) => frames.push(frame),
                    Err(err) => {
                        self.load_err = err.message().to_string();
                        return;
                    }
                }
            }
            // Go 忽略 `nativeChurchPanelFrame` 的錯誤（`s.steady, _ = …`），錯誤時 steady 變成 nil。
            let steady = native_church_panel_frame(&s.source, &s.commands, 0).unwrap_or_default();
            let text = snapshot_native_status_transition(&s.source, s.data.as_ref(), NATIVE_STATUS_TO_COMMANDS, false);
            let s = self.native_player_status.as_mut().unwrap();
            (s.frames, s.steady, s.text) = (frames, steady, text);
            (s.phase, s.frame, s.drawn) = (String::from("transition"), 0, false);
            return;
        }
        let panel = if s.phase == "commands" { &s.commands } else { &s.status };
        let frames = match native_church_panel_frames(&s.source, panel, false) {
            Ok(frames) => frames,
            Err(err) => {
                self.load_err = err.message().to_string();
                return;
            }
        };
        let commands = s.phase == NATIVE_PLAYER_STATUS_COMMANDS;
        let text = snapshot_native_status_transition(&s.source, s.data.as_ref(), NATIVE_STATUS_CLOSING, commands);
        let s = self.native_player_status.as_mut().unwrap();
        (s.text, s.frames) = (text, frames);
        (s.phase, s.frame, s.drawn) = (String::from("closing"), 0, false);
    }
}

/// Go：`native_player_input.go:114`
/// 不 derive `Debug`／`PartialEq`：`NativeUnitStatusData` 帶的面板素材沒有實作。
#[derive(Clone, Default)]
pub(crate) struct NativePlayerStatusState {
    pub(crate) data: Option<NativeUnitStatusData>,
    pub(crate) text: Option<NativeStatusTransitionText>,
    pub(crate) source: Vec<u8>,
    pub(crate) status: Vec<u8>,
    pub(crate) commands: Vec<u8>,
    pub(crate) steady: Vec<u8>,
    pub(crate) frames: Vec<Vec<u8>>,
    pub(crate) frame: i64,
    pub(crate) phase: String,
    pub(crate) drawn: bool,
}

impl Game {
    /// Go：`native_player_input.go:152`，`func (g *Game) stepNativePlayerStatus()`
    pub(crate) fn step_native_player_status(&mut self) {
        let Some(s) = self.native_player_status.as_mut() else {
            return;
        };
        if !s.drawn || s.frames.is_empty() {
            return;
        }
        s.drawn = false;
        s.frame += 1;
        if s.frame < s.frames.len() as i64 {
            return;
        }
        if s.phase == "closing" {
            self.native_player_status = None;
            return;
        }
        (s.frames, s.frame) = (Vec::new(), 0);
        if s.phase == "opening" {
            s.phase = String::from("status");
        } else {
            s.phase = String::from("commands");
        }
    }
}

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// Go：`native_player_input.go:214`，`func (g *Game) drawNativePlayerStatus(screen *ebiten.Image)`
    ///
    /// R5 階段 2 r5h 逐行移植。開場與第一章只走到 `s == nil` 的守門。HD 的三條（`drawModernBattlePlayerStatus`、
    /// `drawModernHDUnitStatus`、`drawModernHDStatusTransition`）在忠實模式（`modernStoryPortraits`、`modernHDText`
    /// 都是 nil 且介面不是 HD）都回 false、沒有副作用；三者任一成立時 `todo!("r6:…")`。
    pub(crate) fn draw_native_player_status(&mut self, screen: &mut Picture) {
        if self.native_player_status.is_none() {
            return;
        }
        if self.facility_hd_draw_active() {
            todo!(
                "r6:modern_hd_panel_transition.go:drawModernBattlePlayerStatus、modern_hd_status_text.go:drawModernHDUnitStatus、drawModernHDStatusTransition"
            );
        }
        let s = self.native_player_status.as_ref().unwrap();
        let frame = if s.frames.is_empty() { s.steady.clone() } else { s.frames[s.frame as usize].clone() };
        let palette = self.native_ui_palette.clone();
        self.present_native_class_frame_with_palette(screen, &frame, &palette);
        self.native_player_status.as_mut().unwrap().drawn = true;
    }
}
